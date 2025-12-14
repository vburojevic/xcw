package cli

import (
	"encoding/json"
	"fmt"

	"github.com/vburojevic/xcw/internal/config"
	"github.com/vburojevic/xcw/internal/output"
)

// ConfigCmd shows or manages configuration
type ConfigCmd struct {
	Show     ConfigShowCmd     `cmd:"" default:"withargs" help:"Show current configuration"`
	Path     ConfigPathCmd     `cmd:"" help:"Show configuration file path"`
	Generate ConfigGenerateCmd `cmd:"" help:"Generate sample configuration file"`
}

// ConfigShowCmd shows current configuration
type ConfigShowCmd struct{}

// Run executes the config show command
func (c *ConfigShowCmd) Run(globals *Globals) error {
	cfg := globals.Config
	if cfg == nil {
		cfg = config.Default()
	}

	sources := globals.ConfigSources
	configFile := globals.ConfigFile
	if sources == nil {
		_, meta, err := config.LoadWithMeta()
		if err == nil && meta != nil {
			configFile = meta.ConfigFile
			sources = config.ComputeSources(meta, globals.FlagsSet)
		} else {
			sources = config.ComputeSources(nil, globals.FlagsSet)
		}
	}
	src := func(key string) string {
		if sources == nil {
			return string(config.SourceDefault)
		}
		if v, ok := sources[key]; ok && v != "" {
			return v
		}
		return string(config.SourceDefault)
	}

	if globals.Format == "ndjson" {
		output := map[string]interface{}{
			"type":          "config",
			"schemaVersion": output.SchemaVersion,
			"config_file":   configFile,
			"format":        globals.Format,
			"level":         globals.Level,
			"quiet":         globals.Quiet,
			"verbose":       globals.Verbose,
			"defaults":      cfg.Defaults,
			"tail":          cfg.Tail,
			"query":         cfg.Query,
			"watch":         cfg.Watch,
			"sources":       sources,
		}
		encoder := json.NewEncoder(globals.Stdout)
		return encoder.Encode(output)
	}

	// Text output
	if _, err := fmt.Fprintln(globals.Stdout, "Current Configuration:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(globals.Stdout); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  format:  %s (%s)\n", globals.Format, src("format")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  level:   %s (%s)\n", globals.Level, src("level")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  quiet:   %v (%s)\n", globals.Quiet, src("quiet")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  verbose: %v (%s)\n", globals.Verbose, src("verbose")); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(globals.Stdout); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(globals.Stdout, "Defaults:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  simulator:   %s (%s)\n", cfg.Defaults.Simulator, src("defaults.simulator")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  app:         %s (%s)\n", cfg.Defaults.App, src("defaults.app")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  buffer_size: %d (%s)\n", cfg.Defaults.BufferSize, src("defaults.buffer_size")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  since:       %s (%s)\n", cfg.Defaults.Since, src("defaults.since")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  limit:       %d (%s)\n", cfg.Defaults.Limit, src("defaults.limit")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "\nTail defaults:\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  simulator:        %s (%s)\n", cfg.Tail.Simulator, src("tail.simulator")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  heartbeat:        %s (%s)\n", cfg.Tail.Heartbeat, src("tail.heartbeat")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  summary_interval: %s (%s)\n", cfg.Tail.SummaryInterval, src("tail.summary_interval")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  session_idle:     %s (%s)\n", cfg.Tail.SessionIdle, src("tail.session_idle")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "\nQuery defaults:\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  simulator: %s (%s)\n", cfg.Query.Simulator, src("query.simulator")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  since:     %s (%s)\n", cfg.Query.Since, src("query.since")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  limit:     %d (%s)\n", cfg.Query.Limit, src("query.limit")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "\nWatch defaults:\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  simulator: %s (%s)\n", cfg.Watch.Simulator, src("watch.simulator")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  cooldown:  %s (%s)\n", cfg.Watch.Cooldown, src("watch.cooldown")); err != nil {
		return err
	}

	if len(cfg.Defaults.ExcludeSubsystems) > 0 {
		if _, err := fmt.Fprintf(globals.Stdout, "  exclude_subsystems: %v (%s)\n", cfg.Defaults.ExcludeSubsystems, src("defaults.exclude_subsystems")); err != nil {
			return err
		}
	}
	if cfg.Defaults.ExcludePattern != "" {
		if _, err := fmt.Fprintf(globals.Stdout, "  exclude_pattern: %s (%s)\n", cfg.Defaults.ExcludePattern, src("defaults.exclude_pattern")); err != nil {
			return err
		}
	}

	if configFile != "" {
		if _, err := fmt.Fprintln(globals.Stdout); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(globals.Stdout, "Loaded from: %s\n", configFile); err != nil {
			return err
		}
	}

	return nil
}

