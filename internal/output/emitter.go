package output

import (
	"io"

	"github.com/vburojevic/xcw/internal/domain"
)

// Emitter wraps NDJSONWriter with helpers that reuse one encoder.
type Emitter struct {
	w *NDJSONWriter
}

func NewEmitter(w io.Writer) *Emitter {
	return &Emitter{w: NewNDJSONWriter(w)}
}

func (e *Emitter) Write(entry *domain.LogEntry) error        { return e.w.Write(entry) }
func (e *Emitter) SessionStart(s *domain.SessionStart) error { return e.w.WriteSessionStart(s) }
func (e *Emitter) SessionEnd(s *domain.SessionEnd) error     { return e.w.WriteSessionEnd(s) }
func (e *Emitter) WriteSummary(s *domain.LogSummary) error   { return e.w.WriteSummary(s) }
func (e *Emitter) Error(code, msg string) error              { return e.w.WriteError(code, msg) }
func (e *Emitter) WriteWarning(msg string) error             { return e.w.WriteWarning(msg) }
func (e *Emitter) WriteHeartbeat(h *Heartbeat) error         { return e.w.WriteHeartbeat(h) }
func (e *Emitter) Ready(ts, sim, udid, app, tailID string, session int) error {
	return e.w.WriteReady(ts, sim, udid, app, tailID, session)
}
func (e *Emitter) ClearBuffer(reason, tailID string, session int) error {
	return e.w.WriteClearBuffer(reason, tailID, session)
}
func (e *Emitter) AgentHints(tailID string, session int, hints []string) error {
	return e.w.WriteAgentHints(tailID, session, hints)
}
func (e *Emitter) Metadata(version, commit, buildDate string) error {
	return e.w.WriteMetadata(version, commit, buildDate)
}
func (e *Emitter) Cutoff(reason, tailID string, session, total int) error {
	return e.w.WriteCutoff(reason, tailID, session, total)
}
func (e *Emitter) Rotation(path, tailID string, session int) error {
	return e.w.WriteRotation(path, tailID, session)
}
func (e *Emitter) WriteReconnect(msg, tailID, severity string) error {
	return e.w.WriteReconnect(msg, tailID, severity)
}
func (e *Emitter) SessionDebug(sd *SessionDebugOutput) error { return e.w.WriteSessionDebug(sd) }
