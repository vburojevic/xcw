package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// DefaultSessionDir is the default directory for session files relative to home
const DefaultSessionDir = ".xcw/sessions"

// SessionTimestampFormat is the timestamp format used in session filenames
const SessionTimestampFormat = "20060102-150405"

// SessionFile represents a session log file
type SessionFile struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
	Size      int64     `json:"size"`
	Prefix    string    `json:"prefix,omitempty"`
}

// GetDefaultSessionDir returns the default session directory path
func GetDefaultSessionDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return DefaultSessionDir
	}
	return filepath.Join(home, DefaultSessionDir)
}

// GenerateSessionPath creates a timestamped session file path
func GenerateSessionPath(dir, prefix string) (string, error) {
	if dir == "" {
		dir = GetDefaultSessionDir()
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create session directory: %w", err)
	}

	timestamp := time.Now().Format(SessionTimestampFormat)
	sanitized := sanitizePrefix(prefix)
	filename := fmt.Sprintf("%s-%s.ndjson", timestamp, sanitized)

	return filepath.Join(dir, filename), nil
}

// sanitizePrefix removes or replaces characters that are problematic in filenames
func sanitizePrefix(prefix string) string {
	// Replace dots and slashes with underscores, keep alphanumerics and hyphens
	re := regexp.MustCompile(`[^a-zA-Z0-9\-]`)
	sanitized := re.ReplaceAllString(prefix, "_")

	// Collapse multiple underscores
	re = regexp.MustCompile(`_+`)
	sanitized = re.ReplaceAllString(sanitized, "_")

	// Trim leading/trailing underscores
	sanitized = strings.Trim(sanitized, "_")

	if sanitized == "" {
		sanitized = "session"
	}

	return sanitized
}

// ListSessions returns session files sorted by time (newest first)
func ListSessions(dir string) ([]SessionFile, error) {
	if dir == "" {
		dir = GetDefaultSessionDir()
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No sessions directory yet
		}
		return nil, fmt.Errorf("failed to read session directory: %w", err)
	}

	var sessions []SessionFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".ndjson") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Parse timestamp from filename (format: 20060102-150405-prefix.ndjson)
		session := SessionFile{
			Path: filepath.Join(dir, name),
			Name: name,
			Size: info.Size(),
		}

		// Try to parse timestamp from filename
		if len(name) >= 15 {
			timestampStr := name[:15] // "20060102-150405"
			if t, err := time.Parse(SessionTimestampFormat, timestampStr); err == nil {
				session.Timestamp = t
			}

			// Extract prefix (everything between timestamp and .ndjson)
			if len(name) > 16 {
				prefixPart := name[16 : len(name)-7] // Skip "20060102-150405-" and ".ndjson"
				session.Prefix = prefixPart
			}
		}

		// Fallback to file mod time if timestamp parsing failed
		if session.Timestamp.IsZero() {
			session.Timestamp = info.ModTime()
		}

		sessions = append(sessions, session)
	}

	// Sort by timestamp, newest first
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Timestamp.After(sessions[j].Timestamp)
	})

	return sessions, nil
}

// LatestSession returns the most recent session file
func LatestSession(dir string) (*SessionFile, error) {
	sessions, err := ListSessions(dir)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, nil
	}
	return &sessions[0], nil
}

// CleanOldSessions removes old session files, keeping the specified number
func CleanOldSessions(dir string, keep int) ([]string, error) {
	sessions, err := ListSessions(dir)
	if err != nil {
		return nil, err
	}

	if len(sessions) <= keep {
		return nil, nil // Nothing to delete
	}

	var deleted []string
	for _, session := range sessions[keep:] {
		if err := os.Remove(session.Path); err != nil {
			return deleted, fmt.Errorf("failed to remove %s: %w", session.Path, err)
		}
		deleted = append(deleted, session.Path)
	}

	return deleted, nil
}
