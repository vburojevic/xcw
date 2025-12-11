package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/vburojevic/xcw/internal/domain"
	"github.com/vburojevic/xcw/internal/output"
	"github.com/vburojevic/xcw/internal/simulator"
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
		fmt.Fprintln(globals.Stdout, output.Styles.Warning.Render("No simulators found"))
		return nil
	}

	// Create table with options for clean output
	table := tablewriter.NewTable(globals.Stdout,
		tablewriter.WithHeader([]string{"NAME", "STATE", "RUNTIME", "UDID"}),
		tablewriter.WithBorders(tw.Border{
			Left:   tw.Off,
			Right:  tw.Off,
			Top:    tw.Off,
			Bottom: tw.Off,
		}),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
	)

	bootedCount := 0
	for _, d := range devices {
		stateStr := string(d.State)
		if d.IsBooted() {
			stateStr = "● " + stateStr
			bootedCount++
		} else {
			stateStr = "○ " + stateStr
		}

		table.Append([]string{
			truncate(d.Name, 35),
			stateStr,
			d.RuntimeIdentifier,
			d.UDID,
		})
	}

	if err := table.Render(); err != nil {
		return err
	}

	// Print summary with styling
	fmt.Fprintf(globals.Stdout, "\n%s %s, %s\n",
		output.Styles.Label.Render("Total:"),
		output.Styles.Value.Render(fmt.Sprintf("%d simulator(s)", len(devices))),
		output.Styles.Success.Render(fmt.Sprintf("%d booted", bootedCount)),
	)

	return nil
}

func (c *ListCmd) outputError(globals *Globals, code, message string) error {
	return outputErrorCommon(globals, code, message)
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
