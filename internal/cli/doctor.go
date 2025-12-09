package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vburojevic/xcw/internal/config"
	"github.com/vburojevic/xcw/internal/simulator"
)

// DoctorCmd checks system requirements and configuration
type DoctorCmd struct{}

// checkResult represents a single diagnostic check
type checkResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "warning", "error"
	Message string `json:"message,omitempty"`
	Details string `json:"details,omitempty"`
}

// doctorReport is the complete diagnostic report
type doctorReport struct {
	Type       string        `json:"type"`
	Timestamp  string        `json:"timestamp"`
	Checks     []checkResult `json:"checks"`
	AllPassed  bool          `json:"all_passed"`
	ErrorCount int           `json:"error_count"`
	WarnCount  int           `json:"warn_count"`
}

// Run executes the doctor command
func (c *DoctorCmd) Run(globals *Globals) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var checks []checkResult

	// Check xcrun
	checks = append(checks, c.checkXcrun(ctx))

	// Check simctl
	checks = append(checks, c.checkSimctl(ctx))

	// Check Xcode
	checks = append(checks, c.checkXcode(ctx))

	// Check tmux
	checks = append(checks, c.checkTmux())

	// Check config file
	checks = append(checks, c.checkConfig())

	// Check simulators
	checks = append(checks, c.checkSimulators(ctx))

	// Count errors and warnings
	errorCount := 0
	warnCount := 0
	for _, check := range checks {
		if check.Status == "error" {
			errorCount++
		} else if check.Status == "warning" {
			warnCount++
		}
	}

	report := doctorReport{
		Type:       "doctor",
		Timestamp:  time.Now().Format(time.RFC3339),
		Checks:     checks,
		AllPassed:  errorCount == 0,
		ErrorCount: errorCount,
		WarnCount:  warnCount,
	}

	if globals.Format == "ndjson" {
		encoder := json.NewEncoder(globals.Stdout)
		return encoder.Encode(report)
	}

	// Text output
	fmt.Fprintln(globals.Stdout, "xcw Doctor")
	fmt.Fprintln(globals.Stdout, "==========")
	fmt.Fprintln(globals.Stdout)

	for _, check := range checks {
		var icon string
		switch check.Status {
		case "ok":
			icon = "✓"
		case "warning":
			icon = "⚠"
		case "error":
			icon = "✗"
		}

		fmt.Fprintf(globals.Stdout, "%s %s\n", icon, check.Name)
		if check.Message != "" {
			fmt.Fprintf(globals.Stdout, "  %s\n", check.Message)
		}
		if check.Details != "" {
			fmt.Fprintf(globals.Stdout, "  %s\n", check.Details)
		}
	}

	fmt.Fprintln(globals.Stdout)
	if errorCount == 0 && warnCount == 0 {
		fmt.Fprintln(globals.Stdout, "All checks passed!")
	} else {
		fmt.Fprintf(globals.Stdout, "Errors: %d, Warnings: %d\n", errorCount, warnCount)
	}

	return nil
}

func (c *DoctorCmd) checkXcrun(ctx context.Context) checkResult {
	cmd := exec.CommandContext(ctx, "xcrun", "--version")
	output, err := cmd.Output()
	if err != nil {
		return checkResult{
			Name:    "xcrun",
			Status:  "error",
			Message: "xcrun not found or not working",
			Details: "Install Xcode Command Line Tools: xcode-select --install",
		}
	}

	version := strings.TrimSpace(string(output))
	return checkResult{
		Name:    "xcrun",
		Status:  "ok",
		Message: version,
	}
}

func (c *DoctorCmd) checkSimctl(ctx context.Context) checkResult {
	cmd := exec.CommandContext(ctx, "xcrun", "simctl", "help")
	if err := cmd.Run(); err != nil {
		return checkResult{
			Name:    "simctl",
			Status:  "error",
			Message: "simctl not accessible",
			Details: "Ensure Xcode is properly installed",
		}
	}

	return checkResult{
		Name:    "simctl",
		Status:  "ok",
		Message: "simctl available",
	}
}

