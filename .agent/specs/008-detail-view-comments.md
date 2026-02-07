# Spec: Detail View Comments

> Spec number: 008
> Status: implemented
> Author: jbeckham
> Date: 2026-02-06

## Summary

Display issue comments on the detail view, loaded optimistically in parallel
with the full issue fetch. Comments appear in a dedicated section after the
existing detail content, showing author, timestamp, and body text extracted
from Jira's ADF format.

## Goals

- [x] Show comments on the issue detail view.
- [x] Load comments asynchronously, in parallel with the issue detail fetch.
- [x] Display a "Loading…" placeholder while comments are in-flight.
- [x] Render comment bodies from ADF to plain text using the existing
      `extractADFText()` utility.

## Non-Goals

- Adding or editing comments from the TUI (future spec).
- Pagination beyond the first 50 comments.
- Rich ADF rendering (bold, links, etc.) — plain text extraction is sufficient.
- Comment reactions or visibility restrictions.

## User Stories

- As a user, I want to see recent comments on an issue so I can understand
  the discussion without switching to the browser.
- As a user, I want comments to load without blocking the detail view so
  navigation stays instant.

## Requirements

### Functional

#### 1. API — `GetComments`

1. New client method: `GetComments(ctx, issueKeyOrID) ([]Comment, error)`.
2. Calls `GET /rest/api/3/issue/{key}/comment?orderBy=-created&maxResults=50`.
3. Returns newest comments first.

#### 2. Types

1. `Comment` struct: `ID string`, `Author *User`, `Body interface{}` (raw ADF),
   `Created string`, `Updated string`.
2. `CommentsResponse` struct: `Comments []Comment`, `StartAt int`,
   `MaxResults int`, `Total int`.

#### 3. Message flow

1. `commentsLoadedMsg{issueKey, comments, err}` carries the result.
2. `cmdFetchComments(issueKey)` command calls `GetComments` and returns the msg.
3. On `enter` key, dispatch `cmdFetchIssue` and `cmdFetchComments` in a
   `tea.Batch`. Both contribute to the inflight counter for spinner management.
4. On `issueCreatedMsg`, also dispatch `cmdFetchComments` alongside
   `cmdFetchIssue`.

#### 3b. Add comment flow

1. `c` key on detail view opens a textarea overlay (`overlayActionAddComment`).
2. On submit, optimistically prepend a placeholder comment (with `"just now"`
   timestamp) to the detail view and rebuild the viewport.
3. `cmdAddComment(issueKey, text)` calls `AddComment` API method.
4. `commentAddedMsg{issueKey, comment, err}` carries the result.
5. On success, replace the placeholder with the real comment from the API.
6. On failure, remove the placeholder and show an error flash.

#### 4. Detail view updates

1. `issueDetailView` gains `comments []jira.Comment` and
   `commentsLoading bool` fields.
2. `commentsLoading` is `true` on construction; set to `false` when
   `commentsLoadedMsg` arrives.
3. `commentsLoadedMsg` handler finds the active detail view on the stack,
   sets comments, and calls `buildViewport()` to re-render.

#### 5. Rendering

1. Comments section appears after the Parent section, before the action hints.
2. While loading: section header "Comments" with dim "Loading…".
3. When loaded with results: section header "Comments (N)" followed by each
   comment rendered as:
   - **Author name** (bold) and timestamp (dim) on one line.
   - Indented body text extracted via `extractADFText()`.
   - Blank line separator between comments.
4. When loaded with zero comments: section is omitted entirely.

### Non-Functional

- Performance: comments load in parallel with issue details; neither blocks
  the other or the initial detail view render.
- Error handling: if comment fetch fails, `commentsLoading` is set to false
  and the section simply doesn't appear (no error flash for supplementary data).

## Technical Notes

- Comment bodies are ADF (Atlassian Document Format), the same as issue
  descriptions. Reuses `extractADFText()` from `adf.go`.
- The inflight counter must be incremented for the comments fetch to keep
  the spinner alive. `startNetwork` handles one increment for the issue fetch;
  a manual `a.inflight++` covers the comments fetch.

## Files Changed

- `internal/jira/types.go` — `Comment`, `CommentsResponse` structs
- `internal/jira/client.go` — `GetComments()`, `AddComment()` methods
- `internal/tui/app.go` — `commentsLoadedMsg`, `commentAddedMsg`, handlers,
  `cmdFetchComments`, `cmdAddComment`, `overlayActionAddComment`,
  dispatch from `enter` and `issueCreatedMsg`
- `internal/tui/detail.go` — `comments`/`commentsLoading` fields, rendering,
  updated hint line

## References

- [Jira REST API — Get comments](https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issue-comments/#api-rest-api-3-issue-issueidorkey-comment-get)
- [Jira REST API — Add comment](https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issue-comments/#api-rest-api-3-issue-issueidorkey-comment-post)
- [Jira REST API — Add comment](https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issue-comments/#api-rest-api-3-issue-issueidorkey-comment-post)
