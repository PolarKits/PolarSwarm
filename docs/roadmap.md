# Roadmap

PolarSwarm starts with a local, serial, dry-run-friendly bootstrap. The first usable product milestone is a minimal IssueOps loop, not the full multi-agent system.

Each milestone owns a local task set:

```text
specs/M{N}/T{NNN}-{SLUG}.md
```

Tasks are executed in filename order with `@specs/...` references. `specs/` is local-only and must remain outside git. Versioned product docs live in `docs/`.

## Milestone Policy

For every milestone:

- Generate small local task files.
- Include docs tasks before implementation when behavior is not already specified.
- Include tests or verification in each implementation task.
- Avoid real GitHub writes by default.
- Run a stronger-model review after individual tasks when possible.
- Run a coverage verifier at milestone end.
- Move incomplete or risky work to the next milestone instead of expanding the current milestone silently.

## M0 Local Bootstrap

Status: Done / documentation bootstrap in progress.

Purpose:

- Create local task conventions.
- Define the dogfood boundary.
- Split product documentation into `docs/`.
- Keep `PolarSwarm.md` and `specs/` out of git.
- Prepare GitHub connection tasks.

Expected artifacts:

- `docs/overview.md`
- `docs/architecture.md`
- `docs/issueops-model.md`
- `docs/github-capabilities.md`
- `docs/agent-protocol.md`
- `docs/security-model.md`
- `docs/development-process.md`
- `docs/testing-strategy.md`
- `docs/roadmap.md`
- Initial `specs/M0/` and `specs/M1/` task files, local-only.

M0 exit criteria:

- Versioned docs explain product scope, bootstrap boundaries, and task execution style.
- Local task format is unambiguous.
- Coverage verifier has checked that the high-level content in `PolarSwarm.md` is represented or intentionally deferred.

## M1 Minimal IssueOps Loop

Status: In Progress / next implementation target.

Purpose:

```text
Issue -> Orchestrator -> Mock Agent -> Comment Plan -> Label Plan
```

Scope:

- Project skeleton.
- Config loader.
- Read one Issue from fake or read-only GitHub client.
- Parse relevant labels and comments.
- Minimal state machine.
- Mock agent runner.
- Render `polarswarm-agent-result`.
- Produce dry-run writeback plans for Comment and Label changes.
- Store processed message IDs and operation IDs where persistence exists.

Non-goals:

- Real agent backend.
- Real GitHub writes by default.
- Worktree creation.
- PR creation.
- TUI.
- Merge handling.
- Security or compliance agents.

M1 acceptance:

- A local command can process one fixture Issue through the minimal loop.
- Default behavior is dry-run.
- Repeated execution is idempotent.
- Tests cover the M1 required coverage in `docs/testing-strategy.md`.

## M2 Agent Backend

Status: Done.

Purpose:

- Define `BackendRunner`.
- Keep mock backend as the default test backend.
- Add the first real CLI backend adapter.
- Enforce worktree `cwd`.
- Override model and output settings from PolarSwarm effective config.
- Capture stdout, stderr, exit code, duration, and normalized error type.
- Mask secrets before logging or audit writes.
- Record auditable agent results.

Non-goals:

- Multi-backend fallback queue, unless a task explicitly scopes it.
- Full worktree lifecycle automation.
- Autonomous merge.

M2 acceptance:

- Mock backend and first CLI backend share the same runner interface.
- Backend output can produce a safe `polarswarm-agent-result`.
- Opt-in live backend checks are separated from default tests.

## M3 Doctor And Readonly TUI

Status: Done.

Purpose:

- Add `doctor github`.
- Add `doctor labels`.
- Add `doctor capabilities`.
- Add explicit opt-in `doctor llm`.
- Build a readonly TUI dashboard.

Non-goals:

- TUI writes.
- Full checkpoint UX.
- Live agent routing from TUI.

M3 acceptance:

- `doctor` reports actionable read-only findings.
- Missing labels are reported without hidden creation.
- Capability status can be `native`, `degraded`, or `unknown`.
- Readonly TUI can display empty state and active Issue state.

## M4 Worktree And Rework Loop

Status: Todo.

Purpose:

- Add task worktree lifecycle.
- Add lease tracking.
- Add `status:rework`.
- Add `polarswarm-rework-request` and `polarswarm-rework-response`.
- Enforce default maximum 3 rework iterations.
- Support safe `full_rewrite` with backup tag and audit Comment.

M4 acceptance:

- Rework reuses the task worktree by default.
- Full rewrite safeguards are tested.
- Lease revocation prevents stale writeback.

## M5 GitHub Writeback And Init

Status: Todo.

Purpose:

- Connect to a real GitHub repository with explicit user action.
- Add safe Comment writeback.
- Add safe Label projection updates.
- Add `polarswarm init` basics for Labels and local config.
- Preserve dry-run as the default for tests.

M5 acceptance:

- Live writes require explicit task scope or CLI flag.
- Init operations are idempotent.
- Missing permissions produce actionable errors.
- `specs/` and `PolarSwarm.md` remain ignored.

## M6 Reviewer, Testers, And PR Shape

Status: Todo.

Purpose:

- Add reviewer result protocol.
- Add tester result protocol.
- Add PR body rendering.
- Add required verification reporting.
- Add review-to-rework closed loop.

M6 acceptance:

- `changes_requested` creates both audit and routing records.
- `rejected` escalates by default.
- Agent PR body includes Summary, Linked Issue, Agent Worklog, Verification, and Risk.

## M7 Capability Cache And Degradation

Status: Todo.

Purpose:

- Implement capability cache.
- Record capability events.
- Add conservative degraded behavior for unavailable GitHub features.
- Keep official docs verification as the basis for product claims.

M7 acceptance:

- Capabilities can self-heal from `degraded` to `native` after revalidation.
- Unknown capabilities are not treated as MVP dependencies.

## M8 Checkpoints And Human UX

Status: Todo.

Purpose:

- Implement `polarswarm-checkpoint`.
- Implement `polarswarm-checkpoint-response`.
- Add human blocking UX in TUI where applicable.
- Support autonomous-mode escalation notices.

M8 acceptance:

- Remote checkpoint Comment is written before local blocking UI.
- Restart can recover checkpoint state from GitHub history.

## M9 Merge, Security, Compliance, Release

Status: Later.

Purpose:

- Add `merger` semantic merge workflow.
- Add security review protocol.
- Add optional compliance agents.
- Add release gate.
- Add merge queue or degraded merge handling.

M9 acceptance:

- `merger` uses `merge-fix/<issue-number>` PRs in MVP-style flow.
- Security and compliance severity mapping is enforced.
- Auto-merge remains gated by human-reviewed policy until explicitly enabled.

## Dogfood Progression

PolarSwarm should dogfood itself only in stages:

1. Manual bootstrap: local task files, explicit tests, area-prefixed commit titles.
2. Partial dogfood: docs tasks, issue triage, dry-run writeback, mock agents.
3. Assisted dogfood: real agent backends for low-risk code and tests.
4. Controlled dogfood: PR creation and reviewer agents with human merge.
5. Future autonomous mode: only after security, permissions, and rollback boundaries are proven.

At every stage, human review is mandatory for changes to orchestrator authority, GitHub permissions, token handling, security gates, compliance gates, auto-merge, and destructive operations.
