# Spec: Quick-Create Issue

> Spec number: 005
> Status: implemented
> Author: agent
> Date: 2026-02-06

## Summary

Adds a `c` hotkey from the list view that creates a new Jira issue through a
multi-step overlay flow: enter a summary, pick an issue type, and submit. The
issue is created in the configured default project, and the active tab refreshes
to show it.

## Goals

- [x] Provide a fast, keyboard-driven way to create issues without leaving the TUI
- [x] Use the existing overlay system for a multi-step input flow
- [x] Dynamically fetch issue types from the Jira project (not hardcoded)
- [x] Refresh the active tab after creation so the new issue is visible

## Non-Goals

- Full issue creation form (description, priority, assignee, etc.) — keep it minimal
- Creating subtasks or epics (subtask types are filtered out)
- Creating issues in arbitrary projects (uses configured `default_project`)

## User Stories

- As a user, I want to press `c` to quickly create an issue so I don't have to
  switch to a browser.
- As a user, I want the app to ask me for the minimum necessary fields (summary
  and type) so creating is fast.
- As a user, I want the new issue to appear in my tab list so I can see it
  immediately.

## Requirements

### Functional

1. Pressing `c` from the list view opens a text input overlay titled
   "New Issue Summary".
2. Submitting the summary opens a selection overlay titled "Issue Type" showing
   issue types fetched from the Jira API for the configured `default_project`.
3. Subtask types are excluded from the issue type list.
4. Selecting a type creates the issue via `POST /rest/api/3/issue` with the
   project key, summary, and issue type name.
5. On success, the active tab refreshes and a flash message shows "Created PROJ-123".
6. On failure, a flash error is displayed.
7. An empty summary is rejected with a flash message "Summary cannot be empty".
8. If `default_project` is not configured, pressing `c` shows a helpful error:
   "Set default_project in config to create issues".
9. If not connected to Jira, pressing `c` shows "Not connected to Jira".
10. Pressing Esc at any step cancels the flow.

### Non-Functional

- Issue type fetch is async (non-blocking).
- Status bar in list view shows `c: create` hint.

## Technical Notes

### Configuration

A new `default_project` field under `jira:` in `config.yaml`:

```yaml
jira:
  base_url: https://yourcompany.atlassian.net
  default_project: PROJ
```

### API Endpoints

- **Issue types:** `GET /rest/api/3/project/{projectKey}/statuses` — returns
  issue types with their statuses; we extract `{id, name, subtask}` and filter
  out subtasks.
- **Create issue:** `POST /rest/api/3/issue` — existing `CreateIssue` client
  method.

### Multi-Step Overlay Flow

The create flow uses two overlay actions in sequence:

1. `overlayActionCreateSummary` — text input overlay for the summary
2. `overlayActionCreateType` — selection overlay for the issue type

The summary is temporarily stored in `App.createSummary` between steps.

### New Types

- `jira.IssueType` — `{ID, Name, Subtask, Description}`
- `jira.Client.GetProjectIssueTypes(ctx, projectKey)` — fetches and filters types
- `issueTypesLoadedMsg` — delivers fetched types to the TUI
- `issueCreatedMsg` — delivers create result

### Files Changed

- `internal/jira/client.go` — `IssueType` struct, `GetProjectIssueTypes` method
- `internal/tui/app.go` — `c` hotkey, `createSummary` field, `defaultProject`
  field, overlay actions, message handlers, command functions, status bar hint
- `internal/config/config.go` — `DefaultProject` field (already existed)
- `cmd/jira-tui/main.go` — passes `cfg.Jira.DefaultProject` to `NewApp`

## Open Questions

- None (implemented and tested).

## References

- [Jira REST API v3 — Create issue](https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issues/#api-rest-api-3-issue-post)
- [Jira REST API v3 — Get project statuses](https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-projects/#api-rest-api-3-project-projectidorkey-statuses-get)
