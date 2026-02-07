# Spec: Optimistic UI

> Spec number: 006
> Status: approved
> Author: jbeckham
> Date: 2026-02-06

## Summary

Make the TUI feel instant by showing what we already know immediately and
fetching the rest in the background. The user should never stare at a blocking
"Loading..." screen when the app already has data to show.

## Goals

- [ ] Open the detail view instantly using list-level data; back-fill full
      detail from the API asynchronously.
- [ ] Returning from the detail view (esc) shows the list immediately with
      stale data; the edited issue is refreshed in the background.
- [ ] Optimistic delete: remove the issue from the list and close the detail
      view immediately, send the API delete in the background. Show an error
      flash if it fails.

## Non-Goals

- Optimistic updates for priority, status, or assignee changes — these are fast
  enough as-is.
- Offline mode or retry queues — a single flash error is sufficient on failure.
- Spinner / loading indicator (deferred to a later spec).

## User Stories

- As a user, I want to press Enter and immediately see the issue detail pre-filled
  with existing data so the navigation feels instant.
- As a user, I want to press Esc from the detail view and immediately see my
  issue list without a loading flash, even if an issue was just edited.
- As a user, I want to press Del → Yes and see the issue vanish from the list
  immediately, without waiting for the API.

## Requirements

### Functional

#### 1. Instant detail view open (Enter)

1. When the user presses Enter on a list issue, push a detail view **immediately**
   with the data already available from the search result (key, summary, status,
   priority, assignee, reporter, type, project, created, updated).
2. Fields that require the full issue fetch (description, subtasks, linked
   issues, parent, labels) should display a small loading placeholder
   (e.g., dim "Loading…") until the API response fills them in.
3. Once `issueDetailMsg` arrives, replace all fields and rebuild the viewport.
4. If the fetch fails, show what we have and display the error in the flash bar.

#### 2. Background refresh on esc-back (Esc)

1. When the user presses Esc from the detail view, pop the stack and show the
   list **immediately** — no `setLoading()`, no full tab reload.
2. If the issue was edited while the detail view was open, apply the edits we
   already know about to the in-memory list data (via `applyIssueUpdate` which
   is already called on every `issueUpdatedMsg`).
3. In the background, re-fetch the single edited issue from the API and patch
   it into the tab data when the response arrives. Only refresh if the detail
   view had an associated edit (track with a flag on the detail view or by
   comparing issue state).
4. Cursor position in the list must be preserved.

#### 3. Optimistic delete (Del → Yes)

1. When the user confirms deletion in the confirm overlay, immediately:
   a. Remove the issue from all tab `issues` slices.
   b. Re-apply filters / rebuild table rows.
   c. Pop the detail view if it's showing the deleted issue.
   d. Move the cursor to the next item (or previous if at the end).
2. Send `cmdDeleteIssue` in the background.
3. On `issueDeletedMsg` with an error:
   a. Show the error in the flash bar.
   b. Do NOT attempt to re-insert the issue (keep it simple).

### Non-Functional

- All existing tests must continue to pass.
- No additional network round-trips — the goal is fewer blocking calls.

## Technical Notes

### Detail view: partial data rendering

The search API only returns fields listed in the `fields` parameter. Tab
configs may only request a subset (e.g., `[key, summary, status]`). To ensure
the detail view always has data for its standard fields, `loadTab` now uses
`mergeSearchFields(cfg.Columns)` which combines configured columns with a base
set: summary, status, priority, issuetype, assignee, reporter, project,
created, updated. Column name "type" is mapped to API name "issuetype"; "key"
is dropped (always returned by Jira).

Fields only available from `GetIssue`: description (ADF), subtasks,
issueLinks, parent, labels. `renderContent()` shows "Loading…" placeholders
for these while `loading == true`, and fills them in once the full fetch
completes.

### Esc-back: surgical refresh

Today pressing Esc does `setLoading()` which wipes tab data and triggers a
full tab reload. Replace this with:
- Pop the view stack.
- If the issue was modified, dispatch `cmdFetchIssue` to refresh just that one
  issue. On `issueDetailMsg` (or a new single-issue refresh message), patch it
  into the tab data via `applyIssueUpdate`.
- The list table stays as-is, cursor preserved.

### Optimistic delete flow

Move the "remove from tabs + pop detail" logic from `issueDeletedMsg` handler
into `handleOverlayResult` for `overlayActionDelete`. The overlay result
handler already runs synchronously before yielding. The `issueDeletedMsg`
handler becomes error-only (flash on failure).

### Create-flow esc-back

`issueCreatedMsg` no longer calls `setLoading()` on the active tab. The tab
reload runs in the background while the tab keeps its existing data visible.
When esc-back pops the detail view, the user sees the list immediately (with or
without the new issue, depending on whether the background reload has completed).

## Open Questions

- None.
