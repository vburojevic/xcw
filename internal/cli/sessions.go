package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/vburojevic/xcw/internal/output"
)

// SessionsCmd manages session log files
type SessionsCmd struct {
	List  SessionsListCmd  `cmd:"" default:"1" help:"List session files"`
	Show  SessionsShowCmd  `cmd:"" help:"Show path to a session file"`
	Clean SessionsCleanCmd `cmd:"" help:"Delete old session files"`
}

// SessionsListCmd lists session files
type SessionsListCmd struct {
	Dir   string `help:"Session directory (default: ~/.xcw/sessions)"`
	Limit int    `default:"20" help:"Max sessions to show"`
}

// Run executes the sessions list command
func (c *SessionsListCmd) Run(globals *Globals) error {
	sessions, err := ListSessions(c.Dir)
	if err != nil {
		return c.outputError(globals, "LIST_SESSIONS_ERROR", err.Error())
	}

	if len(sessions) == 0 {
		if globals.Format == "ndjson" {
			output.NewNDJSONWriter(globals.Stdout).WriteInfo("No session files found", "", "", "", "")
		} else {
			fmt.Fprintln(globals.Stdout, "No session files found")
			dir := c.Dir
			if dir == "" {
				dir = GetDefaultSessionDir()
			}
			fmt.Fprintf(globals.Stdout, "Session directory: %s\n", dir)
		}
		return nil
	}

	// Limit output
	if c.Limit > 0 && len(sessions) > c.Limit {
		sessions = sessions[:c.Limit]
	}

	if globals.Format == "ndjson" {
		for _, s := range sessions {
			so := SessionOutput{
				Type:          "session",
				SchemaVersion: output.SchemaVersion,
				Path:          s.Path,
				Name:          s.Name,
				Timestamp:     s.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
				Size:          s.Size,
				Prefix:        s.Prefix,
			}
			data, _ := json.Marshal(so)
			fmt.Fprintln(globals.Stdout, string(data))
		}
	} else {
		fmt.Fprintf(globals.Stdout, "Session files (%d):\n", len(sessions))
		for i, s := range sessions {
			sizeStr := formatSize(s.Size)
			fmt.Fprintf(globals.Stdout, "  [%d] %s  %s  %s\n",
				i+1,
				s.Timestamp.Format("2006-01-02 15:04:05"),
				sizeStr,
				s.Name)
		}
		dir := c.Dir
		if dir == "" {
			dir = GetDefaultSessionDir()
		}
		fmt.Fprintf(globals.Stdout, "\nDirectory: %s\n", dir)
	}

	return nil
}

func (c *SessionsListCmd) outputError(globals *Globals, code, message string) error {
	if globals.Format == "ndjson" {
		output.NewNDJSONWriter(globals.Stdout).WriteError(code, message)
	} else {
		fmt.Fprintf(globals.Stderr, "Error [%s]: %s\n", code, message)
	}
	return errors.New(message)
}

// SessionsShowCmd shows the path to a session file
type SessionsShowCmd struct {
	Index  int    `arg:"" optional:"" help:"Session index from list (1-based)"`
	Dir    string `help:"Session directory (default: ~/.xcw/sessions)"`
	Latest bool   `help:"Show most recent session"`
}

// Run executes the sessions show command
func (c *SessionsShowCmd) Run(globals *Globals) error {
	var session *SessionFile

	if c.Latest || c.Index == 0 {
		// Get latest session
		s, err := LatestSession(c.Dir)
		if err != nil {
			return c.outputError(globals, "SESSION_ERROR", err.Error())
		}
		if s == nil {
			return c.outputError(globals, "NO_SESSIONS", "no session files found")
		}
		session = s
	} else {
		// Get by index
		sessions, err := ListSessions(c.Dir)
		if err != nil {
			return c.outputError(globals, "LIST_SESSIONS_ERROR", err.Error())
		}
		if len(sessions) == 0 {
			return c.outputError(globals, "NO_SESSIONS", "no session files found")
		}
		if c.Index < 1 || c.Index > len(sessions) {
			return c.outputError(globals, "INVALID_INDEX",
				fmt.Sprintf("index %d out of range (1-%d)", c.Index, len(sessions)))
		}
		session = &sessions[c.Index-1]
	}

	if globals.Format == "ndjson" {
		so := SessionOutput{
			Type:          "session",
			SchemaVersion: output.SchemaVersion,
			Path:          session.Path,
			Name:          session.Name,
			Timestamp:     session.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
			Size:          session.Size,
			Prefix:        session.Prefix,
		}
		data, _ := json.Marshal(so)
		fmt.Fprintln(globals.Stdout, string(data))
	} else {
		// Just output the path for easy piping
		fmt.Fprintln(globals.Stdout, session.Path)
	}

	return nil
}

