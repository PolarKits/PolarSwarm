# Module Docs

Module documents are short contracts for implementation tasks. They derive from the PRD and architecture:

```text
PRD -> Architecture -> Module Docs -> specs tasks
```

They may define responsibilities, non-responsibilities, inputs, outputs, and milestone task entry points. They must not redefine users, product goals, MVP scope, or acceptance criteria.

## Module Index

| Module | Document | Purpose |
|---|---|---|
| `config` | [config.md](config.md) | Load and validate local configuration. |
| `github` | [github.md](github.md) | Provide testable GitHub API access and write plans. |
| `workflow` | [workflow.md](workflow.md) | Reconcile Issue state and choose transitions. |
| `agent` | [agent.md](agent.md) | Run role-bound agent backends and normalize results. |
| `store` | [store.md](store.md) | Persist cursors, leases, idempotency records, and local metrics. |
| `tui` | [tui.md](tui.md) | Display local runtime and Issue state. |
| `doctor` | [doctor.md](doctor.md) | Report local and repository readiness. |
| `init` | [init.md](init.md) | Prepare repository resources and config templates through explicit setup. |

## Task Mapping

Milestone tasks should cite the relevant module document plus the cross-cutting docs they depend on. Example:

```text
@docs/modules/workflow.md
@docs/issueops-model.md
@specs/M1/T004-minimal-state-machine.md
```

When a module needs behavior not covered here, update the smallest relevant docs first. Do not hide product changes inside a task file.
