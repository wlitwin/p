package sanitize

import "testing"

func TestQuoteYAMLValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "Hello World", "Hello World"},
		{"empty string", "", ""},
		{"colon in value", "Migration: Phase 2", `"Migration: Phase 2"`},
		{"hash in value", "Issue #42 Fix", `"Issue #42 Fix"`},
		{"square brackets", "[WIP] New Feature", `"[WIP] New Feature"`},
		{"curly braces", "{draft} API Design", `"{draft} API Design"`},
		{"greater than", "Compare > contrast", `"Compare > contrast"`},
		{"pipe", "Build | Deploy", `"Build | Deploy"`},
		{"ampersand", "R&D Tasks", `"R&D Tasks"`},
		{"asterisk", "Fix *critical* bug", `"Fix *critical* bug"`},
		{"question mark", "How? Why?", `"How? Why?"`},
		{"exclamation", "Fix ASAP!", `"Fix ASAP!"`},
		{"percent", "100% complete", `"100% complete"`},
		{"at sign", "Review @team", `"Review @team"`},
		{"backtick", "Fix `nil` error", "\"Fix `nil` error\""},
		{"starts with double quote", `"big" refactor`, `"\"big\" refactor"`},
		{"starts with single quote", "'twas the night", `"'twas the night"`},
		{"contains backslash", `Path C:\Users\test`, `"Path C:\\Users\\test"`},
		{"multiple specials", "Step 1: Fix #42 [urgent]", `"Step 1: Fix #42 [urgent]"`},
		{"internal double quotes", `The "big" refactor`, `"The \"big\" refactor"`},
		{"single quote inside", "It's done", `"It's done"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := QuoteYAMLValue(tt.input)
			if got != tt.want {
				t.Errorf("QuoteYAMLValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUnquoteYAMLValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "Hello World", "Hello World"},
		{"empty string", "", ""},
		{"single char", "x", "x"},
		{"double-quoted simple", `"Migration: Phase 2"`, "Migration: Phase 2"},
		{"double-quoted escaped quotes", `"The \"big\" refactor"`, `The "big" refactor`},
		{"double-quoted escaped backslash", `"Path C:\\Users\\test"`, `Path C:\Users\test`},
		{"single-quoted simple", "'Migration: Phase 2'", "Migration: Phase 2"},
		{"single-quoted with escaped quote", "''twas''", "'twas'"},
		{"unquoted with colon", "just: text", "just: text"},
		{"only quotes", `""`, ""},
		{"single only quotes", "''", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UnquoteYAMLValue(tt.input)
			if got != tt.want {
				t.Errorf("UnquoteYAMLValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestQuoteUnquoteRoundTrip(t *testing.T) {
	values := []string{
		"Simple title",
		"Migration: Phase 2",
		"Issue #42 Fix",
		"[WIP] New Feature",
		`The "big" refactor`,
		"R&D Tasks",
		"100% complete",
		`Path C:\Users\test`,
		"Step 1: Fix #42 [urgent]",
		"Fix `nil` error",
		"Normal title no specials",
	}

	for _, v := range values {
		quoted := QuoteYAMLValue(v)
		unquoted := UnquoteYAMLValue(quoted)
		if unquoted != v {
			t.Errorf("round-trip failed for %q: quoted=%q, unquoted=%q", v, quoted, unquoted)
		}
	}
}