func (c *DoctorCmd) checkXcode(ctx context.Context) checkResult {
	cmd := exec.CommandContext(ctx, "xcode-select", "-p")
	output, err := cmd.Output()
	if err != nil {
		return checkResult{
			Name:    "Xcode",
			Status:  "error",
			Message: "Xcode not found",
			Details: "Install Xcode from the App Store or run: xcode-select --install",
		}
	}

	path := strings.TrimSpace(string(output))

	// Check if it's the full Xcode or just command line tools
	// Xcode path patterns: Xcode.app, Xcode-16.0.app, Xcode-beta.app, etc.
	if strings.Contains(path, "Xcode") && strings.Contains(path, ".app") {
		// Get Xcode version
		versionCmd := exec.CommandContext(ctx, "xcodebuild", "-version")
		versionOutput, _ := versionCmd.Output()
		version := strings.Split(strings.TrimSpace(string(versionOutput)), "\n")[0]

		return checkResult{
			Name:    "Xcode",
			Status:  "ok",
			Message: version,
			Details: path,
		}
	}

	return checkResult{
		Name:    "Xcode",
		Status:  "warning",
		Message: "Only Command Line Tools installed",
		Details: "Full Xcode is recommended for simulator support: " + path,
	}
}

func (c *DoctorCmd) checkTmux() checkResult {
	path, err := exec.LookPath("tmux")
	if err != nil {
		return checkResult{
			Name:    "tmux",
			Status:  "warning",
			Message: "tmux not found (optional)",
			Details: "Install with: brew install tmux",
		}
	}

	cmd := exec.Command("tmux", "-V")
	output, _ := cmd.Output()
	version := strings.TrimSpace(string(output))

	return checkResult{
		Name:    "tmux",
		Status:  "ok",
		Message: version,
		Details: path,
	}
}

func (c *DoctorCmd) checkConfig() checkResult {
	configPath := config.ConfigFile()
	if configPath == "" {
		return checkResult{
			Name:    "Config",
			Status:  "ok",
			Message: "Using defaults (no config file)",
			Details: "Create with: xcw config generate > ~/.xcw.yaml",
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return checkResult{
			Name:    "Config",
			Status:  "error",
			Message: "Config file has errors",
			Details: err.Error(),
		}
	}

	// Config loaded successfully
	absPath, _ := filepath.Abs(configPath)
	return checkResult{
		Name:    "Config",
		Status:  "ok",
		Message: fmt.Sprintf("Loaded from: %s", absPath),
		Details: fmt.Sprintf("Format: %s, Level: %s", cfg.Format, cfg.Level),
	}
}

func (c *DoctorCmd) checkSimulators(ctx context.Context) checkResult {
	mgr := simulator.NewManager()
	devices, err := mgr.ListDevices(ctx)
	if err != nil {
		return checkResult{
			Name:    "Simulators",
			Status:  "error",
			Message: "Failed to list simulators",
			Details: err.Error(),
		}
	}

	if len(devices) == 0 {
		return checkResult{
			Name:    "Simulators",
			Status:  "warning",
			Message: "No simulators found",
			Details: "Create simulators in Xcode > Window > Devices and Simulators",
		}
	}

	booted := 0
	for _, d := range devices {
		if d.IsBooted() {
			booted++
		}
	}

	// Group by runtime for summary
	runtimes := make(map[string]int)
	for _, d := range devices {
		runtimes[d.RuntimeIdentifier]++
	}

	var runtimeList []string
	for rt, count := range runtimes {
		runtimeList = append(runtimeList, fmt.Sprintf("%s (%d)", rt, count))
	}

	return checkResult{
		Name:    "Simulators",
		Status:  "ok",
		Message: fmt.Sprintf("%d available, %d booted", len(devices), booted),
		Details: strings.Join(runtimeList, ", "),
	}
}

// checkWritePermission checks if we can write to a directory
func (c *DoctorCmd) checkWritePermission(path string) bool {
	testFile := filepath.Join(path, ".xcw_test_"+fmt.Sprint(os.Getpid()))
	f, err := os.Create(testFile)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(testFile)
	return true
}
