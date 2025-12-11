package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vburojevic/xcw/internal/config"
)

func TestNewGlobalsWithConfig_UsesConfigWhenCLILeftDefault(t *testing.T) {
	cli := &CLI{Format: "ndjson", Level: "debug", Quiet: false, Verbose: false}
	cfg := &config.Config{
		Format:  "text",
		Level:   "error",
		Quiet:   true,
		Verbose: true,
	}

	globals := NewGlobalsWithConfig(cli, cfg)

	assert.Equal(t, "text", globals.Format)
	assert.Equal(t, "error", globals.Level)
	assert.True(t, globals.Quiet)
	assert.True(t, globals.Verbose)
}

func TestNewGlobalsWithConfig_PreservesExplicitCLIChoices(t *testing.T) {
	cli := &CLI{Format: "text", Level: "info", Quiet: true, Verbose: true}
	cfg := &config.Config{
		Format:  "ndjson",
		Level:   "error",
		Quiet:   false,
		Verbose: false,
	}

	globals := NewGlobalsWithConfig(cli, cfg)

	assert.Equal(t, "text", globals.Format)
	assert.Equal(t, "info", globals.Level)
	assert.True(t, globals.Quiet)
	assert.True(t, globals.Verbose)
}

func TestApplyTailDefaultsUsesConfig(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultsConfig{
			Simulator:  "booted",
			BufferSize: 250,
		},
		Tail: config.TailConfig{
			Simulator:       "cfg-sim",
			App:             "com.cfg",
			SummaryInterval: "15s",
			Heartbeat:       "5s",
			SessionIdle:     "30s",
			Exclude:         []string{"noise"},
			Where:           []string{"level=error"},
		},
	}

	cmd := &TailCmd{
		BufferSize:      100,
		TailFilterFlags: TailFilterFlags{},
		TailOutputFlags: TailOutputFlags{},
		TailAgentFlags:  TailAgentFlags{},
	}

	applyTailDefaults(cfg, cmd)

	assert.Equal(t, "cfg-sim", cmd.Simulator)
	assert.Equal(t, "com.cfg", cmd.App)
	assert.Equal(t, "15s", cmd.SummaryInterval)
	assert.Equal(t, "5s", cmd.Heartbeat)
	assert.Equal(t, "30s", cmd.SessionIdle)
	assert.Equal(t, []string{"noise"}, cmd.Exclude)
	assert.Equal(t, []string{"level=error"}, cmd.Where)
	assert.Equal(t, 250, cmd.BufferSize)
}

func TestApplyTailDefaultsDoesNotOverrideExplicitValues(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultsConfig{
			BufferSize: 250,
		},
		Tail: config.TailConfig{
			Simulator:       "cfg-sim",
			SummaryInterval: "15s",
			Heartbeat:       "5s",
			SessionIdle:     "30s",
		},
	}

	cmd := &TailCmd{
		Simulator: "cli-sim",
		App:       "com.cli",
		TailFilterFlags: TailFilterFlags{
			Exclude: []string{"keep"},
			Where:   []string{"keep=true"},
		},
		TailOutputFlags: TailOutputFlags{
			SummaryInterval: "25s",
			Heartbeat:       "3s",
		},
		TailAgentFlags: TailAgentFlags{
			SessionIdle: "40s",
		},
		BufferSize: 999,
	}

	applyTailDefaults(cfg, cmd)

	assert.Equal(t, "cli-sim", cmd.Simulator)
	assert.Equal(t, "com.cli", cmd.App)
	assert.Equal(t, "25s", cmd.SummaryInterval)
	assert.Equal(t, "3s", cmd.Heartbeat)
	assert.Equal(t, "40s", cmd.SessionIdle)
	assert.Equal(t, 999, cmd.BufferSize)
	assert.Equal(t, []string{"keep"}, cmd.Exclude)
	assert.Equal(t, []string{"keep=true"}, cmd.Where)
}

func TestApplyQueryDefaults(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultsConfig{
			Simulator: "booted",
		},
		Query: config.QueryConfig{
			Simulator: "cfg-sim",
			App:       "com.query",
			Since:     "10m",
			Limit:     500,
			Exclude:   []string{"noise"},
			Where:     []string{"level=error"},
		},
	}

	cmd := &QueryCmd{
		Since: "5m",
		Limit: 1000,
	}

	applyQueryDefaults(cfg, cmd)

	assert.Equal(t, "cfg-sim", cmd.Simulator)
	assert.Equal(t, "com.query", cmd.App)
	assert.Equal(t, "10m", cmd.Since)
	assert.Equal(t, 500, cmd.Limit)
	assert.Equal(t, []string{"noise"}, cmd.Exclude)
	assert.Equal(t, []string{"level=error"}, cmd.Where)
}

func TestApplyWatchDefaults(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.DefaultsConfig{
			Simulator: "booted",
		},
		Watch: config.WatchConfig{
			Simulator: "cfg-sim",
			App:       "com.watch",
			Cooldown:  "1s",
		},
	}

	cmd := &WatchCmd{
		Cooldown: "5s",
	}

	applyWatchDefaults(cfg, cmd)

	assert.Equal(t, "cfg-sim", cmd.Simulator)
	assert.Equal(t, "com.watch", cmd.App)
	assert.Equal(t, "1s", cmd.Cooldown)
}
