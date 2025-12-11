package cli

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/vburojevic/xcw/internal/domain"
	"github.com/vburojevic/xcw/internal/output"
	"github.com/vburojevic/xcw/internal/simulator"
)

// DiscoverCmd discovers what subsystems, categories, and processes exist in logs
type DiscoverCmd struct {
	Simulator string `short:"s" help:"Simulator name or UDID"`
	Booted    bool   `short:"b" help:"Use booted simulator (error if multiple)"`
	App       string `short:"a" help:"App bundle identifier to filter logs (optional)"`
	Since     string `default:"5m" help:"How far back to query (e.g., '5m', '1h', '30s')"`
	Limit     int    `default:"5000" help:"Maximum number of logs to analyze"`
	TopN      int    `default:"20" help:"Number of top items to show per category"`
}

// Run executes the discover command
func (c *DiscoverCmd) Run(globals *Globals) error {
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

	// Output discovery info if not quiet
	if !globals.Quiet {
		if globals.Format == "ndjson" {
			msg := fmt.Sprintf("Discovering logs from %s (last %s)", device.Name, c.Since)
			if c.App != "" {
				msg = fmt.Sprintf("Discovering logs from %s for %s (last %s)", device.Name, c.App, c.Since)
			}
			output.NewNDJSONWriter(globals.Stdout).WriteInfo(msg, device.Name, device.UDID, c.Since, "discovery")
		} else {
			fmt.Fprintf(globals.Stderr, "Discovering logs from %s (%s)\n", device.Name, device.UDID)
			fmt.Fprintf(globals.Stderr, "Time range: last %s\n", c.Since)
			if c.App != "" {
				fmt.Fprintf(globals.Stderr, "Filtering by app: %s\n", c.App)
			}
			fmt.Fprintln(globals.Stderr)
		}
	}

	// Create query reader with minimal filtering
	reader := simulator.NewQueryReader()
	opts := simulator.QueryOptions{
		BundleID: c.App,
		MinLevel: domain.LogLevelDebug, // Get all levels
		Since:    since,
		Limit:    c.Limit,
	}

	// Execute query
	globals.Debug("Query options: BundleID=%s, Since=%s, Limit=%d", opts.BundleID, opts.Since, opts.Limit)
	globals.Debug("Executing discovery query...")
	entries, err := reader.Query(ctx, device.UDID, opts)
	if err != nil {
		return c.outputError(globals, "QUERY_FAILED", err.Error())
	}
	globals.Debug("Query returned %d entries", len(entries))

	// Aggregate results
	discovery := c.aggregate(entries, c.App)

	// Output results
	if globals.Format == "ndjson" {
		writer := output.NewNDJSONWriter(globals.Stdout)
		if err := writer.WriteRaw(discovery); err != nil {
			return err
		}
	} else {
		c.printTextOutput(globals, discovery)
	}

	return nil
}

