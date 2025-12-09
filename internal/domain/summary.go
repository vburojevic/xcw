package domain

import "time"

// LogSummary provides AI-friendly aggregated log statistics
type LogSummary struct {
	Type          string `json:"type"`          // Always "summary"
	SchemaVersion int    `json:"schemaVersion"` // Schema version for compatibility

	// Time window
	WindowStart time.Time `json:"windowStart"`
	WindowEnd   time.Time `json:"windowEnd"`

	// Counts
	TotalCount   int `json:"totalCount"`
	DebugCount   int `json:"debugCount"`
	InfoCount    int `json:"infoCount"`
	DefaultCount int `json:"defaultCount"`
	ErrorCount   int `json:"errorCount"`
	FaultCount   int `json:"faultCount"`

	// AI markers
	HasErrors bool `json:"hasErrors"`
	HasFaults bool `json:"hasFaults"`

	// Pattern detection
	TopErrors []string `json:"topErrors,omitempty"`
	TopFaults []string `json:"topFaults,omitempty"`

	// Rate information
	ErrorRate float64 `json:"errorRate"` // errors per minute
}

// NewLogSummary creates a new empty summary
func NewLogSummary() *LogSummary {
	return &LogSummary{
		Type: "summary",
	}
}

// ErrorOutput represents a structured error for NDJSON output
type ErrorOutput struct {
	Type          string `json:"type"`          // Always "error"
	SchemaVersion int    `json:"schemaVersion"` // Schema version for compatibility
	Code          string `json:"code"`          // Machine-readable error code
	Message       string `json:"message"`       // Human-readable message
}

// NewErrorOutput creates a new error output
// Note: SchemaVersion should be set by the caller (output package)
func NewErrorOutput(code, message string) *ErrorOutput {
	return &ErrorOutput{
		Type:    "error",
		Code:    code,
		Message: message,
	}
}
