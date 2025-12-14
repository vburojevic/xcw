package simulator

import (
	"path/filepath"
	"regexp"
	"strings"
)

func matchProcess(process string, patterns []string) bool {
	for _, p := range patterns {
		if p == "" {
			continue
		}

		// Regex notation: re:<pattern> or /pattern/
		if strings.HasPrefix(p, "re:") {
			if re, err := regexp.Compile(p[3:]); err == nil && re.MatchString(process) {
				return true
			}
			continue
		}
		if strings.HasPrefix(p, "/") && strings.HasSuffix(p, "/") && len(p) > 1 {
			pat := strings.TrimSuffix(strings.TrimPrefix(p, "/"), "/")
			if re, err := regexp.Compile(pat); err == nil && re.MatchString(process) {
				return true
			}
			continue
		}

		// Glob/prefix matching when wildcards are present
		if strings.ContainsAny(p, "*?[") {
			if ok, _ := filepath.Match(p, process); ok {
				return true
			}
			continue
		}

		// Exact match fallback
		if process == p {
			return true
		}
	}
	return false
}
