# Product Requirements

This document is the product source of truth for PolarSwarm. It defines users, goals, MVP scope, non-goals, and acceptance criteria. Architecture and module documents must derive from this file; they must not redefine product goals.

## Problem

Software teams want to coordinate AI-assisted development through visible, reviewable project artifacts instead of hidden local chat sessions. Existing agent workflows often lack durable audit trails, clear human checkpoints, safe GitHub write behavior, and reproducible task state.

PolarSwarm solves this by using GitHub as the collaboration and audit surface while keeping agent execution local.

## Users

| User | Need |
|---|---|
| Solo developer | Turn one GitHub Issue into a bounded local agent task with visible results and safe dry-run behavior. |
| Maintainer | Review agent work through normal GitHub Issues, Comments, Labels, Pull Requests, and checks. |
| Reviewer | Understand what an agent did, what was verified, what failed, and whether rework is needed. |
| Automation operator | Bootstrap repository labels, templates, workflows, and local configuration without assuming all GitHub features are available. |

## Product Goals

- Coordinate development work from GitHub Issues.
- Keep GitHub as the durable collaboration, review, and audit surface.
- Execute agents locally in controlled runtime boundaries.
- Represent workflow state with structured Comments plus human-readable Labels.
- Support dry-run-first behavior for early milestones and dangerous writes.
- Preserve human approval boundaries for high-risk actions.
- Degrade when optional GitHub capabilities are unavailable.

## MVP Scope

The MVP proves one minimal local IssueOps loop:

```text
Issue -> Orchestrator -> Mock Agent -> Agent Result Comment Plan -> Label Plan
```

MVP includes:

- Local configuration loading.
- Read-only or fake GitHub Issue input.
- Label and Comment parsing for one eligible Issue.
- Minimal workflow transition logic.
- Mock agent execution.
- Rendering a `polarswarm-agent-result` Comment plan.
- Rendering a `status:*` Label update plan.
- Local idempotency state where persistence exists.
- Tests for the M1 path described in `testing-strategy.md`.

## Non-Goals

MVP does not include:

- Real GitHub writes by default.
- Remote agent execution in GitHub Actions.
- Multi-agent concurrency.
- Worktree lifecycle automation.
- Pull Request creation or merge automation.
- Full TUI.
- Security or compliance agent enforcement.
- GitHub Projects dependency.
- Multi-repository orchestration.
- Automatic dogfooding of PolarSwarm by this repository.

## Acceptance Criteria

MVP is acceptable when:

- A local command can process one fixture or fake-client Issue through the minimal loop.
- Default behavior is dry-run and produces inspectable write plans.
- Repeated execution is idempotent by message or operation identity.
- The result Comment format matches `agent-protocol.md`.
- The Label projection follows `issueops-model.md`.
- Tests cover the M1 required coverage in `testing-strategy.md`.
- Implementation tasks cite versioned docs rather than relying on local-only `PolarSwarm.md`.

## Product Change Rule

Changes to users, product goals, MVP scope, non-goals, or product acceptance must update this PRD first. Architecture, module docs, and local task specs may then be adjusted to implement or verify the product change.
