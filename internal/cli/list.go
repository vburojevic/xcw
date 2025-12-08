package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vedranburojevic/xcw/internal/domain"
	"github.com/vedranburojevic/xcw/internal/simulator"
)

// ListCmd lists available simulators
type ListCmd struct {
	BootedOnly bool   `short:"b" help:"Show only booted simulators"`
	Runtime    string `help:"Filter by iOS runtime version (e.g., '17', 'iOS 17')"`
}

// Run executes the list command
func (c *ListCmd) Run(globals *Globals) error {
	ctx := context.Background()
	mgr := simulator.NewManager()

	var devices []domain.Device
	var err error

	if c.BootedOnly {
		devices, err = mgr.ListBootedDevices(ctx)
	} else {
		devices, err = mgr.ListDevices(ctx)
	}

	if err != nil {
		return c.outputError(globals, "LIST_FAILED", err.Error())
	}

	// Filter by runtime if specified
	if c.Runtime != "" {
		devices = filterByRuntime(devices, c.Runtime)
	}

	// Output results
	if globals.Format == "ndjson" {
		return c.outputNDJSON(globals, devices)
	}
	return c.outputText(globals, devices)
}

func (c *ListCmd) outputNDJSON(globals *Globals, devices []domain.Device) error {
	encoder := json.NewEncoder(globals.Stdout)
	for _, d := range devices {
		if err := encoder.Encode(d); err != nil {
			return err
		}
	}
	return nil
}

func (c *ListCmd) outputText(globals *Globals, devices []domain.Device) error {
	if len(devices) == 0 {
		fmt.Fprintln(globals.Stdout, "No simulators found")
		return nil
	}

	// Print header
	fmt.Fprintf(globals.Stdout, "%-40s %-12s %-15s %s\n", "NAME", "STATE", "RUNTIME", "UDID")
	fmt.Fprintln(globals.Stdout, strings.Repeat("-", 100))

	for _, d := range devices {
		stateIndicator := " "
		if d.IsBooted() {
			stateIndicator = "*"
		}
		fmt.Fprintf(globals.Stdout, "%-40s %s%-11s %-15s %s\n",
			truncate(d.Name, 40),
			stateIndicator,
			d.State,
			d.RuntimeIdentifier,
			d.UDID,
		)
	}

	// Print summary
	bootedCount := 0
	for _, d := range devices {
		if d.IsBooted() {
			bootedCount++
		}
	}
	fmt.Fprintf(globals.Stdout, "\n%d simulator(s), %d booted\n", len(devices), bootedCount)

	return nil
}

func (c *ListCmd) outputError(globals *Globals, code, message string) error {
	if globals.Format == "ndjson" {
		errOutput := domain.NewErrorOutput(code, message)
		encoder := json.NewEncoder(globals.Stdout)
		return encoder.Encode(errOutput)
	}
	fmt.Fprintf(globals.Stderr, "Error: %s\n", message)
	return fmt.Errorf(message)
}

func filterByRuntime(devices []domain.Device, runtime string) []domain.Device {
	runtime = strings.ToLower(runtime)
	var filtered []domain.Device
	for _, d := range devices {
		if strings.Contains(strings.ToLower(d.RuntimeIdentifier), runtime) {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
