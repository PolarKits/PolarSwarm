# Workflow Module

## Responsibility

Own reconciliation and state transition decisions. Workflow reads normalized remote state, validates it against policy and local state, dispatches eligible agent work, and requests GitHub write plans.

## Non-Responsibilities

- Do not call GitHub APIs directly; use the GitHub module interface.
- Do not parse local config files directly.
- Do not execute backend commands directly.
- Do not treat Labels as the only source of truth.

## Inputs

- Effective config.
- Normalized Issue, Label, and Comment history.
- Stored cursors, idempotency records, and leases.
- Agent execution results.

## Outputs

- Workflow decisions.
- Agent dispatch requests.
- Comment and Label write plans.
- Pause, block, or escalation decisions.

## M1/M2 Task Entry Points

- M1: minimal loop from one eligible Issue to mock agent dispatch.
- M1: `status:*` transition validation and single active status projection.
- M1: idempotency by `msg_id` and `op_id`.
- M2: consume normalized backend results without binding to a concrete agent tool.
