package filter

import (
	"regexp"

	"github.com/vburojevic/xcw/internal/domain"
)

// RegexFilter filters logs by message pattern
type RegexFilter struct {
	pattern *regexp.Regexp
}

// NewRegexFilter creates a regex filter from a pattern string
func NewRegexFilter(pattern string) (*RegexFilter, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &RegexFilter{pattern: re}, nil
}

// NewRegexFilterFromRegexp creates a regex filter from a compiled regexp
func NewRegexFilterFromRegexp(re *regexp.Regexp) *RegexFilter {
	return &RegexFilter{pattern: re}
}

// Match returns true if the entry message matches the pattern
func (f *RegexFilter) Match(entry *domain.LogEntry) bool {
	if f.pattern == nil {
		return true
	}
	return f.pattern.MatchString(entry.Message)
}
