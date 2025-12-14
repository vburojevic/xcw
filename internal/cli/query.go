package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/vburojevic/xcw/internal/config"
	"github.com/vburojevic/xcw/internal/domain"
	"github.com/vburojevic/xcw/internal/output"
	"github.com/vburojevic/xcw/internal/simulator"
)

// QueryCmd queries historical logs from a simulator
type QueryCmd struct {
	Simulator        string   `short:"s" help:"Simulator name or UDID"`
	Booted           bool     `short:"b" help:"Use booted simulator (error if multiple)"`
	App              string   `short:"a" help:"App bundle identifier to filter logs (required unless --predicate or --all)"`
	All              bool     `help:"Allow querying without --app/--predicate (can be very noisy)"`
	Since            string   `default:"5m" help:"How far back to query (e.g., '5m', '1h', '30s')"`
	Until            string   `help:"End time for query (RFC3339 or relative like '1m')"`
	Pattern          string   `short:"p" aliases:"filter" help:"Regex pattern to filter log messages"`
	Exclude          []string `short:"x" help:"Regex pattern to exclude from log messages (can be repeated)"`
	ExcludeSubsystem []string `help:"Exclude logs from subsystem (can be repeated, supports * wildcard)"`
	Limit            int      `default:"1000" help:"Maximum number of logs to return"`
	Subsystem        []string `help:"Filter by subsystem (can be repeated)"`
	Category         []string `help:"Filter by category (can be repeated)"`
	Process          []string `help:"Filter by process name (can be repeated)"`
	MinLevel         string   `help:"Minimum log level: debug, info, default, error, fault (overrides global --level)"`
	MaxLevel         string   `help:"Maximum log level: debug, info, default, error, fault"`
	Predicate        string   `help:"Raw NSPredicate filter (overrides --app, --subsystem, --category)"`
	Analyze          bool     `help:"Include AI-friendly analysis summary"`
	PersistPatterns  bool     `help:"Save detected patterns for future reference (marks new vs known)"`
	PatternFile      string   `help:"Custom pattern file path (default: ~/.xcw/patterns.json)"`
	Where            []string `short:"w" help:"Field filter expression (supports AND/OR/NOT, parentheses). Operators: =, !=, ~, !~, >=, <=, ^, $. Regex literals: /pattern/i"`
}

