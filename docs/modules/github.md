# GitHub Module

## Responsibility

Provide GitHub-facing interfaces for Issues, Comments, Labels, Pull Requests, repository metadata, and capability checks. M1 should use fake or read-only clients and produce dry-run write plans by default.

## Non-Responsibilities

- Do not execute workflow state transitions.
- Do not run agents.
- Do not assume optional GitHub features without capability checks.
- Do not perform live writes unless a task explicitly opts in.

## Inputs

- Repository identity from config or CLI.
- Runtime token or fake client configuration.
- Issue number, label names, comment IDs, and operation IDs.

## Outputs

- Normalized Issue, Comment, Label, and PR snapshots.
- Dry-run Comment and Label update plans.
- Capability status such as `native`, `degraded`, or `unknown`.
- Actionable permission or rate-limit errors.

## M1/M2 Task Entry Points

- M1: fake Issue reader and dry-run write plan renderer.
- M1: idempotent operation identifiers for planned writes.
- M2: no required work except supporting agent result write plans if needed.
