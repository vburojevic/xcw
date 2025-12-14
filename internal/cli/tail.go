package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/vburojevic/xcw/internal/config"
	"github.com/vburojevic/xcw/internal/domain"
	"github.com/vburojevic/xcw/internal/filter"
	"github.com/vburojevic/xcw/internal/output"
	"github.com/vburojevic/xcw/internal/session"
	"github.com/vburojevic/xcw/internal/simulator"
	"github.com/vburojevic/xcw/internal/tmux"
)

// TailCmd streams real-time logs from a simulator
type TailCmd struct {
	TailFilterFlags
	TailOutputFlags
	TailAgentFlags

	Simulator   string   `short:"s" help:"Simulator name or UDID"`
	Booted      bool     `short:"b" help:"Use booted simulator (error if multiple)"`
	App         string   `short:"a" help:"App bundle identifier to filter logs (required unless --predicate or --all)"`
	All         bool     `help:"Allow streaming without --app/--predicate (can be very noisy)"`
	Subsystem   []string `help:"Filter by subsystem (can be repeated)"`
	Category    []string `help:"Filter by category (can be repeated)"`
	Predicate   string   `help:"Raw NSPredicate filter (overrides --app, --subsystem, --category)"`
	BufferSize  int      `default:"100" help:"Number of recent logs to buffer"`
	Recorder    string   `short:"r" help:"Write raw log stream to file (with sessions)"`
	SessionName string   `help:"Custom session name for recording"`
}

