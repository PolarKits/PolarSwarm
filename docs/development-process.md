# Development Process

The current PolarSwarm repository is developed in:

```text
Vibe Coding + Spec-Guided Bootstrap
```

This repository may use PolarSwarm ideas before the product implements them, but only as manual development discipline. The current project must not pretend that unimplemented PolarSwarm automation is already a required workflow.

## Versioned vs Local Files

Versioned project documentation lives in `docs/`. Source code, tests, README, build metadata, and CI files are normal versioned project files.

Local-only planning files are intentionally excluded from git:

- `specs/`
- `tasks/`
- `PolarSwarm.md`
- `PolarSwarm-DocsGithub-Audit-Report.md`

`PolarSwarm.md` is the local planning draft during bootstrap. It is not the published documentation surface. Its content should be decomposed into complete docs under `docs/` before code tasks rely on it.

## Documentation Source Of Truth

For implementation tasks, the source of truth is:

1. The task file referenced with `@specs/M{N}/T{NNN}-{SLUG}.md`.
2. The versioned docs under `docs/`.
3. Source code and tests in the repository.

`PolarSwarm.md` may be used by document-generation or coverage-verifier tasks, but normal implementation tasks should cite the relevant `docs/` file once docs exist.

## Local Task Format

Tasks are stored locally as:

```text
specs/M{N}/T{NNN}-{SLUG}.md
```

Examples:

```text
specs/M1/T001-create-go-module.md
specs/M1/T012-render-agent-result-comment.md
specs/M2/T004-add-backend-runner-interface.md
```

Task files are executed serially by pointing a model at one file with an `@specs/...` reference. Each task must be small enough for one model to execute and for a stronger reasoning model to review independently.

Every task must include:

- Goal.
- Scope.
- Inputs.
- Outputs.
- Non-goals.
- Acceptance criteria.
- Verification commands.
- Manual checks, if any.
- Files allowed to edit.
- Result section for the executor to fill.

Tasks should avoid hidden assumptions. If a task depends on a previous task, it must name the previous task file or milestone artifact.

## Execution Discipline

The bootstrap process is intentionally serial:

1. Generate a milestone task set.
2. Execute tasks in filename order.
3. Run the task's verification.
4. Record the result in the task file or final response as specified by that task.
5. Use a stronger reasoning model to review completion evidence.
6. Fix only the task's scope before moving to the next task.

Executors must not edit unrelated task files, unrelated docs, or broad code areas unless the task explicitly permits it. If a task discovers that the docs are wrong, it should propose or create a narrow docs update task instead of silently expanding implementation scope.

## Commit Title Convention

Use area-prefixed titles:

```text
<area>: <imperative summary> (#<issue>)
```

Allowed areas:

| Area | Use |
|---|---|
| `github` | GitHub API, IssueOps, Labels, PRs, Checks, Projects, permissions. |
| `workflow` | Orchestrator state machine, task flow, checkpoints, rework. |
| `agent` | Agent protocol, role scheduling, backend adapters. |
| `tui` | Local TUI display, interaction, notifications. |
| `config` | `.polarswarm` config, capability cache, policies. |
| `security` | Permissions, scanning, secrets, risk handling. |
| `compliance` | Compliance policy, reports, audit requirements. |
| `release` | Release workflow, versions, release gates. |
| `docs` | Documentation, ADRs, templates, specifications. |
| `test` | Test framework, test cases, fixtures. |
| `ci` | GitHub Actions, required checks, status output. |
| `build` | Build, packaging, dependencies, toolchain. |

The title area describes the changed domain, not the executing agent.

Recommended footer fields for agent-generated commits:

```text
Issue: #<issue_number>
Agent: <role_id>
Workflow: <workflow_id>
Checkpoint: <checkpoint_id>
Rework: <iteration>/<max>
Verification: <command-or-status>
Generated-By: <agent_id>/<model_id>
```

Not every manual bootstrap commit needs every footer, but automated PolarSwarm commits eventually should provide them.

## Agent PR Standard

Agent-created or agent-updated PRs must include:

```markdown
## Summary
- ...

## Linked Issue
Closes #<issue_number>

## Agent Worklog
- orchestrator: ...
- developer: ...
- reviewer: ...

## Verification
- [ ] <command-or-check>

## Risk
- ...
```

The orchestrator must not mark an agent PR complete unless the related Issue contains at least one `polarswarm-agent-result`. Automatic merge requires a passing `polarswarm-review-result` and required checks or an accepted degraded verification path.

During this repository's bootstrap phase, these PR body rules are design targets. They are not mandatory for local-only `specs/` work unless a task explicitly requires GitHub PR creation.

## Dogfood Boundary

PolarSwarm can gradually use its own workflow, but only after the relevant capability exists and has a manual fallback.

Allowed during bootstrap:

- Small local tasks.
- Explicit acceptance criteria.
- Explicit verification.
- Area-prefixed commit titles.
- Documentation updates when design changes.
- Manual GitHub setup and connection tasks.
- Mock agent or dry-run workflows.

Not required during bootstrap:

- Automatic Issue routing.
- Automatic PR creation.
- Automatic review.
- Automatic merge.
- Full TUI.
- Full capability cache.
- Security or compliance agents.
- Merge queue integration.

High-risk self-development changes always require human review: orchestrator state, permissions, security gates, token handling, auto-merge, and destructive operations.

## GitHub Connection Tasks

The repository connection to GitHub should be implemented as small tasks, not as a hidden prerequisite:

- Initialize git if needed.
- Configure remote.
- Verify `gh auth status`.
- Create or verify the GitHub repository.
- Push the main branch.
- Create baseline labels only when explicitly requested.
- Keep `specs/` and `PolarSwarm.md` ignored.

Tasks that write to GitHub must state whether they are dry-run, read-only, or write-enabled.
