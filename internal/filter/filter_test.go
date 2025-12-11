package filter

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vburojevic/xcw/internal/domain"
)

func TestLevelFilter(t *testing.T) {
	tests := []struct {
		name     string
		minLevel domain.LogLevel
		entry    domain.LogLevel
		expected bool
	}{
		{"debug allows debug", domain.LogLevelDebug, domain.LogLevelDebug, true},
		{"debug allows info", domain.LogLevelDebug, domain.LogLevelInfo, true},
		{"debug allows error", domain.LogLevelDebug, domain.LogLevelError, true},
		{"error filters debug", domain.LogLevelError, domain.LogLevelDebug, false},
		{"error filters info", domain.LogLevelError, domain.LogLevelInfo, false},
		{"error allows error", domain.LogLevelError, domain.LogLevelError, true},
		{"error allows fault", domain.LogLevelError, domain.LogLevelFault, true},
		{"fault only allows fault", domain.LogLevelFault, domain.LogLevelError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewLevelFilter(tt.minLevel)
			entry := &domain.LogEntry{Level: tt.entry}
			assert.Equal(t, tt.expected, filter.Match(entry))
		})
	}
}

func TestChain(t *testing.T) {
	t.Run("empty chain matches all", func(t *testing.T) {
		chain := NewChain()
		entry := &domain.LogEntry{Level: domain.LogLevelDebug}
		assert.True(t, chain.Match(entry))
	})

	t.Run("all filters must pass", func(t *testing.T) {
		// Create filters: level >= Info AND message contains "error"
		levelFilter := NewLevelFilter(domain.LogLevelInfo)
		regexFilter := NewRegexFilterFromRegexp(regexp.MustCompile("error"))
		chain := NewChain(levelFilter, regexFilter)

		// Debug level doesn't pass
		entry1 := &domain.LogEntry{Level: domain.LogLevelDebug, Message: "error occurred"}
		assert.False(t, chain.Match(entry1))

		// Message doesn't contain "error"
		entry2 := &domain.LogEntry{Level: domain.LogLevelError, Message: "something happened"}
		assert.False(t, chain.Match(entry2))

		// Both conditions pass
		entry3 := &domain.LogEntry{Level: domain.LogLevelError, Message: "error occurred"}
		assert.True(t, chain.Match(entry3))
	})

	t.Run("add filter to chain", func(t *testing.T) {
		chain := NewChain()
		chain.Add(NewLevelFilter(domain.LogLevelError))

		debugEntry := &domain.LogEntry{Level: domain.LogLevelDebug}
		errorEntry := &domain.LogEntry{Level: domain.LogLevelError}

		assert.False(t, chain.Match(debugEntry))
		assert.True(t, chain.Match(errorEntry))
	})
}

func TestOrChain(t *testing.T) {
	t.Run("empty OR chain matches all", func(t *testing.T) {
		chain := NewOrChain()
		entry := &domain.LogEntry{Level: domain.LogLevelDebug}
		assert.True(t, chain.Match(entry))
	})

	t.Run("any filter can pass", func(t *testing.T) {
		// Message contains "error" OR "warning"
		errorFilter := NewRegexFilterFromRegexp(regexp.MustCompile("error"))
		warningFilter := NewRegexFilterFromRegexp(regexp.MustCompile("warning"))
		chain := NewOrChain(errorFilter, warningFilter)

		entry1 := &domain.LogEntry{Message: "error occurred"}
		assert.True(t, chain.Match(entry1))

		entry2 := &domain.LogEntry{Message: "warning issued"}
		assert.True(t, chain.Match(entry2))

		entry3 := &domain.LogEntry{Message: "info message"}
		assert.False(t, chain.Match(entry3))
	})
}

