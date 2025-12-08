package simulator

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/vedranburojevic/xcw/internal/domain"
)

// Manager handles simulator discovery and lifecycle operations
type Manager struct {
	xcrunPath    string
	pollInterval time.Duration
}

// NewManager creates a new simulator manager
func NewManager() *Manager {
	return &Manager{
		xcrunPath:    "xcrun",
		pollInterval: 2 * time.Second,
	}
}

// ListDevices returns all available simulators
func (m *Manager) ListDevices(ctx context.Context) ([]domain.Device, error) {
	cmd := exec.CommandContext(ctx, m.xcrunPath, "simctl", "list", "devices", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("simctl list failed: %w", err)
	}

	var resp domain.SimctlDevicesResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse simctl output: %w", err)
	}

	var devices []domain.Device
	for runtime, devs := range resp.Devices {
		for _, d := range devs {
			if !d.IsAvailable {
				continue
			}

			var lastBooted *time.Time
			if d.LastBootedAt != nil {
				if t, err := time.Parse(time.RFC3339, *d.LastBootedAt); err == nil {
					lastBooted = &t
				}
			}

			// Extract iOS version from runtime identifier
			// e.g., "com.apple.CoreSimulator.SimRuntime.iOS-17-0" -> "iOS 17.0"
			runtimeName := parseRuntimeName(runtime)

			devices = append(devices, domain.Device{
				UDID:                 d.UDID,
				Name:                 d.Name,
				State:                domain.DeviceState(d.State),
				IsAvailable:          d.IsAvailable,
				DeviceTypeIdentifier: d.DeviceTypeIdentifier,
				RuntimeIdentifier:    runtimeName,
				DataPath:             d.DataPath,
				LogPath:              d.LogPath,
				LastBootedAt:         lastBooted,
			})
		}
	}

	return devices, nil
}

// ListBootedDevices returns only booted simulators
func (m *Manager) ListBootedDevices(ctx context.Context) ([]domain.Device, error) {
	devices, err := m.ListDevices(ctx)
	if err != nil {
		return nil, err
	}

	var booted []domain.Device
	for _, d := range devices {
		if d.IsBooted() {
			booted = append(booted, d)
		}
	}
	return booted, nil
}

// FindDevice finds a device by name or UDID
func (m *Manager) FindDevice(ctx context.Context, nameOrUDID string) (*domain.Device, error) {
	devices, err := m.ListDevices(ctx)
	if err != nil {
		return nil, err
	}

	// Handle "booted" special case - return first booted device
	if strings.ToLower(nameOrUDID) == "booted" {
		for _, d := range devices {
			if d.IsBooted() {
				return &d, nil
			}
		}
		return nil, fmt.Errorf("no booted simulator found")
	}

	nameOrUDIDLower := strings.ToLower(nameOrUDID)

	// Exact match by UDID (case-insensitive)
	for _, d := range devices {
		if strings.ToLower(d.UDID) == nameOrUDIDLower {
			return &d, nil
		}
	}

	// Exact match by name (case-insensitive)
	for _, d := range devices {
		if strings.ToLower(d.Name) == nameOrUDIDLower {
			return &d, nil
		}
	}

	// Fuzzy match by name (contains)
	for _, d := range devices {
		if strings.Contains(strings.ToLower(d.Name), nameOrUDIDLower) {
			return &d, nil
		}
	}

	return nil, fmt.Errorf("device not found: %s", nameOrUDID)
}

// BootDevice boots a simulator by UDID
func (m *Manager) BootDevice(ctx context.Context, udid string) error {
	cmd := exec.CommandContext(ctx, m.xcrunPath, "simctl", "boot", udid)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if already booted
		if strings.Contains(string(output), "current state: Booted") {
			return nil // Already booted, not an error
		}
		return fmt.Errorf("failed to boot device: %s", string(output))
	}
	return nil
}

// ShutdownDevice shuts down a simulator by UDID
func (m *Manager) ShutdownDevice(ctx context.Context, udid string) error {
	cmd := exec.CommandContext(ctx, m.xcrunPath, "simctl", "shutdown", udid)
	return cmd.Run()
}

// GetDeviceInfo returns the current info for a device by UDID
func (m *Manager) GetDeviceInfo(ctx context.Context, udid string) (*domain.Device, error) {
	devices, err := m.ListDevices(ctx)
	if err != nil {
		return nil, err
	}

	for _, d := range devices {
		if d.UDID == udid {
			return &d, nil
		}
	}

	return nil, fmt.Errorf("device not found: %s", udid)
}

// WaitForBoot waits for a device to finish booting
func (m *Manager) WaitForBoot(ctx context.Context, udid string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for device to boot")
			}

			device, err := m.GetDeviceInfo(ctx, udid)
			if err != nil {
				continue
			}

			if device.IsBooted() {
				return nil
			}
		}
	}
}

// EnsureBooted boots a device if it's not already booted and waits for boot to complete
func (m *Manager) EnsureBooted(ctx context.Context, udid string) error {
	device, err := m.GetDeviceInfo(ctx, udid)
	if err != nil {
		return err
	}

	if device.IsBooted() {
		return nil
	}

	if err := m.BootDevice(ctx, udid); err != nil {
		return err
	}

	return m.WaitForBoot(ctx, udid, 60*time.Second)
}

// parseRuntimeName extracts a human-readable runtime name from the identifier
func parseRuntimeName(runtime string) string {
	// Example: "com.apple.CoreSimulator.SimRuntime.iOS-17-0" -> "iOS 17.0"
	// Example: "com.apple.CoreSimulator.SimRuntime.watchOS-10-0" -> "watchOS 10.0"

	parts := strings.Split(runtime, ".")
	if len(parts) == 0 {
		return runtime
	}

	lastPart := parts[len(parts)-1]

	// Replace dashes with dots for version numbers, but keep first part
	// e.g., "iOS-17-0" -> "iOS 17.0"
	segments := strings.Split(lastPart, "-")
	if len(segments) >= 2 {
		os := segments[0]
		version := strings.Join(segments[1:], ".")
		return fmt.Sprintf("%s %s", os, version)
	}

	return lastPart
}
