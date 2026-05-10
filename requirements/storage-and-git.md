# Storage & Git

## File Format

All data is markdown. No SQLite, no binary formats. The tool reads and writes markdown files directly using structured parsing.

### Markdown Parsing Rules

The tool must parse and emit:

- YAML frontmatter (`---` delimited)
- Checkbox items: `- [ ]`, `- [x]`, `- [-]`
- Inline metadata: `key=value` pairs at the end of a checkbox line
- Indented sub-items (2 spaces per level)
- Obsidian wiki links: `[[target]]`, `[[target|display text]]`, `[[target#heading]]`
- Obsidian tags: `#tagname`
- Obsidian embeds: `![[asset.png]]`
- Standard markdown (headings, lists, tables, code blocks, links)

### What the Tool Owns vs. What the User Owns

| Aspect | Owned by | Notes |
|--------|----------|-------|
| `todos/` file structure | Tool | Tool creates, renames, manages todo list files |
| Todo item metadata | Tool | `created`, `done`, `updated` dates are managed automatically |
| Todo item text | User / AI | Content is written by user or AI |
| `knowledge/` file structure | AI + User | AI organizes, user can also edit directly in Obsidian |
| Knowledge content | AI + User | Both can write; tool updates frontmatter timestamps |
| `.p/config.yaml` | Tool | Tool-managed project metadata |
| `assets/` | Tool | Tool copies files in, manages references |

### Direct Editing

Users can edit any file directly in Obsidian or any text editor. The tool must be resilient to:

- Files being modified outside the tool (don't assume tool-only writes)
- Extra files in `knowledge/` or `assets/` not created by the tool
- Frontmatter fields it doesn't recognize (preserve them)
- Non-standard formatting in knowledge docs (don't reformat unless asked)

Todo list files have a stricter format since the tool needs to parse item state and metadata reliably. The tool should warn if it encounters unparseable lines in todo files.

## Git Integration

### Repository Scope

Each project directory is its own git repository. When `p new <project>` creates a project, it runs `git init` in the project directory if no repo exists.

### Commit Strategy

- One `p` command = one git commit (even if multiple files changed)
- Commit message format: `p: <action description>`
  - `p: add todo "validate optimistic locking" to db-refactor`
  - `p: mark db-refactor #3 as done`
  - `p: add slack thread context to knowledge/architecture`
  - `p: reorganize knowledge base`
- AI-triggered changes include what the AI did: `p: AI added slack thread summary to knowledge/decisions`

### Atomicity

When the AI is making changes via MCP:
1. All file writes happen on disk as the AI works
2. If the AI finishes successfully → stage all changes → commit
3. If the AI fails or user rejects → revert all unstaged changes
4. The tool uses `git stash` or a working-copy approach to ensure atomicity

### No Push

The tool never pushes to a remote. The user manages remote sync themselves (if desired). The tool can note in `p init` that the user can add a remote for backup.

## XDG Configuration

```
~/.config/p/
  config.yaml               # global config (project root path, claude path, defaults)
```

The tool follows XDG Base Directory Specification:
- `$XDG_CONFIG_HOME/p/` (defaults to `~/.config/p/`)
- No data stored in `~/.local/share/p/` for now (all data lives in the project root)
