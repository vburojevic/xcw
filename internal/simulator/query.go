package simulator

import (
	"bufio"
	"context"
	"errors"
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

	// Diagnostics
	CommandTimeout time.Duration     // Optional timeout for the xcrun log show command
	OnStderrLine   func(line string) // Optional callback for xcrun stderr output (trimmed)
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

	cmdCtx := ctx
	cancel := func() {}
	if opts.CommandTimeout > 0 {
		cmdCtx, cancel = context.WithTimeout(ctx, opts.CommandTimeout)
	} else if _, ok := ctx.Deadline(); !ok {
		// Default guardrail to prevent hung xcrun calls when no deadline is set.
		cmdCtx, cancel = context.WithTimeout(ctx, 2*time.Minute)
	}
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "xcrun", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start log show: %w", err)
	}

	// Drain stderr to avoid deadlocks and optionally surface diagnostics.
	stderrErrCh := make(chan error, 1)
	onStderr := opts.OnStderrLine
	go func() {
		sc := bufio.NewScanner(stderr)
		sc.Buffer(make([]byte, 0, 64*1024), 256*1024)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			if onStderr != nil {
				onStderr(line)
			}
		}
		stderrErrCh <- sc.Err()
	}()

	var entries []domain.LogEntry
	scanner := bufio.NewScanner(stdout)
	const maxLineBytes = 1024 * 1024
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)

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

	stdoutErr := scanner.Err()
	if stdoutErr != nil && errors.Is(stdoutErr, bufio.ErrTooLong) {
		stdoutErr = fmt.Errorf("log show output line too long (>%d bytes): %w", maxLineBytes, stdoutErr)
	}
	if stdoutErr != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}

	waitErr := cmd.Wait()
	stderrErr := <-stderrErrCh

	if stdoutErr != nil {
		return nil, stdoutErr
	}
	if stderrErr != nil && cmdCtx.Err() == nil {
		return nil, fmt.Errorf("log show stderr read error: %w", stderrErr)
	}
	if cmdCtx.Err() != nil {
		return nil, cmdCtx.Err()
	}
	if waitErr != nil {
		return nil, fmt.Errorf("log show failed: %w", waitErr)
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
	return buildPredicate(opts.RawPredicate, opts.BundleID, opts.Subsystems, opts.Categories)
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

// matchExcludePatterns checks if message matches any exclude pattern
func matchExcludePatterns(message string, patterns []*regexp.Regexp) bool {
	for _, p := range patterns {
		if p != nil && p.MatchString(message) {
			return true
		}
	}
	return false
}
