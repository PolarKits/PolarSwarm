# TUI Module

## Responsibility

Display local PolarSwarm state for humans. The first TUI milestone is read-only and should make empty state, active Issue state, and doctor findings understandable.

Detailed user-facing behavior is defined in `../tui-design.md`. This module document only defines the implementation boundary.

## Non-Responsibilities

- Do not perform GitHub writes in M3.
- Do not route agents directly.
- Do not replace GitHub as the durable audit surface.
- Do not become required for the M1/M2 command path.

## Inputs

- Local store snapshots.
- Normalized Issue and workflow status.
- Doctor result summaries.
- Runtime progress events where available.

## Outputs

- Read-only terminal UI views.
- Human-readable status and error summaries.
- Later milestones may emit explicit user decisions only through workflow-approved paths.

## Write Boundary

M3 TUI must not perform writes. Later interactive actions must emit workflow-approved commands; the orchestrator remains responsible for policy checks and GitHub audit writes.

## M1/M2 Task Entry Points

- No M1 requirement.
- No M2 requirement.
- M3: readonly dashboard, empty state, active Issue state, and no-write guarantees.
- Later: checkpoint response UX and monitoring views described in `../tui-design.md`.
