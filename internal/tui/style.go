package tui

import "github.com/charmbracelet/lipgloss"

var (
	Green  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	Yellow = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	Red    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	Dim    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	Bold   = lipgloss.NewStyle().Bold(true)
	Cyan   = lipgloss.NewStyle().Foreground(lipgloss.Color("51"))
)

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

func PriorityStyle(p string) string {
	switch p {
	case "backlog":
		return Dim.Render("backlog")
	default:
		return Cyan.Render("now")
	}
}