// ConfigPathCmd shows config file path
type ConfigPathCmd struct{}

// Run executes the config path command
func (c *ConfigPathCmd) Run(globals *Globals) error {
	path := config.ConfigFile()

	if globals.Format == "ndjson" {
		output := map[string]interface{}{
			"type":          "config_path",
			"schemaVersion": output.SchemaVersion,
			"path":          path,
		}
		encoder := json.NewEncoder(globals.Stdout)
		return encoder.Encode(output)
	}

	if path == "" {
		if _, err := fmt.Fprintln(globals.Stdout, "No configuration file found"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(globals.Stdout); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(globals.Stdout, "Create one at:"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(globals.Stdout, "  ./.xcw.yaml (or ./.xcw.yml)"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(globals.Stdout, "  ~/.xcw.yaml (or ~/.xcw.yml)"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(globals.Stdout, "  ~/.config/xcw/config.yaml"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(globals.Stdout, "Config file: %s\n", path); err != nil {
			return err
		}
	}

	return nil
}

// ConfigGenerateCmd generates a sample configuration file
type ConfigGenerateCmd struct{}

// Run executes the config generate command
func (c *ConfigGenerateCmd) Run(globals *Globals) error {
	sampleConfig := `# xcw configuration file
# Place this file at:
#   - ./.xcw.yaml (or ./.xcw.yml)
#   - ~/.xcw.yaml (or ~/.xcw.yml)
#   - ~/.config/xcw/config.yaml
#   - /etc/xcw/config.yaml

# Output format: "ndjson" (default) or "text"
format: ndjson

# Global log level filter: debug, info, default, error, fault
level: default

# Suppress non-log output (info messages, warnings)
quiet: false

# Enable verbose/debug output
verbose: false

# Default values for commands
defaults:
  # Default simulator selection: "booted" or a simulator name/UDID
  simulator: booted

  # Default app bundle identifier for filtering
  # app: com.example.myapp

  # Ring buffer size for tail command
  buffer_size: 100

  # Summary interval for tail --summary
  # summary_interval: 30s

  # Heartbeat interval for tail --heartbeat
  # heartbeat: 10s

  # Default time range for query command
  since: 5m

  # Maximum entries to return from query
  limit: 1000

  # Subsystems to include (empty = all)
  # subsystems:
  #   - com.example.myapp
  #   - com.example.myapp.network

  # Categories to include (empty = all)
  # categories:
  #   - network
  #   - ui

  # Subsystems to exclude (supports * wildcard)
  # exclude_subsystems:
  #   - com.apple.*
  #   - libsystem_*

  # Regex pattern to exclude from messages
  # exclude_pattern: "heartbeat|keepalive"

tail:
  # heartbeat: 10s
  # summary_interval: 30s
  # session_idle: 60s
  # simulator: booted

query:
  # simulator: booted
  # since: 5m
  # limit: 1000

watch:
  # simulator: booted
  # cooldown: 5s
`

	if _, err := fmt.Fprint(globals.Stdout, sampleConfig); err != nil {
		return err
	}
	return nil
}
