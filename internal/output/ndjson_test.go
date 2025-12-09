package output

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vburojevic/xcw/internal/domain"
)

func TestNDJSONWriter_Write(t *testing.T) {
	t.Run("writes log entry with type field and schemaVersion", func(t *testing.T) {
		var buf bytes.Buffer
		w := NewNDJSONWriter(&buf)

		entry := &domain.LogEntry{
			Timestamp: time.Date(2024, 1, 15, 10, 30, 45, 123000000, time.UTC),
			Level:     domain.LogLevelError,
			Process:   "TestApp",
			PID:       1234,
			Subsystem: "com.test.app",
			Category:  "network",
			Message:   "Connection failed",
		}

		err := w.Write(entry)
		require.NoError(t, err)

		// Parse output
		var out OutputEntry
		err = json.Unmarshal(buf.Bytes(), &out)
		require.NoError(t, err)

		assert.Equal(t, "log", out.Type)
		assert.Equal(t, SchemaVersion, out.SchemaVersion)
		assert.Equal(t, "Error", out.Level)
		assert.Equal(t, "TestApp", out.Process)
		assert.Equal(t, 1234, out.PID)
		assert.Equal(t, "com.test.app", out.Subsystem)
		assert.Equal(t, "network", out.Category)
		assert.Equal(t, "Connection failed", out.Message)
	})

	t.Run("omits empty subsystem and category", func(t *testing.T) {
		var buf bytes.Buffer
		w := NewNDJSONWriter(&buf)

		entry := &domain.LogEntry{
			Timestamp: time.Now(),
			Level:     domain.LogLevelInfo,
			Process:   "TestApp",
			PID:       1234,
			Message:   "Hello",
		}

		err := w.Write(entry)
		require.NoError(t, err)

		// Check raw JSON doesn't contain empty fields
		output := buf.String()
		assert.NotContains(t, output, `"subsystem":""`)
		assert.NotContains(t, output, `"category":""`)
	})
}

func TestNDJSONWriter_WriteInfo(t *testing.T) {
	var buf bytes.Buffer
	w := NewNDJSONWriter(&buf)

	err := w.WriteInfo("Test message", "iPhone 15", "ABC123", "5m", "tail")
	require.NoError(t, err)

	var out InfoOutput
	err = json.Unmarshal(buf.Bytes(), &out)
	require.NoError(t, err)

	assert.Equal(t, "info", out.Type)
	assert.Equal(t, SchemaVersion, out.SchemaVersion)
	assert.Equal(t, "Test message", out.Message)
	assert.Equal(t, "iPhone 15", out.Simulator)
	assert.Equal(t, "ABC123", out.UDID)
	assert.Equal(t, "5m", out.Since)
	assert.Equal(t, "tail", out.Mode)
}

func TestNDJSONWriter_WriteWarning(t *testing.T) {
	var buf bytes.Buffer
	w := NewNDJSONWriter(&buf)

	err := w.WriteWarning("Something went wrong")
	require.NoError(t, err)

	var out WarningOutput
	err = json.Unmarshal(buf.Bytes(), &out)
	require.NoError(t, err)

	assert.Equal(t, "warning", out.Type)
	assert.Equal(t, SchemaVersion, out.SchemaVersion)
	assert.Equal(t, "Something went wrong", out.Message)
}

func TestNDJSONWriter_WriteTmux(t *testing.T) {
	var buf bytes.Buffer
	w := NewNDJSONWriter(&buf)

	err := w.WriteTmux("xcw-iphone-15", "tmux attach -t xcw-iphone-15")
	require.NoError(t, err)

	var out TmuxOutput
	err = json.Unmarshal(buf.Bytes(), &out)
	require.NoError(t, err)

	assert.Equal(t, "tmux", out.Type)
	assert.Equal(t, SchemaVersion, out.SchemaVersion)
	assert.Equal(t, "xcw-iphone-15", out.Session)
	assert.Equal(t, "tmux attach -t xcw-iphone-15", out.Attach)
}

func TestNDJSONWriter_WriteTrigger(t *testing.T) {
	var buf bytes.Buffer
	w := NewNDJSONWriter(&buf)

	err := w.WriteTrigger("error", "notify.sh", "Connection timeout")
	require.NoError(t, err)

	var out TriggerOutput
	err = json.Unmarshal(buf.Bytes(), &out)
	require.NoError(t, err)

	assert.Equal(t, "trigger", out.Type)
	assert.Equal(t, SchemaVersion, out.SchemaVersion)
	assert.Equal(t, "error", out.Trigger)
	assert.Equal(t, "notify.sh", out.Command)
	assert.Equal(t, "Connection timeout", out.Message)
}

func TestNDJSONWriter_WriteTriggerError(t *testing.T) {
	var buf bytes.Buffer
	w := NewNDJSONWriter(&buf)

	err := w.WriteTriggerError("notify.sh", "exit status 1")
	require.NoError(t, err)

	var out TriggerErrorOutput
	err = json.Unmarshal(buf.Bytes(), &out)
	require.NoError(t, err)

	assert.Equal(t, "trigger_error", out.Type)
	assert.Equal(t, SchemaVersion, out.SchemaVersion)
	assert.Equal(t, "notify.sh", out.Command)
	assert.Equal(t, "exit status 1", out.Error)
}

func TestNDJSONWriter_WriteError(t *testing.T) {
	var buf bytes.Buffer
	w := NewNDJSONWriter(&buf)

	err := w.WriteError("DEVICE_NOT_FOUND", "No simulator found")
	require.NoError(t, err)

	var out domain.ErrorOutput
	err = json.Unmarshal(buf.Bytes(), &out)
	require.NoError(t, err)

	assert.Equal(t, "error", out.Type)
	assert.Equal(t, SchemaVersion, out.SchemaVersion)
	assert.Equal(t, "DEVICE_NOT_FOUND", out.Code)
	assert.Equal(t, "No simulator found", out.Message)
}

func TestNDJSONWriter_WriteHeartbeat(t *testing.T) {
	var buf bytes.Buffer
	w := NewNDJSONWriter(&buf)

	h := &Heartbeat{
		Type:          "heartbeat",
		SchemaVersion: SchemaVersion,
		Timestamp:     "2024-01-15T10:30:45Z",
		UptimeSeconds: 300,
		LogsSinceLast: 42,
	}

	err := w.WriteHeartbeat(h)
	require.NoError(t, err)

	var out Heartbeat
	err = json.Unmarshal(buf.Bytes(), &out)
	require.NoError(t, err)

	assert.Equal(t, "heartbeat", out.Type)
	assert.Equal(t, SchemaVersion, out.SchemaVersion)
	assert.Equal(t, int64(300), out.UptimeSeconds)
	assert.Equal(t, 42, out.LogsSinceLast)
}

func TestNDJSONWriter_EscapesSpecialCharacters(t *testing.T) {
	var buf bytes.Buffer
	w := NewNDJSONWriter(&buf)

	entry := &domain.LogEntry{
		Timestamp: time.Now(),
		Level:     domain.LogLevelError,
		Process:   "TestApp",
		PID:       1234,
		Message:   "Error: \"quoted\" and\nnewline and\ttab",
	}

	err := w.Write(entry)
	require.NoError(t, err)

	// Should be valid JSON
	var out OutputEntry
	err = json.Unmarshal(buf.Bytes(), &out)
	require.NoError(t, err)

	// Message should be properly preserved
	assert.Contains(t, out.Message, "\"quoted\"")
	assert.Contains(t, out.Message, "\n")
	assert.Contains(t, out.Message, "\t")
}
