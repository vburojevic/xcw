package cli

import (
	"errors"
	"fmt"

	"github.com/vburojevic/xcw/internal/output"
	"github.com/vburojevic/xcw/internal/tmux"
)

// ClearCmd clears a tmux session's content
type ClearCmd struct {
	Session string `required:"" help:"Tmux session name to clear (e.g., 'xcw-iphone-15')"`
}

// Run executes the clear command
func (c *ClearCmd) Run(globals *Globals) error {
	if !tmux.IsTmuxAvailable() {
		return c.outputError(globals, "TMUX_NOT_INSTALLED", "tmux is not installed")
	}

	cfg := &tmux.Config{
		SessionName: c.Session,
		Detached:    true,
	}

	manager, err := tmux.NewManager(cfg)
	if err != nil {
		return c.outputError(globals, "TMUX_ERROR", err.Error())
	}

	if err := manager.GetOrCreateSession(); err != nil {
		return c.outputError(globals, "SESSION_NOT_FOUND", fmt.Sprintf("session %s not found: %v", c.Session, err))
	}

	if err := manager.ClearPaneWithBanner("Session cleared"); err != nil {
		return c.outputError(globals, "CLEAR_FAILED", fmt.Sprintf("failed to clear session: %v", err))
	}

	// Output success
	if globals.Format == "ndjson" {
		output.NewNDJSONWriter(globals.Stdout).WriteInfo(
			fmt.Sprintf("Session %s cleared", c.Session), "", "", "", "")
	} else {
		fmt.Fprintf(globals.Stdout, "Session %s cleared\n", c.Session)
	}

	return nil
}

func (c *ClearCmd) outputError(globals *Globals, code, message string) error {
	if globals.Format == "ndjson" {
		w := output.NewNDJSONWriter(globals.Stdout)
		w.WriteError(code, message)
	} else {
		fmt.Fprintf(globals.Stderr, "Error [%s]: %s\n", code, message)
	}
	return errors.New(message)
}
