# Agent Instructions — jira-tui

> This file is the entry point for any AI coding agent working on this project.
> Read this first, then follow the links below for deeper context.

## Project Overview

**jira-tui** — (description TBD, to be filled in once project specifics are defined)

## Directory Map

```
.agent/              # Agent workspace — specs, plans, checklists, context
  specs/             # Feature and system specifications (agent reads these)
  plans/             # Implementation plans (agent generates and follows these)
  checklists/        # Task checklists for tracking progress
  context/           # Background context, constraints, conventions
  decisions/         # Architecture Decision Records (ADRs)

docs/                # Human-readable documentation
src/                 # Application source code
tests/               # Test files
```

## Workflow for Agents

1. **Read context first** — check `.agent/context/conventions.md` for coding style,
   tech stack, and project constraints before writing any code.
2. **Check for existing plans** — look in `.agent/plans/` for active implementation
   plans before starting new work.
3. **Consult specs** — feature specs in `.agent/specs/` are the source of truth for
   what should be built.
4. **Generate a plan** — for non-trivial work, create a plan in `.agent/plans/`
   before implementing. Include clear steps and acceptance criteria.
5. **Track with checklists** — use `.agent/checklists/` to track multi-step work.
   Update checklist items as you go.
6. **Record decisions** — when making significant architectural choices, write an
   ADR in `.agent/decisions/`.

## Conventions

- See [.agent/context/conventions.md](.agent/context/conventions.md) for full details.
- Write tests alongside implementation.
- Keep commits atomic and well-described.

## Current Status

- **Phase:** Project bootstrapping
- **Active plans:** None yet
- **Blockers:** Awaiting project specification
