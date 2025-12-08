package simulator

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/vedranburojevic/xcw/internal/domain"
)

// QueryOptions configures historical log queries
type QueryOptions struct {
	BundleID   string         // Filter by app bundle identifier
	Subsystems []string       // Filter by subsystems
	Categories []string       // Filter by categories
	MinLevel   domain.LogLevel // Minimum log level
	Pattern    *regexp.Regexp // Regex pattern for message filtering
	Since      time.Duration  // How far back to query
	Until      time.Time      // End time (default: now)
	Limit      int            // Max entries to return
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

		// Apply level filter
		if entry.Level.Priority() < opts.MinLevel.Priority() {
			continue
		}

		// Apply pattern filter
		if opts.Pattern != nil && !opts.Pattern.MatchString(entry.Message) {
			continue
		}

		entries = append(entries, *entry)

		if opts.Limit > 0 && len(entries) >= opts.Limit {
			break
		}
	}

	cmd.Wait()
	return entries, nil
}

func (r *QueryReader) buildArgs(udid string, opts QueryOptions) []string {
	args := []string{"simctl", "spawn", udid, "log", "show", "--style", "ndjson"}

	// Time range
	if opts.Since > 0 {
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

func (r *QueryReader) buildPredicate(opts QueryOptions) string {
	var parts []string

	if opts.BundleID != "" {
		parts = append(parts, fmt.Sprintf(`subsystem BEGINSWITH "%s"`, opts.BundleID))
	}

	for _, sub := range opts.Subsystems {
		parts = append(parts, fmt.Sprintf(`subsystem == "%s"`, sub))
	}

	for _, cat := range opts.Categories {
		parts = append(parts, fmt.Sprintf(`category == "%s"`, cat))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " OR ")
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
