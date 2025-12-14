package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vburojevic/xcw/internal/config"
	"github.com/vburojevic/xcw/internal/domain"
)

// testGlobals creates a Globals struct with captured stdout/stderr
func testGlobals(format string) (*Globals, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return &Globals{
		Format:  format,
		Level:   "default",
		Quiet:   false,
		Verbose: false,
		Stdout:  stdout,
		Stderr:  stderr,
		Config:  config.Default(),
	}, stdout, stderr
}

// --- Config Command Tests ---

func TestConfigShowCmd_Run(t *testing.T) {
	t.Run("outputs config in text format", func(t *testing.T) {
		globals, stdout, _ := testGlobals("text")
		cmd := &ConfigShowCmd{}

		err := cmd.Run(globals)
		require.NoError(t, err)

		output := stdout.String()
		assert.Contains(t, output, "Current Configuration:")
		assert.Contains(t, output, "format:")
		assert.Contains(t, output, "level:")
		assert.Contains(t, output, "Defaults:")
	})

	t.Run("outputs config in NDJSON format", func(t *testing.T) {
		globals, stdout, _ := testGlobals("ndjson")
		cmd := &ConfigShowCmd{}

		err := cmd.Run(globals)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(stdout.Bytes(), &result)
		require.NoError(t, err)

		assert.Equal(t, "config", result["type"])
		assert.Contains(t, result, "format")
		assert.Contains(t, result, "level")
		assert.Contains(t, result, "defaults")
	})
}

func TestConfigPathCmd_Run(t *testing.T) {
	t.Run("outputs path info in text format when no config", func(t *testing.T) {
		globals, stdout, _ := testGlobals("text")
		cmd := &ConfigPathCmd{}

		err := cmd.Run(globals)
		require.NoError(t, err)

		output := stdout.String()
		// Either shows the path or says no config found
		assert.True(t, strings.Contains(output, "Config file:") || strings.Contains(output, "No configuration file found"))
	})

	t.Run("outputs path in NDJSON format", func(t *testing.T) {
		globals, stdout, _ := testGlobals("ndjson")
		cmd := &ConfigPathCmd{}

		err := cmd.Run(globals)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(stdout.Bytes(), &result)
		require.NoError(t, err)

		assert.Equal(t, "config_path", result["type"])
		assert.Contains(t, result, "path")
	})
}

func TestConfigGenerateCmd_Run(t *testing.T) {
	t.Run("outputs sample config YAML", func(t *testing.T) {
		globals, stdout, _ := testGlobals("text")
		cmd := &ConfigGenerateCmd{}

		err := cmd.Run(globals)
		require.NoError(t, err)

		output := stdout.String()
		assert.Contains(t, output, "# xcw configuration file")
		assert.Contains(t, output, "format: ndjson")
		assert.Contains(t, output, "level: default")
		assert.Contains(t, output, "defaults:")
		assert.Contains(t, output, "simulator: booted")
		assert.Contains(t, output, "buffer_size: 100")
	})
}

// --- Schema Command Tests ---

func TestSchemaCmd_Run(t *testing.T) {
	t.Run("outputs all schemas by default with schemaVersion", func(t *testing.T) {
		globals, stdout, _ := testGlobals("ndjson")
		cmd := &SchemaCmd{}

		err := cmd.Run(globals)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(stdout.Bytes(), &result)
		require.NoError(t, err)

		assert.Equal(t, "http://json-schema.org/draft-07/schema#", result["$schema"])
		assert.Equal(t, "XcodeConsoleWatcher Output Schemas", result["title"])
		assert.NotNil(t, result["schemaVersion"], "top-level schemaVersion should exist")

		defs := result["definitions"].(map[string]interface{})
		assert.Contains(t, defs, "log")
		assert.Contains(t, defs, "summary")
		assert.Contains(t, defs, "heartbeat")
		assert.Contains(t, defs, "error")
		assert.Contains(t, defs, "doctor")
		assert.Contains(t, defs, "app")

		// Verify schemaVersion property exists in each definition
		for name, def := range defs {
			defMap := def.(map[string]interface{})
			props := defMap["properties"].(map[string]interface{})
			assert.Contains(t, props, "schemaVersion", "definition %s should have schemaVersion property", name)
		}
	})

	t.Run("filters schemas by type", func(t *testing.T) {
		globals, stdout, _ := testGlobals("ndjson")
		cmd := &SchemaCmd{Type: []string{"log", "error"}}

		err := cmd.Run(globals)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(stdout.Bytes(), &result)
		require.NoError(t, err)

		defs := result["definitions"].(map[string]interface{})
		assert.Len(t, defs, 2)
		assert.Contains(t, defs, "log")
		assert.Contains(t, defs, "error")
		assert.NotContains(t, defs, "summary")
	})
}

