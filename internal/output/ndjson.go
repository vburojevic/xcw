package output

import (
	"encoding/json"
	"io"
	"time"

	"github.com/vburojevic/xcw/internal/domain"
)

// NDJSONWriter writes log entries as NDJSON
type NDJSONWriter struct {
	w       io.Writer
	encoder *json.Encoder
}

// NewNDJSONWriter creates a new NDJSON writer
func NewNDJSONWriter(w io.Writer) *NDJSONWriter {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false) // keep logs unescaped and avoid extra allocations
	return &NDJSONWriter{
		w:       w,
		encoder: enc,
	}
}

// OutputEntry is the simplified NDJSON output format
type OutputEntry struct {
	Type          string `json:"type"`          // Always "log"
	SchemaVersion int    `json:"schemaVersion"` // Schema version for compatibility
	Timestamp     string `json:"timestamp"`
	Level         string `json:"level"`
	Process       string `json:"process"`
	PID           int    `json:"pid"`
	Subsystem     string `json:"subsystem,omitempty"`
	Category      string `json:"category,omitempty"`
	Message       string `json:"message"`
	Session       int    `json:"session,omitempty"` // Session number (1, 2, 3...)
	TailID        string `json:"tail_id,omitempty"` // Tail invocation ID
}

// Heartbeat is a keepalive message for AI agents
type Heartbeat struct {
	Type              string `json:"type"`
	SchemaVersion     int    `json:"schemaVersion"`
	Timestamp         string `json:"timestamp"`
	UptimeSeconds     int64  `json:"uptime_seconds"`
	LogsSinceLast     int    `json:"logs_since_last"`
	TailID            string `json:"tail_id,omitempty"`
	ContractVersion   int    `json:"contract_version,omitempty"`
	LatestSession     int    `json:"latest_session,omitempty"`
	LastSeenTimestamp string `json:"last_seen_timestamp,omitempty"`
}

// InfoOutput represents an informational message
type InfoOutput struct {
	Type          string `json:"type"` // Always "info"
	SchemaVersion int    `json:"schemaVersion"`
	Message       string `json:"message"`
	Simulator     string `json:"simulator,omitempty"`
	UDID          string `json:"udid,omitempty"`
	Since         string `json:"since,omitempty"`
	Mode          string `json:"mode,omitempty"`
}

// WarningOutput represents a warning message
type WarningOutput struct {
	Type          string `json:"type"` // Always "warning"
	SchemaVersion int    `json:"schemaVersion"`
	Message       string `json:"message"`
}

// MetadataOutput describes runtime/tool metadata for agents
type MetadataOutput struct {
	Type            string `json:"type"` // Always "metadata"
	SchemaVersion   int    `json:"schemaVersion"`
	Version         string `json:"version"`
	Commit          string `json:"commit"`
	BuildDate       string `json:"build_date,omitempty"`
	ContractVersion int    `json:"contract_version,omitempty"`
}

// CutoffOutput describes an intentional stream cutoff
type CutoffOutput struct {
	Type          string `json:"type"` // Always "cutoff_reached"
	SchemaVersion int    `json:"schemaVersion"`
	Reason        string `json:"reason"`
	TailID        string `json:"tail_id,omitempty"`
	Session       int    `json:"session,omitempty"`
	TotalLogs     int    `json:"total_logs,omitempty"`
}

// RotationOutput describes a file rotation event for agents.
type RotationOutput struct {
	Type          string `json:"type"` // Always "rotation"
	SchemaVersion int    `json:"schemaVersion"`
	Path          string `json:"path"`
	TailID        string `json:"tail_id,omitempty"`
	Session       int    `json:"session,omitempty"`
}

// ReconnectNotice signals a stream reconnect
type ReconnectNotice struct {
	Type          string `json:"type"` // Always "reconnect_notice"
	SchemaVersion int    `json:"schemaVersion"`
	Message       string `json:"message"`
	TailID        string `json:"tail_id,omitempty"`
	Severity      string `json:"severity,omitempty"`
}

