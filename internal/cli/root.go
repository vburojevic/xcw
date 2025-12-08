package cli

import (
	"io"
	"os"
)

// CLI is the root command structure for XcodeConsoleWatcher
type CLI struct {
	// Global flags
	Format  string `short:"f" default:"ndjson" enum:"ndjson,text" help:"Output format"`
	Level   string `short:"l" default:"default" enum:"debug,info,default,error,fault" help:"Minimum log level"`
	Quiet   bool   `short:"q" help:"Suppress non-log output (only emit log entries)"`
	Version VersionCmd `cmd:"" help:"Show version information"`

	// Commands
	List    ListCmd    `cmd:"" help:"List available simulators"`
	Tail    TailCmd    `cmd:"" default:"withargs" help:"Stream logs from a running simulator"`
	Query   QueryCmd   `cmd:"" help:"Query historical logs from simulator"`
	Summary SummaryCmd `cmd:"" help:"Output summary of recent logs"`
	Clear   ClearCmd   `cmd:"" help:"Clear tmux session content"`
}

// Globals holds shared state for all commands
type Globals struct {
	Format string
	Level  string
	Quiet  bool
	Stdout io.Writer
	Stderr io.Writer
}

// NewGlobals creates a new Globals instance from CLI flags
func NewGlobals(cli *CLI) *Globals {
	return &Globals{
		Format: cli.Format,
		Level:  cli.Level,
		Quiet:  cli.Quiet,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// VersionCmd shows version information
type VersionCmd struct{}

// Run executes the version command
func (v *VersionCmd) Run(globals *Globals) error {
	if globals.Format == "ndjson" {
		io.WriteString(globals.Stdout, `{"type":"version","version":"`+Version+`","commit":"`+Commit+`"}`+"\n")
	} else {
		io.WriteString(globals.Stdout, "xcw version "+Version+" ("+Commit+")\n")
	}
	return nil
}

// Version information (set at build time)
var (
	Version = "dev"
	Commit  = "none"
)
