// Package theme provides preset color themes and applies them to the global
// style variables in the tui package. It supports built-in presets (default,
// high-contrast, light, solarized, dracula, catppuccin, nord), individual
// color overrides, glamour markdown theme configuration, and the NO_COLOR
// standard.
package theme

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/walter/p/internal/config"
	"github.com/walter/p/internal/tui"
)

// ColorPreset defines a complete set of colors for all styled elements.
type ColorPreset struct {
	// Base CLI styles (style.go)
	Green  string
	Yellow string
	Red    string
	Dim    string
	Cyan   string

	// TUI styles (styles.go)
	Border      string
	Open        string
	Done        string
	Blocked     string
	PriorityNow string
	Backlog     string
	Selected    string
	SelectedBG  string
	Title       string
	Help        string
	Error       string
	Status      string
	Cursor      string
	CountOpen   string
	CountDone   string
	CountBlock  string
}

// Presets maps preset names to their color definitions.
var Presets = map[string]*ColorPreset{
	"default": {
		Green:       "42",
		Yellow:      "214",
		Red:         "196",
		Dim:         "245",
		Cyan:        "51",
		Border:      "62",
		Open:        "15",
		Done:        "246",
		Blocked:     "208",
		PriorityNow: "196",
		Backlog:     "244",
		Selected:    "15",
		SelectedBG:  "62",
		Title:       "99",
		Help:        "245",
		Error:       "196",
		Status:      "42",
		Cursor:      "42",
		CountOpen:   "15",
		CountDone:   "242",
		CountBlock:  "208",
	},
	"high-contrast": {
		Green:       "46",
		Yellow:      "220",
		Red:         "203",
		Dim:         "248",
		Cyan:        "87",
		Border:      "135",
		Open:        "15",
		Done:        "248",
		Blocked:     "214",
		PriorityNow: "203",
		Backlog:     "250",
		Selected:    "15",
		SelectedBG:  "135",
		Title:       "141",
		Help:        "250",
		Error:       "203",
		Status:      "46",
		Cursor:      "46",
		CountOpen:   "15",
		CountDone:   "248",
		CountBlock:  "214",
	},
	"light": {
		Green:       "28",
		Yellow:      "172",
		Red:         "160",
		Dim:         "240",
		Cyan:        "30",
		Border:      "55",
		Open:        "0",
		Done:        "240",
		Blocked:     "166",
		PriorityNow: "160",
		Backlog:     "238",
		Selected:    "15",
		SelectedBG:  "55",
		Title:       "55",
		Help:        "238",
		Error:       "160",
		Status:      "28",
		Cursor:      "28",
		CountOpen:   "0",
		CountDone:   "240",
		CountBlock:  "166",
	},
	"solarized": {
		Green:       "#859900", // solarized green
		Yellow:      "#b58900", // solarized yellow
		Red:         "#dc322f", // solarized red
		Dim:         "#586e75", // solarized base01
		Cyan:        "#2aa198", // solarized cyan
		Border:      "#6c71c4", // solarized violet
		Open:        "#839496", // solarized base0
		Done:        "#586e75", // solarized base01
		Blocked:     "#cb4b16", // solarized orange
		PriorityNow: "#dc322f", // solarized red
		Backlog:     "#657b83", // solarized base00
		Selected:    "#fdf6e3", // solarized base3
		SelectedBG:  "#6c71c4", // solarized violet
		Title:       "#268bd2", // solarized blue
		Help:        "#657b83", // solarized base00
		Error:       "#dc322f", // solarized red
		Status:      "#859900", // solarized green
		Cursor:      "#268bd2", // solarized blue
		CountOpen:   "#839496", // solarized base0
		CountDone:   "#586e75", // solarized base01
		CountBlock:  "#cb4b16", // solarized orange
	},
	"dracula": {
		Green:       "#50fa7b", // dracula green
		Yellow:      "#f1fa8c", // dracula yellow
		Red:         "#ff5555", // dracula red
		Dim:         "#6272a4", // dracula comment
		Cyan:        "#8be9fd", // dracula cyan
		Border:      "#bd93f9", // dracula purple
		Open:        "#f8f8f2", // dracula foreground
		Done:        "#6272a4", // dracula comment
		Blocked:     "#ffb86c", // dracula orange
		PriorityNow: "#ff79c6", // dracula pink
		Backlog:     "#6272a4", // dracula comment
		Selected:    "#f8f8f2", // dracula foreground
		SelectedBG:  "#bd93f9", // dracula purple
		Title:       "#bd93f9", // dracula purple
		Help:        "#6272a4", // dracula comment
		Error:       "#ff5555", // dracula red
		Status:      "#50fa7b", // dracula green
		Cursor:      "#50fa7b", // dracula green
		CountOpen:   "#f8f8f2", // dracula foreground
		CountDone:   "#6272a4", // dracula comment
		CountBlock:  "#ffb86c", // dracula orange
	},
	"catppuccin": {
		Green:       "#a6e3a1", // catppuccin mocha green
		Yellow:      "#f9e2af", // catppuccin mocha yellow
		Red:         "#f38ba8", // catppuccin mocha red
		Dim:         "#6c7086", // catppuccin mocha overlay0
		Cyan:        "#94e2d5", // catppuccin mocha teal
		Border:      "#cba6f7", // catppuccin mocha mauve
		Open:        "#cdd6f4", // catppuccin mocha text
		Done:        "#6c7086", // catppuccin mocha overlay0
		Blocked:     "#fab387", // catppuccin mocha peach
		PriorityNow: "#f38ba8", // catppuccin mocha red
		Backlog:     "#7f849c", // catppuccin mocha overlay1
		Selected:    "#1e1e2e", // catppuccin mocha base
		SelectedBG:  "#cba6f7", // catppuccin mocha mauve
		Title:       "#89b4fa", // catppuccin mocha blue
		Help:        "#7f849c", // catppuccin mocha overlay1
		Error:       "#f38ba8", // catppuccin mocha red
		Status:      "#a6e3a1", // catppuccin mocha green
		Cursor:      "#89b4fa", // catppuccin mocha blue
		CountOpen:   "#cdd6f4", // catppuccin mocha text
		CountDone:   "#6c7086", // catppuccin mocha overlay0
		CountBlock:  "#fab387", // catppuccin mocha peach
	},
	"nord": {
		Green:       "#a3be8c", // nord green (aurora)
		Yellow:      "#ebcb8b", // nord yellow (aurora)
		Red:         "#bf616a", // nord red (aurora)
		Dim:         "#4c566a", // nord polar night 4
		Cyan:        "#88c0d0", // nord frost 3
		Border:      "#81a1c1", // nord frost 2
		Open:        "#eceff4", // nord snow storm 3
		Done:        "#4c566a", // nord polar night 4
		Blocked:     "#d08770", // nord orange (aurora)
		PriorityNow: "#bf616a", // nord red (aurora)
		Backlog:     "#616e88", // dimmed nord
		Selected:    "#eceff4", // nord snow storm 3
		SelectedBG:  "#5e81ac", // nord frost 1
		Title:       "#81a1c1", // nord frost 2
		Help:        "#616e88", // dimmed nord
		Error:       "#bf616a", // nord red (aurora)
		Status:      "#a3be8c", // nord green (aurora)
		Cursor:      "#88c0d0", // nord frost 3
		CountOpen:   "#eceff4", // nord snow storm 3
		CountDone:   "#4c566a", // nord polar night 4
		CountBlock:  "#d08770", // nord orange (aurora)
	},
}

