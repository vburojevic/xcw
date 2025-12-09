package cli

import (
	"encoding/json"
	"fmt"

	"github.com/vburojevic/xcw/internal/config"
)

// ConfigCmd shows or manages configuration
type ConfigCmd struct {
	Show ConfigShowCmd `cmd:"" default:"withargs" help:"Show current configuration"`
	Path ConfigPathCmd `cmd:"" help:"Show configuration file path"`
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
