package filter

import (
	"regexp"

	"github.com/vburojevic/xcw/internal/domain"
)

// Pipeline chains pattern/exclude/where predicates so callers can reuse a single matcher.
type Pipeline struct {
	pattern  *regexp.Regexp
	excludes []*regexp.Regexp
	where    *WhereFilter
}

func NewPipeline(pattern *regexp.Regexp, excludes []*regexp.Regexp, where *WhereFilter) *Pipeline {
	if pattern == nil && len(excludes) == 0 && where == nil {
		return nil
	}
	return &Pipeline{pattern: pattern, excludes: excludes, where: where}
}

// Match returns true when the log entry passes all predicates.
func (p *Pipeline) Match(entry *domain.LogEntry) bool {
	if p == nil || entry == nil {
		return true
	}
	if p.pattern != nil && !p.pattern.MatchString(entry.Message) {
		return false
	}
	for _, ex := range p.excludes {
		if ex.MatchString(entry.Message) {
			return false
		}
	}
	if p.where != nil && !p.where.Match(entry) {
		return false
	}
	return true
}
