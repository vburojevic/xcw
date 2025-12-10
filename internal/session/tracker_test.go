package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vburojevic/xcw/internal/domain"
)

func TestNewTracker(t *testing.T) {
	tracker := NewTracker("com.example.app", "iPhone 17 Pro", "ABC123", "tail-1", "1.0", "100")

	assert.Equal(t, 0, tracker.CurrentSession())

	session, pid, logs, errors, faults := tracker.Stats()
	assert.Equal(t, 0, session)
	assert.Equal(t, 0, pid)
	assert.Equal(t, 0, logs)
	assert.Equal(t, 0, errors)
	assert.Equal(t, 0, faults)
}

func TestTrackerFirstEntry(t *testing.T) {
	tracker := NewTracker("com.example.app", "iPhone 17 Pro", "ABC123", "tail-1", "1.0", "100")

	entry := &domain.LogEntry{
		Timestamp: time.Now(),
		Level:     domain.LogLevelInfo,
		Process:   "MyApp",
		PID:       12345,
		Subsystem: "com.example.app",
		Message:   "App started",
	}

	change := tracker.CheckEntry(entry)

	require.NotNil(t, change)
	require.NotNil(t, change.StartSession)
	assert.Nil(t, change.EndSession)

	// First session should not have alert
	assert.Empty(t, change.StartSession.Alert)
	assert.Equal(t, 1, change.StartSession.Session)
	assert.Equal(t, 12345, change.StartSession.PID)
	assert.Equal(t, 0, change.StartSession.PreviousPID)
	assert.Equal(t, "com.example.app", change.StartSession.App)
	assert.Equal(t, "iPhone 17 Pro", change.StartSession.Simulator)

	assert.Equal(t, 1, tracker.CurrentSession())
}

func TestTrackerSameSession(t *testing.T) {
	tracker := NewTracker("com.example.app", "iPhone 17 Pro", "ABC123", "tail-1", "1.0", "100")

	// First entry to initialize
	entry1 := &domain.LogEntry{
		Timestamp: time.Now(),
		Level:     domain.LogLevelInfo,
		PID:       12345,
		Subsystem: "com.example.app",
		Message:   "First message",
	}
	tracker.CheckEntry(entry1)

	// Second entry with same PID
	entry2 := &domain.LogEntry{
		Timestamp: time.Now(),
		Level:     domain.LogLevelError,
		PID:       12345,
		Subsystem: "com.example.app",
		Message:   "Error message",
	}

	change := tracker.CheckEntry(entry2)
	assert.Nil(t, change) // No session change

	session, pid, logs, errors, faults := tracker.Stats()
	assert.Equal(t, 1, session)
	assert.Equal(t, 12345, pid)
	assert.Equal(t, 2, logs)
	assert.Equal(t, 1, errors)
	assert.Equal(t, 0, faults)
}

func TestTrackerPIDChange(t *testing.T) {
	tracker := NewTracker("com.example.app", "iPhone 17 Pro", "ABC123", "tail-1", "1.0", "100")

	// First entry
	entry1 := &domain.LogEntry{
		Timestamp: time.Now(),
		Level:     domain.LogLevelInfo,
		PID:       12345,
		Subsystem: "com.example.app",
		Message:   "First session",
	}
	tracker.CheckEntry(entry1)

	// Add an error to first session
	entry2 := &domain.LogEntry{
		Timestamp: time.Now(),
		Level:     domain.LogLevelError,
		PID:       12345,
		Subsystem: "com.example.app",
		Message:   "Error in first session",
	}
	tracker.CheckEntry(entry2)

	// Entry with different PID (app relaunched)
	entry3 := &domain.LogEntry{
		Timestamp: time.Now(),
		Level:     domain.LogLevelInfo,
		PID:       67890,
		Subsystem: "com.example.app",
		Message:   "Second session",
	}

	change := tracker.CheckEntry(entry3)

	require.NotNil(t, change)
	require.NotNil(t, change.EndSession)
	require.NotNil(t, change.StartSession)

	// Check session end
	assert.Equal(t, 1, change.EndSession.Session)
	assert.Equal(t, 12345, change.EndSession.PID)
	assert.Equal(t, 2, change.EndSession.Summary.TotalLogs)
	assert.Equal(t, 1, change.EndSession.Summary.Errors)
	assert.Equal(t, 0, change.EndSession.Summary.Faults)

	// Check session start
	assert.Equal(t, "APP_RELAUNCHED", change.StartSession.Alert)
	assert.Equal(t, 2, change.StartSession.Session)
	assert.Equal(t, 67890, change.StartSession.PID)
	assert.Equal(t, 12345, change.StartSession.PreviousPID)

	assert.Equal(t, 2, tracker.CurrentSession())
}

func TestTrackerMultipleSessions(t *testing.T) {
	tracker := NewTracker("com.example.app", "iPhone 17 Pro", "ABC123", "tail-1", "1.0", "100")

	// Session 1
	tracker.CheckEntry(&domain.LogEntry{
		PID: 111, Subsystem: "com.example.app",
	})

	// Session 2
	change2 := tracker.CheckEntry(&domain.LogEntry{
		PID: 222, Subsystem: "com.example.app",
	})
	require.NotNil(t, change2)
	assert.Equal(t, 2, change2.StartSession.Session)

	// Session 3
	change3 := tracker.CheckEntry(&domain.LogEntry{
		PID: 333, Subsystem: "com.example.app",
	})
	require.NotNil(t, change3)
	assert.Equal(t, 3, change3.StartSession.Session)

	assert.Equal(t, 3, tracker.CurrentSession())
}

