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
	"syscall"
	"time"

	"github.com/vburojevic/xcw/internal/domain"
	"github.com/vburojevic/xcw/internal/filter"
	"github.com/vburojevic/xcw/internal/output"
	"github.com/vburojevic/xcw/internal/session"
	"github.com/vburojevic/xcw/internal/simulator"
	"github.com/vburojevic/xcw/internal/tmux"
)

// TailCmd streams real-time logs from a simulator
type TailCmd struct {
	Simulator        string   `short:"s" help:"Simulator name or UDID"`
	Booted           bool     `short:"b" help:"Use booted simulator (error if multiple)"`
	App              string   `short:"a" required:"" help:"App bundle identifier to filter logs"`
	Pattern          string   `short:"p" aliases:"filter" help:"Regex pattern to filter log messages"`
	Exclude          []string `short:"x" help:"Regex pattern to exclude from log messages (can be repeated)"`
	ExcludeSubsystem []string `help:"Exclude logs from subsystem (can be repeated, supports * wildcard)"`
	Subsystem        []string `help:"Filter by subsystem (can be repeated)"`
	Category         []string `help:"Filter by category (can be repeated)"`
	Process          []string `help:"Filter by process name (can be repeated)"`
	MinLevel         string   `help:"Minimum log level: debug, info, default, error, fault (overrides global --level)"`
	MaxLevel         string   `help:"Maximum log level: debug, info, default, error, fault"`
	Predicate        string   `help:"Raw NSPredicate filter (overrides --app, --subsystem, --category)"`
	BufferSize       int      `default:"100" help:"Number of recent logs to buffer"`
	SummaryInterval  string   `help:"Emit periodic summaries (e.g., '30s', '1m')"`
	Heartbeat        string   `help:"Emit periodic heartbeat messages (e.g., '10s', '30s')"`
	Output           string   `short:"o" help:"Write output to explicit file path"`
	SessionDir       string   `help:"Directory for session files (default: ~/.xcw/sessions)"`
	SessionPrefix    string   `help:"Prefix for session filename (default: app bundle ID)"`
	Tmux             bool     `help:"Output to tmux session"`
	Session          string   `help:"Custom tmux session name (default: xcw-<simulator>)"`
	WaitForLaunch    bool     `help:"Start streaming immediately, emit 'ready' event when capture is active"`
	Dedupe           bool     `help:"Collapse repeated identical messages"`
	DedupeWindow     string   `help:"Time window for deduplication (e.g., '5s', '1m'). Without this, only consecutive duplicates are collapsed"`
	Where            []string `short:"w" help:"Field filter (e.g., 'level=error', 'message~timeout'). Operators: =, !=, ~, !~, >=, <=, ^, $"`
	SessionIdle      string   `help:"Emit session boundary after idle period with no logs (e.g., '60s')"`
	NoAgentHints     bool     `help:"Suppress agent_hints banners (leave off for AI agents)"`
	DryRunJSON       bool     `help:"Print resolved stream options as JSON and exit (no streaming)"`
	MaxDuration      string   `help:"Stop after duration (e.g., '5m') emitting session_end (agent-safe cutoff)"`
	MaxLogs          int      `help:"Stop after N logs emitting session_end (agent-safe cutoff)"`
}

