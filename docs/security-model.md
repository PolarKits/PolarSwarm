# Security Model

PolarSwarm assumes GitHub Issues, Comments, and Labels can be touched by people who must not automatically trigger local code execution. GitHub is the collaboration and audit surface; the local PolarSwarm process is the execution engine and must verify remote state before every action.

## Threat Model

| Threat | Risk |
|---|---|
| Malicious Issue submission | External users can attempt to consume API quota, model budget, or local execution time. |
| Forged agent Comment | A user can write a Comment that looks like a `polarswarm-msg` and attempts to move the state machine. |
| Unauthorized Label mutation | GitHub does not provide label-level ACL, so users with Issues write access can mutate workflow Labels. |
| Resource exhaustion | High-volume Issues or Comments can exhaust Actions, polling budget, or model budget. |
| Secret leakage | Backend output, logs, PR bodies, or audit records can accidentally include credentials. |
| Stale local execution | A local agent can continue after remote Issue state or lease ownership changed. |

## Authority Model

The authoritative state is the replayable GitHub Issue / PR / Comment event stream plus versioned policy files in Git. Local SQLite stores cursors, idempotency records, leases, and metrics only. It is not the final source of truth.

`status:*`, `agent:*`, and `decision:*` Labels are projections. When Labels conflict with replayed events, the orchestrator must repair Labels from the replayed event state.

Each execution loop follows Fetch-Validate-Execute:

1. Fetch current Issue, PR, Comment, Label, and close state.
2. Replay relevant `polarswarm-msg` and audit events by stable IDs.
3. Validate local leases against remote state.
4. Cancel or revoke stale work before running a local agent.
5. Execute only after the remote state is still valid.
6. Write append-only audit records before changing projection Labels where possible.

If the Issue is closed, the assigned `agent:*` Label is removed, or a later human decision overrides the task, the local lease is revoked. A running agent should be cancelled. If it cannot be cancelled, its output may be logged locally for diagnostics but must not advance GitHub state.

## Actor Gate

PolarSwarm distinguishes:

- `workflow_actor`: the GitHub account that triggered an Action event.
- `authorized_actor`: a user allowed to advance PolarSwarm workflow or run management commands.

The `polarswarm-triage.yml` workflow must not route agent work directly. It only performs gatekeeping side effects:

- Known authorized actors can enter normal flow.
- Non-whitelisted actors enter `status:pending-triage` with `agent:human`.
- Blacklisted actors can be closed automatically.
- Only authorized `/triage` and `/reject` commands move a pending Issue.

External Issues must not consume agent resources before authorization.

## Access Configuration

The product-level access policy is stored in `.polarswarm/access.toml`:

```toml
[whitelist]
mode = "collaborators"
github_team = "polarswarm-agents"
users = ["yangsen", "bot-polarswarm"]

[blacklist]
enabled = true
users = []

[triage]
enabled = true
auto_close_on_blacklist = true
triage_command = "/triage"
reject_command = "/reject"
max_pending_days = 7

[comment_security]
verify_actor = true
hmac_signing = true
hmac_secret_env = "POLARSWARM_HMAC_SECRET"
```

Whitelist modes:

- `collaborators`: recommended default; reads repository collaborators.
- `file`: static list for offline or restricted API environments.
- `github-team`: organization team based authorization.

Human commands such as `/triage` and `/reject` are authorized by actor identity. Machine `polarswarm-msg` comments should use HMAC signing when enabled. Audit-only blocks still require actor validation before they influence workflow decisions.

## Label Protection

GitHub does not provide label-level ACL. PolarSwarm uses detection and repair:

- `polarswarm-label-guard.yml` listens to `issues:labeled` and `issues:unlabeled`.
- Unauthorized changes to `status:*`, `agent:*`, and `decision:*` are reverted.
- The guard writes an alert Comment when it repairs a protected Label.
- The orchestrator still treats replayed events as authority and repairs projections during reconciliation.

This is detection-after-change, not prevention-before-change. High-security deployments should combine this with strict token permissions, HMAC signing, actor allowlists, and conservative runtime tokens.

## Token Boundaries

PolarSwarm uses two token classes:

| Token | Use | Lifetime |
|---|---|---|
| Init token | Create Labels, Issue Types, Projects V2, workflow files, repository rules, and other high-permission setup resources. | Short-lived, used during `polarswarm init`. |
| Runtime token | Poll Issues, read/write Comments, update Labels, manage task PRs, and perform normal reconciliation. | Long-lived, least privilege. |

Runtime code must not silently perform init-only operations. `doctor` must report missing permissions and suggested remediations instead of escalating writes automatically.

## Write Safety

All GitHub write paths must be explicit and auditable:

- Bootstrap and tests use dry-run by default.
- `doctor` commands do not perform hidden writes.
- Write capability probes require an explicit `--write-probe` style opt-in.
- Write probes must use disposable test Issues, PRs, or commits and print the target resources.
- Every retryable write has an operation ID such as `op:{issue_ref}:{operation}:{target}:{attempt}`.
- Append-only Comments are preferred for audit before projection Label updates.
- No workflow advancement is allowed after local lease revocation.

## Human Approval Boundaries

Even during future dogfood mode, these changes require human confirmation:

- Orchestrator state machine changes.
- GitHub write permission or token scope changes.
- Auto-merge behavior.
- Security or compliance gates.
- Token, secret, or HMAC handling.
- Destructive local operations such as branch reset.
- Remote state deletion, rollback, or forced repair.

PolarSwarm may assist its own development, but it must not be the sole approver for changes that alter its authority model.

## Secret Handling

Secrets must not be written into:

- Logs.
- Comments.
- PR bodies.
- Test snapshots.
- `polarswarm-agent-result`.
- Review, rework, merge, or test result audit blocks.

Backend output must be masked before entering structured audit records. The masking layer must run before local persistence and before GitHub writes.

## Security And Compliance Outcomes

Security and compliance findings use four severities:

- `CRITICAL`: reject or block release by default.
- `HIGH`: block until remediation or explicit human risk acceptance.
- `MEDIUM`: record and normally allow flow with follow-up.
- `LOW`: record for visibility.

Compliance agents are disabled by default and enabled per standard. Security-strict fallback profiles must avoid sending security or compliance content to broader third-party fallback queues unless the user explicitly configures that behavior.
