# jira-tui

> A fast, minimalist, keyboard-driven terminal UI for Jira Cloud.

## Features

- **Filter tabs** — configure multiple saved Jira filters as tabs, each with custom columns
- **Vim-style navigation** — `j`/`k` to move, `enter` to open detail view, `esc` to go back
- **Quick filter** — press `/` to filter issues client-side by text
- **Inline editing** — change status (`s`), priority (`p`), assignee (`a`), title (`t`), description (`e`) via overlays
- **Quick actions** — assign to me (`i`), mark done (`d`), delete (`del`)
- **Quick create** — press `c` to create a new issue (summary → type → submit)
- **Add comment** — press `c` on the detail view to add a comment
- **Clipboard** — yank issue key (`y`) or copy URL (`u`)
- **Open in browser** — press `o` to open the current issue in your default browser
- **Detail view** — full scrollable issue detail with fields, subtasks, linked issues
- **Priority icons** — colored Unicode icons in the issue list

## Getting Started

### Prerequisites

- Go 1.24+
- A Jira Cloud instance with API access

### Configuration

Run the binary once — it will create a `.jira-tui/` directory next to the
executable with sample config files:

```bash
./jira-tui          # auto-creates .jira-tui/ on first run
# or explicitly:
./jira-tui init
```

Then edit the two files it creates:

1. `.jira-tui/config.yaml` — Jira URL, tabs, columns:

```yaml
jira:
  base_url: https://yourcompany.atlassian.net
  default_project: PROJ  # used by 'c' (create issue) hotkey

tabs:
  - label: "My Sprint"
    filter_id: "10042"
    columns: [key, summary, status, assignee, priority]
    sort: priority
```

2. `.jira-tui/secrets.yaml` — your credentials:

```yaml
jira:
  email: you@company.com
  api_token: your-api-token
```

Generate an API token at https://id.atlassian.com/manage-profile/security/api-tokens

### Build & Run

```bash
make build
make run
```

## Keyboard Shortcuts

### Navigation
| Key | Action |
|-----|--------|
| `j` / `k` | Move down / up |
| `home` / `end` | Jump to top / bottom |
| `enter` | Open issue detail |
| `esc` | Go back / clear filter |
| `1`-`9` | Switch tabs |
| `/` | Quick filter |
| `r` | Refresh tab |
| `q` | Quit |

### Editing (list & detail views)
| Key | Action |
|-----|--------|
| `s` | Change status |
| `p` | Change priority |
| `a` | Change assignee |
| `t` | Edit title |
| `e` | Edit description |
| `i` | Assign to me |
| `d` | Mark as done |
| `del` | Delete issue |

### Other
| Key | Action |
|-----|--------|
| `c` | Create new issue (list) / Add comment (detail) |
| `y` | Copy issue key |
| `u` | Copy issue URL |
| `o` | Open issue in browser |

## Project Structure

```
AGENTS.md              # Entry point for AI agents — read this first
.agent/                # Agent workspace
  specs/               # What to build (human-authored)
  context/             # Conventions, glossary, constraints
  decisions/           # Architecture Decision Records
src/                   # Source code
tests/                 # Tests
```

