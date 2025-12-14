package cli

import (
	"testing"

	"github.com/vburojevic/xcw/internal/domain"
)

func TestBuildFilters_Valid(t *testing.T) {
	pat, exclude, where, err := buildFilters("foo", []string{"bar"}, []string{"level=error"})
	if err != nil {
		t.Fatalf("buildFilters returned error: %v", err)
	}
	if pat == nil {
		t.Fatal("pattern should be compiled")
	}
	if len(exclude) != 1 {
		t.Fatalf("expected 1 exclude pattern, got %d", len(exclude))
	}
	if where == nil {
		t.Fatal("where filter should be compiled")
	}
}

func TestBuildFilters_InvalidPattern(t *testing.T) {
	if _, _, _, err := buildFilters("[[", nil, nil); err == nil {
		t.Fatal("expected regex compile error")
	}
}

func TestResolveLevels(t *testing.T) {
	min, max := resolveLevels("info", "error", "debug")
	if min != domain.LogLevelInfo {
		t.Fatalf("expected min level info, got %s", min)
	}
	if max != domain.LogLevelError {
		t.Fatalf("expected max level error, got %s", max)
	}

	min, max = resolveLevels("", "", "default")
	if min != domain.LogLevelDefault {
		t.Fatalf("expected min level default, got %s", min)
	}
	if max != "" {
		t.Fatalf("expected no max level when not set, got %s", max)
	}
}
