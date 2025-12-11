package domain

import "time"

// LogLevel represents the unified logging system levels
type LogLevel string

const (
	LogLevelDebug   LogLevel = "Debug"
	LogLevelInfo    LogLevel = "Info"
	LogLevelDefault LogLevel = "Default"
	LogLevelError   LogLevel = "Error"
	LogLevelFault   LogLevel = "Fault"
)

// LogLevelPriority returns the priority of a log level (higher = more severe)
func (l LogLevel) Priority() int {
	switch l {
	case LogLevelDebug:
		return 0
	case LogLevelInfo:
		return 1
	case LogLevelDefault:
		return 2
	case LogLevelError:
		return 3
	case LogLevelFault:
		return 4
	default:
		return 2
	}
}

// ParseLogLevel converts a string to LogLevel
func ParseLogLevel(s string) LogLevel {
	switch s {
	case "Debug", "debug":
		return LogLevelDebug
	case "Info", "info":
		return LogLevelInfo
	case "Default", "default":
		return LogLevelDefault
	case "Error", "error":
		return LogLevelError
	case "Fault", "fault":
		return LogLevelFault
	default:
		return LogLevelDefault
	}
}

// LogEntry represents a parsed log message from the unified logging system
type LogEntry struct {
	Timestamp        time.Time `json:"timestamp"`
	Level            LogLevel  `json:"level"`
	Process          string    `json:"process"`
	PID              int       `json:"pid"`
	TID              int       `json:"tid,omitempty"`
	Subsystem        string    `json:"subsystem,omitempty"`
	Category         string    `json:"category,omitempty"`
	Message          string    `json:"message"`
	ProcessPath      string    `json:"processPath,omitempty"`
	ProcessImageUUID string    `json:"processImageUUID,omitempty"`
	SenderPath       string    `json:"senderPath,omitempty"`
	EventType        string    `json:"eventType,omitempty"`
	TailID           string    `json:"tail_id,omitempty"`

	// Session tracking (populated when session tracking is active)
	Session int `json:"session,omitempty"` // Session number (1, 2, 3...)

	// Dedupe metadata (populated when --dedupe is used)
	DedupeCount int    `json:"dedupe_count,omitempty"` // Number of collapsed duplicates
	DedupeFirst string `json:"dedupe_first,omitempty"` // First occurrence timestamp
	DedupeLast  string `json:"dedupe_last,omitempty"`  // Last occurrence timestamp
}

// RawLogEntry matches the native NDJSON structure from `log stream --style ndjson`
type RawLogEntry struct {
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
	TraceID          int64  `json:"traceID,omitempty"`
	MachTimestamp    int64  `json:"machTimestamp,omitempty"`
}
