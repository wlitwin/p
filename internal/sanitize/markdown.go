package sanitize

import (
	"regexp"
	"strings"
)

// htmlTagRe matches HTML-like tags: <word...> or </word>.
var htmlTagRe = regexp.MustCompile(`<(/?\w[^>]*)>`)

// escapedTagRe matches already-escaped tags: \<...\>.
var escapedTagRe = regexp.MustCompile(`\\<(/?[^>\\]+)\\>`)

// EscapeHTMLTags escapes HTML-like tags in markdown text so that Obsidian
// (and other markdown renderers) don't interpret them as raw HTML. Angle
// brackets in tag-like patterns are backslash-escaped: <div> becomes \<div\>.
// Already-escaped sequences (\<...\>) are left untouched.
func EscapeHTMLTags(s string) string {
	if !strings.ContainsRune(s, '<') {
		return s
	}

	// First, temporarily replace already-escaped tags so we don't double-escape.
	// Use a placeholder that won't appear in normal text.
	const placeholder = "\x00ESCAPED_TAG\x00"
	var escaped []string
	temp := escapedTagRe.ReplaceAllStringFunc(s, func(m string) string {
		escaped = append(escaped, m)
		return placeholder
	})

	// Escape unescaped HTML-like tags.
	temp = htmlTagRe.ReplaceAllString(temp, `\<${1}\>`)

	// Restore the already-escaped tags.
	for _, orig := range escaped {
		temp = strings.Replace(temp, placeholder, orig, 1)
	}

	return temp
}

// UnescapeHTMLTags reverses EscapeHTMLTags, restoring \<...\> back to <...>.
// This is used when parsing item text from markdown files that may have been
// saved with escaped tags.
func UnescapeHTMLTags(s string) string {
	if !strings.Contains(s, `\<`) {
		return s
	}
	return escapedTagRe.ReplaceAllString(s, `<${1}>`)
}
