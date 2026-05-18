# Doctor Module

## Responsibility

Report whether local configuration, repository access, labels, workflows, store, worktrees, and optional backends are ready for the requested milestone.

## Non-Responsibilities

- Do not silently create or mutate GitHub resources.
- Do not run token-consuming backend checks unless explicitly requested.
- Do not replace test suites.
- Do not decide workflow routing.

## Inputs

- Effective config.
- GitHub client or fake/read-only client.
- Local store path and schema state.
- Capability cache.
- Optional backend probe request.

## Outputs

- Structured findings with severity and category.
- Human-readable remediation hints.
- Capability status summaries.
- Explicit indication of skipped or opt-in checks.

## M1/M2 Task Entry Points

- No hard M1 requirement beyond config validation reuse if convenient.
- M2 may expose backend availability checks only behind explicit opt-in.
- M3: `doctor github`, `doctor labels`, `doctor capabilities`, and opt-in `doctor llm`.
