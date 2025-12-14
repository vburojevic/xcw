package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	require.NotNil(t, cfg)
	assert.Equal(t, "ndjson", cfg.Format)
	assert.Equal(t, "debug", cfg.Level)
	assert.False(t, cfg.Quiet)
	assert.False(t, cfg.Verbose)
	assert.Equal(t, "booted", cfg.Defaults.Simulator)
	assert.Equal(t, 100, cfg.Defaults.BufferSize)
	assert.Equal(t, "5m", cfg.Defaults.Since)
	assert.Equal(t, 1000, cfg.Defaults.Limit)
	assert.Equal(t, "booted", cfg.Tail.Simulator)
	assert.Equal(t, "booted", cfg.Query.Simulator)
	assert.Equal(t, "booted", cfg.Watch.Simulator)
	assert.Equal(t, "5m", cfg.Query.Since)
	assert.Equal(t, 1000, cfg.Query.Limit)
	assert.Equal(t, "5s", cfg.Watch.Cooldown)
}

func TestLoad(t *testing.T) {
	t.Run("returns defaults when no config file exists", func(t *testing.T) {
		// Create temp dir with no config
		tmpDir := t.TempDir()
		origDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		t.Cleanup(func() {
			require.NoError(t, os.Chdir(origDir))
		})

		cfg, err := Load()
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Should have default values
		assert.Equal(t, "ndjson", cfg.Format)
	})

	t.Run("loads config from file", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create config file
		configContent := `
format: text
level: error
quiet: true
defaults:
  simulator: "iPhone 15"
  buffer_size: 500
`
		configPath := filepath.Join(tmpDir, "xcw.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := LoadFromFile(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.Equal(t, "text", cfg.Format)
		assert.Equal(t, "error", cfg.Level)
		assert.True(t, cfg.Quiet)
		assert.Equal(t, "iPhone 15", cfg.Defaults.Simulator)
		assert.Equal(t, 500, cfg.Defaults.BufferSize)
	})
}

