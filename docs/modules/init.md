# Init Module

## Responsibility

Prepare a repository for PolarSwarm through explicit user action. Initialization is setup-time behavior, separate from normal runtime reconciliation.

## Non-Responsibilities

- Do not execute agents.
- Do not process workflow Issues.
- Do not silently escalate runtime permissions.
- Do not overwrite user-owned repository resources without ownership proof or explicit confirmation.
- Do not make optional GitHub capabilities mandatory.

## Inputs

- Target repository owner/name or discovered git remote.
- Init token with setup permissions.
- Capability probe results from the GitHub module.
- Desired setup profile, when supported.
- Existing repository files, Labels, templates, workflows, and settings.

## Outputs

- Created, updated, skipped, and degraded resource report.
- Standard Label set or degraded Label plan.
- Issue templates for supported issue types.
- Lightweight GitHub Actions workflow files for guard/notify behavior when permitted.
- `.polarswarm/` configuration templates, including `core.toml` and `access.toml`.
- `.gitignore` entry for the configured worktree base directory.
- Capability cache updates and initialization diagnostics.

## Idempotency

`polarswarm init` must be safe to rerun:

- Existing matching resources are reported as skipped.
- Product-owned resources may be updated to the expected shape.
- Unknown or user-owned resources are not replaced without confirmation.
- Partial success is allowed when degraded behavior is documented.
- The final report must be sufficient for `doctor` to explain remaining setup gaps.

## Permission Boundary

Init may require broader permissions than runtime. Runtime code must not perform init-only operations. If runtime detects missing setup, it should report the issue and suggest `polarswarm init` or manual steps.

## Milestone Task Entry Points

- M1: no init requirement.
- M2/M3: optional local config template tasks only when explicitly scoped.
- M4: baseline init for Labels, templates, `.polarswarm/` config, and `.gitignore`.
- Later: optional Projects V2, Issue Types, branch protection, rulesets, and write probes.
