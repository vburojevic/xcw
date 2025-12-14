package simulator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStreamerBuildPredicate(t *testing.T) {
	tests := []struct {
		name     string
		opts     StreamOptions
		expected string
	}{
		{
			name:     "empty options",
			opts:     StreamOptions{},
			expected: "",
		},
		{
			name: "raw predicate overrides everything",
			opts: StreamOptions{
				RawPredicate: `processImagePath CONTAINS "MyApp"`,
				BundleID:     "com.example.app",
				Categories:   []string{"networking"},
			},
			expected: `processImagePath CONTAINS "MyApp"`,
		},
		{
			name: "bundle ID only",
			opts: StreamOptions{
				BundleID: "com.example.app",
			},
			expected: `subsystem BEGINSWITH "com.example.app"`,
		},
		{
			name: "bundle ID with quotes is escaped",
			opts: StreamOptions{
				BundleID: `com.example."app"`,
			},
			expected: `subsystem BEGINSWITH "com.example.\"app\""`,
		},
		{
			name: "single subsystem",
			opts: StreamOptions{
				Subsystems: []string{"com.apple.network"},
			},
			expected: `subsystem == "com.apple.network"`,
		},
		{
			name: "subsystem with backslashes is escaped",
			opts: StreamOptions{
				Subsystems: []string{`com.apple.network\\debug`},
			},
			expected: `subsystem == "com.apple.network\\\\debug"`,
		},
		{
			name: "bundle ID and subsystem (OR within group)",
			opts: StreamOptions{
				BundleID:   "com.example.app",
				Subsystems: []string{"com.apple.network"},
			},
			expected: `(subsystem BEGINSWITH "com.example.app" OR subsystem == "com.apple.network")`,
		},
		{
			name: "multiple subsystems (OR within group)",
			opts: StreamOptions{
				Subsystems: []string{"com.apple.network", "com.apple.security"},
			},
			expected: `(subsystem == "com.apple.network" OR subsystem == "com.apple.security")`,
		},
		{
			name: "single category",
			opts: StreamOptions{
				Categories: []string{"networking"},
			},
			expected: `category == "networking"`,
		},
		{
			name: "multiple categories (OR within group)",
			opts: StreamOptions{
				Categories: []string{"networking", "security"},
			},
			expected: `(category == "networking" OR category == "security")`,
		},
		{
			name: "bundle ID AND category (AND between groups)",
			opts: StreamOptions{
				BundleID:   "com.example.app",
				Categories: []string{"networking"},
			},
			expected: `subsystem BEGINSWITH "com.example.app" AND category == "networking"`,
		},
		{
			name: "subsystem AND category (AND between groups)",
			opts: StreamOptions{
				Subsystems: []string{"com.apple.network"},
				Categories: []string{"networking"},
			},
			expected: `subsystem == "com.apple.network" AND category == "networking"`,
		},
		{
			name: "complex: multiple subsystems AND multiple categories",
			opts: StreamOptions{
				BundleID:   "com.example.app",
				Subsystems: []string{"com.apple.network"},
				Categories: []string{"networking", "security"},
			},
			expected: `(subsystem BEGINSWITH "com.example.app" OR subsystem == "com.apple.network") AND (category == "networking" OR category == "security")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Streamer{opts: tt.opts}
			result := s.buildPredicate()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueryReaderBuildPredicate(t *testing.T) {
	tests := []struct {
		name     string
		opts     QueryOptions
		expected string
	}{
		{
			name:     "empty options",
			opts:     QueryOptions{},
			expected: "",
		},
		{
			name: "raw predicate overrides everything",
			opts: QueryOptions{
				RawPredicate: `processImagePath CONTAINS "MyApp"`,
				BundleID:     "com.example.app",
				Categories:   []string{"networking"},
			},
			expected: `processImagePath CONTAINS "MyApp"`,
		},
		{
			name: "bundle ID only",
			opts: QueryOptions{
				BundleID: "com.example.app",
			},
			expected: `subsystem BEGINSWITH "com.example.app"`,
		},
		{
			name: "bundle ID with quotes is escaped",
			opts: QueryOptions{
				BundleID: `com.example."app"`,
			},
			expected: `subsystem BEGINSWITH "com.example.\"app\""`,
		},
		{
			name: "bundle ID AND category (AND between groups)",
			opts: QueryOptions{
				BundleID:   "com.example.app",
				Categories: []string{"networking"},
			},
			expected: `subsystem BEGINSWITH "com.example.app" AND category == "networking"`,
		},
		{
			name: "complex: multiple subsystems AND multiple categories",
			opts: QueryOptions{
				BundleID:   "com.example.app",
				Subsystems: []string{"com.apple.network"},
				Categories: []string{"networking", "security"},
			},
			expected: `(subsystem BEGINSWITH "com.example.app" OR subsystem == "com.apple.network") AND (category == "networking" OR category == "security")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewQueryReader()
			result := r.buildPredicate(tt.opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}