func TestRegexFilter(t *testing.T) {
	t.Run("nil pattern matches all", func(t *testing.T) {
		filter := NewRegexFilterFromRegexp(nil)
		entry := &domain.LogEntry{Message: "anything"}
		assert.True(t, filter.Match(entry))
	})

	t.Run("matches message", func(t *testing.T) {
		filter := NewRegexFilterFromRegexp(regexp.MustCompile(`error|warn`))

		assert.True(t, filter.Match(&domain.LogEntry{Message: "An error occurred"}))
		assert.True(t, filter.Match(&domain.LogEntry{Message: "warning: something"})) // lowercase
		assert.False(t, filter.Match(&domain.LogEntry{Message: "Info message"}))
	})

	t.Run("case sensitive by default", func(t *testing.T) {
		filter := NewRegexFilterFromRegexp(regexp.MustCompile(`Error`))
		assert.True(t, filter.Match(&domain.LogEntry{Message: "Error message"}))
		assert.False(t, filter.Match(&domain.LogEntry{Message: "error message"}))
	})

	t.Run("case insensitive with flag", func(t *testing.T) {
		filter := NewRegexFilterFromRegexp(regexp.MustCompile(`(?i)error`))
		assert.True(t, filter.Match(&domain.LogEntry{Message: "Error message"}))
		assert.True(t, filter.Match(&domain.LogEntry{Message: "ERROR message"}))
		assert.True(t, filter.Match(&domain.LogEntry{Message: "error message"}))
	})

	t.Run("string pattern constructor", func(t *testing.T) {
		filter, err := NewRegexFilter(`error\d+`)
		require.NoError(t, err)
		assert.True(t, filter.Match(&domain.LogEntry{Message: "error123"}))
		assert.False(t, filter.Match(&domain.LogEntry{Message: "error"}))
	})

	t.Run("invalid pattern returns error", func(t *testing.T) {
		_, err := NewRegexFilter(`[invalid`)
		assert.Error(t, err)
	})
}

func TestExcludePatternFilter(t *testing.T) {
	t.Run("excludes matching messages", func(t *testing.T) {
		filter, err := NewExcludePatternFilter(`heartbeat|keepalive`)
		require.NoError(t, err)

		assert.False(t, filter.Match(&domain.LogEntry{Message: "heartbeat ping"}))
		assert.False(t, filter.Match(&domain.LogEntry{Message: "keepalive check"}))
		assert.True(t, filter.Match(&domain.LogEntry{Message: "error occurred"}))
	})

	t.Run("invalid pattern returns error", func(t *testing.T) {
		_, err := NewExcludePatternFilter(`[invalid`)
		assert.Error(t, err)
	})
}

func TestExcludeSubsystemFilter(t *testing.T) {
	t.Run("empty list excludes nothing", func(t *testing.T) {
		filter := NewExcludeSubsystemFilter(nil)
		entry := &domain.LogEntry{Subsystem: "com.apple.network"}
		assert.True(t, filter.Match(entry))
	})

	t.Run("exact match exclusion", func(t *testing.T) {
		filter := NewExcludeSubsystemFilter([]string{"com.apple.network"})

		assert.False(t, filter.Match(&domain.LogEntry{Subsystem: "com.apple.network"}))
		assert.True(t, filter.Match(&domain.LogEntry{Subsystem: "com.apple.disk"}))
	})

	t.Run("wildcard exclusion", func(t *testing.T) {
		filter := NewExcludeSubsystemFilter([]string{"com.apple.*"})

		assert.False(t, filter.Match(&domain.LogEntry{Subsystem: "com.apple.network"}))
		assert.False(t, filter.Match(&domain.LogEntry{Subsystem: "com.apple.disk"}))
		assert.True(t, filter.Match(&domain.LogEntry{Subsystem: "com.example.app"}))
	})
}

func TestAppFilter(t *testing.T) {
	t.Run("empty bundle ID matches all", func(t *testing.T) {
		filter := NewAppFilter("")
		entry := &domain.LogEntry{Subsystem: "com.example.app"}
		assert.True(t, filter.Match(entry))
	})

	t.Run("matches subsystem prefix", func(t *testing.T) {
		filter := NewAppFilter("com.example.app")

		assert.True(t, filter.Match(&domain.LogEntry{Subsystem: "com.example.app"}))
		assert.True(t, filter.Match(&domain.LogEntry{Subsystem: "com.example.app.network"}))
		assert.False(t, filter.Match(&domain.LogEntry{Subsystem: "com.other.app"}))
	})
}

