# Overview

This is an orientation summary derived from [Product Requirements](product-requirements.md) and [Architecture](architecture.md). If this document conflicts with the PRD on users, goals, scope, or acceptance, the PRD wins.

PolarSwarm is an IssueOps-first, GitOps-inspired system for coordinating software development with multiple local agents.

GitHub is the collaboration and audit surface. The local PolarSwarm process is the execution engine. Agents work in local worktrees, while Issues, Pull Requests, Comments, Labels, and repository protection rules expose progress and decisions to humans.

## Goals

- Map user-visible development work to GitHub Issues.
- Use GitHub Pull Requests, Comments, Labels, and required checks as the collaboration and audit surface.
- Keep agent execution local, with isolated worktrees and local state.
- Support human-in-the-loop and autonomous modes.
- Allow different agent roles to bind to different execution backends without changing workflow semantics.
- Make every workflow-advancing action auditable through append-only GitHub Comments and linked Pull Requests.
- Adapt to GitHub feature availability at runtime instead of assuming one fixed account plan or permission profile.

## Product Principles

- **IssueOps-first**: user-visible work is represented as Issues. Internal retries, local diagnostics, and backend fallback attempts are not promoted to Issues by default.
- **GitOps-inspired**: workflow policy, access policy, and repository configuration live in versioned files and become effective after normal review.
- **Remote audit, local execution**: GitHub stores the durable event trail; the local runtime performs agent execution, worktree management, and reconciliation.
- **Labels are projections**: `status:*`, `agent:*`, and `decision:*` Labels are human-readable projections, not the only source of truth.
- **Native gates remain authoritative**: branch protection, required checks, CODEOWNERS, environment reviewers, and merge queue remain authoritative when available.
- **Capability-aware degradation**: optional GitHub features must degrade to documented alternatives when unavailable or under-permissioned.

## Non-Goals for MVP

- No remote agent execution in GitHub Actions.
- No automatic merge queue dependency.
- No required GitHub Projects integration.
- No full release automation.
- No security or compliance gate as a hard MVP dependency.
- No multi-repository orchestration.
- No GitLab or local-only backend implementation. Interfaces may leave room for them, but GitHub is the only product backend for the initial line of work.
- No automatic self-dogfooding requirement. The PolarSwarm repository may use lightweight conventions before PolarSwarm itself can enforce them.

## MVP Boundary

The first usable milestone is a minimal local loop:

```text
GitHub Issue
-> local orchestrator
-> mock agent
-> polarswarm-agent-result comment
-> status label update
```

MVP is intentionally single-repository, single-issue, single-agent, and single-threaded. All GitHub writes must support dry-run or explicit confirmation.

The initial implementation should prove the reconciliation path before expanding breadth:

1. Read local configuration and verify the target repository.
2. Connect to GitHub with a runtime token.
3. Read one eligible Issue and its current Labels and Comments.
4. Run a mock or minimal agent backend.
5. Write a `polarswarm-agent-result` Comment.
6. Update the visible `status:*` projection with an idempotent operation.
7. Persist local cursor and idempotency state.

## Operating Modes

PolarSwarm supports two operating modes. Both modes use the same Issue, Comment, Label, and PR protocol.

| Mode | Behavior |
|---|---|
| Human-in-the-loop | The workflow pauses at configured checkpoints and waits for a human decision. The checkpoint request is written to GitHub before the local TUI prompts the user, so the remote Issue remains recoverable after process restart. |
| Autonomous | The orchestrator may make checkpoint decisions itself when policy permits. Each autonomous decision must be recorded as a structured Comment with decision metadata and confidence. |

Configured checkpoints include requirements triage, architecture review, PR merge approval, test acceptance, security findings, compliance results, and release approval. MVP can implement only the minimal checkpoint surface needed for safe local development.

## Bootstrap Position

This repository can use Vibe Coding plus spec-guided bootstrap while the product is being built. That means:

- Formal product behavior is specified in `docs/`.
- Local planning tasks may live outside Git until they are promoted into implementation or documentation.
- The project may manually follow lightweight conventions such as area-prefixed commit titles and PR templates.
- The project must not require the full PolarSwarm workflow to develop PolarSwarm before the minimum local loop exists.

## Documentation Map

- `product-requirements.md`: product users, goals, MVP scope, non-goals, and acceptance criteria.
- `architecture.md`: runtime topology, module boundaries, reconciliation, initialization, and local execution constraints.
- `modules/`: short module-level contracts derived from the PRD and architecture.
- `issueops-model.md`: Issue lifecycle, Labels, Comments, message envelopes, PR requirements, and safety rules.
- `github-capabilities.md`: official-source constraint, GitHub feature assumptions, Capability Cache, and degradation behavior.
