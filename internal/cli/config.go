package cli

import (
	"encoding/json"
	"fmt"

	"github.com/vburojevic/xcw/internal/config"
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

	if globals.Format == "ndjson" {
		output := map[string]interface{}{
			"type":     "config",
			"format":   cfg.Format,
			"level":    cfg.Level,
			"quiet":    cfg.Quiet,
			"verbose":  cfg.Verbose,
			"defaults": cfg.Defaults,
		}
		encoder := json.NewEncoder(globals.Stdout)
		return encoder.Encode(output)
	}

	// Text output
	fmt.Fprintln(globals.Stdout, "Current Configuration:")
	fmt.Fprintln(globals.Stdout, "")
	fmt.Fprintf(globals.Stdout, "  format:  %s\n", cfg.Format)
	fmt.Fprintf(globals.Stdout, "  level:   %s\n", cfg.Level)
	fmt.Fprintf(globals.Stdout, "  quiet:   %v\n", cfg.Quiet)
	fmt.Fprintf(globals.Stdout, "  verbose: %v\n", cfg.Verbose)
	fmt.Fprintln(globals.Stdout, "")
	fmt.Fprintln(globals.Stdout, "Defaults:")
	fmt.Fprintf(globals.Stdout, "  simulator:   %s\n", cfg.Defaults.Simulator)
	fmt.Fprintf(globals.Stdout, "  app:         %s\n", cfg.Defaults.App)
	fmt.Fprintf(globals.Stdout, "  buffer_size: %d\n", cfg.Defaults.BufferSize)
	fmt.Fprintf(globals.Stdout, "  since:       %s\n", cfg.Defaults.Since)
	fmt.Fprintf(globals.Stdout, "  limit:       %d\n", cfg.Defaults.Limit)

	if len(cfg.Defaults.ExcludeSubsystems) > 0 {
		fmt.Fprintf(globals.Stdout, "  exclude_subsystems: %v\n", cfg.Defaults.ExcludeSubsystems)
	}
	if cfg.Defaults.ExcludePattern != "" {
		fmt.Fprintf(globals.Stdout, "  exclude_pattern: %s\n", cfg.Defaults.ExcludePattern)
	}

	if path := config.ConfigFile(); path != "" {
		fmt.Fprintln(globals.Stdout, "")
		fmt.Fprintf(globals.Stdout, "Loaded from: %s\n", path)
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
			"type": "config_path",
			"path": path,
		}
		encoder := json.NewEncoder(globals.Stdout)
		return encoder.Encode(output)
	}

	if path == "" {
		fmt.Fprintln(globals.Stdout, "No configuration file found")
		fmt.Fprintln(globals.Stdout, "")
		fmt.Fprintln(globals.Stdout, "Create one at:")
		fmt.Fprintln(globals.Stdout, "  ~/.xcw.yaml")
		fmt.Fprintln(globals.Stdout, "  ~/.xcwrc")
		fmt.Fprintln(globals.Stdout, "  ./xcw.yaml")
	} else {
		fmt.Fprintf(globals.Stdout, "Config file: %s\n", path)
	}

	return nil
}

// ConfigGenerateCmd generates a sample configuration file
type ConfigGenerateCmd struct{}

// Run executes the config generate command
func (c *ConfigGenerateCmd) Run(globals *Globals) error {
	sampleConfig := `# xcw configuration file
# Place this file at ~/.xcw.yaml, ~/.xcwrc, or ./xcw.yaml

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
`

	fmt.Fprint(globals.Stdout, sampleConfig)
	return nil
}
