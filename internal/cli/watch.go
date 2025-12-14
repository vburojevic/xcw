package cli

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
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
	"golang.org/x/sync/errgroup"
)

// WatchCmd watches logs and triggers commands on specific patterns
type WatchCmd struct {
	Simulator           string   `short:"s" help:"Simulator name or UDID"`
	Booted              bool     `short:"b" help:"Use booted simulator (error if multiple)"`
	App                 string   `short:"a" help:"App bundle identifier to filter logs (required unless --predicate or --all)"`
	All                 bool     `help:"Allow streaming without --app/--predicate (can be very noisy)"`
	Pattern             string   `short:"p" aliases:"filter" help:"Regex pattern to filter log messages"`
	Exclude             []string `short:"x" help:"Regex pattern to exclude from log messages (can be repeated)"`
	ExcludeSubsystem    []string `help:"Exclude logs from subsystem (can be repeated, supports * wildcard)"`
	MinLevel            string   `help:"Minimum log level: debug, info, default, error, fault (overrides global --level)"`
	MaxLevel            string   `help:"Maximum log level: debug, info, default, error, fault (optional; unset = no max)"`
	Where               []string `short:"w" help:"Field filter expression (supports AND/OR/NOT, parentheses). Operators: =, !=, ~, !~, >=, <=, ^, $. Regex literals: /pattern/i"`
	Dedupe              bool     `help:"Collapse repeated identical messages"`
	DedupeWindow        string   `help:"Time window for deduplication (e.g., '5s', '1m'). Without this, only consecutive duplicates are collapsed"`
	Process             []string `help:"Filter by process name (can be repeated)"`
	Predicate           string   `help:"Raw NSPredicate filter (overrides --app)"`
	OnError             string   `help:"Command to run when error-level log detected"`
	OnFault             string   `help:"Command to run when fault-level log detected"`
	OnPattern           []string `help:"Pattern:command pairs (e.g., 'crash:notify.sh') - can be repeated"`
	Cooldown            string   `default:"5s" help:"Minimum time between trigger executions"`
	TriggerTimeout      string   `default:"30s" help:"Maximum time for trigger command execution"`
	MaxParallelTriggers int      `default:"5" help:"Maximum concurrent trigger executions"`
	TriggerOutput       string   `default:"discard" enum:"inherit,discard,capture" help:"Trigger command output handling"`
	TriggerNoShell      bool     `help:"Run trigger commands directly without shell (safer). Command is split on spaces; no shell expansions."`
	DryRunJSON          bool     `help:"Print resolved stream options and triggers as JSON and exit (no streaming; ndjson output only)"`
	Output              string   `short:"o" help:"Write output to explicit file path"`
	SessionDir          string   `help:"Directory for session files (default: ~/.xcw/sessions)"`
	SessionPrefix       string   `help:"Prefix for session filename (default: app bundle ID)"`
	Tmux                bool     `help:"Output to tmux session"`
	Session             string   `help:"Custom tmux session name (default: xcw-<simulator>)"`
}

// triggerConfig holds parsed trigger configuration
type triggerConfig struct {
	pattern *regexp.Regexp
	command string
}

var (
	watchInfoStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("45"))
	watchWarnStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203"))
)

func maybeNoStyleWatch(globals *Globals) {
	if globals == nil {
		return
	}
	if globals.Stdout != nil {
		if f, ok := globals.Stdout.(*os.File); ok {
			if !isatty.IsTerminal(f.Fd()) {
				watchInfoStyle = watchInfoStyle.UnsetForeground().UnsetBold()
				watchWarnStyle = watchWarnStyle.UnsetForeground().UnsetBold()
			}
		}
	}
}

