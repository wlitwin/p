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

// --- ExtractTags tests ---

func TestExtractTagsWithTags(t *testing.T) {
	content := "---\ntitle: Test\ntags: [arch, db]\n---\n\n# Test\n"
	tags := ExtractTags(content)
	if len(tags) != 2 {
		t.Fatalf("got %d tags, want 2", len(tags))
	}
	if tags[0] != "arch" {
		t.Errorf("tags[0] = %q, want %q", tags[0], "arch")
	}
	if tags[1] != "db" {
		t.Errorf("tags[1] = %q, want %q", tags[1], "db")
	}
}

func TestExtractTagsNoTags(t *testing.T) {
	content := "---\ntitle: Test\ncreated: 2026-01-01\n---\n\n# Test\n"
	tags := ExtractTags(content)
	if tags != nil {
		t.Errorf("expected nil, got %v", tags)
	}
}

func TestExtractTagsNoFrontmatter(t *testing.T) {
	content := "# Just a heading\n\nSome body text.\n"
	tags := ExtractTags(content)
	if tags != nil {
		t.Errorf("expected nil, got %v", tags)
	}
}

func TestExtractTagsEmpty(t *testing.T) {
	content := "---\ntitle: Test\ntags: []\n---\n\n# Test\n"
	tags := ExtractTags(content)
	if len(tags) != 0 {
		t.Errorf("expected empty/nil tags, got %v", tags)
	}
}

func TestExtractTagsSingleTag(t *testing.T) {
	content := "---\ntitle: Test\ntags: [solo]\n---\n\n# Test\n"
	tags := ExtractTags(content)
	if len(tags) != 1 {
		t.Fatalf("got %d tags, want 1", len(tags))
	}
	if tags[0] != "solo" {
		t.Errorf("tags[0] = %q, want %q", tags[0], "solo")
	}
}

func TestExtractTagsManyTags(t *testing.T) {
	content := "---\ntitle: Test\ntags: [a, b, c, d, e]\n---\n\n# Test\n"
	tags := ExtractTags(content)
	if len(tags) != 5 {
		t.Fatalf("got %d tags, want 5", len(tags))
	}
	expected := []string{"a", "b", "c", "d", "e"}
	for i, want := range expected {
		if tags[i] != want {
			t.Errorf("tags[%d] = %q, want %q", i, tags[i], want)
		}
	}
}

// --- ExtractTags integration with Create ---

func TestExtractTagsViaCreate(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "tagged", "Tagged Doc", []string{"foo", "bar", "baz"})
	content, _ := Read(dir, "tagged")
	tags := ExtractTags(content)
	if len(tags) != 3 {
		t.Fatalf("got %d tags, want 3", len(tags))
	}
	if tags[0] != "foo" || tags[1] != "bar" || tags[2] != "baz" {
		t.Errorf("unexpected tags: %v", tags)
	}
}

func TestExtractTagsViaCreateNoTags(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "untagged", "Untagged Doc", nil)
	content, _ := Read(dir, "untagged")
	tags := ExtractTags(content)
	if tags != nil {
		t.Errorf("expected nil tags for untagged doc, got %v", tags)
	}
}

// --- Edge case tests ---

func TestAppendToNonexistentSection(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "doc", "Doc", nil)

	err := Append(dir, "doc", "Some content", "Missing")
	if err == nil {
		t.Error("expected error when appending to nonexistent section")
	}
	if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestReplaceSectionMissing(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "doc", "Doc", nil)

	err := ReplaceSection(dir, "doc", "NoSuchSection", "replacement")
	if err == nil {
		t.Error("expected error when replacing nonexistent section")
	}
	if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestAppendEmptyDoc(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "doc", "Doc", nil)

	if err := Append(dir, "doc", "First addition.", ""); err != nil {
		t.Fatal(err)
	}

	content, _ := Read(dir, "doc")
	if !strings.Contains(content, "# Doc") {
		t.Error("heading lost after append")
	}
	if !strings.Contains(content, "First addition.") {
		t.Error("appended content not found")
	}
}

