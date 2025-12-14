package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/vburojevic/xcw/internal/domain"
)

// WhereClause represents a parsed --where condition
type WhereClause struct {
	Field    string
	Operator string
	Value    string
	regex    *regexp.Regexp // Compiled regex for ~ and !~ operators
}

// ParseWhereClause parses a where clause like "level=error" or "message~timeout"
// Supported operators: =, !=, ~, !~, >=, <=, ^, $
func ParseWhereClause(clause string) (*WhereClause, error) {
	// Try operators in order of length (longest first to avoid partial matches)
	operators := []string{"!~", ">=", "<=", "!=", "~", "=", "^", "$"}

	for _, op := range operators {
		idx := strings.Index(clause, op)
		if idx > 0 {
			field := strings.TrimSpace(clause[:idx])
			value := strings.TrimSpace(clause[idx+len(op):])

			if field == "" || value == "" {
				return nil, fmt.Errorf("invalid where clause: %s", clause)
			}

			// Support quoted values so operators can appear in value.
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				unq, err := strconv.Unquote(value)
				if err != nil {
					return nil, fmt.Errorf("invalid quoted value in where clause '%s': %w", clause, err)
				}
				value = unq
			}

			wc := &WhereClause{
				Field:    field,
				Operator: op,
				Value:    value,
			}

			// Pre-compile regex for ~ and !~ operators
			if op == "~" || op == "!~" {
				re, err := regexp.Compile(value)
				if err != nil {
					return nil, fmt.Errorf("invalid regex in where clause '%s': %w", clause, err)
				}
				wc.regex = re
			}

			return wc, nil
		}
	}

	return nil, fmt.Errorf("no valid operator found in where clause: %s (use =, !=, ~, !~, >=, <=, ^, $)", clause)
}

// Match checks if a log entry matches this where clause
func (wc *WhereClause) Match(entry *domain.LogEntry) bool {
	// Get the field value from the entry
	fieldValue := wc.getFieldValue(entry)

	switch wc.Operator {
	case "=":
		// Case-insensitive level equality
		if strings.ToLower(wc.Field) == "level" {
			return entry.Level == domain.ParseLogLevel(wc.Value)
		}
		// Numeric equality for pid/tid
		if strings.ToLower(wc.Field) == "pid" || strings.ToLower(wc.Field) == "tid" {
			return wc.compareNumeric(entry, true, true)
		}
		return fieldValue == wc.Value
	case "!=":
		if strings.ToLower(wc.Field) == "level" {
			return entry.Level != domain.ParseLogLevel(wc.Value)
		}
		if strings.ToLower(wc.Field) == "pid" || strings.ToLower(wc.Field) == "tid" {
			return wc.compareNumeric(entry, false, true)
		}
		return fieldValue != wc.Value
	case "~": // Contains (regex)
		if wc.regex != nil {
			return wc.regex.MatchString(fieldValue)
		}
		return strings.Contains(fieldValue, wc.Value)
	case "!~": // Not contains (regex)
		if wc.regex != nil {
			return !wc.regex.MatchString(fieldValue)
		}
		return !strings.Contains(fieldValue, wc.Value)
	case "^": // Starts with
		return strings.HasPrefix(fieldValue, wc.Value)
	case "$": // Ends with
		return strings.HasSuffix(fieldValue, wc.Value)
	case ">=": // Greater or equal (for levels)
		if strings.ToLower(wc.Field) == "level" {
			return wc.compareLevel(entry, true)
		}
		return wc.compareNumeric(entry, true, false)
	case "<=": // Less or equal (for levels)
		if strings.ToLower(wc.Field) == "level" {
			return wc.compareLevel(entry, false)
		}
		return wc.compareNumeric(entry, false, false)
	}

	return false
}

// getFieldValue extracts the field value from a log entry
func (wc *WhereClause) getFieldValue(entry *domain.LogEntry) string {
	switch strings.ToLower(wc.Field) {
	case "level":
		return string(entry.Level)
	case "subsystem":
		return entry.Subsystem
	case "category":
		return entry.Category
	case "process":
		return entry.Process
	case "message":
		return entry.Message
	case "pid":
		return strconv.Itoa(entry.PID)
	case "tid":
		return strconv.Itoa(entry.TID)
	default:
		return ""
	}
}

// compareLevel handles >= and <= comparisons for log levels
func (wc *WhereClause) compareLevel(entry *domain.LogEntry, greaterOrEqual bool) bool {
	if strings.ToLower(wc.Field) != "level" {
		return false
	}

	targetLevel := domain.ParseLogLevel(wc.Value)
	entryPriority := entry.Level.Priority()
	targetPriority := targetLevel.Priority()

	if greaterOrEqual {
		return entryPriority >= targetPriority
	}
	return entryPriority <= targetPriority
}

// compareNumeric handles integer comparisons for pid/tid.
// If equality is true, greaterOrEqual indicates equality vs inequality for = / !=.
func (wc *WhereClause) compareNumeric(entry *domain.LogEntry, greaterOrEqual bool, equality bool) bool {
	field := strings.ToLower(wc.Field)
	var entryVal int
	switch field {
	case "pid":
		entryVal = entry.PID
	case "tid":
		entryVal = entry.TID
	default:
		return false
	}

	targetVal, err := strconv.Atoi(wc.Value)
	if err != nil {
		return false
	}

	if equality {
		if greaterOrEqual {
			return entryVal == targetVal
		}
		return entryVal != targetVal
	}

	if greaterOrEqual {
		return entryVal >= targetVal
	}
	return entryVal <= targetVal
}

// WhereFilter is a filter that applies multiple where clauses (AND logic)
type WhereFilter struct {
	expr whereExpr
}

// NewWhereFilter creates a filter from multiple where clause strings
func NewWhereFilter(whereClauses []string) (*WhereFilter, error) {
	if len(whereClauses) == 0 {
		return nil, nil
	}

	filter := &WhereFilter{}
	for _, clause := range whereClauses {
		expr, err := parseWhereExpr(clause)
		if err != nil {
			return nil, err
		}
		if filter.expr == nil {
			filter.expr = expr
		} else {
			filter.expr = &whereAndExpr{left: filter.expr, right: expr}
		}
	}

	return filter, nil
}

// Match returns true if the entry matches ALL where clauses (AND logic)
func (f *WhereFilter) Match(entry *domain.LogEntry) bool {
	if f == nil || f.expr == nil {
		return true
	}
	return f.expr.Match(entry)
}
