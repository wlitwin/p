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

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// setupProjectDir creates a minimal project directory structure in a temp dir.
func setupProjectDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"todos", "knowledge", ".p"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatalf("creating subdirectory %s: %v", sub, err)
		}
	}
	return dir
}

func baseTask(dir string) Task {
	return Task{
		ProjectName: "test-project",
		ProjectDir:  dir,
		Input:       "do something useful",
		Mode:        ModeTodo,
	}
}

// ---------------------------------------------------------------------------
// buildPrompt tests
// ---------------------------------------------------------------------------

func TestBuildPromptTodoMode(t *testing.T) {
	dir := setupProjectDir(t)
	task := baseTask(dir)
	task.Mode = ModeTodo

	prompt := buildPrompt(task)

	for _, want := range []string{
		"test-project",
		"todo_add",
		"do something useful",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("todo-mode prompt missing %q", want)
		}
	}
}

func TestBuildPromptPlanMode(t *testing.T) {
	dir := setupProjectDir(t)
	task := baseTask(dir)
	task.Mode = ModePlan

	prompt := buildPrompt(task)

	if !strings.Contains(prompt, "create multiple todo items") ||
		!strings.Contains(prompt, "do something useful") {
		t.Error("plan-mode prompt missing expected plan instructions or user input")
	}
}

func TestBuildPromptAskMode(t *testing.T) {
	dir := setupProjectDir(t)
	task := baseTask(dir)
	task.Mode = ModeAsk

	prompt := buildPrompt(task)

	if !strings.Contains(prompt, "READ-ONLY") {
		t.Error("ask-mode prompt missing READ-ONLY instruction")
	}
}

func TestBuildPromptKnowledgeMode(t *testing.T) {
	dir := setupProjectDir(t)
	task := baseTask(dir)
	task.Mode = ModeKnowledge

	prompt := buildPrompt(task)

	for _, want := range []string{
		"knowledge_append",
		"do something useful",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("knowledge-mode prompt missing %q", want)
		}
	}
}

func TestBuildPromptWithAlsoProjects(t *testing.T) {
	dir := setupProjectDir(t)
	alsoDir := setupProjectDir(t)

	task := baseTask(dir)
	task.AlsoProjects = []string{alsoDir}
	task.AlsoNames = []string{"related-proj"}

	prompt := buildPrompt(task)

	if !strings.Contains(prompt, "Related project: related-proj") {
		t.Error("prompt missing related project heading")
	}
}

// ---------------------------------------------------------------------------
// projectContext tests
// ---------------------------------------------------------------------------

func TestProjectContextEmpty(t *testing.T) {
	dir := setupProjectDir(t)
	task := baseTask(dir)

	ctx := projectContext(task)

	if !strings.Contains(ctx, "No todo lists") {
		t.Error("empty project context should mention 'No todo lists'")
	}
	if !strings.Contains(ctx, "No knowledge docs") {
		t.Error("empty project context should mention 'No knowledge docs'")
	}
}

func TestProjectContextWithData(t *testing.T) {
	dir := setupProjectDir(t)

	// Create a todo list with an item.
	list := &todo.List{Title: "backlog"}
	todo.AddItem(list, "write tests", todo.Now, "")
	if err := todo.SaveList(dir, "backlog", list); err != nil {
		t.Fatalf("saving todo list: %v", err)
	}

	// Create a knowledge doc.
	if err := knowledge.Create(dir, "design", "Design Notes", nil); err != nil {
		t.Fatalf("creating knowledge doc: %v", err)
	}

	// Create project meta with a description.
	meta := project.ProjectMeta{
		Name:        "test-project",
		Description: "A test project for unit tests",
	}
	if err := project.SaveMeta(dir, meta); err != nil {
		t.Fatalf("saving project meta: %v", err)
	}

	task := baseTask(dir)
	ctx := projectContext(task)

	if !strings.Contains(ctx, "backlog") {
		t.Error("context should include the todo list name 'backlog'")
	}
	if !strings.Contains(ctx, "write tests") {
		t.Error("context should include the todo item text 'write tests'")
	}
	if !strings.Contains(ctx, "design.md") {
		t.Error("context should include the knowledge doc 'design.md'")
	}
	if !strings.Contains(ctx, "A test project for unit tests") {
		t.Error("context should include the project description")
	}
}

