package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/vburojevic/xcw/internal/domain"
	"github.com/vburojevic/xcw/internal/output"
	"github.com/vburojevic/xcw/internal/simulator"
)

// PickCmd interactively picks a simulator or app
type PickCmd struct {
	Type      string `arg:"" enum:"simulator,app" help:"What to pick: simulator or app"`
	Simulator string `short:"s" help:"Simulator for app picking (uses booted if omitted)"`
	UserOnly  bool   `help:"Show only user-installed apps (for app picking)"`
}

// pickItem implements list.Item for the picker
type pickItem struct {
	id          string // UDID or bundle_id
	title       string // Display name
	description string // Additional info
}

func (i pickItem) Title() string       { return i.title }
func (i pickItem) Description() string { return i.description }
func (i pickItem) FilterValue() string { return i.title + " " + i.id }

// pickModel is the bubbletea model for the picker
type pickModel struct {
	list     list.Model
	selected pickItem
	quitting bool
	canceled bool
}

func (m pickModel) Init() tea.Cmd {
	return nil
}

func (m pickModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if item, ok := m.list.SelectedItem().(pickItem); ok {
				m.selected = item
				m.quitting = true
				return m, tea.Quit
			}
		case "q", "esc", "ctrl+c":
			m.canceled = true
			m.quitting = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2)
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m pickModel) View() string {
	if m.quitting {
		return ""
	}
	return m.list.View()
}

// Run executes the pick command
func (c *PickCmd) Run(globals *Globals) error {
	// Require interactive terminal
	if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		return c.outputError(globals, "NOT_INTERACTIVE",
			"xcw pick requires an interactive terminal. "+
				"Use --simulator and --app flags directly, or use 'xcw list' and 'xcw apps' for scripting.")
	}

	ctx := context.Background()

	switch c.Type {
	case "simulator":
		return c.pickSimulator(ctx, globals)
	case "app":
		return c.pickApp(ctx, globals)
	default:
		return fmt.Errorf("unknown pick type: %s", c.Type)
	}
}

func (c *PickCmd) pickSimulator(ctx context.Context, globals *Globals) error {
	mgr := simulator.NewManager()
	devices, err := mgr.ListDevices(ctx)
	if err != nil {
		return c.outputError(globals, "LIST_FAILED", err.Error())
	}

	if len(devices) == 0 {
		return c.outputError(globals, "NO_SIMULATORS", "No simulators available")
	}

	// Convert devices to pick items
	items := make([]list.Item, 0, len(devices))
	for _, d := range devices {
		state := ""
		if d.IsBooted() {
			state = " (booted)"
		}
		items = append(items, pickItem{
			id:          d.UDID,
			title:       d.Name + state,
			description: d.RuntimeIdentifier,
		})
	}

	// Run picker
	selected, err := c.runPicker(items, "Select Simulator")
	if err != nil {
		return err
	}

	// Output result
	return c.outputResult(globals, "simulator", selected.id, selected.title, "")
}

func (c *PickCmd) pickApp(ctx context.Context, globals *Globals) error {
	mgr := simulator.NewManager()

	// Find simulator for app listing
	var device *domain.Device
	var err error

	device, err = resolveSimulatorDevice(ctx, mgr, c.Simulator, false)
	if err != nil {
		return c.outputError(globals, "DEVICE_NOT_FOUND", err.Error())
	}

	if !device.IsBooted() {
		return c.outputError(globals, "DEVICE_NOT_BOOTED",
			fmt.Sprintf("device %s is not booted; boot with: xcrun simctl boot %s", device.Name, device.UDID))
	}

	// Get installed apps (reuse AppsCmd logic)
	appsCmd := &AppsCmd{
		Simulator: device.UDID,
		UserOnly:  c.UserOnly,
	}
	apps, err := appsCmd.getInstalledApps(ctx, device.UDID)
	if err != nil {
		return c.outputError(globals, "LIST_APPS_FAILED", err.Error())
	}

	// Filter to user apps if requested
	if c.UserOnly {
		var userApps []appInfo
		for _, app := range apps {
			if app.Type == "user" {
				userApps = append(userApps, app)
			}
		}
		apps = userApps
	}

	if len(apps) == 0 {
		return c.outputError(globals, "NO_APPS", "No apps found on simulator")
	}

	// Convert apps to pick items
	items := make([]list.Item, 0, len(apps))
	for _, app := range apps {
		version := app.Version
		if app.BuildNumber != "" && app.BuildNumber != app.Version {
			version += " (" + app.BuildNumber + ")"
		}
		items = append(items, pickItem{
			id:          app.BundleID,
			title:       app.Name,
			description: app.BundleID + " • " + version,
		})
	}

	// Run picker
	selected, err := c.runPicker(items, "Select App ("+device.Name+")")
	if err != nil {
		return err
	}

	// Output result
	return c.outputResult(globals, "app", "", selected.title, selected.id)
}

func (c *PickCmd) runPicker(items []list.Item, title string) (pickItem, error) {
	// Configure list delegate with styles
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("39")).
		Foreground(lipgloss.Color("39")).
		Padding(0, 0, 0, 1)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("241"))

	// Create list
	l := list.New(items, delegate, 0, 0)
	l.Title = title
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("39")).
		Foreground(lipgloss.Color("0")).
		Padding(0, 1)

	// Create and run the model
	m := pickModel{list: l}
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return pickItem{}, fmt.Errorf("picker error: %w", err)
	}

	result := finalModel.(pickModel)
	if result.canceled {
		return pickItem{}, errors.New("selection canceled")
	}

	return result.selected, nil
}

func (c *PickCmd) outputResult(globals *Globals, pickType, udid, name, bundleID string) error {
	if globals.Format == "ndjson" {
		result := map[string]interface{}{
			"type":          "pick",
			"schemaVersion": output.SchemaVersion,
			"picked":        pickType,
			"name":          name,
		}
		if udid != "" {
			result["udid"] = udid
		}
		if bundleID != "" {
			result["bundle_id"] = bundleID
		}

		w := output.NewNDJSONWriter(globals.Stdout)
		return w.WriteRaw(result)
	}

	// Text format: output just the ID for piping
	id := udid
	if bundleID != "" {
		id = bundleID
	}
	// Clean the name (remove booted indicator)
	cleanName := strings.TrimSuffix(name, " (booted)")
	_, err := io.WriteString(globals.Stdout, id+"\t"+cleanName+"\n")
	return err
}

func (c *PickCmd) outputError(globals *Globals, code, message string) error {
	return outputErrorCommon(globals, code, message)
}
