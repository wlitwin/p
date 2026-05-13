package tui

import "github.com/charmbracelet/lipgloss"

// TUI interactive mode styles — used by the full-screen views.
// These complement the existing CLI styles in style.go.
var (
	// Border style for view frames
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	// Item state styles
	OpenStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))  // white
	DoneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("246")) // gray (readable)
	BlockedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // orange

	// Priority styles
	NowStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // red
	BacklogStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244")) // gray

	// UI element styles
	SelectedStyle = lipgloss.NewStyle().Bold(true).
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("15"))
	TitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	HelpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	ErrorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	StatusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	CursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))

	// Counter styles for open/done/blocked counts
	CountOpenStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	CountDoneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	CountBlockedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))

	// Search match highlighting
	SearchMatchStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("214")).
				Foreground(lipgloss.Color("0")).
				Bold(true)
	SearchCurrentLineStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236"))
	SearchOtherLineStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("234"))
)
