package simulator

import (
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/vedranburojevic/xcw/internal/domain"
)

// RawNDJSONEntry matches the native NDJSON structure from `log stream --style ndjson`
type RawNDJSONEntry struct {
	Timestamp        string `json:"timestamp"`
	MessageType      string `json:"messageType"`
	EventType        string `json:"eventType"`
	EventMessage     string `json:"eventMessage"`
	ProcessID        int    `json:"processID"`
	ProcessImagePath string `json:"processImagePath"`
	ProcessImageUUID string `json:"processImageUUID"`
	Subsystem        string `json:"subsystem"`
	Category         string `json:"category"`
	ThreadID         int    `json:"threadID"`
	FormatString     string `json:"formatString"`
	UserID           int    `json:"userID"`
	SenderImagePath  string `json:"senderImagePath"`
	SenderImageUUID  string `json:"senderImageUUID"`
	TraceID          int64  `json:"traceID"`
	MachTimestamp    int64  `json:"machTimestamp"`
}

// Parser parses raw NDJSON log lines into structured LogEntry
type Parser struct{}

// NewParser creates a new log parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse converts a raw NDJSON line to a LogEntry
func (p *Parser) Parse(line []byte) (*domain.LogEntry, error) {
	var raw RawNDJSONEntry
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
		ts = time.Now() // Fallback to current time
	}

	// Extract process name from path
	processName := filepath.Base(raw.ProcessImagePath)

	return &domain.LogEntry{
		Timestamp:   ts,
		Level:       domain.ParseLogLevel(raw.MessageType),
		Process:     processName,
		PID:         raw.ProcessID,
		TID:         raw.ThreadID,
		Subsystem:   raw.Subsystem,
		Category:    raw.Category,
		Message:     raw.EventMessage,
		ProcessPath: raw.ProcessImagePath,
		SenderPath:  raw.SenderImagePath,
		EventType:   raw.EventType,
	}, nil
}

// parseTimestamp handles the Apple log timestamp format
func parseTimestamp(s string) (time.Time, error) {
	// Format: "2025-12-08 22:11:55.808033+0100"
	layouts := []string{
		"2006-01-02 15:04:05.000000-0700",
		"2006-01-02 15:04:05.000000+0100",
		"2006-01-02 15:04:05-0700",
		"2006-01-02 15:04:05.999999-0700",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	// Try to parse with fixed offset format
	return time.Parse("2006-01-02 15:04:05.999999-0700", s)
}
