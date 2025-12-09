package cli

import (
	"errors"
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
	Simulator        string   `short:"s" help:"Simulator name or UDID"`
	Booted           bool     `short:"b" help:"Use booted simulator (error if multiple)"`
	App              string   `short:"a" required:"" help:"App bundle identifier to filter logs"`
	Since            string   `default:"5m" help:"How far back to query (e.g., '5m', '1h', '30s')"`
	Until            string   `help:"End time for query (RFC3339 or relative like '1m')"`
	Pattern          string   `short:"p" help:"Regex pattern to filter log messages"`
	Exclude          string   `short:"x" help:"Regex pattern to exclude from log messages"`
	ExcludeSubsystem []string `help:"Exclude logs from subsystem (can be repeated, supports * wildcard)"`
	Limit            int      `default:"1000" help:"Maximum number of logs to return"`
	Subsystem        []string `help:"Filter by subsystem (can be repeated)"`
	Category         []string `help:"Filter by category (can be repeated)"`
	Predicate        string   `help:"Raw NSPredicate filter (overrides --app, --subsystem, --category)"`
	Analyze          bool     `help:"Include AI-friendly analysis summary"`
}

// Run executes the query command
func (c *QueryCmd) Run(globals *Globals) error {
	ctx := context.Background()

	// Validate mutual exclusivity of flags
	if c.Simulator != "" && c.Booted {
		return c.outputError(globals, "INVALID_FLAGS", "--simulator and --booted are mutually exclusive")
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
	globals.Debug("Found device: %s (UDID: %s)", device.Name, device.UDID)

	// Parse since duration
	since, err := time.ParseDuration(c.Since)
	if err != nil {
		return c.outputError(globals, "INVALID_DURATION", fmt.Sprintf("invalid since duration: %s", err))
	}

	// Parse until time if provided (RFC3339 or relative duration)
	var until time.Time
	if c.Until != "" {
		until, err = parseTimeOrDuration(c.Until)
		if err != nil {
			return c.outputError(globals, "INVALID_UNTIL", fmt.Sprintf("invalid until time: %s", err))
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

	// Compile exclude pattern regex if provided
	var excludePattern *regexp.Regexp
	if c.Exclude != "" {
		excludePattern, err = regexp.Compile(c.Exclude)
		if err != nil {
			return c.outputError(globals, "INVALID_EXCLUDE_PATTERN", fmt.Sprintf("invalid exclude regex pattern: %s", err))
		}
	}

	// Output query info if not quiet
	if !globals.Quiet {
		if globals.Format == "ndjson" {
			output.NewNDJSONWriter(globals.Stdout).WriteInfo(
				fmt.Sprintf("Querying logs from %s", device.Name),
				device.Name, device.UDID, c.Since, "")
		} else {
			fmt.Fprintf(globals.Stderr, "Querying logs from %s (%s)\n", device.Name, device.UDID)
			fmt.Fprintf(globals.Stderr, "Time range: last %s\n", c.Since)
			fmt.Fprintf(globals.Stderr, "Filtering by app: %s\n\n", c.App)
		}
	}

	// Create query reader
	reader := simulator.NewQueryReader()
	opts := simulator.QueryOptions{
		BundleID:          c.App,
		Subsystems:        c.Subsystem,
		Categories:        c.Category,
		MinLevel:          domain.ParseLogLevel(globals.Level),
		Pattern:           pattern,
		ExcludePattern:    excludePattern,
		ExcludeSubsystems: c.ExcludeSubsystem,
		Since:             since,
		Until:             until,
		Limit:             c.Limit,
		RawPredicate:      c.Predicate,
	}

	// Execute query
	globals.Debug("Query options: BundleID=%s, Since=%s, Limit=%d", opts.BundleID, opts.Since, opts.Limit)
	globals.Debug("Executing query...")
	entries, err := reader.Query(ctx, device.UDID, opts)
	if err != nil {
		return c.outputError(globals, "QUERY_FAILED", err.Error())
	}
	globals.Debug("Query returned %d entries", len(entries))

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
	return errors.New(message)
}

// parseTimeOrDuration parses a time string as either RFC3339 or a duration offset from now
// Examples: "2024-01-15T10:30:00Z" (absolute), "5m" (5 minutes ago), "1h" (1 hour ago)
func parseTimeOrDuration(s string) (time.Time, error) {
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try duration (relative to now)
	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-d), nil
	}

	return time.Time{}, fmt.Errorf("must be RFC3339 (e.g., 2024-01-15T10:30:00Z) or duration (e.g., 5m, 1h)")
}
