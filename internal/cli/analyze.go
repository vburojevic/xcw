package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/vburojevic/xcw/internal/domain"
	"github.com/vburojevic/xcw/internal/output"
)

// AnalyzeCmd analyzes a recorded NDJSON log file
type AnalyzeCmd struct {
	File            string `arg:"" required:"" help:"NDJSON log file to analyze"`
	PersistPatterns bool   `help:"Save detected patterns for future reference (marks new vs known)"`
	PatternFile     string `help:"Custom pattern file path (default: ~/.xcw/patterns.json)"`
}

// Run executes the analyze command
func (c *AnalyzeCmd) Run(globals *Globals) error {
	// Open input file
	file, err := os.Open(c.File)
	if err != nil {
		return c.outputError(globals, "FILE_NOT_FOUND", fmt.Sprintf("cannot open file: %s", err))
	}
	defer file.Close()

	// Read and parse log entries
	var entries []domain.LogEntry
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry domain.LogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			// Try to detect if it's a different type (summary, heartbeat, etc.)
			var typeCheck struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(line, &typeCheck) == nil && typeCheck.Type != "" {
				// Skip non-log entries (summaries, heartbeats, etc.)
				continue
			}
			globals.Debug("Skipping unparseable line %d: %v", lineNum, err)
			continue
		}

		// Skip if no timestamp (invalid entry)
		if entry.Timestamp.IsZero() {
			continue
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return c.outputError(globals, "READ_ERROR", fmt.Sprintf("error reading file: %s", err))
	}

	if len(entries) == 0 {
		return c.outputError(globals, "NO_ENTRIES", "no valid log entries found in file")
	}

	// Analyze entries
	analyzer := output.NewAnalyzer()
	summary := analyzer.Summarize(entries)
	patterns := analyzer.DetectPatterns(entries)

	// Output results
	if globals.Format == "ndjson" {
		writer := output.NewNDJSONWriter(globals.Stdout)

		if c.PersistPatterns {
			store := output.NewPatternStore(c.PatternFile)
			enhanced := store.RecordPatterns(patterns)
			if err := store.Save(); err != nil {
				globals.Debug("Failed to save patterns: %v", err)
			}
			analysisOutput := output.NewEnhancedSummaryOutput(summary, enhanced)
			return writer.WriteRaw(analysisOutput)
		}

		analysisOutput := output.NewSummaryOutput(summary, patterns)
		return writer.WriteRaw(analysisOutput)
	}

	// Text output
	fmt.Fprintf(globals.Stdout, "Analysis of %s\n", c.File)
	fmt.Fprintln(globals.Stdout, "===================")
	fmt.Fprintln(globals.Stdout)

	// Time range
	if !summary.WindowStart.IsZero() && !summary.WindowEnd.IsZero() {
		duration := summary.WindowEnd.Sub(summary.WindowStart)
		fmt.Fprintf(globals.Stdout, "Time Range: %s to %s (%s)\n",
			summary.WindowStart.Format(time.RFC3339),
			summary.WindowEnd.Format(time.RFC3339),
			duration.Round(time.Second))
		fmt.Fprintln(globals.Stdout)
	}

	// Counts
	fmt.Fprintf(globals.Stdout, "Total entries:   %d\n", summary.TotalCount)
	fmt.Fprintf(globals.Stdout, "  Debug:         %d\n", summary.DebugCount)
	fmt.Fprintf(globals.Stdout, "  Info:          %d\n", summary.InfoCount)
	fmt.Fprintf(globals.Stdout, "  Default:       %d\n", summary.DefaultCount)
	fmt.Fprintf(globals.Stdout, "  Error:         %d\n", summary.ErrorCount)
	fmt.Fprintf(globals.Stdout, "  Fault:         %d\n", summary.FaultCount)
	fmt.Fprintln(globals.Stdout)

	if summary.ErrorRate > 0 {
		fmt.Fprintf(globals.Stdout, "Error rate: %.2f/min\n", summary.ErrorRate)
		fmt.Fprintln(globals.Stdout)
	}

	// Patterns
	if len(patterns) > 0 {
		fmt.Fprintln(globals.Stdout, "Error Patterns:")
		if c.PersistPatterns {
			store := output.NewPatternStore(c.PatternFile)
			enhanced := store.RecordPatterns(patterns)
			if err := store.Save(); err != nil {
				globals.Debug("Failed to save patterns: %v", err)
			}
			for _, p := range enhanced {
				status := "[NEW]"
				if !p.IsNew {
					status = "[KNOWN]"
				}
				fmt.Fprintf(globals.Stdout, "  %s (%dx) %s\n", status, p.Count, p.Pattern)
			}
		} else {
			for _, p := range patterns {
				fmt.Fprintf(globals.Stdout, "  (%dx) %s\n", p.Count, p.Pattern)
			}
		}
	}

	return nil
}

func (c *AnalyzeCmd) outputError(globals *Globals, code, message string) error {
	if globals.Format == "ndjson" {
		w := output.NewNDJSONWriter(globals.Stdout)
		w.WriteError(code, message)
	} else {
		fmt.Fprintf(globals.Stderr, "Error [%s]: %s\n", code, message)
	}
	return errors.New(message)
}
