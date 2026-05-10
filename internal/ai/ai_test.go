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
