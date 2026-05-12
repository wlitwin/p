package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/walter/p/internal/config"
	"github.com/walter/p/internal/theme"
)

var configCmd = &cobra.Command{
	Use:   "config [key] [value]",
	Short: "View or set configuration",
	Long: `View all config values, get a specific value, or set a value.

Examples:
  p config                           # show all config
  p config project_root              # show one value
  p config claude_model sonnet       # set a value
  p config theme high-contrast       # set theme preset
  p config colors.dim 248            # override individual color
  p config theme --list              # list available presets`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		listThemes, _ := cmd.Flags().GetBool("list")
		previewThemes, _ := cmd.Flags().GetBool("preview")

		if listThemes || previewThemes {
			return showThemeList(previewThemes)
		}

		switch len(args) {
		case 0:
			return showAllConfig()
		case 1:
			return showConfigKey(args[0])
		case 2:
			return setConfigKey(args[0], args[1])
		}
		return nil
	},
}

func showThemeList(preview bool) error {
	names := theme.PresetNames()
	current := cfg.Theme
	if current == "" {
		current = "default"
	}

	fmt.Println("Available themes:")
	for _, name := range names {
		marker := "  "
		if name == current {
			marker = "* "
		}
		fmt.Printf("  %s%s\n", marker, name)
	}

	if preview {
		fmt.Println()
		for _, name := range names {
			preset := theme.Presets[name]
			if preset == nil {
				continue
			}
			fmt.Printf("  ── %s ──\n", name)
			green := lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Green))
			yellow := lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Yellow))
			red := lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Red))
			dim := lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Dim))
			cyan := lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Cyan))
			title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(preset.Title))
			help := lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Help))
			done := lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Done))
			blocked := lipgloss.NewStyle().Foreground(lipgloss.Color(preset.Blocked))

			fmt.Printf("    %s  %s  %s\n",
				green.Render("[x] done item"),
				yellow.Render("[ ] open item"),
				red.Render("[-] dropped"),
			)
			fmt.Printf("    %s  %s\n",
				cyan.Render("priority=now"),
				dim.Render("priority=backlog"),
			)
			fmt.Printf("    %s  %s  %s\n",
				title.Render("Title Text"),
				help.Render("help/keybindings"),
				done.Render("completed item"),
			)
			fmt.Printf("    %s\n\n",
				blocked.Render("[~] blocked item"),
			)
		}
	}

	return nil
}

func showAllConfig() error {
	path, _ := config.ConfigPath()
	fmt.Printf("Config file: %s\n\n", path)
	fmt.Printf("  %-20s %s\n", "project_root", cfg.ProjectRoot)
	fmt.Printf("  %-20s %s\n", "claude_path", cfg.ClaudePath)
	fmt.Printf("  %-20s %s\n", "claude_model", cfg.ClaudeModel)
	fmt.Printf("  %-20s %s\n", "default_priority", cfg.DefaultPriority)
	themeName := cfg.Theme
	if themeName == "" {
		themeName = "default"
	}
	fmt.Printf("  %-20s %s\n", "theme", themeName)
	glamourTheme := cfg.GlamourTheme
	if glamourTheme == "" {
		glamourTheme = "auto"
	}
	fmt.Printf("  %-20s %s\n", "glamour_theme", glamourTheme)

	// Show color overrides if any are set
	if hasColorOverrides(cfg.Colors) {
		fmt.Println()
		fmt.Println("  Color overrides:")
		printColorIfSet("    dim", cfg.Colors.Dim)
		printColorIfSet("    done", cfg.Colors.Done)
		printColorIfSet("    help", cfg.Colors.Help)
		printColorIfSet("    accent", cfg.Colors.Accent)
		printColorIfSet("    open", cfg.Colors.Open)
		printColorIfSet("    green", cfg.Colors.Green)
		printColorIfSet("    yellow", cfg.Colors.Yellow)
		printColorIfSet("    red", cfg.Colors.Red)
		printColorIfSet("    cyan", cfg.Colors.Cyan)
		printColorIfSet("    blocked", cfg.Colors.Blocked)
		printColorIfSet("    priority_now", cfg.Colors.PriorityNow)
		printColorIfSet("    error", cfg.Colors.Error)
	}
	return nil
}

func hasColorOverrides(c config.ColorConfig) bool {
	return c.Dim != "" || c.Done != "" || c.Help != "" || c.Accent != "" ||
		c.Open != "" || c.Green != "" || c.Yellow != "" || c.Red != "" ||
		c.Cyan != "" || c.Blocked != "" || c.PriorityNow != "" || c.Error != ""
}

func printColorIfSet(label, value string) {
	if value != "" {
		swatch := lipgloss.NewStyle().Foreground(lipgloss.Color(value))
		fmt.Printf("  %-20s %s %s\n", label, value, swatch.Render("████"))
	}
}

