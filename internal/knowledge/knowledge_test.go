package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestCreateAndRead(t *testing.T) {
	dir := setupTestProject(t)

	if err := Create(dir, "overview", "Architecture Overview", []string{"arch", "db"}); err != nil {
		t.Fatal(err)
	}

	content, err := Read(dir, "overview")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(content, "title: Architecture Overview") {
		t.Error("missing title in frontmatter")
	}
	if !strings.Contains(content, "tags: [arch, db]") {
		t.Error("missing tags in frontmatter")
	}
	if !strings.Contains(content, "# Architecture Overview") {
		t.Error("missing heading")
	}
}

func TestCreateDuplicate(t *testing.T) {
	dir := setupTestProject(t)

	if err := Create(dir, "test", "Test", nil); err != nil {
		t.Fatal(err)
	}
	if err := Create(dir, "test", "Test", nil); err == nil {
		t.Error("expected error for duplicate creation")
	}
}

func TestAppend(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "doc", "Doc", nil)

	if err := Append(dir, "doc", "New content here.", ""); err != nil {
		t.Fatal(err)
	}

	content, _ := Read(dir, "doc")
	if !strings.Contains(content, "New content here.") {
		t.Error("appended content not found")
	}
}

func TestAppendToSection(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "doc", "Doc", nil)
	Append(dir, "doc", "## Decisions", "")

	if err := Append(dir, "doc", "We chose PostgreSQL.", "Decisions"); err != nil {
		t.Fatal(err)
	}

	content, _ := Read(dir, "doc")
	if !strings.Contains(content, "We chose PostgreSQL.") {
		t.Error("section content not found")
	}
}

func TestReplaceSection(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "doc", "Doc", nil)
	Append(dir, "doc", "## Overview\n\nOld content.", "")

	if err := ReplaceSection(dir, "doc", "Overview", "New content."); err != nil {
		t.Fatal(err)
	}

	content, _ := Read(dir, "doc")
	if strings.Contains(content, "Old content.") {
		t.Error("old content still present")
	}
	if !strings.Contains(content, "New content.") {
		t.Error("new content not found")
	}
}

func TestListFiles(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "alpha", "Alpha", nil)
	Create(dir, "beta", "Beta", nil)

	files, err := ListFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("got %d files, want 2", len(files))
	}
}

func TestDelete(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "temp", "Temp", nil)

	if err := Delete(dir, "temp"); err != nil {
		t.Fatal(err)
	}

	if _, err := Read(dir, "temp"); err == nil {
		t.Error("expected error reading deleted doc")
	}
}

func TestRename(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "old-name", "Doc", nil)

	if err := Rename(dir, "old-name", "new-name"); err != nil {
		t.Fatal(err)
	}

	if _, err := Read(dir, "new-name"); err != nil {
		t.Error("renamed doc not found")
	}
	if _, err := Read(dir, "old-name"); err == nil {
		t.Error("old name still exists")
	}
}
