package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var wikiLinkRe = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

// RenderWikiLinks converts [[wiki-link]] syntax in text to clickable OSC 8
// terminal hyperlinks pointing to the corresponding knowledge doc files.
// Falls back to plain text on terminals that don't support hyperlinks.
func RenderWikiLinks(text, projectDir string) string {
	if !isTermHyperlinkCapable() {
		return text
	}

	return wikiLinkRe.ReplaceAllStringFunc(text, func(match string) string {
		parts := wikiLinkRe.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		target := parts[1]
		display := target
		if len(parts) > 2 && parts[2] != "" {
			display = parts[2]
		}

		// Strip heading reference for file resolution
		fileTarget := target
		if idx := strings.Index(fileTarget, "#"); idx >= 0 {
			fileTarget = fileTarget[:idx]
		}

		filePath := resolveWikiLink(projectDir, fileTarget)
		if filePath == "" {
			return Cyan.Render("[[" + display + "]]")
		}

		return hyperlinkFile(filePath, "[["+display+"]]")
	})
}

func resolveWikiLink(projectDir, target string) string {
	candidates := []string{
		filepath.Join(projectDir, "knowledge", target+".md"),
		filepath.Join(projectDir, "todos", target+".md"),
		filepath.Join(projectDir, target+".md"),
		filepath.Join(projectDir, target),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			abs, _ := filepath.Abs(path)
			return abs
		}
	}
	return ""
}

func hyperlinkFile(path, display string) string {
	url := "file://" + path
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, Cyan.Render(display))
}

func isTermHyperlinkCapable() bool {
	term := os.Getenv("TERM_PROGRAM")
	switch term {
	case "iTerm.app", "WezTerm", "vscode", "ghostty":
		return true
	}
	if os.Getenv("WT_SESSION") != "" { // Windows Terminal
		return true
	}
	if strings.Contains(os.Getenv("TERM"), "kitty") {
		return true
	}
	// Warp
	if os.Getenv("TERM_PROGRAM") == "WarpTerminal" {
		return true
	}
	return false
}
