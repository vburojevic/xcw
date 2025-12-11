package cli

import (
	"errors"
	"fmt"

	"github.com/vburojevic/xcw/internal/output"
)

// outputErrorCommon normalizes error emission across commands, respecting
// ndjson vs text formats so AI agents always get machine-readable failures.
func outputErrorCommon(globals *Globals, code, message string) error {
	if globals != nil && globals.Format == "ndjson" {
		output.NewNDJSONWriter(globals.Stdout).WriteError(code, message)
	} else if globals != nil {
		fmt.Fprintf(globals.Stderr, "Error [%s]: %s\n", code, message)
	}
	return errors.New(message)
}
