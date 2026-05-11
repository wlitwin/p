# Usage Guide

This guide covers all `p` commands with examples, organized by workflow.

## Table of Contents

- [Getting Started](#getting-started)
- [Project Management](#project-management)
- [Working with Todo Lists](#working-with-todo-lists)
- [Managing Todo Items](#managing-todo-items)
- [Knowledge Base](#knowledge-base)
- [Search & Navigation](#search--navigation)
- [AI-Powered Workflows](#ai-powered-workflows)
- [History & Version Control](#history--version-control)
- [Configuration](#configuration)
- [Advanced Usage](#advanced-usage)
- [Example Workflows](#example-workflows)

---

## Getting Started

### First-time setup

Initialize `p` by telling it where to store projects:

```bash
p init
# Prompts for: Project root directory: ~/projects
```

This creates a config file at `~/.config/p/config.json` with your project root path. All projects will be subdirectories of this root.

### Create your first project

```bash
p new my-app --description "Mobile app backend"
```

This creates a new directory under your project root with its own git repository. Every change you make through `p` is automatically committed to this repo.

### Quick orientation

```bash
p list                    # See all projects
p status                  # Overview of open/blocked/done counts
p how <question>          # Ask how to do something with p
```

---

## Project Management

### Create projects

```bash
p project new api-service --description "REST API for the mobile app"
p project new design-docs
```

Project names must be alphanumeric with hyphens or underscores (no spaces).

### List projects

```bash
p list                    # Active projects with dates
p list --all              # Include archived projects
```

Output shows each project's name, description, created date, and last-updated date.

### View project status

```bash
p status                  # Summary across all projects
p status api-service      # Detailed view of one project
```

### Set project metadata

```bash
p project describe api-service REST API for the mobile app
p project set api-service code_dir ~/code/api-service
```

The `code_dir` setting links a project to a code repository, enabling the `p do` command.

### View project settings

```bash
p project set api-service                       # Show all settings
p project set api-service code_dir              # Show one setting
```

### Archive/unarchive projects

```bash
p project archive old-project       # Hide from default listing
p project unarchive old-project     # Restore to active listing
p list --all                        # See archived projects
```

---

## Working with Todo Lists

### Add items to a list

```bash
p add api-service tasks "Implement user authentication"
p add api-service tasks "Write integration tests" --priority backlog
p add api-service tasks "Fix login bug" --due 2026-05-15
```

If you omit the list name, an interactive picker lets you choose or create one:

```bash
p add api-service "Design the database schema"
# → Shows picker: tasks, bugs, feature-ideas, + Create new list
```

### List todo lists in a project

```bash
p list api-service
```

Output:
```
Todo lists:
  tasks                 open=5   blocked=1   done=12
  bugs                  open=2   blocked=0   done=8
  backlog               open=15  blocked=0   done=0

Knowledge docs:
  architecture          (2340 bytes)
  decisions             (1205 bytes)
```

### View items in a list

```bash
p list api-service tasks
```

Output:
```
# tasks

  1. [ ] Implement user authentication  priority=backlog
  2. [x] Set up CI pipeline  done=2026-05-08
  3. [-] Migrate to PostgreSQL
  4. [ ] Write integration tests  due=2026-05-15
```

### Filter items

```bash
p list api-service tasks --state=open        # Only open items
p list api-service tasks --state=done        # Only completed items
p list api-service tasks --priority=now      # Only priority=now items
p list api-service tasks --tag=bug           # Only items tagged "bug"
```

### View all items across all lists

```bash
p list api-service *                          # All items in all lists
p list api-service all                        # Same (avoids shell quoting)
p list api-service * --state=open             # Open items across all lists
p list api-service * --priority=now --tag=api # Combined filters
```

### Pretty-print a list or document

```bash
p show api-service tasks                # Show todo list with styling
p show api-service architecture         # Show knowledge doc (auto-detected)
p show api-service architecture -k      # Explicitly show as knowledge doc
```

### Delete a todo list

```bash
p todo rm-list api-service old-tasks         # Prompts for confirmation
p todo rm-list api-service old-tasks -y      # Skip confirmation
```

### Archive completed lists

```bash
p todo archive-list api-service feature-a              # Archive one list
p todo archive-list api-service                        # Auto-archive all 100% done lists
p todo archive-list api-service feature-a --restore    # Restore from archive
```

---

## Managing Todo Items

### Item IDs

Items are identified by their position in the list. Nested items use dot notation:

```
  1. [ ] Parent task
    1.1. [ ] First subtask
    1.2. [ ] Second subtask
  2. [ ] Another task
```

### Change item state

```bash
p done api-service tasks 1              # Mark item #1 as done
p todo block api-service tasks 3        # Mark item #3 as blocked
p todo open api-service tasks 2         # Reopen item #2
```

Bulk state changes — mark multiple items at once:

```bash
p done api-service tasks 1 2 3          # Mark items 1, 2, and 3 as done
```

### Set priority

```bash
p todo priority api-service tasks 1 now      # High priority (default)
p todo priority api-service tasks 1 backlog  # Low priority
```

### Set due date

```bash
p todo due api-service tasks 1 2026-05-20
```

### Add/remove tags

```bash
p todo tag api-service tasks 1 bug frontend          # Add tags
p todo tag api-service tasks 1 --remove bug          # Remove a tag
```

### Move items between lists

```bash
p todo move api-service tasks 3 done-items
# Moves item #3 from "tasks" to "done-items" (creates list if needed)
```

### Update item text

```bash
p edit todo update api-service tasks 1 "New text for this item"
```

### Add nested items

```bash
p edit todo add api-service tasks "Subtask text" --parent 2
# Adds as child of item #2 → becomes item 2.1
```

### Remove items

```bash
p edit todo remove api-service tasks 3
```

---

## Knowledge Base

Knowledge docs are markdown files stored in each project's `knowledge/` directory. They support frontmatter (title, tags, dates) and wiki-style `[[links]]` between documents.

### Create a knowledge doc

```bash
p knowledge create api-service architecture "Architecture Overview"
p knowledge create api-service decisions "Decision Log" --tags architecture,adr
```

### Create from a template

Built-in templates: `decision-record`, `meeting-notes`, `runbook`

```bash
p knowledge create api-service adr-001 "Use PostgreSQL" --template decision-record
p knowledge create api-service standup "Weekly Standup" --template meeting-notes
p knowledge create api-service deploy "Deploy Runbook" --template runbook
```

### List knowledge docs

```bash
p knowledge list api-service
p knowledge list api-service --tag architecture     # Filter by tag
```

### Search knowledge docs

```bash
p knowledge search api-service "database"
```

### Delete a knowledge doc

```bash
p knowledge delete api-service old-notes            # Prompts for confirmation
p knowledge delete api-service old-notes -y         # Skip confirmation
```

### Edit knowledge content

Append content:
```bash
p edit knowledge append api-service architecture "## New Section\n\nContent here"
p edit knowledge append api-service architecture "Under this heading" --section "Deployment"
```

Replace a section:
```bash
p edit knowledge replace api-service architecture "New content" --section "Overview"
```

Rename a document:
```bash
p edit knowledge rename api-service old-name new-name
```

### Open in your editor

```bash
p edit open api-service architecture        # Opens in $EDITOR
p edit open api-service tasks               # Can also open todo lists
```

After editing externally, commit the changes:
```bash
p save api-service updated architecture docs
```

---

## Search & Navigation

### Full-text search

Search across all todos and knowledge docs:

```bash
p search api-service "authentication"     # Search one project
p search "authentication"                 # Search all projects
```

Output shows matching items with their list and ID, plus knowledge doc matches with context snippets.

### Contextual help

```bash
p how do I move items between lists
p how to set up a new project with a code repo
p how can I search for todos tagged as bugs
p how to archive completed work
```

The `p how` command uses AI to answer questions about `p` usage with specific command examples.

---

## AI-Powered Workflows

All AI commands require the `claude` CLI to be installed and configured. AI commands use `p`'s built-in MCP server to read and modify project data.

### Ask questions (read-only)

```bash
p ask api-service "What's the current status of the auth migration?"
p ask api-service "What are the biggest risks right now?"
p ask api-service "What should I work on next?"
p ask api-service "Summarize what we've decided so far" --continue  # Continue conversation
```

The `--continue` / `-c` flag resumes the previous conversation with context.

### AI planning (creates todos and knowledge)

```bash
p plan api-service "Break down the auth migration into concrete tasks"
p plan api-service "Review the current state and suggest what's missing"
p plan api-service "Write up v2 TODOs — v1 is complete" -y  # Auto-confirm
```

Plan mode lets the AI create multiple todo items, organize knowledge, and structure work. You'll see a diff of changes and can confirm or revert.

### Multi-project planning

```bash
p plan api-service "Plan the API changes needed for the mobile app" --also=mobile-app
```

The `--also` flag includes context from other projects so the AI can reason across them.

### AI review (reads + writes)

```bash
p ai review api-service
p ai review api-service -y     # Auto-confirm changes
```

The AI reviews recent git history, current todos, and knowledge docs. It can:
- Mark completed items as done
- Add new todos it identifies
- Update knowledge docs
- Flag blockers and stale items

### AI summary (read-only)

```bash
p ai summarize api-service
```

Generates a comprehensive status report covering project health, progress, blockers, and suggested next steps.

### AI-powered add (smart placement)

```bash
p add api-service "Fix the race condition in the queue worker" --ai
p add api-service "https://github.com/org/repo/issues/42"  # URLs auto-trigger AI+knowledge mode
```

With `--ai`, the AI decides which list to put the item in and how to word it. URLs are automatically processed into knowledge entries.

### AI implementation (p do)

The most powerful AI command — spawns Claude in your code repository to implement todo items:

```bash
# First, link a code directory
p set api-service code_dir ~/code/api-service

# Then use p do
p do api-service                          # Pick a list, AI chooses items
p do api-service tasks                    # AI works on all open items in "tasks"
p do api-service tasks 1 2                # AI works on specific items
p do api-service tasks -m "Focus on tests" # Custom instructions
```

The AI gets full context from your knowledge base and todo lists, works in your code repo, and can mark items done as it completes them.

### Custom AI prompts

Customize AI behavior per-project by creating prompt files:

```bash
mkdir -p ~/projects/api-service/.p

# Base prompt (applies to all AI commands)
cat > ~/projects/api-service/.p/prompt.md << 'EOF'
This is a Go project using the standard library and chi router.
Always write tests alongside implementation code.
Follow the existing code style in the repository.
EOF

# Mode-specific prompts (appended to base)
cat > ~/projects/api-service/.p/prompt-do.md << 'EOF'
Run `go test ./...` after making changes to verify nothing is broken.
EOF
```

Supported prompt files:
- `.p/prompt.md` — Base prompt for all AI commands
- `.p/prompt-do.md` — Additional instructions for `p do`
- `.p/prompt-ask.md` — Additional instructions for `p ask`
- `.p/prompt-plan.md` — Additional instructions for `p plan`
- `.p/prompt-review.md` — Additional instructions for `p ai review`
- `.p/prompt-summarize.md` — Additional instructions for `p ai summarize`
- `.p/prompt-add.md` — Additional instructions for `p add --ai`

---

## History & Version Control

Every mutation in `p` auto-commits to the project's git repo. This gives you full history and undo capabilities.

### View history

```bash
p project log api-service               # Last 20 commits
p project log api-service -n 50         # Last 50 commits
```

Output:
```
a1b2c3d  2 hours ago  p: mark tasks #1 as done
e4f5g6h  3 hours ago  p: add todo "Fix login bug" to tasks
i7j8k9l  yesterday    p: AI plan — Break down auth migration
```

### View uncommitted changes

```bash
p project diff api-service
```

Useful after using `p edit open` to manually edit files — shows what will be committed on the next `p save`.

### Undo the last change

```bash
p project revert api-service            # Shows what will be reverted, asks for confirmation
p project revert api-service -y         # Skip confirmation
```

### Commit manual edits

If you edit files outside of `p` (e.g., in Obsidian or a text editor):

```bash
p save api-service                          # Default commit message
p save api-service updated architecture docs  # Custom message
```

---

## Configuration

### Global config

```bash
p config                                    # Show all settings
p config project_root                       # Show one value
p config project_root ~/new-projects        # Set project root
p config claude_model claude-sonnet-4-5   # Change AI model
p config claude_path /usr/local/bin/claude   # Custom claude binary path
p config default_priority backlog           # Change default priority for new items
```

Config is stored at `~/.config/p/config.json` (XDG compliant).

### Per-project settings

```bash
p project set api-service                           # Show all settings
p project set api-service code_dir ~/code/api       # Link code repository
p project set api-service description "The API"     # Set description
```

### Shell completions

Generate completions for your shell:

```bash
p completion bash > ~/.local/share/bash-completion/completions/p
p completion zsh > ~/.zfunc/_p
p completion fish > ~/.config/fish/completions/p.fish
```

Then restart your shell. Tab completion works for project names, list names, and command names.

### Version info

```bash
p version                   # Full version info
p version --short           # Just the version number
```

---

## Advanced Usage

### MCP Server

`p` includes a built-in MCP (Model Context Protocol) server with 22 tools for programmatic access:

```bash
p mcp    # Starts MCP server on stdio
```

This is used internally by AI commands but can also be connected to any MCP-compatible client. Available tools include:

- `todo_list`, `todo_add`, `todo_state`, `todo_update`, `todo_remove`, `todo_move`, `todo_due`, `todo_priority`
- `knowledge_create`, `knowledge_read`, `knowledge_append`, `knowledge_replace`, `knowledge_rename`, `knowledge_delete`, `knowledge_list`, `knowledge_search`
- `project_list`, `project_create`, `project_archive`
- `search`, `status`

### Deterministic edit primitives

For scripting or automation, use the `p edit` subcommands which provide direct, non-interactive operations:

```bash
# Todo operations
p edit todo add api-service tasks "New item" --priority now --due 2026-05-20
p edit todo add api-service tasks "Subtask" --parent 2
p edit todo update api-service tasks 1 "Updated text"
p edit todo state api-service tasks 1 done
p edit todo remove api-service tasks 3

# Knowledge operations
p edit knowledge create api-service notes "My Notes" --tags dev,notes
p edit knowledge append api-service notes "New content" --section "Details"
p edit knowledge replace api-service notes "Replacement" --section "Overview"
p edit knowledge rename api-service old-name new-name
```

### Verbose mode

Show AI subprocess output for debugging:

```bash
p ask api-service "What's next?" -v
p plan api-service "Plan the sprint" --verbose
```

---

## Example Workflows

### Workflow 1: Managing a side project

```bash
# Set up
p project new side-project --description "Weekend iOS app"
p project set side-project code_dir ~/code/ios-app

# Plan the work
p plan side-project "Break this into milestones for an MVP"

# Add specific tasks
p add side-project mvp "Set up Xcode project with SwiftUI"
p add side-project mvp "Design data model" --priority now
p add side-project mvp "Build main list view" --due 2026-05-20

# Work on items
p do side-project mvp 1

# Track progress
p done side-project mvp 1
p status side-project

# Record decisions
p knowledge create side-project arch "Architecture Decisions" --template decision-record

# Weekly check-in
p ai review side-project
```

### Workflow 2: Sprint planning with AI

```bash
# Start of sprint — review current state
p ai summarize api-service

# Have AI analyze and create sprint tasks
p plan api-service "Create sprint-23 todo list with tasks for the next 2 weeks. \
  Focus on the auth migration and the performance issues from last sprint."

# Tag items for the sprint
p todo tag api-service sprint-23 1 sprint-23
p todo tag api-service sprint-23 2 sprint-23

# Mid-sprint check
p ask api-service "What's at risk for this sprint?"

# End of sprint
p ai review api-service
p todo archive-list api-service sprint-23    # Auto-archives if all done
```

### Workflow 3: Knowledge base for a team

```bash
# Set up the project
p project new team-docs --description "Engineering team knowledge base"

# Create structured docs
p knowledge create team-docs onboarding "Onboarding Guide" --tags team,process
p knowledge create team-docs runbook-deploy "Deploy Runbook" --template runbook
p knowledge create team-docs adr-template "ADR Template" --template decision-record

# Add content via AI
p add team-docs "https://internal.wiki/deploy-process" -k

# Search across everything
p search team-docs "deploy"
p knowledge search team-docs "production"
p knowledge list team-docs --tag process

# Manual editing workflow (Obsidian-compatible)
# Edit files in ~/projects/team-docs/knowledge/ with Obsidian
p save team-docs updated onboarding guide with new team member info
```

### Workflow 4: Bug tracking and triage

```bash
# Quick bug entry
p add api-service bugs "Login fails on Safari — reported by customer X"
p todo tag api-service bugs 1 critical safari

# AI triage
p ask api-service "Which bugs are most critical and what order should we fix them?"

# Work on a bug with AI help
p do api-service bugs 1 -m "Check the auth token handling in Safari"

# Track resolution
p done api-service bugs 1
p knowledge create api-service safari-fix "Safari Login Fix" --tags bugs,postmortem
```

### Workflow 5: Multi-project coordination

```bash
# Plan API changes considering the mobile app's needs
p plan api-service "Design the new auth endpoints" --also=mobile-app

# Ask cross-project questions
p ask api-service "What mobile-app features are blocked on API work?" --also=mobile-app

# Search across all projects
p search "authentication"
```

---

## Tips & Tricks

- **Obsidian compatibility**: Project data is plain markdown. Point Obsidian at your project root for a visual knowledge graph with wiki-link support.
- **Shell aliases**: Create aliases for common operations (`alias pa='p add'`, `alias pl='p list'`).
- **URL auto-detection**: Passing a URL to `p add` automatically switches to knowledge mode with AI processing.
- **Recurring tasks**: Items with `recur=weekly` (or `daily`, `monthly`) automatically reopen when marked done.
- **Wiki links**: Use `[[doc-name]]` in knowledge docs and todo item text to cross-reference documents.
- **Auto-archive**: Run `p todo archive-list <project>` without a list name to automatically archive all 100%-done lists.
- **Project locking**: Concurrent access is safe — `p` uses file locks to prevent conflicts when multiple processes access the same project.
