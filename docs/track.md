# Tsunami implementation status

Updated: 2026-11-09. Scope: [goal](project-goal.md). Work order: [roadmap](roadmap.md). Operation: [demo guide](demo-walkthrough.md).

## Current product map

UI/CLI → Go API and durable approval/Stop → Kubernetes Job with scripted Mecatl → trusted scope-enforcing HTTP broker → owned target → captured artifacts → deterministic validator → findings/evidence/remediation and linked retest. The UI reads real state. Configured applications make no external model calls.

| Delivery | Status and evidence |
|---|---|
| D1–D2 | Exposure-preservation regression and owned configuration delivered; [D2 build record](../examples/demo-configuration/README.md). |
| D3–D6 | Authoritative metadata, durable safe activity, independent Stop and connected onboarding/scope UI implemented and deployed in D9. |
| D7–D8 | Validator-linked findings/raw inspection/remediation and fresh-scope fixed-app retest implemented and deployed in D9. |
| D9 | [Recorded build qualified](../examples/demo-journey/README.md): runtime/network controls, six real browser assessments and a paced 300-second automated rehearsal. This does not qualify a later build or current credentials. |
| Stop reliability | Tenant/global Stop now cancels already-revoked assessment contexts before auxiliary setup persistence. Fault tests cover held-request cancellation, preserved failure reporting, admission denial and tenant boundaries. See [Stop contract](stop-lifecycle.md). |
| D10 / D11 | **Open**, optional P1: approval API reviewed-digest binding; server-side Start idempotency. |

Detailed task chronology, build fingerprints, negative results and acceptance evidence are preserved in the [dated delivery log](archive/demo-delivery-log-2026-11.md). The original D2/D9 uncommitted build provenance remains authoritative for those historical images; do not relabel it as a clean-HEAD build.

## Demo acceptance checklist

D9 satisfied the recorded-build gate: owned onboarding, separately approved preflight/scope, observed isolated worker, actual request/rejection activity, evidence-backed object finding, separate property leak/filter/ambiguity controls, active-request Stop and newly approved fixed-app `no_issue_observed`. See the [original checklist](archive/demo-delivery-log-2026-11.md#demo-acceptance-checklist) for evidence and limits.

For each later build, run source/local browser checks, inspect matching deployed source/runtime/network identities and rehearse the selected journey again. Keep actual results/build IDs in a dated qualification record. Do not reinterpret a core-only rehearsal as the full property campaign.

## Stabilization verification

The [dated Stop-patch record](archive/stabilization-2026-11-09.md) records all source/frontend/local browser checks, actual runtime/native/curl network checks and a 300.000881-second core browser rehearsal on the newer build. Actual outcomes were `confirmed`, active-request `stopped`, and linked fixed-app `no_issue_observed`. Local property controls passed separately. The original 351 runs / 92 engagements / 13 setups remain present; five engagement slots remain and no run is pending/running at the recorded final check. Credentials were refreshed and remain time-limited. The full six-assessment Kubernetes/property campaign was not repeated; D9 retains its own historical build record.

## Known limits and next action

- Readiness is temporary: VAmPI credentials last one hour; check expiry, idle work and remaining engagement capacity before presentation. Refresh only disposable fixture data; preserve keys, history, approvals and Stop gates.
- Approval accepts `{}` for the saved immutable scope. The UI binds review to a fingerprint; the API does not yet require an expected digest.
- Start is not server-idempotent. Recover stored child IDs after a lost response; never automatically retry uncertain work.
- Normal Stop needs the trusted API/store. Revocation is not proof of remote undo, worker exit or cleanup. An onboarding persistence failure remains a reported partial failure even when assessment traffic has been cancelled.
- Shared trusted process/storage, broad lab service egress, raw synthetic evidence and limited failure/isolation qualification do not establish production tenant isolation or customer readiness.

Present the narrow slice and collect feedback. Use [remaining work](roadmap.md) for explicit choices; do not automatically resume platform expansion. Historical and frozen experiments stay in the [reference index](archive/README.md), with original evidence bytes/paths preserved.
