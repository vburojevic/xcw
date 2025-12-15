package cli

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ExamplesCmd shows usage examples for xcw commands
type ExamplesCmd struct {
	Command string `arg:"" optional:"" help:"Show examples for specific command (tail, query, watch, etc.)"`
	JSON    bool   `help:"Output as JSON for programmatic access"`
}

// Example represents a single usage example
type Example struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Output      string `json:"output,omitempty"`
	When        string `json:"when,omitempty"`
}

// CommandExamples holds examples for a single command
type CommandExamples struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Examples    []Example `json:"examples"`
}

// AllExamples contains examples for all commands
type AllExamples struct {
	Type      string            `json:"type"`
	Version   string            `json:"version"`
	Commands  []CommandExamples `json:"commands"`
	Workflows []WorkflowExample `json:"workflows"`
}

// WorkflowExample shows a multi-step workflow
type WorkflowExample struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	When        string   `json:"when"`
	Steps       []string `json:"steps"`
}

var commandExamples = map[string]CommandExamples{
	"tail": {
		Name:        "tail",
		Description: "Stream real-time logs from iOS Simulator",
		Examples: []Example{
			{
				Command:     `xcw tail -s "iPhone 17 Pro" -a com.example.myapp`,
				Description: "Basic streaming to stdout",
				Output:      `{"type":"log","level":"Error","message":"Connection failed",...}`,
				When:        "Real-time log monitoring with Codex or script-based agents",
			},
			{
				Command:     `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --tmux`,
				Description: "Background monitoring with tmux",
				Output:      `{"type":"tmux","session":"xcw-iphone-17-pro","attach":"tmux attach -t xcw-iphone-17-pro"}`,
				When:        "Long-running sessions where you need to do other work",
			},
			{
				Command:     `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --output session.ndjson`,
				Description: "Stream to file for later analysis",
				When:        "Need to replay or share logs across analysis passes",
			},
			{
				Command:     `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --max-duration 5m`,
				Description: "Stream for 5 minutes and stop (emits cutoff_reached)",
				When:        "Bounded monitoring windows for agents or scripts",
			},
			{
				Command:     `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --max-logs 1000`,
				Description: "Stop after 1000 logs (emits cutoff_reached)",
				When:        "Keep output volume bounded for downstream consumers",
			},
			{
				Command:     `xcw tail -s "iPhone 17 Pro" -a com.example.myapp -l error`,
				Description: "Only show error and fault level logs",
				When:        "Reduce noise, focus on problems",
			},
			{
				Command:     `xcw tail -s "iPhone 17 Pro" -a com.example.myapp -x "heartbeat|keepalive"`,
				Description: "Exclude noisy log patterns",
				When:        "Filter out repetitive logs",
			},
			{
				Command:     `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --dry-run-json`,
				Description: "Print resolved stream options as JSON and exit",
				When:        "Debugging predicates/filters before starting a stream",
			},
		},
	},
	"query": {
		Name:        "query",
		Description: "Query historical logs from iOS Simulator",
		Examples: []Example{
			{
				Command:     `xcw query -s "iPhone 17 Pro" -a com.example.myapp --since 5m`,
				Description: "Query logs from the last 5 minutes",
				When:        "Check recent activity after a test run",
			},
			{
				Command:     `xcw query -s "iPhone 17 Pro" -a com.example.myapp --since 5m -l error`,
				Description: "Query only error-level logs",
				When:        "Find errors after running tests",
			},
			{
				Command:     `xcw query -s "iPhone 17 Pro" -a com.example.myapp --since 1h --limit 100`,
				Description: "Query last hour, limit to 100 entries",
				When:        "Bounded query for large time ranges",
			},
			{
				Command:     `xcw query -s "iPhone 17 Pro" -a com.example.myapp --since 10m --analyze`,
				Description: "Query with pattern analysis",
				Output:      `{"type":"analysis","summary":{...},"patterns":[...]}`,
				When:        "Get grouped error patterns and counts",
			},
			{
				Command:     `xcw query -s "iPhone 17 Pro" -a com.example.myapp --since 5m --dry-run-json`,
				Description: "Print resolved query options as JSON and exit",
				When:        "Debugging query settings before running",
			},
		},
	},
	"watch": {
		Name:        "watch",
		Description: "Stream logs and run triggers on patterns",
		Examples: []Example{
			{
				Command:     `xcw watch -s "iPhone 17 Pro" -a com.example.myapp --where level>=error --on-error "./notify.sh" --trigger-output capture`,
				Description: "Run a command when error-level logs appear",
				When:        "Auto-notify and capture diagnostics on failures",
			},
			{
				Command:     `xcw watch -s "iPhone 17 Pro" -a com.example.myapp --on-pattern 'crash|fatal:./notify.sh' --cooldown 10s`,
				Description: "Run a command when a regex matches the message",
				When:        "Trigger on crash signatures or keywords",
			},
			{
				Command:     `xcw watch -s "iPhone 17 Pro" -a com.example.myapp --where level>=error --on-error "./notify.sh" --max-duration 5m`,
				Description: "Watch for 5 minutes and stop (agent-safe cutoff)",
				When:        "Bounded monitoring in CI or scripted workflows",
			},
			{
				Command:     `xcw watch -s "iPhone 17 Pro" -a com.example.myapp --where level>=error --on-error "./notify.sh" --dry-run-json`,
				Description: "Print resolved stream options and triggers as JSON and exit",
				When:        "Debugging trigger configuration before starting a stream",
			},
		},
	},
	"discover": {
		Name:        "discover",
		Description: "Discover subsystems, categories, and processes in logs",
		Examples: []Example{
			{
				Command:     `xcw discover -s "iPhone 17 Pro" --since 5m`,
				Description: "Discover recent log sources",
				When:        "Before choosing subsystems/categories for filters",
			},
			{
				Command:     `xcw discover -s "iPhone 17 Pro" -a com.example.myapp --since 10m`,
				Description: "Discover log sources for a specific app",
				When:        "When your app logs across multiple subsystems/categories",
			},
		},
	},
	"list": {
		Name:        "list",
		Description: "List available iOS Simulators",
		Examples: []Example{
			{
				Command:     `xcw list`,
				Description: "List all simulators",
				Output:      `{"type":"simulator","name":"iPhone 17 Pro","udid":"...","state":"Booted"}`,
			},
			{
				Command:     `xcw list --booted-only`,
				Description: "List only booted simulators",
				When:        "Find which simulators are running",
			},
			{
				Command:     `xcw list -f text`,
				Description: "Human-readable output",
				When:        "Manual inspection",
			},
		},
	},
	"launch": {
		Name:        "launch",
		Description: "Launch app and capture stdout/stderr (print statements)",
		Examples: []Example{
			{
				Command:     `xcw launch -s "iPhone 17 Pro" -a com.example.myapp`,
				Description: "Launch app and capture stdout/stderr output",
				When:        "You need to capture Swift print() statements",
			},
			{
				Command:     `xcw launch -s "iPhone 17 Pro" -a com.example.myapp --terminate-existing`,
				Description: "Terminate existing instance before launching",
				When:        "Ensure a clean launch before capturing output",
			},
		},
	},
	"apps": {
		Name:        "apps",
		Description: "List installed apps on a simulator",
		Examples: []Example{
			{
				Command:     `xcw apps -s "iPhone 17 Pro"`,
				Description: "List all installed apps",
				Output:      `{"type":"app","bundle_id":"com.example.myapp","name":"MyApp",...}`,
			},
			{
				Command:     `xcw apps -s "iPhone 17 Pro" --user-only`,
				Description: "List only user-installed apps (exclude system apps)",
				When:        "Find your app's bundle ID",
			},
		},
	},
	"doctor": {
		Name:        "doctor",
		Description: "Check system requirements and configuration",
		Examples: []Example{
			{
				Command:     `xcw doctor`,
				Description: "Run all diagnostic checks",
				Output:      `{"type":"doctor","all_passed":true,"checks":[...]}`,
				When:        "Verify xcw is set up correctly",
			},
		},
	},
	"analyze": {
		Name:        "analyze",
		Description: "Analyze a recorded NDJSON log file",
		Examples: []Example{
			{
				Command:     `xcw analyze session.ndjson`,
				Description: "Analyze a recorded log file",
				Output:      `{"type":"analysis","summary":{...},"patterns":[...]}`,
				When:        "Post-process recorded logs",
			},
		},
	},
	"replay": {
		Name:        "replay",
		Description: "Replay a recorded NDJSON log file",
		Examples: []Example{
			{
				Command:     `xcw replay session.ndjson`,
				Description: "Replay logs with original timing",
				When:        "Reproduce a debugging session",
			},
			{
				Command:     `xcw replay session.ndjson --speed 2`,
				Description: "Replay at 2x speed",
				When:        "Faster review of recorded logs",
			},
		},
	},
	"sessions": {
		Name:        "sessions",
		Description: "Manage session log files",
		Examples: []Example{
			{
				Command:     `xcw sessions list`,
				Description: "List recent session files",
				Output:      `{"type":"session","path":"~/.xcw/sessions/...","name":"...","size":1234}`,
			},
			{
				Command:     `xcw sessions show --latest`,
				Description: "Get path to most recent session",
				When:        "Pipe to analyze: xcw analyze $(xcw sessions show --latest)",
			},
			{
				Command:     `xcw sessions clean --keep 10`,
				Description: "Delete old sessions, keep 10 most recent",
				When:        "Free up disk space",
			},
		},
	},
	"schema": {
		Name:        "schema",
		Description: "Output JSON Schema for xcw output types",
		Examples: []Example{
			{
				Command:     `xcw schema`,
				Description: "Output full JSON Schema",
				When:        "Validate xcw output programmatically",
			},
			{
				Command:     `xcw schema --type log,error`,
				Description: "Output schema for specific types only",
			},
		},
	},
	"examples": {
		Name:        "examples",
		Description: "Show curated usage examples for xcw commands",
		Examples: []Example{
			{
				Command:     `xcw examples`,
				Description: "Show all examples",
			},
			{
				Command:     `xcw examples tail`,
				Description: "Show examples for a single command",
			},
			{
				Command:     `xcw examples --json`,
				Description: "Machine-readable examples",
			},
		},
	},
	"log-schema": {
		Name:        "log-schema",
		Description: "Output minimal log schema docs for agents",
		Examples: []Example{
			{
				Command:     `xcw log-schema`,
				Description: "Minimal log schema fields and example",
				When:        "You need a compact schema definition for agents/parsers",
			},
		},
	},
	"handoff": {
		Name:        "handoff",
		Description: "Emit a compact JSON handoff blob for agents",
		Examples: []Example{
			{
				Command:     `xcw handoff`,
				Description: "Emit contract hints + versions for agent handoff",
				When:        "Transfer context to another agent/tooling stage",
			},
		},
	},
	"version": {
		Name:        "version",
		Description: "Show version information",
		Examples: []Example{
			{
				Command:     `xcw version`,
				Description: "NDJSON version output (default)",
				Output:      `{"type":"version","version":"0.19.10","commit":"none"}`,
			},
			{
				Command:     `xcw -f text version`,
				Description: "Human-readable version output",
				When:        "Manual inspection in a terminal",
			},
		},
	},
	"update": {
		Name:        "update",
		Description: "Show how to upgrade xcw",
		Examples: []Example{
			{
				Command:     `xcw update`,
				Description: "Upgrade instructions (NDJSON by default)",
				Output:      `{"type":"update","current_version":"0.19.10","homebrew":"brew update && brew upgrade xcw",...}`,
			},
			{
				Command:     `xcw -f text update`,
				Description: "Human-readable upgrade instructions",
				When:        "Manual inspection in a terminal",
			},
		},
	},
	"config": {
		Name:        "config",
		Description: "Show or manage configuration",
		Examples: []Example{
			{
				Command:     `xcw config`,
				Description: "Show effective configuration (default: show)",
				Output:      `{"type":"config","config_file":"~/.config/xcw/config.yaml","sources":{...}}`,
			},
			{
				Command:     `xcw config path`,
				Description: "Show resolved config file path",
				Output:      `{"type":"config_path","path":"~/.config/xcw/config.yaml"}`,
			},
			{
				Command:     `xcw config generate > .xcw.yaml`,
				Description: "Generate a sample config file",
				When:        "Bootstrap configuration for team/CI usage",
			},
		},
	},
	"pick": {
		Name:        "pick",
		Description: "Interactively pick a simulator or app",
		Examples: []Example{
			{
				Command:     `xcw pick simulator`,
				Description: "Pick a simulator and print its UDID",
				When:        "You want to avoid copy-pasting simulator IDs",
			},
			{
				Command:     `xcw pick app`,
				Description: "Pick an app on the booted simulator and print bundle id",
				When:        "You want to quickly get a bundle id for tail/query/watch",
			},
		},
	},
	"completion": {
		Name:        "completion",
		Description: "Generate shell completions",
		Examples: []Example{
			{
				Command:     `xcw completion zsh > _xcw`,
				Description: "Generate zsh completion script",
				When:        "Install completions locally",
			},
			{
				Command:     `xcw completion bash > xcw.bash`,
				Description: "Generate bash completion script",
				When:        "Install completions locally",
			},
		},
	},
	"summary": {
		Name:        "summary",
		Description: "Summarize recent logs for an app (bounded query + analysis)",
		Examples: []Example{
			{
				Command:     `xcw summary -a com.example.myapp --window 5m`,
				Description: "Summarize last 5 minutes",
				Output:      `{"type":"analysis","summary":{...},"patterns":[...]}`,
			},
			{
				Command:     `xcw summary -s "iPhone 17 Pro" -a com.example.myapp --window 30m -p "error|fatal"`,
				Description: "Summarize last 30 minutes with a regex filter",
				When:        "Reduce noise and focus on errors during a run",
			},
		},
	},
	"clear": {
		Name:        "clear",
		Description: "Clear a tmux session pane",
		Examples: []Example{
			{
				Command:     `xcw clear --session xcw-iphone-17-pro`,
				Description: "Clear a tmux pane used by tail/watch --tmux",
				When:        "Reset the view between runs",
			},
		},
	},
	"ui": {
		Name:        "ui",
		Description: "Interactive TUI log viewer (human mode)",
		Examples: []Example{
			{
				Command:     `xcw ui -s "iPhone 17 Pro" -a com.example.myapp`,
				Description: "Open TUI for an app on a simulator",
				When:        "Manual interactive log exploration (not suitable for agents)",
			},
		},
	},
}

