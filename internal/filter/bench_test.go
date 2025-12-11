package filter

import (
	"regexp"
	"testing"
	"time"

	"github.com/vburojevic/xcw/internal/domain"
)

func BenchmarkWhereFilterMatch(b *testing.B) {
	where, _ := NewWhereFilter([]string{"level>=error", "message~timeout", "pid>=100"})
	entry := &domain.LogEntry{
		Level:     domain.LogLevelError,
		Message:   "network timeout occurred",
		PID:       1234,
		Subsystem: "com.example",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = where.Match(entry)
	}
}

func BenchmarkDedupeCheck(b *testing.B) {
	f := NewDedupeFilter(0)
	entry := &domain.LogEntry{Message: "same message", Timestamp: time.Now()}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Check(entry)
	}
}

func BenchmarkPipelineMatch(b *testing.B) {
	pat, _ := regexp.Compile("error|timeout")
	ex, _ := regexp.Compile("heartbeat")
	where, _ := NewWhereFilter([]string{"level>=info"})
	p := NewPipeline(pat, []*regexp.Regexp{ex}, where)
	entry := &domain.LogEntry{Level: domain.LogLevelError, Message: "timeout error"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.Match(entry)
	}
}
