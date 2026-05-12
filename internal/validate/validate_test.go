package validate

import (
	"errors"
	"strings"
	"testing"
)

func TestProjectName(t *testing.T) {
	valid := []string{"myproject", "my-project", "project_1", "A", "test123"}
	for _, name := range valid {
		if err := ProjectName(name); err != nil {
			t.Errorf("ProjectName(%q) should be valid: %v", name, err)
		}
	}

	invalid := []string{"", "bad name", "bad/name", "bad.name", "-start", "_start", "a b"}
	for _, name := range invalid {
		if err := ProjectName(name); err == nil {
			t.Errorf("ProjectName(%q) should be invalid", name)
		}
	}
}

func TestProjectNameSentinels(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"empty", "", ErrEmpty},
		{"too long", strings.Repeat("a", 65), ErrTooLong},
		{"invalid chars", "bad/name", ErrInvalidChars},
		{"starts with hyphen", "-start", ErrInvalidChars},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ProjectName(tt.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("errors.Is(err, %v) = false; err = %v", tt.wantErr, err)
			}
		})
	}
}

func TestListName(t *testing.T) {
	if err := ListName("valid-list"); err != nil {
		t.Errorf("ListName should accept valid name: %v", err)
	}
	if err := ListName(""); err == nil {
		t.Error("ListName should reject empty string")
	}
	if err := ListName("bad name"); err == nil {
		t.Error("ListName should reject spaces")
	}
}

func TestListNameSentinels(t *testing.T) {
	err := ListName("")
	if !errors.Is(err, ErrEmpty) {
		t.Errorf("expected ErrEmpty, got %v", err)
	}
	err = ListName(strings.Repeat("x", 129))
	if !errors.Is(err, ErrTooLong) {
		t.Errorf("expected ErrTooLong, got %v", err)
	}
	err = ListName("bad name")
	if !errors.Is(err, ErrInvalidChars) {
		t.Errorf("expected ErrInvalidChars, got %v", err)
	}
}

func TestListNameSubdirectories(t *testing.T) {
	valid := []string{
		"backlog",
		"sprint/week-1",
		"team/backend",
		"project/auth/tasks",
		"a/b/c/d/e", // max 5 levels
	}
	for _, name := range valid {
		if err := ListName(name); err != nil {
			t.Errorf("ListName(%q) should be valid: %v", name, err)
		}
	}

	invalid := []string{
		"",
		"../escape",
		"dir/../escape",
		"/leading",
		"trailing/",
		"a//double",
		"a/b/c/d/e/f", // 6 levels, exceeds max
		"dir/.hidden",
		"has space/file",
	}
	for _, name := range invalid {
		if err := ListName(name); err == nil {
			t.Errorf("ListName(%q) should be invalid", name)
		}
	}
}

func TestListNameSentinelsSubdir(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"path traversal", "sprint/../escape", ErrInvalidChars},
		{"leading slash", "/sprint/week-1", ErrInvalidChars},
		{"trailing slash", "sprint/", ErrInvalidChars},
		{"double slash", "sprint//week-1", ErrInvalidChars},
		{"too many levels", "a/b/c/d/e/f", ErrTooLong},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ListName(tt.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("errors.Is(err, %v) = false; err = %v", tt.wantErr, err)
			}
		})
	}
}

func TestFilenameSentinels(t *testing.T) {
	err := Filename("")
	if !errors.Is(err, ErrEmpty) {
		t.Errorf("expected ErrEmpty, got %v", err)
	}
	err = Filename(strings.Repeat("x", 129))
	if !errors.Is(err, ErrTooLong) {
		t.Errorf("expected ErrTooLong, got %v", err)
	}
	err = Filename("bad.name")
	if !errors.Is(err, ErrInvalidChars) {
		t.Errorf("expected ErrInvalidChars, got %v", err)
	}
}

func TestFilenameSubdirectories(t *testing.T) {
	valid := []string{
		"overview",
		"architecture/overview",
		"architecture/database",
		"decisions/db-migration",
		"deep/nested/path/doc",
	}
	for _, name := range valid {
		if err := Filename(name); err != nil {
			t.Errorf("Filename(%q) should be valid: %v", name, err)
		}
	}

	invalid := []string{
		"",
		"../escape",
		"dir/../escape",
		"dir/./same",
		"/absolute",
		"dir//double",
		"dir/",
		"dir/.hidden",
		"has space/file",
		"dir/has.dot",
	}
	for _, name := range invalid {
		if err := Filename(name); err == nil {
			t.Errorf("Filename(%q) should be invalid", name)
		}
	}
}

func TestPriority(t *testing.T) {
	if err := Priority("now"); err != nil {
		t.Error("should accept 'now'")
	}
	if err := Priority("backlog"); err != nil {
		t.Error("should accept 'backlog'")
	}
	if err := Priority("high"); err == nil {
		t.Error("should reject 'high'")
	}
}

func TestPrioritySentinel(t *testing.T) {
	err := Priority("critical")
	if !errors.Is(err, ErrInvalidPriority) {
		t.Errorf("expected ErrInvalidPriority, got %v", err)
	}
}

func TestState(t *testing.T) {
	for _, s := range []string{"open", "blocked", "done"} {
		if err := State(s); err != nil {
			t.Errorf("should accept %q", s)
		}
	}
	if err := State("pending"); err == nil {
		t.Error("should reject 'pending'")
	}
}

func TestStateSentinel(t *testing.T) {
	err := State("pending")
	if !errors.Is(err, ErrInvalidState) {
		t.Errorf("expected ErrInvalidState, got %v", err)
	}
}

func TestDate(t *testing.T) {
	if err := Date("2026-05-10"); err != nil {
		t.Error("should accept valid date")
	}
	if err := Date(""); err != nil {
		t.Error("should accept empty date")
	}
	if err := Date("05-10-2026"); err == nil {
		t.Error("should reject wrong format")
	}
	if err := Date("not-a-date"); err == nil {
		t.Error("should reject non-date")
	}
}

func TestDateSentinel(t *testing.T) {
	err := Date("not-a-date")
	if !errors.Is(err, ErrInvalidDate) {
		t.Errorf("expected ErrInvalidDate, got %v", err)
	}
	err = Date("13-01-2026")
	if !errors.Is(err, ErrInvalidDate) {
		t.Errorf("expected ErrInvalidDate, got %v", err)
	}
}