var workflows = []WorkflowExample{
	{
		Name:        "codex_streaming",
		Description: "Real-time log streaming for Codex/script-based agents",
		When:        "Agent processes stdout line by line, reacting immediately to errors",
		Steps: []string{
			`xcw tail -s "iPhone 17 Pro" -a com.example.myapp`,
			"# Agent reads NDJSON lines from stdout",
			"# Each line is a complete JSON object",
		},
	},
	{
		Name:        "claude_code_background",
		Description: "Background monitoring for Claude Code",
		When:        "Long-running session where agent does other work while logs stream",
		Steps: []string{
			`xcw tail -s "iPhone 17 Pro" -a com.example.myapp --tmux`,
			"# Returns immediately with session info",
			"# Agent continues working...",
			`xcw query -s "iPhone 17 Pro" -a com.example.myapp --since 5m -l error`,
			"# Query when needed to check for errors",
		},
	},
	{
		Name:        "test_run_analysis",
		Description: "Capture and analyze logs from a test run",
		When:        "Need to save logs for later analysis or sharing",
		Steps: []string{
			`xcw tail -s "iPhone 17 Pro" -a com.example.myapp --output test-run.ndjson`,
			"# Run tests in another terminal...",
			"# Stop tail with Ctrl+C when done",
			`xcw analyze test-run.ndjson`,
			"# Get summary and error patterns",
		},
	},
}

