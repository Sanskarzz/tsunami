# Five-minute demo implementation roadmap

Updated: 2026-11-09. Scope authority: [project goal](project-goal.md). Live completion status: [tracker](track.md). Architecture, screens, and activity contract: [design](design.md).

**Finish the existing HTTP authorization journey before building more platform capabilities.** This roadmap replaces the earlier stage 8-15 expansion plan. D1–D9 now have the scoped evidence recorded in [the tracker](track.md) and [D9 qualification](../examples/demo-journey/README.md). Present this slice and collect feedback; D10/D11 remain optional hardening, with platform expansion deferred.

## Delivery scope

Dashboard -> Add Application -> Scope Review -> Approve -> Assessment -> Finding -> Evidence -> Remediation -> fresh approved Retest. Stop remains independently available throughout active work and gets a separate held-request demonstration.

Use owned VAmPI for BOLA, the existing separate property fixture for field authorization, the worker's fixed invalid intent for scope rejection, and the owned held-request fixture for Stop. Keep scripted execution explicit and external model calls disabled. Do not build a general multi-procedure planner to join these targets.

P0 is required for this delivery; P1 significantly improves it; P2 is deferred. Each task is one independently committable unit with focused checks. Dependencies describe useful delivery order, not permission to leave commits unverified.

## Ordered implementation tasks

### D1 — Preserve earlier object exposure (P0)

- **User-visible outcome:** a later denied read cannot erase an earlier established leak within the same run.
- **Backend:** evaluate earlier matching object probes with ordered prerequisites/rechecks, following the existing property-exposure approach. Preserve evidence references; incomplete earlier exposure must not become a clean result merely because a later response is denied.
- **Frontend:** no change required.
- **Tests:** commit the reproduced leak -> denial regression; cover leak -> generic response, incomplete earlier prerequisites, missing rechecks, expired identity, and the normal fixed case. Preserve historical results rather than silently rewriting archived exports.
- **Security invariants:** known protected proof and valid prerequisites are mandatory; no status-only or model-driven confirmation. Keep all three verdict meanings.
- **Definition of done:** regression passes, earlier supported exposure survives, ambiguous evidence remains inconclusive, existing object/property controls and source gate pass.

### D2 — Freeze and qualify the demo configuration (P0)

- **User-visible outcome:** the owned demo starts reliably with valid disposable access and a known execution boundary.
- **Backend:** reuse existing preparation/check/qualification tooling. Identify selected origins, procedures, images, runtime and credential-readiness window; explicitly refresh disposable fixtures when required. No new provisioning framework.
- **Frontend:** document the single running UI entry point and supported integration selection.
- **Tests:** after D1, run readiness and selected runtime/network/assessment checks on the selected build. Record observations separately from retained results.
- **Security invariants:** preparation must be explicit, require idle owned work, preserve history and unrelated configuration, and never clear Stop gates or widen scope to get a pass.
- **Definition of done:** selected build and owned credentials pass preflight and relevant qualification; setup/recovery instructions are reproducible. Recheck readiness immediately before D9/presentation.

### D3 — Expose authoritative assessment metadata (P0)

- **User-visible outcome:** scope/status screens show real allowances, procedure/execution settings, and honest access availability.
- **Backend:** extend existing metadata projections with request allowance, deadline, procedure version/digest, execution mode, executor, and availability/expiry observations where actually known. An expiry hint or startup route configuration is not a live identity check.
- **Frontend:** replace hardcoded limit text. Distinguish application registration, approved permission, latest attempt outcome, and stale observations.
- **Tests:** projections match stored policy; wrong tenant/run credentials are refused; no secrets or response bodies appear; unknown credential health remains unknown.
- **Security invariants:** metadata creates no permission or new network validation; tenant comes from authentication.
- **Definition of done:** displayed allowances come from backend records, unavailable/unknown facts are explicit, and metadata reads cause no target traffic.

