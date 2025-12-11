package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vburojevic/xcw/internal/config"
	"github.com/vburojevic/xcw/internal/output"
)

func TestHandoffCmd(t *testing.T) {
	var buf bytes.Buffer
	globals := &Globals{
		Format:  "ndjson",
		Stdout:  &buf,
		Stderr:  &buf,
		Config:  config.Default(),
		Verbose: false,
	}

	cmd := &HandoffCmd{}
	require.NoError(t, cmd.Run(globals))

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &m))
	require.Equal(t, "handoff", m["type"])
	require.Equal(t, float64(output.SchemaVersion), m["schemaVersion"])
	require.Equal(t, float64(1), m["contract_version"])
	require.NotEmpty(t, m["hints"])
}
