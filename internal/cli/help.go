package cli

import (
	"encoding/json"
	"fmt"
)

// HelpCmd provides comprehensive documentation
type HelpCmd struct {
	JSON bool `help:"Output complete documentation as JSON for AI agents"`
}

// HelpOutput is the complete documentation structure
type HelpOutput struct {
	Type           string                   `json:"type"`
	Version        string                   `json:"version"`
	Purpose        string                   `json:"purpose"`
	PrimaryCommand string                   `json:"primary_command"`
	AgentGuidance  string                   `json:"agent_guidance"`
	QuickStart     map[string]string        `json:"quick_start"`
	Commands       map[string]CommandDoc    `json:"commands"`
	OutputTypes    map[string]OutputTypeDoc `json:"output_types"`
	ErrorCodes     map[string]ErrorCodeDoc  `json:"error_codes"`
	Workflows      []WorkflowDoc            `json:"workflows"`
	Contract       []string                 `json:"contract"`
}

// CommandDoc documents a single command
type CommandDoc struct {
	Description     string       `json:"description"`
	Usage           string       `json:"usage"`
	Examples        []ExampleDoc `json:"examples"`
	OutputTypes     []string     `json:"output_types,omitempty"`
	RelatedCommands []string     `json:"related_commands,omitempty"`
}

// ExampleDoc is a documented example
type ExampleDoc struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// OutputTypeDoc documents an output type
type OutputTypeDoc struct {
	Description string                 `json:"description"`
	Example     map[string]interface{} `json:"example"`
	When        string                 `json:"when"`
}

// ErrorCodeDoc documents an error code
type ErrorCodeDoc struct {
	Description string `json:"description"`
	Recovery    string `json:"recovery"`
}

// WorkflowDoc documents a workflow
type WorkflowDoc struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	When        string   `json:"when"`
	Steps       []string `json:"steps"`
}

