package cli

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vburojevic/xcw/internal/config"
	"github.com/vburojevic/xcw/internal/domain"
	"github.com/vburojevic/xcw/internal/output"
	"golang.org/x/sync/errgroup"
)

func TestWatchTriggerOutput_SuccessCapture(t *testing.T) {
	var stdout strings.Builder
	w := output.NewNDJSONWriter(&stdout)
	var mu sync.Mutex
	writeStdout := func(fn func(w *output.NDJSONWriter) error) error {
		mu.Lock()
		defer mu.Unlock()
		return fn(w)
	}

	globals := &Globals{
		Format: "ndjson",
		Stdout: &stdout,
		Stderr: &strings.Builder{},
		Config: config.Default(),
	}
	c := &WatchCmd{TriggerNoShell: false}

	entry := domain.LogEntry{
		TailID:  "tail-abc",
		Session: 2,
		Message: "boom",
		Level:   domain.LogLevelError,
	}

	group, ctx := errgroup.WithContext(context.Background())
	sem := make(chan struct{}, 1)
	c.runTrigger(ctx, group, globals, writeStdout, "error", "python3 -c 'print(\"ok\")'", entry, 5*time.Second, sem, "capture")
	require.NoError(t, group.Wait())

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.GreaterOrEqual(t, len(lines), 2)

	var trigger map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &trigger))
	require.Equal(t, "trigger", trigger["type"])
	require.Equal(t, "tail-abc", trigger["tail_id"])
	require.Equal(t, float64(2), trigger["session"])

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &result))
	require.Equal(t, "trigger_result", result["type"])
	require.Equal(t, "tail-abc", result["tail_id"])
	require.Equal(t, float64(2), result["session"])
	require.Equal(t, trigger["trigger_id"], result["trigger_id"])
	require.Equal(t, float64(0), result["exit_code"])
	if timedOut, ok := result["timed_out"]; ok {
		require.Equal(t, false, timedOut)
	}
	require.Equal(t, "ok", result["output"])
}

func TestWatchTriggerOutput_TruncatesCapture(t *testing.T) {
	var stdout strings.Builder
	w := output.NewNDJSONWriter(&stdout)
	var mu sync.Mutex
	writeStdout := func(fn func(w *output.NDJSONWriter) error) error {
		mu.Lock()
		defer mu.Unlock()
		return fn(w)
	}

	globals := &Globals{
		Format: "ndjson",
		Stdout: &stdout,
		Stderr: &strings.Builder{},
		Config: config.Default(),
	}
	c := &WatchCmd{TriggerNoShell: false}

	entry := domain.LogEntry{TailID: "tail-abc", Session: 1, Message: "msg"}

	group, ctx := errgroup.WithContext(context.Background())
	sem := make(chan struct{}, 1)
	c.runTrigger(ctx, group, globals, writeStdout, "pattern:big", "python3 -c 'print(\"a\"*20000)'", entry, 5*time.Second, sem, "capture")
	require.NoError(t, group.Wait())

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.GreaterOrEqual(t, len(lines), 2)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &result))
	require.Equal(t, "trigger_result", result["type"])
	out, _ := result["output"].(string)
	require.Len(t, out, 16*1024)
}

func TestWatchTriggerOutput_FailureEmitsTriggerError(t *testing.T) {
	var stdout strings.Builder
	w := output.NewNDJSONWriter(&stdout)
	var mu sync.Mutex
	writeStdout := func(fn func(w *output.NDJSONWriter) error) error {
		mu.Lock()
		defer mu.Unlock()
		return fn(w)
	}

	globals := &Globals{
		Format: "ndjson",
		Stdout: &stdout,
		Stderr: &strings.Builder{},
		Config: config.Default(),
	}
	c := &WatchCmd{TriggerNoShell: false}

	entry := domain.LogEntry{TailID: "tail-abc", Session: 1, Message: "msg"}

	group, ctx := errgroup.WithContext(context.Background())
	sem := make(chan struct{}, 1)
	c.runTrigger(ctx, group, globals, writeStdout, "error", "exit 3", entry, 5*time.Second, sem, "discard")
	require.NoError(t, group.Wait())

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.GreaterOrEqual(t, len(lines), 3)

	var trigger map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &trigger))
	require.Equal(t, "trigger", trigger["type"])

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &result))
	require.Equal(t, "trigger_result", result["type"])
	require.Equal(t, trigger["trigger_id"], result["trigger_id"])
	require.Equal(t, float64(3), result["exit_code"])
	require.NotEmpty(t, result["error"])

	var terr map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[2]), &terr))
	require.Equal(t, "trigger_error", terr["type"])
	require.Equal(t, trigger["trigger_id"], terr["trigger_id"])
	require.NotEmpty(t, terr["error"])
}
