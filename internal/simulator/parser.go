package simulator

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/tidwall/gjson"
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
	if !gjson.ValidBytes(line) {
		return nil, fmt.Errorf("invalid json")
	}

	// Skip non-log events
	eventType := gjson.GetBytes(line, "eventType").String()
	if eventType != "logEvent" && eventType != "" {
		return nil, nil
	}

	// Parse timestamp: "2025-12-08 22:11:55.808033+0100"
	tsRaw := gjson.GetBytes(line, "timestamp").String()
	ts, err := parseTimestamp(tsRaw)
	if err != nil {
		if p.onTimestampError != nil {
			p.onTimestampError(tsRaw, err)
		}
		ts = time.Now() // Fallback to current time
	}

	// Extract process name from path
	processImagePath := gjson.GetBytes(line, "processImagePath").String()
	processName := ""
	if processImagePath != "" {
		processName = filepath.Base(processImagePath)
	} else {
		// Some log lines may omit processImagePath; best-effort fallback.
		processName = gjson.GetBytes(line, "process").String()
	}

	return &domain.LogEntry{
		Timestamp:        ts,
		Level:            domain.ParseLogLevel(gjson.GetBytes(line, "messageType").String()),
		Process:          processName,
		PID:              int(gjson.GetBytes(line, "processID").Int()),
		TID:              int(gjson.GetBytes(line, "threadID").Int()),
		Subsystem:        gjson.GetBytes(line, "subsystem").String(),
		Category:         gjson.GetBytes(line, "category").String(),
		Message:          gjson.GetBytes(line, "eventMessage").String(),
		ProcessPath:      processImagePath,
		ProcessImageUUID: gjson.GetBytes(line, "processImageUUID").String(),
		SenderPath:       gjson.GetBytes(line, "senderImagePath").String(),
		EventType:        eventType,
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
