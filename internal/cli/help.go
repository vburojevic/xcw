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
	Type           string                  `json:"type"`
	Version        string                  `json:"version"`
	Purpose        string                  `json:"purpose"`
	PrimaryCommand string                  `json:"primary_command"`
	AgentGuidance  string                  `json:"agent_guidance"`
	QuickStart     map[string]string       `json:"quick_start"`
	Commands       map[string]CommandDoc   `json:"commands"`
	OutputTypes    map[string]OutputTypeDoc `json:"output_types"`
	ErrorCodes     map[string]ErrorCodeDoc `json:"error_codes"`
	Workflows      []WorkflowDoc           `json:"workflows"`
}

// CommandDoc documents a single command
type CommandDoc struct {
	Description    string       `json:"description"`
	Usage          string       `json:"usage"`
	Examples       []ExampleDoc `json:"examples"`
	OutputTypes    []string     `json:"output_types,omitempty"`
	RelatedCommands []string    `json:"related_commands,omitempty"`
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
		fmt.Fprintln(globals.Stdout, "Usage: xcw help --json")
		fmt.Fprintln(globals.Stdout, "")
		fmt.Fprintln(globals.Stdout, "Output complete xcw documentation as JSON for AI agents.")
		fmt.Fprintln(globals.Stdout, "")
		fmt.Fprintln(globals.Stdout, "For human-readable help, use: xcw --help")
		fmt.Fprintln(globals.Stdout, "For usage examples, use: xcw examples")
		return nil
	}

	doc := buildDocumentation()
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(globals.Stdout, string(data))
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
			"filter_by_field":      `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --where level=error`,
			"discover_log_sources": `xcw discover -s "iPhone 17 Pro" -a com.example.myapp --since 5m`,
			"record_to_file":       `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --session-dir ~/.xcw/sessions`,
			"analyze_session":      `xcw analyze $(xcw sessions show --latest)`,
			"check_setup":          `xcw doctor`,
		},
		Commands: map[string]CommandDoc{
			"tail": {
				Description: "Stream real-time logs from iOS Simulator",
				Usage:       "xcw tail -s SIMULATOR -a APP [flags]",
				Examples: []ExampleDoc{
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp`, Description: "Basic streaming to stdout"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --tmux`, Description: "Background with tmux (returns session name)"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --output file.ndjson`, Description: "Stream to file"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp -l error`, Description: "Only error/fault level"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --filter "error|warn"`, Description: "Filter by regex (alias for --pattern)"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --wait-for-launch`, Description: "Start capture before app launches (emits ready event)"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --where level=error`, Description: "Filter by field (supports =, !=, ~, !~, >=, <=, ^, $)"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --where "message~timeout"`, Description: "Filter messages containing 'timeout'"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --dedupe`, Description: "Collapse repeated identical messages"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --process MyApp --process MyAppExtension`, Description: "Filter by process name"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp -x noise -x spam`, Description: "Exclude multiple patterns"},
					{Command: `xcw tail -s "iPhone 17 Pro" -a com.example.myapp --session-idle 60s`, Description: "Force a new session boundary after 60s of inactivity"},
				},
				OutputTypes:     []string{"log", "session_start", "session_end", "ready", "summary", "heartbeat", "tmux", "error"},
				RelatedCommands: []string{"query", "watch", "analyze", "discover"},
			},
			"query": {
				Description: "Query historical logs from iOS Simulator",
				Usage:       "xcw query -s SIMULATOR -a APP --since DURATION [flags]",
				Examples: []ExampleDoc{
					{Command: `xcw query -s "iPhone 17 Pro" -a com.example.myapp --since 5m`, Description: "Last 5 minutes"},
					{Command: `xcw query -s "iPhone 17 Pro" -a com.example.myapp --since 5m -l error`, Description: "Errors only"},
					{Command: `xcw query -s "iPhone 17 Pro" -a com.example.myapp --since 10m --analyze`, Description: "With pattern analysis"},
				},
				OutputTypes:     []string{"log", "analysis", "error"},
				RelatedCommands: []string{"tail", "analyze"},
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
			"doctor": {
				Description: "Check system requirements and configuration",
				Usage:       "xcw doctor",
				Examples: []ExampleDoc{
					{Command: `xcw doctor`, Description: "Run all checks"},
				},
				OutputTypes: []string{"doctor", "error"},
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
					"type":           "heartbeat",
					"schemaVersion":  1,
					"timestamp":      "2024-01-15T10:30:45.123Z",
					"uptime_seconds": 300,
					"logs_since_last": 42,
				},
				When: "Periodically when --heartbeat is used",
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
					"type":          "ready",
					"schemaVersion": 1,
					"timestamp":     "2024-01-15T10:30:45.123Z",
					"simulator":     "iPhone 17 Pro",
					"udid":          "ABC123-DEF456-...",
					"app":           "com.example.myapp",
				},
				When: "Immediately after log stream starts when --wait-for-launch is used",
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
			"DEVICE_NOT_FOUND":         {Description: "Simulator not found by name or UDID", Recovery: "Run 'xcw list' to see available simulators"},
			"NO_BOOTED_SIMULATOR":      {Description: "No booted simulator when --booted used", Recovery: "Boot a simulator in Xcode or use 'xcrun simctl boot'"},
			"INVALID_FLAGS":            {Description: "--simulator and --booted used together", Recovery: "Use only one of --simulator or --booted"},
			"INVALID_PATTERN":          {Description: "Regex pattern compilation failed", Recovery: "Check regex syntax"},
			"INVALID_DURATION":         {Description: "Duration parsing failed", Recovery: "Use format like '5m', '1h', '30s'"},
			"STREAM_FAILED":            {Description: "Log streaming failed", Recovery: "Check simulator is running and accessible"},
			"QUERY_FAILED":             {Description: "Historical log query failed", Recovery: "Check simulator is running"},
			"FILE_NOT_FOUND":           {Description: "Input file not found", Recovery: "Check file path exists"},
			"TMUX_NOT_INSTALLED":       {Description: "tmux not installed", Recovery: "Install with 'brew install tmux'"},
			"TMUX_ERROR":               {Description: "tmux operation failed", Recovery: "Check tmux is working: 'tmux list-sessions'"},
			"LIST_APPS_FAILED":         {Description: "Failed to list apps", Recovery: "Check simulator is booted"},
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
