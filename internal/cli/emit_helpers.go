package cli

import (
	"fmt"

	"github.com/vburojevic/xcw/internal/output"
)

// emitWarning respects format/quiet.
func emitWarning(globals *Globals, emitter *output.Emitter, msg string) {
	if globals.Quiet {
		return
	}
	if globals.Format == "ndjson" && emitter != nil {
		emitter.WriteWarning(msg)
		return
	}
	fmt.Fprintf(globals.Stderr, "Warning: %s\n", msg)
}

// emitError respects format/quiet but always returns an error.
func emitError(globals *Globals, emitter *output.Emitter, code, msg string) error {
	if globals.Format == "ndjson" && emitter != nil {
		emitter.Error(code, msg)
		return fmt.Errorf(msg)
	}
	fmt.Fprintf(globals.Stderr, "Error [%s]: %s\n", code, msg)
	return fmt.Errorf(msg)
}