// SessionDebugOutput surfaces verbose session transition info
type SessionDebugOutput struct {
	Type          string                 `json:"type"` // Always "session_debug"
	SchemaVersion int                    `json:"schemaVersion"`
	TailID        string                 `json:"tail_id,omitempty"`
	Session       int                    `json:"session"`
	PrevSession   int                    `json:"prev_session,omitempty"`
	PID           int                    `json:"pid"`
	PrevPID       int                    `json:"prev_pid,omitempty"`
	Reason        string                 `json:"reason"`
	Summary       map[string]interface{} `json:"summary,omitempty"`
}

// TmuxOutput represents tmux session information
type TmuxOutput struct {
	Type          string `json:"type"` // Always "tmux"
	SchemaVersion int    `json:"schemaVersion"`
	Session       string `json:"session"`
	Attach        string `json:"attach"`
}

// TriggerOutput represents a trigger event
type TriggerOutput struct {
	Type          string `json:"type"` // Always "trigger"
	SchemaVersion int    `json:"schemaVersion"`
	Trigger       string `json:"trigger"`
	Command       string `json:"command"`
	Message       string `json:"message"`
}

// TriggerErrorOutput represents a trigger execution error
type TriggerErrorOutput struct {
	Type          string `json:"type"` // Always "trigger_error"
	SchemaVersion int    `json:"schemaVersion"`
	Command       string `json:"command"`
	Error         string `json:"error"`
}

// ClearBufferOutput instructs consumers to discard cached state at session boundaries
type ClearBufferOutput struct {
	Type          string   `json:"type"` // Always "clear_buffer"
	SchemaVersion int      `json:"schemaVersion"`
	Reason        string   `json:"reason"`
	TailID        string   `json:"tail_id,omitempty"`
	Session       int      `json:"session,omitempty"`
	Hints         []string `json:"hints,omitempty"`
}

// ReadyOutput signals that log capture is active and ready
type ReadyOutput struct {
	Type            string `json:"type"` // Always "ready"
	SchemaVersion   int    `json:"schemaVersion"`
	Timestamp       string `json:"timestamp"`
	Simulator       string `json:"simulator"`
	UDID            string `json:"udid"`
	App             string `json:"app"`
	TailID          string `json:"tail_id,omitempty"`
	Session         int    `json:"session,omitempty"`
	ContractVersion int    `json:"contract_version,omitempty"`
}

// AgentHintsOutput provides runtime contract guidance for AI agents
type AgentHintsOutput struct {
	Type             string   `json:"type"` // Always "agent_hints"
	SchemaVersion    int      `json:"schemaVersion"`
	TailID           string   `json:"tail_id,omitempty"`
	Session          int      `json:"session,omitempty"`
	ContractVersion  int      `json:"contract_version"`
	Hints            []string `json:"hints"`
	RecommendedScope string   `json:"recommended_scope,omitempty"`
}

// Write outputs a single log entry as NDJSON
func (w *NDJSONWriter) Write(entry *domain.LogEntry) error {
	out := OutputEntry{
		Type:          "log",
		SchemaVersion: SchemaVersion,
		Timestamp:     entry.Timestamp.Format(time.RFC3339Nano),
		Level:         string(entry.Level),
		Process:       entry.Process,
		PID:           entry.PID,
		Subsystem:     entry.Subsystem,
		Category:      entry.Category,
		Message:       entry.Message,
		Session:       entry.Session,
		TailID:        entry.TailID,
	}
	return w.encoder.Encode(out)
}

// WriteSessionStart outputs a session start event
func (w *NDJSONWriter) WriteSessionStart(session *domain.SessionStart) error {
	return w.encoder.Encode(session)
}

// WriteSessionEnd outputs a session end event
func (w *NDJSONWriter) WriteSessionEnd(session *domain.SessionEnd) error {
	return w.encoder.Encode(session)
}

// WriteSummary outputs a summary marker
func (w *NDJSONWriter) WriteSummary(summary *domain.LogSummary) error {
	summary.SchemaVersion = SchemaVersion
	return w.encoder.Encode(summary)
}

// WriteError outputs an error
func (w *NDJSONWriter) WriteError(code, message string, hint ...string) error {
	err := domain.NewErrorOutput(code, message)
	if len(hint) > 0 {
		err.Hint = hint[0]
	}
	err.SchemaVersion = SchemaVersion
	return w.encoder.Encode(err)
}

