package validate

import "testing"

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
