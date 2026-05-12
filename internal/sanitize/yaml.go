// Package sanitize provides helpers for safely encoding and decoding values
// in YAML frontmatter, ensuring compatibility with Obsidian and strict YAML parsers.
package sanitize

import "strings"

// yamlSpecialChars lists characters that require a YAML value to be quoted
// when used in a simple (unquoted) scalar position.
const yamlSpecialChars = `:#{}[]>|&*?!%@"'` + "`"

// QuoteYAMLValue returns the string double-quoted if it contains characters
// that are special in YAML (colons, hashes, brackets, etc.) or if it starts
// with a quote character. Plain strings are returned unchanged.
func QuoteYAMLValue(s string) string {
	if s == "" {
		return s
	}

	needsQuoting := false

	// Strings containing YAML-special characters need quoting
	for _, c := range yamlSpecialChars {
		if strings.ContainsRune(s, c) {
			needsQuoting = true
			break
		}
	}

	if !needsQuoting {
		return s
	}

	// Double-quote the value, escaping any internal double quotes and backslashes
	var sb strings.Builder
	sb.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		default:
			sb.WriteRune(r)
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

// UnquoteYAMLValue strips surrounding double or single quotes from a YAML
// scalar value, unescaping backslash sequences within double-quoted strings.
// Unquoted values are returned as-is.
func UnquoteYAMLValue(s string) string {
	if len(s) < 2 {
		return s
	}

	// Double-quoted string
	if s[0] == '"' && s[len(s)-1] == '"' {
		inner := s[1 : len(s)-1]
		// Unescape backslash sequences
		var sb strings.Builder
		sb.Grow(len(inner))
		for i := 0; i < len(inner); i++ {
			if inner[i] == '\\' && i+1 < len(inner) {
				next := inner[i+1]
				switch next {
				case '"', '\\':
					sb.WriteByte(next)
					i++
				default:
					sb.WriteByte(inner[i])
				}
			} else {
				sb.WriteByte(inner[i])
			}
		}
		return sb.String()
	}

	// Single-quoted string (no escape processing in YAML single-quoted strings,
	// except '' for a literal single quote)
	if s[0] == '\'' && s[len(s)-1] == '\'' {
		inner := s[1 : len(s)-1]
		return strings.ReplaceAll(inner, "''", "'")
	}

	return s
}
