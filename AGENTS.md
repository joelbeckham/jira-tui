# Agent Instructions — jira-tui

> This file is the entry point for any AI coding agent working on this project.
> Read this first, then follow the links below for deeper context.

## Project Overview

**jira-tui** — A fast, minimalist, keyboard-driven Jira TUI built with Go and
Bubbletea. Connects to Jira Cloud (REST API v3) and displays issues in
configurable filter tabs with vim-style navigation.

## Directory Map

```
.agent/              # Agent workspace
  specs/             # Feature and system specifications (agent reads these)
  context/           # Background context, constraints, conventions
  decisions/         # Architecture Decision Records (ADRs)

cmd/jira-tui/        # Application entry point
internal/
  tui/               # Bubbletea models, views, styles
  jira/              # Jira REST API client
  config/            # Configuration loading
```

## Workflow for Agents

1. **Read context first** — check `.agent/context/conventions.md` for coding style,
   tech stack, and project constraints before writing any code.
2. **Consult specs** — feature specs in `.agent/specs/` are the source of truth for
   what should be built.
3. **Record decisions** — when making significant architectural choices, write an
   ADR in `.agent/decisions/`.
4. **Write tests alongside implementation** — use table-driven tests, mock the
   Jira API with `httptest`, test TUI models by calling `Update()` directly.

## Conventions

- See [.agent/context/conventions.md](.agent/context/conventions.md) for full details.
- Write tests alongside implementation.
- Keep commits atomic and well-described.

## Current Status

- **Phase:** Optimistic UI, performance
- **Completed specs:** 001 (foundation), 002 (filter tabs + issue list), 003 (quick filter bar), 005 (quick-create issue), 006 (optimistic UI)
- **Also done:** Priority icons, search API migration, detail view, ADF extraction,
  overlay system (selection/textinput/textarea/confirm), all hotkey editing
  (s/p/d/i/a/t/e/del), user cache, async transition + user + priority fetching,
  clipboard copy (y/u), quick-create issue (c), cursor preservation on update,
  optimistic detail view (instant open with partial data), background refresh on
  esc-back, optimistic delete
- **Next:** Cache layer, column sorting
