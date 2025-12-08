package output

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vedranburojevic/xcw/internal/domain"
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
	// Remove hex addresses
	msg = regexp.MustCompile(`0x[0-9a-fA-F]+`).ReplaceAllString(msg, "<addr>")
	// Remove numbers
	msg = regexp.MustCompile(`\d+`).ReplaceAllString(msg, "<n>")
	// Remove UUIDs
	msg = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`).ReplaceAllString(msg, "<uuid>")

	// Truncate long messages
	if len(msg) > 100 {
		msg = msg[:100] + "..."
	}

	return strings.TrimSpace(msg)
}

// getTopMessages returns the top N messages by frequency
func (a *Analyzer) getTopMessages(counts map[string]int, limit int) []string {
	type kv struct {
		msg   string
		count int
	}

	var pairs []kv
	for msg, count := range counts {
		pairs = append(pairs, kv{msg, count})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].count > pairs[j].count
	})

	if len(pairs) > limit {
		pairs = pairs[:limit]
	}

	result := make([]string, len(pairs))
	for i, p := range pairs {
		result[i] = p.msg
	}

	return result
}

// DetectPatterns finds recurring error patterns
func (a *Analyzer) DetectPatterns(entries []domain.LogEntry) []PatternMatch {
	// Group similar error messages
	errorGroups := make(map[string][]string)

	for _, entry := range entries {
		if entry.Level == domain.LogLevelError || entry.Level == domain.LogLevelFault {
			pattern := a.normalizeMessage(entry.Message)
			errorGroups[pattern] = append(errorGroups[pattern], entry.Message)
		}
	}

	// Convert to PatternMatch slice
	var patterns []PatternMatch
	for pattern, messages := range errorGroups {
		if len(messages) >= 2 { // Only report patterns that occur multiple times
			samples := messages
			if len(samples) > 3 {
				samples = samples[:3] // Limit samples
			}

			patterns = append(patterns, PatternMatch{
				Pattern: pattern,
				Count:   len(messages),
				Samples: samples,
			})
		}
	}

	// Sort by frequency
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Count > patterns[j].Count
	})

	// Return top patterns
	if len(patterns) > 5 {
		patterns = patterns[:5]
	}

	return patterns
}

// PatternMatch represents a detected error pattern
type PatternMatch struct {
	Pattern string   `json:"pattern"`
	Count   int      `json:"count"`
	Samples []string `json:"samples"`
}

// SummaryOutput wraps a summary for NDJSON output with timing
type SummaryOutput struct {
	Type      string           `json:"type"`
	Timestamp time.Time        `json:"timestamp"`
	Summary   *domain.LogSummary `json:"summary"`
	Patterns  []PatternMatch   `json:"patterns,omitempty"`
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
