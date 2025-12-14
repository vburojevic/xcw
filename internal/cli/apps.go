package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"time"

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
	ApplicationType    string `plist:"ApplicationType"`
	Bundle             string `plist:"Bundle"`
	BundleIdentifier   string `plist:"CFBundleIdentifier"`
	BundleName         string `plist:"CFBundleName"`
	BundleDisplayName  string `plist:"CFBundleDisplayName"`
	BundleVersion      string `plist:"CFBundleVersion"`
	BundleShortVersion string `plist:"CFBundleShortVersionString"`
	Path               string `plist:"Path"`
	DataContainer      string `plist:"DataContainer"`
}

// Run executes the apps command
func (c *AppsCmd) Run(globals *Globals) error {
	ctx := context.Background()

	// Validate mutual exclusivity of flags
	if globals.FlagProvided("simulator") && globals.FlagProvided("booted") {
		return c.outputError(globals, "INVALID_FLAGS", "--simulator and --booted are mutually exclusive")
	}

	// Find the simulator
	mgr := simulator.NewManager()
	device, err := resolveSimulatorDevice(ctx, mgr, c.Simulator, c.Booted)
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
				"type":          "app",
				"schemaVersion": output.SchemaVersion,
				"bundle_id":     app.BundleID,
				"name":          app.Name,
				"version":       app.Version,
				"app_type":      app.Type,
			}
			if app.BuildNumber != "" {
				entry["build_number"] = app.BuildNumber
			}
			if app.Path != "" {
				entry["path"] = app.Path
			}
			if err := encoder.Encode(entry); err != nil {
				return err
			}
		}

		// Summary
		summary := map[string]interface{}{
			"type":          "apps_summary",
			"schemaVersion": output.SchemaVersion,
			"device":        device.Name,
			"udid":          device.UDID,
			"total":         len(apps),
		}
		if err := encoder.Encode(summary); err != nil {
			return err
		}
	} else {
		// Text output
		if !globals.Quiet {
			if _, err := fmt.Fprintf(globals.Stdout, "Installed apps on %s (%s)\n\n", device.Name, device.UDID); err != nil {
				return err
			}
		}

		for _, app := range apps {
			if _, err := fmt.Fprintf(globals.Stdout, "%-50s %s (%s)\n", app.BundleID, app.Name, app.Version); err != nil {
				return err
			}
		}

		if !globals.Quiet {
			if _, err := fmt.Fprintf(globals.Stdout, "\nTotal: %d apps\n", len(apps)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *AppsCmd) getInstalledApps(ctx context.Context, udid string) ([]appInfo, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, "xcrun", "simctl", "listapps", udid)
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
	return outputErrorCommon(globals, code, message)
}
