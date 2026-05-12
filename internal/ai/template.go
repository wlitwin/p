package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
)

// TemplateData holds all variables available for prompt template substitution.
// Templates use Go text/template syntax: {{.ProjectName}}, {{.TodoLists}}, etc.
type TemplateData struct {
	// ProjectName is the project's short name.
	ProjectName string
	// ProjectDir is the absolute path to the project directory.
	ProjectDir string
	// ProjectDescription is the project's description from metadata.
	ProjectDescription string
	// Mode is the AI command being run (ask, plan, do, review, summarize, agent).
	Mode string
	// Input is the user's input text (question, plan description, etc.).
	Input string
	// ListName is the target todo list name, if applicable.
	ListName string
	// TodoLists is all todo lists rendered as markdown.
	TodoLists string
	// TodoList is the specific target list rendered as markdown (for do/agent modes).
	TodoList string
	// KnowledgeDocs is all matching knowledge docs rendered as markdown.
	KnowledgeDocs string
	// GitLog is the recent git history.
	GitLog string
	// CustomPrompt is the content of .p/prompt.md + .p/prompt-{mode}.md (the append-style prompts).
	CustomPrompt string
}

// LoadTemplate checks for a template file at .p/template-{mode}.md and returns
// its content if found. Returns empty string if no template exists.
func LoadTemplate(projectDir, mode string) string {
	if mode == "" {
		return ""
	}
	path := filepath.Join(projectDir, ".p", fmt.Sprintf("template-%s.md", mode))
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// ExecuteTemplate parses and executes a Go text/template with the given data.
// Returns the rendered string or an error if the template is malformed.
func ExecuteTemplate(tmpl string, data TemplateData) (string, error) {
	t, err := template.New("prompt").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parsing prompt template: %w", err)
	}
	var sb strings.Builder
	if err := t.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("executing prompt template: %w", err)
	}
	return sb.String(), nil
}

// BuildTemplateData populates a TemplateData struct from project state.
// contextPatterns controls which knowledge docs are included (nil = all).
func BuildTemplateData(projectName, projectDir, mode, input, listName string, contextPatterns []string) TemplateData {
	data := TemplateData{
		ProjectName: projectName,
		ProjectDir:  projectDir,
		Mode:        mode,
		Input:       input,
		ListName:    listName,
	}

	// Project description
	if meta, err := project.LoadMeta(projectDir); err == nil {
		data.ProjectDescription = meta.Description
	}

	// Todo lists
	if names, err := todo.ListNames(projectDir); err == nil && len(names) > 0 {
		var sb strings.Builder
		for _, name := range names {
			list, err := todo.LoadList(projectDir, name)
			if err != nil {
				continue
			}
			fmt.Fprintf(&sb, "**%s**:\n", name)
			sb.WriteString(todo.Render(list))
			sb.WriteString("\n")
		}
		data.TodoLists = sb.String()
	}

	// Specific target list
	if listName != "" {
		if list, err := todo.LoadList(projectDir, listName); err == nil {
			data.TodoList = todo.Render(list)
		}
	}

	// Knowledge docs (filtered by context patterns)
	var files []string
	if contextPatterns != nil {
		if len(contextPatterns) > 0 {
			files, _ = knowledge.MatchFiles(projectDir, contextPatterns)
		}
	} else {
		files, _ = knowledge.ListFiles(projectDir)
	}
	if len(files) > 0 {
		var sb strings.Builder
		for _, f := range files {
			content, err := knowledge.Read(projectDir, f)
			if err != nil {
				continue
			}
			fmt.Fprintf(&sb, "### %s\n\n%s\n\n", f, content)
		}
		data.KnowledgeDocs = sb.String()
	}

	// Git log
	data.GitLog = recentGitLog(projectDir)

	// Custom prompt (append-style, available as a variable in templates too)
	data.CustomPrompt = LoadCustomPrompt(projectDir, mode)

	return data
}
