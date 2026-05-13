package sanitize

import "testing"

func TestEscapeHTMLTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "Hello World", "Hello World"},
		{"empty string", "", ""},
		{"simple tag", "Use <div> here", `Use \<div\> here`},
		{"closing tag", "Remove </div> tag", `Remove \</div\> tag`},
		{"self-closing-like", "Use <br/> tag", `Use \<br/\> tag`},
		{"tag with attributes", "The <img src=x> tag", `The \<img src=x\> tag`},
		{"script tag", "Inject <script>alert(1)</script>", `Inject \<script\>alert(1)\</script\>`},
		{"multiple tags", "<b>bold</b> and <i>italic</i>", `\<b\>bold\</b\> and \<i\>italic\</i\>`},
		{"not a tag - comparison", "x > 5 and y < 10", "x > 5 and y < 10"},
		{"not a tag - arrow", "map -> filter -> reduce", "map -> filter -> reduce"},
		{"angle brackets no word", "< > test", "< > test"},
		{"already escaped", `Use \<div\> here`, `Use \<div\> here`},
		{"html entity", "Use &lt;div&gt;", "Use &lt;div&gt;"},
		{"mixed content", "Fix <span> in `code`", `Fix \<span\> in ` + "`code`"},
		{"url not affected", "Visit https://example.com", "Visit https://example.com"},
		{"email-like", "Send to <user@example.com>", `Send to \<user@example.com\>`},
		{"generic type", "List<String> type", `List\<String\> type`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EscapeHTMLTags(tt.input)
			if got != tt.want {
				t.Errorf("EscapeHTMLTags(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUnescapeHTMLTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "Hello World", "Hello World"},
		{"empty string", "", ""},
		{"escaped tag", `Use \<div\> here`, "Use <div> here"},
		{"escaped closing tag", `Remove \</div\> tag`, "Remove </div> tag"},
		{"multiple escaped", `\<b\>bold\</b\>`, "<b>bold</b>"},
		{"not escaped - bare backslash lt", `Use \< something`, `Use \< something`},
		{"mixed", `\<div\> and plain text`, "<div> and plain text"},
		{"escaped with attributes", `\<img src=x\>`, "<img src=x>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UnescapeHTMLTags(tt.input)
			if got != tt.want {
				t.Errorf("UnescapeHTMLTags(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEscapeUnescapeRoundTrip(t *testing.T) {
	values := []string{
		"Simple text no tags",
		"Use <div> element",
		"Remove </span> tag",
		"<b>bold</b> and <i>italic</i>",
		"Fix <script>alert(1)</script> injection",
		"Generic List<String> type",
		"Send to <user@example.com>",
	}

	for _, v := range values {
		escaped := EscapeHTMLTags(v)
		unescaped := UnescapeHTMLTags(escaped)
		if unescaped != v {
			t.Errorf("round-trip failed for %q: escaped=%q, unescaped=%q", v, escaped, unescaped)
		}
	}
}

func TestEscapeIdempotent(t *testing.T) {
	// Escaping an already-escaped string should not double-escape
	input := `Use \<div\> here`
	once := EscapeHTMLTags(input)
	twice := EscapeHTMLTags(once)
	if once != twice {
		t.Errorf("EscapeHTMLTags is not idempotent: once=%q, twice=%q", once, twice)
	}
}
