package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/vburojevic/xcw/internal/domain"
	"github.com/vburojevic/xcw/internal/output"
	"github.com/vburojevic/xcw/internal/simulator"
	"github.com/vburojevic/xcw/internal/tmux"
)

// WatchCmd watches logs and triggers commands on specific patterns
type WatchCmd struct {
	Simulator           string   `short:"s" help:"Simulator name or UDID"`
	Booted              bool     `short:"b" help:"Use booted simulator (error if multiple)"`
	App                 string   `short:"a" required:"" help:"App bundle identifier to filter logs"`
	Pattern             string   `short:"p" aliases:"filter" help:"Regex pattern to filter log messages"`
	Exclude             string   `short:"x" help:"Regex pattern to exclude from log messages"`
	ExcludeSubsystem    []string `help:"Exclude logs from subsystem (can be repeated, supports * wildcard)"`
	MinLevel            string   `help:"Minimum log level: debug, info, default, error, fault (overrides global --level)"`
	MaxLevel            string   `help:"Maximum log level: debug, info, default, error, fault"`
	Predicate           string   `help:"Raw NSPredicate filter (overrides --app)"`
	OnError             string   `help:"Command to run when error-level log detected"`
	OnFault             string   `help:"Command to run when fault-level log detected"`
	OnPattern           []string `help:"Pattern:command pairs (e.g., 'crash:notify.sh') - can be repeated"`
	Cooldown            string   `default:"5s" help:"Minimum time between trigger executions"`
	TriggerTimeout      string   `default:"30s" help:"Maximum time for trigger command execution"`
	MaxParallelTriggers int      `default:"5" help:"Maximum concurrent trigger executions"`
	TriggerOutput       string   `default:"discard" enum:"inherit,discard,capture" help:"Trigger command output handling"`
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

// Run executes the watch command
func (c *WatchCmd) Run(globals *Globals) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

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
	if c.Simulator != "" && c.Booted {
		return c.outputError(globals, "INVALID_FLAGS", "--simulator and --booted are mutually exclusive")
	}

	// Find the simulator
	mgr := simulator.NewManager()
	var device *domain.Device
	if c.Simulator != "" {
		device, err = mgr.FindDevice(ctx, c.Simulator)
	} else {
		device, err = mgr.FindBootedDevice(ctx)
	}
	if err != nil {
		return c.outputError(globals, "DEVICE_NOT_FOUND", err.Error())
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
		defer outputFile.Close()

		bufferedWriter = bufio.NewWriter(outputFile)
		defer bufferedWriter.Flush()

		outputWriter = bufferedWriter

		if !globals.Quiet {
			if globals.Format == "ndjson" {
				output.NewNDJSONWriter(globals.Stdout).WriteInfo(
					fmt.Sprintf("Writing logs to %s", outputPath),
					device.Name, device.UDID, "", "")
			} else {
				fmt.Fprintf(globals.Stderr, "Writing logs to %s\n", outputPath)
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
					tmuxMgr.ClearPaneWithBanner(fmt.Sprintf("Watching: %s (%s) [TRIGGER MODE]", device.Name, c.App))

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

	if tmuxMgr != nil {
		defer tmuxMgr.Cleanup()
	}

	// Output watch info
	if !globals.Quiet && tmuxMgr == nil {
		if globals.Format == "ndjson" {
			output.NewNDJSONWriter(globals.Stdout).WriteInfo(
				fmt.Sprintf("Watching logs from %s", device.Name),
				device.Name, device.UDID, "", "trigger")
		} else {
			fmt.Fprintf(globals.Stderr, "Watching logs from %s (%s)\n", device.Name, device.UDID)
			fmt.Fprintf(globals.Stderr, "App: %s\n", c.App)
			if c.OnError != "" {
				fmt.Fprintf(globals.Stderr, "On error: %s\n", c.OnError)
			}
			if c.OnFault != "" {
				fmt.Fprintf(globals.Stderr, "On fault: %s\n", c.OnFault)
			}
			for _, t := range triggers {
				fmt.Fprintf(globals.Stderr, "On pattern '%s': %s\n", t.pattern.String(), t.command)
			}
			fmt.Fprintf(globals.Stderr, "Cooldown: %s\n", c.Cooldown)
			fmt.Fprintln(globals.Stderr, "Press Ctrl+C to stop")
		}
	}

	// Compile pattern regex if provided
	var pattern *regexp.Regexp
	if c.Pattern != "" {
		pattern, err = regexp.Compile(c.Pattern)
		if err != nil {
			return c.outputError(globals, "INVALID_PATTERN", fmt.Sprintf("invalid regex pattern: %s", err))
		}
	}

	// Compile exclude pattern
	var excludePatterns []*regexp.Regexp
	if c.Exclude != "" {
		excludePattern, err := regexp.Compile(c.Exclude)
		if err != nil {
			return c.outputError(globals, "INVALID_EXCLUDE_PATTERN", fmt.Sprintf("invalid exclude pattern: %s", err))
		}
		excludePatterns = append(excludePatterns, excludePattern)
	}

	// Determine log level (command-specific overrides global)
	minLevel := globals.Level
	if c.MinLevel != "" {
		minLevel = c.MinLevel
	}

	// Create streamer
	streamer := simulator.NewStreamer(mgr)
	opts := simulator.StreamOptions{
		BundleID:          c.App,
		MinLevel:          domain.ParseLogLevel(minLevel),
		MaxLevel:          domain.ParseLogLevel(c.MaxLevel),
		Pattern:           pattern,
		ExcludePatterns:   excludePatterns,
		ExcludeSubsystems: c.ExcludeSubsystem,
		BufferSize:        100,
		RawPredicate:      c.Predicate,
	}

	if err := streamer.Start(ctx, device.UDID, opts); err != nil {
		return c.outputError(globals, "STREAM_FAILED", err.Error())
	}
	defer streamer.Stop()

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
			// Output the log entry
			if err := writer.Write(&entry); err != nil {
				return err
			}

			now := time.Now()

			// Check error trigger
			if c.OnError != "" && entry.Level == domain.LogLevelError {
				if now.Sub(lastErrorTrigger) >= cooldown {
					c.runTrigger(globals, "error", c.OnError, &entry, triggerTimeout, triggerSem, c.TriggerOutput)
					lastErrorTrigger = now
				}
			}

			// Check fault trigger
			if c.OnFault != "" && entry.Level == domain.LogLevelFault {
				if now.Sub(lastFaultTrigger) >= cooldown {
					c.runTrigger(globals, "fault", c.OnFault, &entry, triggerTimeout, triggerSem, c.TriggerOutput)
					lastFaultTrigger = now
				}
			}

			// Check pattern triggers
			for i, t := range triggers {
				if t.pattern.MatchString(entry.Message) {
					if now.Sub(lastPatternTriggers[i]) >= cooldown {
						c.runTrigger(globals, "pattern:"+t.pattern.String(), t.command, &entry, triggerTimeout, triggerSem, c.TriggerOutput)
						lastPatternTriggers[i] = now
					}
				}
			}

		case err := <-streamer.Errors():
			em := output.NewEmitter(outputWriter)
			emitWarning(globals, em, err.Error())
		}
	}
}

// runTrigger executes a trigger command with safety limits
func (c *WatchCmd) runTrigger(globals *Globals, triggerType, command string, entry *domain.LogEntry, timeout time.Duration, sem chan struct{}, outputMode string) {
	// Try to acquire semaphore (non-blocking)
	select {
	case sem <- struct{}{}:
		// Acquired
	default:
		// Too many parallel triggers running, skip this one
		if globals.Format == "ndjson" {
			output.NewEmitter(globals.Stdout).WriteWarning(fmt.Sprintf("trigger skipped (max parallel %d reached): %s", cap(sem), command))
		} else if !globals.Quiet {
			fmt.Fprintf(globals.Stderr, "[TRIGGER SKIPPED] Max parallel triggers reached: %s\n", command)
		}
		return
	}

	// Output trigger notification
	if globals.Format == "ndjson" {
		output.NewNDJSONWriter(globals.Stdout).WriteTrigger(triggerType, command, entry.Message)
	} else if !globals.Quiet {
		fmt.Fprintf(globals.Stderr, "[TRIGGER:%s] Running: %s\n", triggerType, command)
	}

	// Run command in background (don't block log processing)
	go func() {
		defer func() { <-sem }() // Release semaphore when done

		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// Set environment variables for the command
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Env = append(os.Environ(),
			"XCW_TRIGGER="+triggerType,
			"XCW_LEVEL="+string(entry.Level),
			"XCW_MESSAGE="+entry.Message,
			"XCW_SUBSYSTEM="+entry.Subsystem,
			"XCW_PROCESS="+entry.Process,
			"XCW_TIMESTAMP="+entry.Timestamp.Format(time.RFC3339),
		)

		// Handle output based on mode
		switch outputMode {
		case "inherit":
			cmd.Stdout = globals.Stdout
			cmd.Stderr = globals.Stderr
		case "capture":
			// Capture and log output
			out, err := cmd.CombinedOutput()
			if err != nil {
				errMsg := err.Error()
				if ctx.Err() == context.DeadlineExceeded {
					errMsg = fmt.Sprintf("timeout after %s", timeout)
				}
				if globals.Format == "ndjson" {
					output.NewNDJSONWriter(globals.Stdout).WriteTriggerError(command, errMsg)
				} else if !globals.Quiet {
					fmt.Fprintf(globals.Stderr, "[TRIGGER ERROR] %s: %s\n", command, errMsg)
				}
			}
			if len(out) > 0 && !globals.Quiet {
				if globals.Format == "ndjson" {
					output.NewNDJSONWriter(globals.Stdout).WriteInfo(
						fmt.Sprintf("trigger output: %s", strings.TrimSpace(string(out))),
						"", "", "", "")
				} else {
					fmt.Fprintf(globals.Stderr, "[TRIGGER OUTPUT] %s\n", strings.TrimSpace(string(out)))
				}
			}
			return
		default: // "discard"
			cmd.Stdout = nil
			cmd.Stderr = nil
		}

		if err := cmd.Run(); err != nil {
			errMsg := err.Error()
			if ctx.Err() == context.DeadlineExceeded {
				errMsg = fmt.Sprintf("timeout after %s", timeout)
			}
			if globals.Format == "ndjson" {
				output.NewNDJSONWriter(globals.Stdout).WriteTriggerError(command, errMsg)
			} else if !globals.Quiet {
				fmt.Fprintf(globals.Stderr, "[TRIGGER ERROR] %s: %s\n", command, errMsg)
			}
		}
	}()
}

func (c *WatchCmd) outputError(globals *Globals, code, message string) error {
	if globals.Format == "ndjson" {
		w := output.NewNDJSONWriter(globals.Stdout)
		w.WriteError(code, message)
	} else {
		fmt.Fprintf(globals.Stderr, "Error [%s]: %s\n", code, message)
	}
	return errors.New(message)
}