// ---------------------------------------------------------------------------
// Instructions tests
// ---------------------------------------------------------------------------

func TestTodoInstructions(t *testing.T) {
	task := Task{Input: "fix the login bug"}
	result := todoInstructions(task)

	if !strings.Contains(result, "fix the login bug") {
		t.Error("todo instructions should contain the user input")
	}
	if !strings.Contains(result, "todo_add") {
		t.Error("todo instructions should mention todo_add tool")
	}
}

func TestTodoInstructionsWithListName(t *testing.T) {
	task := Task{Input: "add a test", ListName: "sprint-1"}
	result := todoInstructions(task)

	if !strings.Contains(result, "sprint-1") {
		t.Error("todo instructions should include the specified ListName")
	}
}

func TestTodoInstructionsWithoutListName(t *testing.T) {
	task := Task{Input: "add a test"}
	result := todoInstructions(task)

	if !strings.Contains(result, "most appropriate existing list") {
		t.Error("todo instructions without ListName should suggest choosing an existing list")
	}
}

func TestPlanInstructions(t *testing.T) {
	task := Task{Input: "plan the migration to v2"}
	result := planInstructions(task)

	if !strings.Contains(result, "plan the migration to v2") {
		t.Error("plan instructions should contain the user input")
	}
	if !strings.Contains(result, "create multiple todo items") {
		t.Error("plan instructions should mention creating multiple todo items")
	}
}

func TestAskInstructions(t *testing.T) {
	task := Task{Input: "what is the project status?"}
	result := askInstructions(task)

	if !strings.Contains(result, "READ-ONLY") {
		t.Error("ask instructions should contain READ-ONLY")
	}
	if !strings.Contains(result, "what is the project status?") {
		t.Error("ask instructions should contain the user input")
	}
}

func TestKnowledgeInstructions(t *testing.T) {
	task := Task{Input: "https://example.com/design-doc"}
	result := knowledgeInstructions(task)

	if !strings.Contains(result, "https://example.com/design-doc") {
		t.Error("knowledge instructions should contain the user input")
	}
	if !strings.Contains(result, "knowledge_append") {
		t.Error("knowledge instructions should mention knowledge_append tool")
	}
}

// ---------------------------------------------------------------------------
// MCPConfig test
// ---------------------------------------------------------------------------

func TestMCPConfig(t *testing.T) {
	cfg := MCPConfig("/usr/local/bin/p")

	def, ok := cfg.MCPServers["p"]
	if !ok {
		t.Fatal("MCPConfig should have a server named 'p'")
	}
	if def.Command != "/usr/local/bin/p" {
		t.Errorf("MCPConfig command = %q, want %q", def.Command, "/usr/local/bin/p")
	}
	if len(def.Args) != 1 || def.Args[0] != "mcp" {
		t.Errorf("MCPConfig args = %v, want [mcp]", def.Args)
	}
}

// ---------------------------------------------------------------------------
// processStreamLine tests
// ---------------------------------------------------------------------------

func TestProcessStreamLineText(t *testing.T) {
	// Build a valid assistant event with a text content block.
	line := `{"type":"assistant","message":{"content":[{"type":"text","text":"hello world"}]}}`

	// Redirect stderr to capture output.
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	os.Stderr = w

	// Reset the package-level markdown renderer so it re-initialises
	// against the pipe (it's safe to nil it out in tests).
	mdRenderer = nil

	processStreamLine(line)

	w.Close()
	os.Stderr = oldStderr

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "hello world") {
		t.Errorf("expected stderr to contain 'hello world', got %q", output)
	}
}