// Run executes the tail command
func (c *TailCmd) Run(globals *Globals) error {
	// Disable styles when stdout is not a TTY
	maybeNoStyle(globals)
	applyTailDefaults(globals.Config, c)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Unique ID for this tail invocation (carried on all events)
	tailID := generateTailID()
	var log *agentLogger
	clk := clock.New()

	// Validate mutual exclusivity of flags
	if globals.FlagProvided("simulator") && globals.FlagProvided("booted") {
		return c.outputError(globals, "INVALID_FLAGS", "--simulator and --booted are mutually exclusive")
	}
	if err := validateFlags(globals, c.DryRunJSON, c.Tmux); err != nil {
		return err
	}
	if err := validateAppPredicateAll(c.App, c.Predicate, c.All, len(c.Subsystem) > 0 || len(c.Category) > 0); err != nil {
		return outputErrorCommon(globals, err.Code, err.Message, err.Hint)
	}

	// Find the simulator
	mgr := simulator.NewManager()
	device, err := resolveSimulatorDevice(ctx, mgr, c.Simulator, c.Booted)
	if err != nil {
		return c.outputError(globals, "DEVICE_NOT_FOUND", err.Error(), hintForStreamOrQuery(err))
	}
	globals.Debug("Found device: %s (UDID: %s, State: %s)", device.Name, device.UDID, device.State)

	// Fetch app version/build for metadata (best-effort)
	appVersion, appBuild := "", ""
	if c.App != "" {
		if v, b, err := mgr.GetAppInfo(ctx, device.UDID, c.App); err == nil {
			appVersion, appBuild = v, b
			globals.Debug("App info: version=%s build=%s", appVersion, appBuild)
		} else {
			globals.Debug("App info unavailable: %v", err)
		}
	}

	// Determine output destination
	var outputWriter io.Writer = globals.Stdout
	var tmuxMgr *tmux.Manager
	rotator := newRotation(nil)

	// Determine output file path builder (supports per-session rotation)
	var pathBuilder func(session int) (string, error)
	if c.Output != "" {
		ext := filepath.Ext(c.Output)
		base := strings.TrimSuffix(c.Output, ext)
		if ext == "" {
			ext = ".ndjson"
		}
		pathBuilder = func(session int) (string, error) {
			return fmt.Sprintf("%s.session%d%s", base, session, ext), nil
		}
	} else if c.SessionDir != "" || c.SessionPrefix != "" {
		prefix := c.SessionPrefix
		if prefix == "" {
			prefix = c.App
		}
		pathBuilder = func(session int) (string, error) {
			return GenerateSessionPath(c.SessionDir, prefix)
		}
	}

	// Helper to open / rotate output file
	openOutput := func(sessionNum int) error {
		if pathBuilder == nil {
			return nil
		}
		rotator.pathBuilder = pathBuilder
		bw, file, path, err := rotator.Open(sessionNum)
		if err != nil {
			return c.outputError(globals, "FILE_CREATE_ERROR", err.Error())
		}
		_ = file
		if bw != nil {
			outputWriter = bw
		}
		if !globals.Quiet && path != "" {
			if globals.Format == "ndjson" {
				w := output.NewNDJSONWriter(globals.Stdout)
				if err := w.WriteInfo(
					fmt.Sprintf("Writing logs to %s", path),
					device.Name, device.UDID, "", ""); err != nil {
					return err
				}
				if err := w.WriteRotation(path, tailID, sessionNum); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(globals.Stderr, "Writing logs to %s\n", path); err != nil {
					globals.Debug("failed to write output path: %v", err)
				}
			}
		}
		return nil
	}

	if !c.Tmux {
		if err := openOutput(1); err != nil {
			return err
		}
		if pathBuilder != nil {
			defer func() {
				if err := rotator.Close(); err != nil {
					globals.Debug("failed to close output file: %v", err)
				}
			}()
		}
	} else {
		// When tmux is on, skip file output to keep behavior consistent with previous versions
		pathBuilder = nil
	}

	if c.Tmux {
		// Setup tmux session
		globals.Debug("Tmux mode enabled")
		sessionName := c.Session
		if sessionName == "" {
			sessionName = tmux.GenerateSessionName(device.Name)
		}
		globals.Debug("Tmux session name: %s", sessionName)

		if !tmux.IsTmuxAvailable() {
			emitWarning(globals, output.NewEmitter(globals.Stdout), "tmux not installed, falling back to stdout")
		} else {
			cfg := &tmux.Config{
				SessionName:   sessionName,
				SimulatorName: device.Name,
				Detached:      true,
			}

			tmuxMgr, err = tmux.NewManager(cfg)
			if err != nil {
				emitWarning(globals, output.NewEmitter(globals.Stdout), fmt.Sprintf("failed to create tmux session: %v, falling back to stdout", err))
			} else {
				if err := tmuxMgr.GetOrCreateSession(); err != nil {
					emitWarning(globals, output.NewEmitter(globals.Stdout), fmt.Sprintf("failed to setup tmux session: %v, falling back to stdout", err))
				} else {
					// Successfully created tmux session
					outputWriter = tmux.NewWriter(tmuxMgr)

					// Clear pane and show banner
					if err := tmuxMgr.ClearPaneWithBanner(fmt.Sprintf("Watching: %s (%s)", device.Name, c.App)); err != nil {
						emitWarning(globals, output.NewEmitter(globals.Stdout), fmt.Sprintf("failed to clear tmux pane: %v", err))
					}

					// Output attach command
					if globals.Format == "ndjson" {
						if err := output.NewNDJSONWriter(globals.Stdout).WriteTmux(sessionName, tmuxMgr.AttachCommand()); err != nil {
							return err
						}
					} else {
						if _, err := fmt.Fprintf(globals.Stdout, "Tmux session: %s\n", sessionName); err != nil {
							return err
						}
						if _, err := fmt.Fprintf(globals.Stdout, "Attach with: %s\n", tmuxMgr.AttachCommand()); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	// Cleanup tmux on exit (session persists)
	if tmuxMgr != nil {
		defer tmuxMgr.Cleanup()
	}

	// Output device info if not quiet and not in tmux mode (tmux already shows banner)
	if !globals.Quiet && tmuxMgr == nil {
		if globals.Format == "ndjson" {
			if err := output.NewNDJSONWriter(globals.Stdout).WriteInfo(
				fmt.Sprintf("Streaming logs from %s (%s)", device.Name, device.UDID),
				device.Name, device.UDID, "", ""); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(globals.Stderr, "Streaming logs from %s (%s)\n", device.Name, device.UDID); err != nil {
				globals.Debug("failed to write stream info: %v", err)
			}
			if _, err := fmt.Fprintf(globals.Stderr, "Filtering by app: %s\n", c.App); err != nil {
				globals.Debug("failed to write stream info: %v", err)
			}
			if c.Pattern != "" {
				if _, err := fmt.Fprintf(globals.Stderr, "Pattern filter: %s\n", c.Pattern); err != nil {
					globals.Debug("failed to write stream info: %v", err)
				}
			}
			if _, err := fmt.Fprintln(globals.Stderr, "Press Ctrl+C to stop"); err != nil {
				globals.Debug("failed to write stream info: %v", err)
			}
		}
	}

	pattern, excludePatterns, whereFilter, err := buildFilters(c.Pattern, c.Exclude, c.Where)
	if err != nil {
		return c.outputError(globals, "INVALID_FILTER", err.Error(), hintForFilter(err))
	}
	// Pattern/exclude are applied in the simulator streamer; keep pipeline for where-only filtering.
	pipeline := filter.NewPipeline(nil, nil, whereFilter)

	// Determine log level (command-specific overrides global)
	minLevel, maxLevel := resolveLevels(c.MinLevel, c.MaxLevel, globals.Level)

	// Create streamer
	streamer := simulator.NewStreamer(mgr)
	opts := simulator.StreamOptions{
		BundleID:          c.App,
		Subsystems:        c.Subsystem,
		Categories:        c.Category,
		Processes:         c.Process,
		MinLevel:          minLevel,
		MaxLevel:          maxLevel,
		Pattern:           pattern,
		ExcludePatterns:   excludePatterns,
		ExcludeSubsystems: c.ExcludeSubsystem,
		BufferSize:        c.BufferSize,
		RawPredicate:      c.Predicate,
		Verbose:           globals.Verbose,
	}

	if c.DryRunJSON {
		enc := json.NewEncoder(globals.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(opts)
	}

	globals.Debug("Stream options: BundleID=%s, MinLevel=%s, BufferSize=%d", opts.BundleID, opts.MinLevel, opts.BufferSize)
	if opts.Pattern != nil {
		globals.Debug("Pattern filter: %s", opts.Pattern.String())
	}
	if len(opts.ExcludePatterns) > 0 {
		globals.Debug("Exclude patterns: %d patterns", len(opts.ExcludePatterns))
	}
	if len(opts.ExcludeSubsystems) > 0 {
		globals.Debug("Exclude subsystems: %v", opts.ExcludeSubsystems)
	}

	globals.Debug("Starting log stream...")
	if err := streamer.Start(ctx, device.UDID, opts); err != nil {
		return c.outputError(globals, "STREAM_FAILED", err.Error(), hintForStreamOrQuery(err))
	}
	defer func() {
		if err := streamer.Stop(); err != nil {
			globals.Debug("failed to stop streamer: %v", err)
		}
	}()
	globals.Debug("Log stream started successfully")

	// Create output writer based on format
	var writer interface {
		Write(entry *domain.LogEntry) error
		WriteSummary(summary *domain.LogSummary) error
		WriteHeartbeat(h *output.Heartbeat) error
	}

	var emitter *output.Emitter
	setWriter := func(w io.Writer) {
		if globals.Format == "ndjson" {
			emitter = output.NewEmitter(w)
			writer = emitter
		} else {
			emitter = nil
			writer = output.NewTextWriter(w)
		}
	}
	setWriter(outputWriter)

	// Create session tracker for detecting app relaunches (only meaningful when tailing an app)
	var sessionTracker tailSessionTracker
	if c.App != "" {
		sessionTracker = session.NewTracker(c.App, device.Name, device.UDID, tailID, appVersion, appBuild)
		globals.Debug("Session tracking enabled")
	} else {
		sessionTracker = &noopSessionTracker{}
		globals.Debug("Session tracking disabled (no --app)")
	}
	log = newAgentLogger(globals, tailID, sessionTracker.CurrentSession)

	// Emit metadata for agents
	if emitter != nil {
		if err := emitter.Metadata(Version, Commit, ""); err != nil {
			return err
		}
	}

	// Emit ready event when --wait-for-launch is used (signals log capture is active)
	if c.WaitForLaunch {
		if emitter != nil {
			if err := emitter.Ready(
				clk.Now().UTC().Format(time.RFC3339Nano),
				device.Name,
				device.UDID,
				c.App,
				tailID,
				sessionTracker.CurrentSession(),
			); err != nil {
				return err
			}
			if !c.NoAgentHints {
				if err := emitter.AgentHints(tailID, sessionTracker.CurrentSession(), defaultHintsForTail(c.App != "")); err != nil {
					return err
				}
			}
		} else {
			target := c.App
			if target == "" {
				target = "all logs"
			}
			if _, err := fmt.Fprintf(globals.Stderr, "%s\n", infoStyle.Render(fmt.Sprintf("Ready: log capture active for %s", target))); err != nil {
				globals.Debug("failed to write ready message: %v", err)
			}
			if !c.NoAgentHints {
				if _, err := fmt.Fprintf(globals.Stderr, "%s\n", warnStyle.Render(fmt.Sprintf("agent_hints: match tail_id=%s", tailID))); err != nil {
					globals.Debug("failed to write agent_hints message: %v", err)
				}
			}
		}
	}

	// Parse summary interval
	var summaryTicker *clock.Ticker
	if c.SummaryInterval != "" {
		interval, err := time.ParseDuration(c.SummaryInterval)
		if err != nil {
			return c.outputError(globals, "INVALID_INTERVAL", fmt.Sprintf("invalid summary interval: %s", err))
		}
		summaryTicker = clk.Ticker(interval)
		defer summaryTicker.Stop()
	}

	// Parse heartbeat interval
	var heartbeatTicker *clock.Ticker
	if c.Heartbeat != "" {
		interval, err := time.ParseDuration(c.Heartbeat)
		if err != nil {
			return c.outputError(globals, "INVALID_HEARTBEAT", fmt.Sprintf("invalid heartbeat interval: %s", err))
		}
		heartbeatTicker = clk.Ticker(interval)
		defer heartbeatTicker.Stop()
	}

	// Max duration/logs cutoffs
	var cutoffTimer *clock.Timer
	if c.MaxDuration != "" {
		dur, err := time.ParseDuration(c.MaxDuration)
		if err != nil {
			return c.outputError(globals, "INVALID_MAX_DURATION", fmt.Sprintf("invalid max duration: %s", err))
		}
		cutoffTimer = clk.Timer(dur)
		defer cutoffTimer.Stop()
	}
	var maxLogs = c.MaxLogs

	// Track metrics for heartbeat
	startTime := clk.Now()
	logsSinceLast := 0
	lastSeen := clk.Now()
	totalLogs := 0

	// Parse idle timeout for session rollover
	var idleTimer *clock.Timer
	var idleDuration time.Duration
	if c.SessionIdle != "" {
		var err error
		idleDuration, err = time.ParseDuration(c.SessionIdle)
		if err != nil {
			return c.outputError(globals, "INVALID_SESSION_IDLE", fmt.Sprintf("invalid session idle duration: %s", err))
		}
		idleTimer = clk.Timer(idleDuration)
		defer idleTimer.Stop()
	}

	// Setup dedupe filter if enabled
	var dedupeFilter *filter.DedupeFilter
	if c.Dedupe {
		var dedupeWindow time.Duration
		if c.DedupeWindow != "" {
			var err error
			dedupeWindow, err = time.ParseDuration(c.DedupeWindow)
			if err != nil {
				return c.outputError(globals, "INVALID_DEDUPE_WINDOW", fmt.Sprintf("invalid dedupe window: %s", err))
			}
		}
		dedupeFilter = filter.NewDedupeFilter(dedupeWindow)
	}

	emitHints := func() {
		if c.NoAgentHints {
			return
		}
		hints := defaultHintsForTail(c.App != "")
		if globals.Format == "ndjson" {
			if err := output.NewNDJSONWriter(globals.Stdout).WriteAgentHints(tailID, sessionTracker.CurrentSession(), hints); err != nil {
				globals.Debug("failed to write agent_hints: %v", err)
			}
		} else {
			if _, err := fmt.Fprintf(globals.Stderr, "[XCW] agent_hints: %s (tail_id=%s, session=%d)\n", strings.Join(hints, "; "), tailID, sessionTracker.CurrentSession()); err != nil {
				globals.Debug("failed to write agent_hints: %v", err)
			}
		}
	}

	emitStats := func(now time.Time) error {
		if emitter == nil {
			return nil
		}
		diag := streamer.GetDiagnostics()
		return emitter.WriteStats(&output.StreamStats{
			Type:                "stats",
			SchemaVersion:       output.SchemaVersion,
			Timestamp:           now.UTC().Format(time.RFC3339Nano),
			TailID:              tailID,
			Session:             sessionTracker.CurrentSession(),
			Reconnects:          diag.Reconnects,
			ParseDrops:          diag.ParseDrops,
			TimestampParseDrops: diag.TimestampParseDrops,
			ChannelDrops:        diag.ChannelDrops,
			Buffered:            diag.Buffered,
			LastSeenTimestamp:   lastSeen.UTC().Format(time.RFC3339Nano),
		})
	}

	// Process logs
	for {
		select {
		case <-ctx.Done():
			// Output final summary
			if err := c.outputSummary(writer, streamer, tailID, clk.Now()); err != nil {
				return err
			}
			if final := sessionTracker.GetFinalSummary(); final != nil {
				if emitter != nil {
					if err := emitter.SessionEnd(final); err != nil {
						return err
					}
					if err := emitter.ClearBuffer("session_end", tailID, final.Session); err != nil {
						return err
					}
					if err := emitter.Cutoff("sigint", tailID, final.Session, totalLogs); err != nil {
						return err
					}
				}
			}
			return nil

		case entry := <-streamer.Logs():
			// Reset idle timer on activity
			if idleTimer != nil {
				if !idleTimer.Stop() {
					select {
					case <-idleTimer.C:
					default:
					}
				}
				idleTimer.Reset(idleDuration)
			}

			// Check for session change (app relaunch)
			if sessionChange := sessionTracker.CheckEntry(&entry); sessionChange != nil {
				if log != nil {
					if sessionChange.StartSession != nil {
						log.Debug("session rollover -> %d (pid=%d)", sessionChange.StartSession.Session, sessionChange.StartSession.PID)
					} else if sessionChange.EndSession != nil {
						log.Debug("session ended -> %d", sessionChange.EndSession.Session)
					}
				}
				if emitter != nil && globals.Verbose && sessionChange.EndSession != nil && sessionChange.StartSession != nil {
					if err := emitter.SessionDebug(&output.SessionDebugOutput{
						Type:        "session_debug",
						TailID:      tailID,
						Session:     sessionChange.StartSession.Session,
						PrevSession: sessionChange.EndSession.Session,
						PID:         sessionChange.StartSession.PID,
						PrevPID:     sessionChange.EndSession.PID,
						Reason:      "relaunch",
						Summary: map[string]interface{}{
							"total_logs": sessionChange.EndSession.Summary.TotalLogs,
							"errors":     sessionChange.EndSession.Summary.Errors,
							"faults":     sessionChange.EndSession.Summary.Faults,
						},
					}); err != nil {
						return err
					}
				}
				// Session changed - emit events
				if sessionChange.EndSession != nil {
					// Output session end with summary
					if emitter != nil {
						if err := emitter.SessionEnd(sessionChange.EndSession); err != nil {
							return err
						}
						if err := emitter.ClearBuffer("session_end", tailID, sessionChange.EndSession.Session); err != nil {
							return err
						}
						emitHints()
					}
				}

				// Rotate file per session when configured
				if pathBuilder != nil && sessionChange.StartSession != nil {
					if err := openOutput(sessionChange.StartSession.Session); err != nil {
						return err
					}
					setWriter(outputWriter)
				}

				if sessionChange.StartSession != nil {
					// Output stderr alert for AI agents
					if sessionChange.StartSession.Alert == "APP_RELAUNCHED" {
						prevSummary := ""
						if sessionChange.EndSession != nil {
							prevSummary = fmt.Sprintf(" - Previous: %d logs, %d errors",
								sessionChange.EndSession.Summary.TotalLogs,
								sessionChange.EndSession.Summary.Errors)
						}
						msg := fmt.Sprintf("🚀 NEW SESSION: App relaunched (PID: %d)%s",
							sessionChange.StartSession.PID, prevSummary)
						if _, err := fmt.Fprintf(globals.Stderr, "%s\n", bannerStyle.Render(msg)); err != nil {
							globals.Debug("failed to write session banner: %v", err)
						}
					}

					// Output JSON session start event
					if emitter != nil {
						if err := emitter.SessionStart(sessionChange.StartSession); err != nil {
							return err
						}
						if err := emitter.ClearBuffer("session_start", tailID, sessionChange.StartSession.Session); err != nil {
							return err
						}
						emitHints()
					}

					// Write tmux session banner if in tmux mode
					if tmuxMgr != nil && sessionChange.StartSession.Alert == "APP_RELAUNCHED" {
						var prevSummary *domain.SessionSummary
						if sessionChange.EndSession != nil {
							prevSummary = &sessionChange.EndSession.Summary
						}
						if err := tmuxMgr.WriteSessionBanner(
							sessionChange.StartSession.Session,
							c.App,
							sessionChange.StartSession.PID,
							prevSummary,
						); err != nil {
							globals.Debug("failed to write tmux session banner: %v", err)
						}
					}
				}
			}

			// Apply where filter if enabled
			if pipeline != nil && !pipeline.Match(&entry) {
				continue // Skip entries that don't match where clauses
			}

			// Apply dedupe filter if enabled
			if dedupeFilter != nil {
				result := dedupeFilter.Check(&entry)
				if !result.ShouldEmit {
					continue // Skip duplicate
				}
				// Add dedupe metadata to first occurrence
				if result.Count > 1 {
					entry.DedupeCount = result.Count
					entry.DedupeFirst = result.FirstSeen.Format(time.RFC3339)
					entry.DedupeLast = result.LastSeen.Format(time.RFC3339)
				}
			}

			// Set session number on entry
			entry.Session = sessionTracker.CurrentSession()
			entry.TailID = tailID

			if err := writer.Write(&entry); err != nil {
				return err
			}
			logsSinceLast++
			totalLogs++
			lastSeen = clk.Now()
			if maxLogs > 0 && totalLogs >= maxLogs {
				if final := sessionTracker.GetFinalSummary(); final != nil && globals.Format == "ndjson" {
					if err := emitter.SessionEnd(final); err != nil {
						return err
					}
					if err := emitter.Cutoff("max_logs", tailID, final.Session, totalLogs); err != nil {
						return err
					}
					if err := emitter.WriteHeartbeat(&output.Heartbeat{
						Type:              "heartbeat",
						SchemaVersion:     output.SchemaVersion,
						Timestamp:         clk.Now().UTC().Format(time.RFC3339Nano),
						UptimeSeconds:     int64(clk.Since(startTime).Seconds()),
						LogsSinceLast:     logsSinceLast,
						TailID:            tailID,
						LatestSession:     sessionTracker.CurrentSession(),
						LastSeenTimestamp: lastSeen.UTC().Format(time.RFC3339Nano),
					}); err != nil {
						return err
					}
					if err := emitStats(clk.Now()); err != nil {
						return err
					}
				}
				return nil
			}

		case err := <-streamer.Errors():
			if !globals.Quiet {
				if strings.HasPrefix(err.Error(), "reconnect_notice:") {
					if emitter != nil {
						if err := emitter.WriteReconnect(err.Error(), tailID, "info"); err != nil {
							return err
						}
					} else {
						if _, werr := fmt.Fprintf(globals.Stderr, "%s\n", err.Error()); werr != nil {
							globals.Debug("failed to write reconnect notice: %v", werr)
						}
					}
				} else {
					emitWarning(globals, emitter, err.Error())
				}
			}

		case <-func() <-chan time.Time {
			if summaryTicker != nil {
				return summaryTicker.C
			}
			return nil
		}():
			if err := c.outputSummary(writer, streamer, tailID, clk.Now()); err != nil {
				return err
			}

		case <-func() <-chan time.Time {
			if cutoffTimer != nil {
				return cutoffTimer.C
			}
			if heartbeatTicker != nil {
				return heartbeatTicker.C
			}
			return nil
		}():
			// cutoff takes precedence
			if cutoffTimer != nil {
				if final := sessionTracker.GetFinalSummary(); final != nil && globals.Format == "ndjson" {
					if err := emitter.SessionEnd(final); err != nil {
						return err
					}
					if err := emitter.Cutoff("max_duration", tailID, final.Session, totalLogs); err != nil {
						return err
					}
					if err := emitter.WriteHeartbeat(&output.Heartbeat{
						Type:              "heartbeat",
						SchemaVersion:     output.SchemaVersion,
						Timestamp:         clk.Now().UTC().Format(time.RFC3339Nano),
						UptimeSeconds:     int64(clk.Since(startTime).Seconds()),
						LogsSinceLast:     logsSinceLast,
						TailID:            tailID,
						LatestSession:     sessionTracker.CurrentSession(),
						LastSeenTimestamp: lastSeen.UTC().Format(time.RFC3339Nano),
					}); err != nil {
						return err
					}
					if err := emitStats(clk.Now()); err != nil {
						return err
					}
				}
				return nil
			}

			heartbeat := heartbeatPool.Get().(*output.Heartbeat)
			*heartbeat = output.Heartbeat{
				Type:              "heartbeat",
				SchemaVersion:     output.SchemaVersion,
				Timestamp:         clk.Now().UTC().Format(time.RFC3339Nano),
				UptimeSeconds:     int64(clk.Since(startTime).Seconds()),
				LogsSinceLast:     logsSinceLast,
				TailID:            tailID,
				LatestSession:     sessionTracker.CurrentSession(),
				LastSeenTimestamp: lastSeen.UTC().Format(time.RFC3339Nano),
			}
			if err := writer.WriteHeartbeat(heartbeat); err != nil {
				heartbeatPool.Put(heartbeat)
				return err
			}
			if err := emitStats(clk.Now()); err != nil {
				heartbeatPool.Put(heartbeat)
				return err
			}
			if log != nil {
				log.Debug("heartbeat logs_since_last=%d latest_session=%d", logsSinceLast, heartbeat.LatestSession)
			}
			logsSinceLast = 0
			heartbeatPool.Put(heartbeat)

		case <-func() <-chan time.Time {
			if idleTimer != nil {
				return idleTimer.C
			}
			return nil
		}():
			if idleTimer != nil {
				// Emit forced rollover due to idle timeout
				if sessionChange := sessionTracker.ForceRollover("IDLE_TIMEOUT"); sessionChange != nil {
					if log != nil {
						log.Debug("idle rollover -> %d (pid=%d)", sessionChange.StartSession.Session, sessionChange.StartSession.PID)
					}
					if emitter != nil && globals.Verbose && sessionChange.EndSession != nil && sessionChange.StartSession != nil {
						if err := emitter.SessionDebug(&output.SessionDebugOutput{
							Type:        "session_debug",
							TailID:      tailID,
							Session:     sessionChange.StartSession.Session,
							PrevSession: sessionChange.EndSession.Session,
							PID:         sessionChange.StartSession.PID,
							PrevPID:     sessionChange.EndSession.PID,
							Reason:      "idle_timeout",
							Summary: map[string]interface{}{
								"total_logs": sessionChange.EndSession.Summary.TotalLogs,
								"errors":     sessionChange.EndSession.Summary.Errors,
								"faults":     sessionChange.EndSession.Summary.Faults,
							},
						}); err != nil {
							return err
						}
					}
					if sessionChange.EndSession != nil && emitter != nil {
						if err := emitter.SessionEnd(sessionChange.EndSession); err != nil {
							return err
						}
						if err := emitter.ClearBuffer("session_end", tailID, sessionChange.EndSession.Session); err != nil {
							return err
						}
						emitHints()
					}
					if pathBuilder != nil && sessionChange.StartSession != nil {
						if err := openOutput(sessionChange.StartSession.Session); err != nil {
							return err
						}
						setWriter(outputWriter)
					}
					if sessionChange.StartSession != nil {
						if emitter != nil {
							if err := emitter.SessionStart(sessionChange.StartSession); err != nil {
								return err
							}
							if err := emitter.ClearBuffer("session_start", tailID, sessionChange.StartSession.Session); err != nil {
								return err
							}
							emitHints()
						}
						if tmuxMgr != nil {
							var prevSummary *domain.SessionSummary
							if sessionChange.EndSession != nil {
								prevSummary = &sessionChange.EndSession.Summary
							}
							if err := tmuxMgr.WriteSessionBanner(
								sessionChange.StartSession.Session,
								c.App,
								sessionChange.StartSession.PID,
								prevSummary,
							); err != nil {
								globals.Debug("failed to write tmux session banner: %v", err)
							}
						}
					}
				}
				// restart timer
				idleTimer.Reset(idleDuration)
			}
		}
	}
}

func (c *TailCmd) outputSummary(writer interface {
	WriteSummary(*domain.LogSummary) error
}, streamer *simulator.Streamer, tailID string, now time.Time) error {
	total, errors, faults := streamer.GetStats()
	summary := &domain.LogSummary{
		Type:       "summary",
		TotalCount: total,
		ErrorCount: errors,
		FaultCount: faults,
		HasErrors:  errors > 0,
		HasFaults:  faults > 0,
		WindowEnd:  now,
		TailID:     tailID,
	}
	return writer.WriteSummary(summary)
}

func (c *TailCmd) outputError(globals *Globals, code, message string, hint ...string) error {
	return outputErrorCommon(globals, code, message, hint...)
}

func generateTailID() string {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("tail-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("tail-%s-%d", hex.EncodeToString(b[:]), time.Now().UnixNano())
}

var heartbeatPool = sync.Pool{New: func() interface{} { return &output.Heartbeat{} }}

func defaultHints() []string {
	return []string{
		"use latest session only",
		"match tail_id to current tail invocation",
		"reset caches on clear_buffer/session_start/session_end",
		"use newest rotated file unless comparing runs",
	}
}

func defaultHintsForTail(hasApp bool) []string {
	if hasApp {
		return defaultHints()
	}
	return []string{
		"match tail_id to current tail invocation",
		"no session tracking when --app is omitted",
		"use --predicate to narrow at source; --all is intentionally broad",
	}
}

var (
	warnStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203"))
	infoStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("45"))
	bannerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("46"))
)

func maybeNoStyle(globals *Globals) {
	if globals == nil {
		return
	}
	if globals.Stdout != nil {
		if f, ok := globals.Stdout.(*os.File); ok {
			if !isatty.IsTerminal(f.Fd()) {
				warnStyle = warnStyle.UnsetForeground().UnsetBold()
				infoStyle = infoStyle.UnsetForeground().UnsetBold()
				bannerStyle = bannerStyle.UnsetForeground().UnsetBold()
			}
		}
	}
}

func applyTailDefaults(cfg *config.Config, c *TailCmd) {
	if cfg == nil {
		return
	}
	if c.Simulator == "" {
		if cfg.Tail.Simulator != "" {
			c.Simulator = cfg.Tail.Simulator
		} else if cfg.Defaults.Simulator != "" {
			c.Simulator = cfg.Defaults.Simulator
		}
	}
	if c.App == "" && c.Predicate == "" && cfg.Tail.App != "" {
		c.App = cfg.Tail.App
	}
	if c.SummaryInterval == "" && cfg.Tail.SummaryInterval != "" {
		c.SummaryInterval = cfg.Tail.SummaryInterval
	}
	if c.Heartbeat == "" && cfg.Tail.Heartbeat != "" {
		c.Heartbeat = cfg.Tail.Heartbeat
	}
	if c.SessionIdle == "" && cfg.Tail.SessionIdle != "" {
		c.SessionIdle = cfg.Tail.SessionIdle
	}
	if len(c.Exclude) == 0 && len(cfg.Tail.Exclude) > 0 {
		c.Exclude = append(c.Exclude, cfg.Tail.Exclude...)
	}
	if len(c.Where) == 0 && len(cfg.Tail.Where) > 0 {
		c.Where = append(c.Where, cfg.Tail.Where...)
	}
	if c.BufferSize == 100 && cfg.Defaults.BufferSize != 0 {
		c.BufferSize = cfg.Defaults.BufferSize
	}
}
