# p — Project Knowledge & Task Manager

## Build & Test

```bash
go build -o p .          # build binary
go test ./...            # run all tests
go test ./internal/todo/ # run parser tests specifically
```

## Project Structure

- `main.go` — entrypoint
- `cmd/` — Cobra CLI commands (one file per command)
- `internal/config/` — XDG config loading/saving
- `internal/project/` — project CRUD (create, list, archive, resolve)
- `internal/todo/` — todo list parsing, rendering, item CRUD
- `internal/knowledge/` — knowledge doc CRUD, section operations
- `internal/git/` — git helpers (init, commit, diff, revert)
- `internal/ai/` — AI orchestration (claude subprocess, system prompt, MCP config)
- `internal/mcpserver/` — MCP stdio server exposing edit primitives
- `internal/tui/` — Bubbletea picker and input components
- `requirements/` — design docs and requirements

## Key Design Decisions

- All data is markdown files (Obsidian-compatible), no SQLite
- Each project is its own git repo under the configured project root
- Every mutation auto-commits to git
- AI uses `claude` CLI subprocess with `p mcp` as an MCP server
- Todo item IDs are positional (1, 2, 2.1, 2.2)
- Inline metadata on todo items: `priority=now due=2026-05-20 created=2026-05-10`
