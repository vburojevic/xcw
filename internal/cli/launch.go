package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/vburojevic/xcw/internal/domain"
	"github.com/vburojevic/xcw/internal/output"
	"github.com/vburojevic/xcw/internal/simulator"
	"golang.org/x/sync/errgroup"
)

// LaunchCmd launches an app and captures stdout/stderr (including print statements)
type LaunchCmd struct {
	Simulator         string `short:"s" help:"Simulator name or UDID"`
	Booted            bool   `short:"b" help:"Use booted simulator (error if multiple)"`
	App               string `short:"a" required:"" help:"App bundle identifier to launch"`
	TerminateExisting bool   `help:"Terminate any running instance of the app first"`
	Wait              bool   `short:"w" help:"Wait for debugger to attach before launching"`
}

// ConsoleOutput represents a line of console output
type ConsoleOutput struct {
	Type          string `json:"type"`
	SchemaVersion int    `json:"schemaVersion"`
	Timestamp     string `json:"timestamp"`
	Stream        string `json:"stream"` // "stdout" or "stderr"
	Message       string `json:"message"`
	Process       string `json:"process,omitempty"`
	PID           int    `json:"pid,omitempty"`
}

// Run executes the launch command
func (c *LaunchCmd) Run(globals *Globals) error {
	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	group, ctx := errgroup.WithContext(signalCtx)

	// Validate mutual exclusivity of flags
	if globals.FlagProvided("simulator") && globals.FlagProvided("booted") {
		return c.outputError(globals, "INVALID_FLAGS", "--simulator and --booted are mutually exclusive")
	}

	// Find the simulator
	mgr := simulator.NewManager()
	device, err := resolveSimulatorDevice(ctx, mgr, c.Simulator, c.Booted)
	if err != nil {
		return c.outputError(globals, "DEVICE_NOT_FOUND", err.Error())
	}
	globals.Debug("Found device: %s (UDID: %s, State: %s)", device.Name, device.UDID, device.State)

	// Check device is booted
	if device.State != "Booted" {
		return c.outputError(globals, "DEVICE_NOT_BOOTED", fmt.Sprintf("simulator %s is not booted", device.Name))
	}

	// Output info message
	if !globals.Quiet {
		c.outputInfo(globals, device)
	}

	// Build the launch command
	args := []string{"simctl", "launch", "--console"}
	if c.TerminateExisting {
		args = append(args, "--terminate-running-process")
	}
	if c.Wait {
		args = append(args, "--wait-for-debugger")
	}
	args = append(args, device.UDID, c.App)

	globals.Debug("Running: xcrun %v", args)

	// Create the command
	cmd := exec.CommandContext(ctx, "xcrun", args...)

	// Capture both stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return c.outputError(globals, "LAUNCH_FAILED", fmt.Sprintf("failed to create stdout pipe: %v", err))
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return c.outputError(globals, "LAUNCH_FAILED", fmt.Sprintf("failed to create stderr pipe: %v", err))
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return c.outputError(globals, "LAUNCH_FAILED", fmt.Sprintf("failed to launch app: %v", err))
	}

	// Read stdout in background
	group.Go(func() error {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			c.outputConsoleLine(globals, "stdout", scanner.Text(), c.App)
		}
		if err := scanner.Err(); err != nil {
			if errors.Is(err, bufio.ErrTooLong) {
				return fmt.Errorf("stdout line too long (>1MiB): %w", err)
			}
			return fmt.Errorf("stdout read error: %w", err)
		}
		return nil
	})

	// Read stderr in background
	group.Go(func() error {
		scanner := bufio.NewScanner(stderr)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			c.outputConsoleLine(globals, "stderr", scanner.Text(), c.App)
		}
		if err := scanner.Err(); err != nil {
			if errors.Is(err, bufio.ErrTooLong) {
				return fmt.Errorf("stderr line too long (>1MiB): %w", err)
			}
			return fmt.Errorf("stderr read error: %w", err)
		}
		return nil
	})

	// Wait for the command to finish and drain output.
	waitErr := cmd.Wait()
	scanErr := group.Wait()

	if signalCtx.Err() != nil {
		// Context was cancelled (signal received)
		return nil
	}
	if waitErr != nil {
		// App exited with error
		globals.Debug("App exited with error: %v", waitErr)
	}
	if scanErr != nil {
		emitWarning(globals, output.NewEmitter(globals.Stdout), "launch output reader error: "+scanErr.Error())
	}

	return nil
}

func (c *LaunchCmd) outputInfo(globals *Globals, device *domain.Device) {
	if globals.Format == "ndjson" {
		info := output.InfoOutput{
			Type:          "info",
			SchemaVersion: output.SchemaVersion,
			Message:       fmt.Sprintf("Launching %s on %s", c.App, device.Name),
			Simulator:     device.Name,
			UDID:          device.UDID,
		}
		enc := json.NewEncoder(globals.Stdout)
		if err := enc.Encode(info); err != nil {
			globals.Debug("failed to write launch info: %v", err)
		}
	} else {
		if _, err := fmt.Fprintf(globals.Stdout, "Launching %s on %s...\n", c.App, device.Name); err != nil {
			globals.Debug("failed to write launch info: %v", err)
		}
	}
}

func (c *LaunchCmd) outputConsoleLine(globals *Globals, stream, message, process string) {
	if globals.Format == "ndjson" {
		out := ConsoleOutput{
			Type:          "console",
			SchemaVersion: output.SchemaVersion,
			Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
			Stream:        stream,
			Message:       message,
			Process:       process,
		}
		enc := json.NewEncoder(globals.Stdout)
		if err := enc.Encode(out); err != nil {
			globals.Debug("failed to write console output: %v", err)
		}
	} else {
		prefix := ""
		if stream == "stderr" {
			prefix = "[stderr] "
		}
		if _, err := fmt.Fprintf(globals.Stdout, "%s%s\n", prefix, message); err != nil {
			globals.Debug("failed to write console output: %v", err)
		}
	}
}

func (c *LaunchCmd) outputError(globals *Globals, code, message string) error {
	return outputErrorCommon(globals, code, message)
}
