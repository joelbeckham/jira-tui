# .agent directory

This directory is the **agent workspace** — the primary interface between human intent
and AI execution.

## Structure

| Directory      | Purpose                                        | Who writes?         |
|----------------|------------------------------------------------|---------------------|
| `specs/`       | Feature & system specifications                | Human (agent reads) |
| `context/`     | Conventions, constraints, tech stack           | Human (agent reads) |
| `decisions/`   | Architecture Decision Records                  | Agent or Human      |

## How It Works

1. **Human writes a spec** (or collaborates with agent) in `specs/`.
2. **Agent reads the spec** and implements it, writing tests alongside code.
3. Significant architectural choices get documented in `decisions/`.
4. Specs are updated to "implemented" status when complete.

## File Naming

- Specs: `specs/NNN-short-name.md` (e.g., `specs/001-project-foundation.md`)
- Decisions: `decisions/ADR-NNN-short-name.md`
