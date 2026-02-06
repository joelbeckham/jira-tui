# Spec: Project Foundation

> Spec number: 001
> Status: approved
> Author: jbeckham
> Date: 2026-02-06

## Summary

Bootstrap jira-tui — a fast, minimalist, keyboard-driven Terminal User Interface
for managing day-to-day Jira work. Built in Go with Bubbletea.

## Goals

- [x] Establish Go project structure with modules
- [x] Set up Bubbletea TUI skeleton (app shell with quit keybinding)
- [x] Create Jira API client package (interface + types, no implementation yet)
- [x] Set up config loading from YAML
- [x] Establish test patterns for each package
- [x] Makefile with build/test/lint targets

## Non-Goals

- No feature implementation yet (boards, issues, search come in later specs)
- No Jira authentication flow yet (that's spec 002)
- No CI/CD setup

## Requirements

### Functional

1. `go build ./...` compiles cleanly
2. `go test ./...` passes with at least one test per package
3. Running the binary shows a minimal TUI screen that quits on `q` or `ctrl+c`

### Non-Functional

- Startup time under 100ms
- Binary size under 20MB
- Zero runtime dependencies beyond the terminal

## Technical Notes

- Use Go 1.21+ for modern features
- `internal/` directory enforces package encapsulation
- Bubbletea Elm Architecture: Model → Update → View
- Config location: `~/.config/jira-tui/config.yaml`
