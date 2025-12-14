package simulator

import "testing"

func TestMatchProcess(t *testing.T) {
	tests := []struct {
		name     string
		process  string
		patterns []string
		want     bool
	}{
		{name: "exact match", process: "MyApp", patterns: []string{"MyApp"}, want: true},
		{name: "glob match", process: "MyAppExtension", patterns: []string{"MyApp*"}, want: true},
		{name: "regex re: match", process: "MyAppExtension", patterns: []string{`re:^MyApp.*$`}, want: true},
		{name: "regex /pat/ match", process: "MyAppExtension", patterns: []string{`/^MyApp.*$/`}, want: true},
		{name: "invalid regex ignored", process: "MyApp", patterns: []string{`re:(`}, want: false},
		{name: "empty patterns", process: "MyApp", patterns: []string{""}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchProcess(tt.process, tt.patterns); got != tt.want {
				t.Fatalf("matchProcess(%q, %v) = %v, want %v", tt.process, tt.patterns, got, tt.want)
			}
		})
	}
}
