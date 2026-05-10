package validate

import (
	"fmt"
	"regexp"
	"time"
)

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

func ProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("project name too long (max 64 characters)")
	}
	if !nameRe.MatchString(name) {
		return fmt.Errorf("project name %q contains invalid characters — use letters, numbers, hyphens, underscores", name)
	}
	return nil
}

func ListName(name string) error {
	if name == "" {
		return fmt.Errorf("list name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("list name too long (max 64 characters)")
	}
	if !nameRe.MatchString(name) {
		return fmt.Errorf("list name %q contains invalid characters — use letters, numbers, hyphens, underscores", name)
	}
	return nil
}

func Filename(name string) error {
	if name == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("filename too long (max 64 characters)")
	}
	if !nameRe.MatchString(name) {
		return fmt.Errorf("filename %q contains invalid characters — use letters, numbers, hyphens, underscores", name)
	}
	return nil
}

func Priority(p string) error {
	if p != "now" && p != "backlog" {
		return fmt.Errorf("invalid priority %q — use 'now' or 'backlog'", p)
	}
	return nil
}

func State(s string) error {
	if s != "open" && s != "blocked" && s != "done" {
		return fmt.Errorf("invalid state %q — use 'open', 'blocked', or 'done'", s)
	}
	return nil
}

func Date(d string) error {
	if d == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", d); err != nil {
		return fmt.Errorf("invalid date %q — use YYYY-MM-DD format", d)
	}
	return nil
}