func showConfigKey(key string) error {
	val, err := getConfigValue(key)
	if err != nil {
		return err
	}
	fmt.Println(val)
	return nil
}

func setConfigKey(key, value string) error {
	// Handle colors.* keys
	if strings.HasPrefix(key, "colors.") {
		colorKey := strings.TrimPrefix(key, "colors.")
		if err := setColorKey(colorKey, value); err != nil {
			return err
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		if value == "" {
			fmt.Printf("Cleared %s (using preset default)\n", key)
		} else {
			swatch := lipgloss.NewStyle().Foreground(lipgloss.Color(value))
			fmt.Printf("Set %s = %s %s\n", key, value, swatch.Render("████"))
		}
		return nil
	}

	switch key {
	case "project_root":
		cfg.ProjectRoot = expandHome(value)
	case "claude_path":
		cfg.ClaudePath = value
	case "claude_model":
		cfg.ClaudeModel = value
	case "default_priority":
		cfg.DefaultPriority = value
	case "theme":
		// Validate theme name
		valid := false
		for _, name := range theme.PresetNames() {
			if value == name {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("unknown theme: %s\nAvailable themes: %s", value, strings.Join(theme.PresetNames(), ", "))
		}
		cfg.Theme = value
	case "glamour_theme":
		switch value {
		case "auto", "dark", "light", "notty":
			cfg.GlamourTheme = value
		default:
			return fmt.Errorf("unknown glamour_theme: %s\nValid values: auto, dark, light, notty", value)
		}
	default:
		validKeys := "project_root, claude_path, claude_model, default_priority, theme, glamour_theme, colors.*"
		return fmt.Errorf("unknown config key: %s\nValid keys: %s", key, validKeys)
	}

	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Printf("Set %s = %s\n", key, value)
	return nil
}

func setColorKey(colorKey, value string) error {
	switch colorKey {
	case "dim":
		cfg.Colors.Dim = value
	case "done":
		cfg.Colors.Done = value
	case "help":
		cfg.Colors.Help = value
	case "accent":
		cfg.Colors.Accent = value
	case "open":
		cfg.Colors.Open = value
	case "green":
		cfg.Colors.Green = value
	case "yellow":
		cfg.Colors.Yellow = value
	case "red":
		cfg.Colors.Red = value
	case "cyan":
		cfg.Colors.Cyan = value
	case "blocked":
		cfg.Colors.Blocked = value
	case "priority_now":
		cfg.Colors.PriorityNow = value
	case "error":
		cfg.Colors.Error = value
	default:
		validColors := "dim, done, help, accent, open, green, yellow, red, cyan, blocked, priority_now, error"
		return fmt.Errorf("unknown color key: %s\nValid color keys: %s", colorKey, validColors)
	}
	return nil
}

func getConfigValue(key string) (string, error) {
	// Handle colors.* keys
	if strings.HasPrefix(key, "colors.") {
		colorKey := strings.TrimPrefix(key, "colors.")
		return getColorValue(colorKey)
	}

	switch key {
	case "project_root":
		return cfg.ProjectRoot, nil
	case "claude_path":
		return cfg.ClaudePath, nil
	case "claude_model":
		return cfg.ClaudeModel, nil
	case "default_priority":
		return cfg.DefaultPriority, nil
	case "theme":
		t := cfg.Theme
		if t == "" {
			t = "default"
		}
		return t, nil
	case "glamour_theme":
		g := cfg.GlamourTheme
		if g == "" {
			g = "auto"
		}
		return g, nil
	default:
		validKeys := "project_root, claude_path, claude_model, default_priority, theme, glamour_theme, colors.*"
		return "", fmt.Errorf("unknown config key: %s\nValid keys: %s", key, validKeys)
	}
}

func getColorValue(colorKey string) (string, error) {
	switch colorKey {
	case "dim":
		return cfg.Colors.Dim, nil
	case "done":
		return cfg.Colors.Done, nil
	case "help":
		return cfg.Colors.Help, nil
	case "accent":
		return cfg.Colors.Accent, nil
	case "open":
		return cfg.Colors.Open, nil
	case "green":
		return cfg.Colors.Green, nil
	case "yellow":
		return cfg.Colors.Yellow, nil
	case "red":
		return cfg.Colors.Red, nil
	case "cyan":
		return cfg.Colors.Cyan, nil
	case "blocked":
		return cfg.Colors.Blocked, nil
	case "priority_now":
		return cfg.Colors.PriorityNow, nil
	case "error":
		return cfg.Colors.Error, nil
	default:
		validColors := "dim, done, help, accent, open, green, yellow, red, cyan, blocked, priority_now, error"
		return "", fmt.Errorf("unknown color key: %s\nValid color keys: %s", colorKey, validColors)
	}
}

func init() {
	configCmd.Flags().Bool("list", false, "List available theme presets")
	configCmd.Flags().Bool("preview", false, "Preview available theme presets with sample colors")
	rootCmd.AddCommand(configCmd)
}
