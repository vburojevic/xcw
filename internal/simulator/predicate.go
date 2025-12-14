package simulator

import (
	"fmt"
	"strings"
)

func predicateQuoteLiteral(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// buildPredicate constructs an NSPredicate string for unified log filtering.
// - Uses AND between groups (subsystem, category).
// - Uses OR within a group.
// - rawPredicate, when provided, overrides all other inputs.
func buildPredicate(rawPredicate, bundleID string, subsystems, categories []string) string {
	if rawPredicate != "" {
		return rawPredicate
	}

	var groups []string

	// Subsystem group: bundle ID and/or explicit subsystems (OR within group)
	var subsystemParts []string
	if strings.TrimSpace(bundleID) != "" {
		subsystemParts = append(subsystemParts, fmt.Sprintf("subsystem BEGINSWITH %s", predicateQuoteLiteral(bundleID)))
	}
	for _, sub := range subsystems {
		if strings.TrimSpace(sub) == "" {
			continue
		}
		subsystemParts = append(subsystemParts, fmt.Sprintf("subsystem == %s", predicateQuoteLiteral(sub)))
	}
	if len(subsystemParts) > 0 {
		if len(subsystemParts) == 1 {
			groups = append(groups, subsystemParts[0])
		} else {
			groups = append(groups, "("+strings.Join(subsystemParts, " OR ")+")")
		}
	}

	// Category group (OR within group)
	var categoryParts []string
	for _, cat := range categories {
		if strings.TrimSpace(cat) == "" {
			continue
		}
		categoryParts = append(categoryParts, fmt.Sprintf("category == %s", predicateQuoteLiteral(cat)))
	}
	if len(categoryParts) > 0 {
		if len(categoryParts) == 1 {
			groups = append(groups, categoryParts[0])
		} else {
			groups = append(groups, "("+strings.Join(categoryParts, " OR ")+")")
		}
	}

	if len(groups) == 0 {
		return ""
	}

	return strings.Join(groups, " AND ")
}
