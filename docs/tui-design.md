# TUI Design

The TUI is a local human interface for observing PolarSwarm and, in later milestones, responding to approved workflow prompts. GitHub remains the durable audit and collaboration surface.

## Design Principles

- Read from local snapshots and workflow-approved command interfaces.
- Do not write directly to GitHub, worktrees, or SQLite from UI widgets.
- Show empty, degraded, blocked, and error states explicitly.
- Treat checkpoint prompts as recovery views for already-published GitHub checkpoint Comments.
- Keep all write-capable actions auditable through structured Comments or approved workflow commands.

## Views

| View | Purpose | Milestone |
|---|---|---|
| Dashboard | Current repository, active Issues, agent status, dry-run/write mode, and degraded capabilities. | M3 |
| Issue Detail | One Issue's status, labels, latest routeable messages, write plans, and agent result summary. | M3+ |
| Doctor | Local config, GitHub access, labels, workflows, store, and backend readiness summaries. | M3 |
| Monitor | Token/cost, fallback, rework, capability events, and lease health from local observability data. | Later |
| Checkpoint | Blocking human decision prompt derived from a remote `polarswarm-checkpoint` Comment. | Later |

## M3 Readonly Dashboard

M3 is display-only. It must support:

- Empty repository or no active Issue state.
- One or more active Issue summaries.
- Dry-run mode visibility.
- Doctor findings and degraded capability summaries.
- Clear error state when config or store cannot be read.
- No GitHub writes, no agent routing, and no direct workflow transitions.

## Checkpoint UX

Checkpoint interaction is not required before later milestones. When implemented, the order is fixed:

1. Orchestrator writes a `polarswarm-checkpoint` Comment to GitHub and updates decision/status projection.
2. TUI detects the unresolved checkpoint from GitHub-derived state.
3. TUI shows a blocking prompt with Issue context, agent summary, options, and optional human note.
4. User choice is emitted through a workflow-approved command path.
5. Orchestrator writes `polarswarm-checkpoint-response` to GitHub.
6. TUI returns to the normal view after remote state confirms the response.

If the local process restarts, the TUI must reconstruct unresolved checkpoint prompts from GitHub state, not from local-only memory.

## Future Interactions

Future write-capable TUI actions may include approve/reject checkpoint, request rework, pause/resume local polling, and open a local worktree. These actions must be routed through workflow commands that enforce actor policy, dry-run mode, and audit writes.

## Non-Goals

- The TUI is not the source of truth.
- The TUI does not replace GitHub Issues or PRs.
- The TUI does not embed agent business logic.
- The TUI does not directly mutate repository files or GitHub resources.
