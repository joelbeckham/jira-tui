# ADR-001: Foundational Architecture Decisions

> Status: accepted
> Date: 2026-02-06

## Context

Before building features, we need to lock down API target, MVP scope,
navigation model, and data strategy. These decisions affect every package.

## Decisions

### 1. Jira Cloud only (REST API v3)

- Target `*.atlassian.net` instances
- Auth: email + API token via Basic Auth
- API v3 uses ADF (Atlassian Document Format) for rich text fields
- No need to abstract over Server/DC APIs — keeps the client simple

### 2. MVP views: Filter Tabs → Issue Detail → Create Issue

- **Filter Tabs (base layer)** — a set of tabs across the top, each backed by a
  Jira saved filter. Configured in `config.yaml`, not the UI. Each tab has:
  - A label (displayed on the tab)
  - A Jira filter ID or filter URL
  - A list of columns to display
  - Sort order
  - Issues render as a scrollable list/table
- **Issue Detail** — pushed onto the stack when selecting an issue from any tab
- **Create Issue** — quick-create an issue from within the TUI
- Sprint Board, JQL Search, and Backlog are deferred to later specs

#### Example config

```yaml
tabs:
  - label: "My Work"
    filter_id: "10100"
    columns: ["key", "summary", "status", "priority", "updated"]
    sort: "updated DESC"
  - label: "Team Review"
    filter_url: "https://mysite.atlassian.net/issues/?filter=10101"
    columns: ["key", "summary", "assignee", "status"]
    sort: "priority ASC"
  - label: "Bugs"
    filter_id: "10102"
    columns: ["key", "summary", "status", "reporter", "created"]
    sort: "created DESC"
```

### 3. Navigation: tabs as base layer + push/pop stack on top

- **Tabs are the base layer** — always visible, not part of the stack
  - Switch tabs with number keys `1`, `2`, `3`, etc.
  - Switching tabs does NOT push/pop — it changes the base view
  - Whichever tab was last active is what you return to when popping the stack
- **Stack sits on top of tabs** — drilling into an issue pushes onto the stack
  - `Enter` on an issue → push Issue Detail onto stack
  - `Esc` → pop back to the tab/issue list you came from
  - Repeatedly hitting `Esc` always lands you back at the active tab
- The root `App` model holds: `activeTab int` + `viewStack []View`
  - When `len(viewStack) == 0`, render the active tab's issue list
  - When `len(viewStack) > 0`, render the top of the stack
  - Tab bar is always visible at the top regardless of stack depth

### 4. Local cache with TTL + global refresh hotkey

- Cache Jira responses locally (in-memory) with a configurable TTL
- Avoids redundant API calls for the same data within a session
- Global `r` hotkey to force-refresh the current view's data
- Cache is per-session (not persisted to disk) — keeps it simple
- TTL default: 5 minutes (configurable in config.yaml)

## Consequences

### Positive

- Filter tabs are infinitely customizable without code changes
- Users can tailor views to their exact workflow via saved Jira filters
- Tab + stack separation is a clean mental model
- Stack navigation is trivial to implement in Bubbletea
- In-memory cache is simple and avoids stale-on-disk bugs
- Cloud-only means one API surface to support

### Negative

- No Server/DC support (acceptable for now)
- Cache adds a layer of indirection in the client
- ADF parsing will be needed for issue descriptions (non-trivial)
- Requires users to set up Jira saved filters first (minor onboarding friction)