// PresetNames returns the list of available theme preset names.
func PresetNames() []string {
	return []string{"default", "high-contrast", "light", "solarized", "dracula", "catppuccin", "nord"}
}

// Apply configures all tui style variables based on the config's theme preset
// and individual color overrides. It also handles the NO_COLOR env var and
// stores the glamour theme setting. Must be called after config load and before
// any rendering.
func Apply(cfg config.Config) {
	// Handle NO_COLOR — disable all color output
	if _, noColor := os.LookupEnv("NO_COLOR"); noColor {
		applyNoColor()
		tui.GlamourThemeSetting = "notty"
		return
	}

	// Resolve preset
	preset := Presets[cfg.Theme]
	if preset == nil {
		preset = Presets["default"]
	}

	// Apply base CLI styles (style.go)
	tui.Green = lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Green))
	tui.Yellow = lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Yellow))
	tui.Red = lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Red))
	tui.Dim = lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Dim))
	tui.Cyan = lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Cyan))

	// Apply TUI styles (styles.go)
	tui.BorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(preset.Border))
	tui.OpenStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Open))
	tui.DoneStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Done))
	tui.BlockedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Blocked))
	tui.NowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(preset.PriorityNow))
	tui.BacklogStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Backlog))
	tui.SelectedStyle = lipgloss.NewStyle().Bold(true).
		Background(lipgloss.Color(preset.SelectedBG)).
		Foreground(lipgloss.Color(preset.Selected))
	tui.TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(preset.Title))
	tui.HelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Help))
	tui.ErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Error))
	tui.StatusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Status))
	tui.CursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Cursor))
	tui.CountOpenStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(preset.CountOpen))
	tui.CountDoneStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(preset.CountDone))
	tui.CountBlockedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(preset.CountBlock))

	// Apply individual color overrides on top of preset
	applyOverrides(cfg.Colors)

	// Set glamour theme
	glamourTheme := cfg.GlamourTheme
	if glamourTheme == "" {
		glamourTheme = "auto"
	}
	tui.GlamourThemeSetting = glamourTheme
}

