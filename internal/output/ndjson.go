package output

import (
	"encoding/json"
	"io"
	"time"

	"github.com/vedranburojevic/xcw/internal/domain"
)

// NDJSONWriter writes log entries as NDJSON
type NDJSONWriter struct {
	w       io.Writer
	encoder *json.Encoder
}

// NewNDJSONWriter creates a new NDJSON writer
func NewNDJSONWriter(w io.Writer) *NDJSONWriter {
	return &NDJSONWriter{
		w:       w,
		encoder: json.NewEncoder(w),
	}
}

// OutputEntry is the simplified NDJSON output format
type OutputEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Process   string `json:"process"`
	PID       int    `json:"pid"`
	Subsystem string `json:"subsystem,omitempty"`
	Category  string `json:"category,omitempty"`
	Message   string `json:"message"`
}

// Write outputs a single log entry as NDJSON
func (w *NDJSONWriter) Write(entry *domain.LogEntry) error {
	out := OutputEntry{
		Timestamp: entry.Timestamp.Format(time.RFC3339Nano),
		Level:     string(entry.Level),
		Process:   entry.Process,
		PID:       entry.PID,
		Subsystem: entry.Subsystem,
		Category:  entry.Category,
		Message:   entry.Message,
	}
	return w.encoder.Encode(out)
}

// WriteSummary outputs a summary marker
func (w *NDJSONWriter) WriteSummary(summary *domain.LogSummary) error {
	return w.encoder.Encode(summary)
}

// WriteError outputs an error
func (w *NDJSONWriter) WriteError(code, message string) error {
	return w.encoder.Encode(domain.NewErrorOutput(code, message))
}

// WriteRaw outputs raw JSON data
func (w *NDJSONWriter) WriteRaw(v interface{}) error {
	return w.encoder.Encode(v)
}

// TextWriter writes log entries as formatted text
type TextWriter struct {
	w io.Writer
}

// NewTextWriter creates a new text writer
func NewTextWriter(w io.Writer) *TextWriter {
	return &TextWriter{w: w}
}

// Write outputs a single log entry as text
func (w *TextWriter) Write(entry *domain.LogEntry) error {
	levelIndicator := getLevelIndicator(entry.Level)
	timestamp := entry.Timestamp.Format("15:04:05.000")

	line := timestamp + " " + levelIndicator + " [" + entry.Process + "] "
	if entry.Subsystem != "" {
		line += entry.Subsystem
		if entry.Category != "" {
			line += "/" + entry.Category
		}
		line += ": "
	}
	line += entry.Message + "\n"

	_, err := io.WriteString(w.w, line)
	return err
}

// WriteSummary outputs a summary as text
func (w *TextWriter) WriteSummary(summary *domain.LogSummary) error {
	line := "\n--- Summary ---\n"
	line += "Total: " + itoa(summary.TotalCount) + " | "
	line += "Errors: " + itoa(summary.ErrorCount) + " | "
	line += "Faults: " + itoa(summary.FaultCount) + "\n"
	_, err := io.WriteString(w.w, line)
	return err
}

// WriteError outputs an error as text
func (w *TextWriter) WriteError(code, message string) error {
	_, err := io.WriteString(w.w, "Error ["+code+"]: "+message+"\n")
	return err
}

func getLevelIndicator(level domain.LogLevel) string {
	switch level {
	case domain.LogLevelDebug:
		return "DBG"
	case domain.LogLevelInfo:
		return "INF"
	case domain.LogLevelDefault:
		return "DEF"
	case domain.LogLevelError:
		return "ERR"
	case domain.LogLevelFault:
		return "FLT"
	default:
		return "???"
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	var buf [20]byte
	pos := len(buf)
	negative := i < 0
	if negative {
		i = -i
	}

	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}

	if negative {
		pos--
		buf[pos] = '-'
	}

	return string(buf[pos:])
}