func TestLogSchema(t *testing.T) {
	schema := logSchema()

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, "Log Entry", schema["title"])

	props := schema["properties"].(map[string]interface{})
	assert.Contains(t, props, "schemaVersion")
	assert.Contains(t, props, "timestamp")
	assert.Contains(t, props, "level")
	assert.Contains(t, props, "process")
	assert.Contains(t, props, "pid")
	assert.Contains(t, props, "message")

	// Verify schemaVersion is required
	required := schema["required"].([]string)
	assert.Contains(t, required, "schemaVersion")
}

func TestDoctorSchema(t *testing.T) {
	schema := doctorSchema()

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, "Doctor Report", schema["title"])

	props := schema["properties"].(map[string]interface{})
	assert.Contains(t, props, "schemaVersion")
	assert.Contains(t, props, "checks")
	assert.Contains(t, props, "all_passed")
	assert.Contains(t, props, "error_count")

	// Verify schemaVersion is required
	required := schema["required"].([]string)
	assert.Contains(t, required, "schemaVersion")
}

func TestAppSchema(t *testing.T) {
	schema := appSchema()

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, "Installed App", schema["title"])

	props := schema["properties"].(map[string]interface{})
	assert.Contains(t, props, "schemaVersion")
	assert.Contains(t, props, "bundle_id")
	assert.Contains(t, props, "name")
	assert.Contains(t, props, "version")
	assert.Contains(t, props, "app_type")

	// Verify schemaVersion is required
	required := schema["required"].([]string)
	assert.Contains(t, required, "schemaVersion")
}

func TestPickSchema(t *testing.T) {
	schema := pickSchema()

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, "Pick Result", schema["title"])

	props := schema["properties"].(map[string]interface{})
	assert.Contains(t, props, "schemaVersion")
	assert.Contains(t, props, "picked")
	assert.Contains(t, props, "udid")
	assert.Contains(t, props, "name")
	assert.Contains(t, props, "bundle_id")

	// Verify schemaVersion is required
	required := schema["required"].([]string)
	assert.Contains(t, required, "schemaVersion")
	assert.Contains(t, required, "picked")
	assert.Contains(t, required, "name")
}

// --- Pick Command Tests ---

func TestPickCmd_RequiresInteractive(t *testing.T) {
	// This test verifies the pick command exists and has proper struct
	cmd := PickCmd{
		Type:     "simulator",
		UserOnly: false,
	}

	assert.Equal(t, "simulator", cmd.Type)
	assert.False(t, cmd.UserOnly)
}

func TestPickItem(t *testing.T) {
	item := pickItem{
		id:          "ABC-123-DEF",
		title:       "iPhone 15 Pro",
		description: "iOS 17.0",
	}

	assert.Equal(t, "iPhone 15 Pro", item.Title())
	assert.Equal(t, "iOS 17.0", item.Description())
	assert.Contains(t, item.FilterValue(), "iPhone 15 Pro")
	assert.Contains(t, item.FilterValue(), "ABC-123-DEF")
}

// --- Session Tests ---

