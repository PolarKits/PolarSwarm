# Observability

PolarSwarm records local runtime evidence for diagnosis, cost awareness, fallback analysis, and TUI display. GitHub remains the durable workflow audit surface; local observability data is a cache and operational history.

## Storage Boundary

Observability uses a dual-track local design:

| Store | Purpose | Authority |
|---|---|---|
| SQLite | Structured queries, aggregated views, cursors, idempotency, leases, and metrics. | Local runtime cache only. |
| JSONL | Append-only event trail for human inspection and recovery. | Local diagnostic trail only. |

Neither store may replace GitHub Comments, Labels, PRs, or checks as workflow authority.

## File Layout

Default paths are configured in `.polarswarm/core.toml`:

| Path | Purpose |
|---|---|
| `.polarswarm/data/metrics.db` | SQLite database. |
| `.polarswarm/data/logs/agent_calls.jsonl` | Agent/backend invocation events. |
| `.polarswarm/data/logs/fallback_events.jsonl` | Model/backend fallback decisions. |
| `.polarswarm/data/logs/workflow_events.jsonl` | Local workflow transitions, rework, leases, and cleanup diagnostics. |
| `.polarswarm/data/logs/capability_events.jsonl` | GitHub capability state changes. |

Tasks may use in-memory stores in M1 when persistence is not required. Persistent paths must be configurable before write-enabled local runs.

## SQLite Data Sets

The schema should be versioned. Exact table definitions belong in implementation tasks, but these data sets must be represented:

| Data Set | Required Content |
|---|---|
| `schema_version` | Current local schema and migration state. |
| `cursors` | Repository, Issue, Comment, PR, and capability polling cursors. |
| `idempotency` | Processed `msg_id`, `op_id`, remote event IDs, and write-plan identities. |
| `leases` | Active local work claims, owner process, Issue number, branch/worktree, expiry, and revocation state. |
| `capability_cache` | Capability key, state, probe source, degraded behavior, last checked time. |
| `agent_runs` | Role, backend, model, issue, status, duration, token/cost summary, fallback marker. |
| `fallback_events` | Fallback trigger, original target, selected fallback, result, and safety profile. |
| `rework_events` | Review outcome, requested changes, iteration count, and escalation reason. |
| `workflow_events` | Local transition decisions and remote reconciliation diagnostics. |

Secrets, raw prompts containing secrets, full tokens, and unredacted environment data must not be stored.

## Metrics

The runtime and TUI should derive these metrics when data exists:

| Metric | Use |
|---|---|
| Token and cost totals by backend/model/role | Cost visibility and backend tuning. |
| Fallback rate by role and backend | Detect backend degradation. |
| Rework count and rate by role | Detect poor task decomposition or weak review. |
| Capability state changes | Explain degraded GitHub behavior. |
| Issue lifecycle duration | Identify blocked or slow workflow stages. |
| Lease revocations and cleanup blocks | Detect stale local execution state. |

M1 does not require full metrics. M1 only needs enough idempotency state to prove rerun safety.

## Event Rules

- JSONL events are append-only.
- Event records must include timestamp, category, stable ID, and redacted context.
- Duplicate events are acceptable only when the idempotency key differs.
- Capability, fallback, rework, and lease revocation events must be visible to `doctor` or TUI once those surfaces exist.
- Retention is controlled by config; deletion must not affect GitHub audit history.

## Milestone Boundary

- M1: idempotency and processed state may be in-memory or SQLite; no full metrics requirement.
- M2: agent run records should exist when backend execution is implemented.
- M3: readonly TUI may display store snapshots and doctor findings.
- Later: cost dashboards, fallback trends, rework analytics, retention cleanup, and schema migrations.
