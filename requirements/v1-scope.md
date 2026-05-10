# v1 Scope

## Goal

A working CLI that can create projects, add todos with AI assistance, and manage a basic knowledge base. No visualization, no web server, no TUI beyond simple pickers.

## In Scope for v1

### Core
- [ ] `p init` — interactive setup (set project root)
- [ ] `p new <project>` — create project with directory structure + git init
- [ ] `p archive / unarchive <project>`
- [ ] `p list` — list projects, lists, items

### Todos
- [ ] `p add <project> [<list>] "<text>"` — add todo, with TUI list picker if list omitted
- [ ] `p done / block / open` — state changes
- [ ] `p priority / due` — metadata updates
- [ ] `p edit todo *` — all deterministic edit primitives
- [ ] Positional item IDs (1, 2, 2.1, 2.2, 3...)
- [ ] Auto-set `created` and `done` dates

### Knowledge
- [ ] `p add <project> -k "<text>"` — AI-assisted knowledge addition
- [ ] `p edit knowledge *` — all deterministic edit primitives
- [ ] Frontmatter auto-management (title, updated, tags)

### AI
- [ ] MCP server mode exposing edit primitives + read-only context tools
- [ ] `claude` subprocess invocation with dynamic system prompt
- [ ] Post-AI diff review and commit/revert flow
- [ ] URL detection: when input is a URL, default to knowledge mode and let AI fetch/summarize

### Git
- [ ] Auto-init repos per project
- [ ] One commit per command
- [ ] Descriptive commit messages

### Config
- [ ] XDG config at `~/.config/p/config.yaml`
- [ ] Project root path setting

## Out of Scope for v1

- Web server / web UI
- TUI dashboard or rich terminal views
- `p search` (full-text search)
- `p status` (aggregate overview)
- Shell alias generation
- Asset management (`p edit asset add`)
- Knowledge reorganization (AI-driven restructuring)
- Todo cross-references and `blocked-by` linking
- `p move` between lists
- `--yes` flag for scripting
- Multiple project roots

## Implementation Order

1. **Config & project scaffolding**: `p init`, `p new`, directory structure, config loading
2. **Todo CRUD**: `p edit todo *` primitives, `p list`, `p done/block/open`, positional IDs
3. **Knowledge CRUD**: `p edit knowledge *` primitives, frontmatter management
4. **Git integration**: auto-commit on every mutation
5. **MCP server**: expose edit + read tools over MCP protocol
6. **AI orchestration**: `p add` with AI, system prompt assembly, diff review
7. **Polish**: TUI picker for list selection, error messages, edge cases

## Tech Stack

- **Language**: Go
- **CLI framework**: Cobra
- **Markdown parsing**: Goldmark (full AST parsing, extensible for custom syntax)
- **TUI**: Bubbletea (Charm) — pickers now, richer views later
- **MCP**: mcp-go (github.com/mark3labs/mcp-go) — handles JSON-RPC/stdio transport
- **Git**: shell out to `git` CLI
- **Build**: Nix flake for reproducible dev shell and builds
- **Testing**: standard Go testing + table-driven tests for markdown parsing
