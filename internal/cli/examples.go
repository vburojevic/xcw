package cli

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ExamplesCmd shows usage examples for xcw commands
type ExamplesCmd struct {
	Command string `arg:"" optional:"" help:"Show examples for specific command (tail, query, apps, list, etc.)"`
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
	Type      string             `json:"type"`
	Version   string             `json:"version"`
	Commands  []CommandExamples  `json:"commands"`
	Workflows []WorkflowExample  `json:"workflows"`
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
				Command:     `xcw tail -s "iPhone 17 Pro" -a com.example.myapp -l error`,
				Description: "Only show error and fault level logs",
				When:        "Reduce noise, focus on problems",
			},
			{
				Command:     `xcw tail -s "iPhone 17 Pro" -a com.example.myapp -x "heartbeat|keepalive"`,
				Description: "Exclude noisy log patterns",
				When:        "Filter out repetitive logs",
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
		for _, cmd := range []string{"tail", "query", "list", "apps", "doctor", "analyze", "replay", "sessions", "schema"} {
			if examples, ok := commandExamples[cmd]; ok {
				all.Commands = append(all.Commands, examples)
			}
		}
	}

	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(globals.Stdout, string(data))
	return nil
}

func (c *ExamplesCmd) outputText(globals *Globals) error {
	var sb strings.Builder

	if c.Command != "" {
		// Single command
		if examples, ok := commandExamples[c.Command]; ok {
			c.formatCommandExamples(&sb, examples)
		} else {
			return fmt.Errorf("unknown command: %s\nAvailable: tail, query, list, apps, doctor, analyze, replay, sessions, schema", c.Command)
		}
	} else {
		// All commands
		sb.WriteString("XCW USAGE EXAMPLES\n")
		sb.WriteString("==================\n\n")

		for _, cmd := range []string{"tail", "query", "list", "apps", "doctor", "analyze", "replay", "sessions", "schema"} {
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

	fmt.Fprint(globals.Stdout, sb.String())
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
