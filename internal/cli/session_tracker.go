package cli

import (
	"github.com/vburojevic/xcw/internal/domain"
	"github.com/vburojevic/xcw/internal/session"
)

type tailSessionTracker interface {
	CurrentSession() int
	CheckEntry(entry *domain.LogEntry) *session.SessionChange
	GetFinalSummary() *domain.SessionEnd
	ForceRollover(alert string) *session.SessionChange
}

type noopSessionTracker struct{}

func (t *noopSessionTracker) CurrentSession() int { return 0 }
func (t *noopSessionTracker) CheckEntry(entry *domain.LogEntry) *session.SessionChange {
	return nil
}
func (t *noopSessionTracker) GetFinalSummary() *domain.SessionEnd { return nil }
func (t *noopSessionTracker) ForceRollover(alert string) *session.SessionChange {
	return nil
}
