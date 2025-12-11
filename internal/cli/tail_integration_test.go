package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Basic compilation-level test to ensure emit helpers are wired.
// Full integration would require simulator, so we keep it minimal here.
func TestTailEmitHelperWiring(t *testing.T) {
	// ensure defaultHints is non-empty
	require.NotEmpty(t, defaultHints())
}
