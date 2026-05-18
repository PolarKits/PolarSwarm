# GitHub Capabilities

PolarSwarm must treat GitHub platform capabilities as runtime-dependent.

## Source Constraint

Any claim about GitHub APIs, plans, permissions, limits, or event semantics must be backed by `docs.github.com` or by a runtime capability probe. Blog posts, community discussions, and third-party articles can be used as leads only.

If official documentation is silent or ambiguous, the feature must be marked `unknown` or `degraded` until a safe probe or real business operation proves availability. Product documentation must not make MVP behavior depend on unverified GitHub capabilities.

## Default MVP Capabilities

MVP may rely on:

- Issues
- Issue Comments
- Labels
- Pull Requests
- Milestones
- Commit statuses or Actions job status as fallback status surfaces
- GitHub Actions as lightweight guards and notifications
- Repository contents writes for configuration and workflow files, when explicitly initialized

## Optional Capabilities

The following must be capability-probed or manually configured:

- Sub-Issues
- Issue Dependencies
- Issue Types
- Projects V2
- CODEOWNERS required review
- Rulesets
- Checks API write
- Environment required reviewers
- Merge Queue
- Webhook-based wake-up

Optional capabilities must not be required for the first local loop.

## API Preference

| Area | Preferred access | Notes |
|---|---|---|
| Issues, Labels, Comments, Pull Requests, Milestones | REST | Mature baseline. All calls must handle pagination and secondary rate limits. |
| Sub-Issues and Issue Dependencies | REST when documented | Probe before use; fall back to body lists, labels, or comments. |
| Issue Types | REST when organization permission exists | Organization-level feature; degraded mode is `type:*` Labels. |
| Projects V2 | GraphQL first, REST only for documented subresources | Projects are optional visualization and should not be workflow authority. |
| Checks API write | GitHub App preferred | Fine-grained PAT behavior must be probed before enabling. |
| Merge Queue | Branch protection or ruleset capability | Use only where official docs and repository settings prove availability. |

## Capability Cache

Capability state has three values:

- `unknown`: not probed yet.
- `native`: native API is available.
- `degraded`: native API is unavailable or permission is insufficient.

Read probes may run automatically. Write probes must be explicit and must use test resources.

The cache is part of the GitHub client behavior, not a separate static feature profile. It should be persisted locally and periodically revalidated so account plan or permission changes can recover from degraded mode.

Design requirements:

- Do not infer capability solely from account plan names.
- Prefer safe read probes from official endpoints.
- Treat write probes as side-effecting unless explicitly proven otherwise.
- Store degraded decisions so the runtime avoids repeatedly calling known-unavailable paths.
- Revalidate degraded entries after a configured interval.
- Record capability state changes for diagnostics and observability.

## Probe Behavior

| Stage | Trigger | Behavior |
|---|---|---|
| First read probe | Cache miss | Call safe read endpoint or GraphQL query; store `native` on success or `degraded` on permission/not-found response where appropriate. |
| Cached call | Cache hit | Use cached path and avoid repeating known failing calls. |
| Revalidation | Stale cache entry | Re-run safe probe; upgrade to `native` if the repository or token gained access. |
| Explicit write probe | User runs a write-probe doctor command | Use disposable test resources only; write an audit record and clean up when possible. |

Write probes must never mutate business Issues, production branch rules, merge queues, or real PR state.

## Degradation Rules

| Feature | Degraded Behavior |
|---|---|
| Issue Types | Use `type:*` labels |
| Sub-Issues | Use Markdown lists or comment links |
| Issue Dependencies | Use dependency labels or comments |
| Projects V2 | Skip project integration |
| Checks API write | Use commit statuses or Actions state |
| Merge Queue | Use manual merge plus required checks, or local post-merge check |
| Branch Protection unavailable | Use local validation and visible warning |

## Plan and Permission Expectations

GitHub feature availability differs by repository visibility, account type, organization policy, and token permissions. PolarSwarm therefore documents minimum expectations but relies on probes for the actual runtime decision.

Baseline expectations:

- Public repositories can usually use the core Issue, Label, Comment, Pull Request, and Actions surface.
- Private repositories may require paid plans for branch protection, required checks, required reviews, CODEOWNERS enforcement, or environment review gates.
- Merge Queue is not a baseline requirement and must be treated as optional.
- Organization-level features such as Issue Types require organization permissions and may be unavailable to a repository-scoped token.
- Projects V2 is a visualization enhancement and must not be required for orchestration correctness.

## Initialization Behavior

`polarswarm init` should apply native setup when capability and permissions allow it, and emit an explicit degraded report otherwise.

Expected setup decisions:

| Resource | Native setup | Degraded setup |
|---|---|---|
| Labels | Create or update standard Labels | Initialization fails only if Labels cannot be managed at all. |
| Issue Types | Create or update organization Issue Types | Use `type:*` Labels and templates. |
| Issue Templates | Write YAML templates | Report missing contents permission. |
| Workflows | Write triage, Label guard, and notify workflows | Report manual setup steps if contents or actions permission is missing. |
| Projects V2 | Create project and fields when permitted | Skip integration and continue with Issues and Labels. |
| Branch protection and required checks | Configure when permitted and requested | Warn and rely on local validation plus visible status projection. |

Initialization must be idempotent. Existing resources should be updated only when the product owns the expected shape or the user explicitly confirms replacement.

## Doctor Behavior

`polarswarm doctor github:permissions` and `polarswarm doctor capabilities` provide the user-facing capability report.

The report should distinguish:

- `native`: feature is available and configured or ready.
- `degraded`: feature is unavailable or permission is insufficient, with the chosen fallback.
- `unknown`: feature has not been probed or requires explicit write probing.
- `misconfigured`: feature is available but repository settings are inconsistent with PolarSwarm expectations.

`doctor capabilities --probe` may run safe read probes. `doctor capabilities --write-probe` must be opt-in and use disposable resources.

## Rate and Polling Constraints

The local runtime polls GitHub, so API use must be budgeted.

- Use ETag or conditional requests for repeated reads where supported.
- Keep a local cursor for repositories, Issues, and Comments.
- Apply exponential backoff on API failure.
- Reduce polling for completed, abandoned, blocked, or actively leased Issues when safe.
- Prefer batched GraphQL reads only when it reduces request count and remains within documented behavior.
- Surface rate-limit pressure in `doctor` or runtime diagnostics.

## Capability Events

Capability transitions should be observable. When a feature changes state, PolarSwarm should record:

- capability key
- old state and new state
- probe or operation that caused the change
- GitHub response class, without leaking secrets
- degraded behavior selected
- timestamp

This event trail helps explain why a repository runs in reduced mode after initialization or plan changes.