func TestGenerateSessionPath(t *testing.T) {
	t.Run("generates path in specified directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		path, err := GenerateSessionPath(tmpDir, "com.example.app")
		require.NoError(t, err)

		assert.True(t, strings.HasPrefix(path, tmpDir))
		// Prefix is sanitized (dots become underscores)
		assert.Contains(t, path, "com_example_app")
		assert.True(t, strings.HasSuffix(path, ".ndjson"))
	})

	t.Run("creates directory if missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "sessions")
		path, err := GenerateSessionPath(subDir, "test")
		require.NoError(t, err)

		_, err = os.Stat(subDir)
		assert.NoError(t, err, "directory should be created")
		assert.True(t, strings.HasPrefix(path, subDir))
	})

	t.Run("sanitizes prefix", func(t *testing.T) {
		tmpDir := t.TempDir()
		path, err := GenerateSessionPath(tmpDir, "com.example/app:test")
		require.NoError(t, err)

		filename := filepath.Base(path)
		assert.NotContains(t, filename, "/")
		assert.NotContains(t, filename, ":")
	})
}

func TestListSessions(t *testing.T) {
	t.Run("lists sessions sorted by time", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create some session files with different timestamps
		files := []string{
			"20251209-100000-app.ndjson",
			"20251209-120000-app.ndjson",
			"20251209-110000-app.ndjson",
		}
		for _, f := range files {
			require.NoError(t, os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0644))
		}

		sessions, err := ListSessions(tmpDir)
		require.NoError(t, err)
		require.Len(t, sessions, 3)

		// Should be sorted newest first
		assert.Equal(t, "20251209-120000-app.ndjson", sessions[0].Name)
		assert.Equal(t, "20251209-110000-app.ndjson", sessions[1].Name)
		assert.Equal(t, "20251209-100000-app.ndjson", sessions[2].Name)
	})

	t.Run("returns empty for nonexistent directory", func(t *testing.T) {
		sessions, err := ListSessions("/nonexistent/path")
		require.NoError(t, err)
		assert.Empty(t, sessions)
	})

	t.Run("ignores non-ndjson files", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "20251209-100000-app.ndjson"), []byte("test"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "other.txt"), []byte("test"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte("test"), 0644))

		sessions, err := ListSessions(tmpDir)
		require.NoError(t, err)
		assert.Len(t, sessions, 1)
	})
}

func TestLatestSession(t *testing.T) {
	t.Run("returns most recent session", func(t *testing.T) {
		tmpDir := t.TempDir()

		files := []string{
			"20251209-100000-app.ndjson",
			"20251209-120000-app.ndjson",
			"20251209-110000-app.ndjson",
		}
		for _, f := range files {
			require.NoError(t, os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0644))
		}

		session, err := LatestSession(tmpDir)
		require.NoError(t, err)
		require.NotNil(t, session)
		assert.Equal(t, "20251209-120000-app.ndjson", session.Name)
	})

	t.Run("returns nil for empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		session, err := LatestSession(tmpDir)
		require.NoError(t, err)
		assert.Nil(t, session)
	})
}

func TestCleanOldSessions(t *testing.T) {
	t.Run("deletes oldest sessions keeping specified count", func(t *testing.T) {
		tmpDir := t.TempDir()

		files := []string{
			"20251209-100000-app.ndjson",
			"20251209-110000-app.ndjson",
			"20251209-120000-app.ndjson",
			"20251209-130000-app.ndjson",
			"20251209-140000-app.ndjson",
		}
		for _, f := range files {
			require.NoError(t, os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0644))
		}

		deleted, err := CleanOldSessions(tmpDir, 2)
		require.NoError(t, err)
		assert.Len(t, deleted, 3)

		// Verify only 2 remain
		remaining, err := ListSessions(tmpDir)
		require.NoError(t, err)
		assert.Len(t, remaining, 2)

		// Should keep the newest
		assert.Equal(t, "20251209-140000-app.ndjson", remaining[0].Name)
		assert.Equal(t, "20251209-130000-app.ndjson", remaining[1].Name)
	})

	t.Run("does nothing if fewer sessions than keep count", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "20251209-100000-app.ndjson"), []byte("test"), 0644))

		deleted, err := CleanOldSessions(tmpDir, 5)
		require.NoError(t, err)
		assert.Empty(t, deleted)
	})
}

