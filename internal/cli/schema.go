package cli

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SchemaCmd outputs JSON Schema for xcw output types
type SchemaCmd struct {
	Type []string `short:"t" help:"Output types to include (log,summary,heartbeat,error,tmux,info,warning,trigger,doctor,app). Default: all"`
}

// Run executes the schema command
func (c *SchemaCmd) Run(globals *Globals) error {
	schemas := map[string]interface{}{
		"log":       logSchema(),
		"summary":   summarySchema(),
		"heartbeat": heartbeatSchema(),
		"error":     errorSchema(),
		"tmux":      tmuxSchema(),
		"info":      infoSchema(),
		"warning":   warningSchema(),
		"trigger":   triggerSchema(),
		"doctor":    doctorSchema(),
		"app":       appSchema(),
	}

	// Determine which schemas to output
	typesToOutput := c.Type
	if len(typesToOutput) == 0 {
		typesToOutput = []string{"log", "summary", "heartbeat", "error", "tmux", "info", "warning", "trigger", "doctor", "app"}
	}

	// Build output
	output := map[string]interface{}{
		"$schema":     "http://json-schema.org/draft-07/schema#",
		"title":       "XcodeConsoleWatcher Output Schemas",
		"description": "JSON Schema definitions for all xcw NDJSON output types",
		"definitions": map[string]interface{}{},
	}

	defs := output["definitions"].(map[string]interface{})
	for _, t := range typesToOutput {
		t = strings.ToLower(strings.TrimSpace(t))
		if schema, ok := schemas[t]; ok {
			defs[t] = schema
		}
	}

	// Output as JSON
	encoder := json.NewEncoder(globals.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func logSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Log Entry",
		"description": "A single log entry from the iOS Simulator",
		"properties": map[string]interface{}{
			"timestamp": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "ISO8601 timestamp of the log entry",
			},
			"level": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"Debug", "Info", "Default", "Error", "Fault"},
				"description": "Log level/severity",
			},
			"process": map[string]interface{}{
				"type":        "string",
				"description": "Name of the process that generated the log",
			},
			"pid": map[string]interface{}{
				"type":        "integer",
				"description": "Process ID",
			},
			"subsystem": map[string]interface{}{
				"type":        "string",
				"description": "Subsystem identifier (usually bundle ID)",
			},
			"category": map[string]interface{}{
				"type":        "string",
				"description": "Log category within the subsystem",
			},
			"message": map[string]interface{}{
				"type":        "string",
				"description": "The log message content",
			},
		},
		"required": []string{"timestamp", "level", "process", "pid", "message"},
	}
}

func summarySchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Log Summary",
		"description": "Periodic summary of log statistics",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "summary",
			},
			"totalCount": map[string]interface{}{
				"type":        "integer",
				"description": "Total number of log entries",
			},
			"debugCount": map[string]interface{}{
				"type":        "integer",
				"description": "Number of debug-level entries",
			},
			"infoCount": map[string]interface{}{
				"type":        "integer",
				"description": "Number of info-level entries",
			},
			"defaultCount": map[string]interface{}{
				"type":        "integer",
				"description": "Number of default-level entries",
			},
			"errorCount": map[string]interface{}{
				"type":        "integer",
				"description": "Number of error-level entries",
			},
			"faultCount": map[string]interface{}{
				"type":        "integer",
				"description": "Number of fault-level entries",
			},
			"hasErrors": map[string]interface{}{
				"type":        "boolean",
				"description": "True if any errors were detected",
			},
			"hasFaults": map[string]interface{}{
				"type":        "boolean",
				"description": "True if any faults were detected",
			},
			"errorRate": map[string]interface{}{
				"type":        "number",
				"description": "Errors per minute rate",
			},
			"topErrors": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Most common error messages",
			},
			"topFaults": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Most common fault messages",
			},
		},
		"required": []string{"type", "totalCount"},
	}
}

func heartbeatSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Heartbeat",
		"description": "Keepalive message indicating stream is active",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "heartbeat",
			},
			"timestamp": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "ISO8601 timestamp of the heartbeat",
			},
			"uptime_seconds": map[string]interface{}{
				"type":        "integer",
				"description": "Seconds since stream started",
			},
			"logs_since_last": map[string]interface{}{
				"type":        "integer",
				"description": "Number of logs since last heartbeat",
			},
		},
		"required": []string{"type", "timestamp", "uptime_seconds", "logs_since_last"},
	}
}

func errorSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Error",
		"description": "Error message from xcw",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "error",
			},
			"code": map[string]interface{}{
				"type":        "string",
				"description": "Error code (e.g., DEVICE_NOT_FOUND, INVALID_PATTERN)",
				"enum": []string{
					"DEVICE_NOT_FOUND",
					"NO_BOOTED_SIMULATOR",
					"INVALID_PATTERN",
					"INVALID_EXCLUDE_PATTERN",
					"INVALID_DURATION",
					"INVALID_INTERVAL",
					"INVALID_HEARTBEAT",
					"STREAM_FAILED",
					"QUERY_FAILED",
				},
			},
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Human-readable error description",
			},
		},
		"required": []string{"type", "code", "message"},
	}
}

func tmuxSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Tmux Session Info",
		"description": "Information about created tmux session",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "tmux",
			},
			"session": map[string]interface{}{
				"type":        "string",
				"description": "Tmux session name",
			},
			"attach": map[string]interface{}{
				"type":        "string",
				"description": "Command to attach to the session",
			},
		},
		"required": []string{"type", "session", "attach"},
	}
}

func infoSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Info Message",
		"description": "Informational message from xcw",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "info",
			},
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Info message content",
			},
			"device": map[string]interface{}{
				"type":        "string",
				"description": "Device name if applicable",
			},
			"udid": map[string]interface{}{
				"type":        "string",
				"description": "Device UDID if applicable",
			},
		},
		"required": []string{"type", "message"},
	}
}

func warningSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Warning Message",
		"description": "Warning message from xcw",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "warning",
			},
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Warning message content",
			},
		},
		"required": []string{"type", "message"},
	}
}

func triggerSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Trigger Event",
		"description": "Notification when a trigger fires",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "trigger",
			},
			"trigger_type": map[string]interface{}{
				"type":        "string",
				"description": "Type of trigger (error, fault, or pattern:regex)",
			},
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Command being executed",
			},
			"match": map[string]interface{}{
				"type":        "string",
				"description": "Log message that triggered the action",
			},
		},
		"required": []string{"type", "trigger_type", "command"},
	}
}

func doctorSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Doctor Report",
		"description": "System diagnostic report from xcw doctor",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "doctor",
			},
			"timestamp": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "When the check was performed",
			},
			"checks": map[string]interface{}{
				"type":        "array",
				"description": "Individual check results",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Name of the check",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"ok", "warning", "error"},
							"description": "Check result status",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Result message",
						},
						"details": map[string]interface{}{
							"type":        "string",
							"description": "Additional details or remediation steps",
						},
					},
					"required": []string{"name", "status"},
				},
			},
			"all_passed": map[string]interface{}{
				"type":        "boolean",
				"description": "True if all checks passed without errors",
			},
			"error_count": map[string]interface{}{
				"type":        "integer",
				"description": "Number of checks with error status",
			},
			"warn_count": map[string]interface{}{
				"type":        "integer",
				"description": "Number of checks with warning status",
			},
		},
		"required": []string{"type", "timestamp", "checks", "all_passed"},
	}
}

func appSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Installed App",
		"description": "Information about an installed app on the simulator",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "app",
			},
			"bundle_id": map[string]interface{}{
				"type":        "string",
				"description": "App bundle identifier",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "App display name",
			},
			"version": map[string]interface{}{
				"type":        "string",
				"description": "App version string",
			},
			"build_number": map[string]interface{}{
				"type":        "string",
				"description": "App build number",
			},
			"app_type": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"user", "system"},
				"description": "Whether app is user-installed or system",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to app bundle",
			},
		},
		"required": []string{"type", "bundle_id", "name", "app_type"},
	}
}

// Helper to output a quick reference
func (c *SchemaCmd) outputTextHelp(globals *Globals) {
	fmt.Fprintln(globals.Stdout, "XcodeConsoleWatcher Output Types:")
	fmt.Fprintln(globals.Stdout, "")
	fmt.Fprintln(globals.Stdout, "  log       - Log entry from simulator")
	fmt.Fprintln(globals.Stdout, "  summary   - Periodic log statistics")
	fmt.Fprintln(globals.Stdout, "  heartbeat - Keepalive message")
	fmt.Fprintln(globals.Stdout, "  error     - Error from xcw")
	fmt.Fprintln(globals.Stdout, "  tmux      - Tmux session info")
	fmt.Fprintln(globals.Stdout, "  info      - Informational message")
	fmt.Fprintln(globals.Stdout, "  warning   - Warning message")
	fmt.Fprintln(globals.Stdout, "  trigger   - Trigger event notification")
	fmt.Fprintln(globals.Stdout, "  doctor    - System diagnostic report")
	fmt.Fprintln(globals.Stdout, "  app       - Installed app info")
	fmt.Fprintln(globals.Stdout, "")
	fmt.Fprintln(globals.Stdout, "Use --type to filter: xcw schema --type log,error")
}
