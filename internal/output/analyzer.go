package output

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/vburojevic/xcw/internal/domain"
)

// Precompiled regexes for message normalization
var (
	hexAddrRegex = regexp.MustCompile(`0x[0-9a-fA-F]+`)
	numberRegex  = regexp.MustCompile(`\d+`)
	uuidRegex    = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
)

// Analyzer provides AI-friendly log analysis and summarization
type Analyzer struct {
	errorPatterns []*regexp.Regexp
}

// NewAnalyzer creates a new log analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		errorPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)error[:\s]`),
			regexp.MustCompile(`(?i)fail(?:ed|ure)?[:\s]`),
			regexp.MustCompile(`(?i)exception[:\s]`),
			regexp.MustCompile(`(?i)crash(?:ed)?[:\s]`),
			regexp.MustCompile(`(?i)timeout[:\s]`),
			regexp.MustCompile(`(?i)denied[:\s]`),
			regexp.MustCompile(`(?i)invalid[:\s]`),
			regexp.MustCompile(`(?i)nil|null pointer`),
		},
	}
}

// Summarize generates an AI-friendly summary from log entries
func (a *Analyzer) Summarize(entries []domain.LogEntry) *domain.LogSummary {
	summary := domain.NewLogSummary()

	if len(entries) == 0 {
		return summary
	}

	// Set time range
	summary.WindowStart = entries[0].Timestamp
	summary.WindowEnd = entries[len(entries)-1].Timestamp

	// Count by level
	errorMessages := make(map[string]int)
	faultMessages := make(map[string]int)

	for _, entry := range entries {
		summary.TotalCount++

		switch entry.Level {
		case domain.LogLevelDebug:
			summary.DebugCount++
		case domain.LogLevelInfo:
			summary.InfoCount++
		case domain.LogLevelDefault:
			summary.DefaultCount++
		case domain.LogLevelError:
			summary.ErrorCount++
			normalized := a.normalizeMessage(entry.Message)
			errorMessages[normalized]++
		case domain.LogLevelFault:
			summary.FaultCount++
			normalized := a.normalizeMessage(entry.Message)
			faultMessages[normalized]++
		}
	}

	// Set flags
	summary.HasErrors = summary.ErrorCount > 0
	summary.HasFaults = summary.FaultCount > 0

	// Calculate error rate (per minute)
	duration := summary.WindowEnd.Sub(summary.WindowStart)
	if duration > 0 {
		minutes := duration.Minutes()
		if minutes > 0 {
			summary.ErrorRate = float64(summary.ErrorCount+summary.FaultCount) / minutes
		}
	}

	// Get top errors
	summary.TopErrors = a.getTopMessages(errorMessages, 5)
	summary.TopFaults = a.getTopMessages(faultMessages, 5)

	return summary
}

// normalizeMessage removes variable parts to group similar messages
func (a *Analyzer) normalizeMessage(msg string) string {
	// Remove UUIDs first (before numbers, since UUIDs contain numbers)
	msg = uuidRegex.ReplaceAllString(msg, "<uuid>")
	// Remove hex addresses
	msg = hexAddrRegex.ReplaceAllString(msg, "<addr>")
	// Remove numbers
	msg = numberRegex.ReplaceAllString(msg, "<n>")

	// Truncate long messages
	if len(msg) > 100 {
		msg = msg[:100] + "..."
	}

	return strings.TrimSpace(msg)
}

// getTopMessages returns the top N messages by frequency
func (a *Analyzer) getTopMessages(counts map[string]int, limit int) []string {
	// Convert map to slice of entries and sort by count
	entries := lo.Entries(counts)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Value > entries[j].Value
	})

	// Take top N and extract just the keys
	topEntries := lo.Slice(entries, 0, limit)
	return lo.Map(topEntries, func(e lo.Entry[string, int], _ int) string {
		return e.Key
	})
}

// DetectPatterns finds recurring error patterns
func (a *Analyzer) DetectPatterns(entries []domain.LogEntry) []PatternMatch {
	// Filter to only error/fault entries
	errorEntries := lo.Filter(entries, func(e domain.LogEntry, _ int) bool {
		return e.Level == domain.LogLevelError || e.Level == domain.LogLevelFault
	})

	// Group similar error messages by normalized pattern
	errorGroups := lo.GroupBy(errorEntries, func(e domain.LogEntry) string {
		return a.normalizeMessage(e.Message)
	})

	// Convert to PatternMatch slice, filtering those with >= 2 occurrences
	patterns := lo.FilterMap(lo.Entries(errorGroups), func(e lo.Entry[string, []domain.LogEntry], _ int) (PatternMatch, bool) {
		if len(e.Value) < 2 {
			return PatternMatch{}, false
		}

		// Get message samples (max 3)
		samples := lo.Map(lo.Slice(e.Value, 0, 3), func(entry domain.LogEntry, _ int) string {
			return entry.Message
		})

		return PatternMatch{
			Pattern: e.Key,
			Count:   len(e.Value),
			Samples: samples,
		}, true
	})

	// Sort by frequency (descending)
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Count > patterns[j].Count
	})

	// Return top 5 patterns
	return lo.Slice(patterns, 0, 5)
}

// PatternMatch represents a detected error pattern
type PatternMatch struct {
	Pattern string   `json:"pattern"`
	Count   int      `json:"count"`
	Samples []string `json:"samples"`
}

// SummaryOutput wraps a summary for NDJSON output with timing
type SummaryOutput struct {
	Type      string             `json:"type"`
	Timestamp time.Time          `json:"timestamp"`
	Summary   *domain.LogSummary `json:"summary"`
	Patterns  []PatternMatch     `json:"patterns,omitempty"`
}

// NewSummaryOutput creates a summary output wrapper
func NewSummaryOutput(summary *domain.LogSummary, patterns []PatternMatch) *SummaryOutput {
	return &SummaryOutput{
		Type:      "analysis",
		Timestamp: time.Now(),
		Summary:   summary,
		Patterns:  patterns,
	}
}

// EnhancedSummaryOutput includes enhanced pattern info with known/new status
type EnhancedSummaryOutput struct {
	Type             string                 `json:"type"`
	Timestamp        time.Time              `json:"timestamp"`
	Summary          *domain.LogSummary     `json:"summary"`
	Patterns         []EnhancedPatternMatch `json:"patterns,omitempty"`
	NewPatternCount  int                    `json:"new_pattern_count"`
	KnownPatternCount int                   `json:"known_pattern_count"`
}

// NewEnhancedSummaryOutput creates an enhanced summary output wrapper
func NewEnhancedSummaryOutput(summary *domain.LogSummary, patterns []EnhancedPatternMatch) *EnhancedSummaryOutput {
	newCount := 0
	knownCount := 0
	for _, p := range patterns {
		if p.IsNew {
			newCount++
		} else {
			knownCount++
		}
	}

	return &EnhancedSummaryOutput{
		Type:              "analysis",
		Timestamp:         time.Now(),
		Summary:           summary,
		Patterns:          patterns,
		NewPatternCount:   newCount,
		KnownPatternCount: knownCount,
	}
}
