package simulator

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/vburojevic/xcw/internal/domain"
)

// Parser parses raw NDJSON log lines into structured LogEntry
type Parser struct {
	onTimestampError func(raw string, err error)
}

// NewParser creates a new log parser
func NewParser() *Parser {
	return &Parser{}
}

// SetTimestampErrorHandler sets a callback for timestamp parse failures.
// Pass nil to disable.
func (p *Parser) SetTimestampErrorHandler(fn func(raw string, err error)) {
	p.onTimestampError = fn
}

// Parse converts a raw NDJSON line to a LogEntry
func (p *Parser) Parse(line []byte) (*domain.LogEntry, error) {
	var raw domain.RawLogEntry
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, err
	}

	// Skip non-log events
	if raw.EventType != "logEvent" && raw.EventType != "" {
		return nil, nil
	}

	// Parse timestamp: "2025-12-08 22:11:55.808033+0100"
	ts, err := parseTimestamp(raw.Timestamp)
	if err != nil {
		if p.onTimestampError != nil {
			p.onTimestampError(raw.Timestamp, err)
		}
		ts = time.Now() // Fallback to current time
	}

	// Extract process name from path
	processName := filepath.Base(raw.ProcessImagePath)

	return &domain.LogEntry{
		Timestamp:        ts,
		Level:            domain.ParseLogLevel(raw.MessageType),
		Process:          processName,
		PID:              raw.ProcessID,
		TID:              raw.ThreadID,
		Subsystem:        raw.Subsystem,
		Category:         raw.Category,
		Message:          raw.EventMessage,
		ProcessPath:      raw.ProcessImagePath,
		ProcessImageUUID: raw.ProcessImageUUID,
		SenderPath:       raw.SenderImagePath,
		EventType:        raw.EventType,
	}, nil
}

// parseTimestamp handles the Apple log timestamp format
func parseTimestamp(s string) (time.Time, error) {
	// Apple unified log format: "2006-01-02 15:04:05[.fraction]+ZZZZ"
	// Fractional seconds may be 1-9 digits; offset is numeric without colon.
	layouts := []string{
		"2006-01-02 15:04:05.999999999-0700",
		"2006-01-02 15:04:05-0700",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized timestamp: %q", s)
}
