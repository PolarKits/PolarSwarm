# Agent Protocol

PolarSwarm treats agents as workflow roles. A role is stable product semantics; a backend is a runtime implementation detail. GitHub Issues, Labels, Comments, PRs, and audit records must refer to roles, not to concrete tools or model providers.

## Role And Backend Separation

Workflow logic targets role identifiers:

```text
Issue state machine
  -> role_id
  -> agents.toml mapping
  -> backend runner
```

Backends can be `opencode`, `claude-code`, `codex`, `gemini`, `api:openai`, `api:anthropic`, an OpenAI-compatible REST backend, or a mock runner. Backends are configured locally and may change without rewriting Issue history.

Rules:

- `role_id` is written to GitHub-facing records.
- `backend_id` is runtime configuration and must not be required to replay an Issue history.
- `agent:*` Labels use coarse role projection only, such as `agent:tester` and `agent:compliance`.
- Fine-grained subtypes use message payload fields, such as `payload.subtype = "unit"` or `payload.subtype = "mlps3"`.
- Product and documentation text use `OpenCode`; configuration and command IDs use `opencode`.

## Role Tiers

Tier 1 roles are core product roles and are enabled by default:

| Role | Identifier | Responsibility |
|---|---|---|
| Orchestrator | `orchestrator` | Task decomposition, routing, state machine enforcement, deadlock detection, escalation, reconciliation. |
| Architect | `architect` | Design review, technical decisions, API contracts, ADR output. |
| Developer | `developer` | Feature implementation, bug fixes, refactoring, worktree-local edits. |
| Reviewer | `reviewer` | PR review, architecture review, interface contract validation, coding standard checks. |
| Security | `security` | SAST, dependency risk review, threat modeling, secure code review. |
| Documenter | `documenter` | API docs, architecture docs, comments, changelog, user docs. |
| Debugger | `debugger` | Reproduction, root cause analysis, hotfix implementation. |
| Merger | `merger` | Semantic conflict analysis and merge-fix PR creation. |

Tier 2 tester roles are enabled as needed:

| Role | Identifier | Trigger |
|---|---|---|
| Unit tester | `tester:unit` | Every PR or module-level change. |
| Integration tester | `tester:integration` | Cross-module, database, queue, or service boundary changes. |
| E2E tester | `tester:e2e` | Completed user-facing feature paths. |
| API tester | `tester:api` | API contract, validation, or error-code changes. |
| Performance tester | `tester:performance` | Release preparation or performance-sensitive changes. |
| Regression tester | `tester:regression` | Release preparation and suite maintenance. |
| Accessibility tester | `tester:accessibility` | UI changes. |
| Chaos tester | `tester:chaos` | Manual or explicit resilience tasks. |

Tier 3 roles are optional: `devops`, `analyst`, `dba`, `localizer`, and `dependency`.

Tier 4 compliance roles are optional and disabled by default: `compliance:mlps2`, `compliance:mlps3`, `compliance:mlps4`, `compliance:pipl`, `compliance:dsl`, `compliance:csl`, `compliance:gdpr`, `compliance:ccpa`, `compliance:pci-dss`, `compliance:hipaa`, `compliance:soc2`, and `compliance:iso27001`.

## Worktree Contract

Writing roles work in task-scoped Git worktrees. A Task Issue owns one branch and one worktree:

```text
branch:   task/<issue-number>
worktree: .polarswarm/worktrees/task-<issue-number>-<slug>/
```

Merge repair work uses:

```text
branch:   merge-fix/<issue-number>
```

Required behavior:

- Create the task worktree when the Issue enters `status:assigned`.
- Keep the main checkout clean.
- Inject stable environment variables: `POLARSWARM_TASK_ID`, `POLARSWARM_WORKTREE_ID`, `POLARSWARM_PORT_OFFSET`, and `TMPDIR`.
- Prefer dynamic ports in tests; derive fixed ports from `POLARSWARM_PORT_OFFSET` only when dynamic ports are not possible.
- Reuse the same worktree during `status:rework`.
- Only reset a task branch when a rework request sets `"full_rewrite": true`.

`full_rewrite` is destructive and requires all of these safeguards:

- Target branch must match `task/<issue-number>`.
- Create `backup/task-<issue-number>-pre-rewrite-<ts>` before reset.
- Write an audit Comment with the old head SHA.

## Message Envelope

Agent routing uses Issue Comments with a human-readable summary and a machine-readable `polarswarm-msg` block:

````markdown
[POLARSWARM] task_assign -> agent:developer

Implement the task described by this Issue.

```polarswarm-msg
{
  "version": "1",
  "msg_id": "msg:orchestrator-20260517-143000-0001",
  "correlation_id": "task:45",
  "causation_id": null,
  "from": "agent:orchestrator",
  "to": "agent:developer",
  "type": "task_assign",
  "issue_ref": 45,
  "payload": {
    "subtype": null,
    "instructions": "Implement the accepted task scope.",
    "artifacts_expected": ["internal/workflow/runner.go"],
    "timeout_minutes": 120,
    "backend_hint": "opencode"
  },
  "ts": "2026-05-17T14:30:00Z"
}
```
````

Required fields:

