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

### Project Management

```bash
p init                              # interactive setup: set project root, etc.
p new <project> [--description ""]  # create a new project
p archive <project>                 # mark project as archived
p unarchive <project>               # unarchive
p list                              # list all projects (excludes archived)
p list --all                        # include archived
p list <project>                    # list all todo lists in a project
p list <project> <list>             # show items in a specific list
```

### Adding Content

```bash
# Add a todo — if <list> is omitted, show TUI picker to choose/create list
p add <project> [<list>] "<text>"

# Add knowledge — AI decides placement within knowledge base
p add <project> --knowledge "<text or URL>"
p add <project> -k "<text or URL>"

# Shorthand: if input looks like a URL, default to knowledge mode
# (can be overridden with explicit --todo flag)
```

### Editing Todos

```bash
p done <project> <list> <item-id>         # mark done
p block <project> <list> <item-id>        # mark blocked
p open <project> <list> <item-id>         # reopen
p move <project> <list> <item-id> <target-list>  # move to another list
p priority <project> <list> <item-id> now|backlog
p due <project> <list> <item-id> YYYY-MM-DD
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
