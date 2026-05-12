package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
)

func TestLoadTemplateNoFile(t *testing.T) {
	dir := setupProjectDir(t)
	result := LoadTemplate(dir, "ask")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestLoadTemplateEmptyMode(t *testing.T) {
	dir := setupProjectDir(t)
	result := LoadTemplate(dir, "")
	if result != "" {
		t.Errorf("expected empty string for empty mode, got %q", result)
	}
}

func TestLoadTemplateFound(t *testing.T) {
	dir := setupProjectDir(t)
	tmplPath := filepath.Join(dir, ".p", "template-ask.md")
	content := "You are a helpful assistant for {{.ProjectName}}."
	if err := os.WriteFile(tmplPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	result := LoadTemplate(dir, "ask")
	if result != content {
		t.Errorf("got %q, want %q", result, content)
	}
}

func TestLoadTemplateWhitespace(t *testing.T) {
	dir := setupProjectDir(t)
	tmplPath := filepath.Join(dir, ".p", "template-plan.md")
	if err := os.WriteFile(tmplPath, []byte("  \n  template content  \n  "), 0o644); err != nil {
		t.Fatal(err)
	}

	result := LoadTemplate(dir, "plan")
	if result != "template content" {
		t.Errorf("got %q, want %q", result, "template content")
	}
}

func TestExecuteTemplateSimple(t *testing.T) {
	tmpl := "Hello {{.ProjectName}}, mode is {{.Mode}}."
	data := TemplateData{ProjectName: "myproject", Mode: "ask"}

	result, err := ExecuteTemplate(tmpl, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello myproject, mode is ask." {
		t.Errorf("got %q", result)
	}
}

func TestExecuteTemplateAllFields(t *testing.T) {
	tmpl := `Project: {{.ProjectName}}
Dir: {{.ProjectDir}}
Description: {{.ProjectDescription}}
Mode: {{.Mode}}
Input: {{.Input}}
List: {{.ListName}}
Todos: {{.TodoLists}}
Target: {{.TodoList}}
Knowledge: {{.KnowledgeDocs}}
Git: {{.GitLog}}
Custom: {{.CustomPrompt}}`

	data := TemplateData{
		ProjectName:        "test",
		ProjectDir:         "/tmp/test",
		ProjectDescription: "A test project",
		Mode:               "plan",
		Input:              "do stuff",
		ListName:           "sprint-1",
		TodoLists:          "- [ ] item 1",
		TodoList:           "- [ ] specific item",
		KnowledgeDocs:      "# Overview\nSome docs",
		GitLog:             "abc123 fixed bug",
		CustomPrompt:       "Be concise.",
	}

	result, err := ExecuteTemplate(tmpl, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, want := range []string{
		"Project: test",
		"Dir: /tmp/test",
		"Description: A test project",
		"Mode: plan",
		"Input: do stuff",
		"List: sprint-1",
		"Todos: - [ ] item 1",
		"Target: - [ ] specific item",
		"Knowledge: # Overview\nSome docs",
		"Git: abc123 fixed bug",
		"Custom: Be concise.",
	} {
		if !strings.Contains(result, want) {
			t.Errorf("result missing %q", want)
		}
	}
}

func TestExecuteTemplateInvalidSyntax(t *testing.T) {
	tmpl := "Hello {{.Broken"
	data := TemplateData{ProjectName: "test"}

	_, err := ExecuteTemplate(tmpl, data)
	if err == nil {
		t.Error("expected error for malformed template")
	}
}

func TestExecuteTemplateConditional(t *testing.T) {
	tmpl := `{{if .Input}}Question: {{.Input}}{{else}}No input provided.{{end}}`

	// With input
	data := TemplateData{Input: "what is the status?"}
	result, err := ExecuteTemplate(tmpl, data)
	if err != nil {
		t.Fatal(err)
	}
	if result != "Question: what is the status?" {
		t.Errorf("got %q", result)
	}

	// Without input
	data = TemplateData{}
	result, err = ExecuteTemplate(tmpl, data)
	if err != nil {
		t.Fatal(err)
	}
	if result != "No input provided." {
		t.Errorf("got %q", result)
	}
}

func TestBuildTemplateData(t *testing.T) {
	dir := setupProjectDir(t)

	// Create project metadata
	meta := project.ProjectMeta{
		Name:        "test-project",
		Description: "A test project",
	}
	if err := project.SaveMeta(dir, meta); err != nil {
		t.Fatal(err)
	}

	// Create a todo list
	list := &todo.List{Title: "backlog"}
	todo.AddItem(list, "write tests", todo.Now, "")
	if err := todo.SaveList(dir, "backlog", list); err != nil {
		t.Fatal(err)
	}

	// Create a knowledge doc
	if err := knowledge.Create(dir, "overview", "Overview", nil); err != nil {
		t.Fatal(err)
	}

	data := BuildTemplateData("test-project", dir, "ask", "what is the status?", "backlog", nil)

	if data.ProjectName != "test-project" {
		t.Errorf("ProjectName = %q", data.ProjectName)
	}
	if data.ProjectDir != dir {
		t.Errorf("ProjectDir = %q", data.ProjectDir)
	}
	if data.ProjectDescription != "A test project" {
		t.Errorf("ProjectDescription = %q", data.ProjectDescription)
	}
	if data.Mode != "ask" {
		t.Errorf("Mode = %q", data.Mode)
	}
	if data.Input != "what is the status?" {
		t.Errorf("Input = %q", data.Input)
	}
	if data.ListName != "backlog" {
		t.Errorf("ListName = %q", data.ListName)
	}
	if !strings.Contains(data.TodoLists, "write tests") {
		t.Error("TodoLists should contain 'write tests'")
	}
	if !strings.Contains(data.TodoList, "write tests") {
		t.Error("TodoList should contain 'write tests' for specified list")
	}
	if !strings.Contains(data.KnowledgeDocs, "overview") {
		t.Error("KnowledgeDocs should contain 'overview'")
	}
}

func TestBuildTemplateDataNoList(t *testing.T) {
	dir := setupProjectDir(t)

	data := BuildTemplateData("test", dir, "ask", "hello", "", nil)

	if data.TodoList != "" {
		t.Errorf("TodoList should be empty when no list specified, got %q", data.TodoList)
	}
}

func TestBuildTemplateDataWithContextPatterns(t *testing.T) {
	dir := setupProjectDir(t)

	// Create multiple knowledge docs
	for _, name := range []string{"overview", "api-design", "db-schema"} {
		if err := knowledge.Create(dir, name, name, nil); err != nil {
			t.Fatal(err)
		}
	}

	// Only include api-* docs
	data := BuildTemplateData("test", dir, "ask", "", "", []string{"api-*"})

	if !strings.Contains(data.KnowledgeDocs, "api-design") {
		t.Error("KnowledgeDocs should contain api-design")
	}
	if strings.Contains(data.KnowledgeDocs, "overview") {
		t.Error("KnowledgeDocs should NOT contain overview (filtered)")
	}
	if strings.Contains(data.KnowledgeDocs, "db-schema") {
		t.Error("KnowledgeDocs should NOT contain db-schema (filtered)")
	}
}

func TestBuildPromptWithTemplate(t *testing.T) {
	dir := setupProjectDir(t)

	// Write a template that fully replaces the default prompt
	tmplPath := filepath.Join(dir, ".p", "template-ask.md")
	tmpl := `You are a custom assistant for {{.ProjectName}}.

{{.Input}}

{{if .KnowledgeDocs}}## Knowledge
{{.KnowledgeDocs}}{{end}}`

	if err := os.WriteFile(tmplPath, []byte(tmpl), 0o644); err != nil {
		t.Fatal(err)
	}

	task := baseTask(dir)
	task.Mode = ModeAsk
	task.CommandName = "ask"
	task.Input = "what is the status?"

	prompt := buildPrompt(task)

	// Should use template, not default prompt
	if !strings.Contains(prompt, "custom assistant for test-project") {
		t.Error("prompt should use template content")
	}
	if !strings.Contains(prompt, "what is the status?") {
		t.Error("prompt should contain the input")
	}
	// Should NOT contain default prompt markers
	if strings.Contains(prompt, "project knowledge manager") {
		t.Error("prompt should NOT contain default prompt text when template is used")
	}
}

func TestBuildPromptFallsBackOnBadTemplate(t *testing.T) {
	dir := setupProjectDir(t)

	// Write a broken template
	tmplPath := filepath.Join(dir, ".p", "template-ask.md")
	if err := os.WriteFile(tmplPath, []byte("{{.Broken"), 0o644); err != nil {
		t.Fatal(err)
	}

	task := baseTask(dir)
	task.Mode = ModeAsk
	task.CommandName = "ask"

	prompt := buildPrompt(task)

	// Should fall back to default prompt
	if !strings.Contains(prompt, "project knowledge manager") {
		t.Error("should fall back to default prompt on template error")
	}
}
