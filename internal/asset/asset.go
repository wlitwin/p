// Package asset provides CRUD operations for project asset files.
// Assets are arbitrary files (images, PDFs, attachments, etc.) stored
// in the assets/ directory of a project.
package asset

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Dir returns the assets directory path for a project.
func Dir(projectDir string) string {
	return filepath.Join(projectDir, "assets")
}

// Path returns the full path for an asset file in a project.
func Path(projectDir, filename string) string {
	return filepath.Join(Dir(projectDir), filename)
}

// Copy copies a file from srcPath into the project's assets directory,
// preserving the original filename. If an asset with the same name already
// exists, it returns an error.
func Copy(projectDir, srcPath string) (string, error) {
	info, err := os.Stat(srcPath)
	if err != nil {
		return "", fmt.Errorf("source file not found: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("source is a directory, not a file")
	}

	filename := filepath.Base(srcPath)
	dstPath := Path(projectDir, filename)

	if _, err := os.Stat(dstPath); err == nil {
		return "", fmt.Errorf("asset %q already exists", filename)
	}

	// Ensure the assets directory exists
	if err := os.MkdirAll(Dir(projectDir), 0o755); err != nil {
		return "", fmt.Errorf("creating assets directory: %w", err)
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("opening source: %w", err)
	}
	defer src.Close() //nolint:errcheck

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("creating destination: %w", err)
	}
	defer dst.Close() //nolint:errcheck

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(dstPath) //nolint:errcheck // best-effort cleanup
		return "", fmt.Errorf("copying file: %w", err)
	}

	return filename, nil
}

// List returns the names of all files in the project's assets directory.
func List(projectDir string) ([]string, error) {
	dir := Dir(projectDir)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading assets directory: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		names = append(names, e.Name())
	}
	return names, nil
}

// Delete removes an asset file from the project's assets directory.
func Delete(projectDir, filename string) error {
	path := Path(projectDir, filename)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("asset %q not found", filename)
	}
	return os.Remove(path)
}

// Info holds metadata about an asset file.
type Info struct {
	Name string
	Size int64
}

// ListWithInfo returns asset files with their sizes.
func ListWithInfo(projectDir string) ([]Info, error) {
	dir := Dir(projectDir)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading assets directory: %w", err)
	}

	var assets []Info
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		assets = append(assets, Info{
			Name: e.Name(),
			Size: info.Size(),
		})
	}
	return assets, nil
}
