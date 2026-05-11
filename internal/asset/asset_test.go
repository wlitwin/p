package asset

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func createTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCopy(t *testing.T) {
	dir := setupTestDir(t)
	srcPath := createTempFile(t, "test.png", "fake image data")

	filename, err := Copy(dir, srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if filename != "test.png" {
		t.Errorf("expected filename test.png, got %s", filename)
	}

	// Verify the file was copied
	data, err := os.ReadFile(Path(dir, "test.png"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fake image data" {
		t.Errorf("expected 'fake image data', got %q", string(data))
	}
}

func TestCopyDuplicate(t *testing.T) {
	dir := setupTestDir(t)
	srcPath := createTempFile(t, "test.png", "data")

	if _, err := Copy(dir, srcPath); err != nil {
		t.Fatal(err)
	}

	// Copying again should fail
	_, err := Copy(dir, srcPath)
	if err == nil {
		t.Fatal("expected error for duplicate asset")
	}
}

func TestCopyMissingSource(t *testing.T) {
	dir := setupTestDir(t)

	_, err := Copy(dir, "/nonexistent/file.png")
	if err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestCopyDirectory(t *testing.T) {
	dir := setupTestDir(t)
	srcDir := t.TempDir()

	_, err := Copy(dir, srcDir)
	if err == nil {
		t.Fatal("expected error when source is a directory")
	}
}

func TestList(t *testing.T) {
	dir := setupTestDir(t)

	// Empty assets directory
	names, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Errorf("expected 0 assets, got %d", len(names))
	}

	// Add some files
	os.WriteFile(Path(dir, "image.png"), []byte("png"), 0o644)
	os.WriteFile(Path(dir, "doc.pdf"), []byte("pdf"), 0o644)
	os.WriteFile(Path(dir, ".hidden"), []byte("hidden"), 0o644)

	names, err = List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 {
		t.Errorf("expected 2 assets (excluding hidden), got %d: %v", len(names), names)
	}
}

func TestListNonexistentDir(t *testing.T) {
	dir := t.TempDir() // no assets/ subdirectory

	names, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if names != nil {
		t.Errorf("expected nil, got %v", names)
	}
}

func TestDelete(t *testing.T) {
	dir := setupTestDir(t)
	os.WriteFile(Path(dir, "file.txt"), []byte("data"), 0o644)

	if err := Delete(dir, "file.txt"); err != nil {
		t.Fatal(err)
	}

	// Verify it's gone
	if _, err := os.Stat(Path(dir, "file.txt")); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
}

func TestDeleteNotFound(t *testing.T) {
	dir := setupTestDir(t)

	err := Delete(dir, "nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for missing asset")
	}
}

func TestListWithInfo(t *testing.T) {
	dir := setupTestDir(t)

	os.WriteFile(Path(dir, "small.txt"), []byte("hi"), 0o644)
	os.WriteFile(Path(dir, "bigger.txt"), []byte("hello world"), 0o644)

	infos, err := ListWithInfo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 2 {
		t.Fatalf("expected 2 assets, got %d", len(infos))
	}

	// Verify sizes are populated
	for _, info := range infos {
		if info.Size == 0 {
			t.Errorf("expected non-zero size for %s", info.Name)
		}
	}
}

func TestDir(t *testing.T) {
	got := Dir("/projects/myproject")
	want := filepath.Join("/projects/myproject", "assets")
	if got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
}

func TestPath(t *testing.T) {
	got := Path("/projects/myproject", "image.png")
	want := filepath.Join("/projects/myproject", "assets", "image.png")
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func TestCopyCreatesAssetsDir(t *testing.T) {
	// Test that Copy creates the assets/ directory if it doesn't exist
	dir := t.TempDir() // no assets/ subdirectory
	srcPath := createTempFile(t, "test.txt", "data")

	filename, err := Copy(dir, srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if filename != "test.txt" {
		t.Errorf("expected test.txt, got %s", filename)
	}

	// Verify the assets directory was created
	if _, err := os.Stat(Dir(dir)); err != nil {
		t.Error("expected assets directory to exist")
	}
}