// Run executes the watch command
func (c *WatchCmd) Run(globals *Globals) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	maybeNoStyleWatch(globals)
	applyWatchDefaults(globals.Config, c)
	tailID := generateTailID()
	clk := clock.New()
	triggerGroup, triggerCtx := errgroup.WithContext(ctx)
	defer func() {
		stop()
		_ = triggerGroup.Wait()
	}()

	var stdoutMu sync.Mutex
	stdoutNDJSON := output.NewNDJSONWriter(globals.Stdout)
	writeStdout := func(fn func(w *output.NDJSONWriter) error) error {
		stdoutMu.Lock()
		defer stdoutMu.Unlock()
		return fn(stdoutNDJSON)
	}

	// Parse cooldown duration
	cooldown, err := time.ParseDuration(c.Cooldown)
	if err != nil {
		return c.outputError(globals, "INVALID_COOLDOWN", fmt.Sprintf("invalid cooldown duration: %s", err))
	}

	// Parse trigger timeout
	triggerTimeout, err := time.ParseDuration(c.TriggerTimeout)
	if err != nil {
		return c.outputError(globals, "INVALID_TRIGGER_TIMEOUT", fmt.Sprintf("invalid trigger timeout: %s", err))
	}

	// Create semaphore for limiting parallel triggers
	triggerSem := make(chan struct{}, c.MaxParallelTriggers)

	// Parse pattern triggers
	var triggers []triggerConfig
	for _, pt := range c.OnPattern {
		parts := strings.SplitN(pt, ":", 2)
		if len(parts) != 2 {
			return c.outputError(globals, "INVALID_TRIGGER", fmt.Sprintf("invalid pattern:command format: %s", pt))
		}
		re, err := regexp.Compile(parts[0])
		if err != nil {
			return c.outputError(globals, "INVALID_TRIGGER_PATTERN", fmt.Sprintf("invalid trigger pattern: %s", err))
		}
		triggers = append(triggers, triggerConfig{pattern: re, command: parts[1]})
	}

	// Validate mutual exclusivity of flags
	if globals.FlagProvided("simulator") && globals.FlagProvided("booted") {
		return c.outputError(globals, "INVALID_FLAGS", "--simulator and --booted are mutually exclusive")
	}
	if err := validateFlags(globals, c.DryRunJSON, c.Tmux); err != nil {
		return err
	}
	if err := validateAppPredicateAll(c.App, c.Predicate, c.All, false); err != nil {
		return outputErrorCommon(globals, err.Code, err.Message, err.Hint)
	}

	// Find the simulator
	mgr := simulator.NewManager()
	device, err := resolveSimulatorDevice(ctx, mgr, c.Simulator, c.Booted)
	if err != nil {
		return c.outputError(globals, "DEVICE_NOT_FOUND", err.Error(), hintForStreamOrQuery(err))
	}

	// Session tracking is only meaningful when watching an app.
	var sessionTracker tailSessionTracker
	if c.App != "" {
		sessionTracker = session.NewTracker(c.App, device.Name, device.UDID, tailID, "", "")
	} else {
		sessionTracker = &noopSessionTracker{}
	}

	// Compile filters (pattern, exclude, where unsupported here)
	pattern, excludePatterns, whereFilter, err := buildFilters(c.Pattern, c.Exclude, c.Where)
	if err != nil {
		return c.outputError(globals, "INVALID_FILTER", err.Error(), hintForFilter(err))
	}
	// Pattern/exclude are applied in the simulator streamer; keep pipeline for where-only filtering.
	pipeline := filter.NewPipeline(nil, nil, whereFilter)

	// Setup dedupe filter if enabled
	var dedupeFilter *filter.DedupeFilter
	if c.Dedupe {
		var dedupeWindow time.Duration
		if c.DedupeWindow != "" {
			dedupeWindow, err = time.ParseDuration(c.DedupeWindow)
			if err != nil {
				return c.outputError(globals, "INVALID_DEDUPE_WINDOW", fmt.Sprintf("invalid dedupe window: %s", err))
			}
		}
		dedupeFilter = filter.NewDedupeFilter(dedupeWindow)
	}

	// Determine log level (command-specific overrides global)
	minLevel, maxLevel := resolveLevels(c.MinLevel, c.MaxLevel, globals.Level)

	// Create streamer
	streamer := simulator.NewStreamer(mgr)
	opts := simulator.StreamOptions{
		BundleID:          c.App,
		MinLevel:          minLevel,
		MaxLevel:          maxLevel,
		Pattern:           pattern,
		ExcludePatterns:   excludePatterns,
		ExcludeSubsystems: c.ExcludeSubsystem,
		Processes:         c.Process,
		BufferSize:        100,
		RawPredicate:      c.Predicate,
		Verbose:           globals.Verbose,
	}

	if c.DryRunJSON {
		if globals.Format != "ndjson" {
			return c.outputError(globals, "INVALID_FLAGS", "--dry-run-json requires ndjson output", "add --format ndjson or remove --dry-run-json")
		}
		enc := json.NewEncoder(globals.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(struct {
			Stream              simulator.StreamOptions `json:"stream"`
			Cooldown            string                  `json:"cooldown"`
			TriggerTimeout      string                  `json:"trigger_timeout"`
			MaxParallelTriggers int                     `json:"max_parallel_triggers"`
			TriggerOutput       string                  `json:"trigger_output"`
			TriggerNoShell      bool                    `json:"trigger_no_shell"`
			OnError             string                  `json:"on_error,omitempty"`
			OnFault             string                  `json:"on_fault,omitempty"`
			OnPattern           []string                `json:"on_pattern,omitempty"`
		}{
			Stream:              opts,
			Cooldown:            c.Cooldown,
			TriggerTimeout:      c.TriggerTimeout,
			MaxParallelTriggers: c.MaxParallelTriggers,
			TriggerOutput:       c.TriggerOutput,
			TriggerNoShell:      c.TriggerNoShell,
			OnError:             c.OnError,
			OnFault:             c.OnFault,
			OnPattern:           c.OnPattern,
		})
	}

	// Determine output destination
	var outputWriter io.Writer = globals.Stdout
	var tmuxMgr *tmux.Manager
	var outputFile *os.File
	var bufferedWriter *bufio.Writer

	// Determine output file path
	var outputPath string
	if c.Output != "" {
		// Explicit --output overrides session behavior
		outputPath = c.Output
	} else if c.SessionDir != "" || c.SessionPrefix != "" {
		// Session-based file output
		prefix := c.SessionPrefix
		if prefix == "" {
			prefix = c.App
		}
		path, err := GenerateSessionPath(c.SessionDir, prefix)
		if err != nil {
			return c.outputError(globals, "SESSION_DIR_ERROR", err.Error())
		}
		outputPath = path
	}

	// Create file output if path is set
	if outputPath != "" {
		var err error
		outputFile, err = os.Create(outputPath)
		if err != nil {
			return c.outputError(globals, "FILE_CREATE_ERROR", fmt.Sprintf("failed to create output file: %s", err))
		}
		defer func() {
			if err := outputFile.Close(); err != nil {
				globals.Debug("failed to close output file: %v", err)
			}
		}()

		bufferedWriter = bufio.NewWriter(outputFile)
		defer func() {
			if err := bufferedWriter.Flush(); err != nil {
				globals.Debug("failed to flush output buffer: %v", err)
			}
		}()

		outputWriter = bufferedWriter

		if !globals.Quiet {
			if globals.Format == "ndjson" {
				if err := output.NewNDJSONWriter(globals.Stdout).WriteInfo(
					fmt.Sprintf("Writing logs to %s", outputPath),
					device.Name, device.UDID, "", ""); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(globals.Stderr, "Writing logs to %s\n", outputPath); err != nil {
					globals.Debug("failed to write output path: %v", err)
				}
			}
		}
	}

	if c.Tmux {
		sessionName := c.Session
		if sessionName == "" {
			sessionName = tmux.GenerateSessionName(device.Name)
		}

		if tmux.IsTmuxAvailable() {
			cfg := &tmux.Config{
				SessionName:   sessionName,
				SimulatorName: device.Name,
				Detached:      true,
			}

			tmuxMgr, err = tmux.NewManager(cfg)
			if err == nil {
				if err := tmuxMgr.GetOrCreateSession(); err == nil {
					outputWriter = tmux.NewWriter(tmuxMgr)
					if err := tmuxMgr.ClearPaneWithBanner(fmt.Sprintf("Watching: %s (%s) [TRIGGER MODE]", device.Name, c.App)); err != nil {
						emitWarning(globals, output.NewEmitter(globals.Stdout), fmt.Sprintf("failed to clear tmux pane: %v", err))
					}

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

	if tmuxMgr != nil {
		defer tmuxMgr.Cleanup()
	}

	// Output watch info
	if !globals.Quiet && tmuxMgr == nil {
		if globals.Format == "ndjson" {
			if err := output.NewNDJSONWriter(globals.Stdout).WriteInfo(
				fmt.Sprintf("Watching logs from %s", device.Name),
				device.Name, device.UDID, "", "trigger"); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(globals.Stderr, "%s\n", watchInfoStyle.Render(fmt.Sprintf("Watching logs from %s (%s)", device.Name, device.UDID))); err != nil {
				globals.Debug("failed to write watch info: %v", err)
			}
			if _, err := fmt.Fprintf(globals.Stderr, "App: %s\n", c.App); err != nil {
				globals.Debug("failed to write watch info: %v", err)
			}
			if c.OnError != "" {
				if _, err := fmt.Fprintf(globals.Stderr, "%s\n", watchWarnStyle.Render(fmt.Sprintf("On error: %s", c.OnError))); err != nil {
					globals.Debug("failed to write watch info: %v", err)
				}
			}
			if c.OnFault != "" {
				if _, err := fmt.Fprintf(globals.Stderr, "%s\n", watchWarnStyle.Render(fmt.Sprintf("On fault: %s", c.OnFault))); err != nil {
					globals.Debug("failed to write watch info: %v", err)
				}
			}
			for _, t := range triggers {
				if _, err := fmt.Fprintf(globals.Stderr, "On pattern '%s': %s\n", t.pattern.String(), t.command); err != nil {
					globals.Debug("failed to write watch info: %v", err)
				}
			}
			if _, err := fmt.Fprintf(globals.Stderr, "Cooldown: %s\n", c.Cooldown); err != nil {
				globals.Debug("failed to write watch info: %v", err)
			}
			if _, err := fmt.Fprintln(globals.Stderr, "Press Ctrl+C to stop"); err != nil {
				globals.Debug("failed to write watch info: %v", err)
			}
		}
	}

	if err := streamer.Start(ctx, device.UDID, opts); err != nil {
		return c.outputError(globals, "STREAM_FAILED", err.Error(), hintForStreamOrQuery(err))
	}
	defer func() {
		if err := streamer.Stop(); err != nil {
			globals.Debug("failed to stop streamer: %v", err)
		}
	}()

	// Track last trigger times for cooldown
	lastErrorTrigger := time.Time{}
	lastFaultTrigger := time.Time{}
	lastPatternTriggers := make(map[int]time.Time)

	// Create output writer
	var writer interface {
		Write(entry *domain.LogEntry) error
	}

	if globals.Format == "ndjson" {
		writer = output.NewNDJSONWriter(outputWriter)
	} else {
		writer = output.NewTextWriter(outputWriter)
	}

	// Process logs
	for {
		select {
		case <-ctx.Done():
			return nil

		case entry := <-streamer.Logs():
			// Apply where filtering (post-stream)
			if pipeline != nil && !pipeline.Match(&entry) {
				continue
			}

			// Apply session tracking and annotate entries for correlation
			_ = sessionTracker.CheckEntry(&entry)
			entry.Session = sessionTracker.CurrentSession()
			entry.TailID = tailID

			// Apply deduplication
			if dedupeFilter != nil {
				result := dedupeFilter.Check(&entry)
				if !result.ShouldEmit {
					continue
				}
			}

			// Output the log entry
			if globals.Format == "ndjson" && outputWriter == globals.Stdout {
				if err := writeStdout(func(w *output.NDJSONWriter) error { return w.Write(&entry) }); err != nil {
					return err
				}
			} else {
				if err := writer.Write(&entry); err != nil {
					return err
				}
			}

			now := clk.Now()

			// Check error trigger
			if c.OnError != "" && entry.Level == domain.LogLevelError {
				if now.Sub(lastErrorTrigger) >= cooldown {
					c.runTrigger(triggerCtx, triggerGroup, globals, writeStdout, "error", c.OnError, entry, triggerTimeout, triggerSem, c.TriggerOutput)
					lastErrorTrigger = now
				}
			}

			// Check fault trigger
			if c.OnFault != "" && entry.Level == domain.LogLevelFault {
				if now.Sub(lastFaultTrigger) >= cooldown {
					c.runTrigger(triggerCtx, triggerGroup, globals, writeStdout, "fault", c.OnFault, entry, triggerTimeout, triggerSem, c.TriggerOutput)
					lastFaultTrigger = now
				}
			}

			// Check pattern triggers
			for i, t := range triggers {
				if t.pattern.MatchString(entry.Message) {
					if now.Sub(lastPatternTriggers[i]) >= cooldown {
						c.runTrigger(triggerCtx, triggerGroup, globals, writeStdout, "pattern:"+t.pattern.String(), t.command, entry, triggerTimeout, triggerSem, c.TriggerOutput)
						lastPatternTriggers[i] = now
					}
				}
			}

		case err := <-streamer.Errors():
			if globals.Format == "ndjson" && outputWriter == globals.Stdout {
				if werr := writeStdout(func(w *output.NDJSONWriter) error { return w.WriteWarning(err.Error()) }); werr != nil {
					return werr
				}
				continue
			}
			em := output.NewEmitter(outputWriter)
			emitWarning(globals, em, err.Error())
		}
	}
}

// runTrigger executes a trigger command with safety limits
func (c *WatchCmd) runTrigger(ctx context.Context, group *errgroup.Group, globals *Globals, writeStdout func(fn func(w *output.NDJSONWriter) error) error, triggerType, command string, entry domain.LogEntry, timeout time.Duration, sem chan struct{}, outputMode string) {
	// Try to acquire semaphore (non-blocking)
	select {
	case sem <- struct{}{}:
		// Acquired
	default:
		// Too many parallel triggers running, skip this one
		if globals.Format == "ndjson" {
			if err := writeStdout(func(w *output.NDJSONWriter) error {
				return w.WriteWarning(fmt.Sprintf("trigger skipped (max parallel %d reached): %s", cap(sem), command))
			}); err != nil {
				globals.Debug("failed to write trigger warning: %v", err)
			}
		} else if !globals.Quiet {
			if _, err := fmt.Fprintf(globals.Stderr, "[TRIGGER SKIPPED] Max parallel triggers reached: %s\n", command); err != nil {
				globals.Debug("failed to write trigger warning: %v", err)
			}
		}
		return
	}

	triggerID := generateTriggerID()
	triggerTimestamp := time.Now().UTC().Format(time.RFC3339Nano)

	// Output trigger notification
	if globals.Format == "ndjson" {
		if err := writeStdout(func(w *output.NDJSONWriter) error {
			return w.WriteTrigger(&output.TriggerOutput{
				Type:          "trigger",
				Timestamp:     triggerTimestamp,
				TailID:        entry.TailID,
				Session:       entry.Session,
				TriggerID:     triggerID,
				Trigger:       triggerType,
				Command:       command,
				Message:       entry.Message,
				SchemaVersion: 0,
			})
		}); err != nil {
			globals.Debug("failed to write trigger: %v", err)
		}
	} else if !globals.Quiet {
		if _, err := fmt.Fprintf(globals.Stderr, "[TRIGGER:%s] Running: %s\n", triggerType, command); err != nil {
			globals.Debug("failed to write trigger: %v", err)
		}
	}

	// Run command in background (don't block log processing)
	group.Go(func() error {
		defer func() { <-sem }() // Release semaphore when done

		start := time.Now()

		// Create context with timeout (and cancel on parent ctx)
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// Build command (shell or direct exec)
		var cmd *exec.Cmd
		if c.TriggerNoShell {
			argv := strings.Fields(command)
			if len(argv) == 0 {
				c.emitTriggerFailure(globals, writeStdout, triggerID, triggerType, command, entry, triggerTimestamp, "empty trigger command", -1, 0, false, "")
				return nil
			}
			cmd = exec.CommandContext(ctx, argv[0], argv[1:]...)
		} else {
			cmd = exec.CommandContext(ctx, "sh", "-c", command)
		}

		// Set environment variables for the command
		cmd.Env = append(os.Environ(),
			"XCW_TRIGGER="+triggerType,
			"XCW_LEVEL="+string(entry.Level),
			"XCW_MESSAGE="+entry.Message,
			"XCW_SUBSYSTEM="+entry.Subsystem,
			"XCW_PROCESS="+entry.Process,
			"XCW_TIMESTAMP="+entry.Timestamp.Format(time.RFC3339),
		)

		// Handle output based on mode
		var out []byte
		var err error
		switch outputMode {
		case "inherit":
			cmd.Stdout = globals.Stdout
			cmd.Stderr = globals.Stderr
			err = cmd.Run()
		case "capture":
			out, err = cmd.CombinedOutput()
		default: // "discard"
			cmd.Stdout = nil
			cmd.Stderr = nil
			err = cmd.Run()
		}

		durationMs := time.Since(start).Milliseconds()
		timedOut := ctx.Err() == context.DeadlineExceeded

		exitCode := 0
		if err != nil {
			exitCode = exitCodeFromError(err)
		}

		outStr := strings.TrimSpace(string(out))
		const maxTriggerOutputBytes = 16 * 1024
		if len(outStr) > maxTriggerOutputBytes {
			outStr = outStr[:maxTriggerOutputBytes]
		}

		if err != nil {
			errMsg := err.Error()
			if timedOut {
				errMsg = fmt.Sprintf("timeout after %s", timeout)
			}
			c.emitTriggerFailure(globals, writeStdout, triggerID, triggerType, command, entry, triggerTimestamp, errMsg, exitCode, durationMs, timedOut, outStr)
			return nil
		}

		switch outputMode {
		case "capture":
			// Include captured output on success as well.
			_ = c.emitTriggerResult(globals, writeStdout, triggerID, triggerType, command, entry, triggerTimestamp, exitCode, durationMs, timedOut, outStr, "")
		default:
			_ = c.emitTriggerResult(globals, writeStdout, triggerID, triggerType, command, entry, triggerTimestamp, exitCode, durationMs, timedOut, "", "")
		}
		return nil
	})
}

func exitCodeFromError(err error) int {
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return -1
}

func (c *WatchCmd) emitTriggerFailure(globals *Globals, writeStdout func(fn func(w *output.NDJSONWriter) error) error, triggerID, triggerType, command string, entry domain.LogEntry, timestamp string, errMsg string, exitCode int, durationMs int64, timedOut bool, out string) {
	if globals.Format == "ndjson" {
		_ = c.emitTriggerResult(globals, writeStdout, triggerID, triggerType, command, entry, timestamp, exitCode, durationMs, timedOut, out, errMsg)
		_ = writeStdout(func(w *output.NDJSONWriter) error {
			return w.WriteTriggerError(&output.TriggerErrorOutput{
				Type:      "trigger_error",
				Timestamp: timestamp,
				TailID:    entry.TailID,
				Session:   entry.Session,
				TriggerID: triggerID,
				Trigger:   triggerType,
				Command:   command,
				Error:     errMsg,
			})
		})
		return
	}
	if globals.Quiet {
		return
	}
	if _, err := fmt.Fprintf(globals.Stderr, "[TRIGGER ERROR] %s: %s\n", command, errMsg); err != nil {
		globals.Debug("failed to write trigger error: %v", err)
	}
}

func (c *WatchCmd) emitTriggerResult(globals *Globals, writeStdout func(fn func(w *output.NDJSONWriter) error) error, triggerID, triggerType, command string, entry domain.LogEntry, timestamp string, exitCode int, durationMs int64, timedOut bool, out string, errMsg string) error {
	if globals.Format != "ndjson" {
		return nil
	}
	return writeStdout(func(w *output.NDJSONWriter) error {
		return w.WriteTriggerResult(&output.TriggerResultOutput{
			Type:       "trigger_result",
			Timestamp:  timestamp,
			TailID:     entry.TailID,
			Session:    entry.Session,
			TriggerID:  triggerID,
			Trigger:    triggerType,
			Command:    command,
			ExitCode:   exitCode,
			DurationMs: durationMs,
			TimedOut:   timedOut,
			Output:     out,
			Error:      errMsg,
		})
	})
}

func generateTriggerID() string {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("trigger-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("trigger-%s-%d", hex.EncodeToString(b[:]), time.Now().UnixNano())
}

func (c *WatchCmd) outputError(globals *Globals, code, message string, hint ...string) error {
	return outputErrorCommon(globals, code, message, hint...)
}

func applyWatchDefaults(cfg *config.Config, c *WatchCmd) {
	if cfg == nil {
		return
	}
	if c.Simulator == "" {
		if cfg.Watch.Simulator != "" {
			c.Simulator = cfg.Watch.Simulator
		} else if cfg.Defaults.Simulator != "" {
			c.Simulator = cfg.Defaults.Simulator
		}
	}
	if c.App == "" && c.Predicate == "" && cfg.Watch.App != "" {
		c.App = cfg.Watch.App
	}
	if c.Cooldown == "5s" && cfg.Watch.Cooldown != "" {
		c.Cooldown = cfg.Watch.Cooldown
	}
}
