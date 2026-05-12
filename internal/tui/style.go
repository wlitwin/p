// Package tui provides terminal UI components including an interactive picker,
// text input, confirmation prompts, styled output, and wiki-link rendering.
package tui

import "github.com/charmbracelet/lipgloss"

// GlamourThemeSetting controls the glamour markdown rendering style.
// Set by the theme system. Values: "auto", "dark", "light", "notty".
var GlamourThemeSetting = "auto"

// Base styles used by both CLI output and TUI views.
// These are var (not const) so the theme system can reassign them at startup.
var (
	Green  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	Yellow = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	Red    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	Dim    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	Bold   = lipgloss.NewStyle().Bold(true)
	Cyan   = lipgloss.NewStyle().Foreground(lipgloss.Color("51"))
)

// StateStyle returns a colorized rendering of a checkbox marker.
func StateStyle(marker string) string {
	switch marker {
	case "[x]":
		return Green.Render("[x]")
	case "[-]":
		return Red.Render("[-]")
	default:
		return Yellow.Render("[ ]")
	}
}

// PriorityStyle returns a colorized rendering of a priority value.
func PriorityStyle(p string) string {
	switch p {
	case "backlog":
		return Dim.Render("backlog")
	default:
		return Cyan.Render("now")
	}
}
