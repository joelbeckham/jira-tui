# Project Conventions

> This file tells the agent how to write code for this project.
> Update this as the project evolves.

## Tech Stack

- **Language:** Go (1.21+)
- **TUI Framework:** Bubbletea (github.com/charmbracelet/bubbletea)
- **Styling:** Lipgloss (github.com/charmbracelet/lipgloss)
- **Components:** Bubbles (github.com/charmbracelet/bubbles)
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
cmd/jira-tui/          # Entry point (main.go)
internal/
  tui/                 # Bubbletea models, views, components
    views/             # Full-screen views (board, issues, detail, search)
    components/        # Reusable UI widgets
  jira/                # Jira REST API client
  config/              # Configuration loading
docs/                  # Human-facing documentation
.agent/                # Agent workspace (specs, plans, checklists)
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
- Check `.agent/plans/` for active plans before making changes
- Don't modify specs — those are human-authored; ask for clarification instead
- Update checklists as you complete steps
- When uncertain, document the assumption in the plan/checklist notes
