# Spec 004 — Issue Detail View & Hotkey Editing

> Status: draft
> Date: 2025-07-17

## Summary

Replace the stub detail view with a full issue detail screen. Display all
important fields in a readable layout with scrolling support. Integrate the
project's hotkey system so users can edit fields (status, priority, title,
description, assignee) from both the list view and the detail view. Add
quick-create for new issues with sensible defaults.

## Goals

- [ ] Full read-only detail layout (all core fields + relations)
- [ ] Hotkey-driven inline field editing (s/p/d/e/t/i/a)
- [ ] Delete issue with confirmation (del)
- [ ] Quick-create new issue (c) with defaults from config
- [ ] New Jira API methods (GetIssue, UpdateIssue, CreateIssue, GetTransitions,
      TransitionIssue, DeleteIssue, SearchUsers)

## Non-Goals

- Editing subtasks, linked issues, labels, or parent (read-only display only)
- Rich-text / ADF description editing (plain text for now)
- Attachment viewing or uploading
- Comment viewing or adding
- Bulk editing
- Column sorting (Shift+number) — separate spec

---

## Part 1 — Detail View Layout

### Opening the Detail View

When the user presses **Enter** on a highlighted issue in the list view, the
TUI fetches the full issue (GET /rest/api/3/issue/{key}) and pushes the detail
view onto the view stack. A loading indicator is shown while the fetch is in
flight. If the search-result data is sufficient (all displayed fields are
already present), skip the extra fetch and open immediately.

### Layout

The detail view is a single scrollable pane with the following sections top to
bottom:

```
┌──────────────────────────────────────────────────┐
│  PROJ-123  ▸ Epic Name (if parent exists)        │  ← header row
│  Bug · In Progress · ≡ Medium                    │  ← type · status · priority
│                                                  │
│  Title (editable via `t`)                        │
│  Fix the login redirect loop                     │
│                                                  │
│  Description (editable via `e`)                  │
│  When a user tries to log in with SSO...         │
│  (multi-line, wrapped to terminal width)         │
│                                                  │
│  ─────────── Fields ───────────                  │
│  Assignee    Jane Doe                            │
│  Reporter    John Smith                          │
│  Labels      bug, auth, sso                      │
│  Created     2025-07-01 10:23                    │
│  Updated     2025-07-15 14:05                    │
│                                                  │
│  ─────────── Subtasks (3) ───────────            │
│  ✓ PROJ-124  Write unit tests                    │
│  · PROJ-125  Update docs                         │
│  · PROJ-126  Deploy to staging                   │
│                                                  │
│  ─────────── Linked Issues (2) ──────────        │
│  blocks     PROJ-200  API rate limiting           │
│  is blocked by  PROJ-180  Auth service deploy    │
│                                                  │
│  ─────────── Parent ───────────                  │
│  PROJ-100  Epic: Authentication overhaul          │
└──────────────────────────────────────────────────┘
```

### Scrolling

The detail view must scroll with **j/k** (or arrow keys) when content exceeds
the terminal height. Use a viewport model (lipgloss viewport or custom) to
handle this.

### Field Display Rules

| Field        | Display                                              |
|------------- |------------------------------------------------------|
| Key          | Bold, styled with `titleStyle`                       |
| Parent       | Shown as `▸ PARENT-KEY Parent summary` after the key |
| Type         | Issue type name (Bug, Story, Task, etc.)             |
| Status       | Status name, colored by category (blue/yellow/green) |
| Priority     | `priorityLabel()` — styled icon + name               |
| Summary      | Plain text, full width                               |
| Description  | Multi-line plain text (ADF → plain text extraction)  |
| Assignee     | Display name, or "Unassigned"                        |
| Reporter     | Display name                                         |
| Labels       | Comma-separated list, or "None"                      |
| Created      | Formatted date (e.g., "2025-07-01 10:23")            |
| Updated      | Formatted date                                       |
| Subtasks     | List of `status-icon KEY Summary`                    |
| Linked Issues| Grouped by link type with direction label            |
| Parent       | `KEY Summary` link                                   |

### ADF Description Handling

