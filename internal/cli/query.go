package cli

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/vedranburojevic/xcw/internal/domain"
	"github.com/vedranburojevic/xcw/internal/output"
	"github.com/vedranburojevic/xcw/internal/simulator"
)

// QueryCmd queries historical logs from a simulator
type QueryCmd struct {
	Simulator string   `short:"s" default:"booted" help:"Simulator name, UDID, or 'booted' for auto-detect"`
	App       string   `short:"a" required:"" help:"App bundle identifier to filter logs"`
	Since     string   `default:"5m" help:"How far back to query (e.g., '5m', '1h', '30s')"`
	Until     string   `help:"End time for query (RFC3339 or relative like '1m')"`
	Pattern   string   `short:"p" help:"Regex pattern to filter log messages"`
	Limit     int      `default:"1000" help:"Maximum number of logs to return"`
	Subsystem []string `help:"Filter by subsystem (can be repeated)"`
	Category  []string `help:"Filter by category (can be repeated)"`
	Analyze   bool     `help:"Include AI-friendly analysis summary"`
}

// Run executes the query command
func (c *QueryCmd) Run(globals *Globals) error {
	ctx := context.Background()

	// Find the simulator
	mgr := simulator.NewManager()
	device, err := mgr.FindDevice(ctx, c.Simulator)
	if err != nil {
		return c.outputError(globals, "DEVICE_NOT_FOUND", err.Error())
	}

	// Parse since duration
	since, err := time.ParseDuration(c.Since)
	if err != nil {
		return c.outputError(globals, "INVALID_DURATION", fmt.Sprintf("invalid since duration: %s", err))
	}

	// Compile pattern regex if provided
	var pattern *regexp.Regexp
	if c.Pattern != "" {
		pattern, err = regexp.Compile(c.Pattern)
		if err != nil {
			return c.outputError(globals, "INVALID_PATTERN", fmt.Sprintf("invalid regex pattern: %s", err))
		}
	}

	// Output query info if not quiet
	if !globals.Quiet {
		if globals.Format == "ndjson" {
			fmt.Fprintf(globals.Stdout, `{"type":"info","message":"Querying logs from %s","simulator":"%s","since":"%s"}`+"\n",
				device.Name, device.UDID, c.Since)
		} else {
			fmt.Fprintf(globals.Stderr, "Querying logs from %s (%s)\n", device.Name, device.UDID)
			fmt.Fprintf(globals.Stderr, "Time range: last %s\n", c.Since)
			fmt.Fprintf(globals.Stderr, "Filtering by app: %s\n\n", c.App)
		}
	}

	// Create query reader
	reader := simulator.NewQueryReader()
	opts := simulator.QueryOptions{
		BundleID:   c.App,
		Subsystems: c.Subsystem,
		Categories: c.Category,
		MinLevel:   domain.ParseLogLevel(globals.Level),
		Pattern:    pattern,
		Since:      since,
		Limit:      c.Limit,
	}

	// Execute query
	entries, err := reader.Query(ctx, device.UDID, opts)
	if err != nil {
		return c.outputError(globals, "QUERY_FAILED", err.Error())
	}

	// Create output writer
	if globals.Format == "ndjson" {
		writer := output.NewNDJSONWriter(globals.Stdout)

		// Output entries
		for _, entry := range entries {
			if err := writer.Write(&entry); err != nil {
				return err
			}
		}

		// Output analysis if requested
		if c.Analyze {
			analyzer := output.NewAnalyzer()
			summary := analyzer.Summarize(entries)
			patterns := analyzer.DetectPatterns(entries)
			analysisOutput := output.NewSummaryOutput(summary, patterns)
			if err := writer.WriteRaw(analysisOutput); err != nil {
				return err
			}
		}
	} else {
		writer := output.NewTextWriter(globals.Stdout)

		// Output entries
		for _, entry := range entries {
			if err := writer.Write(&entry); err != nil {
				return err
			}
		}

		// Output summary
		fmt.Fprintf(globals.Stdout, "\n--- Query Results ---\n")
		fmt.Fprintf(globals.Stdout, "Total: %d entries\n", len(entries))

		if c.Analyze {
			analyzer := output.NewAnalyzer()
			summary := analyzer.Summarize(entries)
			fmt.Fprintf(globals.Stdout, "Errors: %d, Faults: %d\n", summary.ErrorCount, summary.FaultCount)
			if len(summary.TopErrors) > 0 {
				fmt.Fprintln(globals.Stdout, "\nTop Errors:")
				for _, e := range summary.TopErrors {
					fmt.Fprintf(globals.Stdout, "  - %s\n", e)
				}
			}
		}
	}

	return nil
}

func (c *QueryCmd) outputError(globals *Globals, code, message string) error {
	if globals.Format == "ndjson" {
		w := output.NewNDJSONWriter(globals.Stdout)
		w.WriteError(code, message)
	} else {
		fmt.Fprintf(globals.Stderr, "Error [%s]: %s\n", code, message)
	}
	return fmt.Errorf(message)
}
