# Architecture

PolarSwarm separates coordination from execution.

The architecture is pull-based. The local runtime repeatedly reads GitHub state, validates it against policy and local leases, performs bounded local work, and writes durable results back to GitHub.

## Architecture Boundary

This document defines system structure, runtime boundaries, module responsibilities, data flow, and technical constraints. It does not define users, product goals, MVP scope, or product acceptance criteria; those belong in [Product Requirements](product-requirements.md).

Architecture may choose mechanisms that satisfy the PRD, such as pull-based reconciliation, local execution, module boundaries, idempotency, and capability-aware degradation. If a proposed architecture change alters product scope or acceptance, update the PRD first.

## Traceability

PolarSwarm documentation follows one direction of authority:

```text
PRD -> Architecture -> Module Docs -> specs tasks -> implementation/tests
```

Rules:

- PRD defines product intent and acceptance.
- Architecture defines how the system is organized to satisfy the PRD.
- Module docs define focused responsibilities and interfaces inside the architecture.
- Local `specs/M{N}/T{NNN}-{SLUG}.md` files decompose module work into executable tasks.
- Module docs and task specs must not introduce new product goals or widen MVP scope.
- If implementation discovers a product-level gap, create or update a docs task that changes PRD, then architecture, then module docs.

## Control Plane

GitHub is the remote control plane:

- Issues represent user-visible work.
- Comments carry append-only commands, events, decisions, and audit records.
- Labels project current state for humans.
- Pull Requests carry code review, CI, and merge gates.
- Required checks and branch protection remain the hard repository gate when available.
- Repository configuration and workflow policy are versioned in Git.
- Optional GitHub features are used only when the Capability Cache reports them as available.

## Local Runtime

The local PolarSwarm process owns execution:

- Poll GitHub for active Issues and new Comments.
- Reconcile remote state before executing work.
- Dispatch work to agent backends.
- Store cursors, idempotency keys, leases, and metrics locally.
- Write results back to GitHub only after local validation.

GitHub Actions are limited to lightweight guards and notifications. They do not access local worktrees and do not run agents.

## Runtime Topology

| Layer | Location | Responsibility |
|---|---|---|
| GitHub | GitHub servers | Durable Issues, Comments, Labels, Pull Requests, milestones, branch gates, and optional merge queue. |
| GitHub Actions | GitHub-hosted runners | Lightweight triage, Label guard, and optional notification workflows. |
| PolarSwarm local process | Developer machine | Orchestrator, agent dispatch, local CLI invocation, worktree management, local store writes, and reconciliation. |
| Agent backends | Developer machine or configured local CLI/API boundary | Role-specific execution through tools such as Codex, Claude Code, OpenCode, or mock backends. |

GitHub is not an agent execution environment. Actions must not depend on local worktrees, local SQLite state, or local agent credentials.

## Core Modules

| Module | Responsibility |
|---|---|
| `config` | Load `.polarswarm/core.toml` and local policy |
| `github` | GitHub API access behind testable interfaces |
| `workflow` | Orchestration state transitions and label projections |
| `agent` | Agent runner abstractions and backend adapters |
| `store` | Local cursors, leases, idempotency records, and metrics |
| `doctor` | Local and repository readiness checks |
| `tui` | Read-only and later interactive human runtime views |
| `app` | CLI entry points and module composition |

The initial codebase should keep these boundaries testable. GitHub API access should be behind interfaces so workflow tests can run without live network access.

Short module contracts live under [modules/](modules/README.md). The table above is the architectural source for module existence and ownership; module docs refine responsibilities without redefining product scope.

## Reconciliation Rule

Every execution cycle follows:

```text
Fetch -> Validate -> Execute -> Write Back
```

The local runtime must fetch remote Issue, PR, Comment, Label, and close state before executing an agent step. If remote state has changed in a way that invalidates the local lease, the local action must stop and write no workflow-advancing result.

Remote state has priority over local intent. If an Issue was closed, a later human decision was added, a guarded Label was removed, or a PR branch moved unexpectedly, the local lease must be revoked or paused before any workflow-advancing write.

## GitHub Actions Boundary

Only three workflow families are part of the product baseline:

| Workflow | Trigger | Responsibility |
|---|---|---|
| `polarswarm-triage.yml` | Issue and Issue Comment events | Apply whitelist, blacklist, `/triage`, and `/reject` gates. |
| `polarswarm-label-guard.yml` | Issue Label events | Detect and undo unauthorized `status:*`, `agent:*`, and `decision:*` Label changes. |
| `polarswarm-notify.yml` | Issue Comment events | Optional lightweight notification or wake-up hint. |

Actions do not route agent messages. Message routing is handled by the local process when it polls and parses structured Comments.

## Token Classes

PolarSwarm separates high-permission setup from lower-permission runtime.

| Token | Lifetime | Responsibility |
|---|---|---|
| Init token | Short lived | Create or update Labels, Issue Types, issue templates, workflow files, optional Projects V2 resources, and repository rules when permitted. |
| Runtime token | Long lived | Poll Issues and Comments, write audit Comments, update Labels, manage PRs, and read capability state using least privilege. |

Higher-risk automation such as Checks API writes should prefer a GitHub App when supported. Fine-grained PAT use must remain capability-probed and permission-minimized.

## Initialization Scope

`polarswarm init` prepares a repository for PolarSwarm. It must be safe to rerun.

Baseline responsibilities:

- Verify GitHub connectivity and token permissions.
- Create or update the standard Label set.
- Create Issue Types when organization permissions allow it; otherwise use `type:*` Labels.
- Write YAML Issue Templates for the product issue types.
- Write the three lightweight GitHub Actions workflows.
- Write `.polarswarm/` configuration templates, including access policy.
- Append the configured worktree base directory to `.gitignore`.
- Optionally create Projects V2 views and fields when capability and permission checks pass.

Initialization output must report created, updated, skipped, and degraded items. A partial lack of optional permissions should not fail the whole initialization if a documented degraded mode exists.

## Doctor Scope

`polarswarm doctor` reports whether the local environment and target repository are ready. It should support focused categories so users can run cheap checks during development.

Required product categories:

- `config`: parse and cross-check `.polarswarm` configuration.
- `github`: validate API reachability, token identity, and repository access.
- `github:permissions`: inspect permission-sensitive features.
- `capabilities`: display Capability Cache state and optionally run safe probes.
- `labels`: compare repository Labels with the standard Label set.
- `workflows`: check for the expected workflow files.
- `store`: validate local data directory and SQLite schema.
- `worktrees`: report orphaned or blocked worktrees.
- `llm`: optional backend connectivity test that may consume tokens and must be explicitly requested.

## Local Execution Constraints

- Each task that writes code should use an isolated `task/<issue-number>` branch and local worktree.
- Local state stores cursors, leases, idempotency records, metrics, and capability cache entries. It is not the final source of workflow truth.
- Worktree cleanup is asynchronous and conservative. Cleanup must not remove active leases or unmerged branches.
- Every write to GitHub must have an idempotency key or equivalent operation record.
- GitHub API polling should use cursors, conditional requests, and backoff to control rate usage.