func TestTrackerFaultCounting(t *testing.T) {
	tracker := NewTracker("com.example.app", "iPhone 17 Pro", "ABC123", "tail-1", "1.0", "100")

	entries := []*domain.LogEntry{
		{PID: 100, Subsystem: "com.example.app", Level: domain.LogLevelInfo},
		{PID: 100, Subsystem: "com.example.app", Level: domain.LogLevelError},
		{PID: 100, Subsystem: "com.example.app", Level: domain.LogLevelFault},
		{PID: 100, Subsystem: "com.example.app", Level: domain.LogLevelFault},
		{PID: 100, Subsystem: "com.example.app", Level: domain.LogLevelError},
	}

	for _, e := range entries {
		tracker.CheckEntry(e)
	}

	session, pid, logs, errors, faults := tracker.Stats()
	assert.Equal(t, 1, session)
	assert.Equal(t, 100, pid)
	assert.Equal(t, 5, logs)
	assert.Equal(t, 2, errors)
	assert.Equal(t, 2, faults)
}

func TestTrackerGetFinalSummary(t *testing.T) {
	tracker := NewTracker("com.example.app", "iPhone 17 Pro", "ABC123", "tail-1", "1.0", "100")

	// Uninitialized tracker
	assert.Nil(t, tracker.GetFinalSummary())

	// Initialize with some entries
	tracker.CheckEntry(&domain.LogEntry{
		PID: 100, Subsystem: "com.example.app", Level: domain.LogLevelInfo,
	})
	tracker.CheckEntry(&domain.LogEntry{
		PID: 100, Subsystem: "com.example.app", Level: domain.LogLevelError,
	})

	summary := tracker.GetFinalSummary()
	require.NotNil(t, summary)
	assert.Equal(t, "session_end", summary.Type)
	assert.Equal(t, 1, summary.Session)
	assert.Equal(t, 100, summary.PID)
	assert.Equal(t, 2, summary.Summary.TotalLogs)
	assert.Equal(t, 1, summary.Summary.Errors)
}

func TestTrackerDetectsPIDChangeAnySubsystem(t *testing.T) {
	tracker := NewTracker("com.example.app", "iPhone 17 Pro", "ABC123", "tail-1", "1.0", "100")

	// Initialize with our app
	tracker.CheckEntry(&domain.LogEntry{
		PID: 100, Subsystem: "com.example.app", Level: domain.LogLevelInfo,
	})

	// Log with different PID and different subsystem - should detect relaunch
	// because streamer already filters by app, so any log we receive is relevant
	change := tracker.CheckEntry(&domain.LogEntry{
		PID: 200, Subsystem: "com.apple.system", Level: domain.LogLevelInfo,
	})

	// Should trigger session change since PID changed
	require.NotNil(t, change)
	assert.Equal(t, "APP_RELAUNCHED", change.StartSession.Alert)
	assert.Equal(t, 2, change.StartSession.Session)
	assert.Equal(t, 200, change.StartSession.PID)
	assert.Equal(t, 100, change.StartSession.PreviousPID)
}

func TestTrackerDetectsPIDChangeEmptySubsystem(t *testing.T) {
	tracker := NewTracker("com.example.app", "iPhone 17 Pro", "ABC123", "tail-1", "1.0", "100")

	// Initialize with our app
	tracker.CheckEntry(&domain.LogEntry{
		PID: 100, Subsystem: "com.example.app", Level: domain.LogLevelInfo,
	})

	// Log with different PID and EMPTY subsystem - this was the bug case
	change := tracker.CheckEntry(&domain.LogEntry{
		PID: 200, Subsystem: "", Level: domain.LogLevelInfo,
	})

	// Should trigger session change since PID changed
	require.NotNil(t, change)
	assert.Equal(t, "APP_RELAUNCHED", change.StartSession.Alert)
	assert.Equal(t, 2, change.StartSession.Session)
}

func TestTrackerForceRollover(t *testing.T) {
	tracker := NewTracker("com.example.app", "iPhone 17 Pro", "ABC123", "tail-1", "1.0", "100")

	// initialize
	tracker.CheckEntry(&domain.LogEntry{
		PID: 100, Subsystem: "com.example.app", Level: domain.LogLevelInfo, ProcessImageUUID: "UUID-A",
	})
	tracker.CheckEntry(&domain.LogEntry{
		PID: 100, Subsystem: "com.example.app", Level: domain.LogLevelError, ProcessImageUUID: "UUID-A",
	})

	change := tracker.ForceRollover("IDLE_TIMEOUT")
	require.NotNil(t, change)
	require.NotNil(t, change.EndSession)
	require.NotNil(t, change.StartSession)
	assert.Equal(t, 1, change.EndSession.Session)
	assert.Equal(t, 2, change.StartSession.Session)
	assert.Equal(t, "IDLE_TIMEOUT", change.StartSession.Alert)
	assert.Equal(t, "tail-1", change.StartSession.TailID)

	// Session stats reset
	session, pid, logs, errors, faults := tracker.Stats()
	assert.Equal(t, 2, session)
	assert.Equal(t, 100, pid)
	assert.Equal(t, 0, logs)
	assert.Equal(t, 0, errors)
	assert.Equal(t, 0, faults)
}
