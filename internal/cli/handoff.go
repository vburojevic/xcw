package cli

import (
	"encoding/json"
	"time"

	"github.com/vburojevic/xcw/internal/output"
)

// HandoffCmd emits a compact JSON blob for AI agents to transfer context.
type HandoffCmd struct{}

type handoffPayload struct {
	Type            string   `json:"type"`
	Version         string   `json:"version"`
	SchemaVersion   int      `json:"schemaVersion"`
	Timestamp       string   `json:"timestamp"`
	ContractVersion int      `json:"contract_version"`
	Hints           []string `json:"hints"`
}

func (c *HandoffCmd) Run(globals *Globals) error {
	payload := handoffPayload{
		Type:            "handoff",
		Version:         Version,
		SchemaVersion:   output.SchemaVersion,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		ContractVersion: 1,
		Hints:           defaultHints(),
	}
	enc := json.NewEncoder(globals.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}
