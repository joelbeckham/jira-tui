# Spec: Open Issue in Browser

> Spec number: 009
> Status: implemented
> Author: jbeckham
> Date: 2026-02-06

## Summary

Press `o` on any issue (in the list view or detail view) to open it in the
default system browser. Uses the existing `BrowseURL()` client method and
platform-specific browser-launch commands.

## Goals

- [x] Open the currently selected/viewed issue in the default browser with `o`.
- [x] Works from both the issue list and the detail view.
- [x] Cross-platform: Linux (`xdg-open`), macOS (`open`), Windows (`rundll32`).

## Non-Goals

- Configurable browser or custom URL patterns.
- Opening other Jira pages (boards, filters, etc.).

## User Stories

- As a user, I want to press `o` to quickly jump to the full Jira web view
  when I need features not available in the TUI (attachments, workflows, etc.).

## Requirements

### Functional

1. `o` key is handled in `handleEditHotkey`, which runs for both list and
   detail views.
2. Constructs the URL via the existing `client.BrowseURL(issueKey)` method
   (`{baseURL}/browse/{key}`).
3. Launches the URL with the platform-appropriate command:
   - Linux/FreeBSD: `xdg-open`
   - macOS: `open`
   - Windows: `rundll32 url.dll,FileProtocolHandler`
4. Shows a flash message on success ("Opened KEY in browser") or failure
   ("Could not open browser").

### Non-Functional

- The browser launch is non-blocking (uses `cmd.Start()`, not `cmd.Run()`).
- No new dependencies — uses `os/exec` and `runtime` from the standard library.

## Files Changed

- `internal/tui/app.go` — added `o` to `editHotkeys`, handler in
  `handleEditHotkey`, `openBrowser()` helper function, imports for
  `os/exec` and `runtime`
- `README.md` — features list and keyboard shortcuts table
- `AGENTS.md` — status update

## References

- [Go `os/exec` package](https://pkg.go.dev/os/exec)