// applyOverrides applies individual color overrides from the config on top of
// the current preset. Only non-empty values are applied.
func applyOverrides(colors config.ColorConfig) {
	if colors.Dim != "" {
		tui.Dim = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Dim))
	}
	if colors.Green != "" {
		tui.Green = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Green))
	}
	if colors.Yellow != "" {
		tui.Yellow = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Yellow))
	}
	if colors.Red != "" {
		tui.Red = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Red))
	}
	if colors.Cyan != "" {
		tui.Cyan = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Cyan))
	}
	if colors.Done != "" {
		tui.DoneStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Done))
		tui.CountDoneStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Done))
	}
	if colors.Help != "" {
		tui.HelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Help))
	}
	if colors.Accent != "" {
		c := lipgloss.Color(colors.Accent)
		tui.BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c)
		tui.TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(c)
		tui.SelectedStyle = lipgloss.NewStyle().Bold(true).
			Background(c).
			Foreground(lipgloss.Color("15"))
		tui.CursorStyle = lipgloss.NewStyle().Foreground(c)
	}
	if colors.Open != "" {
		tui.OpenStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Open))
		tui.CountOpenStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Open))
	}
	if colors.Blocked != "" {
		tui.BlockedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Blocked))
		tui.CountBlockedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Blocked))
	}
	if colors.PriorityNow != "" {
		tui.NowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.PriorityNow))
	}
	if colors.Error != "" {
		tui.ErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colors.Error))
	}
}

// applyNoColor disables all color output by setting every style to plain.
func applyNoColor() {
	plain := lipgloss.NewStyle()
	bold := lipgloss.NewStyle().Bold(true)

	tui.Green = plain
	tui.Yellow = plain
	tui.Red = plain
	tui.Dim = plain
	tui.Cyan = plain
	tui.Bold = bold

	tui.BorderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	tui.OpenStyle = plain
	tui.DoneStyle = plain
	tui.BlockedStyle = plain
	tui.NowStyle = plain
	tui.BacklogStyle = plain
	tui.SelectedStyle = bold
	tui.TitleStyle = bold
	tui.HelpStyle = plain
	tui.ErrorStyle = plain
	tui.StatusStyle = plain
	tui.CursorStyle = plain
	tui.CountOpenStyle = plain
	tui.CountDoneStyle = plain
	tui.CountBlockedStyle = plain
}
