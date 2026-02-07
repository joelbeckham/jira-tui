# Spec: Cross-Compilation

> Spec number: 007
> Status: implemented
> Author: jbeckham
> Date: 2026-02-06

## Summary

Support cross-compiling jira-tui for multiple OS/architecture targets so the
binary can be distributed to teammates on different platforms.

## Goals

- [x] Build for Linux amd64 (primary dev platform)
- [x] Build for macOS ARM (Apple Silicon / M-series Macs)
- [x] Build for Windows amd64 (Windows 11)

## Non-Goals

- Automated release pipelines (GitHub Actions, GoReleaser) — future work.
- Code signing or MSI/DMG packaging.
- 32-bit or other exotic architectures.

## Requirements

### Functional

1. `make build-all` produces three binaries under `bin/`:
   - `jira-tui-linux-amd64`
   - `jira-tui-darwin-arm64`
   - `jira-tui-windows-amd64.exe`
2. `make build` continues to build for the host platform only.
3. All binaries are statically linked (no CGo dependencies).

### Non-Functional

- Binaries are stripped (`-ldflags "-s -w"`) to reduce size.

## Technical Notes

Go's built-in cross-compilation (`GOOS`/`GOARCH` env vars) handles everything.
No CGo is used — `atotto/clipboard` uses pure-Go system calls on all three
platforms:
- Linux: reads `$DISPLAY` / `xclip` / `xsel` / `wl-copy`
- macOS: `pbcopy` / `pbpaste`
- Windows: `GetClipboardData` / `SetClipboardData` via syscall

All other dependencies (Bubbletea, Lipgloss, Bubbles, yaml.v3) are pure Go.

## Open Questions

- None.
