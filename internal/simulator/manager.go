package simulator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vburojevic/xcw/internal/domain"
	"howett.net/plist"
)

// Manager handles simulator discovery and lifecycle operations
type Manager struct {
	xcrunPath    string
	pollInterval time.Duration
	cacheTTL     time.Duration

	cacheMu       sync.Mutex
	cachedDevices []domain.Device
	cacheAt       time.Time
}

const (
	simctlListDevicesTimeout     = 10 * time.Second
	simctlBootTimeout            = 30 * time.Second
	simctlShutdownTimeout        = 30 * time.Second
	simctlGetAppContainerTimeout = 10 * time.Second
)

// NewManager creates a new simulator manager
func NewManager() *Manager {
	return &Manager{
		xcrunPath:    "xcrun",
		pollInterval: 2 * time.Second,
		cacheTTL:     2 * time.Second,
	}
}

// ListDevices returns all available simulators
func (m *Manager) ListDevices(ctx context.Context) ([]domain.Device, error) {
	// Serve from short-lived cache to avoid repeated simctl calls
	m.cacheMu.Lock()
	if m.cachedDevices != nil && time.Since(m.cacheAt) < m.cacheTTL {
		devs := make([]domain.Device, len(m.cachedDevices))
		copy(devs, m.cachedDevices)
		m.cacheMu.Unlock()
		return devs, nil
	}
	m.cacheMu.Unlock()

	cmdCtx, cancel := context.WithTimeout(ctx, simctlListDevicesTimeout)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, m.xcrunPath, "simctl", "list", "devices", "--json")
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

	// Update cache
	m.cacheMu.Lock()
	m.cachedDevices = devices
	m.cacheAt = time.Now()
	m.cacheMu.Unlock()

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

// MultipleBootedError indicates multiple simulators are booted
type MultipleBootedError struct {
	Devices []domain.Device
}

func (e *MultipleBootedError) Error() string {
	// Show most recently booted first, then stable by name/UDID.
	devs := make([]domain.Device, len(e.Devices))
	copy(devs, e.Devices)
	sort.SliceStable(devs, func(i, j int) bool {
		a := devs[i].LastBootedAt
		b := devs[j].LastBootedAt
		switch {
		case a == nil && b == nil:
			return devs[i].Name < devs[j].Name
		case a == nil:
			return false
		case b == nil:
			return true
		default:
			if a.Equal(*b) {
				return devs[i].Name < devs[j].Name
			}
			return a.After(*b)
		}
	})

	var names []string
	for _, d := range devs {
		names = append(names, fmt.Sprintf("%s (%s)", d.Name, d.UDID))
	}
	return fmt.Sprintf("multiple booted simulators found (most recent first):\n  %s\nUse --simulator to specify one (or run `xcw pick simulator`)", strings.Join(names, "\n  "))
}

// AmbiguousDeviceError is returned when a fuzzy search matches multiple devices
type AmbiguousDeviceError struct {
	Query   string
	Matches []domain.Device
}

func (e *AmbiguousDeviceError) Error() string {
	var names []string
	for _, d := range e.Matches {
		names = append(names, fmt.Sprintf("%s (%s)", d.Name, d.UDID))
	}
	return fmt.Sprintf("ambiguous device query %q matches multiple simulators:\n  %s\nBe more specific or use the full UDID", e.Query, strings.Join(names, "\n  "))
}

// FindBootedDevice finds a single booted device, errors if 0 or multiple
func (m *Manager) FindBootedDevice(ctx context.Context) (*domain.Device, error) {
	booted, err := m.ListBootedDevices(ctx)
	if err != nil {
		return nil, err
	}
	if len(booted) == 0 {
		return nil, fmt.Errorf("no booted simulator found")
	}
	if len(booted) > 1 {
		return nil, &MultipleBootedError{Devices: booted}
	}
	return &booted[0], nil
}

// FindDevice finds a device by name or UDID
func (m *Manager) FindDevice(ctx context.Context, nameOrUDID string) (*domain.Device, error) {
	devices, err := m.ListDevices(ctx)
	if err != nil {
		return nil, err
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

	// Fuzzy match by name (contains) - collect all matches
	var fuzzyMatches []domain.Device
	for _, d := range devices {
		if strings.Contains(strings.ToLower(d.Name), nameOrUDIDLower) {
			fuzzyMatches = append(fuzzyMatches, d)
		}
	}

	if len(fuzzyMatches) == 1 {
		return &fuzzyMatches[0], nil
	}
	if len(fuzzyMatches) > 1 {
		return nil, &AmbiguousDeviceError{Query: nameOrUDID, Matches: fuzzyMatches}
	}

	return nil, fmt.Errorf("device not found: %s", nameOrUDID)
}

// BootDevice boots a simulator by UDID
func (m *Manager) BootDevice(ctx context.Context, udid string) error {
	cmdCtx, cancel := context.WithTimeout(ctx, simctlBootTimeout)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, m.xcrunPath, "simctl", "boot", udid)
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
	cmdCtx, cancel := context.WithTimeout(ctx, simctlShutdownTimeout)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, m.xcrunPath, "simctl", "shutdown", udid)
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

// GetAppInfo returns version/build for an installed app (best-effort)
func (m *Manager) GetAppInfo(ctx context.Context, udid, bundleID string) (version, build string, err error) {
	if bundleID == "" {
		return "", "", fmt.Errorf("bundle ID required")
	}

	// Get app container path
	cmdCtx, cancel := context.WithTimeout(ctx, simctlGetAppContainerTimeout)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, m.xcrunPath, "simctl", "get_app_container", udid, bundleID, "--app")
	containerPathBytes, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("get_app_container failed: %w", err)
	}

	containerPath := strings.TrimSpace(string(containerPathBytes))
	infoPlist := filepath.Join(containerPath, "Info.plist")

	var data map[string]interface{}
	raw, err := os.ReadFile(infoPlist)
	if err != nil {
		return "", "", fmt.Errorf("read Info.plist: %w", err)
	}
	if _, err := plist.Unmarshal(raw, &data); err != nil {
		return "", "", fmt.Errorf("parse Info.plist: %w", err)
	}

	if v, ok := data["CFBundleShortVersionString"].(string); ok {
		version = v
	}
	if b, ok := data["CFBundleVersion"].(string); ok {
		build = b
	}

	return version, build, nil
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
