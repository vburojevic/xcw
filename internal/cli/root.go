package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/vburojevic/xcw/internal/config"
)

// CLI is the root command structure for XcodeConsoleWatcher
type CLI struct {
	// Global flags
	Format  string `short:"f" default:"ndjson" enum:"ndjson,text" help:"Output format"`
	Level   string `short:"l" default:"default" enum:"debug,info,default,error,fault" help:"Minimum log level"`
	Quiet   bool   `short:"q" help:"Suppress non-log output (only emit log entries)"`
	Verbose bool   `short:"v" help:"Show debug output (predicates, reconnections, internal state)"`
	Version VersionCmd `cmd:"" help:"Show version information"`
	Update  UpdateCmd  `cmd:"" help:"Show how to upgrade xcw"`

	// Commands
	Help       HelpCmd       `cmd:"" help:"Show comprehensive documentation (use --json for AI agents)"`
	Examples   ExamplesCmd   `cmd:"" help:"Show usage examples for xcw commands"`
	List       ListCmd       `cmd:"" help:"List available simulators"`
	Tail       TailCmd       `cmd:"" default:"withargs" help:"Stream logs from a running simulator"`
	Query      QueryCmd      `cmd:"" help:"Query historical logs from simulator"`
	Summary    SummaryCmd    `cmd:"" help:"Output summary of recent logs"`
	Watch      WatchCmd      `cmd:"" help:"Watch logs and trigger commands on patterns"`
	Clear      ClearCmd      `cmd:"" help:"Clear tmux session content"`
	Apps       AppsCmd       `cmd:"" help:"List installed apps on a simulator"`
	Pick       PickCmd       `cmd:"" help:"Interactively pick a simulator or app"`
	Analyze    AnalyzeCmd    `cmd:"" help:"Analyze a recorded NDJSON log file"`
	Replay     ReplayCmd     `cmd:"" help:"Replay a recorded NDJSON log file"`
	Schema     SchemaCmd     `cmd:"" help:"Output JSON Schema for xcw output types"`
	Config     ConfigCmd     `cmd:"" help:"Show or manage configuration"`
	Doctor     DoctorCmd     `cmd:"" help:"Check system requirements and configuration"`
	Completion CompletionCmd `cmd:"" help:"Generate shell completions"`
	UI         UICmd         `cmd:"" help:"Interactive TUI log viewer"`
	Sessions   SessionsCmd   `cmd:"" help:"Manage session log files"`
}

// Globals holds shared state for all commands
type Globals struct {
	Format  string
	Level   string
	Quiet   bool
	Verbose bool
	Stdout  io.Writer
	Stderr  io.Writer
	Config  *config.Config
}

// NewGlobals creates a new Globals instance from CLI flags
func NewGlobals(cli *CLI) *Globals {
	return &Globals{
		Format:  cli.Format,
		Level:   cli.Level,
		Quiet:   cli.Quiet,
		Verbose: cli.Verbose,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Config:  config.Default(),
	}
}

// NewGlobalsWithConfig creates a new Globals instance with config fallbacks
func NewGlobalsWithConfig(cli *CLI, cfg *config.Config) *Globals {
	g := &Globals{
		Format:  cli.Format,
		Level:   cli.Level,
		Quiet:   cli.Quiet,
		Verbose: cli.Verbose,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Config:  cfg,
	}

	// Apply config values if CLI flags weren't explicitly set
	if cfg != nil {
		// If quiet wasn't set via CLI, use config value
		if !cli.Quiet && cfg.Quiet {
			g.Quiet = cfg.Quiet
		}
		// If verbose wasn't set via CLI, use config value
		if !cli.Verbose && cfg.Verbose {
			g.Verbose = cfg.Verbose
		}
	}

	return g
}

// Debug prints a debug message if verbose mode is enabled
func (g *Globals) Debug(format string, args ...interface{}) {
	if g.Verbose {
		fmt.Fprintf(g.Stderr, "[DEBUG] "+format+"\n", args...)
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