func TestSessionSchema(t *testing.T) {
	schema := sessionSchema()

	assert.Equal(t, "object", schema["type"])
	assert.Equal(t, "Session File", schema["title"])

	props := schema["properties"].(map[string]interface{})
	assert.Contains(t, props, "schemaVersion")
	assert.Contains(t, props, "path")
	assert.Contains(t, props, "name")
	assert.Contains(t, props, "timestamp")
	assert.Contains(t, props, "size")

	required := schema["required"].([]string)
	assert.Contains(t, required, "schemaVersion")
	assert.Contains(t, required, "path")
}

// --- Analyze Command Tests ---

func TestAnalyzeCmd_Run(t *testing.T) {
	// Create a temporary NDJSON log file
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.ndjson")

	entries := []domain.LogEntry{
		{Timestamp: time.Now().Add(-5 * time.Minute), Level: domain.LogLevelInfo, Process: "TestApp", PID: 123, Message: "Info message 1"},
		{Timestamp: time.Now().Add(-4 * time.Minute), Level: domain.LogLevelInfo, Process: "TestApp", PID: 123, Message: "Info message 2"},
		{Timestamp: time.Now().Add(-3 * time.Minute), Level: domain.LogLevelError, Process: "TestApp", PID: 123, Message: "Error: something failed"},
		{Timestamp: time.Now().Add(-2 * time.Minute), Level: domain.LogLevelError, Process: "TestApp", PID: 123, Message: "Error: something failed again"},
		{Timestamp: time.Now().Add(-1 * time.Minute), Level: domain.LogLevelFault, Process: "TestApp", PID: 123, Message: "Fault: critical failure"},
	}

	// Write entries to file
	f, err := os.Create(logFile)
	require.NoError(t, err)
	encoder := json.NewEncoder(f)
	for _, entry := range entries {
		require.NoError(t, encoder.Encode(entry))
	}
	require.NoError(t, f.Close())

	t.Run("analyzes log file in text format", func(t *testing.T) {
		globals, stdout, _ := testGlobals("text")
		cmd := &AnalyzeCmd{File: logFile}

		err := cmd.Run(globals)
		require.NoError(t, err)

		output := stdout.String()
		assert.Contains(t, output, "Analysis of")
		assert.Contains(t, output, "Total entries:")
		assert.Contains(t, output, "Error:")
		assert.Contains(t, output, "Fault:")
	})

	t.Run("analyzes log file in NDJSON format", func(t *testing.T) {
		globals, stdout, _ := testGlobals("ndjson")
		cmd := &AnalyzeCmd{File: logFile}

		err := cmd.Run(globals)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(stdout.Bytes(), &result)
		require.NoError(t, err)

		assert.Equal(t, "analysis", result["type"])
		assert.Contains(t, result, "summary")
		// patterns may be omitted if empty (omitempty)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		globals, _, _ := testGlobals("text")
		cmd := &AnalyzeCmd{File: "/nonexistent/file.ndjson"}

		err := cmd.Run(globals)
		assert.Error(t, err)
	})

	t.Run("returns error for empty file", func(t *testing.T) {
		emptyFile := filepath.Join(tmpDir, "empty.ndjson")
		require.NoError(t, os.WriteFile(emptyFile, []byte{}, 0644))

		globals, _, _ := testGlobals("text")
		cmd := &AnalyzeCmd{File: emptyFile}

		err := cmd.Run(globals)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no valid log entries")
	})

	t.Run("with pattern persistence", func(t *testing.T) {
		patternFile := filepath.Join(tmpDir, "patterns.json")
		globals, stdout, _ := testGlobals("ndjson")
		cmd := &AnalyzeCmd{
			File:            logFile,
			PersistPatterns: true,
			PatternFile:     patternFile,
		}

		err := cmd.Run(globals)
		require.NoError(t, err)

		// Pattern file should be created
		_, err = os.Stat(patternFile)
		assert.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(stdout.Bytes(), &result)
		require.NoError(t, err)

		assert.Contains(t, result, "new_pattern_count")
		assert.Contains(t, result, "known_pattern_count")
	})
}