Jira Cloud returns descriptions in Atlassian Document Format (ADF), a JSON
document. For this iteration, extract plain text by recursively walking the
ADF tree and concatenating `text` nodes, inserting newlines at paragraph
boundaries. A full ADF renderer is out of scope.

---

## Part 2 — Hotkey System

Hotkeys apply to the **focused issue**: in the list view that's the highlighted
row; in the detail view that's the displayed issue.

### Hotkey Table

| Key   | Action                  | Behavior                                    |
|-------|-------------------------|---------------------------------------------|
| `s`   | Change status           | Fetch transitions → show selection list → execute transition on Enter, Esc aborts |
| `p`   | Change priority         | Show priority selection list → PUT on Enter, Esc aborts |
| `d`   | Mark as done            | Find the "done" category transition → execute immediately, show error if none |
| `e`   | Edit description        | Open full-screen text editor overlay, Enter saves, Esc aborts |
| `t`   | Edit title              | Show single-line text input pre-filled with current title, Enter saves, Esc aborts |
| `i`   | Assign to me            | PUT assignee = current user's accountId immediately |
| `a`   | Choose assignee         | Show user search/select list → PUT on Enter, Esc aborts |
| `del` | Delete issue            | Show "Delete PROJ-123? (y/n)" confirmation, `y` deletes, `n`/`Esc` aborts |
| `c`   | Create issue            | Create with defaults → open detail view of new issue |

### Overlay Architecture

Hotkey actions that require user input (s, p, a, t, e) push a temporary
**overlay** on top of the current view. The overlay captures all input until
dismissed. Overlays are NOT full stack views — they float on top and render
as a centered box or inline widget.

Overlay types:

1. **Selection list** — for status, priority, assignee. A filterable list of
   options. Enter selects, Esc dismisses. Arrow keys / j/k navigate.
2. **Text input** — for title. Single-line input pre-filled with current value.
   Enter saves, Esc aborts.
3. **Text editor** — for description. Multi-line text area. Ctrl+S / Enter
   saves, Esc aborts. (Start simple: a multi-line text input.)
4. **Confirmation** — for delete. "Delete PROJ-123? (y/n)" inline prompt.

### Overlay Implementation

```go
// overlay is a transient input capture that floats on top of any view.
type overlay interface {
    Update(tea.Msg) (overlay, tea.Cmd)
    View() string
    // done returns true when the overlay should be dismissed.
    // result is nil if aborted, or contains the user's selection/input.
    done() (bool, interface{})
}
```

Key routing order with overlays:

1. `ctrl+c` → always quit
2. Active overlay → all keys routed to overlay
3. View stack → esc pops
4. Filter focused → filter keys
5. Tab-level keys

### Hotkey in List View vs Detail View

The same hotkey handler runs regardless of which view has focus. The handler
resolves the "target issue" from context:

- **List view**: `a.tabs[a.activeTab].selectedIssue()`
- **Detail view**: `a.viewStack[top].issue`

After a successful edit, the TUI should:

1. Update the issue in the local tab data (optimistic update)
2. If in detail view, update the displayed issue
3. Optionally refresh the tab in the background

### Status Change (s)

1. Call `GET /rest/api/3/issue/{key}/transitions` to get available transitions
2. Display as a selection list overlay
3. On Enter: call `POST /rest/api/3/issue/{key}/transitions` with the selected
   transition ID
4. Update local issue status to the transition's `to` status

### Mark as Done (d)

1. Call `GET /rest/api/3/issue/{key}/transitions`
2. Find the first transition whose `to.statusCategory.key == "done"`
3. If found, execute it immediately
4. If not found, show an error in the status bar

### Priority Change (p)

1. Show a hardcoded list of priorities (can be fetched from
   `GET /rest/api/3/priority` later, but hardcode for now: Highest, High,
   Medium, Low, Lowest)
2. On Enter: call `PUT /rest/api/3/issue/{key}` with
   `{"fields": {"priority": {"name": "High"}}}`

### Assign to Me (i)

1. Call `PUT /rest/api/3/issue/{key}/assignee` with
   `{"accountId": "<current user accountId>"}`
