package filter

import (
	"strings"

	"github.com/vburojevic/xcw/internal/domain"
)

// AppFilter filters logs by bundle identifier (subsystem prefix)
type AppFilter struct {
	bundleID string
}

// NewAppFilter creates an app filter
func NewAppFilter(bundleID string) *AppFilter {
	return &AppFilter{bundleID: bundleID}
}

// Match returns true if the entry subsystem starts with the bundle ID
func (f *AppFilter) Match(entry *domain.LogEntry) bool {
	if f.bundleID == "" {
		return true
	}
	return strings.HasPrefix(entry.Subsystem, f.bundleID)
}

// SubsystemFilter filters logs by exact subsystem match
type SubsystemFilter struct {
	subsystems []string
}

// NewSubsystemFilter creates a subsystem filter
func NewSubsystemFilter(subsystems []string) *SubsystemFilter {
	return &SubsystemFilter{subsystems: subsystems}
}

// Match returns true if the entry subsystem is in the list
func (f *SubsystemFilter) Match(entry *domain.LogEntry) bool {
	if len(f.subsystems) == 0 {
		return true
	}
	for _, s := range f.subsystems {
		if entry.Subsystem == s {
			return true
		}
	}
	return false
}

// CategoryFilter filters logs by category
type CategoryFilter struct {
	categories []string
}

// NewCategoryFilter creates a category filter
func NewCategoryFilter(categories []string) *CategoryFilter {
	return &CategoryFilter{categories: categories}
}

// Match returns true if the entry category is in the list
func (f *CategoryFilter) Match(entry *domain.LogEntry) bool {
	if len(f.categories) == 0 {
		return true
	}
	for _, c := range f.categories {
		if entry.Category == c {
			return true
		}
	}
	return false
}

// ProcessFilter filters logs by process name
type ProcessFilter struct {
	processes []string
}

// NewProcessFilter creates a process filter
func NewProcessFilter(processes []string) *ProcessFilter {
	return &ProcessFilter{processes: processes}
}

// Match returns true if the entry process is in the list
func (f *ProcessFilter) Match(entry *domain.LogEntry) bool {
	if len(f.processes) == 0 {
		return true
	}
	for _, p := range f.processes {
		if entry.Process == p {
			return true
		}
	}
	return false
}