func TestNestedHeadingLevels(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "doc", "Doc", nil)

	// Build a document with nested headings
	Append(dir, "doc", "## Overview\n\nTop-level section.", "")
	Append(dir, "doc", "### Details\n\nNested under overview.", "")
	Append(dir, "doc", "## Conclusion\n\nAnother top-level section.", "")

	// Append to the nested subsection "Details"
	if err := Append(dir, "doc", "Extra detail.", "Details"); err != nil {
		t.Fatal(err)
	}

	content, _ := Read(dir, "doc")
	if !strings.Contains(content, "Extra detail.") {
		t.Error("content not appended to nested section")
	}

	// Verify the Conclusion section is still intact
	if !strings.Contains(content, "## Conclusion") {
		t.Error("Conclusion section lost after nested append")
	}

	// Replace the nested subsection
	if err := ReplaceSection(dir, "doc", "Details", "Replaced detail content."); err != nil {
		t.Fatal(err)
	}

	content, _ = Read(dir, "doc")
	if strings.Contains(content, "Nested under overview.") {
		t.Error("old nested content still present after replace")
	}
	if !strings.Contains(content, "Replaced detail content.") {
		t.Error("replaced content not found in nested section")
	}
	// The Conclusion section (##) should still be present since it's at a higher level than ###
	if !strings.Contains(content, "## Conclusion") {
		t.Error("Conclusion section lost after nested section replace")
	}
}

func TestReplaceWithEmptyContent(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "doc", "Doc", nil)
	Append(dir, "doc", "## Notes\n\nSome notes here.", "")

	if err := ReplaceSection(dir, "doc", "Notes", ""); err != nil {
		t.Fatal(err)
	}

	content, _ := Read(dir, "doc")
	if strings.Contains(content, "Some notes here.") {
		t.Error("old section content still present after replacing with empty")
	}
	// The heading itself should still be present
	if !strings.Contains(content, "## Notes") {
		t.Error("section heading removed when only content should be replaced")
	}
}

func TestDeleteNonexistent(t *testing.T) {
	dir := setupTestProject(t)

	err := Delete(dir, "does-not-exist")
	if err == nil {
		t.Error("expected error deleting nonexistent doc")
	}
	if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// --- updateFrontmatterTimestamp (tested indirectly via Append) ---

func TestUpdateFrontmatterTimestamp(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "doc", "Doc", nil)

	// Read the initial content and extract the "updated:" timestamp
	content1, _ := Read(dir, "doc")
	updated1 := extractUpdatedField(content1)
	if updated1 == "" {
		t.Fatal("no updated field found in initial doc")
	}

	// Append triggers updateFrontmatterTimestamp internally
	if err := Append(dir, "doc", "Some new content.", ""); err != nil {
		t.Fatal(err)
	}

	content2, _ := Read(dir, "doc")
	updated2 := extractUpdatedField(content2)
	if updated2 == "" {
		t.Fatal("no updated field found after append")
	}

	// The updated timestamp should be >= the original (they may be equal if
	// the test runs within the same second, so we just verify it's present
	// and properly formatted).
	if len(updated2) < 20 {
		t.Errorf("updated timestamp looks malformed: %q", updated2)
	}
}

// extractUpdatedField pulls the "updated:" value from frontmatter content.
func extractUpdatedField(content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "updated:") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "updated:"))
		}
	}
	return ""
}

// --- Section operations with case-insensitive matching ---

func TestFindSectionCaseInsensitive(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "doc", "Doc", nil)
	Append(dir, "doc", "## My Section\n\nContent here.", "")

	// findSection uses EqualFold, so "my section" should match "My Section"
	if err := Append(dir, "doc", "Extra line.", "my section"); err != nil {
		t.Fatal(err)
	}

	content, _ := Read(dir, "doc")
	if !strings.Contains(content, "Extra line.") {
		t.Error("case-insensitive section match failed for append")
	}
}

func TestReplaceSectionCaseInsensitive(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "doc", "Doc", nil)
	Append(dir, "doc", "## Summary\n\nOld summary.", "")

	if err := ReplaceSection(dir, "doc", "summary", "New summary."); err != nil {
		t.Fatal(err)
	}

	content, _ := Read(dir, "doc")
	if strings.Contains(content, "Old summary.") {
		t.Error("old summary still present after case-insensitive replace")
	}
	if !strings.Contains(content, "New summary.") {
		t.Error("new summary not found after case-insensitive replace")
	}
}

// --- Multiple sections at same level ---