2. Update local issue assignee to current user
3. No overlay needed — immediate action

### Choose Assignee (a)

User list is loaded at startup and cached to a local file. If the cache file
exists, it is used directly (no API call). The user deletes the file to force
a refresh. Deactivated users are excluded.

1. Show a filterable selection list of cached users
2. As user types, filter the local list (no API calls)
3. On Enter: call `PUT /rest/api/3/issue/{key}/assignee` with selected user's
   accountId
4. On Esc: abort

#### User Cache

- On startup, if `~/.config/jira-tui/users.json` exists, load it
- If it doesn't exist, call `GET /rest/api/3/users/search?maxResults=1000`
  (paginate if needed), filter out `active == false`, write to `users.json`
- The cache is a flat JSON array of `{accountId, displayName, emailAddress}`
- To refresh: delete `users.json` and restart

### Edit Title (t)

1. Show a single-line text input pre-filled with current summary
2. On Enter: call `PUT /rest/api/3/issue/{key}` with
   `{"fields": {"summary": "<new title>"}}`

### Edit Description (e)

1. Show a multi-line text area pre-filled with current plain-text description
2. On **Ctrl+Enter**: save — call `PUT /rest/api/3/issue/{key}` with
   `{"fields": {"description": <ADF document>}}`
3. On **Esc**: abort, discard changes
4. Convert the plain text back to a simple ADF paragraph document for the API

### Delete Issue (del)

1. Show inline confirmation: "Delete PROJ-123? (y/n)"
2. On `y`: call `DELETE /rest/api/3/issue/{key}?deleteSubtasks=true`
3. Pop the detail view (if open), remove the issue from the tab's local data
4. On `n` or Esc: dismiss

---

## Part 3 — Quick Create

### Trigger

Press **`c`** from the list view (not from detail view to avoid conflicts with
future keybindings).

### Flow

1. Immediately create an issue via `POST /rest/api/3/issue` with defaults:
   - **Project**: from `config.yaml` → `jira.default_project` (project key)
   - **Issue type**: "Task" (or first available type)
   - **Summary**: "New issue" (placeholder)
   - **Assignee**: current user
2. On success, push the new issue's detail view onto the stack
3. Automatically trigger the `t` hotkey (edit title) so the user can
   immediately name the issue

### Config Addition

```yaml
jira:
  base_url: https://company.atlassian.net
  default_project: PROJ   # <-- new field
```

---

## Part 4 — New Types & API Methods

### New/Updated Types (internal/jira/types.go)

```go
// IssueFields — add these fields:
type IssueFields struct {
    // ... existing fields ...
    Subtasks   []Issue      `json:"subtasks"`
    IssueLinks []IssueLink  `json:"issuelinks"`
    Parent     *ParentIssue `json:"parent"`
    Labels     []string     `json:"labels"`
}

// ParentIssue is a minimal issue reference for the parent field.
type ParentIssue struct {
    ID     string      `json:"id"`
    Key    string      `json:"key"`
    Fields *IssueFields `json:"fields,omitempty"`
}

// IssueLink represents a link between two issues.
type IssueLink struct {
    ID          string     `json:"id"`
    Type        LinkType   `json:"type"`
    InwardIssue  *Issue    `json:"inwardIssue,omitempty"`
    OutwardIssue *Issue    `json:"outwardIssue,omitempty"`
}

// LinkType describes the type of issue link.
type LinkType struct {
    ID      string `json:"id"`
    Name    string `json:"name"`
    Inward  string `json:"inward"`
    Outward string `json:"outward"`
}

// Transition represents an available workflow transition.
type Transition struct {
    ID   string  `json:"id"`
    Name string  `json:"name"`
    To   *Status `json:"to"`
}

// TransitionsResponse wraps the list returned by the API.
type TransitionsResponse struct {
    Transitions []Transition `json:"transitions"`
}

// CreateIssueRequest is the body for POST /rest/api/3/issue.
type CreateIssueRequest struct {
    Fields map[string]interface{} `json:"fields"`
}

// CreateIssueResponse is the response from POST /rest/api/3/issue.
type CreateIssueResponse struct {
    ID   string `json:"id"`
    Key  string `json:"key"`
    Self string `json:"self"`
}
```

