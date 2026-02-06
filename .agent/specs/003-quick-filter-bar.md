# Spec 003 — Quick Filter Bar

> Status: approved
> Date: 2026-02-06

## Summary

Add a client-side quick filter bar above the issue list. The filter narrows
the currently loaded issues by matching typed text against all visible fields.
No new API requests are made — filtering is purely local.

## Requirements

### Activation

- `/` activates the filter bar and focuses the text input
- The filter bar appears between the tab bar and the issue table
- While the filter input is focused, all keypresses go to the text input
  (not to table navigation or tab switching)

### Filtering Behavior

- As the user types, the visible issue list is filtered in real-time
- Matching is case-insensitive substring search across ALL visible columns
  for each issue (e.g., key, summary, status, assignee, etc.)
- An issue is shown if ANY visible field contains the search string
- The table cursor resets to the top when the filter changes
- The filter is per-tab — switching tabs does not carry the filter

### Confirming / Dismissing

- **Enter** while in the filter input: confirms the filter, moves focus back
  to the issue table. The filtered view persists. The filter text remains
  visible in the bar as a reminder.
- **Esc** while in the filter input: clears the filter text entirely, removes
  the filter bar, and returns focus to the issue table (full unfiltered list)
- **Esc** while on the issue table (filter active): clears the filter and
  restores the full list
- **`/` → Enter** (empty filter): clears any active filter

### Display

- Filter bar shows: `/ ` followed by the text input
- When filter is confirmed (input not focused), bar shows the current
  filter text dimmed, as a visual indicator
- Filter match count shown: e.g., "3 of 15 issues"

## Non-goals

- Regex / fuzzy matching
- Persisting filter across tab switches
- Server-side filtering (new API calls)
- Highlighting matched text in the table

## Acceptance Criteria

- [ ] `/` opens the filter bar with a focused text input
- [ ] Typing filters the issue list in real-time across all visible fields
- [ ] Enter confirms the filter and returns focus to the table
- [ ] Esc from the filter input clears the filter and returns to full list
- [ ] Esc from the table (with active filter) clears the filter
- [ ] Switching tabs clears any active filter
- [ ] Filter count (N of M issues) is displayed
- [ ] All existing + new tests pass
