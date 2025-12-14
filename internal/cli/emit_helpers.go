package cli

import (
	"fmt"

	"github.com/vburojevic/xcw/internal/output"
)

// emitWarning respects format/quiet.
func emitWarning(globals *Globals, emitter *output.Emitter, msg string) {
	if globals == nil || globals.Quiet {
		return
	}
	if globals.Format == "ndjson" && emitter != nil {
		if err := emitter.WriteWarning(msg); err != nil {
			globals.Debug("failed to emit warning: %v", err)
		}
		return
	}
	if _, err := fmt.Fprintf(globals.Stderr, "Warning: %s\n", msg); err != nil {
		globals.Debug("failed to write warning: %v", err)
	}
}