// Run executes the query command
func (c *QueryCmd) Run(globals *Globals) error {
	applyQueryDefaults(globals.Config, c)

	ctx := context.Background()

	// Validate mutual exclusivity of flags
	if globals.FlagProvided("simulator") && globals.FlagProvided("booted") {
		return c.outputError(globals, "INVALID_FLAGS", "--simulator and --booted are mutually exclusive")
	}
	if err := validateAppPredicateAll(c.App, c.Predicate, c.All, len(c.Subsystem) > 0 || len(c.Category) > 0); err != nil {
		return outputErrorCommon(globals, err.Code, err.Message, err.Hint)
	}

	// Find the simulator
	mgr := simulator.NewManager()
	device, err := resolveSimulatorDevice(ctx, mgr, c.Simulator, c.Booted)
	if err != nil {
		return c.outputError(globals, "DEVICE_NOT_FOUND", err.Error(), hintForStreamOrQuery(err))
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

	// Compile filters (pattern, exclude, where)
	pattern, excludePatterns, whereFilter, err := buildFilters(c.Pattern, c.Exclude, c.Where)
	if err != nil {
		return c.outputError(globals, "INVALID_FILTER", err.Error(), hintForFilter(err))
	}

	// Output query info if not quiet
	if !globals.Quiet {
		if globals.Format == "ndjson" {
			if err := output.NewNDJSONWriter(globals.Stdout).WriteInfo(
				fmt.Sprintf("Querying logs from %s", device.Name),
				device.Name, device.UDID, c.Since, ""); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(globals.Stderr, "Querying logs from %s (%s)\n", device.Name, device.UDID); err != nil {
				globals.Debug("failed to write query info: %v", err)
			}
			if _, err := fmt.Fprintf(globals.Stderr, "Time range: last %s\n", c.Since); err != nil {
				globals.Debug("failed to write query info: %v", err)
			}
			if _, err := fmt.Fprintf(globals.Stderr, "Filtering by app: %s\n\n", c.App); err != nil {
				globals.Debug("failed to write query info: %v", err)
			}
		}
	}

	// Determine log level (command-specific overrides global)
	minLevel, maxLevel := resolveLevels(c.MinLevel, c.MaxLevel, globals.Level)

	// Create query reader
	reader := simulator.NewQueryReader()
	var diagEmitter *output.Emitter
	if globals.Format == "ndjson" {
		diagEmitter = output.NewEmitter(globals.Stdout)
	}
	opts := simulator.QueryOptions{
		BundleID:          c.App,
		Subsystems:        c.Subsystem,
		Categories:        c.Category,
		Processes:         c.Process,
		MinLevel:          minLevel,
		MaxLevel:          maxLevel,
		Pattern:           pattern,
		ExcludePatterns:   excludePatterns,
		ExcludeSubsystems: c.ExcludeSubsystem,
		Since:             since,
		Until:             until,
		Limit:             c.Limit,
		RawPredicate:      c.Predicate,
	}
	if globals.Verbose {
		opts.OnStderrLine = func(line string) {
			emitWarning(globals, diagEmitter, "xcrun_stderr: "+line)
		}
	}

	// Execute query
	globals.Debug("Query options: BundleID=%s, Since=%s, Limit=%d", opts.BundleID, opts.Since, opts.Limit)
	globals.Debug("Executing query...")
	entries, err := reader.Query(ctx, device.UDID, opts)
	if err != nil {
		return c.outputError(globals, "QUERY_FAILED", err.Error(), hintForStreamOrQuery(err))
	}
	globals.Debug("Query returned %d entries", len(entries))

	// Apply where filter after the query reader's filtering (pattern/exclude are already applied there).
	if whereFilter != nil {
		globals.Debug("Where filter: %d clause(s)", len(c.Where))
		var filtered []domain.LogEntry
		for _, entry := range entries {
			if whereFilter.Match(&entry) {
				filtered = append(filtered, entry)
			}
		}
		globals.Debug("After where filter: %d entries (was %d)", len(filtered), len(entries))
		entries = filtered
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

			if c.PersistPatterns {
				// Use pattern store for enhanced analysis
				store := output.NewPatternStore(c.PatternFile)
				enhanced := store.RecordPatterns(patterns)
				if err := store.Save(); err != nil {
					globals.Debug("Failed to save patterns: %v", err)
				}
				analysisOutput := output.NewEnhancedSummaryOutput(summary, enhanced)
				if err := writer.WriteRaw(analysisOutput); err != nil {
					return err
				}
			} else {
				analysisOutput := output.NewSummaryOutput(summary, patterns)
				if err := writer.WriteRaw(analysisOutput); err != nil {
					return err
				}
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
		if _, err := fmt.Fprintf(globals.Stdout, "\n--- Query Results ---\n"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(globals.Stdout, "Total: %d entries\n", len(entries)); err != nil {
			return err
		}

		if c.Analyze {
			analyzer := output.NewAnalyzer()
			summary := analyzer.Summarize(entries)
			patterns := analyzer.DetectPatterns(entries)
			if _, err := fmt.Fprintf(globals.Stdout, "Errors: %d, Faults: %d\n", summary.ErrorCount, summary.FaultCount); err != nil {
				return err
			}

			if c.PersistPatterns && len(patterns) > 0 {
				store := output.NewPatternStore(c.PatternFile)
				enhanced := store.RecordPatterns(patterns)
				if err := store.Save(); err != nil {
					globals.Debug("Failed to save patterns: %v", err)
				}

				if _, err := fmt.Fprintln(globals.Stdout, "\nError Patterns:"); err != nil {
					return err
				}
				for _, p := range enhanced {
					status := "[NEW]"
					if !p.IsNew {
						status = "[KNOWN]"
					}
					if _, err := fmt.Fprintf(globals.Stdout, "  %s %s (count: %d)\n", status, p.Pattern, p.Count); err != nil {
						return err
					}
				}
			} else if len(summary.TopErrors) > 0 {
				if _, err := fmt.Fprintln(globals.Stdout, "\nTop Errors:"); err != nil {
					return err
				}
				for _, e := range summary.TopErrors {
					if _, err := fmt.Fprintf(globals.Stdout, "  - %s\n", e); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (c *QueryCmd) outputError(globals *Globals, code, message string, hint ...string) error {
	return outputErrorCommon(globals, code, message, hint...)
}

func applyQueryDefaults(cfg *config.Config, c *QueryCmd) {
	if cfg == nil {
		return
	}
	if c.Simulator == "" {
		if cfg.Query.Simulator != "" {
			c.Simulator = cfg.Query.Simulator
		} else if cfg.Defaults.Simulator != "" {
			c.Simulator = cfg.Defaults.Simulator
		}
	}
	if c.App == "" && c.Predicate == "" && cfg.Query.App != "" {
		c.App = cfg.Query.App
	}
	if c.Since == "5m" && cfg.Query.Since != "" {
		c.Since = cfg.Query.Since
	}
	if c.Limit == 1000 && cfg.Query.Limit != 0 {
		c.Limit = cfg.Query.Limit
	}
	if len(c.Exclude) == 0 && len(cfg.Query.Exclude) > 0 {
		c.Exclude = append(c.Exclude, cfg.Query.Exclude...)
	}
	if len(c.Where) == 0 && len(cfg.Query.Where) > 0 {
		c.Where = append(c.Where, cfg.Query.Where...)
	}
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
