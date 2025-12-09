package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"

	"github.com/vburojevic/xcw/internal/domain"
	"github.com/vburojevic/xcw/internal/output"
	"github.com/vburojevic/xcw/internal/simulator"
	"howett.net/plist"
)

// AppsCmd lists installed apps on a simulator
type AppsCmd struct {
	Simulator string `short:"s" help:"Simulator name or UDID"`
	Booted    bool   `short:"b" help:"Use booted simulator (error if multiple)"`
	UserOnly  bool   `help:"Show only user-installed apps (exclude system apps)"`
}

// appInfo represents information about an installed app
type appInfo struct {
	BundleID    string `json:"bundle_id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	BuildNumber string `json:"build_number,omitempty"`
	Path        string `json:"path,omitempty"`
	DataPath    string `json:"data_path,omitempty"`
	Type        string `json:"type"` // "user" or "system"
}

// plistAppInfo is the structure from simctl listapps plist output
type plistAppInfo struct {
	ApplicationType      string `plist:"ApplicationType"`
	Bundle               string `plist:"Bundle"`
	BundleIdentifier     string `plist:"CFBundleIdentifier"`
	BundleName           string `plist:"CFBundleName"`
	BundleDisplayName    string `plist:"CFBundleDisplayName"`
	BundleVersion        string `plist:"CFBundleVersion"`
	BundleShortVersion   string `plist:"CFBundleShortVersionString"`
	Path                 string `plist:"Path"`
	DataContainer        string `plist:"DataContainer"`
}

// Run executes the apps command
func (c *AppsCmd) Run(globals *Globals) error {
	ctx := context.Background()

	// Validate mutual exclusivity of flags
	if c.Simulator != "" && c.Booted {
		return c.outputError(globals, "INVALID_FLAGS", "--simulator and --booted are mutually exclusive")
	}

	// Find the simulator
	mgr := simulator.NewManager()
	var device *domain.Device
	var err error

	if c.Simulator != "" {
		device, err = mgr.FindDevice(ctx, c.Simulator)
	} else {
		device, err = mgr.FindBootedDevice(ctx)
	}
	if err != nil {
		return c.outputError(globals, "DEVICE_NOT_FOUND", err.Error())
	}

	// Ensure device is booted for listapps to work
	if !device.IsBooted() {
		return c.outputError(globals, "DEVICE_NOT_BOOTED",
			fmt.Sprintf("device %s is not booted; boot with: xcrun simctl boot %s", device.Name, device.UDID))
	}

	// Get installed apps
	apps, err := c.getInstalledApps(ctx, device.UDID)
	if err != nil {
		return c.outputError(globals, "LIST_APPS_FAILED", err.Error())
	}

	// Filter if user-only requested
	if c.UserOnly {
		var userApps []appInfo
		for _, app := range apps {
			if app.Type == "user" {
				userApps = append(userApps, app)
			}
		}
		apps = userApps
	}

	// Sort by bundle ID
	sort.Slice(apps, func(i, j int) bool {
		return apps[i].BundleID < apps[j].BundleID
	})

	// Output
	if globals.Format == "ndjson" {
		encoder := json.NewEncoder(globals.Stdout)
		for _, app := range apps {
			entry := map[string]interface{}{
				"type":       "app",
				"bundle_id":  app.BundleID,
				"name":       app.Name,
				"version":    app.Version,
				"app_type":   app.Type,
			}
			if app.BuildNumber != "" {
				entry["build_number"] = app.BuildNumber
			}
			if app.Path != "" {
				entry["path"] = app.Path
			}
			encoder.Encode(entry)
		}

		// Summary
		summary := map[string]interface{}{
			"type":       "apps_summary",
			"device":     device.Name,
			"udid":       device.UDID,
			"total":      len(apps),
		}
		encoder.Encode(summary)
	} else {
		// Text output
		if !globals.Quiet {
			fmt.Fprintf(globals.Stdout, "Installed apps on %s (%s)\n\n", device.Name, device.UDID)
		}

		for _, app := range apps {
			fmt.Fprintf(globals.Stdout, "%-50s %s (%s)\n", app.BundleID, app.Name, app.Version)
		}

		if !globals.Quiet {
			fmt.Fprintf(globals.Stdout, "\nTotal: %d apps\n", len(apps))
		}
	}

	return nil
}

func (c *AppsCmd) getInstalledApps(ctx context.Context, udid string) ([]appInfo, error) {
	cmd := exec.CommandContext(ctx, "xcrun", "simctl", "listapps", udid)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("simctl listapps failed: %w", err)
	}

	// Parse plist output
	var appsDict map[string]plistAppInfo
	_, err = plist.Unmarshal(output, &appsDict)
	if err != nil {
		return nil, fmt.Errorf("failed to parse apps plist: %w", err)
	}

	var apps []appInfo
	for bundleID, info := range appsDict {
		name := info.BundleDisplayName
		if name == "" {
			name = info.BundleName
		}
		if name == "" {
			name = bundleID
		}

		version := info.BundleShortVersion
		if version == "" {
			version = info.BundleVersion
		}

		appType := "system"
		if info.ApplicationType == "User" {
			appType = "user"
		}

		apps = append(apps, appInfo{
			BundleID:    bundleID,
			Name:        name,
			Version:     version,
			BuildNumber: info.BundleVersion,
			Path:        info.Path,
			DataPath:    info.DataContainer,
			Type:        appType,
		})
	}

	return apps, nil
}

func (c *AppsCmd) outputError(globals *Globals, code, message string) error {
	if globals.Format == "ndjson" {
		w := output.NewNDJSONWriter(globals.Stdout)
		w.WriteError(code, message)
	} else {
		fmt.Fprintf(globals.Stderr, "Error [%s]: %s\n", code, message)
	}
	return errors.New(message)
}
