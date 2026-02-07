# Project Conventions

> This file tells the agent how to write code for this project.
> Update this as the project evolves.

## Tech Stack

- **Language:** Go 1.24+
- **TUI Framework:** Bubbletea v1.3 (github.com/charmbracelet/bubbletea)
- **Styling:** Lipgloss v1.1 (github.com/charmbracelet/lipgloss)
- **Components:** Bubbles v0.21 (github.com/charmbracelet/bubbles) — table, textinput, textarea, viewport
- **Config:** YAML via gopkg.in/yaml.v3
- **Build tool:** `go build` / Makefile
- **Test framework:** Go built-in (`go test`)
- **Linter/Formatter:** `gofmt` / `golangci-lint`

## Code Style

- Follow standard Go conventions (Effective Go, Go Code Review Comments)
- Use `gofmt` for formatting (non-negotiable)
- Run `golangci-lint` before committing
- Keep functions short and focused
- Use table-driven tests
- Return errors, don't panic

## Project Structure

```
cmd/jira-tui/          # Entry point (main.go) — init subcommand, auto-init
internal/
  tui/                 # Bubbletea models, views, key handling, styles
    app.go             # Root model (App), Update, View, key routing, overlay dispatch
    tab.go             # Tab model, issue-to-row rendering, fieldValue
    detail.go          # Issue detail view (scrollable viewport)
    overlay.go         # Overlay system (selection, textinput, textarea, confirm)
    adf.go             # ADF (Atlassian Document Format) extraction/creation
    filter.go          # Client-side quick filter (issueFilter)
    priority.go        # Priority icon/label mappings
    columns.go         # Auto-proportional column width builder
    styles.go          # All lipgloss styles
  jira/                # Jira REST API client (Cloud v3, Basic Auth)
  config/              # Config + secrets loading (YAML), init, user cache
.jira-tui/             # Runtime config dir (next to binary, gitignored)
  config.yaml          # User's Jira config
  secrets.yaml         # Credentials (never committed)
  users.json           # User cache
.agent/                # Agent workspace (specs, context, decisions)
```

## Naming Conventions

- Files: `snake_case.go` (Go convention: lowercase, underscores)
- Functions/Methods: `PascalCase` (exported), `camelCase` (unexported)
- Types/Interfaces: `PascalCase`
- Constants: `PascalCase` (exported), `camelCase` (unexported)
- Packages: short, lowercase, single word when possible

## Git Conventions

- Branch naming: `feature/short-description`, `fix/short-description`
- Commit messages: imperative mood, concise (e.g., "Add Jira auth flow")
- Keep commits atomic — one logical change per commit

## Testing

- Write tests alongside implementation (same package, `_test.go` suffix)
- Use table-driven tests for functions with multiple cases
- Jira client tests: use `httptest.NewServer` to mock API responses
- TUI model tests: call `Update()` directly — models are pure functions
- Integration tests: use `//go:build integration` tag, run with `go test -tags=integration`
- Run all tests: `make test`

## Error Handling

- Return `error` as the last return value — standard Go convention
- Wrap errors with context: `fmt.Errorf("doing X: %w", err)`
- Never ignore errors silently
- User-facing errors should be displayed in the TUI status bar

## Dependencies

- Minimize external dependencies
- Document reasons for adding new deps in the relevant ADR or plan

## Agent-Specific Rules

- Always read this file before starting work
- Consult specs in `.agent/specs/` for feature requirements
- Don't modify specs — those are human-authored; ask for clarification instead
- Record significant architectural choices in `.agent/decisions/`
- When uncertain, document the assumption and proceed with the safer choice
