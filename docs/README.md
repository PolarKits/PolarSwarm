# PolarSwarm Documentation

PolarSwarm is a local IssueOps orchestrator for multi-agent software development. It uses GitHub Issues, Pull Requests, Labels, and Comments as the collaboration and audit surface, while local processes perform the actual agent execution.

## Reading Order

1. [Product Requirements](product-requirements.md): product users, goals, scope, MVP, non-goals, and acceptance.
2. [Architecture](architecture.md): system structure, boundaries, modules, data flow, and technical constraints.
3. Cross-cutting specs such as [Configuration](configuration.md), [Observability](observability.md), [IssueOps Model](issueops-model.md), [Agent Protocol](agent-protocol.md), [Security Model](security-model.md), and [TUI Design](tui-design.md).
4. [Module Docs](modules/README.md): short module-level contracts derived from the architecture.
5. Local task specs: `specs/M{N}/T{NNN}-{SLUG}.md`, used to execute small implementation tasks.

`overview.md` is a compact orientation document. It should not introduce product requirements that are absent from the PRD.

## Documents

- [Product Requirements](product-requirements.md): user problems, product goals, MVP scope, non-goals, and acceptance criteria.
- [Overview](overview.md): product scope, non-goals, and MVP boundary.
- [Architecture](architecture.md): local orchestrator, GitHub control plane, storage, and execution model.
- [Modules](modules/README.md): module index and per-module responsibility boundaries.
- [Configuration](configuration.md): `.polarswarm/*.toml` files, discovery, merge rules, and minimum schema.
- [Observability](observability.md): local SQLite/JSONL boundaries, event data, and metrics.
- [IssueOps Model](issueops-model.md): Issues, Labels, Comments, PRs, checkpoints, and state projection.
- [Agent Protocol](agent-protocol.md): agent roles, message envelopes, result comments, and review flow.
- [GitHub Capabilities](github-capabilities.md): capability probing, degradation, and platform constraints.
- [Security Model](security-model.md): actor gates, write safety, label protection, and human approval boundaries.
- [TUI Design](tui-design.md): local dashboard, checkpoint UX, monitoring views, and no-write UI boundary.
- [Development Process](development-process.md): current bootstrap process, dogfood boundary, commit and PR conventions.
- [Testing Strategy](testing-strategy.md): unit, fake-client, dry-run, and manual acceptance testing.
- [Roadmap](roadmap.md): staged implementation milestones.

## Source of Truth

These files are the versioned project documentation. Local task specs under `specs/` and the original planning draft `PolarSwarm.md` are intentionally ignored by git and used only as working material.

Traceability flows one way:

```text
PRD -> Architecture -> Module Docs -> specs tasks -> implementation/tests
```

Module documents may refine responsibilities and interfaces that already fit the architecture. They must not redefine users, product goals, MVP scope, or acceptance criteria. If module work exposes a product change, update the PRD first, then adjust architecture and task specs.
