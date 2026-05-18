# Configuration

This document defines the versioned configuration surface used by implementation tasks. Product goals remain in `product-requirements.md`; module behavior remains in `architecture.md` and `modules/`.

## Scope

Configuration is stored under `.polarswarm/` in the repository unless an explicit CLI config path is provided. M1 only requires enough configuration to run the minimal dry-run loop. Later milestones add agent backends, access policy, observability, and TUI preferences.

## File Set

| File | Purpose | M1/M2 Requirement |
|---|---|---|
| `.polarswarm/core.toml` | Repository identity, dry-run defaults, worktree path, polling, retention, and observability toggles. | M1 requires repository identity and dry-run default. M2 may add worktree path. |
| `.polarswarm/agents.toml` | Role definitions and role-to-backend mapping. | M2 requires minimal role/backend mapping for one agent. |
| `.polarswarm/llm.toml` | Backend providers, model aliases, default models, and fallback profiles. | M2 may use mock/local backend only; real backend config is later. |
| `.polarswarm/access.toml` | Actor whitelist, blacklist, triage behavior, and HMAC policy. | M1 may ignore; M2+ security checks should read it when present. |
| `.polarswarm/tui.toml` | Theme and local display preferences. | Not required before M3. |

`polarswarm init` owns template creation. Normal runtime commands must not create missing repository configuration files implicitly.

## Discovery And Precedence

Configuration discovery uses this order, highest precedence first:

| Rank | Source | Use |
|---|---|---|
| 1 | Explicit CLI config path | Task-specific or test-specific override. |
| 2 | Environment override named by a task | Only when the task defines the exact variable and precedence. |
| 3 | Project `.polarswarm/` | Default project configuration. |
| 4 | User config such as `~/.polarswarm/` | Optional defaults for local preferences and backend credentials. |

Project configuration should be versioned. User configuration must not be required to replay repository workflow state.

## Merge Rules

- The effective config is built per file and per logical section.
- Project config overrides user defaults for repository behavior.
- Explicit CLI paths override discovered project files.
- Secrets are referenced by environment variable name or credential provider name; they are never stored in versioned config.
- Unknown fields are warnings in early milestones and may become errors once a schema version is declared.
- Cross-file references must validate: agent roles must reference known backend IDs, model aliases, and fallback profiles.

## Minimal Schema

`core.toml` must support these logical sections over time:

| Section | Required Fields | Notes |
|---|---|---|
| `repository` | `owner`, `name` | M1 may allow fixture-only mode without these fields. |
| `runtime` | `dry_run` | Default must be `true` unless a task explicitly enables writes. |
| `worktree` | `base_dir` | Default is `.polarswarm/worktrees`; M2+ worktree tasks use it. |
| `observability` | `storage`, `db_path`, `log_dir`, `retention_days` | Defined by `observability.md`; not required for M1 unless store tasks need it. |
| `polling` | `interval_seconds`, `rate_budget` | Later GitHub polling milestones. |

`agents.toml` must define role entries with stable role IDs:

| Field | Requirement |
|---|---|
| `role` | Stable role ID such as `developer`, `reviewer`, or `merger`. |
| `backend` | Backend ID configured in `llm.toml` or a built-in mock backend. |
| `model` | Optional direct model ID or model alias. |
| `fallback` | Optional fallback profile reference. |
| `enabled` | Boolean; disabled roles must not be scheduled. |

`llm.toml` must define backend IDs, model aliases, backend defaults, and fallback profiles. Fallback profiles are ordered queues. A profile may restrict provider family, model class, or safety posture. The initial required profiles are:

| Profile | Purpose |
|---|---|
| `anthropic-degraded` | Stay within an Anthropic-compatible backend family. |
| `full-degraded` | Allow broader backend fallback for non-sensitive tasks. |
| `security-strict` | Restrict security/compliance work to explicitly approved backends. |

`access.toml` must define actor gates:

| Section | Requirement |
|---|---|
| `whitelist` | Users, teams, or bots allowed to submit routeable messages. |
| `blacklist` | Actors that always force `pending-triage` or rejection. |
| `triage` | Behavior for unknown actors. Default should be conservative. |
| `hmac` | Optional signed-command validation policy. |

## Validation Output

Config validation errors must name the file, section, field, and expected behavior. Diagnostics must redact secrets and token-like values. `doctor config` is the user-facing command for full validation.

## Milestone Boundary

- M1: parse minimal `core.toml`, support explicit config path, default dry-run behavior, and actionable validation errors.
- M2: validate one role/backend mapping and mock or CLI backend references.
- M3: read `tui.toml` only for local display preferences.
- Later: enforce full fallback profiles, access policy, polling budget, and observability retention.
