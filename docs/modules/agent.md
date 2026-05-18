# Agent Module

## Responsibility

Expose a role-oriented runner interface and backend adapters. Agent execution must normalize stdout, stderr, exit code, duration, verification, confidence, and error type into auditable results.

## Non-Responsibilities

- Do not decide product workflow transitions.
- Do not write GitHub Comments or Labels directly.
- Do not encode GitHub actor policy.
- Do not require a real backend for default tests.

## Inputs

- Role ID and backend mapping from effective config.
- Task instructions and bounded context.
- Working directory or future task worktree path.
- Timeout, model, and output settings where configured.

## Outputs

- Normalized agent result.
- Safe verification summary.
- Redacted logs or error summaries.
- Backend capability or availability errors.

## M1/M2 Task Entry Points

- M1: mock backend result for the minimal IssueOps loop.
- M1: renderable data for `polarswarm-agent-result`.
- M2: `BackendRunner` interface and first real CLI backend adapter.
- M2: secret masking, timeout handling, and cwd enforcement.
