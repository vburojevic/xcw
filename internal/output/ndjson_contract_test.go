package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vburojevic/xcw/internal/domain"
)

func decodeAll(t *testing.T, buf *bytes.Buffer) []map[string]interface{} {
	t.Helper()

	dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	var out []map[string]interface{}
	for {
		var m map[string]interface{}
		err := dec.Decode(&m)
		if err == nil {
			out = append(out, m)
			continue
		}
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
	}
	return out
}

func getByType(t *testing.T, items []map[string]interface{}, typ string) map[string]interface{} {
	t.Helper()
	for _, m := range items {
		if m["type"] == typ {
			return m
		}
	}
	require.FailNowf(t, "missing NDJSON type", "type=%s", typ)
	return nil
}

func TestNDJSONWriterContract_AllTypesHaveSchemaVersion(t *testing.T) {
	now := time.Date(2025, 12, 11, 10, 0, 0, 0, time.UTC)

	buf := &bytes.Buffer{}
	w := NewNDJSONWriter(buf)

	require.NoError(t, w.Write(&domain.LogEntry{
		Timestamp: now,
		Level:     domain.LogLevelInfo,
		Process:   "MyProcess",
		PID:       123,
		Message:   "hello",
		TailID:    "tail-1",
		Session:   2,
	}))

	require.NoError(t, w.WriteSessionStart(domain.NewSessionStartWithMeta(1, 123, 0, "com.example", "Sim", "UDID", "tail-1", "1.0", "1", "uuid-1", "")))
	require.NoError(t, w.WriteSessionEnd(domain.NewSessionEndWithMeta(1, 123, domain.SessionSummary{TotalLogs: 5}, "tail-1")))

	summary := domain.NewLogSummary()
	summary.TotalCount = 1
	require.NoError(t, w.WriteSummary(summary))

	require.NoError(t, w.WriteError("E_CODE", "something went wrong"))
	require.NoError(t, w.WriteInfo("info", "Sim", "UDID", "5m", "tail"))
	require.NoError(t, w.WriteWarning("warn"))
	require.NoError(t, w.WriteMetadata("0.0.0", "deadbeef", "2025-12-11"))
	require.NoError(t, w.WriteCutoff("max_duration", "tail-1", 2, 42))
	require.NoError(t, w.WriteRotation("/tmp/out.ndjson", "tail-1", 2))
	require.NoError(t, w.WriteReconnect("reconnecting", "tail-1", "warn"))
	require.NoError(t, w.WriteSessionDebug(&SessionDebugOutput{
		Type:        "session_debug",
		TailID:      "tail-1",
		Session:     2,
		PrevSession: 1,
		PID:         234,
		PrevPID:     123,
		Reason:      "pid_change",
	}))
	require.NoError(t, w.WriteTmux("xcw", "tmux attach -t xcw"))
	require.NoError(t, w.WriteTrigger("foo", "echo hi", "matched"))
	require.NoError(t, w.WriteTriggerError("echo hi", "exit status 1"))
	require.NoError(t, w.WriteReady(now.Format(time.RFC3339Nano), "Sim", "UDID", "com.example", "tail-1", 2))
	require.NoError(t, w.WriteClearBuffer("session_end", "tail-1", 2))
	require.NoError(t, w.WriteAgentHints("tail-1", 2, []string{"h1"}))

	require.NoError(t, w.WriteHeartbeat(&Heartbeat{
		Timestamp:         now.Format(time.RFC3339Nano),
		UptimeSeconds:     5,
		LogsSinceLast:     2,
		TailID:            "tail-1",
		LatestSession:     2,
		LastSeenTimestamp: now.Format(time.RFC3339Nano),
	}))
	require.NoError(t, w.WriteStats(&StreamStats{
		Timestamp:         now.Format(time.RFC3339Nano),
		TailID:            "tail-1",
		Session:           2,
		Reconnects:        1,
		ParseDrops:        0,
		ChannelDrops:      0,
		Buffered:          10,
		LastSeenTimestamp: now.Format(time.RFC3339Nano),
	}))

	require.NoError(t, w.WriteRaw(NewSummaryOutput(summary, nil)))

	items := decodeAll(t, buf)
	require.GreaterOrEqual(t, len(items), 1)

	for _, it := range items {
		require.Contains(t, it, "type")
		require.Contains(t, it, "schemaVersion")
		require.EqualValues(t, SchemaVersion, it["schemaVersion"])
	}

	ready := getByType(t, items, "ready")
	require.EqualValues(t, 1, ready["contract_version"])

	meta := getByType(t, items, "metadata")
	require.EqualValues(t, 1, meta["contract_version"])

	hints := getByType(t, items, "agent_hints")
	require.EqualValues(t, 1, hints["contract_version"])

	hb := getByType(t, items, "heartbeat")
	require.EqualValues(t, 1, hb["contract_version"])

	stats := getByType(t, items, "stats")
	require.Contains(t, stats, "timestamp")

	analysis := getByType(t, items, "analysis")
	require.Contains(t, analysis, "timestamp")
}