// Run executes the tail command
func (c *TailCmd) Run(globals *Globals) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Unique ID for this tail invocation (carried on all events)
	tailID := generateTailID()
	var log *agentLogger

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Validate mutual exclusivity of flags
	if c.Simulator != "" && c.Booted {
		return c.outputError(globals, "INVALID_FLAGS", "--simulator and --booted are mutually exclusive")
	}
	if c.DryRunJSON && c.Tmux {
		return c.outputError(globals, "INVALID_FLAGS", "--dry-run-json cannot be combined with --tmux")
	}
	if c.DryRunJSON && globals.Format != "ndjson" {
		return c.outputError(globals, "INVALID_FLAGS", "--dry-run-json requires ndjson output")
	}
	if globals.Format == "text" && globals.Quiet {
		return c.outputError(globals, "INVALID_FLAGS", "--quiet has no effect with text output; use ndjson for agents")
	}
	if globals.Format == "text" && globals.Config != nil && globals.Config.Quiet && !c.NoAgentHints && globals.Config.Format == "ndjson" {
		// no-op, just clarity
	}

	// Find the simulator
	mgr := simulator.NewManager()
	var device *domain.Device
	var err error

	if c.Simulator != "" {
		globals.Debug("Finding simulator by name/UDID: %s", c.Simulator)
		device, err = mgr.FindDevice(ctx, c.Simulator)
	} else {
		globals.Debug("Finding booted simulator (auto-detect)")
		device, err = mgr.FindBootedDevice(ctx)
	}
	if err != nil {
		return c.outputError(globals, "DEVICE_NOT_FOUND", err.Error())
	}
	globals.Debug("Found device: %s (UDID: %s, State: %s)", device.Name, device.UDID, device.State)

	// Fetch app version/build for metadata (best-effort)
	appVersion, appBuild := "", ""
	if v, b, err := mgr.GetAppInfo(ctx, device.UDID, c.App); err == nil {
		appVersion, appBuild = v, b
		globals.Debug("App info: version=%s build=%s", appVersion, appBuild)
	} else {
		globals.Debug("App info unavailable: %v", err)
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
				output.NewNDJSONWriter(globals.Stdout).WriteInfo(
					fmt.Sprintf("Writing logs to %s", path),
					device.Name, device.UDID, "", "")
			} else {
				fmt.Fprintf(globals.Stderr, "Writing logs to %s\n", path)
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
				rotator.Close()
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
			if !globals.Quiet {
				fmt.Fprintln(globals.Stderr, "Warning: tmux not installed, falling back to stdout")
			}
		} else {
			cfg := &tmux.Config{
				SessionName:   sessionName,
				SimulatorName: device.Name,
				Detached:      true,
			}

			tmuxMgr, err = tmux.NewManager(cfg)
			if err != nil {
				if !globals.Quiet {
					fmt.Fprintf(globals.Stderr, "Warning: failed to create tmux session: %v, falling back to stdout\n", err)
				}
			} else {
				if err := tmuxMgr.GetOrCreateSession(); err != nil {
					if !globals.Quiet {
						fmt.Fprintf(globals.Stderr, "Warning: failed to setup tmux session: %v, falling back to stdout\n", err)
					}
				} else {
					// Successfully created tmux session
					outputWriter = tmux.NewWriter(tmuxMgr)

					// Clear pane and show banner
					tmuxMgr.ClearPaneWithBanner(fmt.Sprintf("Watching: %s (%s)", device.Name, c.App))

					// Output attach command
					if globals.Format == "ndjson" {
						output.NewNDJSONWriter(globals.Stdout).WriteTmux(sessionName, tmuxMgr.AttachCommand())
					} else {
						fmt.Fprintf(globals.Stdout, "Tmux session: %s\n", sessionName)
						fmt.Fprintf(globals.Stdout, "Attach with: %s\n", tmuxMgr.AttachCommand())
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
			output.NewNDJSONWriter(globals.Stdout).WriteInfo(
				fmt.Sprintf("Streaming logs from %s (%s)", device.Name, device.UDID),
				device.Name, device.UDID, "", "")
		} else {
			fmt.Fprintf(globals.Stderr, "Streaming logs from %s (%s)\n", device.Name, device.UDID)
			fmt.Fprintf(globals.Stderr, "Filtering by app: %s\n", c.App)
			if c.Pattern != "" {
				fmt.Fprintf(globals.Stderr, "Pattern filter: %s\n", c.Pattern)
			}
			fmt.Fprintln(globals.Stderr, "Press Ctrl+C to stop")
		}
	}

	pattern, excludePatterns, whereFilter, err := buildFilters(c.Pattern, c.Exclude, c.Where)
	if err != nil {
		return c.outputError(globals, "INVALID_FILTER", err.Error())
	}

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
		return c.outputError(globals, "STREAM_FAILED", err.Error())
	}
	defer streamer.Stop()
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

	// Create session tracker for detecting app relaunches
	sessionTracker := session.NewTracker(c.App, device.Name, device.UDID, tailID, appVersion, appBuild)
	log = newAgentLogger(globals, tailID, sessionTracker.CurrentSession)
	globals.Debug("Session tracking enabled")

	// Emit metadata for agents
	if emitter != nil {
		emitter.Metadata(Version, Commit, "")
	}

	// Emit ready event when --wait-for-launch is used (signals log capture is active)
	if c.WaitForLaunch {
		if emitter != nil {
			emitter.Ready(
				time.Now().UTC().Format(time.RFC3339Nano),
				device.Name,
				device.UDID,
				c.App,
				tailID,
				sessionTracker.CurrentSession(),
			)
			if !c.NoAgentHints {
				emitter.AgentHints(tailID, sessionTracker.CurrentSession(), defaultHints())
			}
		} else {
			fmt.Fprintf(globals.Stderr, "Ready: log capture active for %s\n", c.App)
			if !c.NoAgentHints {
				fmt.Fprintf(globals.Stderr, "[XCW] agent_hints: follow latest session, match tail_id=%s\n", tailID)
			}
		}
	}

	// Parse summary interval
	var summaryTicker *time.Ticker
	if c.SummaryInterval != "" {
		interval, err := time.ParseDuration(c.SummaryInterval)
		if err != nil {
			return c.outputError(globals, "INVALID_INTERVAL", fmt.Sprintf("invalid summary interval: %s", err))
		}
		summaryTicker = time.NewTicker(interval)
		defer summaryTicker.Stop()
	}

	// Parse heartbeat interval
	var heartbeatTicker *time.Ticker
	if c.Heartbeat != "" {
		interval, err := time.ParseDuration(c.Heartbeat)
		if err != nil {
			return c.outputError(globals, "INVALID_HEARTBEAT", fmt.Sprintf("invalid heartbeat interval: %s", err))
		}
		heartbeatTicker = time.NewTicker(interval)
		defer heartbeatTicker.Stop()
	}

	// Max duration/logs cutoffs
	var cutoffTimer *time.Timer
	if c.MaxDuration != "" {
		dur, err := time.ParseDuration(c.MaxDuration)
		if err != nil {
			return c.outputError(globals, "INVALID_MAX_DURATION", fmt.Sprintf("invalid max duration: %s", err))
		}
		cutoffTimer = time.NewTimer(dur)
		defer cutoffTimer.Stop()
	}
	var maxLogs = c.MaxLogs

	// Track metrics for heartbeat
	startTime := time.Now()
	logsSinceLast := 0
	lastSeen := time.Now()
	totalLogs := 0

	// Parse idle timeout for session rollover
	var idleTimer *time.Timer
	var idleDuration time.Duration
	if c.SessionIdle != "" {
		var err error
		idleDuration, err = time.ParseDuration(c.SessionIdle)
		if err != nil {
			return c.outputError(globals, "INVALID_SESSION_IDLE", fmt.Sprintf("invalid session idle duration: %s", err))
		}
		idleTimer = time.NewTimer(idleDuration)
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
		hints := defaultHints()
		if globals.Format == "ndjson" {
			output.NewNDJSONWriter(globals.Stdout).WriteAgentHints(tailID, sessionTracker.CurrentSession(), hints)
		} else {
			fmt.Fprintf(globals.Stderr, "[XCW] agent_hints: %s (tail_id=%s, session=%d)\n", strings.Join(hints, "; "), tailID, sessionTracker.CurrentSession())
		}
	}

	// Process logs
	for {
		select {
		case <-ctx.Done():
			// Output final summary
			c.outputSummary(writer, streamer, tailID)
			if final := sessionTracker.GetFinalSummary(); final != nil {
				if emitter != nil {
					emitter.SessionEnd(final)
					emitter.ClearBuffer("session_end", tailID, final.Session)
					emitter.Cutoff("sigint", tailID, final.Session, totalLogs)
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
				// Session changed - emit events
				if sessionChange.EndSession != nil {
					// Output session end with summary
					if emitter != nil {
						emitter.SessionEnd(sessionChange.EndSession)
						emitter.ClearBuffer("session_end", tailID, sessionChange.EndSession.Session)
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
						fmt.Fprintf(globals.Stderr, "[XCW] 🚀 NEW SESSION: App relaunched (PID: %d)%s\n",
							sessionChange.StartSession.PID, prevSummary)
					}

					// Output JSON session start event
					if emitter != nil {
						emitter.SessionStart(sessionChange.StartSession)
						emitter.ClearBuffer("session_start", tailID, sessionChange.StartSession.Session)
						emitHints()
					}

					// Write tmux session banner if in tmux mode
					if tmuxMgr != nil && sessionChange.StartSession.Alert == "APP_RELAUNCHED" {
						var prevSummary *domain.SessionSummary
						if sessionChange.EndSession != nil {
							prevSummary = &sessionChange.EndSession.Summary
						}
						tmuxMgr.WriteSessionBanner(
							sessionChange.StartSession.Session,
							c.App,
							sessionChange.StartSession.PID,
							prevSummary,
						)
					}
				}
			}

			// Apply where filter if enabled
			if whereFilter != nil && !whereFilter.Match(&entry) {
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
			lastSeen = time.Now()
			if maxLogs > 0 && totalLogs >= maxLogs {
				if final := sessionTracker.GetFinalSummary(); final != nil && globals.Format == "ndjson" {
					emitter.SessionEnd(final)
					emitter.Cutoff("max_logs", tailID, final.Session, totalLogs)
					emitter.WriteHeartbeat(&output.Heartbeat{
						Type:              "heartbeat",
						SchemaVersion:     output.SchemaVersion,
						Timestamp:         time.Now().UTC().Format(time.RFC3339Nano),
						UptimeSeconds:     int64(time.Since(startTime).Seconds()),
						LogsSinceLast:     logsSinceLast,
						TailID:            tailID,
						LatestSession:     sessionTracker.CurrentSession(),
						LastSeenTimestamp: lastSeen.UTC().Format(time.RFC3339Nano),
					})
				}
				return nil
			}

		case err := <-streamer.Errors():
			if !globals.Quiet {
				if strings.HasPrefix(err.Error(), "reconnect_notice:") {
					if emitter != nil {
						emitter.WriteReconnect(err.Error(), tailID, "info")
					} else {
						fmt.Fprintf(globals.Stderr, "%s\n", err.Error())
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
			c.outputSummary(writer, streamer, tailID)

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
					emitter.SessionEnd(final)
					emitter.Cutoff("max_duration", tailID, final.Session, totalLogs)
					emitter.WriteHeartbeat(&output.Heartbeat{
						Type:              "heartbeat",
						SchemaVersion:     output.SchemaVersion,
						Timestamp:         time.Now().UTC().Format(time.RFC3339Nano),
						UptimeSeconds:     int64(time.Since(startTime).Seconds()),
						LogsSinceLast:     logsSinceLast,
						TailID:            tailID,
						LatestSession:     sessionTracker.CurrentSession(),
						LastSeenTimestamp: lastSeen.UTC().Format(time.RFC3339Nano),
					})
				}
				return nil
			}

			heartbeat := &output.Heartbeat{
				Type:              "heartbeat",
				SchemaVersion:     output.SchemaVersion,
				Timestamp:         time.Now().UTC().Format(time.RFC3339Nano),
				UptimeSeconds:     int64(time.Since(startTime).Seconds()),
				LogsSinceLast:     logsSinceLast,
				TailID:            tailID,
				LatestSession:     sessionTracker.CurrentSession(),
				LastSeenTimestamp: lastSeen.UTC().Format(time.RFC3339Nano),
			}
			writer.WriteHeartbeat(heartbeat)
			if log != nil {
				log.Debug("heartbeat logs_since_last=%d latest_session=%d", logsSinceLast, heartbeat.LatestSession)
			}
			logsSinceLast = 0

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
					if sessionChange.EndSession != nil && emitter != nil {
						emitter.SessionEnd(sessionChange.EndSession)
						emitter.ClearBuffer("session_end", tailID, sessionChange.EndSession.Session)
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
							emitter.SessionStart(sessionChange.StartSession)
							emitter.ClearBuffer("session_start", tailID, sessionChange.StartSession.Session)
							emitHints()
						}
						if tmuxMgr != nil {
							var prevSummary *domain.SessionSummary
							if sessionChange.EndSession != nil {
								prevSummary = &sessionChange.EndSession.Summary
							}
							tmuxMgr.WriteSessionBanner(
								sessionChange.StartSession.Session,
								c.App,
								sessionChange.StartSession.PID,
								prevSummary,
							)
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
}, streamer *simulator.Streamer, tailID string) {
	total, errors, faults := streamer.GetStats()
	summary := &domain.LogSummary{
		Type:       "summary",
		TotalCount: total,
		ErrorCount: errors,
		FaultCount: faults,
		HasErrors:  errors > 0,
		HasFaults:  faults > 0,
		WindowEnd:  time.Now(),
		TailID:     tailID,
	}
	writer.WriteSummary(summary)
}

func (c *TailCmd) outputError(globals *Globals, code, message string) error {
	return outputErrorCommon(globals, code, message)
}

func generateTailID() string {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("tail-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("tail-%s-%d", hex.EncodeToString(b[:]), time.Now().UnixNano())
}

func defaultHints() []string {
	return []string{
		"use latest session only",
		"match tail_id to current tail invocation",
		"reset caches on clear_buffer/session_start/session_end",
		"use newest rotated file unless comparing runs",
	}
}
