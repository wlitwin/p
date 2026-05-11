# CLI Design

## Command Name

Primary binary: `p`

On first run or via `p init`, the tool can offer to install shell aliases for common operations.

## Configuration

Tool configuration lives in XDG paths:

```
~/.config/p/
  config.yaml          # global config
```

```yaml
# ~/.config/p/config.yaml
project_root: ~/vault           # where projects live
claude_path: claude              # path to claude CLI binary
default_priority: now
```

## Command Structure

Commands are organized into groups. The most-used commands remain at the top level for quick access.

### Top-level Commands (Daily Use)

```bash
p init                              # interactive setup: set project root, etc.
p list                              # list all projects (excludes archived)
p list --all                        # include archived
p list <project>                    # list all todo lists in a project
p list <project> <list|all>         # show items (use 'all' for all lists)
p show <project> <list-or-doc>      # pretty-print a list or knowledge doc
p status [project]                  # overview: open/blocked/done counts
p add <project> [<list>] "<text>"   # add a todo (TUI picker if list omitted)
p add <project> -k "<text or URL>"  # add knowledge
p done <project> <list> <id>...     # mark items done
p search [project] <query>          # full-text search
p save <project> [message...]       # commit manual edits
p config [key] [value]              # view/set global config
p how <question>                    # ask how to do something with p
```

### AI Commands (Top-level)

```bash
p do <project> [list] [ids...] [-m 'msg'] # AI implements todos in code repo
p plan <project> '<desc>' [--also=P] [-y]  # AI planning — creates todos/knowledge
p ask <project> '<question>' [-c]          # read-only AI queries
```

### p project — Project Lifecycle

```bash
p project new <project> [--description ""]   # create a new project
p project archive <project>                  # mark project as archived
p project unarchive <project>                # unarchive
p project set <project> [key] [value]        # view/set project settings
p project describe <project> <text...>       # set project description
p project log <project> [-n COUNT]           # git history
p project diff <project>                     # uncommitted changes
p project revert <project> [-y]              # undo last commit
```

### p todo — Item Management

```bash
p todo block <project> <list> <id>...        # mark blocked
p todo open <project> <list> <id>...         # reopen
p todo priority <project> <list> <id> now|backlog
p todo due <project> <list> <id> YYYY-MM-DD
p todo tag <project> <list> <id> <tags...> [--remove]
p todo move <project> <list> <id> <target-list>
p todo rm-list <project> <list> [-y]         # delete a todo list
p todo archive-list <project> [list] [--restore]  # archive finished lists
```

### p ai — Specialized AI Commands

```bash
p ai review <project> [-y]          # AI reviews and can update project
p ai summarize <project>            # AI-generated status report (read-only)
```

### p knowledge — Knowledge Docs

```bash
p knowledge create <project> <name> <title> [--template T] [--tags a,b]
p knowledge delete <project> <doc> [-y]
p knowledge search <project> <query>
p knowledge list <project> [--tag TAG]
p knowledge archive <project> <doc>
```

### Internal Edit Primitives

These are the structured, deterministic commands used by both the user and the AI agent (via MCP). They are the "sub-tools" that ensure all file mutations go through the tool.

```bash
# Todo operations
p edit todo add <project> <list> "<text>" [--priority now|backlog] [--due YYYY-MM-DD] [--parent <item-id>]
p edit todo update <project> <list> <item-id> "<new text>"
p edit todo state <project> <list> <item-id> open|blocked|done
p edit todo move <project> <list> <item-id> <target-list> [--parent <item-id>]
p edit todo remove <project> <list> <item-id>

# Knowledge operations
p edit knowledge create <project> <filename> "<title>" [--tags tag1,tag2]
p edit knowledge append <project> <filename> "<content>" [--section "Section Name"]
p edit knowledge replace <project> <filename> --section "Section Name" "<new content>"
p edit knowledge move <project> <filename> --section "Section Name" <target-filename>
p edit knowledge rename <project> <old-filename> <new-filename>

# Asset operations
p edit asset add <project> <filepath>     # copy file into project's assets/
```

### Viewing & Status

```bash
p status                              # overview: all projects, open/blocked counts
p status <project>                    # project detail: lists with item counts
p show <project> <list>               # render a todo list
p show <project> -k <filename>        # render a knowledge doc
p search <query>                      # full-text search across all projects
p search <project> <query>            # search within a project
```

### Item Identification

Todo items are identified by their position in the list (1-indexed) when displayed. The tool assigns stable short IDs internally if positional addressing proves fragile. For the first version, positional IDs are simpler:

```bash
p list serviceA db-refactor
# 1. [ ] Audit current schema          priority=now
# 2. [ ] Validate optimistic locking   priority=now due=2026-05-20
#   2.1 [ ] Check conflict rate in logs
#   2.2 [ ] Talk to platform team
# 3. [x] Set up migration framework    done=2026-05-08
# 4. [-] Update ORM mappings           blocked

p done serviceA db-refactor 2.1
```

## Shell Aliases

`p init` or `p aliases` can generate shell aliases:

```bash
# Suggested aliases (user can customize)
alias pa="p add"
alias pl="p list"
alias pd="p done"
alias ps="p status"
```

## TUI Interactions

When information is ambiguous or missing, the tool falls back to interactive TUI prompts rather than failing:

- **List picker**: When adding a todo without specifying a list, show a filterable list of existing lists + "Create new..."
- **Project picker**: When project is omitted and multiple exist, show picker
- **Confirmation**: After AI makes changes, show a summary and optionally the diff before committing
