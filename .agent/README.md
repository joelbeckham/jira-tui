# .agent directory

This directory is the **agent workspace** — the primary interface between human intent
and AI execution.

## Structure

| Directory      | Purpose                                        | Who writes?         |
|----------------|------------------------------------------------|---------------------|
| `specs/`       | Feature & system specifications                | Human (agent reads) |
| `plans/`       | Implementation plans with steps                | Agent (human reviews) |
| `checklists/`  | Progress tracking for multi-step work          | Agent (updates live) |
| `context/`     | Conventions, constraints, tech stack           | Human (agent reads) |
| `decisions/`   | Architecture Decision Records                  | Agent or Human      |

## How It Works

1. **Human writes a spec** in `specs/` describing what they want.
2. **Agent reads the spec**, generates a **plan** in `plans/`.
3. Human reviews and approves (or iterates on) the plan.
4. **Agent executes the plan**, tracking progress in `checklists/`.
5. Significant choices get documented in `decisions/`.

## File Naming

- Specs: `specs/NNN-short-name.md` (e.g., `specs/001-jira-auth.md`)
- Plans: `plans/NNN-short-name.md` (matches spec number when applicable)
- Checklists: `checklists/NNN-short-name.md`
- Decisions: `decisions/ADR-NNN-short-name.md`