### D4 — Persist safe activity events (P0)

- **User-visible outcome:** actual requests, scope rejection, validation and revocation are observable without sensitive-body streaming.
- **Backend:** extend the small existing event mechanism with typed bounded records, monotonic per-run sequence, and tenant-authorized cursor reads. Use the [event contract](design.md#activity-events-d4); avoid a message bus or separate tracing service.
- **Frontend:** add only the API client support needed for the next task; do not invent a timeline before events exist.
- **Tests:** ordering/cursor bounds, restart persistence, tenant isolation, denial before dial, credential/raw-body exclusion, incomplete operations and Stop races.
- **Security invariants:** trusted components assign facts. Worker telemetry cannot claim enforcement/validation; arbitrary URLs, errors, prompts and tool results are not event text. Persist durable-transition events atomically where applicable.
- **Definition of done:** a real run yields a bounded metadata timeline; missing historical events stay missing; request admission is not mislabeled target receipt.

### D5 — Connect actual activity and independent Stop (P0)

- **User-visible outcome:** assessment detail shows observed activity/scope rejection and can revoke active work without waiting for a viewer request.
- **Backend:** reuse D4 events and existing lifecycle status; preserve uncertainty and observation timestamps.
- **Frontend:** poll with cursors and no overlapping loops. Keep Stop independent of Start/export/observation, show stale/error states, and separate permission, activity, execution and cleanup.
- **Tests:** real active held-request Stop, Stop during pending export/Start, unacknowledged Stop, lost response recovery, navigation/disconnect and restart uncertainty.
- **Security invariants:** pause/close never means Stop; deletion acceptance/object absence never means process exit. Local drain is not target-observed cancellation.
- **Definition of done:** actual broker rejection and active revocation appear in the UI; missing acknowledgements remain uncertain and Stop is usable while reads are pending.

### D6 — Simplify Dashboard, Add Application and Scope Review (P0)

- **User-visible outcome:** an operator adds the owned integration and approves understandable permissions without JSON editing or Kubernetes knowledge.
- **Backend:** reuse onboarding/preflight/engagement APIs and D3 metadata; avoid a second onboarding or orchestration model.
- **Frontend:** show real application/history rows, readiness, exact account/action matrix, targets/addresses, access rule, limits, expiry and unsupported actions. Use progressive disclosure; collapse API/raw-scope details. Label reused lab integration and checked-rule setup honestly.
- **Tests:** connected browser journey, unsupported authentication, changed/stale setup, invalid token, scope rendering/injection, no target traffic before permission, narrow layout.
- **Security invariants:** preflight and assessment permissions remain separate; protected tokens never enter conversation/explorer/storage; registration alone is not authorization.
- **Definition of done:** Add -> approved four-read validation -> saved draft -> reviewed approval -> Start works against the owned application; every visible state maps to stored or explicitly derived facts.

### D7 — Connect findings, evidence and remediation (P0)

- **User-visible outcome:** a finding explains who accessed what, expected versus observed behavior, supporting evidence, and correction.
- **Backend:** provide a metadata-only candidate projection without requiring full inline-export transfer for normal listing. Reuse validator/evidence/export semantics and deterministic remediation.
- **Frontend:** use a failure-specific finding title; link candidate evidence to owner/probe/recheck captures; show comparison and remediation. Load raw evidence explicitly; keep hashes/raw scope in optional inspection.
- **Tests:** wrong owner/proof, expired identity, missing evidence/rechecks, malicious display text, candidate-to-artifact correspondence and both supported profiles.
- **Security invariants:** no raw-body polling, invented impact/severity, automatic human review, or model confirmation. Hashes identify bytes, not independent authenticity.
- **Definition of done:** reviewer can follow each conclusion to exact captured support; unknown/missing data remains visible and downloads preserve artifact integrity.

### D8 — Link a fresh fixed-app retest (P0)

- **User-visible outcome:** retest compares a fresh fixed-app result while preserving the original finding and evidence.
- **Backend:** add a minimal tenant-checked source-assessment reference and create new scope/runs through existing APIs. No new workflow engine or credential inheritance.
- **Frontend:** select the fixed registered application, show scope differences, approve/Start, and compare procedure version, actual outcomes and evidence. Label incompatible comparisons.
- **Tests:** cross-tenant link rejection, changed-target approval, stopped/expired source handling, incompatible procedures, original-history preservation and fixed result.
- **Security invariants:** never copy credentials to another origin, reuse old approval for a changed target, overwrite evidence, or call the application secure. Different bindings have different digests even when procedure semantics match.
- **Definition of done:** vulnerable `confirmed` and fresh fixed `no_issue_observed` are linked with separate approvals/runs/evidence; incomplete retests remain inconclusive or unfinished.

### D9 — Qualify and rehearse the complete product journey (P0)

- **User-visible outcome:** one five-minute presentation proves the implemented journey on an identified build.
- **Backend:** extend existing browser/showcase tooling only for new behavior; retain exact build/asset/run IDs and target observations.
- **Frontend:** exercise Dashboard/onboarding, approval, actual activity, finding/evidence/remediation, independent Stop and linked retest. Update the presenter guide to describe only landed behavior.
- **Tests:** fresh source gate, browser journey, BOLA vulnerable/fixed, property leak/filter/ambiguity, actual scope rejection, active Stop, pending viewer read, lost Start response, and relevant runtime/network checks. Check readiness again after any fixture refresh.
- **Security invariants:** owned synthetic targets only, no external model calls, no fabricated display/outcome, no automatic uncertain replay, failed records retained. Preparation is outside the presentation and explicitly described.
- **Definition of done:** [tracker acceptance checklist](track.md#demo-acceptance-checklist) is satisfied with fresh evidence and a timed rehearsal. Retained fallback evidence is labeled; late results stay pending. Demo completion is separate from customer readiness.

## Optional hardening

### D10 — Bind approval to the reviewed digest (P1)

- **User-visible outcome:** backend approval identifies the exact displayed scope the operator reviewed.
- **Backend:** require expected scope digest in approval; coordinate the API compatibility change across CLI, UI, drivers and documentation. Current `{}` contract remains actual behavior until this lands.
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

Land **D1 -> D2 -> D3 -> D4 -> D5 -> D6 -> D7 -> D8 -> D9**. D10/D11 are separate improvements, not prerequisites for adding more features. Keep [track.md](track.md) updated with commits and actual validation; writing this plan does not complete a task.

Once D9 meets the demo gate, present it and collect feedback. Do not automatically begin the old execution-profile stages. New work must improve a demonstrated problem and name its permissions, evidence and acceptance criteria.

## Deferred scope (P2)

Freeze expansion of VM/MCP, network/mobile/cloud/crypto, scanners/general shell, specialist agents, live-model application planning, generic discovery/login, private connectors, toolkit-wide coverage and transport adapters. Keep existing working paths/tests/guides; no runtime deletion is authorized by this document.

Customer login/billing, multi-replica storage, distributed budgets, and capacity engineering are also outside the local demo. Before a customer pilot, independently qualify procedures, tenant isolation, identity/roles, secrets/evidence/redaction/retention, TLS/key lifecycle, data/model handling, backup/recovery, emergency controls, and relevant failure behavior. No production guarantee follows from this demo.

## Documentation and evidence policy

Use three planning sources: goal for scope, tracker for status, roadmap for tasks. [Design](design.md) supplies screen/state/event details; operating guides describe current commands. Historical implementation evidence lives in [validation results](validation-results.md) and the linked `examples/` records.

Remove fictional disconnected UI previews and duplicate future-stage plans from the development entry path. Preserve security contracts, failure records, build identities, evidence exports and reproducibility guides, including frozen capabilities. Update stale links and claims when scope changes; never rewrite historical failures into successes.
