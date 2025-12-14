package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/vburojevic/xcw/internal/cli"
	"github.com/vburojevic/xcw/internal/config"
)

const quickStart = `xcw - iOS Simulator log streaming for AI agents

START HERE (this is the command you want):
  xcw tail -s "iPhone 17 Pro" -a com.example.myapp

Flags:
  -s    Simulator name (run 'xcw list' to see available)
  -a    Your app's bundle ID (run 'xcw apps' to list installed)

Other useful commands:
  xcw list                              List simulators
  xcw apps                              List installed apps
  xcw help --json                       Full docs for AI agents
`

func main() {
	// Show quick start if no args provided
	if len(os.Args) == 1 {
		fmt.Print(quickStart)
		return
	}

	// Load configuration from files/environment (plus provenance metadata).
	cfg, meta, err := config.LoadWithMeta()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = config.Default()
		meta = nil
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
		kong.Description("XcodeConsoleWatcher: Stream iOS Simulator logs\n\nSTART HERE: xcw tail -a <bundle_id>\n\nAI agents: run 'xcw help --json' for complete documentation"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
		vars,
	)

	// Create globals with config fallbacks
	globals := cli.NewGlobalsWithConfig(&c, cfg)
	// Record which flags were explicitly provided so commands can distinguish
	// CLI overrides from config defaults.
	flagsSet := map[string]bool{}
	for _, p := range ctx.Path {
		if p.Flag != nil {
			flagsSet[p.Flag.Name] = true
		}
	}
	globals.FlagsSet = flagsSet
	if meta != nil {
		globals.ConfigFile = meta.ConfigFile
		globals.ConfigSources = config.ComputeSources(meta, flagsSet)
	} else {
		globals.ConfigSources = config.ComputeSources(nil, flagsSet)
	}
	err = ctx.Run(globals)
	if err != nil {
		os.Exit(1)
	}
}
