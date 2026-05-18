# PolarSwarm — Agent Guide

> This file is for AI coding agents. It assumes you know nothing about the project. Read this before editing code or tests.

## Project Overview

PolarSwarm is a **local IssueOps orchestrator for multi-agent software development**, written in Go. It coordinates AI-assisted development by mapping work to GitHub Issues, while keeping agent execution local on the developer's machine.

- **GitHub** is the remote control plane: Issues, Comments, Labels, Pull Requests, and branch protection rules provide the collaboration and audit surface.
- **The local PolarSwarm process** performs execution: polling GitHub, reconciling state, dispatching work to agent backends, managing local worktrees, and writing results back.
- **Agents** operate in isolated Git worktrees and are bound to roles (e.g., `developer`, `reviewer`, `security`). Roles are stable semantics; backends (OpenCode, Claude Code, Codex, API providers, or mock runners) are runtime configuration.

The project is currently in a **spec-guided bootstrap** phase. The first usable milestone (M1) is a minimal dry-run loop:

```text
Issue -> Orchestrator -> Mock Agent -> Agent Result Comment Plan -> Label Plan
```

## Technology Stack

| Layer | Choice |
|---|---|
| Language | Go 1.22 |
| Dependencies | None outside the Go standard library (as of current bootstrap) |
| Build system | Standard Go toolchain (`go build`, `go test`) |
| CLI framework | Hand-rolled flag parsing in `internal/app/` |
| Configuration | Custom TOML-like parser in `internal/config/` |
| Testing | Standard `testing` package with table-driven tests |
| CI / Docker | Not yet present |

## Build and Test Commands

```sh
# Run all tests
go test ./...

# Run the CLI
go run ./cmd/polarswarm help
go run ./cmd/polarswarm version
go run ./cmd/polarswarm config check [--config path]
go run ./cmd/polarswarm issue read --repo owner/name --number N --fixture path

# Build the binary
go build ./cmd/polarswarm
```

All tests should pass before committing changes. There are no build scripts, Makefiles, or container definitions yet.

## Code Organization

```
cmd/polarswarm/main.go       # CLI entry point
internal/app/                # CLI commands and module composition
internal/config/             # Configuration loading and validation
internal/github/             # GitHub API abstractions + fake test client
internal/workflow/           # Orchestration state machine and label projections
internal/agent/              # Agent runner interface and mock runner
internal/store/              # Local cursors, leases, and idempotency records
```

Each `internal/` package contains a `doc.go` with a one-sentence ownership statement. Tests live in the same package (`*_test.go`).

## Code Style Guidelines

- Follow standard Go conventions (`gofmt`, idiomatic error handling).
- Keep package comments in `doc.go`.
- Prefix errors with the operation context: `fmt.Errorf("read issue %s#%d: %w", ...)`.
- Use table-driven tests with `t.Run` subtests.
- Config summaries must **never** leak secrets or token-like values.
- All GitHub-facing records refer to **roles** (e.g., `agent:developer`), never concrete backend tools or model providers.

## Testing Instructions

### Test Pyramid

1. **Unit tests** for pure logic (state machines, parsers, rendering, config loading).
2. **Fake client tests** for API-facing behavior (`internal/github/fake.go`).
3. **Dry-run integration tests** for full local plans.
4. **Optional live GitHub tests** against dedicated test resources only.

### Critical Rules

- **No test should write to GitHub by default.**
- Real write tests must be explicit opt-in, use disposable resources, print targets before writing, and be safe to rerun.
- Verify idempotency for every message and write operation.
- Test security gates before enabling any real agent execution.
- Use `t.TempDir()` for temporary test files.

### Running Tests

```sh
go test ./...
```

Current packages with tests: `internal/agent`, `internal/app`, `internal/config`, `internal/github`, `internal/workflow`.

## Security Considerations

- **Dry-run is the default.** The config parser defaults `dry_run` to `true`.
- **Secrets are never stored in versioned config.** API keys are referenced via environment variables (e.g., `${ANTHROPIC_API_KEY}`).
- **Access control** is defined in `.polarswarm/access.toml` (whitelist, blacklist, HMAC policy).
- **Unauthorized actors** must enter `status:pending-triage`, not be auto-processed.
- **Label mutations** by unauthorized actors are detected and corrected by reconciliation.
- **Lease revocation** prevents stale local processes from advancing workflows after remote state has changed.
- **Secret masking** is required before any secret appears in logs, audit output, or agent results.

## Development Conventions

### Commit Titles

Use area-prefixed imperative summaries:

```text
<area>: <imperative summary> (#<issue>)
```

Allowed areas: `github`, `workflow`, `agent`, `tui`, `config`, `security`, `compliance`, `release`, `docs`, `test`, `ci`, `build`.

### Documentation Source of Truth

Traceability flows one way:

```text
PRD -> Architecture -> Module Docs -> specs tasks -> implementation/tests
```

- `docs/product-requirements.md` defines users, goals, and acceptance.
- `docs/architecture.md` defines system structure and boundaries.
- `docs/modules/` defines per-module contracts.
- `specs/M{N}/T{NNN}-{SLUG}.md` are local-only task files (ignored by git).
- Implementation tasks must cite versioned docs, not the local draft `PolarSwarm.md`.

### Files Ignored by Git

The following are intentionally excluded from version control:

- `specs/` — local task specs
- `tasks/` — local planning tasks
- `PolarSwarm.md` — local planning draft
- `CLAUDE.md` — Claude collaboration notes
- `.polarswarm/` — local runtime state and config

## Configuration

PolarSwarm reads configuration from `.polarswarm/core.toml` (or an explicit `--config` path).

### Discovery Precedence (highest first)

1. `--config <path>` CLI flag
2. `POLARSWARM_*` environment variables
3. `<project>/.polarswarm/` or `<project>/polarswarm.toml`
4. `~/.polarswarm/`
5. `$XDG_CONFIG_HOME/polarswarm/`

### Minimal Schema (`core.toml`)

```toml
[github]
owner = "PolarKits"
repo  = "PolarSwarm"

[workflow]
target_label = "status:new"
dry_run      = true
```

Aliases accepted:
- `[github]` or `[repository]` with `owner` + `repo` / `name`
- `[workflow]` or `[runtime]` with `dry_run`

### Validation

- `owner` and `repo` are required.
- `dry_run` defaults to `true`.
- Unknown fields currently warn; they may become errors once a schema version is declared.

## Current Milestone (M1)

M1 proves a minimal IssueOps loop without real GitHub writes:

- Config loading with aliases and validation
- Fake GitHub client (`--fixture` JSON files)
- Issue reading with label and comment parsing
- Workflow state transitions (`new -> assigned -> in-progress -> review -> done`)
- Mock agent runner with deterministic results
- `polarswarm-agent-result` rendering plan
- Dry-run label projection plan
- Idempotency by `msg_id` and `op_id`

Do not expand scope into real backends, worktrees, PR creation, TUI, or security agents unless a task explicitly requires it.

## Agent PR Standard (Design Target)

Agent-created PRs must eventually include:

```markdown
## Summary
- ...

## Linked Issue
Closes #<issue_number>

## Agent Worklog
- orchestrator: ...
- developer: ...
- reviewer: ...

## Verification
- [ ] <command-or-check>

## Risk
- ...
```

During bootstrap, these rules are design targets, not mandatory for every commit.
