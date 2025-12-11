package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vburojevic/xcw/internal/domain"
	"github.com/vburojevic/xcw/internal/output"
)

// ReplayCmd replays a recorded NDJSON log file
type ReplayCmd struct {
	File     string  `arg:"" required:"" help:"NDJSON log file to replay"`
	Realtime bool    `help:"Replay with original timing (sleep between entries)"`
	Speed    float64 `default:"1.0" help:"Playback speed multiplier (e.g., 2.0 for 2x speed)"`
	Follow   bool    `help:"Follow file for new entries (like tail -f)"`
}

// Run executes the replay command
func (c *ReplayCmd) Run(globals *Globals) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Open input file
	file, err := os.Open(c.File)
	if err != nil {
		return c.outputError(globals, "FILE_NOT_FOUND", fmt.Sprintf("cannot open file: %s", err))
	}
	defer file.Close()

	// Create output writer
	var writer interface {
		Write(entry *domain.LogEntry) error
	}

	if globals.Format == "ndjson" {
		writer = output.NewNDJSONWriter(globals.Stdout)
	} else {
		writer = output.NewTextWriter(globals.Stdout)
	}

	if !globals.Quiet {
		if globals.Format == "ndjson" {
			output.NewNDJSONWriter(globals.Stdout).WriteInfo(
				fmt.Sprintf("Replaying logs from %s", c.File),
				"", "", "", "replay")
		} else {
			fmt.Fprintf(globals.Stderr, "Replaying logs from %s\n", c.File)
			if c.Realtime {
				fmt.Fprintf(globals.Stderr, "Speed: %.1fx\n", c.Speed)
			}
			fmt.Fprintln(globals.Stderr, "Press Ctrl+C to stop")
		}
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var lastTimestamp time.Time
	entryCount := 0

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if !scanner.Scan() {
			if c.Follow {
				// Wait for more data
				time.Sleep(100 * time.Millisecond)
				continue
			}
			break
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry domain.LogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			// Check if it's a different message type
			var typeCheck struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(line, &typeCheck) == nil && typeCheck.Type != "" {
				// Replay non-log entries too
				globals.Stdout.Write(line)
				globals.Stdout.Write([]byte("\n"))
				continue
			}
			globals.Debug("Skipping unparseable line: %v", err)
			continue
		}

		// Skip if no timestamp
		if entry.Timestamp.IsZero() {
			continue
		}

		// Apply realtime delay if enabled
		if c.Realtime && !lastTimestamp.IsZero() {
			delay := entry.Timestamp.Sub(lastTimestamp)
			if delay > 0 {
				adjustedDelay := time.Duration(float64(delay) / c.Speed)
				// Cap max delay at 5 seconds to avoid long waits
				if adjustedDelay > 5*time.Second {
					adjustedDelay = 5 * time.Second
				}
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(adjustedDelay):
				}
			}
		}
		lastTimestamp = entry.Timestamp

		if err := writer.Write(&entry); err != nil {
			return err
		}
		entryCount++
	}

	if err := scanner.Err(); err != nil {
		return c.outputError(globals, "READ_ERROR", fmt.Sprintf("error reading file: %s", err))
	}

	if !globals.Quiet && globals.Format != "ndjson" {
		fmt.Fprintf(globals.Stderr, "\nReplayed %d entries\n", entryCount)
	}

	return nil
}

func (c *ReplayCmd) outputError(globals *Globals, code, message string) error {
	return outputErrorCommon(globals, code, message)
}
