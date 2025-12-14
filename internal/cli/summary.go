package cli

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/vburojevic/xcw/internal/domain"
	"github.com/vburojevic/xcw/internal/output"
	"github.com/vburojevic/xcw/internal/simulator"
)

// SummaryCmd outputs a summary of recent logs
type SummaryCmd struct {
	Simulator string `short:"s" default:"booted" help:"Simulator name, UDID, or 'booted' for auto-detect"`
	App       string `short:"a" required:"" help:"App bundle identifier to filter logs"`
	Window    string `default:"5m" help:"Time window for summary (e.g., '5m', '1h')"`
	Pattern   string `short:"p" aliases:"filter" help:"Regex pattern to filter log messages"`
}

// Run executes the summary command
func (c *SummaryCmd) Run(globals *Globals) error {
	ctx := context.Background()

	// Find the simulator
	mgr := simulator.NewManager()
	device, err := mgr.FindDevice(ctx, c.Simulator)
	if err != nil {
		return c.outputError(globals, "DEVICE_NOT_FOUND", err.Error(), hintForStreamOrQuery(err))
	}

	// Parse window duration
	window, err := time.ParseDuration(c.Window)
	if err != nil {
		return c.outputError(globals, "INVALID_DURATION", fmt.Sprintf("invalid window duration: %s", err))
	}

	// Compile pattern regex if provided
	var pattern *regexp.Regexp
	if c.Pattern != "" {
		pattern, err = regexp.Compile(c.Pattern)
		if err != nil {
			return c.outputError(globals, "INVALID_PATTERN", fmt.Sprintf("invalid regex pattern: %s", err))
		}
	}

	// Query recent logs
	reader := simulator.NewQueryReader()
	opts := simulator.QueryOptions{
		BundleID: c.App,
		MinLevel: domain.ParseLogLevel(globals.Level),
		Pattern:  pattern,
		Since:    window,
		Limit:    10000, // Get enough for good analysis
	}

	entries, err := reader.Query(ctx, device.UDID, opts)
	if err != nil {
		return c.outputError(globals, "QUERY_FAILED", err.Error(), hintForStreamOrQuery(err))
	}

	// Analyze logs
	analyzer := output.NewAnalyzer()
	summary := analyzer.Summarize(entries)
	patterns := analyzer.DetectPatterns(entries)

	// Output results
	if globals.Format == "ndjson" {
		writer := output.NewNDJSONWriter(globals.Stdout)
		analysisOutput := output.NewSummaryOutput(summary, patterns)
		return writer.WriteRaw(analysisOutput)
	}

	// Text output
	if _, err := fmt.Fprintf(globals.Stdout, "=== Log Summary for %s ===\n", device.Name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "App: %s\n", c.App); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "Window: last %s\n\n", c.Window); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(globals.Stdout, "Counts:\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  Total:   %d\n", summary.TotalCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  Debug:   %d\n", summary.DebugCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  Info:    %d\n", summary.InfoCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  Default: %d\n", summary.DefaultCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  Error:   %d\n", summary.ErrorCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  Fault:   %d\n", summary.FaultCount); err != nil {
		return err
	}

	if summary.ErrorRate > 0 {
		if _, err := fmt.Fprintf(globals.Stdout, "\nError rate: %.2f/min\n", summary.ErrorRate); err != nil {
			return err
		}
	}

	if len(summary.TopErrors) > 0 {
		if _, err := fmt.Fprintln(globals.Stdout, "\nTop Errors:"); err != nil {
			return err
		}
		for i, e := range summary.TopErrors {
			if _, err := fmt.Fprintf(globals.Stdout, "  %d. %s\n", i+1, e); err != nil {
				return err
			}
		}
	}

	if len(summary.TopFaults) > 0 {
		if _, err := fmt.Fprintln(globals.Stdout, "\nTop Faults:"); err != nil {
			return err
		}
		for i, f := range summary.TopFaults {
			if _, err := fmt.Fprintf(globals.Stdout, "  %d. %s\n", i+1, f); err != nil {
				return err
			}
		}
	}

	if len(patterns) > 0 {
		if _, err := fmt.Fprintln(globals.Stdout, "\nRepeated Error Patterns:"); err != nil {
			return err
		}
		for _, p := range patterns {
			if _, err := fmt.Fprintf(globals.Stdout, "  [%dx] %s\n", p.Count, p.Pattern); err != nil {
				return err
			}
		}
	}

	// AI-friendly status
	if _, err := fmt.Fprintln(globals.Stdout, "\n--- AI Status ---"); err != nil {
		return err
	}
	if summary.HasFaults {
		if _, err := fmt.Fprintln(globals.Stdout, "STATUS: FAULTS DETECTED"); err != nil {
			return err
		}
	} else if summary.HasErrors {
		if _, err := fmt.Fprintln(globals.Stdout, "STATUS: ERRORS DETECTED"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(globals.Stdout, "STATUS: OK"); err != nil {
			return err
		}
	}

	return nil
}

func (c *SummaryCmd) outputError(globals *Globals, code, message string, hint ...string) error {
	return outputErrorCommon(globals, code, message, hint...)
}
