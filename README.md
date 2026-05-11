# p — Project Knowledge & Task Manager

A CLI tool for managing projects as collections of **todo lists** and **knowledge documents**, stored as plain markdown files with automatic git version history.

```
$ p status
╭──────────────────────────────────────────────╮
│  3 projects · 12 open · 3 blocked · 47 done  │
╰──────────────────────────────────────────────╯

  api-service    open=5  blocked=1  done=20
  mobile-app     open=4  blocked=2  done=15
  team-docs      open=3  blocked=0  done=12
```

## Why p?

- **Plain markdown** — All data is Obsidian-compatible `.md` files. No database, no lock-in.
- **Git-backed** — Every change auto-commits. Full history, diffs, and undo for free.
- **AI-native** — Built-in Claude integration for planning, reviewing, and even implementing tasks in your code.
- **MCP server** — 22 tools exposed via Model Context Protocol for programmatic access from any AI client.
- **Fast & simple** — Single binary, no dependencies, instant startup.

## Installation

### From source (recommended)

```bash
go install github.com/walter/p@latest
```

### Pre-built binaries

Download from [GitHub Releases](https://github.com/walter/p/releases) — available for Linux, macOS, and Windows (amd64 + arm64).

```bash
# macOS / Linux
curl -Lo p.tar.gz https://github.com/walter/p/releases/latest/download/p_$(uname -s)_$(uname -m).tar.gz
tar xzf p.tar.gz
sudo mv p /usr/local/bin/
```

### Linux packages

`.deb` and `.rpm` packages are available on the [releases page](https://github.com/walter/p/releases).

```bash
# Debian/Ubuntu
sudo dpkg -i p_*.deb

# Fedora/RHEL
sudo rpm -i p_*.rpm
```

### Build from source

```bash
git clone https://github.com/walter/p.git
cd p
go build -o p .
```

## Quick Start

### 1. Initialize

```bash
p init
# → Project root directory: ~/projects
```

### 2. Create a project

```bash
p new my-app --description "Side project tracker"
```

### 3. Add tasks

```bash
p add my-app tasks "Set up CI pipeline" --priority now
p add my-app tasks "Write user auth" --due 2026-05-20
p add my-app tasks "Add metrics dashboard"
```

### 4. Track progress

```bash
p list my-app tasks        # View items
p done my-app tasks 1      # Mark done
p status my-app            # See overview
```

### 5. Use AI

```bash
p ask my-app "What should I work on next?"
p plan my-app "Break down the auth feature into tasks"
p do my-app tasks 2        # AI implements the task in your code repo
```

## Features

### Project Management

```bash
p new <project> [--description "..."]   # Create a project
p list                                   # List all projects
p list --all                             # Include archived projects
p status                                 # Dashboard across all projects
p status <project>                       # Detailed project view
p project archive <project>              # Archive a project
p project unarchive <project>            # Restore archived project
p project set <project> code_dir ~/code  # Link to a code repository
```

### Todo Lists

```bash
# Add items
p add <project> <list> "Task description"
p add <project> <list> "Urgent task" --priority now --due 2026-06-01

# View items
p list <project> <list>                  # One list
p list <project> *                       # All lists
p list <project> <list> --state=open     # Filter by state
p list <project> <list> --priority=now   # Filter by priority
p list <project> <list> --tag=bug        # Filter by tag

# Manage state
p done <project> <list> 1 2 3            # Mark items done (bulk)
p todo block <project> <list> 4          # Mark blocked
p todo open <project> <list> 4           # Reopen

# Organize
p todo priority <project> <list> 1 now   # Set priority
p todo due <project> <list> 1 2026-05-20 # Set due date
p todo tag <project> <list> 1 bug api    # Add tags
p todo move <project> <list> 3 done-list # Move between lists
```

Items support **nested sub-tasks** (IDs like `2.1`, `2.2`) and **inline metadata**:

```
1. [ ] Implement auth  priority=now due=2026-05-20 tags=api,security
  1.1. [ ] Add JWT validation
  1.2. [ ] Write middleware
2. [x] Set up CI  done=2026-05-08
3. [-] Database migration  (blocked)
```

### Knowledge Base

Store project knowledge as markdown documents with frontmatter, tags, and wiki-style `[[links]]`.

```bash
# Create docs
p knowledge create <project> architecture "Architecture Overview"
p knowledge create <project> adr-001 "Use PostgreSQL" --template decision-record

# Browse and search
p knowledge list <project>
p knowledge list <project> --tag architecture
p knowledge search <project> "database"

# Edit content
p edit knowledge append <project> arch "## New Section\n\nContent here"
p edit knowledge replace <project> arch "Updated content" --section "Overview"
p edit open <project> architecture    # Open in $EDITOR
```

Built-in templates: `decision-record`, `meeting-notes`, `runbook`

### Search

```bash
p search <project> "query"    # Search one project
p search "query"              # Search all projects
```

Searches across both todo items and knowledge documents, showing matches with context.

### AI-Powered Workflows

All AI features use the [Claude CLI](https://docs.anthropic.com/en/docs/claude-cli) and communicate through `p`'s built-in MCP server.

```bash
# Ask questions (read-only)
p ask <project> "What's the current status?"
p ask <project> "What are the biggest risks?" --continue   # Resume conversation

# AI planning (creates todos & knowledge docs)
p plan <project> "Break this into milestones for MVP"
p plan <project> "Plan API changes" --also=mobile-app      # Multi-project context

# AI review (analyzes and updates project)
p ai review <project>          # Marks done items, flags blockers, suggests next steps
p ai summarize <project>       # Generates status report

# AI implementation (works in your code repo)
p project set <project> code_dir ~/code/my-app
p do <project> tasks           # AI implements open items
p do <project> tasks 1 2       # AI implements specific items
p do <project> tasks -m "Focus on test coverage"

# Smart add
p add <project> "Fix the race condition in the worker" --ai   # AI picks the right list
p add <project> "https://github.com/org/repo/issues/42"       # URLs auto-process
```

#### Custom AI Prompts

Customize AI behavior per-project with prompt files:

```bash
mkdir -p ~/projects/my-app/.p

# Base prompt (all AI commands)
echo "This is a Go project using chi router. Always write tests." > ~/projects/my-app/.p/prompt.md

# Mode-specific prompts
echo "Run go test ./... after changes." > ~/projects/my-app/.p/prompt-do.md
```

Supported: `prompt.md`, `prompt-do.md`, `prompt-ask.md`, `prompt-plan.md`, `prompt-review.md`, `prompt-summarize.md`, `prompt-add.md`

### History & Version Control

Every mutation auto-commits to the project's git repo.

```bash
p project log <project>              # View commit history
p project log <project> -n 50       # Last 50 commits
p project diff <project>            # Show uncommitted changes
p project revert <project>          # Undo last change
p save <project> updated the docs   # Commit manual edits
```

### MCP Server

`p` includes a built-in [MCP](https://modelcontextprotocol.io/) server with 22 tools, usable by any MCP-compatible AI client:

```bash
p mcp    # Start MCP server on stdio
```

**Available tools:** `todo_list`, `todo_add`, `todo_state`, `todo_update`, `todo_remove`, `todo_move`, `todo_due`, `todo_priority`, `knowledge_create`, `knowledge_read`, `knowledge_append`, `knowledge_replace`, `knowledge_rename`, `knowledge_delete`, `knowledge_list`, `knowledge_search`, `project_list`, `project_create`, `project_archive`, `search`, `status`, `todo_rm_list`

To use with Claude Desktop or other MCP clients, add to your MCP config:

```json
{
  "mcpServers": {
    "p": {
      "command": "p",
      "args": ["mcp"]
    }
  }
}
```

## Configuration

```bash
p config                                  # Show all settings
p config project_root ~/new-projects      # Set project root
p config claude_model claude-sonnet-4-5 # Change AI model
p config default_priority backlog         # Default priority for new items
```

Config is stored at `~/.config/p/config.json` (XDG-compliant).

### Shell Completions

```bash
# Bash
p completion bash > ~/.local/share/bash-completion/completions/p

# Zsh
p completion zsh > ~/.zfunc/_p

# Fish
p completion fish > ~/.config/fish/completions/p.fish
```

Tab completion works for project names, list names, and subcommands.

## Command Reference

| Command | Description |
|---------|-------------|
| `p init` | First-time setup — set project root directory |
| `p new` | Create a new project |
| `p list` | List projects, todo lists, or items (with filters) |
| `p status` | Dashboard of open/blocked/done counts |
| `p show` | Pretty-print a list or knowledge doc |
| `p add` | Add a todo item to a list |
| `p done` | Mark items as done |
| `p search` | Full-text search across projects |
| `p how` | AI-powered help for using `p` |
| **Todo** | |
| `p todo block` | Mark items as blocked |
| `p todo open` | Reopen items |
| `p todo priority` | Set item priority |
| `p todo due` | Set item due date |
| `p todo tag` | Add/remove tags on items |
| `p todo move` | Move items between lists |
| `p todo rm-list` | Delete a todo list |
| `p todo archive-list` | Archive completed lists |
| **Knowledge** | |
| `p knowledge create` | Create a knowledge doc (with optional template) |
| `p knowledge list` | List knowledge docs (with tag filter) |
| `p knowledge search` | Search knowledge doc content |
| `p knowledge delete` | Delete a knowledge doc |
| `p knowledge archive` | Archive a knowledge doc |
| **AI** | |
| `p ask` | Ask questions about project state |
| `p plan` | AI creates tasks and organizes work |
| `p do` | AI implements tasks in your code repo |
| `p ai review` | AI reviews and updates the project |
| `p ai summarize` | AI generates a status report |
| **Project** | |
| `p project new` | Create a project (alias for `p new`) |
| `p project set` | View/set project settings |
| `p project describe` | Set project description |
| `p project archive` | Archive/unarchive a project |
| `p project log` | View git commit history |
| `p project diff` | Show uncommitted changes |
| `p project revert` | Undo the last change |
| **Editing** | |
| `p edit todo add/update/remove/state` | Deterministic todo edit operations |
| `p edit knowledge append/replace/rename` | Deterministic knowledge edit operations |
| `p edit open` | Open a file in `$EDITOR` |
| `p save` | Commit manual edits |
| **Other** | |
| `p mcp` | Start MCP server on stdio |
| `p config` | View/set global configuration |
| `p completion` | Generate shell completions |
| `p version` | Show version and build info |

## Data Format

All data is plain markdown, fully compatible with [Obsidian](https://obsidian.md/).

```
~/projects/
  my-app/                    # Each project is a git repo
    todos/
      tasks.md               # Todo lists are markdown files
      bugs.md
    knowledge/
      architecture.md        # Knowledge docs with YAML frontmatter
      decisions.md
    .p/
      prompt.md              # Optional: custom AI prompts
      settings.json          # Project-level settings
```

**Todo list format:**
```markdown
# tasks

- [ ] Open item  priority=now due=2026-05-20 tags=api
  - [ ] Nested sub-task
- [x] Completed item  done=2026-05-08
- [-] Blocked item
```

**Knowledge doc format:**
```markdown
---
title: Architecture Overview
created: 2026-05-10T18:00:00Z
updated: 2026-05-10T20:00:00Z
tags: [architecture, design]
---

# Architecture Overview

Content with [[wiki-links]] to other docs...
```

## Example Workflows

### Side Project Tracking

```bash
p new side-project --description "Weekend iOS app"
p project set side-project code_dir ~/code/ios-app
p plan side-project "Break this into milestones for an MVP"
p do side-project mvp 1           # AI implements first task
p done side-project mvp 1         # Mark complete
p ai review side-project          # Weekly check-in
```

### Sprint Planning with AI

```bash
p ai summarize api-service                          # Current state
p plan api-service "Create sprint-23 tasks for the next 2 weeks"
p ask api-service "What's at risk this sprint?"     # Mid-sprint check
p ai review api-service                             # End of sprint
p todo archive-list api-service sprint-23           # Clean up
```

### Team Knowledge Base

```bash
p new team-docs --description "Engineering knowledge base"
p knowledge create team-docs onboarding "Onboarding Guide" --tags team,process
p knowledge create team-docs runbook "Deploy Runbook" --template runbook
p search team-docs "deploy"
# Edit in Obsidian, then: p save team-docs updated docs
```

## Requirements

- **Go 1.21+** for building from source
- **Git** for version history (auto-detected)
- **Claude CLI** (optional) for AI features — [install instructions](https://docs.anthropic.com/en/docs/claude-cli)

## License

[MIT](LICENSE) — Walter Litwinczyk
