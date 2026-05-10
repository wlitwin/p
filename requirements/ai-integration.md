# AI Integration

## Architecture

```
User runs `p add serviceA "https://slack.com/thread/..."`
  │
  ├─ Tool parses input, determines AI is needed
  │
  ├─ Tool starts itself as an MCP server (temporary, for this invocation)
  │
  ├─ Tool spawns `claude` subprocess:
  │     claude --mcp-server "p mcp" \
  │       -p "<system prompt with project context and task>"
  │
  ├─ Claude calls MCP tools (p edit knowledge append, p edit todo add, etc.)
  │   These are the same deterministic edit primitives from the CLI
  │
  ├─ Claude finishes, subprocess exits, MCP server stops
  │
  ├─ Tool shows summary of changes (files modified, diff preview)
  │
  └─ Tool commits all changes as a single git commit
```

## When AI Is Used

The tool invokes AI for tasks that require judgment:

| Task | AI Decides | Tool Handles |
|------|-----------|--------------|
| Add todo from free text | Wording, list placement (if ambiguous), nesting, linking to related items | File mutation, metadata (created date), git commit |
| Add knowledge from text/URL | Where in knowledge base it belongs, how to phrase it, whether to create new doc or append to existing | File creation/mutation, frontmatter updates, git commit |
| Add knowledge from URL | Fetching content (via Claude's own MCP tools for Slack/JIRA), summarization, placement | File mutation, asset management, git commit |
| Complete a todo | Whether to update knowledge base, what to write | State change, metadata updates, git commit |
| Reorganize | How to restructure knowledge docs, re-categorize | All file operations, git commit |

The tool does NOT invoke AI for:

- Marking items done/open/blocked (pure state change)
- Moving items between lists (explicit user action)
- Setting priority or due dates
- Creating/archiving projects
- Listing or searching

## MCP Server

When `p` runs as an MCP server, it exposes the `p edit` subcommands as tools:

### Exposed Tools

```
todo_add(project, list, text, priority?, due?, parent_id?)
todo_update(project, list, item_id, text)
todo_state(project, list, item_id, state)
todo_move(project, list, item_id, target_list, parent_id?)
todo_remove(project, list, item_id)

knowledge_create(project, filename, title, tags?)
knowledge_append(project, filename, content, section?)
knowledge_replace(project, filename, section, content)
knowledge_move(project, filename, section, target_filename)
knowledge_rename(project, old_filename, new_filename)

asset_add(project, filepath)

# Read-only tools for context
project_list()
todo_list(project, list?)
knowledge_read(project, filename?)
```

### System Prompt

The system prompt provided to Claude includes:

1. Role description: "You are a project knowledge manager..."
2. Current project structure (list of knowledge docs, todo lists)
3. The specific task (add todo, add knowledge, reorganize, etc.)
4. The user's input text
5. Relevant existing content (e.g., current state of the target knowledge doc)
6. Guidelines for organization (keep docs focused, use wiki links, etc.)

The system prompt is assembled dynamically by the tool based on the command context.

## Agent Confirmation

After the AI makes changes, the tool shows:

```
AI made 3 changes:
  modified  knowledge/architecture.md  (+12 -3 lines)
  modified  knowledge/decisions.md     (+5 lines)
  created   todos/feature-a.md         (+8 lines)

[View diff] [Commit] [Revert]
```

The user can review the diff, commit, or revert. A `--yes` flag skips confirmation for scripting.

## Error Handling

- If `claude` CLI is not installed or not in PATH, show install instructions
- If the subprocess fails (timeout, API error), no files are modified (changes are staged but not committed; tool reverts)
- If the AI produces invalid edits (e.g., references a nonexistent list), the MCP tool returns an error and Claude can retry
