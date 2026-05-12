// Package knowledge provides CRUD and search operations for markdown knowledge
// documents with YAML frontmatter, section-level editing, and glob-based filtering.
package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/walter/p/internal/sanitize"
)

// Dir returns the absolute path to the knowledge docs directory within a project.
func Dir(projectDir string) string {
	return filepath.Join(projectDir, "knowledge")
}

// FilePath returns the absolute file path for a knowledge doc, appending
// the .md extension if not already present. Supports subdirectory paths.
func FilePath(projectDir, filename string) string {
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}
	return filepath.Join(Dir(projectDir), filename)
}

// ListFiles returns the names of all knowledge docs in the project,
// walking subdirectories recursively. Hidden directories (like .archive)
// are skipped. Names are returned without the .md extension.
func ListFiles(projectDir string) ([]string, error) {
	dir := Dir(projectDir)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	var names []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip hidden directories like .archive
			if strings.HasPrefix(d.Name(), ".") && path != dir {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		names = append(names, strings.TrimSuffix(rel, ".md"))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return names, nil
}

// Create writes a new knowledge doc with YAML frontmatter (title, timestamps,
// tags) and an H1 heading. Parent directories are created as needed.
// Returns an error if the file already exists.
func Create(projectDir, filename, title string, tags []string) error {
	path := FilePath(projectDir, filename)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("knowledge file %q already exists", filename)
	}

	// Create parent directories for nested filenames (e.g. "architecture/overview")
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "title: %s\n", sanitize.QuoteYAMLValue(title))
	fmt.Fprintf(&sb, "created: %s\n", now)
	fmt.Fprintf(&sb, "updated: %s\n", now)
	if len(tags) > 0 {
		fmt.Fprintf(&sb, "tags: [%s]\n", strings.Join(tags, ", "))
	}
	sb.WriteString("---\n\n")
	fmt.Fprintf(&sb, "# %s\n", title)

	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

// Read returns the full content of a knowledge doc as a string.
func Read(projectDir, filename string) (string, error) {
	data, err := os.ReadFile(FilePath(projectDir, filename))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Append adds content to a knowledge doc. If section is non-empty, the content
// is inserted before the next sibling heading; otherwise it is appended to the
// end of the file. The frontmatter updated timestamp is refreshed.
func Append(projectDir, filename, content, section string) error {
	path := FilePath(projectDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	text := string(data)

	if section != "" {
		idx, level := findSection(text, section)
		if idx == -1 {
			return fmt.Errorf("section %q not found in %s", section, filename)
		}
		nextSection := findNextSection(text, idx, level)
		insertAt := nextSection
		if insertAt == -1 {
			insertAt = len(text)
		}
		text = text[:insertAt] + "\n" + content + "\n" + text[insertAt:]
	} else {
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		text += "\n" + content + "\n"
	}

	text = updateFrontmatterTimestamp(text)
	return os.WriteFile(path, []byte(text), 0o644)
}

// ReplaceSection replaces the body of a markdown section (identified by heading
// text) with newContent, preserving the heading itself. Returns an error if the
// section is not found.
func ReplaceSection(projectDir, filename, section, newContent string) error {
	path := FilePath(projectDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	text := string(data)
	start, level := findSection(text, section)
	if start == -1 {
		return fmt.Errorf("section %q not found in %s", section, filename)
	}

	headerEnd := strings.Index(text[start:], "\n")
	if headerEnd == -1 {
		headerEnd = len(text[start:])
	}
	headerEnd += start + 1

	end := findNextSection(text, start, level)
	if end == -1 {
		end = len(text)
	}

	header := text[start:headerEnd]
	text = text[:start] + header + "\n" + newContent + "\n" + text[end:]
	text = updateFrontmatterTimestamp(text)

	return os.WriteFile(path, []byte(text), 0o644)
}

// ExtractTags parses the tags field from YAML frontmatter, returning them
// as a string slice. Returns nil if no tags field is found.
func ExtractTags(content string) []string {
	lines := strings.Split(content, "\n")
	inFrontmatter := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if inFrontmatter {
				return nil
			}
			inFrontmatter = true
			continue
		}
		if inFrontmatter && strings.HasPrefix(trimmed, "tags:") {
			val := strings.TrimPrefix(trimmed, "tags:")
			val = strings.TrimSpace(val)
			val = strings.Trim(val, "[]")
			var tags []string
			for _, t := range strings.Split(val, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
			return tags
		}
	}
	return nil
}

// Search performs case-insensitive full-text search across all knowledge docs,
// returning filenames that contain the query.
func Search(projectDir, query string) ([]string, error) {
	files, err := ListFiles(projectDir)
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	var matches []string
	for _, f := range files {
		content, err := Read(projectDir, f)
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(content), queryLower) {
			matches = append(matches, f)
		}
	}
	return matches, nil
}

// MatchFiles returns knowledge doc names matching any of the given glob patterns.
// If patterns is nil or empty, returns all files (backwards compatible).
// Patterns are matched against doc names (relative paths without .md extension).
// Supported syntax: filepath.Match patterns plus "**" for recursive matching.
func MatchFiles(projectDir string, patterns []string) ([]string, error) {
	allFiles, err := ListFiles(projectDir)
	if err != nil {
		return nil, err
	}

	// nil/empty patterns = return all (backwards compatible)
	if len(patterns) == 0 {
		return allFiles, nil
	}

	seen := make(map[string]bool)
	var matched []string
	for _, name := range allFiles {
		for _, pattern := range patterns {
			ok, err := matchGlob(pattern, name)
			if err != nil {
				return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
			}
			if ok && !seen[name] {
				seen[name] = true
				matched = append(matched, name)
				break
			}
		}
	}
	return matched, nil
}

// matchGlob matches a name against a glob pattern.
// It supports filepath.Match syntax plus "**" for recursive directory matching.
func matchGlob(pattern, name string) (bool, error) {
	// Handle "**" — matches everything recursively
	if pattern == "**" {
		return true, nil
	}

	// Handle patterns containing "**/"
	if strings.Contains(pattern, "**/") {
		// "dir/**" matches all files under dir/ recursively
		// "dir/**/foo" matches dir/foo, dir/a/foo, dir/a/b/foo, etc.
		prefix := strings.SplitN(pattern, "/**/", 2)
		if len(prefix) == 2 {
			// pattern is "prefix/**/suffix"
			if !strings.HasPrefix(name, prefix[0]+"/") {
				return false, nil
			}
			rest := name[len(prefix[0])+1:]
			// Try matching suffix against every sub-path
			return matchGlob(prefix[1], rest)
		}
	}

	// Handle trailing "/**" — matches everything under a directory
	if strings.HasSuffix(pattern, "/**") {
		dirPrefix := strings.TrimSuffix(pattern, "/**")
		return strings.HasPrefix(name, dirPrefix+"/"), nil
	}

	// Use filepath.Match for standard glob patterns
	return filepath.Match(pattern, name)
}

// FindReferencingLists scans all todo list frontmatters for context patterns
// that would match the given knowledge doc name. Returns list names that
// reference the doc.
func FindReferencingLists(projectDir, docName string) []string {
	dir := filepath.Join(projectDir, "todos")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var referencing []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		patterns := extractContextPatterns(string(data))
		if patterns == nil {
			continue
		}
		for _, pattern := range patterns {
			ok, err := matchGlob(pattern, docName)
			if err == nil && ok {
				referencing = append(referencing, strings.TrimSuffix(e.Name(), ".md"))
				break
			}
		}
	}
	return referencing
}

