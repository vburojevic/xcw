package cli

import (
	"encoding/json"
	"strings"

	"github.com/vburojevic/xcw/internal/output"
)

// SchemaCmd outputs JSON Schema for xcw output types
type SchemaCmd struct {
	Type      []string `short:"t" help:"Output types to include (log,summary,analysis,heartbeat,stats,metadata,ready,session_start,session_end,clear_buffer,agent_hints,cutoff_reached,reconnect_notice,error,rotation,console,discovery,simulator,tmux,info,warning,trigger,trigger_error,doctor,app,apps_summary,pick,update,config,config_path,session,session_debug). Default: all"`
	Changelog bool     `help:"Output schema changelog instead of full schema"`
}

// Run executes the schema command
func (c *SchemaCmd) Run(globals *Globals) error {
	if c.Changelog {
		entries := []map[string]interface{}{
			{
				"version":   output.SchemaVersion,
				"date":      "2025-12-11",
				"changes":   []string{"Added agent_hints, clear_buffer, tail_id propagation, contract fields on ready/heartbeat"},
				"breaking":  false,
				"contracts": []string{"Agents must match tail_id and latest session; reset on clear_buffer/session boundaries"},
			},
		}
		enc := json.NewEncoder(globals.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	}

	schemas := map[string]interface{}{
		"log":              logSchema(),
		"summary":          summarySchema(),
		"analysis":         analysisSchema(),
		"heartbeat":        heartbeatSchema(),
		"stats":            statsSchema(),
		"metadata":         metadataSchema(),
		"ready":            readySchema(),
		"session_start":    sessionStartSchema(),
		"session_end":      sessionEndSchema(),
		"clear_buffer":     clearBufferSchema(),
		"agent_hints":      agentHintsSchema(),
		"cutoff_reached":   cutoffSchema(),
		"reconnect_notice": reconnectSchema(),
		"error":            errorSchema(),
		"rotation":         rotationSchema(),
		"console":          consoleSchema(),
		"discovery":        discoverySchema(),
		"simulator":        simulatorSchema(),
		"tmux":             tmuxSchema(),
		"info":             infoSchema(),
		"warning":          warningSchema(),
		"trigger":          triggerSchema(),
		"trigger_error":    triggerErrorSchema(),
		"doctor":           doctorSchema(),
		"app":              appSchema(),
		"apps_summary":     appsSummarySchema(),
		"pick":             pickSchema(),
		"update":           updateSchema(),
		"config":           configSchema(),
		"config_path":      configPathSchema(),
		"session":          sessionSchema(),
		"session_debug":    sessionDebugSchema(),
	}

	// Determine which schemas to output
	typesToOutput := c.Type
	if len(typesToOutput) == 0 {
		typesToOutput = []string{
			"log",
			"summary",
			"analysis",
			"heartbeat",
			"stats",
			"metadata",
			"ready",
			"session_start",
			"session_end",
			"clear_buffer",
			"agent_hints",
			"cutoff_reached",
			"reconnect_notice",
			"error",
			"rotation",
			"console",
			"discovery",
			"simulator",
			"tmux",
			"info",
			"warning",
			"trigger",
			"trigger_error",
			"doctor",
			"app",
			"apps_summary",
			"pick",
			"update",
			"config",
			"config_path",
			"session",
			"session_debug",
		}
	}

	// Build output
	schemaOutput := map[string]interface{}{
		"$schema":       "http://json-schema.org/draft-07/schema#",
		"title":         "XcodeConsoleWatcher Output Schemas",
		"description":   "JSON Schema definitions for all xcw NDJSON output types",
		"schemaVersion": output.SchemaVersion,
		"definitions":   map[string]interface{}{},
	}

	defs := schemaOutput["definitions"].(map[string]interface{})
	for _, t := range typesToOutput {
		t = strings.ToLower(strings.TrimSpace(t))
		if schema, ok := schemas[t]; ok {
			defs[t] = schema
		}
	}

	// Output as JSON
	encoder := json.NewEncoder(globals.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(schemaOutput)
}

// schemaVersionProperty returns the schemaVersion property definition
func schemaVersionProperty() map[string]interface{} {
	return map[string]interface{}{
		"type":        "integer",
		"const":       output.SchemaVersion,
		"description": "Schema version for compatibility detection",
	}
}

func logSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Log Entry",
		"description": "A single log entry from the iOS Simulator",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "log",
			},
			"schemaVersion": schemaVersionProperty(),
			"tail_id": map[string]interface{}{
				"type":        "string",
				"description": "Tail invocation ID",
			},
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
			"session": map[string]interface{}{
				"type":        "integer",
				"description": "Session number (1, 2, 3...) when session tracking is active",
			},
		},
		"required": []string{"type", "schemaVersion", "timestamp", "level", "process", "pid", "message"},
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
			"schemaVersion": schemaVersionProperty(),
			"tail_id": map[string]interface{}{
				"type":        "string",
				"description": "Tail invocation identifier",
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
		"required": []string{"type", "schemaVersion", "totalCount"},
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
			"schemaVersion": schemaVersionProperty(),
			"tail_id": map[string]interface{}{
				"type":        "string",
				"description": "Tail invocation identifier",
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
			"contract_version": map[string]interface{}{
				"type":        "integer",
				"description": "Agent contract version for heartbeat semantics",
			},
			"latest_session": map[string]interface{}{
				"type":        "integer",
				"description": "Latest session number observed",
			},
			"last_seen_timestamp": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "Timestamp of the most recently emitted log entry",
			},
		},
		"required": []string{"type", "schemaVersion", "timestamp", "uptime_seconds", "logs_since_last"},
	}
}

func rotationSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Rotation",
		"description": "File rotation notice indicating active output file path",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "rotation",
			},
			"schemaVersion": schemaVersionProperty(),
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the rotated output file",
			},
			"tail_id": map[string]interface{}{
				"type":        "string",
				"description": "Tail invocation ID",
			},
			"session": map[string]interface{}{
				"type":        "integer",
				"description": "Session number for this file",
			},
		},
		"required": []string{"type", "schemaVersion", "path"},
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
			"schemaVersion": schemaVersionProperty(),
			"code": map[string]interface{}{
				"type":        "string",
				"description": "Machine-readable error code for programmatic handling",
				"enum": []string{
					"DEVICE_NOT_FOUND",
					"NO_BOOTED_SIMULATOR",
					"INVALID_FLAGS",
					"INVALID_PATTERN",
					"INVALID_EXCLUDE_PATTERN",
					"INVALID_DURATION",
					"INVALID_UNTIL",
					"INVALID_INTERVAL",
					"INVALID_HEARTBEAT",
					"INVALID_COOLDOWN",
					"INVALID_TRIGGER",
					"INVALID_TRIGGER_PATTERN",
					"INVALID_TRIGGER_TIMEOUT",
					"STREAM_FAILED",
					"QUERY_FAILED",
					"LIST_FAILED",
					"LIST_APPS_FAILED",
					"FILE_NOT_FOUND",
					"FILE_CREATE_ERROR",
					"READ_ERROR",
					"NO_ENTRIES",
					"DEVICE_NOT_BOOTED",
					"TMUX_NOT_INSTALLED",
					"TMUX_ERROR",
					"SESSION_NOT_FOUND",
					"SESSION_DIR_ERROR",
					"SESSION_ERROR",
					"LIST_SESSIONS_ERROR",
					"INVALID_INDEX",
					"NO_SESSIONS",
					"CLEAN_ERROR",
					"CLEAR_FAILED",
					"NOT_INTERACTIVE",
					"NO_SIMULATORS",
					"NO_APPS",
				},
			},
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Human-readable error description",
			},
			"hint": map[string]interface{}{
				"type":        "string",
				"description": "Optional recovery hint for agents",
			},
		},
		"required": []string{"type", "schemaVersion", "code", "message"},
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
			"schemaVersion": schemaVersionProperty(),
			"session": map[string]interface{}{
				"type":        "string",
				"description": "Tmux session name",
			},
			"attach": map[string]interface{}{
				"type":        "string",
				"description": "Command to attach to the session",
			},
		},
		"required": []string{"type", "schemaVersion", "session", "attach"},
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
			"schemaVersion": schemaVersionProperty(),
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
		"required": []string{"type", "schemaVersion", "message"},
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
			"schemaVersion": schemaVersionProperty(),
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Warning message content",
			},
		},
		"required": []string{"type", "schemaVersion", "message"},
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
			"schemaVersion": schemaVersionProperty(),
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
		"required": []string{"type", "schemaVersion", "trigger_type", "command"},
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
			"schemaVersion": schemaVersionProperty(),
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
		"required": []string{"type", "schemaVersion", "timestamp", "checks", "all_passed"},
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
			"schemaVersion": schemaVersionProperty(),
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
		"required": []string{"type", "schemaVersion", "bundle_id", "name", "app_type"},
	}
}

func pickSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Pick Result",
		"description": "Result from interactive simulator or app selection",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "pick",
			},
			"schemaVersion": schemaVersionProperty(),
			"picked": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"simulator", "app"},
				"description": "What was picked: simulator or app",
			},
			"udid": map[string]interface{}{
				"type":        "string",
				"description": "Simulator UDID (when picking simulator)",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Display name of the selected item",
			},
			"bundle_id": map[string]interface{}{
				"type":        "string",
				"description": "App bundle identifier (when picking app)",
			},
		},
		"required": []string{"type", "schemaVersion", "picked", "name"},
	}
}

func updateSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Update Instructions",
		"description": "Instructions for upgrading xcw",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "update",
			},
			"schemaVersion": schemaVersionProperty(),
			"current_version": map[string]interface{}{
				"type":        "string",
				"description": "Currently installed version",
			},
			"commit": map[string]interface{}{
				"type":        "string",
				"description": "Git commit hash of current version",
			},
			"homebrew": map[string]interface{}{
				"type":        "string",
				"description": "Command to upgrade via Homebrew",
			},
			"go_install": map[string]interface{}{
				"type":        "string",
				"description": "Command to upgrade via Go install",
			},
			"releases_url": map[string]interface{}{
				"type":        "string",
				"format":      "uri",
				"description": "URL to release notes",
			},
		},
		"required": []string{"type", "schemaVersion", "current_version", "homebrew", "go_install", "releases_url"},
	}
}

func sessionSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Session File",
		"description": "Information about a session log file",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "session",
			},
			"schemaVersion": schemaVersionProperty(),
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Full path to the session file",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Session filename",
			},
			"timestamp": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "When the session was created",
			},
			"size": map[string]interface{}{
				"type":        "integer",
				"description": "File size in bytes",
			},
			"prefix": map[string]interface{}{
				"type":        "string",
				"description": "Session prefix (usually app bundle ID)",
			},
		},
		"required": []string{"type", "schemaVersion", "path", "name", "timestamp", "size"},
	}
}

func sessionDebugSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Session Debug",
		"description": "Verbose session transition event for diagnostics",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "session_debug",
			},
			"schemaVersion": schemaVersionProperty(),
			"tail_id": map[string]interface{}{
				"type":        "string",
				"description": "Tail invocation identifier",
			},
			"session": map[string]interface{}{
				"type":        "integer",
				"description": "Current session number",
			},
			"prev_session": map[string]interface{}{
				"type":        "integer",
				"description": "Previous session number",
			},
			"pid": map[string]interface{}{
				"type":        "integer",
				"description": "Current PID",
			},
			"prev_pid": map[string]interface{}{
				"type":        "integer",
				"description": "Previous PID",
			},
			"reason": map[string]interface{}{
				"type":        "string",
				"description": "Reason for transition (relaunch, idle_timeout)",
			},
			"summary": map[string]interface{}{
				"type":        "object",
				"description": "Previous session summary snapshot",
			},
		},
		"required": []string{"type", "schemaVersion", "session", "pid", "reason"},
	}
}

func analysisSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Analysis",
		"description": "Analyzer output containing a summary and detected patterns",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "analysis",
			},
			"schemaVersion": schemaVersionProperty(),
			"timestamp": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "When the analysis was produced",
			},
			"summary": map[string]interface{}{
				"type":        "object",
				"description": "Log summary object",
			},
			"patterns": map[string]interface{}{
				"type":        "array",
				"description": "Detected error/fault patterns",
				"items": map[string]interface{}{
					"type": "object",
				},
			},
			"new_pattern_count": map[string]interface{}{
				"type":        "integer",
				"description": "Number of newly observed patterns (when persistence enabled)",
			},
			"known_pattern_count": map[string]interface{}{
				"type":        "integer",
				"description": "Number of previously known patterns (when persistence enabled)",
			},
		},
		"required": []string{"type", "schemaVersion", "timestamp", "summary"},
	}
}

func statsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Stats",
		"description": "Periodic stream diagnostics emitted alongside heartbeats",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "stats",
			},
			"schemaVersion": schemaVersionProperty(),
			"timestamp": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "When the stats snapshot was taken",
			},
			"tail_id": map[string]interface{}{
				"type":        "string",
				"description": "Tail invocation identifier",
			},
			"session": map[string]interface{}{
				"type":        "integer",
				"description": "Current session number",
			},
			"reconnects": map[string]interface{}{
				"type":        "integer",
				"description": "Number of reconnects since tail start",
			},
			"parse_drops": map[string]interface{}{
				"type":        "integer",
				"description": "Number of NDJSON parse drops",
			},
			"timestamp_parse_drops": map[string]interface{}{
				"type":        "integer",
				"description": "Number of timestamp parse drops",
			},
			"channel_drops": map[string]interface{}{
				"type":        "integer",
				"description": "Number of dropped log entries due to backpressure",
			},
			"buffered": map[string]interface{}{
				"type":        "integer",
				"description": "Approximate number of buffered log entries",
			},
			"last_seen_timestamp": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "Timestamp of the most recently emitted log entry",
			},
		},
		"required": []string{"type", "schemaVersion", "timestamp"},
	}
}

func metadataSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Metadata",
		"description": "Tool metadata emitted at tail start for agents",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "metadata",
			},
			"schemaVersion": schemaVersionProperty(),
			"version": map[string]interface{}{
				"type":        "string",
				"description": "xcw version string",
			},
			"commit": map[string]interface{}{
				"type":        "string",
				"description": "git commit hash (short)",
			},
			"build_date": map[string]interface{}{
				"type":        "string",
				"description": "Build date (optional)",
			},
			"contract_version": map[string]interface{}{
				"type":        "integer",
				"description": "Agent contract version for stream semantics",
			},
		},
		"required": []string{"type", "schemaVersion", "version", "commit", "contract_version"},
	}
}

func readySchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Ready",
		"description": "Signals that log capture is active and ready",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "ready",
			},
			"schemaVersion": schemaVersionProperty(),
			"timestamp": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "When capture became active",
			},
			"simulator": map[string]interface{}{
				"type":        "string",
				"description": "Simulator display name",
			},
			"udid": map[string]interface{}{
				"type":        "string",
				"description": "Simulator UDID",
			},
			"app": map[string]interface{}{
				"type":        "string",
				"description": "App bundle identifier (when filtering by app)",
			},
			"tail_id": map[string]interface{}{
				"type":        "string",
				"description": "Tail invocation identifier",
			},
			"session": map[string]interface{}{
				"type":        "integer",
				"description": "Current session number",
			},
			"contract_version": map[string]interface{}{
				"type":        "integer",
				"description": "Agent contract version for ready semantics",
			},
		},
		"required": []string{"type", "schemaVersion", "timestamp", "simulator", "udid", "app"},
	}
}

func sessionStartSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Session Start",
		"description": "Emitted when a new app session begins (PID change detected)",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "session_start",
			},
			"schemaVersion": schemaVersionProperty(),
			"alert": map[string]interface{}{
				"type":        "string",
				"description": "Alert string (eg. APP_RELAUNCHED) when previous session existed",
			},
			"session": map[string]interface{}{
				"type":        "integer",
				"description": "Session number (1, 2, 3...)",
			},
			"pid": map[string]interface{}{
				"type":        "integer",
				"description": "Current process ID",
			},
			"previous_pid": map[string]interface{}{
				"type":        "integer",
				"description": "Previous process ID (if app relaunched)",
			},
			"app": map[string]interface{}{
				"type":        "string",
				"description": "App bundle identifier",
			},
			"simulator": map[string]interface{}{
				"type":        "string",
				"description": "Simulator name",
			},
			"udid": map[string]interface{}{
				"type":        "string",
				"description": "Simulator UDID",
			},
			"timestamp": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "When the session started",
			},
			"tail_id": map[string]interface{}{
				"type":        "string",
				"description": "Tail invocation identifier",
			},
			"version": map[string]interface{}{
				"type":        "string",
				"description": "App version (CFBundleShortVersionString)",
			},
			"build": map[string]interface{}{
				"type":        "string",
				"description": "App build number (CFBundleVersion)",
			},
			"binary_uuid": map[string]interface{}{
				"type":        "string",
				"description": "Mach-O UUID from process image",
			},
		},
		"required": []string{"type", "schemaVersion", "session", "pid", "app", "simulator", "udid", "timestamp"},
	}
}

func sessionEndSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Session End",
		"description": "Emitted when an app session ends (PID changes or stream stops)",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "session_end",
			},
			"schemaVersion": schemaVersionProperty(),
			"session": map[string]interface{}{
				"type":        "integer",
				"description": "Session number that ended",
			},
			"pid": map[string]interface{}{
				"type":        "integer",
				"description": "Process ID that ended",
			},
			"tail_id": map[string]interface{}{
				"type":        "string",
				"description": "Tail invocation identifier",
			},
			"summary": map[string]interface{}{
				"type":        "object",
				"description": "Summary of the ended session",
				"properties": map[string]interface{}{
					"total_logs": map[string]interface{}{
						"type": "integer",
					},
					"errors": map[string]interface{}{
						"type": "integer",
					},
					"faults": map[string]interface{}{
						"type": "integer",
					},
					"duration_seconds": map[string]interface{}{
						"type": "integer",
					},
				},
			},
		},
		"required": []string{"type", "schemaVersion", "session", "pid", "summary"},
	}
}

func clearBufferSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Clear Buffer",
		"description": "Instructs consumers to reset caches at a session boundary",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "clear_buffer",
			},
			"schemaVersion": schemaVersionProperty(),
			"reason": map[string]interface{}{
				"type":        "string",
				"description": "Reason for buffer reset (session_start, session_end, idle_timeout, etc.)",
			},
			"tail_id": map[string]interface{}{
				"type":        "string",
				"description": "Tail invocation identifier",
			},
			"session": map[string]interface{}{
				"type":        "integer",
				"description": "Session number",
			},
			"hints": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Optional recovery hints for consumers",
			},
		},
		"required": []string{"type", "schemaVersion", "reason"},
	}
}

func agentHintsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Agent Hints",
		"description": "Runtime contract guidance for AI agents consuming NDJSON",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "agent_hints",
			},
			"schemaVersion": schemaVersionProperty(),
			"tail_id": map[string]interface{}{
				"type":        "string",
				"description": "Tail invocation identifier",
			},
			"session": map[string]interface{}{
				"type":        "integer",
				"description": "Current session number",
			},
			"contract_version": map[string]interface{}{
				"type":        "integer",
				"description": "Agent contract version",
			},
			"hints": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "List of runtime hints for agents",
			},
			"recommended_scope": map[string]interface{}{
				"type":        "string",
				"description": "Recommended correlation scope (eg. tail_id + latest session only)",
			},
		},
		"required": []string{"type", "schemaVersion", "contract_version", "hints"},
	}
}

func cutoffSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Cutoff Reached",
		"description": "Emitted when max-duration or max-logs cutoff stops streaming",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "cutoff_reached",
			},
			"schemaVersion": schemaVersionProperty(),
			"reason": map[string]interface{}{
				"type":        "string",
				"description": "Cutoff reason (max_duration, max_logs, sigint, etc.)",
			},
			"tail_id": map[string]interface{}{
				"type":        "string",
				"description": "Tail invocation identifier",
			},
			"session": map[string]interface{}{
				"type":        "integer",
				"description": "Session number at cutoff time",
			},
			"total_logs": map[string]interface{}{
				"type":        "integer",
				"description": "Total logs emitted before cutoff",
			},
		},
		"required": []string{"type", "schemaVersion", "reason"},
	}
}

func reconnectSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Reconnect Notice",
		"description": "Signals that the log stream reconnected; consumers should consider potential gaps",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "reconnect_notice",
			},
			"schemaVersion": schemaVersionProperty(),
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Reconnect message",
			},
			"tail_id": map[string]interface{}{
				"type":        "string",
				"description": "Tail invocation identifier",
			},
			"severity": map[string]interface{}{
				"type":        "string",
				"description": "Severity label (info, warn, error)",
			},
		},
		"required": []string{"type", "schemaVersion", "message"},
	}
}

func consoleSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Console Output",
		"description": "A line of stdout/stderr from `xcw launch`",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "console",
			},
			"schemaVersion": schemaVersionProperty(),
			"timestamp": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "When the line was emitted",
			},
			"stream": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"stdout", "stderr"},
				"description": "Which stream produced the line",
			},
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Console line",
			},
			"process": map[string]interface{}{
				"type":        "string",
				"description": "App bundle ID (when known)",
			},
			"pid": map[string]interface{}{
				"type":        "integer",
				"description": "PID (when available)",
			},
		},
		"required": []string{"type", "schemaVersion", "timestamp", "stream", "message"},
	}
}

func discoverySchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Discovery",
		"description": "Discovery results showing subsystems, categories, processes, and levels",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "discovery",
			},
			"schemaVersion": schemaVersionProperty(),
			"app": map[string]interface{}{
				"type":        "string",
				"description": "App bundle identifier (when filtering by app)",
			},
			"time_range": map[string]interface{}{
				"type":        "object",
				"description": "Start/end time range",
				"properties": map[string]interface{}{
					"start": map[string]interface{}{
						"type":   "string",
						"format": "date-time",
					},
					"end": map[string]interface{}{
						"type":   "string",
						"format": "date-time",
					},
				},
			},
			"total_count": map[string]interface{}{
				"type":        "integer",
				"description": "Total number of logs analyzed",
			},
			"subsystems": map[string]interface{}{
				"type":        "array",
				"description": "Subsystem aggregates",
				"items": map[string]interface{}{
					"type": "object",
				},
			},
			"categories": map[string]interface{}{
				"type":        "array",
				"description": "Category aggregates",
				"items": map[string]interface{}{
					"type": "object",
				},
			},
			"processes": map[string]interface{}{
				"type":        "array",
				"description": "Process aggregates",
				"items": map[string]interface{}{
					"type": "object",
				},
			},
			"levels": map[string]interface{}{
				"type":        "object",
				"description": "Level histogram",
			},
		},
		"required": []string{"type", "schemaVersion", "time_range", "total_count"},
	}
}

func simulatorSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Simulator",
		"description": "Simulator device information",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "simulator",
			},
			"schemaVersion": schemaVersionProperty(),
			"udid": map[string]interface{}{
				"type":        "string",
				"description": "Simulator UDID",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Simulator name",
			},
			"state": map[string]interface{}{
				"type":        "string",
				"description": "Current simulator state",
			},
			"isAvailable": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether the device is available",
			},
			"deviceTypeIdentifier": map[string]interface{}{
				"type":        "string",
				"description": "Device type identifier",
			},
			"runtime": map[string]interface{}{
				"type":        "string",
				"description": "Runtime identifier (eg. iOS 18.0)",
			},
			"dataPath": map[string]interface{}{
				"type":        "string",
				"description": "Simulator data path (optional)",
			},
			"logPath": map[string]interface{}{
				"type":        "string",
				"description": "Simulator log path (optional)",
			},
			"lastBootedAt": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "Most recent boot time (optional)",
			},
		},
		"required": []string{"type", "schemaVersion", "udid", "name", "state", "runtime"},
	}
}

func triggerErrorSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Trigger Error",
		"description": "Error emitted when a watch trigger command fails",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "trigger_error",
			},
			"schemaVersion": schemaVersionProperty(),
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Trigger command",
			},
			"error": map[string]interface{}{
				"type":        "string",
				"description": "Error message",
			},
		},
		"required": []string{"type", "schemaVersion", "command", "error"},
	}
}

func appsSummarySchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Apps Summary",
		"description": "Summary row emitted after listing apps",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "apps_summary",
			},
			"schemaVersion": schemaVersionProperty(),
			"device": map[string]interface{}{
				"type":        "string",
				"description": "Simulator name",
			},
			"udid": map[string]interface{}{
				"type":        "string",
				"description": "Simulator UDID",
			},
			"total": map[string]interface{}{
				"type":        "integer",
				"description": "Total number of apps emitted",
			},
		},
		"required": []string{"type", "schemaVersion", "total"},
	}
}

func configSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Config",
		"description": "Effective configuration and provenance information",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "config",
			},
			"schemaVersion": schemaVersionProperty(),
			"config_file": map[string]interface{}{
				"type":        "string",
				"description": "Config file path used (if any)",
			},
			"format": map[string]interface{}{
				"type":        "string",
				"description": "Effective output format",
			},
			"level": map[string]interface{}{
				"type":        "string",
				"description": "Effective minimum log level",
			},
			"quiet": map[string]interface{}{
				"type":        "boolean",
				"description": "Effective quiet mode",
			},
			"verbose": map[string]interface{}{
				"type":        "boolean",
				"description": "Effective verbose mode",
			},
			"defaults": map[string]interface{}{
				"type":        "object",
				"description": "Global defaults section",
			},
			"tail": map[string]interface{}{
				"type":        "object",
				"description": "Tail defaults section",
			},
			"query": map[string]interface{}{
				"type":        "object",
				"description": "Query defaults section",
			},
			"watch": map[string]interface{}{
				"type":        "object",
				"description": "Watch defaults section",
			},
			"sources": map[string]interface{}{
				"type":        "object",
				"description": "Per-key provenance map: flag|env|config|default",
			},
		},
		"required": []string{"type", "schemaVersion"},
	}
}

func configPathSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"title":       "Config Path",
		"description": "Config file path resolution result",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":  "string",
				"const": "config_path",
			},
			"schemaVersion": schemaVersionProperty(),
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Config file path (empty if not found)",
			},
		},
		"required": []string{"type", "schemaVersion", "path"},
	}
}
