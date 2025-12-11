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
		return c.outputError(globals, "DEVICE_NOT_FOUND", err.Error())
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
		return c.outputError(globals, "QUERY_FAILED", err.Error())
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
	fmt.Fprintf(globals.Stdout, "=== Log Summary for %s ===\n", device.Name)
	fmt.Fprintf(globals.Stdout, "App: %s\n", c.App)
	fmt.Fprintf(globals.Stdout, "Window: last %s\n\n", c.Window)

	fmt.Fprintf(globals.Stdout, "Counts:\n")
	fmt.Fprintf(globals.Stdout, "  Total:   %d\n", summary.TotalCount)
	fmt.Fprintf(globals.Stdout, "  Debug:   %d\n", summary.DebugCount)
	fmt.Fprintf(globals.Stdout, "  Info:    %d\n", summary.InfoCount)
	fmt.Fprintf(globals.Stdout, "  Default: %d\n", summary.DefaultCount)
	fmt.Fprintf(globals.Stdout, "  Error:   %d\n", summary.ErrorCount)
	fmt.Fprintf(globals.Stdout, "  Fault:   %d\n", summary.FaultCount)

	if summary.ErrorRate > 0 {
		fmt.Fprintf(globals.Stdout, "\nError rate: %.2f/min\n", summary.ErrorRate)
	}

	if len(summary.TopErrors) > 0 {
		fmt.Fprintln(globals.Stdout, "\nTop Errors:")
		for i, e := range summary.TopErrors {
			fmt.Fprintf(globals.Stdout, "  %d. %s\n", i+1, e)
		}
	}

	if len(summary.TopFaults) > 0 {
		fmt.Fprintln(globals.Stdout, "\nTop Faults:")
		for i, f := range summary.TopFaults {
			fmt.Fprintf(globals.Stdout, "  %d. %s\n", i+1, f)
		}
	}

	if len(patterns) > 0 {
		fmt.Fprintln(globals.Stdout, "\nRepeated Error Patterns:")
		for _, p := range patterns {
			fmt.Fprintf(globals.Stdout, "  [%dx] %s\n", p.Count, p.Pattern)
		}
	}

	// AI-friendly status
	fmt.Fprintln(globals.Stdout, "\n--- AI Status ---")
	if summary.HasFaults {
		fmt.Fprintln(globals.Stdout, "STATUS: FAULTS DETECTED")
	} else if summary.HasErrors {
		fmt.Fprintln(globals.Stdout, "STATUS: ERRORS DETECTED")
	} else {
		fmt.Fprintln(globals.Stdout, "STATUS: OK")
	}

	return nil
}

func (c *SummaryCmd) outputError(globals *Globals, code, message string) error {
	return outputErrorCommon(globals, code, message)
}
