# p — Project Knowledge & Task Manager

## Vision

`p` is a CLI tool for managing project knowledge bases and task lists. It combines a structured markdown wiki with hierarchical todo tracking, using AI to handle unstructured input (Slack threads, ad-hoc notes) and organize it into the knowledge base.

## Core Principles

- **Markdown-native**: All data is structured markdown, compatible with Obsidian for browsing and searching. The tool is opinionated about layout but the files are always human-readable.
- **Deterministic edits**: The tool itself handles all file mutations (adding todos, updating metadata, moving items). Edits are structured and predictable. AI is only used for "messy" decisions — wording, summarization, placement of knowledge.
- **Git-backed history**: Every `p` command that mutates state produces a single git commit. History is the git log. No custom changelog.
- **Single user**: No collaboration features. File-based storage for easy backup and migration.
- **CLI-first**: Primary interface is the terminal. Visualization (TUI, web) is additive, not required.

## What `p` Is Not

- Not a full project management suite (no sprints, no velocity, no burndown)
- Not a note-taking app (use Obsidian directly for freeform notes)
- Not a JIRA/Linear replacement (it links *to* those tools)

## Key Concepts

| Concept        | Description                                                                 |
|----------------|-----------------------------------------------------------------------------|
| **Project**    | Long-lived container (e.g., "ServiceA"). Has its own knowledge base and todo lists. Can be archived. |
| **Knowledge**  | Wiki-style markdown docs within a project. Organized by the AI agent. Covers decisions, requirements, architecture, context. |
| **Todo List**  | A named list of tasks within a project, representing a thread/topic (e.g., "DB Refactor", "Feature A"). |
| **Todo Item**  | A single actionable task. Has state (open/blocked/done), priority (now/backlog), optional due date. Can nest and cross-reference. |

## Typical Workflows

### Adding a task
```
p add serviceA "validate if optimistic locking is needed"
# → TUI picker shows existing lists or [create new]
# → AI decides placement within the list, wording, nesting
# → Commits change
```

### Adding knowledge from external source
```
p add serviceA "https://slack.com/link/to/thread add this context"
# → AI fetches/summarizes the Slack thread (via its own MCP tools)
# → AI decides where in the knowledge base this belongs
# → Tool writes structured markdown, commits
```

### Completing a task
```
p done serviceA db-refactor 3
# → Marks item 3 as done
# → Optionally: AI reviews what was accomplished and updates knowledge base
```

### Browsing
```
p list                        # all projects
p list serviceA               # all lists in serviceA
p list serviceA db-refactor   # items in the db-refactor list
p status                      # summary across all projects (open/blocked counts)
```