// WriteRaw outputs raw JSON data
func (w *NDJSONWriter) WriteRaw(v interface{}) error {
	return w.encoder.Encode(v)
}

// WriteHeartbeat outputs a heartbeat keepalive message
func (w *NDJSONWriter) WriteHeartbeat(h *Heartbeat) error {
	return w.encoder.Encode(h)
}

// WriteInfo outputs an informational message
func (w *NDJSONWriter) WriteInfo(message, simulator, udid, since, mode string) error {
	return w.encoder.Encode(&InfoOutput{
		Type:          "info",
		SchemaVersion: SchemaVersion,
		Message:       message,
		Simulator:     simulator,
		UDID:          udid,
		Since:         since,
		Mode:          mode,
	})
}

// WriteWarning outputs a warning message
func (w *NDJSONWriter) WriteWarning(message string) error {
	return w.encoder.Encode(&WarningOutput{
		Type:          "warning",
		SchemaVersion: SchemaVersion,
		Message:       message,
	})
}

// WriteMetadata outputs runtime metadata
func (w *NDJSONWriter) WriteMetadata(version, commit, buildDate string) error {
	return w.encoder.Encode(&MetadataOutput{
		Type:            "metadata",
		SchemaVersion:   SchemaVersion,
		Version:         version,
		Commit:          commit,
		BuildDate:       buildDate,
		ContractVersion: 1,
	})
}

// WriteCutoff outputs a cutoff marker
func (w *NDJSONWriter) WriteCutoff(reason, tailID string, session, total int) error {
	return w.encoder.Encode(&CutoffOutput{
		Type:          "cutoff_reached",
		SchemaVersion: SchemaVersion,
		Reason:        reason,
		TailID:        tailID,
		Session:       session,
		TotalLogs:     total,
	})
}

// WriteRotation outputs a rotation event indicating the active output file path.
func (w *NDJSONWriter) WriteRotation(path, tailID string, session int) error {
	return w.encoder.Encode(&RotationOutput{
		Type:          "rotation",
		SchemaVersion: SchemaVersion,
		Path:          path,
		TailID:        tailID,
		Session:       session,
	})
}

// WriteReconnect outputs a reconnect notice
func (w *NDJSONWriter) WriteReconnect(message, tailID, severity string) error {
	return w.encoder.Encode(&ReconnectNotice{
		Type:          "reconnect_notice",
		SchemaVersion: SchemaVersion,
		Message:       message,
		TailID:        tailID,
		Severity:      severity,
	})
}

// WriteSessionDebug outputs a verbose session transition for diagnostics
func (w *NDJSONWriter) WriteSessionDebug(sd *SessionDebugOutput) error {
	sd.SchemaVersion = SchemaVersion
	return w.encoder.Encode(sd)
}

// WriteTmux outputs tmux session information
func (w *NDJSONWriter) WriteTmux(session, attach string) error {
	return w.encoder.Encode(&TmuxOutput{
		Type:          "tmux",
		SchemaVersion: SchemaVersion,
		Session:       session,
		Attach:        attach,
	})
}

// WriteTrigger outputs a trigger event
func (w *NDJSONWriter) WriteTrigger(trigger, command, message string) error {
	return w.encoder.Encode(&TriggerOutput{
		Type:          "trigger",
		SchemaVersion: SchemaVersion,
		Trigger:       trigger,
		Command:       command,
		Message:       message,
	})
}

// WriteTriggerError outputs a trigger execution error
func (w *NDJSONWriter) WriteTriggerError(command, errMsg string) error {
	return w.encoder.Encode(&TriggerErrorOutput{
		Type:          "trigger_error",
		SchemaVersion: SchemaVersion,
		Command:       command,
		Error:         errMsg,
	})
}

// WriteReady outputs a ready signal indicating log capture is active
func (w *NDJSONWriter) WriteReady(timestamp, simulator, udid, app, tailID string, session int) error {
	return w.encoder.Encode(&ReadyOutput{
		Type:            "ready",
		SchemaVersion:   SchemaVersion,
		Timestamp:       timestamp,
		Simulator:       simulator,
		UDID:            udid,
		App:             app,
		TailID:          tailID,
		Session:         session,
		ContractVersion: 1,
	})
}

