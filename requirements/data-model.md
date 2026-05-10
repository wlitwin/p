# Data Model

## Directory Layout

The user configures a root project directory (e.g., pointing to an existing Obsidian vault). All projects live as subdirectories within it.

```
<project-root>/                     # e.g. ~/vault/ or ~/projects/
  serviceA/
    .p/                             # tool metadata for this project
      config.yaml                   # project-level config (archived status, etc.)
    knowledge/
      overview.md
      decisions.md
      architecture.md
      ...                           # AI-organized, grows over time
    todos/
      db-refactor.md                # one file per todo list
      feature-a.md
      onboarding.md
    assets/                         # screenshots, diagrams, etc.
      screenshot-2026-05-10.png
  serviceB/
    .p/
      config.yaml
    knowledge/
      ...
    todos/
      ...
```

Each project is its own git repository (auto-initialized by `p` if not already present).

## Project

A project is a directory under the project root containing a `.p/config.yaml` file.

```yaml
# .p/config.yaml
name: serviceA
created: 2026-05-10T12:00:00Z
archived: false
description: "New payments service"
```

## Knowledge Base

Knowledge files are standard markdown in `knowledge/`. They use Obsidian-compatible formatting:

- `[[wiki links]]` for cross-references within the project
- `#tags` for categorization
- Standard markdown headings, lists, tables
- Frontmatter for metadata

```markdown
---
title: Architecture Overview
updated: 2026-05-10T14:30:00Z
tags: [architecture, database]
---

# Architecture Overview

ServiceA uses a PostgreSQL database with...

## Related

- [[decisions#DB Choice]]
- [[../serviceB/knowledge/api-contract|ServiceB API Contract]]
```

The AI agent is responsible for organizing knowledge files — splitting, merging, restructuring sections as the knowledge base grows. The tool provides the primitives (create file, update section, move section) and the AI decides when and how to use them.

## Todo Lists

Each todo list is a markdown file in `todos/`. The file represents a single thread/topic within the project.

### Todo List Format

```markdown
---
title: DB Refactor
created: 2026-05-10T12:00:00Z
updated: 2026-05-10T15:00:00Z
---

# DB Refactor

- [ ] Audit current schema for unused columns priority=now
- [ ] Validate if optimistic locking is needed priority=now due=2026-05-20
  - [ ] Check current conflict rate in prod logs
  - [ ] Talk to platform team about their approach — see [[knowledge/decisions#Locking Strategy]]
- [x] Set up migration framework priority=now done=2026-05-08
- [-] Update ORM mappings priority=backlog blocked-by=[[todos/feature-a#New entity models]]
  Blocked on new entity models being finalized first.
```

### Todo Item Syntax

```
- [ ] <text> [key=value...]           # open
- [x] <text> [key=value...]           # done
- [-] <text> [key=value...]           # blocked

Supported metadata keys:
  priority=now|backlog                 # default: now
  due=YYYY-MM-DD                       # optional due date
  created=YYYY-MM-DD                   # set automatically by tool
  done=YYYY-MM-DD                      # set automatically when completed
  blocked-by=<reference>               # wiki link to blocking item
```

Nesting is represented by indentation (2 spaces per level). Sub-items inherit the parent's context but have independent state.

### Cross-references

Todo items can reference:
- Knowledge docs: `[[knowledge/decisions#Locking Strategy]]`
- Other todo items: `[[todos/feature-a#New entity models]]`
- External URLs: plain `https://...` links (JIRA, Slack, etc.)

## Assets

Binary files (screenshots, diagrams) live in `assets/` and are referenced from knowledge or todo files using standard markdown image syntax or Obsidian embeds (`![[assets/diagram.png]]`).