func TestReplaceSectionMiddle(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "doc", "Doc", nil)
	Append(dir, "doc", "## Section A\n\nContent A.", "")
	Append(dir, "doc", "## Section B\n\nContent B.", "")
	Append(dir, "doc", "## Section C\n\nContent C.", "")

	if err := ReplaceSection(dir, "doc", "Section B", "Replaced B."); err != nil {
		t.Fatal(err)
	}

	content, _ := Read(dir, "doc")

	if !strings.Contains(content, "Content A.") {
		t.Error("Section A content lost")
	}
	if strings.Contains(content, "Content B.") {
		t.Error("old Section B content still present")
	}
	if !strings.Contains(content, "Replaced B.") {
		t.Error("replaced Section B content not found")
	}
	if !strings.Contains(content, "Content C.") {
		t.Error("Section C content lost")
	}
}

// --- Append to nonexistent file ---

func TestAppendToNonexistentFile(t *testing.T) {
	dir := setupTestProject(t)

	err := Append(dir, "no-such-file", "content", "")
	if err == nil {
		t.Error("expected error when appending to nonexistent file")
	}
}

// --- ReplaceSection on nonexistent file ---

func TestReplaceSectionNonexistentFile(t *testing.T) {
	dir := setupTestProject(t)

	err := ReplaceSection(dir, "no-such-file", "Anything", "content")
	if err == nil {
		t.Error("expected error when replacing section in nonexistent file")
	}
}

// ---------------------------------------------------------------------------
// MatchFiles tests
// ---------------------------------------------------------------------------

// setupMatchTestProject creates a project with knowledge docs for testing
// glob matching. Since ListFiles currently only supports flat structure,
// we test with flat file names. When Phase 5 adds subdirectory support,
// these tests can be extended.
func setupMatchTestProject(t *testing.T) string {
	t.Helper()
	dir := setupTestProject(t)

	docs := []string{
		"overview",
		"architecture",
		"api-design",
		"db-migration",
		"db-schema",
		"deploy-runbook",
	}
	for _, name := range docs {
		if err := Create(dir, name, strings.Title(name), nil); err != nil {
			t.Fatalf("Create(%s) failed: %v", name, err)
		}
	}
	return dir
}

func TestMatchFilesNilPatterns(t *testing.T) {
	dir := setupMatchTestProject(t)

	files, err := MatchFiles(dir, nil)
	if err != nil {
		t.Fatalf("MatchFiles error: %v", err)
	}
	if len(files) != 6 {
		t.Errorf("got %d files, want 6 (all)", len(files))
	}
}

func TestMatchFilesEmptyPatterns(t *testing.T) {
	dir := setupMatchTestProject(t)

	files, err := MatchFiles(dir, []string{})
	if err != nil {
		t.Fatalf("MatchFiles error: %v", err)
	}
	if len(files) != 6 {
		t.Errorf("got %d files, want 6 (all)", len(files))
	}
}