### New API Methods (internal/jira/client.go)

| Method            | HTTP                                          | Notes                               |
|-------------------|-----------------------------------------------|-------------------------------------|
| `GetIssue`        | `GET /rest/api/3/issue/{key}`                 | Optional — only if search data is insufficient |
| `UpdateIssue`     | `PUT /rest/api/3/issue/{key}`                 | For title, description, priority    |
| `CreateIssue`     | `POST /rest/api/3/issue`                      | For quick-create                    |
| `DeleteIssue`     | `DELETE /rest/api/3/issue/{key}?deleteSubtasks=true` | With subtask cascade       |
| `GetTransitions`  | `GET /rest/api/3/issue/{key}/transitions`     | For status change (s) and done (d)  |
| `TransitionIssue` | `POST /rest/api/3/issue/{key}/transitions`    | Execute a status transition         |
| `AssignIssue`     | `PUT /rest/api/3/issue/{key}/assignee`        | Dedicated assign endpoint           |
| `SearchUsers`     | `GET /rest/api/3/users/search?maxResults=1000` | Bulk fetch for local cache          |

---

## Part 5 — Implementation Order

This is a large spec. Implement in these phases:

### Phase A: Types + API Methods
1. Add new types to `types.go`
2. Add all new client methods with tests (httptest)
3. Add `default_project` to config

### Phase B: Detail View (read-only)
1. Build the full detail view layout in a new `detail.go` file
2. Add viewport scrolling
3. Fetch full issue on Enter (or use search data)
4. ADF plain-text extraction
5. Display subtasks, links, parent, labels
6. Tests for rendering

### Phase C: Overlay System + Hotkeys
1. Build the overlay interface and key routing
2. Implement selection list overlay
3. Implement text input overlay
4. Implement confirmation overlay
5. Wire up hotkeys: `i` (simplest) → `d` → `s` → `p` → `t` → `e` → `a` → `del`
6. Optimistic local updates after edits
7. Tests

### Phase D: Quick Create
1. Wire `c` hotkey
2. CreateIssue API call with defaults
3. Open detail view → auto-trigger title edit
4. Tests

---

## Acceptance Criteria

- [ ] Enter on list view opens a full detail view with all fields displayed
- [ ] Detail view scrolls with j/k when content overflows
- [ ] Subtasks shown with status indicator (✓/·) and key + summary
- [ ] Linked issues shown grouped by link type with direction
- [ ] Parent shown with key + summary
- [ ] Labels shown as comma-separated list
- [ ] `s` opens transition picker, Enter executes transition
- [ ] `p` opens priority picker, Enter updates priority
- [ ] `d` marks issue as done (finds done-category transition)
- [ ] `t` opens title editor, Enter saves
- [ ] `e` opens description editor, save persists
- [ ] `i` assigns issue to current user immediately
- [ ] `a` opens user search/picker, Enter assigns
- [ ] `del` shows confirmation, `y` deletes issue
- [ ] `c` creates issue with defaults and opens it for editing
- [ ] Hotkeys work from detail view (list view deferred)
- [ ] All existing tests continue to pass
- [ ] New tests for detail view, overlays, and API methods

## Resolved Questions

- [x] Description editor saves with **Ctrl+Enter**, aborts with Esc.
- [x] Assignee picker uses a **local file cache** (`users.json`), loaded at
      startup. Filters locally — no live API search. Delete file to refresh.
      Exclude deactivated users.
- [x] Quick-create always creates a **Task**.
- [x] Hotkeys implemented in **detail view first**; list view wiring deferred
      to a follow-up.

## References

- [Hotkeys spec](./../context/Hotkeys.md)
- [Jira REST API v3 — Issues](https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issues/)
- [Jira REST API v3 — Transitions](https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issues/#api-rest-api-3-issue-issueidorkey-transitions-get)
- [Jira REST API v3 — User Search](https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-user-search/)