// --- Replay Command Tests ---

func TestReplayCmd_Run(t *testing.T) {
	// Create a temporary NDJSON log file
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.ndjson")

	entries := []domain.LogEntry{
		{Timestamp: time.Now().Add(-3 * time.Second), Level: domain.LogLevelInfo, Process: "TestApp", PID: 123, Message: "Message 1"},
		{Timestamp: time.Now().Add(-2 * time.Second), Level: domain.LogLevelInfo, Process: "TestApp", PID: 123, Message: "Message 2"},
		{Timestamp: time.Now().Add(-1 * time.Second), Level: domain.LogLevelInfo, Process: "TestApp", PID: 123, Message: "Message 3"},
	}

	// Write entries to file
	f, err := os.Create(logFile)
	require.NoError(t, err)
	encoder := json.NewEncoder(f)
	for _, entry := range entries {
		require.NoError(t, encoder.Encode(entry))
	}
	require.NoError(t, f.Close())

	t.Run("replays log file in text format", func(t *testing.T) {
		globals, stdout, _ := testGlobals("text")
		globals.Quiet = true
		cmd := &ReplayCmd{File: logFile}

		err := cmd.Run(globals)
		require.NoError(t, err)

		output := stdout.String()
		assert.Contains(t, output, "Message 1")
		assert.Contains(t, output, "Message 2")
		assert.Contains(t, output, "Message 3")
	})

	t.Run("replays log file in NDJSON format", func(t *testing.T) {
		globals, stdout, _ := testGlobals("ndjson")
		globals.Quiet = true
		cmd := &ReplayCmd{File: logFile}

		err := cmd.Run(globals)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		assert.Len(t, lines, 3)

		for _, line := range lines {
			var entry map[string]interface{}
			err := json.Unmarshal([]byte(line), &entry)
			require.NoError(t, err)
			assert.Contains(t, entry, "timestamp")
			assert.Contains(t, entry, "level")
			assert.Contains(t, entry, "message")
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		globals, _, _ := testGlobals("text")
		cmd := &ReplayCmd{File: "/nonexistent/file.ndjson"}

		err := cmd.Run(globals)
		assert.Error(t, err)
	})

	t.Run("shows replay info when not quiet", func(t *testing.T) {
		globals, _, stderr := testGlobals("text")
		globals.Quiet = false
		cmd := &ReplayCmd{File: logFile}

		err := cmd.Run(globals)
		require.NoError(t, err)

		output := stderr.String()
		assert.Contains(t, output, "Replaying logs from")
	})

	t.Run("shows entry count at end", func(t *testing.T) {
		globals, _, stderr := testGlobals("text")
		globals.Quiet = false
		cmd := &ReplayCmd{File: logFile}

		err := cmd.Run(globals)
		require.NoError(t, err)

		output := stderr.String()
		assert.Contains(t, output, "Replayed 3 entries")
	})
}

// --- Doctor Command Tests ---

func TestDoctorCmd_checkResult(t *testing.T) {
	t.Run("check result struct", func(t *testing.T) {
		result := checkResult{
			Name:    "Test Check",
			Status:  "ok",
			Message: "Check passed",
			Details: "Additional info",
		}

		data, err := json.Marshal(result)
		require.NoError(t, err)

		var decoded checkResult
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, "Test Check", decoded.Name)
		assert.Equal(t, "ok", decoded.Status)
		assert.Equal(t, "Check passed", decoded.Message)
		assert.Equal(t, "Additional info", decoded.Details)
	})
}

