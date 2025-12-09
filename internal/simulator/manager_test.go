package simulator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vburojevic/xcw/internal/domain"
)

func TestMultipleBootedError(t *testing.T) {
	t.Run("formats single device correctly", func(t *testing.T) {
		err := &MultipleBootedError{
			Devices: []domain.Device{
				{Name: "iPhone 15", UDID: "ABC123"},
			},
		}

		msg := err.Error()
		assert.Contains(t, msg, "multiple booted simulators found")
		assert.Contains(t, msg, "iPhone 15 (ABC123)")
		assert.Contains(t, msg, "Use --simulator to specify one")
	})

	t.Run("formats multiple devices correctly", func(t *testing.T) {
		err := &MultipleBootedError{
			Devices: []domain.Device{
				{Name: "iPhone 15", UDID: "ABC123"},
				{Name: "iPhone 14 Pro", UDID: "DEF456"},
				{Name: "iPad Air", UDID: "GHI789"},
			},
		}

		msg := err.Error()
		assert.Contains(t, msg, "multiple booted simulators found")
		assert.Contains(t, msg, "iPhone 15 (ABC123)")
		assert.Contains(t, msg, "iPhone 14 Pro (DEF456)")
		assert.Contains(t, msg, "iPad Air (GHI789)")
		assert.Contains(t, msg, "Use --simulator to specify one")

		// Verify devices are on separate lines
		lines := strings.Split(msg, "\n")
		assert.GreaterOrEqual(t, len(lines), 3)
	})

	t.Run("implements error interface", func(t *testing.T) {
		var err error = &MultipleBootedError{
			Devices: []domain.Device{
				{Name: "iPhone 15", UDID: "ABC123"},
			},
		}

		// Should be assignable to error interface
		assert.NotNil(t, err)
		assert.NotEmpty(t, err.Error())
	})
}

func TestNewManager(t *testing.T) {
	t.Run("creates manager with default values", func(t *testing.T) {
		mgr := NewManager()
		assert.NotNil(t, mgr)
		assert.Equal(t, "xcrun", mgr.xcrunPath)
		assert.NotZero(t, mgr.pollInterval)
	})
}

func TestParseRuntimeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"com.apple.CoreSimulator.SimRuntime.iOS-17-0", "iOS 17.0"},
		{"com.apple.CoreSimulator.SimRuntime.iOS-17-2", "iOS 17.2"},
		{"com.apple.CoreSimulator.SimRuntime.iOS-18-0", "iOS 18.0"},
		{"com.apple.CoreSimulator.SimRuntime.watchOS-10-0", "watchOS 10.0"},
		{"com.apple.CoreSimulator.SimRuntime.tvOS-17-0", "tvOS 17.0"},
		{"com.apple.CoreSimulator.SimRuntime.visionOS-1-0", "visionOS 1.0"},
		{"iOS-17-0", "iOS 17.0"},
		{"simple", "simple"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseRuntimeName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
