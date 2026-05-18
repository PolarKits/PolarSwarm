# Store Module

## Responsibility

Persist local runtime state that supports reconciliation. Store is a cache and coordination aid, not the authoritative workflow history.

## Non-Responsibilities

- Do not replace GitHub Comments, Labels, PRs, or checks as the remote audit trail.
- Do not decide workflow transitions.
- Do not hold unmasked secrets.
- Do not make cleanup decisions without workflow leases.

## Inputs

- Cursor keys and remote event identifiers.
- Message IDs and operation IDs.
- Lease records.
- Capability cache entries and runtime metrics where implemented.

## Outputs

- Cursor positions.
- Idempotency lookup results.
- Lease acquisition and revocation results.
- Local diagnostics for doctor and TUI.

## M1/M2 Task Entry Points

- M1: in-memory or SQLite-backed idempotency for `msg_id` and `op_id`.
- M1: store processed Issue state enough to prove rerun safety.
- M2: backend execution records only if needed for result rendering.