func TestCombinedFilters(t *testing.T) {
	// Real-world scenario: filter for errors from specific app, excluding heartbeats
	levelFilter := NewLevelFilter(domain.LogLevelError)
	appFilter := NewAppFilter("com.myapp")
	excludeFilter, _ := NewExcludePatternFilter(`heartbeat`)

	chain := NewChain(levelFilter, appFilter, excludeFilter)

	tests := []struct {
		name     string
		entry    domain.LogEntry
		expected bool
	}{
		{
			"passes all filters",
			domain.LogEntry{Level: domain.LogLevelError, Subsystem: "com.myapp", Message: "Connection failed"},
			true,
		},
		{
			"wrong level",
			domain.LogEntry{Level: domain.LogLevelDebug, Subsystem: "com.myapp", Message: "Connection failed"},
			false,
		},
		{
			"wrong app",
			domain.LogEntry{Level: domain.LogLevelError, Subsystem: "com.other", Message: "Connection failed"},
			false,
		},
		{
			"excluded pattern",
			domain.LogEntry{Level: domain.LogLevelError, Subsystem: "com.myapp", Message: "heartbeat timeout"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chain.Match(&tt.entry)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Benchmark tests
func BenchmarkLevelFilter(b *testing.B) {
	filter := NewLevelFilter(domain.LogLevelError)
	entry := &domain.LogEntry{Level: domain.LogLevelError}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.Match(entry)
	}
}

func BenchmarkChainFilter(b *testing.B) {
	chain := NewChain(
		NewLevelFilter(domain.LogLevelError),
		NewRegexFilterFromRegexp(regexp.MustCompile(`error`)),
	)
	entry := &domain.LogEntry{Level: domain.LogLevelError, Message: "error occurred"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chain.Match(entry)
	}
}

func TestNewFilters(t *testing.T) {
	t.Run("NewLevelFilter", func(t *testing.T) {
		f := NewLevelFilter(domain.LogLevelError)
		require.NotNil(t, f)
	})

	t.Run("NewRegexFilter", func(t *testing.T) {
		f, err := NewRegexFilter(`test`)
		require.NoError(t, err)
		require.NotNil(t, f)
	})

	t.Run("NewRegexFilterFromRegexp", func(t *testing.T) {
		f := NewRegexFilterFromRegexp(regexp.MustCompile(`test`))
		require.NotNil(t, f)
	})

	t.Run("NewAppFilter", func(t *testing.T) {
		f := NewAppFilter("com.test")
		require.NotNil(t, f)
	})

	t.Run("NewChain", func(t *testing.T) {
		f := NewChain()
		require.NotNil(t, f)
	})

	t.Run("NewOrChain", func(t *testing.T) {
		f := NewOrChain()
		require.NotNil(t, f)
	})
}

func TestWhereClause(t *testing.T) {
	t.Run("parse equals operator", func(t *testing.T) {
		wc, err := ParseWhereClause("level=error")
		require.NoError(t, err)
		assert.Equal(t, "level", wc.Field)
		assert.Equal(t, "=", wc.Operator)
		assert.Equal(t, "error", wc.Value)
	})

	t.Run("parse not equals operator", func(t *testing.T) {
		wc, err := ParseWhereClause("level!=debug")
		require.NoError(t, err)
		assert.Equal(t, "level", wc.Field)
		assert.Equal(t, "!=", wc.Operator)
		assert.Equal(t, "debug", wc.Value)
	})

	t.Run("parse contains operator", func(t *testing.T) {
		wc, err := ParseWhereClause("message~timeout")
		require.NoError(t, err)
		assert.Equal(t, "message", wc.Field)
		assert.Equal(t, "~", wc.Operator)
		assert.Equal(t, "timeout", wc.Value)
	})

	t.Run("parse not contains operator", func(t *testing.T) {
		wc, err := ParseWhereClause("message!~heartbeat")
		require.NoError(t, err)
		assert.Equal(t, "message", wc.Field)
		assert.Equal(t, "!~", wc.Operator)
		assert.Equal(t, "heartbeat", wc.Value)
	})

	t.Run("parse greater or equal operator", func(t *testing.T) {
		wc, err := ParseWhereClause("level>=error")
		require.NoError(t, err)
		assert.Equal(t, "level", wc.Field)
		assert.Equal(t, ">=", wc.Operator)
		assert.Equal(t, "error", wc.Value)
	})

	t.Run("parse starts with operator", func(t *testing.T) {
		wc, err := ParseWhereClause("subsystem^com.example")
		require.NoError(t, err)
		assert.Equal(t, "subsystem", wc.Field)
		assert.Equal(t, "^", wc.Operator)
		assert.Equal(t, "com.example", wc.Value)
	})

	t.Run("parse ends with operator", func(t *testing.T) {
		wc, err := ParseWhereClause("message$failed")
		require.NoError(t, err)
		assert.Equal(t, "message", wc.Field)
		assert.Equal(t, "$", wc.Operator)
		assert.Equal(t, "failed", wc.Value)
	})

	t.Run("invalid clause no operator", func(t *testing.T) {
		_, err := ParseWhereClause("levelxerror")
		assert.Error(t, err)
	})

	t.Run("invalid regex", func(t *testing.T) {
		_, err := ParseWhereClause("message~[invalid")
		assert.Error(t, err)
	})
}

func TestWhereClauseMatch(t *testing.T) {
	t.Run("equals level", func(t *testing.T) {
		wc, _ := ParseWhereClause("level=Error")
		entry := &domain.LogEntry{Level: domain.LogLevelError}
		assert.True(t, wc.Match(entry))

		entry2 := &domain.LogEntry{Level: domain.LogLevelDebug}
		assert.False(t, wc.Match(entry2))
	})

	t.Run("not equals", func(t *testing.T) {
		wc, _ := ParseWhereClause("level!=Debug")
		entry := &domain.LogEntry{Level: domain.LogLevelError}
		assert.True(t, wc.Match(entry))

		entry2 := &domain.LogEntry{Level: domain.LogLevelDebug}
		assert.False(t, wc.Match(entry2))
	})

	t.Run("contains regex", func(t *testing.T) {
		wc, _ := ParseWhereClause("message~timeout")
		entry := &domain.LogEntry{Message: "Connection timeout occurred"}
		assert.True(t, wc.Match(entry))

		entry2 := &domain.LogEntry{Message: "Success"}
		assert.False(t, wc.Match(entry2))
	})

	t.Run("not contains regex", func(t *testing.T) {
		wc, _ := ParseWhereClause("message!~heartbeat")
		entry := &domain.LogEntry{Message: "Error occurred"}
		assert.True(t, wc.Match(entry))

		entry2 := &domain.LogEntry{Message: "heartbeat ping"}
		assert.False(t, wc.Match(entry2))
	})

	t.Run("starts with", func(t *testing.T) {
		wc, _ := ParseWhereClause("subsystem^com.example")
		entry := &domain.LogEntry{Subsystem: "com.example.app"}
		assert.True(t, wc.Match(entry))

		entry2 := &domain.LogEntry{Subsystem: "com.other.app"}
		assert.False(t, wc.Match(entry2))
	})

		t.Run("ends with", func(t *testing.T) {
			wc, _ := ParseWhereClause("message$failed")
			entry := &domain.LogEntry{Message: "Connection failed"}
			assert.True(t, wc.Match(entry))

			entry2 := &domain.LogEntry{Message: "Success"}
			assert.False(t, wc.Match(entry2))
		})

		t.Run("numeric pid comparisons", func(t *testing.T) {
			wc, _ := ParseWhereClause("pid>=100")
			entry := &domain.LogEntry{PID: 120}
			assert.True(t, wc.Match(entry))
			entry2 := &domain.LogEntry{PID: 80}
			assert.False(t, wc.Match(entry2))

			wcEq, _ := ParseWhereClause("pid=120")
			assert.True(t, wcEq.Match(entry))
			assert.False(t, wcEq.Match(entry2))
		})

		t.Run("numeric tid comparisons", func(t *testing.T) {
			wc, _ := ParseWhereClause("tid<=5")
			entry := &domain.LogEntry{TID: 3}
			assert.True(t, wc.Match(entry))
			entry2 := &domain.LogEntry{TID: 7}
			assert.False(t, wc.Match(entry2))
		})

		t.Run("quoted where values", func(t *testing.T) {
			wc, err := ParseWhereClause(`message~"foo=bar"`)
			require.NoError(t, err)
			entry := &domain.LogEntry{Message: "foo=bar baz"}
			assert.True(t, wc.Match(entry))
		})

		t.Run("greater or equal level", func(t *testing.T) {
			wc, _ := ParseWhereClause("level>=error")

		assert.True(t, wc.Match(&domain.LogEntry{Level: domain.LogLevelError}))
		assert.True(t, wc.Match(&domain.LogEntry{Level: domain.LogLevelFault}))
		assert.False(t, wc.Match(&domain.LogEntry{Level: domain.LogLevelInfo}))
		assert.False(t, wc.Match(&domain.LogEntry{Level: domain.LogLevelDebug}))
	})

	t.Run("less or equal level", func(t *testing.T) {
		wc, _ := ParseWhereClause("level<=info")

		assert.True(t, wc.Match(&domain.LogEntry{Level: domain.LogLevelDebug}))
		assert.True(t, wc.Match(&domain.LogEntry{Level: domain.LogLevelInfo}))
		assert.False(t, wc.Match(&domain.LogEntry{Level: domain.LogLevelError}))
	})

	t.Run("match by process", func(t *testing.T) {
		wc, _ := ParseWhereClause("process=MyApp")
		entry := &domain.LogEntry{Process: "MyApp"}
		assert.True(t, wc.Match(entry))

		entry2 := &domain.LogEntry{Process: "OtherApp"}
		assert.False(t, wc.Match(entry2))
	})

	t.Run("match by category", func(t *testing.T) {
		wc, _ := ParseWhereClause("category=network")
		entry := &domain.LogEntry{Category: "network"}
		assert.True(t, wc.Match(entry))
	})

	t.Run("match by pid", func(t *testing.T) {
		wc, _ := ParseWhereClause("pid=1234")
		entry := &domain.LogEntry{PID: 1234}
		assert.True(t, wc.Match(entry))

		entry2 := &domain.LogEntry{PID: 5678}
		assert.False(t, wc.Match(entry2))
	})
}

func TestWhereFilter(t *testing.T) {
	t.Run("nil for empty clauses", func(t *testing.T) {
		f, err := NewWhereFilter(nil)
		require.NoError(t, err)
		assert.Nil(t, f)
	})

	t.Run("AND logic for multiple clauses", func(t *testing.T) {
		f, err := NewWhereFilter([]string{"level=Error", "message~timeout"})
		require.NoError(t, err)

		// Both conditions match
		entry1 := &domain.LogEntry{Level: domain.LogLevelError, Message: "Connection timeout"}
		assert.True(t, f.Match(entry1))

		// Level matches but message doesn't
		entry2 := &domain.LogEntry{Level: domain.LogLevelError, Message: "Success"}
		assert.False(t, f.Match(entry2))

		// Message matches but level doesn't
		entry3 := &domain.LogEntry{Level: domain.LogLevelDebug, Message: "Connection timeout"}
		assert.False(t, f.Match(entry3))
	})

	t.Run("invalid clause returns error", func(t *testing.T) {
		_, err := NewWhereFilter([]string{"invalid"})
		assert.Error(t, err)
	})
}

func TestDedupeFilter(t *testing.T) {
	t.Run("first occurrence always emits", func(t *testing.T) {
		f := NewDedupeFilter(0)
		entry := &domain.LogEntry{Message: "test message"}
		result := f.Check(entry)
		assert.True(t, result.ShouldEmit)
		assert.Equal(t, 1, result.Count)
	})

	t.Run("consecutive duplicates suppressed", func(t *testing.T) {
		f := NewDedupeFilter(0)
		entry := &domain.LogEntry{Message: "test message"}

		result1 := f.Check(entry)
		assert.True(t, result1.ShouldEmit)

		result2 := f.Check(entry)
		assert.False(t, result2.ShouldEmit)
		assert.Equal(t, 2, result2.Count)

		result3 := f.Check(entry)
		assert.False(t, result3.ShouldEmit)
		assert.Equal(t, 3, result3.Count)
	})

	t.Run("different message resets", func(t *testing.T) {
		f := NewDedupeFilter(0)
		entry1 := &domain.LogEntry{Message: "first message"}
		entry2 := &domain.LogEntry{Message: "second message"}

		result1 := f.Check(entry1)
		assert.True(t, result1.ShouldEmit)

		result2 := f.Check(entry1)
		assert.False(t, result2.ShouldEmit)

		// Different message should emit
		result3 := f.Check(entry2)
		assert.True(t, result3.ShouldEmit)
		assert.Equal(t, 1, result3.Count)

		// Going back to first message should emit again in consecutive mode
		result4 := f.Check(entry1)
		assert.True(t, result4.ShouldEmit)
	})

	t.Run("reset clears state", func(t *testing.T) {
		f := NewDedupeFilter(0)
		entry := &domain.LogEntry{Message: "test message"}

		f.Check(entry)
		f.Check(entry)

		f.Reset()

		result := f.Check(entry)
		assert.True(t, result.ShouldEmit)
		assert.Equal(t, 1, result.Count)
	})

	t.Run("get pending duplicates", func(t *testing.T) {
		f := NewDedupeFilter(0)
		entry := &domain.LogEntry{Message: "test message"}

		f.Check(entry)
		f.Check(entry)
		f.Check(entry)

		pending := f.GetPendingDuplicates()
		assert.Len(t, pending, 1)
		assert.Equal(t, 3, pending["test message"].count)
	})

	t.Run("windowed duplicates suppressed within window", func(t *testing.T) {
		f := NewDedupeFilter(5 * time.Second)
		entry1 := &domain.LogEntry{Message: "repeat", Timestamp: time.Unix(0, 0)}
		entry2 := &domain.LogEntry{Message: "repeat", Timestamp: time.Unix(3, 0)}
		entry3 := &domain.LogEntry{Message: "repeat", Timestamp: time.Unix(10, 0)}

		res1 := f.Check(entry1)
		assert.True(t, res1.ShouldEmit)
		res2 := f.Check(entry2)
		assert.False(t, res2.ShouldEmit)
		res3 := f.Check(entry3)
		assert.True(t, res3.ShouldEmit)
	})
}
