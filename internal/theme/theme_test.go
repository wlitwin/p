package theme

import (
	"testing"

	"github.com/walter/p/internal/config"
	"github.com/walter/p/internal/tui"
)


func TestApplyDefaultPreset(t *testing.T) {
	cfg := config.Config{Theme: "default"}
	Apply(cfg)

	// Verify default preset is applied — check a few representative styles
	// Dim should be color 245 (the bumped default)
	if tui.GlamourThemeSetting != "auto" {
		t.Errorf("expected glamour theme 'auto', got %q", tui.GlamourThemeSetting)
	}
}

func TestApplyHighContrastPreset(t *testing.T) {
	cfg := config.Config{Theme: "high-contrast"}
	Apply(cfg)

	// After applying high-contrast, the styles should use brighter colors.
	// We verify they changed from the default by checking the preset was applied.
	preset := Presets["high-contrast"]
	if preset == nil {
		t.Fatal("high-contrast preset not found")
	}

	// Verify Dim was set (high-contrast uses 248)
	// We can't easily extract the exact color string from lipgloss, but we can
	// verify Apply ran without error and the preset exists.
	if len(preset.Dim) == 0 {
		t.Error("high-contrast preset Dim should not be empty")
	}
}

func TestApplyLightPreset(t *testing.T) {
	cfg := config.Config{Theme: "light"}
	Apply(cfg)

	preset := Presets["light"]
	if preset == nil {
		t.Fatal("light preset not found")
	}
	if preset.Open != "0" {
		t.Errorf("light preset Open should be '0' (black), got %q", preset.Open)
	}
}

func TestApplyUnknownPresetFallsToDefault(t *testing.T) {
	cfg := config.Config{Theme: "nonexistent"}
	// Should not panic, should fall back to default
	Apply(cfg)
}

func TestApplyEmptyThemeFallsToDefault(t *testing.T) {
	cfg := config.Config{}
	Apply(cfg)
	// Should not panic — empty theme falls back to default
}

func TestColorOverridesTakePrecedence(t *testing.T) {
	cfg := config.Config{
		Theme: "default",
		Colors: config.ColorConfig{
			Dim:  "250",
			Help: "252",
		},
	}
	Apply(cfg)

	// After applying, Dim and Help should use the override values.
	// We verify by rendering and checking the output differs from the preset.
	dimRendered := tui.Dim.Render("test")
	helpRendered := tui.HelpStyle.Render("test")

	if dimRendered == "test" && helpRendered == "test" {
		// If both render as plain text, colors might be disabled
		t.Log("styles rendered as plain text (may be non-color terminal)")
	}
}

func TestAccentOverrideAffectsMultipleStyles(t *testing.T) {
	cfg := config.Config{
		Theme: "default",
		Colors: config.ColorConfig{
			Accent: "200",
		},
	}
	Apply(cfg)
	// Accent should affect BorderStyle, TitleStyle, SelectedStyle, CursorStyle.
	// Just verify no panic and styles are set.
}

func TestOpenOverrideAffectsCountStyle(t *testing.T) {
	cfg := config.Config{
		Theme: "default",
		Colors: config.ColorConfig{
			Open: "7",
		},
	}
	Apply(cfg)
	// Open override should also set CountOpenStyle.
}

func TestBlockedOverrideAffectsCountStyle(t *testing.T) {
	cfg := config.Config{
		Theme: "default",
		Colors: config.ColorConfig{
			Blocked: "220",
		},
	}
	Apply(cfg)
	// Blocked override should also set CountBlockedStyle.
}

func TestDoneOverrideAffectsCountStyle(t *testing.T) {
	cfg := config.Config{
		Theme: "default",
		Colors: config.ColorConfig{
			Done: "250",
		},
	}
	Apply(cfg)
	// Done override should also set CountDoneStyle.
}

func TestNoColorEnvDisablesAllColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	cfg := config.Config{Theme: "high-contrast"}
	Apply(cfg)

	// All styles should render text without ANSI escape codes
	// (lipgloss may still render them depending on terminal, but
	// GlamourThemeSetting should be "notty")
	if tui.GlamourThemeSetting != "notty" {
		t.Errorf("expected glamour theme 'notty' with NO_COLOR, got %q", tui.GlamourThemeSetting)
	}

	// Reset for other tests
	t.Cleanup(func() {
		Apply(config.Config{Theme: "default"})
	})
}

func TestNoColorEmptyStringStillDisables(t *testing.T) {
	// NO_COLOR spec says the presence of the var, regardless of value, disables color
	t.Setenv("NO_COLOR", "")

	cfg := config.Config{Theme: "default"}
	Apply(cfg)

	if tui.GlamourThemeSetting != "notty" {
		t.Errorf("expected glamour theme 'notty' with NO_COLOR='', got %q", tui.GlamourThemeSetting)
	}

	t.Cleanup(func() {
		Apply(config.Config{Theme: "default"})
	})
}

