package cli

import (
	"errors"
	"fmt"

	"github.com/vburojevic/xcw/internal/output"
)

// outputErrorCommon normalizes error emission across commands, respecting
// ndjson vs text formats so AI agents always get machine-readable failures.
func outputErrorCommon(globals *Globals, code, message string, hint ...string) error {
	if globals != nil && globals.Format == "ndjson" {
		if err := output.NewNDJSONWriter(globals.Stdout).WriteError(code, message, hint...); err != nil {
			return err
		}
	} else if globals != nil {
		if _, err := fmt.Fprintf(globals.Stderr, "Error [%s]: %s", code, message); err != nil {
			return err
		}
		if len(hint) > 0 && hint[0] != "" {
			if _, err := fmt.Fprintf(globals.Stderr, " (hint: %s)", hint[0]); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(globals.Stderr); err != nil {
			return err
		}
	}
	return errors.New(message)
}