func (c *SessionsShowCmd) outputError(globals *Globals, code, message string) error {
	if globals.Format == "ndjson" {
		output.NewNDJSONWriter(globals.Stdout).WriteError(code, message)
	} else {
		fmt.Fprintf(globals.Stderr, "Error [%s]: %s\n", code, message)
	}
	return errors.New(message)
}

// SessionsCleanCmd deletes old session files
type SessionsCleanCmd struct {
	Dir    string `help:"Session directory (default: ~/.xcw/sessions)"`
	Keep   int    `default:"10" help:"Number of sessions to keep"`
	DryRun bool   `help:"Show what would be deleted without deleting"`
}

// Run executes the sessions clean command
func (c *SessionsCleanCmd) Run(globals *Globals) error {
	if c.DryRun {
		// Show what would be deleted
		sessions, err := ListSessions(c.Dir)
		if err != nil {
			return c.outputError(globals, "LIST_SESSIONS_ERROR", err.Error())
		}

		if len(sessions) <= c.Keep {
			if globals.Format == "ndjson" {
				output.NewNDJSONWriter(globals.Stdout).WriteInfo(
					fmt.Sprintf("Nothing to clean (have %d, keeping %d)", len(sessions), c.Keep),
					"", "", "", "")
			} else {
				fmt.Fprintf(globals.Stdout, "Nothing to clean (have %d sessions, keeping %d)\n", len(sessions), c.Keep)
			}
			return nil
		}

		toDelete := sessions[c.Keep:]
		if globals.Format == "ndjson" {
			for _, s := range toDelete {
				output.NewNDJSONWriter(globals.Stdout).WriteInfo(
					fmt.Sprintf("Would delete: %s", s.Name),
					"", "", "", "")
			}
		} else {
			fmt.Fprintf(globals.Stdout, "Would delete %d session(s):\n", len(toDelete))
			for _, s := range toDelete {
				fmt.Fprintf(globals.Stdout, "  %s\n", s.Name)
			}
		}
		return nil
	}

	// Actually delete
	deleted, err := CleanOldSessions(c.Dir, c.Keep)
	if err != nil {
		return c.outputError(globals, "CLEAN_ERROR", err.Error())
	}

	if len(deleted) == 0 {
		if globals.Format == "ndjson" {
			output.NewNDJSONWriter(globals.Stdout).WriteInfo("No sessions to clean", "", "", "", "")
		} else {
			fmt.Fprintln(globals.Stdout, "No sessions to clean")
		}
	} else {
		if globals.Format == "ndjson" {
			output.NewNDJSONWriter(globals.Stdout).WriteInfo(
				fmt.Sprintf("Deleted %d session(s)", len(deleted)),
				"", "", "", "")
		} else {
			fmt.Fprintf(globals.Stdout, "Deleted %d session(s):\n", len(deleted))
			for _, p := range deleted {
				fmt.Fprintf(globals.Stdout, "  %s\n", filepath.Base(p))
			}
		}
	}

	return nil
}

func (c *SessionsCleanCmd) outputError(globals *Globals, code, message string) error {
	if globals.Format == "ndjson" {
		output.NewNDJSONWriter(globals.Stdout).WriteError(code, message)
	} else {
		fmt.Fprintf(globals.Stderr, "Error [%s]: %s\n", code, message)
	}
	return errors.New(message)
}

// SessionOutput is the NDJSON output format for session info
type SessionOutput struct {
	Type          string `json:"type"`
	SchemaVersion int    `json:"schemaVersion"`
	Path          string `json:"path"`
	Name          string `json:"name"`
	Timestamp     string `json:"timestamp"`
	Size          int64  `json:"size"`
	Prefix        string `json:"prefix,omitempty"`
}

// formatSize formats bytes into human-readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return strconv.FormatInt(bytes, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// CatSession reads and outputs the contents of a session file
func CatSession(path string, tail int) (*bufio.Scanner, *os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1MB max line

	return scanner, file, nil
}
