package cli

import (
	"encoding/json"

	"github.com/vburojevic/xcw/internal/output"
)

// LogSchemaCmd outputs minimal log schema docs for agents
type LogSchemaCmd struct{}

type logSchemaDoc struct {
	Type          string                 `json:"type"`
	SchemaVersion int                    `json:"schemaVersion"`
	Fields        map[string]string      `json:"fields"`
	Example       map[string]interface{} `json:"example"`
}

func (c *LogSchemaCmd) Run(globals *Globals) error {
	doc := logSchemaDoc{
		Type:          "log_schema",
		SchemaVersion: output.SchemaVersion,
		Fields: map[string]string{
			"timestamp": "ISO8601 UTC",
			"level":     "Debug|Info|Default|Error|Fault",
			"process":   "Process name",
			"pid":       "Process ID",
			"subsystem": "Subsystem (bundle id)",
			"category":  "Category",
			"message":   "Log message",
			"session":   "Session number",
			"tail_id":   "Tail invocation identifier",
		},
		Example: map[string]interface{}{
			"type":          "log",
			"schemaVersion": output.SchemaVersion,
			"timestamp":     "2025-12-11T10:30:45.123Z",
			"level":         "Error",
			"process":       "MyApp",
			"pid":           1234,
			"subsystem":     "com.example.myapp",
			"category":      "network",
			"message":       "Connection failed",
			"session":       2,
			"tail_id":       "tail-abc",
		},
	}

	enc := json.NewEncoder(globals.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}
