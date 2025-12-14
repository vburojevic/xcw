package tmux

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/GianlucaP106/gotmux/gotmux"
)

// Config holds tmux session configuration
type Config struct {
	SessionName    string // e.g., "xcw-iphone-15"
	SimulatorName  string // Original simulator name
	StartDirectory string // Working directory
	Detached       bool   // Run detached (default for AI agents)
}

// Manager handles all tmux session operations
type Manager struct {
	tmux    *gotmux.Tmux
	session *gotmux.Session
	pane    *gotmux.Pane
	config  *Config
	mu      sync.Mutex
}

// Errors
var (
	ErrTmuxNotInstalled   = fmt.Errorf("tmux is not installed")
	ErrNoSessionAvailable = fmt.Errorf("no tmux session available")
	ErrNoPaneAvailable    = fmt.Errorf("no tmux pane available")
)

// IsTmuxAvailable checks if tmux is installed
func IsTmuxAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// NewManager creates a new tmux manager instance
func NewManager(cfg *Config) (*Manager, error) {
	if !IsTmuxAvailable() {
		return nil, ErrTmuxNotInstalled
	}

	tmux, err := gotmux.DefaultTmux()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tmux: %w", err)
	}

	return &Manager{
		tmux:   tmux,
		config: cfg,
	}, nil
}

// GetOrCreateSession finds existing session or creates new one
func (m *Manager) GetOrCreateSession() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Try to find existing session
	sessions, err := m.tmux.ListSessions()
	if err == nil {
		for _, s := range sessions {
			if s.Name == m.config.SessionName {
				m.session = s
				return m.attachToExistingPane()
			}
		}
	}

	// Create new session
	return m.createNewSession()
}

func (m *Manager) createNewSession() error {
	session, err := m.tmux.NewSession(&gotmux.SessionOptions{
		Name:           m.config.SessionName,
		StartDirectory: m.config.StartDirectory,
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	m.session = session

	// Get the default window and pane
	windows, err := session.ListWindows()
	if err != nil {
		return fmt.Errorf("failed to list windows: %w", err)
	}

	if len(windows) > 0 {
		panes, err := windows[0].ListPanes()
		if err != nil {
			return fmt.Errorf("failed to list panes: %w", err)
		}
		if len(panes) > 0 {
			m.pane = panes[0]
		}
	}

	return nil
}

func (m *Manager) attachToExistingPane() error {
	windows, err := m.session.ListWindows()
	if err != nil {
		return err
	}

	if len(windows) > 0 {
		panes, err := windows[0].ListPanes()
		if err != nil {
			return err
		}
		if len(panes) > 0 {
			m.pane = panes[0]
		}
	}
	return nil
}

// SessionName returns the current session name
func (m *Manager) SessionName() string {
	return m.config.SessionName
}

// AttachCommand returns the command string for attaching to this session
func (m *Manager) AttachCommand() string {
	return fmt.Sprintf("tmux attach -t %s", m.config.SessionName)
}

// IsAttachable checks if the session can be attached to
func (m *Manager) IsAttachable() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.session == nil {
		return false
	}

	sessions, err := m.tmux.ListSessions()
	if err != nil {
		return false
	}

	for _, s := range sessions {
		if s.Name == m.config.SessionName {
			return true
		}
	}
	return false
}

// Cleanup cleans up internal references (session persists)
func (m *Manager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.session = nil
	m.pane = nil
}

// KillSession explicitly destroys the session
func (m *Manager) KillSession() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.session != nil {
		return m.session.Kill()
	}
	return nil
}

// GetPane returns the current pane
func (m *Manager) GetPane() *gotmux.Pane {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pane
}

// GenerateSessionName creates a tmux-safe session name from simulator name
func GenerateSessionName(simulatorName string) string {
	// Convert to lowercase
	name := strings.ToLower(simulatorName)

	// Replace spaces and special characters with hyphens
	re := regexp.MustCompile(`[^a-z0-9]+`)
	name = re.ReplaceAllString(name, "-")

	// Remove leading/trailing hyphens
	name = strings.Trim(name, "-")

	// Prefix with xcw
	return fmt.Sprintf("xcw-%s", name)
}
