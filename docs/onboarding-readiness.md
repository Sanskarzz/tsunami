# Onboarding readiness boundaries

Updated: 2026-11-07. Implemented contract and usage: [application onboarding](application-onboarding.md). Current development scope: [project goal](project-goal.md); planned UI simplification: [D6](roadmap.md#d6--simplify-dashboard-add-application-and-scope-review-p0).

The existing form-first setup uses checked rules, durable typed questions, protected origin-bound bearer access, explicitly approved four-read validation, and atomic supported-application/unapproved-draft registration. It is not a live-model setup planner. Reuse it for the demo rather than building a second setup system.

## Required boundaries

1. Saving details makes no target request. Unknown backend/API facts remain unknown; ask only for missing facts.
2. A supplied/discovered host is a proposal, never implicit scope. Preserve approved origin/address/path validation; unsupported login or response contracts stay unavailable.
3. Credentials use protected fields, not conversation, URLs, ordinary activity or reports. Changing access/scope invalidates relevant readiness observations.
4. Baseline validation needs its own explicit revision/digest-bound grant: four legitimate reads, bounded attempts/time and no cross-user probe.
5. Readiness requires observed distinct identities, legitimate objects and an explicit intended access rule. A generic HTTP 200 or model statement is insufficient.
6. Registration prepares an unapproved engagement. Final assessment approval and Start are separate actions; do not collapse them into a readiness label.
7. Lost responses recover stored state without automatic replay. Stop applies to preflight and linked assessment; restart uncertainty remains visible.

The new demo UI should explain these facts with readable scope and progressive disclosure. Reusing an existing owned integration is explicit. A checkbox/reference records operator authorization context; it does not automatically verify legal ownership.

## Deferred extensions

Password/cookie/SSO adapters, documentation ingestion, frontend/API discovery, live-model planning, private customer onboarding and arbitrary application contracts are outside this delivery. They need separate permissions, data handling, enforcement and qualification if later selected. Do not collect real customer access or contact an external dev application to make the local demo appear complete.
