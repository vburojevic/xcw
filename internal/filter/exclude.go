package filter

import (
	"regexp"
	"strings"

	"github.com/vburojevic/xcw/internal/domain"
)

// ExcludePatternFilter excludes logs matching a regex pattern
type ExcludePatternFilter struct {
	pattern *regexp.Regexp
}

// NewExcludePatternFilter creates an exclusion filter from a pattern string
func NewExcludePatternFilter(pattern string) (*ExcludePatternFilter, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &ExcludePatternFilter{pattern: re}, nil
}

// Match returns true if the entry does NOT match the exclusion pattern
func (f *ExcludePatternFilter) Match(entry *domain.LogEntry) bool {
	if f.pattern == nil {
		return true
	}
	// Return true (pass) if the message does NOT match the exclusion pattern
	return !f.pattern.MatchString(entry.Message)
}

// ExcludeSubsystemFilter excludes logs from specific subsystems
type ExcludeSubsystemFilter struct {
	subsystems []string
}

// NewExcludeSubsystemFilter creates an exclusion filter for subsystems
func NewExcludeSubsystemFilter(subsystems []string) *ExcludeSubsystemFilter {
	return &ExcludeSubsystemFilter{subsystems: subsystems}
}

// Match returns true if the entry's subsystem is NOT in the exclusion list
func (f *ExcludeSubsystemFilter) Match(entry *domain.LogEntry) bool {
	if len(f.subsystems) == 0 {
		return true
	}
	for _, sub := range f.subsystems {
		// Support prefix matching with wildcard
		if strings.HasSuffix(sub, "*") {
			prefix := strings.TrimSuffix(sub, "*")
			if strings.HasPrefix(entry.Subsystem, prefix) {
				return false
			}
		} else if entry.Subsystem == sub {
			return false
		}
	}
	return true
}