// Run executes the help command
func (c *HelpCmd) Run(globals *Globals) error {
	if !c.JSON {
		if _, err := fmt.Fprintln(globals.Stdout, "Usage: xcw help --json"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(globals.Stdout); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(globals.Stdout, "Output complete xcw documentation as JSON for AI agents."); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(globals.Stdout); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(globals.Stdout, "For human-readable help, use: xcw --help"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(globals.Stdout, "For usage examples, use: xcw examples"); err != nil {
			return err
		}
		return nil
	}

	doc := buildDocumentation()
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(globals.Stdout, string(data)); err != nil {
		return err
	}
	return nil
}

func buildDocumentation() *HelpOutput {
	return &HelpOutput{
		Type:           "documentation",
		Version:        Version,
		Purpose:        "iOS Simulator log streaming CLI optimized for AI agents. Outputs structured NDJSON for real-time and historical log access.",
		PrimaryCommand: "tail",
		AgentGuidance:  "ALWAYS START WITH: xcw tail -s <simulator> -a <bundle_id>. This is THE primary command for monitoring iOS app logs. Run 'xcw list' to get simulator names, and 'xcw apps' to get bundle IDs. Only use other commands (query, watch, analyze) for specific use cases AFTER you've tried tail. The 'query' command reads from macOS system logs and is only useful when you forgot to start tail. Tmux (--tmux) is for human visual monitoring; AI agents should use stdout or --output instead.",
		QuickStart: map[string]string{
			"list_simulators":      `xcw list`,
			"list_apps":            `xcw apps -s "iPhone 17 Pro"`,
			"stream_logs":          `xcw tail -s "iPhone 17 Pro" -a com.example.myapp`,
			"machine_friendly":     `xcw --machine-friendly tail -s "iPhone 17 Pro" -a com.example.myapp`,
			"filter_by_field":      `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --where level=error`,
			"filter_by_expr":       `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --where '(level=error OR level=fault) AND message~timeout'`,
			"discover_log_sources": `xcw discover -s "iPhone 17 Pro" -a com.example.myapp --since 5m`,
			"record_to_file":       `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --session-dir ~/.xcw/sessions`,
			"analyze_session":      `xcw analyze $(xcw sessions show --latest)`,
			"check_setup":          `xcw doctor`,
		},
		Contract: defaultHints(),
		Commands: map[string]CommandDoc{
			"version": {
				Description: "Show version information",
				Usage:       "xcw version",
				Examples: []ExampleDoc{
					{Command: `xcw version`, Description: "Print version (NDJSON by default)"},
					{Command: `xcw -f text version`, Description: "Human-readable version"},
				},
				OutputTypes: []string{"version"},
			},
			"update": {
				Description: "Show how to upgrade xcw (Homebrew / go install)",
				Usage:       "xcw update",
				Examples: []ExampleDoc{
					{Command: `xcw update`, Description: "Upgrade instructions (NDJSON by default)"},
					{Command: `xcw -f text update`, Description: "Human-readable upgrade instructions"},
				},
				OutputTypes: []string{"update"},
			},
			"tail": {
				Description: "Stream real-time logs from iOS Simulator (use -a unless --predicate/--all)",
				Usage:       "xcw tail -s SIMULATOR [-a APP] [flags]",
				Examples: []ExampleDoc{
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp`, Description: "Basic streaming to stdout"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --tmux`, Description: "Background with tmux (returns session name)"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --output file.ndjson`, Description: "Stream to file"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp -l error`, Description: "Only error/fault level"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --filter "error|warn"`, Description: "Filter by regex (alias for --pattern)"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --dry-run-json`, Description: "Print resolved stream options as JSON and exit"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --wait-for-launch`, Description: "Start capture before app launches (emits ready event)"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --where level=error`, Description: "Filter by field/expression (=, !=, ~, !~, >=, <=, ^, $, AND/OR/NOT, parentheses, /regex/i)"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --where '(level=error OR level=fault) AND message~timeout'`, Description: "Boolean where expression"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --where "message~timeout"`, Description: "Filter messages containing 'timeout'"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --dedupe`, Description: "Collapse repeated identical messages"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --process MyApp --process MyAppExtension`, Description: "Filter by process name"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp -x noise -x spam`, Description: "Exclude multiple patterns"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --session-idle 60s`, Description: "Force a new session boundary after 60s of inactivity"},
					{Command: `xcw tail -s "iPhone 17 Pro" --predicate 'process == \"MyApp\"'`, Description: "Stream without -a using a raw predicate (advanced)"},
				},
				OutputTypes:     []string{"log", "session_start", "session_end", "ready", "summary", "heartbeat", "tmux", "error"},
				RelatedCommands: []string{"query", "watch", "analyze", "discover"},
			},
			"query": {
				Description: "Query historical logs from iOS Simulator (use -a unless --predicate/--all)",
				Usage:       "xcw query -s SIMULATOR [-a APP] --since DURATION [flags]",
				Examples: []ExampleDoc{
					{Command: `xcw query -s "iPhone 17 Pro" -a com.example.myapp --since 5m`, Description: "Last 5 minutes"},
					{Command: `xcw query -s "iPhone 17 Pro" -a com.example.myapp --since 5m -l error`, Description: "Errors only"},
					{Command: `xcw query -s "iPhone 17 Pro" -a com.example.myapp --since 10m --analyze`, Description: "With pattern analysis"},
					{Command: `xcw query -s "iPhone 17 Pro" -a com.example.myapp --since 10m --where '(level=error OR level=fault) AND message~timeout'`, Description: "Where expression"},
					{Command: `xcw query -s "iPhone 17 Pro" -a com.example.myapp --since 5m --dry-run-json`, Description: "Print resolved query options as JSON and exit"},
				},
				OutputTypes:     []string{"log", "analysis", "error"},
				RelatedCommands: []string{"tail", "analyze"},
			},
			"summary": {
				Description: "Summarize recent logs for an app (runs a bounded query and outputs analysis)",
				Usage:       "xcw summary -a APP [--window DURATION] [flags]",
				Examples: []ExampleDoc{
					{Command: `xcw summary -a com.example.myapp --window 5m`, Description: "Analyze the last 5 minutes of logs"},
					{Command: `xcw summary -s "iPhone 17 Pro" -a com.example.myapp --window 30m -p "error|fatal"`, Description: "Analyze last 30 minutes with pattern filter"},
				},
				OutputTypes:     []string{"analysis", "error"},
				RelatedCommands: []string{"query", "tail", "analyze", "discover"},
			},
			"watch": {
				Description: "Stream logs like tail, but run trigger commands on errors/faults or message patterns (supports agent-safe cutoffs via --max-duration/--max-logs)",
				Usage:       "xcw watch -s SIMULATOR [-a APP] [flags]",
				Examples: []ExampleDoc{
					{Command: `xcw watch -s "iPhone 17 Pro" -a com.example.myapp --where level>=error --on-error "./notify.sh" --trigger-output capture`, Description: "Run a command when an error-level log appears"},
					{Command: `xcw watch -s "iPhone 17 Pro" -a com.example.myapp --on-pattern 'crash|fatal:./notify.sh' --cooldown 10s`, Description: "Run a command when a regex matches the message"},
					{Command: `xcw watch -s "iPhone 17 Pro" -a com.example.myapp --where level>=error --on-error "./notify.sh" --max-duration 5m`, Description: "Watch for 5 minutes and stop"},
					{Command: `xcw watch -s "iPhone 17 Pro" -a com.example.myapp --where level>=error --on-error "./notify.sh" --dry-run-json`, Description: "Print resolved stream options and triggers as JSON and exit"},
				},
				OutputTypes:     []string{"log", "trigger", "trigger_result", "trigger_error", "cutoff_reached", "tmux", "error"},
				RelatedCommands: []string{"tail", "query", "discover"},
			},
			"list": {
				Description: "List available iOS Simulators",
				Usage:       "xcw list [flags]",
				Examples: []ExampleDoc{
					{Command: `xcw list`, Description: "All simulators"},
					{Command: `xcw list --booted-only`, Description: "Only running simulators"},
				},
				OutputTypes: []string{"simulator", "error"},
			},
			"apps": {
				Description: "List installed apps on a simulator",
				Usage:       "xcw apps -s SIMULATOR [flags]",
				Examples: []ExampleDoc{
					{Command: `xcw apps -s "iPhone 17 Pro"`, Description: "All apps"},
					{Command: `xcw apps -s "iPhone 17 Pro" --user-only`, Description: "User apps only"},
				},
				OutputTypes: []string{"app", "error"},
			},
			"pick": {
				Description: "Interactively pick a simulator or app (prints an ID suitable for scripting)",
				Usage:       "xcw pick [simulator|app] [flags]",
				Examples: []ExampleDoc{
					{Command: `xcw pick simulator`, Description: "Interactively select a simulator (prints UDID)"},
					{Command: `xcw pick app`, Description: "Interactively select an app on the booted simulator (prints bundle id)"},
				},
				OutputTypes:     []string{"pick", "error"},
				RelatedCommands: []string{"list", "apps"},
			},
			"launch": {
				Description: "Launch an app and capture stdout/stderr (print statements). Use this when you need to see print() output that isn't captured by unified logging.",
				Usage:       "xcw launch -s SIMULATOR -a APP [flags]",
				Examples: []ExampleDoc{
					{Command: `xcw launch -s "iPhone 17 Pro" -a com.example.myapp`, Description: "Launch and capture stdout/stderr"},
					{Command: `xcw launch -s "iPhone 17 Pro" -a com.example.myapp --terminate-existing`, Description: "Terminate existing instance first"},
					{Command: `xcw launch -b -a com.example.myapp`, Description: "Launch on booted simulator"},
				},
				OutputTypes:     []string{"console", "info", "error"},
				RelatedCommands: []string{"tail", "apps"},
			},
			"ui": {
				Description: "Interactive TUI log viewer (for humans; not suitable for agents)",
				Usage:       "xcw ui -s SIMULATOR [-a APP] [flags]",
				Examples: []ExampleDoc{
					{Command: `xcw ui -s "iPhone 17 Pro" -a com.example.myapp`, Description: "Open TUI for an app"},
					{Command: `xcw ui -b -a com.example.myapp`, Description: "Open TUI on booted simulator"},
				},
				OutputTypes:     []string{"error"},
				RelatedCommands: []string{"tail", "watch"},
			},
			"clear": {
				Description: "Clear a tmux session pane (useful with tail/watch --tmux)",
				Usage:       "xcw clear --session SESSION",
				Examples: []ExampleDoc{
					{Command: `xcw clear --session xcw-iphone-17-pro`, Description: "Clear the tmux pane"},
				},
				OutputTypes:     []string{"info", "error"},
				RelatedCommands: []string{"tail", "watch"},
			},
			"config": {
				Description: "Show or manage configuration (flags/env/config file precedence)",
				Usage:       "xcw config [show|path|generate]",
				Examples: []ExampleDoc{
					{Command: `xcw config`, Description: "Show effective config (default: show)"},
					{Command: `xcw config path`, Description: "Show config file path resolution"},
					{Command: `xcw config generate`, Description: "Print a sample config file"},
				},
				OutputTypes: []string{"config", "config_path", "error"},
			},
			"doctor": {
				Description: "Check system requirements and configuration",
				Usage:       "xcw doctor",
				Examples: []ExampleDoc{
					{Command: `xcw doctor`, Description: "Run all checks"},
				},
				OutputTypes: []string{"doctor", "error"},
			},
			"completion": {
				Description: "Generate shell completion scripts from the Kong CLI model",
				Usage:       "xcw completion [bash|zsh|fish]",
				Examples: []ExampleDoc{
					{Command: `xcw completion zsh > _xcw`, Description: "Generate zsh completion script"},
					{Command: `xcw completion bash > xcw.bash`, Description: "Generate bash completion script"},
				},
				RelatedCommands: []string{"help", "examples"},
			},
			"analyze": {
				Description: "Analyze a recorded NDJSON log file",
				Usage:       "xcw analyze FILE [flags]",
				Examples: []ExampleDoc{
					{Command: `xcw analyze session.ndjson`, Description: "Analyze recorded logs"},
				},
				OutputTypes:     []string{"analysis", "error"},
				RelatedCommands: []string{"tail", "replay"},
			},
			"replay": {
				Description: "Replay a recorded NDJSON log file with timing",
				Usage:       "xcw replay FILE [flags]",
				Examples: []ExampleDoc{
					{Command: `xcw replay session.ndjson`, Description: "Replay with original timing"},
					{Command: `xcw replay session.ndjson --speed 2`, Description: "2x speed"},
				},
				OutputTypes:     []string{"log", "error"},
				RelatedCommands: []string{"analyze", "tail"},
			},
			"sessions": {
				Description: "Manage session log files",
				Usage:       "xcw sessions [list|show|clean]",
				Examples: []ExampleDoc{
					{Command: `xcw sessions list`, Description: "List recent sessions"},
					{Command: `xcw sessions show --latest`, Description: "Path to latest session"},
					{Command: `xcw sessions clean --keep 10`, Description: "Keep 10 most recent"},
				},
				OutputTypes: []string{"session", "info", "error"},
			},
			"discover": {
				Description: "Discover what subsystems, categories, and processes exist in logs. Essential first step for understanding an app's logging landscape.",
				Usage:       "xcw discover -s SIMULATOR [-a APP] --since DURATION",
				Examples: []ExampleDoc{
					{Command: `xcw discover -s "iPhone 17 Pro" --since 5m`, Description: "Discover all logs from last 5 minutes"},
					{Command: `xcw discover -s "iPhone 17 Pro" -a com.example.myapp --since 10m`, Description: "Discover logs for specific app"},
					{Command: `xcw discover -b --since 1h --top-n 30`, Description: "More items, booted sim, 1 hour"},
				},
				OutputTypes:     []string{"discovery", "error"},
				RelatedCommands: []string{"tail", "query"},
			},
			"log-schema": {
				Description: "Output a minimal log schema document for agents",
				Usage:       "xcw log-schema",
				Examples: []ExampleDoc{
					{Command: `xcw log-schema`, Description: "Minimal log schema fields + example"},
				},
				OutputTypes:     []string{"log_schema"},
				RelatedCommands: []string{"schema", "help"},
			},
			"handoff": {
				Description: "Emit a compact JSON blob for AI agents (contract hints + versions)",
				Usage:       "xcw handoff",
				Examples: []ExampleDoc{
					{Command: `xcw handoff`, Description: "Agent handoff payload"},
				},
				OutputTypes:     []string{"handoff"},
				RelatedCommands: []string{"help", "schema"},
			},
			"schema": {
				Description: "Output JSON Schema for xcw output types",
				Usage:       "xcw schema [flags]",
				Examples: []ExampleDoc{
					{Command: `xcw schema`, Description: "Full schema"},
					{Command: `xcw schema --type log,error`, Description: "Specific types"},
				},
			},
			"examples": {
				Description: "Show usage examples for xcw commands",
				Usage:       "xcw examples [command] [flags]",
				Examples: []ExampleDoc{
					{Command: `xcw examples`, Description: "All examples"},
					{Command: `xcw examples tail`, Description: "Tail examples"},
					{Command: `xcw examples --json`, Description: "Machine-readable"},
				},
			},
		},
		OutputTypes: map[string]OutputTypeDoc{
			"info": {
				Description: "Informational message from xcw (non-log event)",
				Example: map[string]interface{}{
					"type":          "info",
					"schemaVersion": 1,
					"message":       "Watching logs from iPhone 17 Pro",
					"simulator":     "iPhone 17 Pro",
					"udid":          "ABC123-DEF456-...",
				},
				When: "Startup banners and status messages (unless --quiet)",
			},
			"warning": {
				Description: "Warning from xcw (non-fatal)",
				Example: map[string]interface{}{
					"type":          "warning",
					"schemaVersion": 1,
					"message":       "reconnect_notice: reconnecting log stream",
				},
				When: "Non-fatal issues (eg. reconnects, trigger skips, diagnostic warnings)",
			},
			"log": {
				Description: "Individual log entry from iOS Simulator. Includes session number for tracking app relaunches.",
				Example: map[string]interface{}{
					"type":          "log",
					"schemaVersion": 1,
					"tail_id":       "tail-abc",
					"timestamp":     "2024-01-15T10:30:45.123Z",
					"level":         "Error",
					"process":       "MyApp",
					"pid":           1234,
					"subsystem":     "com.example.myapp",
					"category":      "network",
					"message":       "Connection failed: timeout",
					"session":       1,
				},
				When: "Each log entry during tail or query",
			},
			"metadata": {
				Description: "Tool metadata emitted at start of tail for agents.",
				Example: map[string]interface{}{
					"type":             "metadata",
					"schemaVersion":    1,
					"version":          "0.10.0",
					"commit":           "none",
					"contract_version": 1,
				},
				When: "At tail start",
			},
			"cutoff_reached": {
				Description: "Emitted when max-duration or max-logs cutoff stops streaming.",
				Example: map[string]interface{}{
					"type":          "cutoff_reached",
					"schemaVersion": 1,
					"reason":        "max_duration",
					"tail_id":       "tail-abc",
					"session":       2,
					"total_logs":    500,
				},
				When: "When cutoff thresholds are hit",
			},
			"reconnect_notice": {
				Description: "Signals that the log stream reconnected; agents should consider potential gaps.",
				Example: map[string]interface{}{
					"type":          "reconnect_notice",
					"schemaVersion": 1,
					"message":       "reconnect_notice: reconnecting log stream",
					"tail_id":       "tail-abc",
				},
				When: "After reconnect attempts",
			},
			"gap_detected": {
				Description: "Signals that a gap in the stream was detected (and may be backfilled when --resume is enabled).",
				Example: map[string]interface{}{
					"type":           "gap_detected",
					"schemaVersion":  1,
					"timestamp":      "2024-01-15T10:31:00.000Z",
					"tail_id":        "tail-abc",
					"session":        2,
					"from_timestamp": "2024-01-15T10:30:45.123Z",
					"to_timestamp":   "2024-01-15T10:31:00.000Z",
					"reason":         "reconnect",
					"will_fill":      true,
				},
				When: "When xcw detects a potential gap and --resume is enabled (NDJSON only)",
			},
			"gap_filled": {
				Description: "Signals that a previously detected gap was backfilled via query.",
				Example: map[string]interface{}{
					"type":           "gap_filled",
					"schemaVersion":  1,
					"timestamp":      "2024-01-15T10:31:01.000Z",
					"tail_id":        "tail-abc",
					"session":        2,
					"from_timestamp": "2024-01-15T10:30:45.123Z",
					"to_timestamp":   "2024-01-15T10:31:00.000Z",
					"reason":         "reconnect",
					"filled_count":   42,
					"limit":          5000,
				},
				When: "After xcw backfills a gap when --resume is enabled (NDJSON only)",
			},
			"session_start": {
				Description: "Emitted when a new app session begins. When alert='APP_RELAUNCHED', the app was relaunched (PID changed). AI agents should watch for this to know logs are from a fresh app instance.",
				Example: map[string]interface{}{
					"type":          "session_start",
					"schemaVersion": 1,
					"tail_id":       "tail-abc",
					"alert":         "APP_RELAUNCHED",
					"session":       2,
					"pid":           67890,
					"previous_pid":  12345,
					"app":           "com.example.myapp",
					"simulator":     "iPhone 17 Pro",
					"udid":          "ABC123-DEF456-...",
					"version":       "1.4.0",
					"build":         "2201",
					"binary_uuid":   "C0FFEE-UUID",
					"timestamp":     "2024-01-15T10:30:45.123Z",
				},
				When: "When xcw tail detects the app was relaunched (PID changed)",
			},
			"session_end": {
				Description: "Emitted when an app session ends (before session_start when app relaunches). Contains summary of the ended session.",
				Example: map[string]interface{}{
					"type":          "session_end",
					"schemaVersion": 1,
					"tail_id":       "tail-abc",
					"session":       1,
					"pid":           12345,
					"summary": map[string]interface{}{
						"total_logs":       142,
						"errors":           3,
						"faults":           0,
						"duration_seconds": 45,
					},
				},
				When: "When xcw tail detects the app was relaunched (before session_start)",
			},
			"clear_buffer": {
				Description: "Instructs consumers to reset caches at a session boundary (start/end/idle rollover).",
				Example: map[string]interface{}{
					"type":          "clear_buffer",
					"schemaVersion": 1,
					"tail_id":       "tail-abc",
					"session":       2,
					"reason":        "session_start",
				},
				When: "Immediately after session_start/session_end/idle rollover in NDJSON mode",
			},
			"tmux": {
				Description: "Tmux session information when --tmux is used",
				Example: map[string]interface{}{
					"type":          "tmux",
					"schemaVersion": 1,
					"session":       "xcw-iphone-17-pro",
					"attach":        "tmux attach -t xcw-iphone-17-pro",
				},
				When: "Immediately when xcw tail --tmux is called",
			},
			"summary": {
				Description: "Log statistics summary",
				Example: map[string]interface{}{
					"type":          "summary",
					"schemaVersion": 1,
					"tail_id":       "tail-abc",
					"totalCount":    142,
					"errorCount":    3,
					"faultCount":    0,
					"hasErrors":     true,
					"hasFaults":     false,
					"errorRate":     1.2,
				},
				When: "At end of tail, with --summary-interval, or in analysis",
			},
			"analysis": {
				Description: "Pattern analysis of logs",
				Example: map[string]interface{}{
					"type":          "analysis",
					"schemaVersion": 1,
					"summary": map[string]interface{}{
						"totalCount": 100,
						"errorCount": 5,
					},
					"patterns": []map[string]interface{}{
						{"pattern": "Connection failed", "count": 3, "is_new": true},
					},
					"new_pattern_count":   1,
					"known_pattern_count": 2,
				},
				When: "When --analyze flag is used with query or analyze command",
			},
			"heartbeat": {
				Description: "Keepalive message for stream health",
				Example: map[string]interface{}{
					"type":                "heartbeat",
					"schemaVersion":       1,
					"timestamp":           "2024-01-15T10:30:45.123Z",
					"uptime_seconds":      300,
					"logs_since_last":     42,
					"tail_id":             "tail-abc",
					"contract_version":    1,
					"latest_session":      2,
					"last_seen_timestamp": "2024-01-15T10:30:44.999Z",
				},
				When: "Periodically when --heartbeat is used",
			},
			"stats": {
				Description: "Periodic stream diagnostics emitted alongside heartbeats (useful for detecting reconnects, parse drops, and backpressure)",
				Example: map[string]interface{}{
					"type":                  "stats",
					"schemaVersion":         1,
					"timestamp":             "2024-01-15T10:30:45.123Z",
					"tail_id":               "tail-abc",
					"session":               2,
					"reconnects":            1,
					"parse_drops":           0,
					"timestamp_parse_drops": 0,
					"channel_drops":         0,
					"buffered":              10,
					"last_seen_timestamp":   "2024-01-15T10:30:44.999Z",
				},
				When: "Periodically alongside heartbeat output (tail only)",
			},
			"console": {
				Description: "Console output (stdout/stderr) from xcw launch. Captures print() statements not available via unified logging.",
				Example: map[string]interface{}{
					"type":          "console",
					"schemaVersion": 1,
					"timestamp":     "2024-01-15T10:30:45.123Z",
					"stream":        "stdout",
					"message":       "Hello from print()",
					"process":       "com.example.myapp",
				},
				When: "Each line of stdout/stderr from xcw launch",
			},
			"ready": {
				Description: "Signals that log capture is active and ready. Use with --wait-for-launch to start capturing before the app launches.",
				Example: map[string]interface{}{
					"type":             "ready",
					"schemaVersion":    1,
					"timestamp":        "2024-01-15T10:30:45.123Z",
					"simulator":        "iPhone 17 Pro",
					"udid":             "ABC123-DEF456-...",
					"app":              "com.example.myapp",
					"tail_id":          "tail-abc",
					"session":          1,
					"contract_version": 1,
				},
				When: "Immediately after log stream starts when --wait-for-launch is used",
			},
			"trigger": {
				Description: "Emitted when a watch trigger starts running",
				Example: map[string]interface{}{
					"type":          "trigger",
					"schemaVersion": 1,
					"timestamp":     "2024-01-15T10:30:45.123Z",
					"tail_id":       "tail-abc",
					"session":       1,
					"trigger_id":    "trigger-xyz",
					"trigger":       "error",
					"command":       "notify.sh",
					"message":       "Connection failed: timeout",
				},
				When: "When xcw watch starts a trigger command",
			},
			"trigger_result": {
				Description: "Emitted when a watch trigger completes (exit code, duration, optional output)",
				Example: map[string]interface{}{
					"type":          "trigger_result",
					"schemaVersion": 1,
					"timestamp":     "2024-01-15T10:30:45.456Z",
					"tail_id":       "tail-abc",
					"session":       1,
					"trigger_id":    "trigger-xyz",
					"trigger":       "error",
					"command":       "notify.sh",
					"exit_code":     0,
					"duration_ms":   120,
					"output":        "ok",
				},
				When: "After a watch trigger command exits",
			},
			"trigger_error": {
				Description: "Emitted when a watch trigger fails to execute or exits non-zero",
				Example: map[string]interface{}{
					"type":          "trigger_error",
					"schemaVersion": 1,
					"timestamp":     "2024-01-15T10:30:45.456Z",
					"tail_id":       "tail-abc",
					"session":       1,
					"trigger_id":    "trigger-xyz",
					"trigger":       "error",
					"command":       "notify.sh",
					"error":         "exit status 1",
				},
				When: "After a watch trigger command fails",
			},
			"discovery": {
				Description: "Log discovery results showing subsystems, categories, processes, and levels",
				Example: map[string]interface{}{
					"type":          "discovery",
					"schemaVersion": 1,
					"app":           "com.example.myapp",
					"time_range":    map[string]string{"start": "2024-01-15T10:25:45Z", "end": "2024-01-15T10:30:45Z"},
					"total_count":   1250,
					"subsystems": []map[string]interface{}{
						{"name": "com.example.myapp", "count": 450, "levels": map[string]int{"debug": 300, "info": 100, "error": 50}},
					},
					"categories": []map[string]interface{}{
						{"name": "network", "count": 300},
					},
					"processes": []map[string]interface{}{
						{"name": "MyApp", "count": 800},
					},
					"levels": map[string]int{"debug": 700, "info": 350, "error": 80},
				},
				When: "From xcw discover command",
			},
			"error": {
				Description: "Error from xcw (not the app being monitored)",
				Example: map[string]interface{}{
					"type":          "error",
					"schemaVersion": 1,
					"code":          "DEVICE_NOT_FOUND",
					"message":       "Simulator 'iPhone 99' not found",
				},
				When: "When xcw encounters an error",
			},
			"simulator": {
				Description: "Simulator device information",
				Example: map[string]interface{}{
					"type":          "simulator",
					"schemaVersion": 1,
					"name":          "iPhone 17 Pro",
					"udid":          "ABC123-DEF456-...",
					"state":         "Booted",
					"runtime":       "iOS 18.0",
				},
				When: "From xcw list command",
			},
			"app": {
				Description: "Installed app information",
				Example: map[string]interface{}{
					"type":          "app",
					"schemaVersion": 1,
					"bundle_id":     "com.example.myapp",
					"name":          "MyApp",
					"version":       "1.0.0",
					"app_type":      "user",
				},
				When: "From xcw apps command",
			},
			"session": {
				Description: "Session file information",
				Example: map[string]interface{}{
					"type":          "session",
					"schemaVersion": 1,
					"path":          "~/.xcw/sessions/20241215-103045-com.example.myapp.ndjson",
					"name":          "20241215-103045-com.example.myapp.ndjson",
					"size":          12345,
					"timestamp":     "2024-12-15T10:30:45Z",
				},
				When: "From xcw sessions list/show commands",
			},
			"version": {
				Description: "xcw version information",
				Example: map[string]interface{}{
					"type":    "version",
					"version": "0.19.9",
					"commit":  "none",
				},
				When: "From xcw version",
			},
			"update": {
				Description: "xcw upgrade instructions",
				Example: map[string]interface{}{
					"type":            "update",
					"schemaVersion":   1,
					"current_version": "0.19.9",
					"commit":          "none",
					"homebrew":        "brew update && brew upgrade xcw",
					"go_install":      "go install github.com/vburojevic/xcw/cmd/xcw@latest",
					"releases_url":    "https://github.com/vburojevic/xcw/releases",
				},
				When: "From xcw update (NDJSON mode)",
			},
			"config": {
				Description: "Effective configuration and provenance (sources)",
				Example: map[string]interface{}{
					"type":          "config",
					"schemaVersion": 1,
					"config_file":   "~/.config/xcw/config.yaml",
					"format":        "ndjson",
					"level":         "debug",
					"quiet":         false,
					"verbose":       false,
				},
				When: "From xcw config show (NDJSON mode)",
			},
			"config_path": {
				Description: "Resolved configuration file path (if any)",
				Example: map[string]interface{}{
					"type":          "config_path",
					"schemaVersion": 1,
					"path":          "~/.config/xcw/config.yaml",
				},
				When: "From xcw config path (NDJSON mode)",
			},
			"pick": {
				Description: "Pick result from xcw pick (simulator/app)",
				Example: map[string]interface{}{
					"type":          "pick",
					"schemaVersion": 1,
					"picked":        "app",
					"name":          "MyApp",
					"bundle_id":     "com.example.myapp",
				},
				When: "From xcw pick (NDJSON mode)",
			},
			"log_schema": {
				Description: "Minimal schema doc for log events (agents)",
				Example: map[string]interface{}{
					"type":          "log_schema",
					"schemaVersion": 1,
					"fields":        map[string]string{"timestamp": "ISO8601 UTC", "level": "Debug|Info|Default|Error|Fault"},
				},
				When: "From xcw log-schema",
			},
			"handoff": {
				Description: "Compact handoff blob for agents (hints + versions)",
				Example: map[string]interface{}{
					"type":             "handoff",
					"version":          "0.19.9",
					"schemaVersion":    1,
					"contract_version": 1,
					"timestamp":        "2025-12-15T00:00:00Z",
					"hints":            []string{"ALWAYS START WITH: xcw tail ..."},
				},
				When: "From xcw handoff",
			},
			"doctor": {
				Description: "System diagnostic report",
				Example: map[string]interface{}{
					"type":          "doctor",
					"schemaVersion": 1,
					"all_passed":    true,
					"error_count":   0,
					"warn_count":    1,
					"checks": []map[string]interface{}{
						{"name": "Xcode", "status": "ok", "message": "Xcode 16.0 found"},
						{"name": "Simulator", "status": "ok", "message": "1 booted simulator"},
					},
				},
				When: "From xcw doctor command",
			},
		},
		ErrorCodes: map[string]ErrorCodeDoc{
			"DEVICE_NOT_FOUND":    {Description: "Simulator not found by name or UDID", Recovery: "Run 'xcw list' to see available simulators"},
			"NO_BOOTED_SIMULATOR": {Description: "No booted simulator when --booted used", Recovery: "Boot a simulator in Xcode or use 'xcrun simctl boot'"},
			"INVALID_FLAGS":       {Description: "--simulator and --booted used together", Recovery: "Use only one of --simulator or --booted"},
			"FILTER_REQUIRED":     {Description: "No source filter provided (refusing to run unfiltered by default)", Recovery: "Provide -a/--app or --predicate, or pass --all to intentionally stream/query all logs"},
			"INVALID_PATTERN":     {Description: "Regex pattern compilation failed", Recovery: "Check regex syntax"},
			"INVALID_FILTER":      {Description: "Filter parsing/compilation failed", Recovery: "Check regex/--where syntax; quote complex expressions"},
			"INVALID_DURATION":    {Description: "Duration parsing failed", Recovery: "Use format like '5m', '1h', '30s'"},
			"STREAM_FAILED":       {Description: "Log streaming failed", Recovery: "Check simulator is running and accessible"},
			"QUERY_FAILED":        {Description: "Historical log query failed", Recovery: "Check simulator is running"},
			"FILE_NOT_FOUND":      {Description: "Input file not found", Recovery: "Check file path exists"},
			"TMUX_NOT_INSTALLED":  {Description: "tmux not installed", Recovery: "Install with 'brew install tmux'"},
			"TMUX_ERROR":          {Description: "tmux operation failed", Recovery: "Check tmux is working: 'tmux list-sessions'"},
			"LIST_APPS_FAILED":    {Description: "Failed to list apps", Recovery: "Check simulator is booted"},
			"TUI_FAILED":          {Description: "TUI exited with an error", Recovery: "Rerun with -v for debug output or use 'xcw tail' for non-interactive streaming"},
		},
		Workflows: []WorkflowDoc{
			{
				Name:        "codex_streaming",
				Description: "Real-time log streaming for Codex/script-based agents",
				When:        "Agent processes stdout line by line, reacting immediately to errors",
				Steps: []string{
					`xcw tail -s "iPhone 17 Pro" -a com.example.myapp`,
					"# Agent reads NDJSON lines from stdout",
					"# Each line is a complete JSON object to parse",
				},
			},
			{
				Name:        "claude_code_background",
				Description: "Background monitoring for Claude Code with on-demand queries",
				When:        "Long-running session where agent does other work while logs stream",
				Steps: []string{
					`xcw tail -s "iPhone 17 Pro" -a com.example.myapp --tmux`,
					"# Returns tmux session info immediately",
					"# Agent continues with other work...",
					`xcw query -s "iPhone 17 Pro" -a com.example.myapp --since 5m -l error`,
					"# Query errors when needed",
				},
			},
			{
				Name:        "test_run_capture",
				Description: "Capture logs during test run for analysis",
				When:        "Need persistent log record for replay or sharing",
				Steps: []string{
					`xcw tail -s "iPhone 17 Pro" -a com.example.myapp --output test-run.ndjson`,
					"# Run tests...",
					"# Stop with Ctrl+C",
					`xcw analyze test-run.ndjson`,
				},
			},
		},
	}
}
