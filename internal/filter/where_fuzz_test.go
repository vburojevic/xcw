package filter

import (
	"testing"

	"github.com/vburojevic/xcw/internal/domain"
)

func FuzzNewWhereFilter(f *testing.F) {
	f.Add(`level=error`)
	f.Add(`(level=Error OR level=Fault) AND message~/timeout|crash/i`)
	f.Add(`pid>=123 && process^MyApp`)
	f.Add(`!message~"hello"`)
	f.Add(`unterminated"`)

	entry := &domain.LogEntry{
		Process:   "MyApp",
		Subsystem: "com.example",
		Category:  "net",
		Message:   "timeout while connecting",
		PID:       123,
		TID:       1,
		Level:     domain.LogLevelError,
	}

	f.Fuzz(func(t *testing.T, expr string) {
		wf, err := NewWhereFilter([]string{expr})
		if err != nil {
			return
		}
		if wf != nil {
			_ = wf.Match(entry)
		}
	})
}