func TestLoadFromFile(t *testing.T) {
	t.Run("returns error for non-existent file", func(t *testing.T) {
		cfg, err := LoadFromFile("/nonexistent/path/config.yaml")
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "bad.yaml")
		err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
		require.NoError(t, err)

		cfg, err := LoadFromFile(configPath)
		assert.Error(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("parses all config fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		configContent := `
format: ndjson
level: debug
quiet: false
verbose: true
defaults:
  simulator: booted
  app: com.test.app
  buffer_size: 200
  summary_interval: 30s
  heartbeat: 10s
  since: 10m
  limit: 500
  subsystems:
    - com.test.app
  categories:
    - network
  exclude_subsystems:
    - com.apple.*
  exclude_pattern: "heartbeat|keepalive"
tail:
  simulator: "iPhone 17 Pro"
  app: com.tail
  summary_interval: 15s
  heartbeat: 3s
  session_idle: 45s
  exclude:
    - noise
  where:
    - level=error
query:
  simulator: booted
  app: com.query
  since: 20m
  limit: 250
  exclude:
    - spam
  where:
    - process=MyProcess
watch:
  simulator: "iPhone 17 Pro"
  app: com.watch
  cooldown: 2s
`
		configPath := filepath.Join(tmpDir, "xcw.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := LoadFromFile(configPath)
		require.NoError(t, err)

		assert.Equal(t, "ndjson", cfg.Format)
		assert.Equal(t, "debug", cfg.Level)
		assert.False(t, cfg.Quiet)
		assert.True(t, cfg.Verbose)
		assert.Equal(t, "booted", cfg.Defaults.Simulator)
		assert.Equal(t, "com.test.app", cfg.Defaults.App)
		assert.Equal(t, 200, cfg.Defaults.BufferSize)
		assert.Equal(t, "30s", cfg.Defaults.SummaryInterval)
		assert.Equal(t, "10s", cfg.Defaults.Heartbeat)
		assert.Equal(t, "10m", cfg.Defaults.Since)
		assert.Equal(t, 500, cfg.Defaults.Limit)
		assert.Contains(t, cfg.Defaults.Subsystems, "com.test.app")
		assert.Contains(t, cfg.Defaults.Categories, "network")
		assert.Contains(t, cfg.Defaults.ExcludeSubsystems, "com.apple.*")
		assert.Equal(t, "heartbeat|keepalive", cfg.Defaults.ExcludePattern)
		assert.Equal(t, "iPhone 17 Pro", cfg.Tail.Simulator)
		assert.Equal(t, "com.tail", cfg.Tail.App)
		assert.Equal(t, "15s", cfg.Tail.SummaryInterval)
		assert.Equal(t, "3s", cfg.Tail.Heartbeat)
		assert.Equal(t, "45s", cfg.Tail.SessionIdle)
		assert.Equal(t, []string{"noise"}, cfg.Tail.Exclude)
		assert.Equal(t, []string{"level=error"}, cfg.Tail.Where)
		assert.Equal(t, "booted", cfg.Query.Simulator)
		assert.Equal(t, "com.query", cfg.Query.App)
		assert.Equal(t, "20m", cfg.Query.Since)
		assert.Equal(t, 250, cfg.Query.Limit)
		assert.Equal(t, []string{"spam"}, cfg.Query.Exclude)
		assert.Equal(t, []string{"process=MyProcess"}, cfg.Query.Where)
		assert.Equal(t, "iPhone 17 Pro", cfg.Watch.Simulator)
		assert.Equal(t, "com.watch", cfg.Watch.App)
		assert.Equal(t, "2s", cfg.Watch.Cooldown)
	})
}

func TestConfigEnvironmentVariables(t *testing.T) {
	// Set env variables
	t.Setenv("XCW_FORMAT", "text")
	t.Setenv("XCW_APP", "com.env.app")

	// Load config (should pick up env vars)
	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "text", cfg.Format)
	assert.Equal(t, "com.env.app", cfg.Defaults.App)
}

func TestDefaultsConfig(t *testing.T) {
	defaults := DefaultsConfig{
		Simulator:         "booted",
		App:               "com.test",
		BufferSize:        100,
		SummaryInterval:   "30s",
		Heartbeat:         "10s",
		Subsystems:        []string{"sub1", "sub2"},
		Categories:        []string{"cat1"},
		Since:             "5m",
		Limit:             1000,
		ExcludeSubsystems: []string{"exclude1"},
		ExcludePattern:    "pattern",
	}

	assert.Equal(t, "booted", defaults.Simulator)
	assert.Equal(t, "com.test", defaults.App)
	assert.Equal(t, 100, defaults.BufferSize)
	assert.Len(t, defaults.Subsystems, 2)
	assert.Len(t, defaults.Categories, 1)
}

func TestFindConfigFile(t *testing.T) {
	t.Run("finds .xcw.yaml in current directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		t.Cleanup(func() {
			require.NoError(t, os.Chdir(origDir))
		})

		// Create config file
		configPath := filepath.Join(tmpDir, ".xcw.yaml")
		err = os.WriteFile(configPath, []byte("format: text"), 0644)
		require.NoError(t, err)

		found := findConfigFile()
		// Resolve symlinks for comparison (macOS /var -> /private/var)
		expectedPath, err := filepath.EvalSymlinks(configPath)
		require.NoError(t, err)
		foundPath, err := filepath.EvalSymlinks(found)
		require.NoError(t, err)
		assert.Equal(t, expectedPath, foundPath)
	})

	t.Run("finds .xcw.yml in current directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		t.Cleanup(func() {
			require.NoError(t, os.Chdir(origDir))
		})

		configPath := filepath.Join(tmpDir, ".xcw.yml")
		err = os.WriteFile(configPath, []byte("format: text"), 0644)
		require.NoError(t, err)

		found := findConfigFile()
		expectedPath, err := filepath.EvalSymlinks(configPath)
		require.NoError(t, err)
		foundPath, err := filepath.EvalSymlinks(found)
		require.NoError(t, err)
		assert.Equal(t, expectedPath, foundPath)
	})

	t.Run("prefers .xcw.yaml over .xcw.yml", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		t.Cleanup(func() {
			require.NoError(t, os.Chdir(origDir))
		})

		// Create both files
		yamlPath := filepath.Join(tmpDir, ".xcw.yaml")
		ymlPath := filepath.Join(tmpDir, ".xcw.yml")
		err = os.WriteFile(yamlPath, []byte("format: yaml"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(ymlPath, []byte("format: yml"), 0644)
		require.NoError(t, err)

		found := findConfigFile()
		expectedPath, err := filepath.EvalSymlinks(yamlPath)
		require.NoError(t, err)
		foundPath, err := filepath.EvalSymlinks(found)
		require.NoError(t, err)
		assert.Equal(t, expectedPath, foundPath)
	})

	t.Run("returns empty string when no config found", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		t.Cleanup(func() {
			require.NoError(t, os.Chdir(origDir))
		})

		found := findConfigFile()
		assert.Empty(t, found)
	})
}

func TestEnvOverridesViaViper(t *testing.T) {
	t.Run("format overrides from env", func(t *testing.T) {
		t.Setenv("XCW_FORMAT", "text")
		cfg, err := Load()
		require.NoError(t, err)
		assert.Equal(t, "text", cfg.Format)
	})

	t.Run("quiet overrides from env", func(t *testing.T) {
		t.Setenv("XCW_QUIET", "true")
		cfg, err := Load()
		require.NoError(t, err)
		assert.True(t, cfg.Quiet)
	})

	t.Run("tail heartbeat override via env replacer", func(t *testing.T) {
		t.Setenv("XCW_TAIL_HEARTBEAT", "2s")
		cfg, err := Load()
		require.NoError(t, err)
		assert.Equal(t, "2s", cfg.Tail.Heartbeat)
	})

	t.Run("simulator shortcut propagates to defaults and tail/query/watch", func(t *testing.T) {
		t.Setenv("XCW_SIMULATOR", "iPhone 17 Pro")
		cfg, err := Load()
		require.NoError(t, err)
		assert.Equal(t, "iPhone 17 Pro", cfg.Defaults.Simulator)
		assert.Equal(t, "iPhone 17 Pro", cfg.Tail.Simulator)
		assert.Equal(t, "iPhone 17 Pro", cfg.Query.Simulator)
		assert.Equal(t, "iPhone 17 Pro", cfg.Watch.Simulator)
	})

	t.Run("query limit override via env", func(t *testing.T) {
		t.Setenv("XCW_QUERY_LIMIT", "42")
		cfg, err := Load()
		require.NoError(t, err)
		assert.Equal(t, 42, cfg.Query.Limit)
	})
}
