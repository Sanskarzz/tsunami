# Remaining work and stop rule

Updated: 2026-11-09. Scope: [project goal](project-goal.md). Delivery status: [tracker](track.md). Operating guide: [controlled demo](demo-walkthrough.md).

## Ordered implementation tasks

D1–D9 are delivered for the recorded owned-lab build. Their independently committable contracts and definitions of done remain in the [dated delivery plan](archive/demo-delivery-plan-2026-11.md#ordered-implementation-tasks); exact qualification is in [D9 evidence](../examples/demo-journey/README.md). Do not restart the old sequence or treat historical results as current readiness.

The stabilization work separates documentation cleanup from the tenant/global Stop cancellation fix. After relevant source/browser/runtime checks and a timed rehearsal, present the slice and collect feedback. No feature expansion is required for that presentation.

## Optional hardening

### D10 — Bind approval to the reviewed digest (P1, implemented)

- **User-visible outcome:** backend approval identifies the exact displayed scope the operator reviewed.
- **Backend:** require expected scope digest in approval; coordinate the API compatibility change across CLI, UI, drivers and documentation. Implemented source contract: approval requires `scope_digest`; missing or mismatched values fail without granting permission.
- **Frontend:** submit the digest bound to D6's displayed-scope review checkbox; retain clear review and explicit approval without manual retyping.
- **Tests:** missing/wrong/stale digest, cross-tenant approval, unapproved Start rejection, updated clients.
- **Security invariants:** approval cannot modify or widen scope; agent/run credentials cannot approve.
- **Definition of done:** direct API clients receive the same reviewed-scope protection and all supported clients use the new contract.

### D11 — Make Start idempotent (P1)

- **User-visible outcome:** a lost Start response can recover the same attempt safely.
- **Backend:** persist an engagement-bound idempotency key and request fingerprint in the same transaction as child creation. A mismatched request cannot reuse the key.
- **Frontend:** reuse the key only to recover the same action; keep explicit new-attempt behavior and uncertain-state presentation.
- **Tests:** duplicate/concurrent Start, lost response, conflicting body, tenant binding, Stop race, and restart.
- **Security invariants:** recovery cannot create replacement work, refund budgets, replay target operations or restore revoked permission. After Stop it may identify the original run, never reactivate it.
- **Definition of done:** identical repeated Start identifies the original run; conflicting requests fail and revoked execution stays closed.

## Work order and stop rule

1. Keep the selected build, credentials, capacity and evidence checks reproducible. Preserve failed records and acknowledge unavailable observations.
2. Present the bounded journey to reviewers. Choose D10, then D11 if the review justifies API hardening; they are not required permission to begin more platform work.
3. Treat suggested [Mecatl harness reuse](mecatl-harness.md) as evaluated options, not an implementation queue. Keep scope, Stop, credentials and verdict authority outside the agent.

## Deferred scope (P2)

Freeze VM/MCP, network/mobile/cloud, scanners/general shell, specialist agents, live-model application planning, generic discovery/login, private connectors, toolkit-wide coverage and new transport adapters. Existing working paths/tests remain as references; no runtime deletion is authorized by this roadmap.

Customer login/billing, multi-replica storage, distributed budgets and capacity engineering are outside this local demo. A customer pilot additionally needs independent procedure review, tenant/identity isolation, secrets/evidence/redaction/retention, TLS/key lifecycle, data/model handling, backup/recovery, emergency controls and relevant failure qualification. The lab establishes no production guarantee.

## Documentation and evidence policy

Use goal for scope, tracker for status and this roadmap for open tasks. [Design](design.md) records states/events; the [demo guide](demo-walkthrough.md) is the sole presentation runbook. [Historical/reference index](archive/README.md) keeps older decisions, limitations and experiments reachable. Preserve artifact bytes, failure records, provenance and build identities; do not delete evidence to simplify navigation.
