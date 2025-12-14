package cli

import (
	"regexp"

	"github.com/vburojevic/xcw/internal/domain"
	"github.com/vburojevic/xcw/internal/filter"
)

// buildFilters compiles regex and where filters for tail-like commands.
func buildFilters(patternStr string, exclude []string, where []string) (*regexp.Regexp, []*regexp.Regexp, *filter.WhereFilter, error) {
	var pattern *regexp.Regexp
	if patternStr != "" {
		var err error
		pattern, err = regexp.Compile(patternStr)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	var excludePatterns []*regexp.Regexp
	for _, excl := range exclude {
		re, err := regexp.Compile(excl)
		if err != nil {
			return nil, nil, nil, err
		}
		excludePatterns = append(excludePatterns, re)
	}

	var whereFilter *filter.WhereFilter
	if len(where) > 0 {
		wf, err := filter.NewWhereFilter(where)
		if err != nil {
			return nil, nil, nil, err
		}
		whereFilter = wf
	}

	return pattern, excludePatterns, whereFilter, nil
}

// resolveLevels picks min/max level given cmd overrides and globals
func resolveLevels(minOverride, maxOverride string, globalsMin string) (domain.LogLevel, domain.LogLevel) {
	minLevel := globalsMin
	if minOverride != "" {
		minLevel = minOverride
	}
	var maxLevel domain.LogLevel
	if maxOverride != "" {
		maxLevel = domain.ParseLogLevel(maxOverride)
	}
	return domain.ParseLogLevel(minLevel), maxLevel
}