// extractContextPatterns reads the context field from todo list frontmatter.
func extractContextPatterns(content string) []string {
	lines := strings.Split(content, "\n")
	inFrontmatter := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if inFrontmatter {
				return nil // end of frontmatter, no context found
			}
			inFrontmatter = true
			continue
		}
		if !inFrontmatter {
			continue
		}
		if strings.HasPrefix(trimmed, "context:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "context:"))
			if val == "[]" {
				return []string{}
			}
			if val != "" {
				return []string{val}
			}
			// Multi-line list
			var patterns []string
			for j := i + 1; j < len(lines); j++ {
				t := strings.TrimSpace(lines[j])
				if strings.HasPrefix(t, "- ") {
					patterns = append(patterns, strings.TrimSpace(strings.TrimPrefix(t, "- ")))
				} else {
					break
				}
			}
			return patterns
		}
	}
	return nil
}

// Delete removes a knowledge doc from disk. Returns an error if the file
// does not exist.
func Delete(projectDir, filename string) error {
	path := FilePath(projectDir, filename)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("knowledge doc %q not found", filename)
	}
	return os.Remove(path)
}

// Rename moves a knowledge doc from oldName to newName on disk.
func Rename(projectDir, oldName, newName string) error {
	oldPath := FilePath(projectDir, oldName)
	newPath := FilePath(projectDir, newName)
	return os.Rename(oldPath, newPath)
}

func headingLevel(line string) int {
	trimmed := strings.TrimSpace(line)
	level := 0
	for _, c := range trimmed {
		if c == '#' {
			level++
		} else {
			break
		}
	}
	return level
}

func headingText(line string) string {
	trimmed := strings.TrimSpace(line)
	i := 0
	for i < len(trimmed) && trimmed[i] == '#' {
		i++
	}
	return strings.TrimSpace(trimmed[i:])
}

func findSection(text, section string) (int, int) {
	lines := strings.Split(text, "\n")
	pos := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			if strings.EqualFold(headingText(line), section) {
				return pos, headingLevel(line)
			}
		}
		pos += len(line) + 1
	}
	return -1, 0
}

func findNextSection(text string, afterPos int, atLevel int) int {
	lines := strings.Split(text[afterPos:], "\n")
	pos := afterPos
	first := true
	for _, line := range lines {
		if first {
			first = false
			pos += len(line) + 1
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") && headingLevel(line) <= atLevel {
			return pos
		}
		pos += len(line) + 1
	}
	return -1
}

func updateFrontmatterTimestamp(text string) string {
	if !strings.HasPrefix(text, "---\n") {
		return text
	}
	end := strings.Index(text[4:], "\n---")
	if end == -1 {
		return text
	}
	end += 4

	frontmatter := text[:end]
	rest := text[end:]
	now := time.Now().UTC().Format(time.RFC3339)

	lines := strings.Split(frontmatter, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "updated:") {
			lines[i] = "updated: " + now
		}
	}

	return strings.Join(lines, "\n") + rest
}
