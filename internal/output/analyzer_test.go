package output

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vburojevic/xcw/internal/domain"
)

func TestAnalyzer_NormalizeMessage(t *testing.T) {
	a := NewAnalyzer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "replaces hex addresses",
			input:    "Pointer at 0x7fff5fbff8c0 is invalid",
			expected: "Pointer at <addr> is invalid",
		},
		{
			name:     "replaces numbers",
			input:    "Failed after 123 attempts with code 456",
			expected: "Failed after <n> attempts with code <n>",
		},
		{
			name:     "replaces UUIDs",
			input:    "Device 12345678-1234-1234-1234-123456789abc not found",
			expected: "Device <uuid> not found",
		},
		{
			name:     "handles mixed content",
			input:    "Error at 0xABCDEF: request 42 for UUID 11111111-2222-3333-4444-555555555555 failed",
			expected: "Error at <addr>: request <n> for UUID <uuid> failed",
		},
		{
			name:     "truncates long messages",
			input:    "This is a very long message that exceeds one hundred characters and should be truncated at the limit to prevent overly verbose output",
			expected: "This is a very long message that exceeds one hundred characters and should be truncated at the limit...",
		},
		{
			name:     "trims whitespace",
			input:    "  Message with spaces  ",
			expected: "Message with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.normalizeMessage(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnalyzer_Summarize(t *testing.T) {
	a := NewAnalyzer()

	t.Run("returns empty summary for no entries", func(t *testing.T) {
		summary := a.Summarize([]domain.LogEntry{})
		assert.Equal(t, 0, summary.TotalCount)
		assert.Equal(t, 0, summary.ErrorCount)
		assert.False(t, summary.HasErrors)
	})

	t.Run("counts entries by level", func(t *testing.T) {
		entries := []domain.LogEntry{
			{Timestamp: time.Now(), Level: domain.LogLevelDebug, Message: "debug"},
			{Timestamp: time.Now(), Level: domain.LogLevelInfo, Message: "info"},
			{Timestamp: time.Now(), Level: domain.LogLevelDefault, Message: "default"},
			{Timestamp: time.Now(), Level: domain.LogLevelError, Message: "error1"},
			{Timestamp: time.Now(), Level: domain.LogLevelError, Message: "error2"},
			{Timestamp: time.Now(), Level: domain.LogLevelFault, Message: "fault"},
		}

		summary := a.Summarize(entries)

		assert.Equal(t, 6, summary.TotalCount)
		assert.Equal(t, 1, summary.DebugCount)
		assert.Equal(t, 1, summary.InfoCount)
		assert.Equal(t, 1, summary.DefaultCount)
		assert.Equal(t, 2, summary.ErrorCount)
		assert.Equal(t, 1, summary.FaultCount)
		assert.True(t, summary.HasErrors)
		assert.True(t, summary.HasFaults)
	})

	t.Run("sets time window from entries", func(t *testing.T) {
		start := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		end := time.Date(2024, 1, 15, 10, 5, 0, 0, time.UTC)

		entries := []domain.LogEntry{
			{Timestamp: start, Level: domain.LogLevelInfo, Message: "first"},
			{Timestamp: end, Level: domain.LogLevelInfo, Message: "last"},
		}

		summary := a.Summarize(entries)

		assert.Equal(t, start, summary.WindowStart)
		assert.Equal(t, end, summary.WindowEnd)
	})

	t.Run("extracts top errors", func(t *testing.T) {
		entries := []domain.LogEntry{
			{Timestamp: time.Now(), Level: domain.LogLevelError, Message: "Connection timeout 1"},
			{Timestamp: time.Now(), Level: domain.LogLevelError, Message: "Connection timeout 2"},
			{Timestamp: time.Now(), Level: domain.LogLevelError, Message: "Connection timeout 3"},
			{Timestamp: time.Now(), Level: domain.LogLevelError, Message: "Auth failed"},
		}

		summary := a.Summarize(entries)

		assert.Len(t, summary.TopErrors, 2)
		// Normalized "Connection timeout <n>" should be most common
		assert.Contains(t, summary.TopErrors[0], "Connection timeout")
	})
}

func TestAnalyzer_DetectPatterns(t *testing.T) {
	a := NewAnalyzer()

	t.Run("groups similar error messages", func(t *testing.T) {
		entries := []domain.LogEntry{
			{Timestamp: time.Now(), Level: domain.LogLevelError, Message: "Request 1 failed"},
			{Timestamp: time.Now(), Level: domain.LogLevelError, Message: "Request 2 failed"},
			{Timestamp: time.Now(), Level: domain.LogLevelError, Message: "Request 3 failed"},
			{Timestamp: time.Now(), Level: domain.LogLevelInfo, Message: "Request 4 succeeded"},
		}

		patterns := a.DetectPatterns(entries)

		assert.Len(t, patterns, 1)
		assert.Equal(t, 3, patterns[0].Count)
		assert.Contains(t, patterns[0].Pattern, "Request")
	})

	t.Run("ignores patterns with single occurrence", func(t *testing.T) {
		entries := []domain.LogEntry{
			{Timestamp: time.Now(), Level: domain.LogLevelError, Message: "Unique error A"},
			{Timestamp: time.Now(), Level: domain.LogLevelError, Message: "Unique error B"},
		}

		patterns := a.DetectPatterns(entries)

		assert.Empty(t, patterns)
	})

	t.Run("limits samples to 3", func(t *testing.T) {
		entries := []domain.LogEntry{
			{Timestamp: time.Now(), Level: domain.LogLevelError, Message: "Error 1"},
			{Timestamp: time.Now(), Level: domain.LogLevelError, Message: "Error 2"},
			{Timestamp: time.Now(), Level: domain.LogLevelError, Message: "Error 3"},
			{Timestamp: time.Now(), Level: domain.LogLevelError, Message: "Error 4"},
			{Timestamp: time.Now(), Level: domain.LogLevelError, Message: "Error 5"},
		}

		patterns := a.DetectPatterns(entries)

		assert.Len(t, patterns, 1)
		assert.Len(t, patterns[0].Samples, 3)
	})
}

func TestPrecompiledRegexes(t *testing.T) {
	// Test that the package-level regexes are properly compiled
	t.Run("hexAddrRegex matches hex addresses", func(t *testing.T) {
		assert.True(t, hexAddrRegex.MatchString("0x7fff5fbff8c0"))
		assert.True(t, hexAddrRegex.MatchString("0xABCDEF"))
		assert.False(t, hexAddrRegex.MatchString("not-a-hex"))
	})

	t.Run("numberRegex matches numbers", func(t *testing.T) {
		assert.True(t, numberRegex.MatchString("123"))
		assert.True(t, numberRegex.MatchString("0"))
		assert.False(t, numberRegex.MatchString("abc"))
	})

	t.Run("uuidRegex matches UUIDs", func(t *testing.T) {
		assert.True(t, uuidRegex.MatchString("12345678-1234-1234-1234-123456789abc"))
		assert.True(t, uuidRegex.MatchString("ABCDEF12-3456-7890-ABCD-EF1234567890"))
		assert.False(t, uuidRegex.MatchString("not-a-uuid"))
	})
}