// WriteClearBuffer emits a cache/reset hint
func (w *NDJSONWriter) WriteClearBuffer(reason string, tailID string, session int) error {
	return w.encoder.Encode(&ClearBufferOutput{
		Type:          "clear_buffer",
		SchemaVersion: SchemaVersion,
		Reason:        reason,
		TailID:        tailID,
		Session:       session,
		Hints:         []string{"reset caches/state for tail_id", "switch to latest session"},
	})
}

// WriteAgentHints outputs guidance for AI agents
func (w *NDJSONWriter) WriteAgentHints(tailID string, session int, hints []string) error {
	return w.encoder.Encode(&AgentHintsOutput{
		Type:             "agent_hints",
		SchemaVersion:    SchemaVersion,
		TailID:           tailID,
		Session:          session,
		ContractVersion:  1,
		Hints:            hints,
		RecommendedScope: "tail_id + latest session only",
	})
}

// TextWriter writes log entries as formatted text
type TextWriter struct {
	w io.Writer
}

// NewTextWriter creates a new text writer
func NewTextWriter(w io.Writer) *TextWriter {
	return &TextWriter{w: w}
}

// Write outputs a single log entry as styled text
func (w *TextWriter) Write(entry *domain.LogEntry) error {
	// Use lipgloss styled output
	levelStr := string(entry.Level)
	levelIndicator := LevelIndicator(levelStr)
	timestamp := Styles.Timestamp.Render(entry.Timestamp.Format("15:04:05.000"))
	process := Styles.Process.Render("[" + entry.Process + "]")

	line := timestamp + " " + levelIndicator + " " + process + " "
	if entry.Subsystem != "" {
		subsystem := Styles.Subsystem.Render(entry.Subsystem)
		if entry.Category != "" {
			subsystem += "/" + entry.Category
		}
		line += subsystem + ": "
	}

	// Style message based on level
	msgStyle := LevelStyle(levelStr)
	line += msgStyle.Render(entry.Message) + "\n"

	_, err := io.WriteString(w.w, line)
	return err
}

// WriteSummary outputs a styled summary
func (w *TextWriter) WriteSummary(summary *domain.LogSummary) error {
	header := Styles.Header.Render("Summary")
	line := "\n" + header + "\n"
	line += Styles.Label.Render("Total: ") + Styles.Value.Render(itoa(summary.TotalCount)) + " | "

	// Color errors/faults based on count
	if summary.ErrorCount > 0 {
		line += Styles.Warning.Render("Errors: "+itoa(summary.ErrorCount)) + " | "
	} else {
		line += Styles.Label.Render("Errors: ") + Styles.Value.Render(itoa(summary.ErrorCount)) + " | "
	}

	if summary.FaultCount > 0 {
		line += Styles.Danger.Render("Faults: " + itoa(summary.FaultCount))
	} else {
		line += Styles.Label.Render("Faults: ") + Styles.Value.Render(itoa(summary.FaultCount))
	}
	line += "\n"

	_, err := io.WriteString(w.w, line)
	return err
}

// WriteError outputs a styled error
func (w *TextWriter) WriteError(code, message string) error {
	errorLabel := Styles.Danger.Render("Error")
	codeStr := Styles.Warning.Render("[" + code + "]")
	line := errorLabel + " " + codeStr + ": " + message + "\n"
	_, err := io.WriteString(w.w, line)
	return err
}

// WriteHeartbeat outputs a styled heartbeat
func (w *TextWriter) WriteHeartbeat(h *Heartbeat) error {
	label := Styles.Info.Render("[HEARTBEAT]")
	line := label + " " + Styles.Label.Render("uptime=") + Styles.Value.Render(itoa(int(h.UptimeSeconds))+"s")
	line += " " + Styles.Label.Render("logs_since_last=") + Styles.Value.Render(itoa(h.LogsSinceLast)) + "\n"
	_, err := io.WriteString(w.w, line)
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
