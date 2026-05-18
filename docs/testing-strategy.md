# Testing Strategy

PolarSwarm uses staged testing. Early milestones must avoid real GitHub writes unless a task explicitly opts in and names the disposable target resources.

## Principles

- Prefer pure unit tests for state machines, parsers, rendering, config loading, and policy decisions.
- Use fake GitHub clients for API-facing behavior.
- Use dry-run integration tests before live writes.
- Treat GitHub live tests as optional, explicit, and auditable.
- Verify idempotency for every message and write operation.
- Test security gates before enabling any real agent execution.

## Test Pyramid

1. Unit tests for pure logic.
2. Golden tests for rendered Comments, PR bodies, and config output.
3. Fake GitHub client tests for API-facing behavior.
4. Dry-run integration tests for full local plans.
5. Optional real GitHub tests against dedicated test Issues, PRs, commits, or labels.

## GitHub Write Rule

No test should write to GitHub by default.

Real write tests must:

- Require explicit opt-in.
- Use a test Issue, test PR, test commit, or disposable repository resource.
- Print all target resources before writing.
- Be safe to rerun.
- Leave an audit trail.
- Clean up only resources they created, or clearly report manual cleanup steps.

`doctor capabilities --write-probe` style behavior belongs behind explicit user confirmation and must not run as part of default test suites.

## M1 Required Coverage

M1 proves the minimal IssueOps loop without real GitHub writes:

- Config file missing.
- Required config field missing.
- `--config` or explicit config path precedence where implemented.
- Dry-run default for write plans.
- Issue label filtering.
- `status:*` transition validation.
- Single active `status:*` label plan.
- Message `msg_id` idempotency.
- Operation `op_id` idempotency.
- Mock agent success.
- Mock agent failure.
- `polarswarm-agent-result` rendering.
- Context reader strips machine JSON blocks.
- Label update plan for comment plus status projection.

M1 acceptance should demonstrate:

```text
Issue -> Orchestrator -> Mock Agent -> Agent Result Comment Plan -> Label Plan
```

## M2 Required Coverage

M2 introduces backend execution:

- `BackendRunner` interface behavior.
- Mock backend remains available.
- First real CLI backend command construction.
- Worktree `cwd` enforcement.
- Model and output-format override precedence.
- Timeout handling.
- Exit-code normalization.
- Stderr and stdout capture with secret masking.
- Backend failure mapped to retry, fallback, or escalation.
- Agent result contains safe verification output only.

Real backend invocation tests may be opt-in if they consume tokens or depend on local tools.

## M3 Required Coverage

M3 adds `doctor` and readonly TUI:

- `doctor github` read-only checks.
- `doctor labels` detects missing labels without creating them by default.
- `doctor capabilities` reports `native`, `degraded`, or `unknown`.
- `doctor llm` is explicit opt-in because it consumes tokens.
- Readonly TUI renders empty state.
- Readonly TUI renders active Issue state.
- TUI never performs writes.

## Security Tests

Security behavior should be tested before any live workflow writes:

- Non-whitelisted actor enters `status:pending-triage`.
- Authorized `/triage` activates a pending Issue.
- Authorized `/reject` closes or rejects a pending Issue.
- Blacklisted actor is rejected.
- Forged `polarswarm-msg` is ignored when actor or HMAC validation fails.
- Unauthorized protected Label mutation is planned for repair.
- Lease revocation prevents workflow advancement.
- Secrets are masked in rendered audit output and logs.

## Protocol Tests

Protocol coverage should include:

- Required fields in `polarswarm-msg`.
- Unknown protocol version handling.
- Unknown message type handling.
- Duplicate `msg_id` handling.
- Edited comment does not replay state transitions.
- `polarswarm-review-result=changes_requested` creates rework routing and audit requirements.
- `review_result=rejected` escalates rather than silently abandoning work.
- `rework_iteration` stops at default maximum 3.
- `full_rewrite` requires branch and backup safeguards.

## Review Model

Small implementation models can execute individual `@specs/M{N}/T{NNN}-{SLUG}.md` tasks. A stronger reasoning model should review:

- Whether the task stayed inside allowed files.
- Whether acceptance criteria were met.
- Whether verification commands ran and passed.
- Whether added tests cover important edge cases.
- Whether security and GitHub write boundaries were respected.
- Whether implementation diverged from `docs/`.

Review findings should be fixed within the same task scope before moving to the next task. If the fix requires broader work, create a new task.

## Coverage Verifier

After each milestone, a coverage verifier should compare:

- Milestone tasks under `specs/M{N}/`.
- Versioned docs under `docs/`.
- Implemented code and tests.
- Remaining source content in local `PolarSwarm.md`.

The verifier should report missing coverage, contradictions, stale local-only assumptions, and docs that still depend on unimplemented automation.
