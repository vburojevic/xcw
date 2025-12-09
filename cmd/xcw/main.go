package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/vburojevic/xcw/internal/cli"
	"github.com/vburojevic/xcw/internal/config"
)

func main() {
	// Load configuration from files/environment
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = config.Default()
	}

	var c cli.CLI

	// Apply config defaults before parsing
	// These will be overridden by CLI flags if specified
	vars := kong.Vars{
		"config_format":    cfg.Format,
		"config_level":     cfg.Level,
		"config_simulator": cfg.Defaults.Simulator,
		"config_since":     cfg.Defaults.Since,
	}

	ctx := kong.Parse(&c,
		kong.Name("xcw"),
		kong.Description("XcodeConsoleWatcher: Tail iOS Simulator logs for AI agents"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
		vars,
	)

	// Create globals with config fallbacks
	globals := cli.NewGlobalsWithConfig(&c, cfg)
	err = ctx.Run(globals)
	if err != nil {
		os.Exit(1)
	}
}