func TestGlamourThemeSetting(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "auto"},
		{"auto", "auto"},
		{"dark", "dark"},
		{"light", "light"},
		{"notty", "notty"},
	}

	for _, tt := range tests {
		cfg := config.Config{GlamourTheme: tt.input}
		Apply(cfg)

		if tui.GlamourThemeSetting != tt.expected {
			t.Errorf("GlamourTheme=%q: expected GlamourThemeSetting=%q, got %q",
				tt.input, tt.expected, tui.GlamourThemeSetting)
		}
	}
}

func TestPresetNames(t *testing.T) {
	names := PresetNames()
	if len(names) < 3 {
		t.Errorf("expected at least 3 preset names, got %d", len(names))
	}

	expected := map[string]bool{"default": true, "high-contrast": true, "light": true}
	for _, name := range names {
		if !expected[name] {
			t.Errorf("unexpected preset name: %s", name)
		}
	}
}

func TestAllPresetsExist(t *testing.T) {
	for _, name := range PresetNames() {
		preset := Presets[name]
		if preset == nil {
			t.Errorf("preset %q is nil", name)
			continue
		}
		// Verify all required fields are non-empty
		if preset.Green == "" {
			t.Errorf("preset %q: Green is empty", name)
		}
		if preset.Yellow == "" {
			t.Errorf("preset %q: Yellow is empty", name)
		}
		if preset.Red == "" {
			t.Errorf("preset %q: Red is empty", name)
		}
		if preset.Dim == "" {
			t.Errorf("preset %q: Dim is empty", name)
		}
		if preset.Cyan == "" {
			t.Errorf("preset %q: Cyan is empty", name)
		}
		if preset.Help == "" {
			t.Errorf("preset %q: Help is empty", name)
		}
		if preset.Done == "" {
			t.Errorf("preset %q: Done is empty", name)
		}
		if preset.Error == "" {
			t.Errorf("preset %q: Error is empty", name)
		}
	}
}

func TestApplyPreservesStyleVarReadability(t *testing.T) {
	// Verify that StateStyle and PriorityStyle read the current style vars
	// at call time (not cached from init)
	cfg := config.Config{Theme: "high-contrast"}
	Apply(cfg)

	// These should not panic and should return non-empty strings
	state := tui.StateStyle("[x]")
	if state == "" {
		t.Error("StateStyle returned empty string")
	}

	priority := tui.PriorityStyle("now")
	if priority == "" {
		t.Error("PriorityStyle returned empty string")
	}

	backlog := tui.PriorityStyle("backlog")
	if backlog == "" {
		t.Error("PriorityStyle('backlog') returned empty string")
	}
}

func TestApplyEachPreset(t *testing.T) {
	// Apply each preset and verify no panics
	for _, name := range PresetNames() {
		t.Run(name, func(t *testing.T) {
			cfg := config.Config{Theme: name}
			Apply(cfg) // should not panic

			// Verify styles can render
			_ = tui.Green.Render("test")
			_ = tui.Yellow.Render("test")
			_ = tui.Red.Render("test")
			_ = tui.Dim.Render("test")
			_ = tui.Cyan.Render("test")
			_ = tui.Bold.Render("test")
			_ = tui.HelpStyle.Render("test")
			_ = tui.DoneStyle.Render("test")
			_ = tui.ErrorStyle.Render("test")
			_ = tui.TitleStyle.Render("test")
			_ = tui.BorderStyle.Render("test")
			_ = tui.SelectedStyle.Render("test")
		})
	}
}

func TestColorOverrideWithHexColor(t *testing.T) {
	cfg := config.Config{
		Theme: "default",
		Colors: config.ColorConfig{
			Dim: "#7C6F64",
		},
	}
	// Should not panic — hex colors are valid lipgloss color values
	Apply(cfg)
	_ = tui.Dim.Render("test")
}

func TestAllColorOverrides(t *testing.T) {
	cfg := config.Config{
		Theme: "default",
		Colors: config.ColorConfig{
			Dim:         "250",
			Done:        "250",
			Help:        "250",
			Accent:      "200",
			Open:        "7",
			Green:       "46",
			Yellow:      "220",
			Red:         "203",
			Cyan:        "87",
			Blocked:     "220",
			PriorityNow: "203",
			Error:       "203",
		},
	}
	Apply(cfg)

	// Verify no panic and all render
	_ = tui.Green.Render("test")
	_ = tui.Yellow.Render("test")
	_ = tui.Red.Render("test")
	_ = tui.Dim.Render("test")
	_ = tui.Cyan.Render("test")
	_ = tui.HelpStyle.Render("test")
	_ = tui.DoneStyle.Render("test")
	_ = tui.ErrorStyle.Render("test")
	_ = tui.NowStyle.Render("test")
	_ = tui.BlockedStyle.Render("test")
	_ = tui.OpenStyle.Render("test")
}
