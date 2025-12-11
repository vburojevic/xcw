package output

import (
	"testing"
	"time"

	"github.com/vburojevic/xcw/internal/domain"
)

func BenchmarkNormalizeMessage(b *testing.B) {
	a := NewAnalyzer()
	msg := "Error: failed to connect to 0x1234abcd with id 123e4567-e89b-12d3-a456-426614174000 after 42ms"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.normalizeMessage(msg)
	}
}

func BenchmarkSummarize(b *testing.B) {
	a := NewAnalyzer()
	entries := make([]domain.LogEntry, 1000)
	now := time.Now()
	for i := range entries {
		entries[i] = domain.LogEntry{
			Timestamp: now.Add(time.Duration(i) * time.Millisecond),
			Level:     domain.LogLevelError,
			Message:   "Connection timeout occurred",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.Summarize(entries)
	}
}