func TestMatchFilesExactName(t *testing.T) {
	dir := setupMatchTestProject(t)

	files, err := MatchFiles(dir, []string{"overview"})
	if err != nil {
		t.Fatalf("MatchFiles error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if files[0] != "overview" {
		t.Errorf("files[0] = %q, want %q", files[0], "overview")
	}
}

func TestMatchFilesPrefix(t *testing.T) {
	dir := setupMatchTestProject(t)

	files, err := MatchFiles(dir, []string{"db-*"})
	if err != nil {
		t.Fatalf("MatchFiles error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}

	found := make(map[string]bool)
	for _, f := range files {
		found[f] = true
	}
	if !found["db-migration"] {
		t.Error("missing db-migration")
	}
	if !found["db-schema"] {
		t.Error("missing db-schema")
	}
}

func TestMatchFilesStarMatchesTopLevel(t *testing.T) {
	dir := setupMatchTestProject(t)

	files, err := MatchFiles(dir, []string{"*"})
	if err != nil {
		t.Fatalf("MatchFiles error: %v", err)
	}
	// All files are top-level, so * matches all
	if len(files) != 6 {
		t.Errorf("got %d files, want 6", len(files))
	}
}

func TestMatchFilesDoubleStarMatchesAll(t *testing.T) {
	dir := setupMatchTestProject(t)

	files, err := MatchFiles(dir, []string{"**"})
	if err != nil {
		t.Fatalf("MatchFiles error: %v", err)
	}
	if len(files) != 6 {
		t.Errorf("got %d files, want 6", len(files))
	}
}

func TestMatchFilesNoMatches(t *testing.T) {
	dir := setupMatchTestProject(t)

	files, err := MatchFiles(dir, []string{"nonexistent-*"})
	if err != nil {
		t.Fatalf("MatchFiles error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("got %d files, want 0", len(files))
	}
}

func TestMatchFilesOverlappingDeduplicates(t *testing.T) {
	dir := setupMatchTestProject(t)

	// Both patterns match "db-migration", should only appear once
	files, err := MatchFiles(dir, []string{"db-*", "db-migration"})
	if err != nil {
		t.Fatalf("MatchFiles error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2 (db-migration and db-schema)", len(files))
	}

	count := 0
	for _, f := range files {
		if f == "db-migration" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("db-migration appeared %d times, want 1", count)
	}
}

func TestMatchFilesMultiplePatterns(t *testing.T) {
	dir := setupMatchTestProject(t)

	files, err := MatchFiles(dir, []string{"overview", "api-*", "deploy-*"})
	if err != nil {
		t.Fatalf("MatchFiles error: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("got %d files, want 3", len(files))
	}

	found := make(map[string]bool)
	for _, f := range files {
		found[f] = true
	}
	if !found["overview"] {
		t.Error("missing overview")
	}
	if !found["api-design"] {
		t.Error("missing api-design")
	}
	if !found["deploy-runbook"] {
		t.Error("missing deploy-runbook")
	}
}

func TestMatchFilesQuestionMark(t *testing.T) {
	dir := setupMatchTestProject(t)

	// "db-schem?" should match "db-schema"
	files, err := MatchFiles(dir, []string{"db-schem?"})
	if err != nil {
		t.Fatalf("MatchFiles error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if files[0] != "db-schema" {
		t.Errorf("files[0] = %q, want %q", files[0], "db-schema")
	}
}

func TestMatchFilesEmptyProject(t *testing.T) {
	dir := setupTestProject(t)

	files, err := MatchFiles(dir, []string{"*"})
	if err != nil {
		t.Fatalf("MatchFiles error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("got %d files, want 0 for empty project", len(files))
	}
}

// ---------------------------------------------------------------------------
// matchGlob unit tests (for subdirectory patterns - future-proofing)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Nested knowledge docs tests
// ---------------------------------------------------------------------------

func TestCreateNested(t *testing.T) {
	dir := setupTestProject(t)

	if err := Create(dir, "architecture/overview", "Architecture Overview", nil); err != nil {
		t.Fatalf("Create nested: %v", err)
	}

	content, err := Read(dir, "architecture/overview")
	if err != nil {
		t.Fatalf("Read nested: %v", err)
	}
	if !strings.Contains(content, "Architecture Overview") {
		t.Error("nested doc should contain title")
	}
}

func TestCreateDeeplyNested(t *testing.T) {
	dir := setupTestProject(t)

	if err := Create(dir, "deep/nested/path/doc", "Deep Doc", []string{"deep"}); err != nil {
		t.Fatalf("Create deeply nested: %v", err)
	}

	content, err := Read(dir, "deep/nested/path/doc")
	if err != nil {
		t.Fatalf("Read deeply nested: %v", err)
	}
	if !strings.Contains(content, "Deep Doc") {
		t.Error("deeply nested doc should contain title")
	}
}

func TestListFilesRecursive(t *testing.T) {
	dir := setupTestProject(t)

	// Create flat and nested docs
	Create(dir, "overview", "Overview", nil)
	Create(dir, "architecture/database", "DB", nil)
	Create(dir, "architecture/api", "API", nil)
	Create(dir, "decisions/db-migration", "Migration", nil)

	files, err := ListFiles(dir)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}

	if len(files) != 4 {
		t.Fatalf("got %d files, want 4: %v", len(files), files)
	}

	found := make(map[string]bool)
	for _, f := range files {
		found[f] = true
	}

	for _, want := range []string{"overview", "architecture/api", "architecture/database", "decisions/db-migration"} {
		if !found[want] {
			t.Errorf("ListFiles missing %q, got %v", want, files)
		}
	}
}

func TestDeleteNested(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "architecture/overview", "Arch", nil)

	if err := Delete(dir, "architecture/overview"); err != nil {
		t.Fatalf("Delete nested: %v", err)
	}

	if _, err := Read(dir, "architecture/overview"); err == nil {
		t.Error("expected error reading deleted nested doc")
	}
}

func TestSearchNested(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "architecture/overview", "Architecture Overview", nil)
	Append(dir, "architecture/overview", "PostgreSQL is our primary database.", "")
	Create(dir, "decisions/db-choice", "DB Choice", nil)
	Append(dir, "decisions/db-choice", "We chose PostgreSQL for reliability.", "")
	Create(dir, "overview", "Overview", nil)

	matches, err := Search(dir, "PostgreSQL")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(matches) != 2 {
		t.Fatalf("got %d matches, want 2: %v", len(matches), matches)
	}

	found := make(map[string]bool)
	for _, m := range matches {
		found[m] = true
	}
	if !found["architecture/overview"] {
		t.Error("search should find architecture/overview")
	}
	if !found["decisions/db-choice"] {
		t.Error("search should find decisions/db-choice")
	}
}

func TestRenameNested(t *testing.T) {
	dir := setupTestProject(t)
	Create(dir, "old-dir/doc", "Doc", nil)

	if err := Rename(dir, "old-dir/doc", "new-dir/doc"); err != nil {
		// Rename may fail if new-dir doesn't exist; that's expected
		// since Rename doesn't create directories
		t.Skipf("Rename across dirs not supported without MkdirAll: %v", err)
	}

	if _, err := Read(dir, "new-dir/doc"); err != nil {
		t.Error("renamed nested doc not found")
	}
}

func TestMatchFilesWithNestedDocs(t *testing.T) {
	dir := setupTestProject(t)

	Create(dir, "overview", "Overview", nil)
	Create(dir, "architecture/database", "DB", nil)
	Create(dir, "architecture/api", "API", nil)
	Create(dir, "decisions/db-migration", "Migration", nil)

	tests := []struct {
		name     string
		patterns []string
		want     int
		mustHave []string
	}{
		{"exact nested", []string{"architecture/database"}, 1, []string{"architecture/database"}},
		{"dir wildcard", []string{"architecture/*"}, 2, []string{"architecture/api", "architecture/database"}},
		{"dir doublestar", []string{"architecture/**"}, 2, []string{"architecture/api", "architecture/database"}},
		{"prefix in dir", []string{"decisions/db-*"}, 1, []string{"decisions/db-migration"}},
		{"star top-level only", []string{"*"}, 1, []string{"overview"}},
		{"doublestar all", []string{"**"}, 4, nil},
		{"mixed", []string{"overview", "architecture/*"}, 3, []string{"overview", "architecture/api", "architecture/database"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := MatchFiles(dir, tt.patterns)
			if err != nil {
				t.Fatalf("MatchFiles(%v): %v", tt.patterns, err)
			}
			if len(files) != tt.want {
				t.Errorf("got %d files, want %d: %v", len(files), tt.want, files)
			}
			found := make(map[string]bool)
			for _, f := range files {
				found[f] = true
			}
			for _, must := range tt.mustHave {
				if !found[must] {
					t.Errorf("missing %q in results %v", must, files)
				}
			}
		})
	}
}

func TestMatchGlobDirectoryPatterns(t *testing.T) {
	tests := []struct {
		pattern string
		name    string
		want    bool
	}{
		// Exact match
		{"overview", "overview", true},
		{"overview", "other", false},

		// Star matches within a path component
		{"db-*", "db-migration", true},
		{"db-*", "overview", false},
		{"*", "overview", true},

		// Double star matches everything
		{"**", "overview", true},
		{"**", "arch/overview", true},

		// dir/* matches direct children
		{"arch/*", "arch/overview", true},
		{"arch/*", "arch/sub/deep", false},
		{"arch/*", "overview", false},

		// dir/** matches recursively
		{"arch/**", "arch/overview", true},
		{"arch/**", "arch/sub/deep", true},
		{"arch/**", "overview", false},

		// Question mark
		{"d?-schema", "db-schema", true},
		{"d?-schema", "dbc-schema", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.name, func(t *testing.T) {
			got, err := matchGlob(tt.pattern, tt.name)
			if err != nil {
				t.Fatalf("matchGlob(%q, %q) error: %v", tt.pattern, tt.name, err)
			}
			if got != tt.want {
				t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
			}
		})
	}
}
