package simulator

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/vburojevic/xcw/internal/domain"
)

// QueryOptions configures historical log queries
type QueryOptions struct {
	BundleID          string           // Filter by app bundle identifier
	Subsystems        []string         // Filter by subsystems
	Categories        []string         // Filter by categories
	Processes         []string         // Filter by process names
	MinLevel          domain.LogLevel  // Minimum log level (inclusive)
	MaxLevel          domain.LogLevel  // Maximum log level (inclusive, empty = no max)
	Pattern           *regexp.Regexp   // Regex pattern for message filtering
	ExcludePatterns   []*regexp.Regexp // Regex patterns to exclude from messages
	ExcludeSubsystems []string         // Subsystems to exclude
	Since             time.Duration    // How far back to query
	Until             time.Time        // End time (default: now)
	Limit             int              // Max entries to return
	RawPredicate      string           // Raw NSPredicate string (overrides other filters)
}

// QueryReader reads historical logs from a simulator
type QueryReader struct {
	parser *Parser
}

// NewQueryReader creates a new query reader
func NewQueryReader() *QueryReader {
	return &QueryReader{
		parser: NewParser(),
	}
}

// Query reads historical logs matching the criteria
func (r *QueryReader) Query(ctx context.Context, udid string, opts QueryOptions) ([]domain.LogEntry, error) {
	args := r.buildArgs(udid, opts)

	cmd := exec.CommandContext(ctx, "xcrun", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start log show: %w", err)
	}

	var entries []domain.LogEntry
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		entry, err := r.parser.Parse(scanner.Bytes())
		if err != nil {
			continue
		}

		if entry == nil {
			continue
		}

		// Apply level filter (min)
		if entry.Level.Priority() < opts.MinLevel.Priority() {
			continue
		}

		// Apply level filter (max) - only if MaxLevel is set
		if opts.MaxLevel != "" && entry.Level.Priority() > opts.MaxLevel.Priority() {
			continue
		}

		// Apply pattern filter
		if opts.Pattern != nil && !opts.Pattern.MatchString(entry.Message) {
			continue
		}

		// Apply exclusion pattern filters (any match excludes)
		if matchExcludePatterns(entry.Message, opts.ExcludePatterns) {
			continue
		}

		// Apply subsystem exclusion filter
		if len(opts.ExcludeSubsystems) > 0 && shouldExcludeSubsystem(entry.Subsystem, opts.ExcludeSubsystems) {
			continue
		}

		// Apply process filter
		if len(opts.Processes) > 0 && !matchProcess(entry.Process, opts.Processes) {
			continue
		}

		entries = append(entries, *entry)

		if opts.Limit > 0 && len(entries) >= opts.Limit {
			break
		}
	}

	if err := cmd.Wait(); err != nil {
		return entries, fmt.Errorf("log show failed: %w", err)
	}
	return entries, nil
}

func (r *QueryReader) buildArgs(udid string, opts QueryOptions) []string {
	args := []string{"simctl", "spawn", udid, "log", "show", "--style", "ndjson"}

	// Time range: use --start/--end when Until is set, otherwise --last
	if !opts.Until.IsZero() {
		// Absolute time range with --start and --end
		start := time.Now().Add(-opts.Since)
		args = append(args, "--start", start.Format(time.RFC3339))
		args = append(args, "--end", opts.Until.Format(time.RFC3339))
	} else if opts.Since > 0 {
		// Relative duration with --last
		args = append(args, "--last", formatDuration(opts.Since))
	}

	// Include all log levels to allow filtering
	args = append(args, "--info", "--debug")

	// Build predicate
	predicate := r.buildPredicate(opts)
	if predicate != "" {
		args = append(args, "--predicate", predicate)
	}

	return args
}

// buildPredicate constructs an NSPredicate string for log filtering
// Uses AND between groups (subsystem, category) for narrowing results
// Uses OR within groups for matching any of multiple values
func (r *QueryReader) buildPredicate(opts QueryOptions) string {
	// If raw predicate provided, use it directly
	if opts.RawPredicate != "" {
		return opts.RawPredicate
	}

	var groups []string

	// Subsystem group: bundle ID and/or explicit subsystems (OR within group)
	var subsystemParts []string
	if opts.BundleID != "" {
		subsystemParts = append(subsystemParts, fmt.Sprintf(`subsystem BEGINSWITH "%s"`, opts.BundleID))
	}
	for _, sub := range opts.Subsystems {
		subsystemParts = append(subsystemParts, fmt.Sprintf(`subsystem == "%s"`, sub))
	}
	if len(subsystemParts) > 0 {
		if len(subsystemParts) == 1 {
			groups = append(groups, subsystemParts[0])
		} else {
			groups = append(groups, "("+strings.Join(subsystemParts, " OR ")+")")
		}
	}

	// Category group (OR within group)
	var categoryParts []string
	for _, cat := range opts.Categories {
		categoryParts = append(categoryParts, fmt.Sprintf(`category == "%s"`, cat))
	}
	if len(categoryParts) > 0 {
		if len(categoryParts) == 1 {
			groups = append(groups, categoryParts[0])
		} else {
			groups = append(groups, "("+strings.Join(categoryParts, " OR ")+")")
		}
	}

	if len(groups) == 0 {
		return ""
	}

	// AND between groups for narrowing
	return strings.Join(groups, " AND ")
}

func formatDuration(d time.Duration) string {
	if d >= 24*time.Hour {
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	}
	if d >= time.Hour {
		hours := int(d.Hours())
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(d.Minutes())
	if minutes < 1 {
		return "1m"
	}
	return fmt.Sprintf("%dm", minutes)
}

// shouldExcludeSubsystem checks if a subsystem should be excluded
func shouldExcludeSubsystem(subsystem string, excludeList []string) bool {
	for _, excl := range excludeList {
		// Support wildcard matching with *
		if strings.HasSuffix(excl, "*") {
			prefix := strings.TrimSuffix(excl, "*")
			if strings.HasPrefix(subsystem, prefix) {
				return true
			}
		} else if subsystem == excl {
			return true
		}
	}
	return false
}

// matchProcess checks if a process matches the filter list
func matchProcess(process string, processes []string) bool {
	for _, p := range processes {
		if process == p {
			return true
		}
	}
	return false
}

// matchExcludePatterns checks if message matches any exclude pattern
func matchExcludePatterns(message string, patterns []*regexp.Regexp) bool {
	for _, p := range patterns {
		if p != nil && p.MatchString(message) {
			return true
		}
	}
	return false
}