func TestProcessStreamLineToolUse(t *testing.T) {
	line := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"mcp__p__todo_add","input":{}}]}}`

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	os.Stderr = w
	mdRenderer = nil

	processStreamLine(line)

	w.Close()
	os.Stderr = oldStderr

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "todo_add") {
		t.Errorf("expected stderr to contain 'todo_add', got %q", output)
	}
}

func TestProcessStreamLineMalformed(t *testing.T) {
	// Should not panic on invalid JSON.
	processStreamLine("this is not json {{{")
}

func TestProcessStreamLineEmpty(t *testing.T) {
	// Should not panic on empty string.
	processStreamLine("")
}

func TestProcessStreamLineResult(t *testing.T) {
	line := `{"type":"result","subtype":"error_max_turns"}`

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	os.Stderr = w
	mdRenderer = nil

	processStreamLine(line)

	w.Close()
	os.Stderr = oldStderr

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "max turns") {
		t.Errorf("expected stderr to contain 'max turns', got %q", output)
	}
}

// ---------------------------------------------------------------------------
// LoadCustomPrompt tests
// ---------------------------------------------------------------------------

func TestLoadCustomPromptNoFiles(t *testing.T) {
	dir := setupProjectDir(t)
	result := LoadCustomPrompt(dir, "ask")
	if result != "" {
		t.Errorf("expected empty string when no prompt files exist, got %q", result)
	}
}

func TestLoadCustomPromptBaseOnly(t *testing.T) {
	dir := setupProjectDir(t)
	promptPath := filepath.Join(dir, ".p", "prompt.md")
	if err := os.WriteFile(promptPath, []byte("This is a Rust project. Use idiomatic Rust patterns."), 0o644); err != nil {
		t.Fatal(err)
	}

	result := LoadCustomPrompt(dir, "ask")
	if result != "This is a Rust project. Use idiomatic Rust patterns." {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestLoadCustomPromptModeOnly(t *testing.T) {
	dir := setupProjectDir(t)
	modePath := filepath.Join(dir, ".p", "prompt-do.md")
	if err := os.WriteFile(modePath, []byte("Always run tests after changes."), 0o644); err != nil {
		t.Fatal(err)
	}

	result := LoadCustomPrompt(dir, "do")
	if result != "Always run tests after changes." {
		t.Errorf("unexpected result: %q", result)
	}

	// Different mode should not pick up prompt-do.md
	result = LoadCustomPrompt(dir, "ask")
	if result != "" {
		t.Errorf("expected empty for mode 'ask', got %q", result)
	}
}

func TestLoadCustomPromptBaseAndMode(t *testing.T) {
	dir := setupProjectDir(t)

	basePath := filepath.Join(dir, ".p", "prompt.md")
	if err := os.WriteFile(basePath, []byte("This is a Go project."), 0o644); err != nil {
		t.Fatal(err)
	}

	modePath := filepath.Join(dir, ".p", "prompt-do.md")
	if err := os.WriteFile(modePath, []byte("Run go test ./... after changes."), 0o644); err != nil {
		t.Fatal(err)
	}

	result := LoadCustomPrompt(dir, "do")
	if !strings.Contains(result, "This is a Go project.") {
		t.Error("result should contain base prompt")
	}
	if !strings.Contains(result, "Run go test ./... after changes.") {
		t.Error("result should contain mode-specific prompt")
	}

	// Verify ordering: base comes first
	baseIdx := strings.Index(result, "This is a Go project.")
	modeIdx := strings.Index(result, "Run go test ./... after changes.")
	if baseIdx >= modeIdx {
		t.Error("base prompt should come before mode-specific prompt")
	}
}

func TestLoadCustomPromptEmptyMode(t *testing.T) {
	dir := setupProjectDir(t)

	basePath := filepath.Join(dir, ".p", "prompt.md")
	if err := os.WriteFile(basePath, []byte("Base instructions here."), 0o644); err != nil {
		t.Fatal(err)
	}

	// Empty mode string should still load base prompt
	result := LoadCustomPrompt(dir, "")
	if result != "Base instructions here." {
		t.Errorf("unexpected result with empty mode: %q", result)
	}
}

func TestLoadCustomPromptWhitespaceOnly(t *testing.T) {
	dir := setupProjectDir(t)

	basePath := filepath.Join(dir, ".p", "prompt.md")
	if err := os.WriteFile(basePath, []byte("  \n  \n  "), 0o644); err != nil {
		t.Fatal(err)
	}

	result := LoadCustomPrompt(dir, "ask")
	if result != "" {
		t.Errorf("whitespace-only file should be treated as empty, got %q", result)
	}
}

func TestBuildPromptWithCustomPrompt(t *testing.T) {
	dir := setupProjectDir(t)

	basePath := filepath.Join(dir, ".p", "prompt.md")
	if err := os.WriteFile(basePath, []byte("This project uses PostgreSQL and follows conventional commits."), 0o644); err != nil {
		t.Fatal(err)
	}

	task := baseTask(dir)
	task.Mode = ModePlan
	task.CommandName = "plan"

	prompt := buildPrompt(task)

	if !strings.Contains(prompt, "Custom instructions") {
		t.Error("prompt should contain 'Custom instructions' section header")
	}
	if !strings.Contains(prompt, "PostgreSQL") {
		t.Error("prompt should contain custom prompt content")
	}
}

func TestBuildPromptWithModeSpecificPrompt(t *testing.T) {
	dir := setupProjectDir(t)

	basePath := filepath.Join(dir, ".p", "prompt.md")
	if err := os.WriteFile(basePath, []byte("We use Go with standard library patterns."), 0o644); err != nil {
		t.Fatal(err)
	}
	modePath := filepath.Join(dir, ".p", "prompt-review.md")
	if err := os.WriteFile(modePath, []byte("Focus on test coverage gaps."), 0o644); err != nil {
		t.Fatal(err)
	}

	task := baseTask(dir)
	task.Mode = ModePlan
	task.CommandName = "review"

	prompt := buildPrompt(task)

	if !strings.Contains(prompt, "We use Go with standard library patterns.") {
		t.Error("prompt should contain base custom instructions")
	}
	if !strings.Contains(prompt, "Focus on test coverage gaps.") {
		t.Error("prompt should contain review-specific instructions")
	}
}

func TestBuildPromptNoCustomPrompt(t *testing.T) {
	dir := setupProjectDir(t)
	task := baseTask(dir)
	task.Mode = ModeAsk

	prompt := buildPrompt(task)

	if strings.Contains(prompt, "Custom instructions") {
		t.Error("prompt should NOT contain 'Custom instructions' when no prompt files exist")
	}
}

// ---------------------------------------------------------------------------
// ResolveContext tests
// ---------------------------------------------------------------------------

func TestResolveContextListContextSet(t *testing.T) {
	dir := setupProjectDir(t)
	list := &todo.List{
		Title:   "Test",
		Context: []string{"arch/*", "design"},
	}

	patterns := ResolveContext(dir, list)
	if len(patterns) != 2 {
		t.Fatalf("got %d patterns, want 2", len(patterns))
	}
	if patterns[0] != "arch/*" || patterns[1] != "design" {
		t.Errorf("patterns = %v, want [arch/* design]", patterns)
	}
}

func TestResolveContextListContextEmpty(t *testing.T) {
	dir := setupProjectDir(t)
	list := &todo.List{
		Title:   "Test",
		Context: []string{}, // explicit empty — means "no knowledge docs"
	}

	patterns := ResolveContext(dir, list)
	if patterns == nil {
		t.Fatal("expected non-nil (empty slice), got nil")
	}
	if len(patterns) != 0 {
		t.Errorf("got %d patterns, want 0", len(patterns))
	}
}

func TestResolveContextFallsBackToProjectDefault(t *testing.T) {
	dir := setupProjectDir(t)

	// Set project default context
	meta := project.ProjectMeta{
		Name:           "test",
		DefaultContext: []string{"overview", "api-*"},
	}
	if err := project.SaveMeta(dir, meta); err != nil {
		t.Fatal(err)
	}

	// List has no context field (nil)
	list := &todo.List{Title: "Test"}

	patterns := ResolveContext(dir, list)
	if len(patterns) != 2 {
		t.Fatalf("got %d patterns, want 2", len(patterns))
	}
	if patterns[0] != "overview" || patterns[1] != "api-*" {
		t.Errorf("patterns = %v, want [overview api-*]", patterns)
	}
}

func TestResolveContextNilListUsesProjectDefault(t *testing.T) {
	dir := setupProjectDir(t)

	meta := project.ProjectMeta{
		Name:           "test",
		DefaultContext: []string{"overview"},
	}
	if err := project.SaveMeta(dir, meta); err != nil {
		t.Fatal(err)
	}

	patterns := ResolveContext(dir, nil)
	if len(patterns) != 1 || patterns[0] != "overview" {
		t.Errorf("patterns = %v, want [overview]", patterns)
	}
}

func TestResolveContextNoContextAnywhere(t *testing.T) {
	dir := setupProjectDir(t)

	// No project default, no list context
	meta := project.ProjectMeta{Name: "test"}
	if err := project.SaveMeta(dir, meta); err != nil {
		t.Fatal(err)
	}

	patterns := ResolveContext(dir, &todo.List{Title: "Test"})
	if patterns != nil {
		t.Errorf("expected nil (include all), got %v", patterns)
	}
}

func TestResolveContextListOverridesProjectDefault(t *testing.T) {
	dir := setupProjectDir(t)

	meta := project.ProjectMeta{
		Name:           "test",
		DefaultContext: []string{"default-*"},
	}
	if err := project.SaveMeta(dir, meta); err != nil {
		t.Fatal(err)
	}

	list := &todo.List{
		Title:   "Test",
		Context: []string{"specific-*"},
	}

	patterns := ResolveContext(dir, list)
	if len(patterns) != 1 || patterns[0] != "specific-*" {
		t.Errorf("patterns = %v, want [specific-*] (list should override project default)", patterns)
	}
}

// ---------------------------------------------------------------------------
// projectContext filtering tests
// ---------------------------------------------------------------------------

func TestProjectContextFiltersKnowledge(t *testing.T) {
	dir := setupProjectDir(t)

	// Create multiple knowledge docs
	for _, name := range []string{"overview", "architecture", "api-design", "db-schema"} {
		if err := knowledge.Create(dir, name, name, nil); err != nil {
			t.Fatalf("creating knowledge doc %s: %v", name, err)
		}
	}

	// Task with context patterns — only include "api-*" and "overview"
	task := baseTask(dir)
	task.ContextPatterns = []string{"api-*", "overview"}

	ctx := projectContext(task)

	if !strings.Contains(ctx, "api-design.md") {
		t.Error("context should include api-design.md (matches api-*)")
	}
	if !strings.Contains(ctx, "overview.md") {
		t.Error("context should include overview.md (exact match)")
	}
	if strings.Contains(ctx, "architecture.md") {
		t.Error("context should NOT include architecture.md (not matched)")
	}
	if strings.Contains(ctx, "db-schema.md") {
		t.Error("context should NOT include db-schema.md (not matched)")
	}
}

func TestProjectContextNilPatternsIncludesAll(t *testing.T) {
	dir := setupProjectDir(t)

	for _, name := range []string{"overview", "architecture", "api-design"} {
		if err := knowledge.Create(dir, name, name, nil); err != nil {
			t.Fatalf("creating knowledge doc %s: %v", name, err)
		}
	}

	// Task with nil context patterns — include all
	task := baseTask(dir)
	task.ContextPatterns = nil

	ctx := projectContext(task)

	if !strings.Contains(ctx, "overview.md") {
		t.Error("nil patterns: context should include overview.md")
	}
	if !strings.Contains(ctx, "architecture.md") {
		t.Error("nil patterns: context should include architecture.md")
	}
	if !strings.Contains(ctx, "api-design.md") {
		t.Error("nil patterns: context should include api-design.md")
	}
}

func TestProjectContextEmptyPatternsNoKnowledge(t *testing.T) {
	dir := setupProjectDir(t)

	for _, name := range []string{"overview", "architecture"} {
		if err := knowledge.Create(dir, name, name, nil); err != nil {
			t.Fatalf("creating knowledge doc %s: %v", name, err)
		}
	}

	// Task with empty context patterns (context: []) means "no knowledge docs"
	task := baseTask(dir)
	task.ContextPatterns = []string{}

	ctx := projectContext(task)

	if strings.Contains(ctx, "overview.md") || strings.Contains(ctx, "architecture.md") {
		t.Error("empty context patterns should include no knowledge docs")
	}
}
