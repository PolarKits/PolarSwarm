# Config Module

## Responsibility

Load, merge, and validate PolarSwarm configuration for local execution. M1 focuses on a minimal `.polarswarm/core.toml` path or explicit config path; later milestones can add role/backend and policy composition.

## Non-Responsibilities

- Do not call GitHub.
- Do not execute agents.
- Do not decide workflow transitions.
- Do not create repository files during normal runtime; initialization belongs to future init tasks.

## Inputs

- Default config path such as `.polarswarm/core.toml`.
- Explicit CLI config path where supported.
- Environment variables only when a task defines precedence.

## Outputs

- Parsed effective config.
- Validation errors with actionable field names.
- Redacted config summary for diagnostics where needed.

## M1/M2 Task Entry Points

- M1: config path discovery, required field validation, test fixtures.
- M1: default dry-run setting exposed to workflow/app code.
- M2: backend mapping fields used by the agent runner.
