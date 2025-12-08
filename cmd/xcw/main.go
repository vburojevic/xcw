package main

import (
	"os"

	"github.com/alecthomas/kong"
	"github.com/vedranburojevic/xcw/internal/cli"
)

func main() {
	var c cli.CLI

	ctx := kong.Parse(&c,
		kong.Name("xcw"),
		kong.Description("XcodeConsoleWatcher: Tail iOS Simulator logs for AI agents"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
	)

	globals := cli.NewGlobals(&c)
	err := ctx.Run(globals)
	if err != nil {
		os.Exit(1)
	}
}
