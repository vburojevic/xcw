package filter

import (
	"regexp"
	"testing"

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
