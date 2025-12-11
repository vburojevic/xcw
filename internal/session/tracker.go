package session

import (
	"sync"
	"time"

	"github.com/vburojevic/xcw/internal/domain"
)

// Tracker monitors log entries for PID changes to detect app relaunches
type Tracker struct {
	mu                sync.Mutex
	currentSession    int
	currentPID        int
	currentBinaryUUID string
	sessionStart      time.Time
	logCount          int
	errorCount        int
	faultCount        int
	app               string
	simulator         string
	udid              string
	tailID            string
	appVersion        string
	appBuild          string
	initialized       bool
}

// SessionChange contains events emitted when a session changes
type SessionChange struct {
	EndSession   *domain.SessionEnd
	StartSession *domain.SessionStart
}

// NewTracker creates a new session tracker
func NewTracker(app, simulator, udid, tailID, version, build string) *Tracker {
	return &Tracker{
		app:        app,
		simulator:  simulator,
		udid:       udid,
		tailID:     tailID,
		appVersion: version,
		appBuild:   build,
	}
}

// CheckEntry processes a log entry and returns a SessionChange if the app was relaunched
func (t *Tracker) CheckEntry(entry *domain.LogEntry) *SessionChange {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Check PID for ALL logs - the streamer already filters by app, so any log
	// we receive is relevant. Don't filter by subsystem since many apps emit
	// logs with empty subsystem, process name, or Apple's default subsystems.
	pid := entry.PID

	// First entry - initialize session
	if !t.initialized {
		t.initialized = true
		t.currentSession = 1
		t.currentPID = pid
		t.currentBinaryUUID = entry.ProcessImageUUID
		t.sessionStart = time.Now()
		t.logCount = 1
		t.updateCounts(entry)

		// Return initial session start
		return &SessionChange{
			StartSession: domain.NewSessionStartWithMeta(
				t.currentSession,
				pid,
				0, // no previous PID
				t.app,
				t.simulator,
				t.udid,
				t.tailID,
				t.appVersion,
				t.appBuild,
				entry.ProcessImageUUID,
				"",
			),
		}
	}

	if t.shouldStartNewSession(pid, entry.ProcessImageUUID) {
		previousPID := t.currentPID
		previousSession := t.currentSession

		// Create session end summary
		summary := domain.SessionSummary{
			TotalLogs:       t.logCount,
			Errors:          t.errorCount,
			Faults:          t.faultCount,
			DurationSeconds: int(time.Since(t.sessionStart).Seconds()),
		}

		// Start new session
		t.currentSession++
		t.currentPID = pid
		t.currentBinaryUUID = entry.ProcessImageUUID
		t.sessionStart = time.Now()
		t.logCount = 1
		t.errorCount = 0
		t.faultCount = 0
		t.updateCounts(entry)

		return &SessionChange{
			EndSession: domain.NewSessionEndWithMeta(previousSession, previousPID, summary, t.tailID),
			StartSession: domain.NewSessionStartWithMeta(
				t.currentSession,
				pid,
				previousPID,
				t.app,
				t.simulator,
				t.udid,
				t.tailID,
				t.appVersion,
				t.appBuild,
				entry.ProcessImageUUID,
				"",
			),
		}
	}

	// Same session - just increment counts
	t.logCount++
	t.currentBinaryUUID = entry.ProcessImageUUID
	t.updateCounts(entry)
	return nil
}

// shouldStartNewSession decides if PID or binary UUID change indicates a relaunch.
func (t *Tracker) shouldStartNewSession(pid int, imageUUID string) bool {
	binaryChanged := imageUUID != "" && imageUUID != t.currentBinaryUUID
	pidChanged := pid != t.currentPID && pid > 0
	return pidChanged || binaryChanged
}

// updateCounts updates error/fault counts based on log level
func (t *Tracker) updateCounts(entry *domain.LogEntry) {
	switch entry.Level {
	case domain.LogLevelError:
		t.errorCount++
	case domain.LogLevelFault:
		t.faultCount++
	}
}

// CurrentSession returns the current session number
func (t *Tracker) CurrentSession() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.currentSession
}

// GetFinalSummary returns a summary for the current session (for stream end)
func (t *Tracker) GetFinalSummary() *domain.SessionEnd {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.initialized {
		return nil
	}

	return domain.NewSessionEndWithMeta(
		t.currentSession,
		t.currentPID,
		domain.SessionSummary{
			TotalLogs:       t.logCount,
			Errors:          t.errorCount,
			Faults:          t.faultCount,
			DurationSeconds: int(time.Since(t.sessionStart).Seconds()),
		},
		t.tailID,
	)
}

// ForceRollover ends the current session and starts a new one using the same PID.
// Useful for idle timeouts or manual boundaries.
func (t *Tracker) ForceRollover(alert string) *SessionChange {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.initialized {
		return nil
	}

	previousSession := t.currentSession
	previousPID := t.currentPID
	prevBinary := t.currentBinaryUUID

	summary := domain.SessionSummary{
		TotalLogs:       t.logCount,
		Errors:          t.errorCount,
		Faults:          t.faultCount,
		DurationSeconds: int(time.Since(t.sessionStart).Seconds()),
	}

	// Start new session with same PID; counters reset
	t.currentSession++
	t.sessionStart = time.Now()
	t.logCount = 0
	t.errorCount = 0
	t.faultCount = 0

	return &SessionChange{
		EndSession: domain.NewSessionEndWithMeta(previousSession, previousPID, summary, t.tailID),
		StartSession: domain.NewSessionStartWithMeta(
			t.currentSession,
			previousPID,
			previousPID,
			t.app,
			t.simulator,
			t.udid,
			t.tailID,
			t.appVersion,
			t.appBuild,
			prevBinary,
			alert,
		),
	}
}

// Stats returns current session statistics
func (t *Tracker) Stats() (session, pid, logs, errors, faults int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.currentSession, t.currentPID, t.logCount, t.errorCount, t.faultCount
}
