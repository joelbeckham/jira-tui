# Spec 002 — Filter Tabs + Issue List

> Status: implemented
> Date: 2026-02-06

## Summary

Build the core TUI: a tab bar across the top backed by Jira saved filters,
with each tab displaying an issue list in a scrollable table. This is the
primary interface users will interact with.

## Requirements

### Tab Bar

- Tabs are defined in `config.yaml` under `tabs:`
- Rendered across the top of the screen, always visible
- Active tab is highlighted; inactive tabs are dimmed
- Switch tabs with number keys `1`–`9`
- Tab switching changes the base view, does not push onto the view stack

### Issue List (per tab)

- Each tab calls `GetFilter(filterID)` to get the JQL, then `SearchIssues(jql)`
- Results rendered in a `bubbles/table` with configurable columns
- Column widths are auto-proportional to terminal width:
  - `key`: 12 chars fixed
  - `status`, `priority`, `assignee`, `reporter`: ~15 chars each
  - `summary`: expands to fill remaining width
  - `created`, `updated`: 12 chars
- Scrolling via `j`/`k` (vim) and arrow keys (bubbles default)
- `Enter` on a selected issue will push Issue Detail onto the stack (stub for now)

### Data Loading

- Eager: all tabs load in parallel after auth succeeds
- Each tab shows "Loading..." until its data arrives
- On error (bad filter, permissions), show inline error in the table area
- On empty results, show "No issues found"

### Refresh

- Global `r` hotkey force-refreshes the active tab's data

## Non-goals (this spec)

- Issue detail view (stub only — shows issue key)
- Create issue
- Pagination / infinite scroll
- Custom column widths in config

## Acceptance Criteria

- [ ] Tab bar renders from config, active tab highlighted
- [ ] Number keys switch tabs
- [ ] Each tab fetches filter JQL + issues from Jira
- [ ] Issues render in a table with correct columns
- [ ] Loading, error, and empty states display correctly
- [ ] `r` refreshes the active tab
- [ ] `Enter` pushes a stub view onto the stack, `Esc` pops back
- [ ] All existing + new tests pass
