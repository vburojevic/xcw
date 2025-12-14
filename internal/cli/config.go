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
			"tail":     cfg.Tail,
			"query":    cfg.Query,
			"watch":    cfg.Watch,
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
	if _, err := fmt.Fprintf(globals.Stdout, "  format:  %s\n", cfg.Format); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  level:   %s\n", cfg.Level); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  quiet:   %v\n", cfg.Quiet); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  verbose: %v\n", cfg.Verbose); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(globals.Stdout); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(globals.Stdout, "Defaults:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  simulator:   %s\n", cfg.Defaults.Simulator); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  app:         %s\n", cfg.Defaults.App); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  buffer_size: %d\n", cfg.Defaults.BufferSize); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  since:       %s\n", cfg.Defaults.Since); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "  limit:       %d\n", cfg.Defaults.Limit); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "Tail defaults: simulator=%s heartbeat=%s summary_interval=%s session_idle=%s\n",
		cfg.Tail.Simulator, cfg.Tail.Heartbeat, cfg.Tail.SummaryInterval, cfg.Tail.SessionIdle); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "Query defaults: simulator=%s since=%s limit=%d\n",
		cfg.Query.Simulator, cfg.Query.Since, cfg.Query.Limit); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(globals.Stdout, "Watch defaults: simulator=%s cooldown=%s\n",
		cfg.Watch.Simulator, cfg.Watch.Cooldown); err != nil {
		return err
	}

	if len(cfg.Defaults.ExcludeSubsystems) > 0 {
		if _, err := fmt.Fprintf(globals.Stdout, "  exclude_subsystems: %v\n", cfg.Defaults.ExcludeSubsystems); err != nil {
			return err
		}
	}
	if cfg.Defaults.ExcludePattern != "" {
		if _, err := fmt.Fprintf(globals.Stdout, "  exclude_pattern: %s\n", cfg.Defaults.ExcludePattern); err != nil {
			return err
		}
	}

	if path := config.ConfigFile(); path != "" {
		if _, err := fmt.Fprintln(globals.Stdout); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(globals.Stdout, "Loaded from: %s\n", path); err != nil {
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
			"type": "config_path",
			"path": path,
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
		if _, err := fmt.Fprintln(globals.Stdout, "  ~/.xcw.yaml"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(globals.Stdout, "  ~/.xcwrc"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(globals.Stdout, "  ./xcw.yaml"); err != nil {
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