// aggregate builds discovery statistics from log entries
func (c *DiscoverCmd) aggregate(entries []domain.LogEntry, app string) *domain.Discovery {
	// Track aggregates
	subsystems := make(map[string]map[string]int) // subsystem -> level -> count
	categories := make(map[string]int)
	processes := make(map[string]int)
	levels := make(map[string]int)

	var earliest, latest time.Time

	for _, entry := range entries {
		// Track time range
		if earliest.IsZero() || entry.Timestamp.Before(earliest) {
			earliest = entry.Timestamp
		}
		if latest.IsZero() || entry.Timestamp.After(latest) {
			latest = entry.Timestamp
		}

		// Count by subsystem and level
		sub := entry.Subsystem
		if sub == "" {
			sub = "(none)"
		}
		if subsystems[sub] == nil {
			subsystems[sub] = make(map[string]int)
		}
		subsystems[sub][string(entry.Level)]++

		// Count by category
		cat := entry.Category
		if cat == "" {
			cat = "(none)"
		}
		categories[cat]++

		// Count by process
		proc := entry.Process
		if proc == "" {
			proc = "(unknown)"
		}
		processes[proc]++

		// Count by level
		levels[string(entry.Level)]++
	}

	// Convert to sorted slices
	subsystemList := make([]domain.SubsystemInfo, 0, len(subsystems))
	for name, levelCounts := range subsystems {
		total := 0
		for _, count := range levelCounts {
			total += count
		}
		subsystemList = append(subsystemList, domain.SubsystemInfo{
			Name:   name,
			Count:  total,
			Levels: levelCounts,
		})
	}
	sort.Slice(subsystemList, func(i, j int) bool {
		return subsystemList[i].Count > subsystemList[j].Count
	})
	if len(subsystemList) > c.TopN {
		subsystemList = subsystemList[:c.TopN]
	}

	categoryList := make([]domain.CategoryInfo, 0, len(categories))
	for name, count := range categories {
		categoryList = append(categoryList, domain.CategoryInfo{Name: name, Count: count})
	}
	sort.Slice(categoryList, func(i, j int) bool {
		return categoryList[i].Count > categoryList[j].Count
	})
	if len(categoryList) > c.TopN {
		categoryList = categoryList[:c.TopN]
	}

	processList := make([]domain.ProcessInfo, 0, len(processes))
	for name, count := range processes {
		processList = append(processList, domain.ProcessInfo{Name: name, Count: count})
	}
	sort.Slice(processList, func(i, j int) bool {
		return processList[i].Count > processList[j].Count
	})
	if len(processList) > c.TopN {
		processList = processList[:c.TopN]
	}

	return &domain.Discovery{
		Type:          "discovery",
		SchemaVersion: 1,
		App:           app,
		TimeRange: domain.DiscoveryTimeRange{
			Start: earliest.Format(time.RFC3339),
			End:   latest.Format(time.RFC3339),
		},
		TotalCount: len(entries),
		Subsystems: subsystemList,
		Categories: categoryList,
		Processes:  processList,
		Levels:     levels,
	}
}

// printTextOutput outputs discovery results in human-readable format
func (c *DiscoverCmd) printTextOutput(globals *Globals, d *domain.Discovery) {
	fmt.Fprintf(globals.Stdout, "=== Log Discovery ===\n\n")
	fmt.Fprintf(globals.Stdout, "Total logs: %d\n", d.TotalCount)
	fmt.Fprintf(globals.Stdout, "Time range: %s to %s\n\n", d.TimeRange.Start, d.TimeRange.End)

	// Level breakdown
	fmt.Fprintf(globals.Stdout, "Levels:\n")
	levelOrder := []string{"Debug", "Info", "Default", "Error", "Fault"}
	for _, level := range levelOrder {
		if count, ok := d.Levels[level]; ok {
			fmt.Fprintf(globals.Stdout, "  %-10s %d\n", level+":", count)
		}
	}
	fmt.Fprintln(globals.Stdout)

	// Subsystems
	fmt.Fprintf(globals.Stdout, "Top Subsystems:\n")
	for _, s := range d.Subsystems {
		fmt.Fprintf(globals.Stdout, "  %-50s %5d", s.Name, s.Count)
		if s.Levels["Error"] > 0 || s.Levels["Fault"] > 0 {
			fmt.Fprintf(globals.Stdout, "  (")
			if s.Levels["Error"] > 0 {
				fmt.Fprintf(globals.Stdout, "%d errors", s.Levels["Error"])
			}
			if s.Levels["Error"] > 0 && s.Levels["Fault"] > 0 {
				fmt.Fprintf(globals.Stdout, ", ")
			}
			if s.Levels["Fault"] > 0 {
				fmt.Fprintf(globals.Stdout, "%d faults", s.Levels["Fault"])
			}
			fmt.Fprintf(globals.Stdout, ")")
		}
		fmt.Fprintln(globals.Stdout)
	}
	fmt.Fprintln(globals.Stdout)

	// Categories
	fmt.Fprintf(globals.Stdout, "Top Categories:\n")
	for _, cat := range d.Categories {
		fmt.Fprintf(globals.Stdout, "  %-50s %5d\n", cat.Name, cat.Count)
	}
	fmt.Fprintln(globals.Stdout)

	// Processes
	fmt.Fprintf(globals.Stdout, "Top Processes:\n")
	for _, p := range d.Processes {
		fmt.Fprintf(globals.Stdout, "  %-50s %5d\n", p.Name, p.Count)
	}
}

func (c *DiscoverCmd) outputError(globals *Globals, code, message string) error {
	return outputErrorCommon(globals, code, message)
}