| Field | Requirement |
|---|---|
| `version` | Protocol version string. MVP uses `"1"`. |
| `msg_id` | Globally unique message ID generated by sender. |
| `correlation_id` | Business flow root, usually `task:<issue-number>` or `release:<issue-number>`. |
| `causation_id` | Parent `msg_id`; `null` for root messages. |
| `from` | Sender role, prefixed with `agent:` or `human:` where applicable. |
| `to` | Recipient role projection. |
| `type` | Message type from the allowed list below. |
| `issue_ref` | GitHub Issue number. |
| `payload` | Type-specific structured data. |
| `ts` | ISO 8601 UTC timestamp. |

The context reader must strip machine JSON blocks before injecting Issue history into an LLM. It may keep the human summary and non-machine prose.

## Message Types

| Type | Route | State effect |
|---|---|---|
| `task_assign` | `orchestrator -> agent` | Add `agent:X`, set `status:assigned`. |
| `task_start` | `agent -> orchestrator` | Set `status:in-progress`. |
| `task_done` | `agent -> orchestrator` | Set `status:review`, remove `agent:X`. |
| `task_handoff` | `agent A -> agent B` | Replace `agent:A` with `agent:B`. |
| `blocked_notify` | `agent -> orchestrator` | Set `status:blocked`, record dependency. |
| `unblocked_notify` | `orchestrator -> agent` | Remove blocked state, set `status:assigned`. |
| `request_review` | `agent -> orchestrator` | Trigger reviewer or human checkpoint. |
| `escalate` | `agent -> orchestrator` | Add `agent:human` and `decision:needs-info`. |
| `checkpoint_approve` | `human/orchestrator -> orchestrator` | Add `decision:approved`, continue checkpoint transition. |
| `complete_approve` | `reviewer/human -> orchestrator` | Complete acceptance when checks are satisfied. |
| `reject` | `human/orchestrator -> agent` | Move to `status:rework` or `status:abandoned`. |
| `merge_request` | `orchestrator -> merger` | Trigger semantic merge analysis. |
| `merge_result` | `merger -> orchestrator` | Return conflict analysis and confidence. |
| `review_result` | `reviewer/security/compliance:* -> orchestrator` | Publish structured review outcome. |
| `rework_request` | `reviewer/tester/security/compliance:* -> agent` | Route rework and enter `status:rework`. |
| `rework_done` | `developer/documenter/architect -> orchestrator` | Return to review. |
| `test_result` | `tester:* -> orchestrator` | Publish test summary and failures. |
| `triage_approve` | `human -> orchestrator` | Activate `status:pending-triage` Issue. |
| `triage_reject` | `human -> orchestrator` | Reject and close a pending Issue. |

All message handling must be idempotent by `msg_id`. `issue_comment:edited` must refresh display state only; it must not replay workflow transitions.

## Audit Comment Types

`polarswarm-msg` is the only generic routing envelope. Other machine blocks are audit events and must not be double-routed as message types.

| Code block | Purpose |
|---|---|
| `polarswarm-agent-result` | Agent stage result with branch, commit, verification, confidence. |
| `polarswarm-checkpoint` | Human checkpoint request. |
| `polarswarm-checkpoint-response` | Human or autonomous checkpoint decision. |
| `polarswarm-escalation` | Autonomous mode escalation notice. |
| `polarswarm-review-result` | Reviewer, security, or compliance outcome. |
| `polarswarm-rework-request` | Rework feedback items and iteration count. |
| `polarswarm-rework-response` | Rework completion summary. |
| `polarswarm-merge-result` | Merge confidence and conflict analysis. |
| `polarswarm-test-result` | Test result summary and failure details. |

## Agent Result

Every auditable agent stage writes a `polarswarm-agent-result` Comment:

````markdown
[POLARSWARM:AGENT-RESULT] developer - completed

```polarswarm-agent-result
{
  "agent": "developer",
  "role": "implementation",
  "issue": 45,
  "branch": "task/45",
  "commit": "abc123",
  "status": "completed",
  "verification": ["go test ./..."],
  "confidence": 0.86,
  "ts": "2026-05-17T14:00:00Z"
}
```
````

This record is used for recovery, audit, context compression, release notes, and future dogfood workflows. Labels are only projections and cannot replace this record.

## Review And Rework

Reviewer output must be written as `polarswarm-review-result`.

Allowed review outcomes:

- `pass`: may proceed when required checks or accepted degraded checks also pass.
- `changes_requested`: must create a `polarswarm-rework-request`, route a `rework_request`, and enter `status:rework`.
- `rejected`: reserved for unacceptable design, safety issues, or compliance blockers. It should normally escalate for human confirmation before `status:abandoned`.

Rework is iterative. The orchestrator tracks `rework_iteration`; the default maximum is 3. After the maximum is exceeded, PolarSwarm must stop automatic rework and escalate to `agent:human`.

## Merge Agent

The `merger` role is separate from `orchestrator`. In MVP, `merger` must not mutate the target PR directly. It creates a `merge-fix/<issue-number>` PR containing:

- Base, ours, and theirs intent summary.
- Merge strategy.
- Kept, rewritten, or discarded fragments.
- Risk points.
- Files that need reviewer attention.
- Confidence score.

The default semantic merge confidence threshold is `0.85`. Lower confidence escalates to human review without blocking unrelated workflows.
