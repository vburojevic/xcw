package output

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles holds all lipgloss styles for text output
var Styles = struct {
	// Log level styles
	Debug   lipgloss.Style
	Info    lipgloss.Style
	Default lipgloss.Style
	Error   lipgloss.Style
	Fault   lipgloss.Style

	// Component styles
	Timestamp lipgloss.Style
	Process   lipgloss.Style
	Subsystem lipgloss.Style
	Message   lipgloss.Style

	// Summary styles
	Header  lipgloss.Style
	Label   lipgloss.Style
	Value   lipgloss.Style
	Success lipgloss.Style
	Warning lipgloss.Style
	Danger  lipgloss.Style

	// TUI styles
	Title     lipgloss.Style
	StatusBar lipgloss.Style
	Selected  lipgloss.Style
	Help      lipgloss.Style
}{
	// Log levels - distinctive colors
	Debug:   lipgloss.NewStyle().Foreground(lipgloss.Color("243")),                            // Gray
	Info:    lipgloss.NewStyle().Foreground(lipgloss.Color("39")),                             // Cyan
	Default: lipgloss.NewStyle().Foreground(lipgloss.Color("252")),                            // White
	Error:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),                 // Red bold
	Fault:   lipgloss.NewStyle().Foreground(lipgloss.Color("201")).Bold(true).Underline(true), // Magenta bold underline

	// Components
	Timestamp: lipgloss.NewStyle().Foreground(lipgloss.Color("244")), // Gray
	Process:   lipgloss.NewStyle().Foreground(lipgloss.Color("33")),  // Blue
	Subsystem: lipgloss.NewStyle().Foreground(lipgloss.Color("142")), // Yellow-green
	Message:   lipgloss.NewStyle(),

	// Summary
	Header:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).BorderStyle(lipgloss.NormalBorder()).BorderBottom(true).BorderForeground(lipgloss.Color("239")),
	Label:   lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
	Value:   lipgloss.NewStyle().Bold(true),
	Success: lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true),  // Green
	Warning: lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true), // Orange
	Danger:  lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true), // Red

	// TUI
	Title:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Padding(0, 1),
	StatusBar: lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("252")).Padding(0, 1),
	Selected:  lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("39")),
	Help:      lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
}

// LevelStyle returns the appropriate style for a log level string
func LevelStyle(level string) lipgloss.Style {
	switch level {
	case "Debug":
		return Styles.Debug
	case "Info":
		return Styles.Info
	case "Default":
		return Styles.Default
	case "Error":
		return Styles.Error
	case "Fault":
		return Styles.Fault
	default:
		return Styles.Default
	}
}

// LevelIndicator returns a styled level indicator
func LevelIndicator(level string) string {
	style := LevelStyle(level)
	switch level {
	case "Debug":
		return style.Render("DBG")
	case "Info":
		return style.Render("INF")
	case "Default":
		return style.Render("DEF")
	case "Error":
		return style.Render("ERR")
	case "Fault":
		return style.Render("FLT")
	default:
		return style.Render("???")
	}
}

// StatusStyle returns a style based on status
func StatusStyle(hasErrors, hasFaults bool) lipgloss.Style {
	if hasFaults {
		return Styles.Danger
	}
	if hasErrors {
		return Styles.Warning
	}
	return Styles.Success
}

// StatusText returns styled status text
func StatusText(hasErrors, hasFaults bool) string {
	if hasFaults {
		return Styles.Danger.Render("FAULTS DETECTED")
	}
	if hasErrors {
		return Styles.Warning.Render("ERRORS DETECTED")
	}
	return Styles.Success.Render("OK")
}
