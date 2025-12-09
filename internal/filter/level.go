package filter

import (
	"github.com/vburojevic/xcw/internal/domain"
)

// LevelFilter filters logs by minimum log level
type LevelFilter struct {
	minLevel domain.LogLevel
}

// NewLevelFilter creates a level filter
func NewLevelFilter(minLevel domain.LogLevel) *LevelFilter {
	return &LevelFilter{minLevel: minLevel}
}

// Match returns true if the entry level is >= minimum level
func (f *LevelFilter) Match(entry *domain.LogEntry) bool {
	return entry.Level.Priority() >= f.minLevel.Priority()
}
