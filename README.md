# jira-tui

> A terminal user interface for Jira. (Details TBD)

## Getting Started

(To be filled in once the tech stack and build setup are defined.)

## Project Structure

```
AGENTS.md              # Entry point for AI agents — read this first
.agent/                # Agent workspace
  specs/               # What to build (human-authored)
  plans/               # How to build it (agent-generated)
  checklists/          # Progress tracking
  context/             # Conventions, glossary, constraints
  decisions/           # Architecture Decision Records
docs/                  # Human documentation
src/                   # Source code
tests/                 # Tests
```

## Contributing

This project uses an **agent-first workflow**:

1. Write a spec in `.agent/specs/` describing the feature or change.
2. Have the agent generate a plan in `.agent/plans/`.
3. Review and approve the plan.
4. Let the agent implement, tracking progress in `.agent/checklists/`.

See [AGENTS.md](AGENTS.md) for full details.
