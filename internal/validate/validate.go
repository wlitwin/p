// Package validate provides input validation for user-supplied names, dates,
// priorities, and states. Returned errors wrap package-level sentinel values
// so callers can use errors.Is for programmatic error checking.
package validate

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Sentinel errors for programmatic error checking via errors.Is.
var (
	// ErrEmpty is returned when a required field is empty.
	ErrEmpty = errors.New("value cannot be empty")

	// ErrTooLong is returned when a value exceeds the maximum length.
	ErrTooLong = errors.New("value too long")

	// ErrInvalidChars is returned when a value contains disallowed characters.
	ErrInvalidChars = errors.New("contains invalid characters")

	// ErrInvalidPriority is returned for unrecognized priority values.
	ErrInvalidPriority = errors.New("invalid priority")

	// ErrInvalidState is returned for unrecognized state values.
	ErrInvalidState = errors.New("invalid state")

	// ErrInvalidDate is returned for malformed date strings.
	ErrInvalidDate = errors.New("invalid date")
)

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// filenameRe allows subdirectory paths: each segment must match nameRe rules.
var filenameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*(/[a-zA-Z0-9][a-zA-Z0-9_-]*)*$`)

// ProjectName validates a project name. It must be non-empty, at most 64
// characters, and contain only letters, numbers, hyphens, and underscores.
func ProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty: %w", ErrEmpty)
	}
	if len(name) > 64 {
		return fmt.Errorf("project name too long (max 64 characters): %w", ErrTooLong)
	}
	if !nameRe.MatchString(name) {
		return fmt.Errorf("project name %q contains invalid characters — use letters, numbers, hyphens, underscores: %w", name, ErrInvalidChars)
	}
	return nil
}

// ListName validates a todo list name. Same rules as ProjectName.
func ListName(name string) error {
	if name == "" {
		return fmt.Errorf("list name cannot be empty: %w", ErrEmpty)
	}
	if len(name) > 64 {
		return fmt.Errorf("list name too long (max 64 characters): %w", ErrTooLong)
	}
	if !nameRe.MatchString(name) {
		return fmt.Errorf("list name %q contains invalid characters — use letters, numbers, hyphens, underscores: %w", name, ErrInvalidChars)
	}
	return nil
}

// Filename validates a knowledge document filename. Allows subdirectory paths
// with / separators (e.g. "architecture/overview") but rejects ".." for path
// traversal prevention. Each path segment must start with a letter or digit
// and contain only letters, digits, hyphens, and underscores.
func Filename(name string) error {
	if name == "" {
		return fmt.Errorf("filename cannot be empty: %w", ErrEmpty)
	}
	if len(name) > 128 {
		return fmt.Errorf("filename too long (max 128 characters): %w", ErrTooLong)
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("filename %q contains '..': %w", name, ErrInvalidChars)
	}
	if !filenameRe.MatchString(name) {
		return fmt.Errorf("filename %q contains invalid characters — use letters, numbers, hyphens, underscores, and / for subdirectories: %w", name, ErrInvalidChars)
	}
	return nil
}

// Priority validates a priority value. Must be "now" or "backlog".
func Priority(p string) error {
	if p != "now" && p != "backlog" {
		return fmt.Errorf("invalid priority %q — use 'now' or 'backlog': %w", p, ErrInvalidPriority)
	}
	return nil
}

// State validates an item state value. Must be "open", "blocked", or "done".
func State(s string) error {
	if s != "open" && s != "blocked" && s != "done" {
		return fmt.Errorf("invalid state %q — use 'open', 'blocked', or 'done': %w", s, ErrInvalidState)
	}
	return nil
}

// Date validates a due date string. An empty string is allowed (clears the
// date). Non-empty values must be in YYYY-MM-DD format.
func Date(d string) error {
	if d == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", d); err != nil {
		return fmt.Errorf("invalid date %q — use YYYY-MM-DD format: %w", d, ErrInvalidDate)
	}
	return nil
}
