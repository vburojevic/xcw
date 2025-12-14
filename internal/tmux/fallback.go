package tmux

import (
	"io"
	"os"
)

// OutputMode represents the output destination
type OutputMode int

const (
	OutputModeTmux   OutputMode = iota // Output to tmux pane
	OutputModeStdout                   // Output to stdout
)

// OutputManager provides fallback output handling
type OutputManager struct {
	mode     OutputMode
	tmux     *Manager
	writer   io.Writer
	flushErr error
}

// NewOutputManager creates an output manager with appropriate fallback
func NewOutputManager(preferTmux bool, tmuxConfig *Config) (*OutputManager, error) {
	om := &OutputManager{}

	// Try tmux first if preferred
	if preferTmux && IsTmuxAvailable() {
		mgr, err := NewManager(tmuxConfig)
		if err != nil {
			// Fall back to stdout
			om.mode = OutputModeStdout
			om.writer = os.Stdout
			return om, nil
		}

		if err := mgr.GetOrCreateSession(); err != nil {
			// Fall back to stdout
			om.mode = OutputModeStdout
			om.writer = os.Stdout
			return om, nil
		}

		om.mode = OutputModeTmux
		om.tmux = mgr
		om.writer = NewWriter(mgr)
		return om, nil
	}

	// Default: stdout
	om.mode = OutputModeStdout
	om.writer = os.Stdout
	return om, nil
}

// Writer returns the io.Writer for output
func (om *OutputManager) Writer() io.Writer {
	return om.writer
}

// Mode returns the current output mode
func (om *OutputManager) Mode() OutputMode {
	return om.mode
}

// TmuxManager returns the tmux manager if in tmux mode
func (om *OutputManager) TmuxManager() *Manager {
	return om.tmux
}

// IsTmuxMode returns true if outputting to tmux
func (om *OutputManager) IsTmuxMode() bool {
	return om.mode == OutputModeTmux
}

// AttachCommand returns the tmux attach command if in tmux mode
func (om *OutputManager) AttachCommand() string {
	if om.tmux != nil {
		return om.tmux.AttachCommand()
	}
	return ""
}

// SessionName returns the tmux session name if in tmux mode
func (om *OutputManager) SessionName() string {
	if om.tmux != nil {
		return om.tmux.SessionName()
	}
	return ""
}

// Cleanup cleans up resources
func (om *OutputManager) Cleanup() {
	if om.tmux != nil {
		// Flush any remaining output
		if w, ok := om.writer.(*Writer); ok {
			if err := w.Flush(); err != nil {
				om.flushErr = err
			}
		}
		// Clean up (session persists)
		om.tmux.Cleanup()
	}
}

// ModeString returns a human-readable description of the output mode
func (om *OutputManager) ModeString() string {
	switch om.mode {
	case OutputModeTmux:
		return "tmux session: " + om.SessionName()
	case OutputModeStdout:
		return "stdout"
	default:
		return "unknown"
	}
}