func TestDoctorCmd_doctorReport(t *testing.T) {
	t.Run("doctor report struct", func(t *testing.T) {
		report := doctorReport{
			Type:      "doctor",
			Timestamp: time.Now().Format(time.RFC3339),
			Checks: []checkResult{
				{Name: "check1", Status: "ok", Message: "passed"},
				{Name: "check2", Status: "warning", Message: "needs attention"},
				{Name: "check3", Status: "error", Message: "failed"},
			},
			AllPassed:  false,
			ErrorCount: 1,
			WarnCount:  1,
		}

		data, err := json.Marshal(report)
		require.NoError(t, err)

		var decoded doctorReport
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, "doctor", decoded.Type)
		assert.Len(t, decoded.Checks, 3)
		assert.False(t, decoded.AllPassed)
		assert.Equal(t, 1, decoded.ErrorCount)
		assert.Equal(t, 1, decoded.WarnCount)
	})
}

func TestDoctorCmd_checkWritePermission(t *testing.T) {
	cmd := &DoctorCmd{}

	t.Run("returns true for writable directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		assert.True(t, cmd.checkWritePermission(tmpDir))
	})

	t.Run("returns false for non-writable directory", func(t *testing.T) {
		// Try a directory that doesn't exist
		assert.False(t, cmd.checkWritePermission("/nonexistent/path"))
	})
}

// --- Apps Command Tests ---

func TestAppsCmd_appInfo(t *testing.T) {
	t.Run("app info struct marshals correctly", func(t *testing.T) {
		app := appInfo{
			BundleID:    "com.example.app",
			Name:        "Example App",
			Version:     "1.0.0",
			BuildNumber: "123",
			Path:        "/path/to/app",
			DataPath:    "/path/to/data",
			Type:        "user",
		}

		data, err := json.Marshal(app)
		require.NoError(t, err)

		var decoded appInfo
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, "com.example.app", decoded.BundleID)
		assert.Equal(t, "Example App", decoded.Name)
		assert.Equal(t, "1.0.0", decoded.Version)
		assert.Equal(t, "user", decoded.Type)
	})
}

// --- parseTimeOrDuration Tests ---

func TestParseTimeOrDuration(t *testing.T) {
	t.Run("parses RFC3339 time", func(t *testing.T) {
		input := "2024-01-15T10:30:00Z"
		result, err := parseTimeOrDuration(input)
		require.NoError(t, err)
		assert.Equal(t, 2024, result.Year())
		assert.Equal(t, time.January, result.Month())
		assert.Equal(t, 15, result.Day())
	})

	t.Run("parses duration", func(t *testing.T) {
		input := "5m"
		before := time.Now().Add(-6 * time.Minute)
		result, err := parseTimeOrDuration(input)
		require.NoError(t, err)
		assert.True(t, result.After(before))
	})

	t.Run("parses hours", func(t *testing.T) {
		input := "1h"
		before := time.Now().Add(-2 * time.Hour)
		result, err := parseTimeOrDuration(input)
		require.NoError(t, err)
		assert.True(t, result.After(before))
	})

	t.Run("returns error for invalid input", func(t *testing.T) {
		_, err := parseTimeOrDuration("invalid")
		assert.Error(t, err)
	})
}

// --- Version Command Tests ---

func TestVersionCmd_Run(t *testing.T) {
	t.Run("outputs version in text format", func(t *testing.T) {
		globals, stdout, _ := testGlobals("text")
		cmd := &VersionCmd{}

		err := cmd.Run(globals)
		require.NoError(t, err)

		output := stdout.String()
		assert.Contains(t, output, "xcw version")
	})

	t.Run("outputs version in NDJSON format", func(t *testing.T) {
		globals, stdout, _ := testGlobals("ndjson")
		cmd := &VersionCmd{}

		err := cmd.Run(globals)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(stdout.Bytes(), &result)
		require.NoError(t, err)

		assert.Equal(t, "version", result["type"])
		assert.Contains(t, result, "version")
		assert.Contains(t, result, "commit")
	})
}