// Run executes the examples command
func (c *ExamplesCmd) Run(globals *Globals) error {
	if c.JSON {
		return c.outputJSON(globals)
	}
	return c.outputText(globals)
}

func (c *ExamplesCmd) outputJSON(globals *Globals) error {
	all := AllExamples{
		Type:      "examples",
		Version:   Version,
		Workflows: workflows,
	}

	if c.Command != "" {
		// Single command
		if examples, ok := commandExamples[c.Command]; ok {
			all.Commands = []CommandExamples{examples}
		} else {
			return fmt.Errorf("unknown command: %s", c.Command)
		}
	} else {
		// All commands
		for _, cmd := range []string{"tail", "query", "watch", "summary", "discover", "list", "apps", "pick", "launch", "ui", "clear", "doctor", "config", "schema", "log-schema", "handoff", "completion", "examples", "update", "version", "analyze", "replay", "sessions"} {
			if examples, ok := commandExamples[cmd]; ok {
				all.Commands = append(all.Commands, examples)
			}
		}
	}

	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(globals.Stdout, string(data)); err != nil {
		return err
	}
	return nil
}

func (c *ExamplesCmd) outputText(globals *Globals) error {
	var sb strings.Builder

	if c.Command != "" {
		// Single command
		if examples, ok := commandExamples[c.Command]; ok {
			c.formatCommandExamples(&sb, examples)
		} else {
			return fmt.Errorf("unknown command: %s\nAvailable: tail, query, watch, summary, discover, list, apps, pick, launch, ui, clear, doctor, config, schema, log-schema, handoff, completion, examples, update, version, analyze, replay, sessions", c.Command)
		}
	} else {
		// All commands
		sb.WriteString("XCW USAGE EXAMPLES\n")
		sb.WriteString("==================\n\n")

		for _, cmd := range []string{"tail", "query", "watch", "summary", "discover", "list", "apps", "pick", "launch", "ui", "clear", "doctor", "config", "schema", "log-schema", "handoff", "completion", "examples", "update", "version", "analyze", "replay", "sessions"} {
			if examples, ok := commandExamples[cmd]; ok {
				c.formatCommandExamples(&sb, examples)
				sb.WriteString("\n")
			}
		}

		// Workflows
		sb.WriteString("WORKFLOWS\n")
		sb.WriteString("---------\n\n")
		for _, wf := range workflows {
			sb.WriteString(fmt.Sprintf("## %s\n", wf.Name))
			sb.WriteString(fmt.Sprintf("%s\n", wf.Description))
			sb.WriteString(fmt.Sprintf("When: %s\n\n", wf.When))
			for _, step := range wf.Steps {
				sb.WriteString(fmt.Sprintf("  %s\n", step))
			}
			sb.WriteString("\n")
		}
	}

	if _, err := fmt.Fprint(globals.Stdout, sb.String()); err != nil {
		return err
	}
	return nil
}

func (c *ExamplesCmd) formatCommandExamples(sb *strings.Builder, cmd CommandExamples) {
	sb.WriteString(fmt.Sprintf("## %s\n", strings.ToUpper(cmd.Name)))
	sb.WriteString(fmt.Sprintf("%s\n\n", cmd.Description))

	for _, ex := range cmd.Examples {
		sb.WriteString(fmt.Sprintf("  %s\n", ex.Command))
		sb.WriteString(fmt.Sprintf("    %s\n", ex.Description))
		if ex.Output != "" {
			sb.WriteString(fmt.Sprintf("    Output: %s\n", ex.Output))
		}
		if ex.When != "" {
			sb.WriteString(fmt.Sprintf("    When: %s\n", ex.When))
		}
		sb.WriteString("\n")
	}
}
