# Changelog

All notable changes to **p** are documented in this file.

## v2.1.0 — 2026-05-10

### Added
- `p do` command — spawn Claude to implement todo items in your code repo
- `p how` command — AI-powered contextual help
- `p save` command — commit manual edits to project data
- `p set` command — view/set per-project settings (e.g., `code_dir`)
- `p archive-list` command — archive fully completed todo lists
- Auto-archive: completed todo lists are archived automatically after state changes
- Global `--verbose` flag to control Claude subprocess stderr output
- Project-level locking for concurrency safety
- Full CLI exposed as 22 MCP tools over stdio
- Styled terminal markdown rendering for AI output
- Clickable wiki links in terminal output
- Project path and timestamps shown in `p list` output

### Fixed
- Path traversal validation on all user inputs
- Data loss prevention in concurrent edits (lock race condition)
- UTF-8 handling in todo item text
- Recurring task duplication edge cases
- Git stdout leaking into MCP server responses
- Section heading matching in knowledge docs
- Lock fallthrough on handler errors
- Parser edge cases with malformed metadata

### Changed
- Claude stderr suppressed by default (use `--verbose` to show)
- AI uses user's existing MCP servers when available

## v2.0.0 — 2026-05-10

### Added

#### CLI UX
- `p show` command — pretty-print todo lists and knowledge docs with color
- `p config` command — view/set configuration values
- `p log` command — show git history for a project
- `p diff` command — show uncommitted changes
- `p revert` command — undo the last commit
- `p status` command — dashboard of open items, due dates, recent activity
- Shell completions for bash, zsh, and fish
- Lipgloss-styled colored output throughout
- `p edit` supports knowledge docs (not just todo lists)

#### Task Management
- Tags on todo items (`tags=bug,frontend` inline metadata)
- `p move` command — move/reorder items within or between lists
- `p search` command — full-text search across todos and knowledge
- Recurring tasks (`recur=daily|weekly|monthly` metadata)
- `p list` filter flags (`--state`, `--priority`, `--due`, `--tag`)
- Bulk state changes — mark multiple items done at once
- `p rm-list` command — delete an entire todo list
- `todo_move` MCP tool

#### Knowledge
- `p knowledge delete` command
- Improved `p knowledge list` with tags, dates, sizes
- `p knowledge search` — full-text search across docs
- Tag filtering on knowledge list
- `knowledge_delete` MCP tool
- Template support for knowledge doc creation

#### AI
- `p ask` command — free-form questions about project state (read-only)
- `p summarize` command — AI-generated project summary
- `p review` command — AI reviews recent changes and suggests next steps
- Conversation history persistence across invocations
- Better context injection (recent git history, related docs)
- Multi-project AI reasoning

#### Quality
- Integration tests for all CLI commands
- Unit tests for all 7 internal packages
- 173 tests across 11 packages
- CI pipeline with 60%+ coverage threshold
- Input validation for project names, list names, dates, item IDs
- Improved error messages with suggestions

### Breaking Changes
- MCP tool count increased from 11 to 22 — clients should refresh tool lists
- Todo item metadata format now includes `tags=`, `recur=`, `done=` fields
- Knowledge frontmatter now includes `tags` in YAML header

## v1.0.0 — 2026-05-10

### Added
- Project management: `p init`, `p list`, `p archive`, `p unarchive`
- Todo lists: `p add`, `p done`, `p block`, `p open`, `p priority`, `p due`, `p edit`
- Knowledge docs: `p knowledge create`, `p knowledge append`, `p knowledge replace`, `p knowledge rename`
- Git-backed storage — every mutation auto-commits
- AI integration via `p plan` using Claude CLI subprocess
- MCP server (`p mcp`) exposing 11 tools over stdio
- Bubbletea-based interactive picker for project/list selection
- XDG-compliant configuration
- Nested todo items with positional IDs (e.g., `2.1`, `2.2`)
- Inline metadata on todo items (`priority=now due=2026-05-20`)
- Obsidian-compatible markdown format for all data files
