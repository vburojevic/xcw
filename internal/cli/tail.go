package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/vedranburojevic/xcw/internal/domain"
	"github.com/vedranburojevic/xcw/internal/output"
	"github.com/vedranburojevic/xcw/internal/simulator"
	"github.com/vedranburojevic/xcw/internal/tmux"
)

// TailCmd streams real-time logs from a simulator
type TailCmd struct {
	Simulator       string   `short:"s" default:"booted" help:"Simulator name, UDID, or 'booted' for auto-detect"`
	App             string   `short:"a" required:"" help:"App bundle identifier to filter logs"`
	Pattern         string   `short:"p" help:"Regex pattern to filter log messages"`
	Subsystem       []string `help:"Filter by subsystem (can be repeated)"`
	Category        []string `help:"Filter by category (can be repeated)"`
	BufferSize      int      `default:"100" help:"Number of recent logs to buffer"`
	SummaryInterval string   `help:"Emit periodic summaries (e.g., '30s', '1m')"`
	Tmux            bool     `help:"Output to tmux session"`
	Session         string   `help:"Custom tmux session name (default: xcw-<simulator>)"`
}

// Run executes the tail command
func (c *TailCmd) Run(globals *Globals) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Find the simulator
	mgr := simulator.NewManager()
	device, err := mgr.FindDevice(ctx, c.Simulator)
	if err != nil {
		return c.outputError(globals, "DEVICE_NOT_FOUND", err.Error())
	}

	// Determine output destination
	var outputWriter io.Writer = globals.Stdout
	var tmuxMgr *tmux.Manager

	if c.Tmux {
		// Setup tmux session
		sessionName := c.Session
		if sessionName == "" {
			sessionName = tmux.GenerateSessionName(device.Name)
		}

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
						fmt.Fprintf(globals.Stdout, `{"type":"tmux","session":"%s","attach":"%s"}`+"\n",
							sessionName, tmuxMgr.AttachCommand())
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
			fmt.Fprintf(globals.Stdout, `{"type":"info","message":"Streaming logs from %s (%s)","simulator":"%s","udid":"%s"}`+"\n",
				device.Name, device.UDID, device.Name, device.UDID)
		} else {
			fmt.Fprintf(globals.Stderr, "Streaming logs from %s (%s)\n", device.Name, device.UDID)
			fmt.Fprintf(globals.Stderr, "Filtering by app: %s\n", c.App)
			if c.Pattern != "" {
				fmt.Fprintf(globals.Stderr, "Pattern filter: %s\n", c.Pattern)
			}
			fmt.Fprintln(globals.Stderr, "Press Ctrl+C to stop\n")
		}
	}

	// Compile pattern regex if provided
	var pattern *regexp.Regexp
	if c.Pattern != "" {
		var err error
		pattern, err = regexp.Compile(c.Pattern)
		if err != nil {
			return c.outputError(globals, "INVALID_PATTERN", fmt.Sprintf("invalid regex pattern: %s", err))
		}
	}

	// Create streamer
	streamer := simulator.NewStreamer(mgr)
	opts := simulator.StreamOptions{
		BundleID:   c.App,
		Subsystems: c.Subsystem,
		Categories: c.Category,
		MinLevel:   domain.ParseLogLevel(globals.Level),
		Pattern:    pattern,
		BufferSize: c.BufferSize,
	}

	if err := streamer.Start(ctx, device.UDID, opts); err != nil {
		return c.outputError(globals, "STREAM_FAILED", err.Error())
	}
	defer streamer.Stop()

	// Create output writer based on format
	var writer interface {
		Write(entry *domain.LogEntry) error
		WriteSummary(summary *domain.LogSummary) error
	}

	if globals.Format == "ndjson" {
		writer = output.NewNDJSONWriter(outputWriter)
	} else {
		writer = output.NewTextWriter(outputWriter)
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

	// Process logs
	for {
		select {
		case <-ctx.Done():
			// Output final summary
			c.outputSummary(writer, streamer)
			return nil

		case entry := <-streamer.Logs():
			if err := writer.Write(&entry); err != nil {
				return err
			}

		case err := <-streamer.Errors():
			if !globals.Quiet {
				if globals.Format == "ndjson" {
					fmt.Fprintf(outputWriter, `{"type":"warning","message":"%s"}`+"\n", err.Error())
				} else {
					fmt.Fprintf(globals.Stderr, "Warning: %s\n", err.Error())
				}
			}

		case <-func() <-chan time.Time {
			if summaryTicker != nil {
				return summaryTicker.C
			}
			return nil
		}():
			c.outputSummary(writer, streamer)
		}
	}
}

func (c *TailCmd) outputSummary(writer interface{ WriteSummary(*domain.LogSummary) error }, streamer *simulator.Streamer) {
	total, errors, faults := streamer.GetStats()
	summary := &domain.LogSummary{
		Type:       "summary",
		TotalCount: total,
		ErrorCount: errors,
		FaultCount: faults,
		HasErrors:  errors > 0,
		HasFaults:  faults > 0,
		WindowEnd:  time.Now(),
	}
	writer.WriteSummary(summary)
}

func (c *TailCmd) outputError(globals *Globals, code, message string) error {
	if globals.Format == "ndjson" {
		w := output.NewNDJSONWriter(globals.Stdout)
		w.WriteError(code, message)
	} else {
		fmt.Fprintf(globals.Stderr, "Error [%s]: %s\n", code, message)
	}
	return fmt.Errorf(message)
}
