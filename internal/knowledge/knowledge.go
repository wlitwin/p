package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Dir(projectDir string) string {
	return filepath.Join(projectDir, "knowledge")
}

func FilePath(projectDir, filename string) string {
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}
	return filepath.Join(Dir(projectDir), filename)
}

func ListFiles(projectDir string) ([]string, error) {
	dir := Dir(projectDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ".md"))
	}
	return names, nil
}

func Create(projectDir, filename, title string, tags []string) error {
	path := FilePath(projectDir, filename)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("knowledge file %q already exists", filename)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "title: %s\n", title)
	fmt.Fprintf(&sb, "created: %s\n", now)
	fmt.Fprintf(&sb, "updated: %s\n", now)
	if len(tags) > 0 {
		fmt.Fprintf(&sb, "tags: [%s]\n", strings.Join(tags, ", "))
	}
	sb.WriteString("---\n\n")
	fmt.Fprintf(&sb, "# %s\n", title)

	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

func Read(projectDir, filename string) (string, error) {
	data, err := os.ReadFile(FilePath(projectDir, filename))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func Append(projectDir, filename, content, section string) error {
	path := FilePath(projectDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	text := string(data)

	if section != "" {
		idx := findSection(text, section)
		if idx == -1 {
			return fmt.Errorf("section %q not found in %s", section, filename)
		}
		nextSection := findNextSection(text, idx)
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

func ReplaceSection(projectDir, filename, section, newContent string) error {
	path := FilePath(projectDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	text := string(data)
	start := findSection(text, section)
	if start == -1 {
		return fmt.Errorf("section %q not found in %s", section, filename)
	}

	headerEnd := strings.Index(text[start:], "\n")
	if headerEnd == -1 {
		headerEnd = len(text[start:])
	}
	headerEnd += start + 1

	end := findNextSection(text, start)
	if end == -1 {
		end = len(text)
	}

	header := text[start:headerEnd]
	text = text[:start] + header + "\n" + newContent + "\n" + text[end:]
	text = updateFrontmatterTimestamp(text)

	return os.WriteFile(path, []byte(text), 0o644)
}

func Rename(projectDir, oldName, newName string) error {
	oldPath := FilePath(projectDir, oldName)
	newPath := FilePath(projectDir, newName)
	return os.Rename(oldPath, newPath)
}

func findSection(text, section string) int {
	lines := strings.Split(text, "\n")
	pos := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			heading := strings.TrimLeft(trimmed, "# ")
			if strings.EqualFold(heading, section) {
				return pos
			}
		}
		pos += len(line) + 1
	}
	return -1
}

func findNextSection(text string, afterPos int) int {
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
		if strings.HasPrefix(trimmed, "#") {
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
