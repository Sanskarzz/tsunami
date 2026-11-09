# Tsunami implementation tracker

Updated: 2026-11-09. Current direction: [goal and scope](project-goal.md); task requirements and work order: [roadmap](roadmap.md).

**Position: the bounded five-minute product journey is deployed and qualified on the selected D9 owned-lab build. Present it and collect feedback. Keep execution-profile and customer-platform expansion deferred. This is an automated paced browser qualification, not production readiness or recorded human narration.**

Status meanings: **Implemented** means code exists with the stated lab checks; **Partial** means an existing foundation needs work for this delivery; **Open** means the proposed task has not been completed. None implies production readiness.

The demo completion commit packages D1–D9, documentation cleanup and the retained qualification records. Historical sections below describe their state at the time of each task. In particular, D2/D9 were built from uncommitted working trees; their recorded fingerprints and build identities remain authoritative for those deployments.

## Current product map

```text
Local UI / CLI
  -> Go control API and durable bbolt state
  -> approved engagement and child-run reservation
  -> Kubernetes Job / Mecatl worker
  -> run-authenticated request intent
  -> trusted scope-enforcing broker
  -> approved owned target
  -> broker-captured responses
  -> deterministic validator
  -> result, unrated finding candidate, evidence export, remediation
```

The API, broker, validator, and optional model gateway share one trusted process. Workers do not receive target credentials or unrestricted fetch/shell tools. Configured application assessments use scripted Mecatl; the demo does not need external inference.

| Area | Status and actual behavior | Reference |
|---|---|---|
| Application onboarding | Implemented for synthetic bearer/JSON integrations: saved form/questions, encrypted origin-bound access, explicitly approved four-read preflight, atomic registration plus unapproved draft. No general login/discovery. | [Guide](application-onboarding.md) |
| Scope and approval | Implemented immutable scope and separate approval/Start. D6 binds the UI review checkbox to the displayed digest without typing; CLI requires reviewed digest. Current approval API still accepts `{}`; D10 remains open. | [Engagements](engagements.md) |
| Execution | Implemented Jobs, distinct run credentials, no automatic replay, bounded native/fixed-curl paths. gVisor/Cilium require the dedicated lab and actual qualification. | [Sandbox](sandbox-lab.md), [adapter](tool-execution.md) |
| Broker | Implemented origin/scheme/port/address/account/path checks, redirect validation, numeric-address dialing, TLS verification, reservations, deadlines, evidence capture, and cancellation. | [Security](security.md) |
| Activity | D4 implements a bounded typed journal and tenant-authorized cursor reads. D5 presents actual activity/rejection, cursor gaps and stale/error states; source and disposable local browser checks pass. No raw body/URL/credential streaming or worker-authored event ingestion. | [Event contract](design.md#activity-events-d4) |
| BOLA | Implemented seven-read configured application procedure with identity/baseline/recheck validation. D1 preserves earlier supported exposure across repeated probes; source checks and D2's selected deployment qualification pass. | [Object profile](application-adapters.md) |
| Property exposure | Implemented known private-value leak, correct omission, ambiguous outcome, and earlier-exposure preservation. Separate owned fixture. | [Property profile](property-access.md) |
| Local UI | D6 Dashboard/onboarding/scope, D7 findings/evidence/remediation, D8 linked retest, D3 metadata and D5 actual activity/Stop are deployed and qualified in D9, with exact assets/runtime/evidence and a paced 300-second dry run. | [UI guide](local-ui.md), [D9 record](../examples/demo-journey/README.md) |
| Findings/evidence | D7 adds body-free candidate/capture metadata, validator-selected baseline/probe/recheck links, expected versus observed behavior and explicit single-artifact inspection/download. Existing export IDs/hashes/reviews are preserved; UI creates no human review/severity. | [Read contract](engagements.md#finding-and-evidence-reads-d7) |
| Stop | Implemented durable run/engagement/tenant/global revocation, cancellation and separate lifecycle observations. Selected gVisor/curl failure cases are recorded; broader partitions/load are unqualified. | [Lifecycle](stop-lifecycle.md) |
| CLI and preparation | Implemented operator journey, binding compiler, explicit fixture preparation, prerequisite checks, showcase and browser drivers. Keep as engineering/support tools. | [CLI](operator-cli.md), [showcase](showcase.md) |

The application procedure normally makes seven target reads within an eight-request allowance: two identities, two own-object baselines, peer probe, owner recheck, peer identity recheck. Six checks support one probe. The broker permits approved pairs within the budget; the displayed sequence is not an exact-once execution guarantee.

## Review findings and verification baseline

Source review baseline: commit `b0ede2a`, reviewed 2026-11-07. These observations do not qualify a later implementation build.

| Check | Observed result |
|---|---|
| `make verify` | Passed Go race tests, vet, 20 Python tests, and integrity checks for 32 bundles/240 artifacts plus historical hashes. Actual external toolkit-schema validation was not requested. |
| Frontend syntax | `node --check` passed for `app.js` and `onboarding.js`. |
| `make showcase-check` | Failed: VAmPI credentials expired or too near expiry. No fixture reset or deployment was performed. Refresh explicitly before qualification. |
| Temporary repeated-object-probe regression | At review time, reproduced a failure: four valid prerequisites -> protected object leaked to peer -> repeated probe denied -> valid rechecks -> final `no_issue_observed`. Temporary test removed; the subsequent D1 implementation below adds permanent coverage and fixes the source. |
| Fresh Kubernetes/browser campaign | Not rerun during this review. Existing records retain their original builds and conditions. |

The review's P0 defect was latest-probe selection erasing earlier established exposure. D1 addresses it in [object validation](../internal/platform/application.go) while retaining the ordered prerequisites/rechecks and all three verdict meanings.

### D1 implementation and source verification

Implemented on 2026-11-07 in the working tree based on `b0ede2a`; changes are not yet committed. [Permanent regressions](../internal/platform/application_test.go) first reproduced the failure on the old validator, then passed with the correction. D2 subsequently deployed and qualified this binary as recorded below.

- Every matching protected-object exposure under a deny rule is evaluated against its ordered evidence. A later denial/generic response cannot erase a supported finding.
- Incomplete earlier exposure keeps a later clean denial inconclusive. Earlier exposure references remain inspectable even when the latest response is already inconclusive; property validation retains that evidence too.
- Regressions cover repeated denial/generic/expired probes, late prerequisites, missing/expired/deleted rechecks, invalid proof/account/destination captures, intentional sharing, persisted results, and candidate/export evidence links. Existing fixed/property controls pass.
- Focused race tests and fresh `make verify` passed: all Go race tests, vet, 20 Python tests, 32 materialized bundles/240 artifacts and 20 historical trial hashes. No external toolkit-schema run was requested.
- The D1 source-fix step used no fixture refresh, deployment, browser/runtime campaign, customer target, or external inference; D2 separately performs owned deployment qualification. Archived exports and stored historical results were not migrated. Existing read-time progress/export revalidation can show the corrected interpretation of older affected captures; see [compatibility notes](application-adapters.md).

### D2 selected configuration and deployment qualification

Completed 2026-11-07 on the dedicated owned `kind-tsunami-lab` cluster. [The operating contract](demo-configuration.md) fixes the origins, procedures, runtime, limits, readiness window, UI entry point and recovery steps. [Retained fresh evidence](../examples/demo-configuration/README.md) identifies actual builds and outcomes separately from earlier campaigns.

- Initial preflight reproduced expired/near-expiry VAmPI access. Explicit `make showcase-prepare` built/deployed unique matching native/tools tags and rotated only disposable VAmPI/Stop/property fixtures. Existing UI/onboarding configuration remained enabled.
- Selected tags: `tsunami:showcase-20261107t093304z` and `tsunami-tools:showcase-20261107t093304z`. Both contain the same D1 binary; observed control/held/onboarded worker image identities match the [build record](../examples/demo-configuration/build.json). The dirty working tree is fingerprinted, not called a clean-commit build.
- Actual CRI runtime smoke checks and selected native IPv4/TCP/DNS boundary checks passed. All 54 curl direct-network attempts were denied, with reachable before/after controls and actual gVisor tools pods.
- Extended showcase passed all seven BOLA/property/Stop controls, with expected confirmed, no-issue, inconclusive and stopped outcomes, captured scope rejection, unrated findings, target-side cancellation and separate API cleanup.
- Fresh browser/onboarding qualification passed four real worker runs, approved four-read setup, actual onboarding worker runtime inspection, lost committed Start recovery and Stop during a held viewer export. All deployed UI assets matched local source bytes. This qualifies the existing UI, not D3-D9's proposed additions.
- Fresh `make verify` passed. The existing fixed gate retains 32 older bundles/240 artifacts; separate reused evidence validators checked the 11 fresh D2 bundles/82 artifacts in the new record. Exact assigned credential/key comparisons found no matches in the retained files/decoded artifacts; this is not general redaction.
- Original 334 run, 75 engagement and 9 setup IDs were retained; operator Secret, onboarding key, state volume and unrelated control configuration were preserved. Qualification ended with 345 runs, 86 engagements and 12 setups, all campaign work terminal, and **14 engagement slots remaining**. Plan capacity before D9/live presentation; never silently delete history or clear Stop gates.
- Final readiness passed for all seven selected plans, with credential expiry hints recorded at their timestamp. Access remains time-limited: recheck before presentation and after source/build/fixture changes. No customer targets, external model calls, production guarantees or new executor capabilities were introduced.

D2's qualification does not make credentials or build readiness permanent; it identifies the deployed D1 build, not later source changes.

### D3 authoritative metadata and presentation

Implemented and source-verified on 2026-11-07, uncommitted and not redeployed. [The API contract](engagements.md#read-only-assessment-metadata-d3) and [UI guide](local-ui.md) describe the additive response fields and their limits.

- Application-list items gain sibling metadata; engagement reads/approval gain a separate metadata projection. Public plans, scope digests, approval bodies, stored records and evidence-export shapes are unchanged.
- Prospective scope limits/procedure version/digest/execution settings come from backend policy. Run status separately projects its stored request allowance, actual deadline, state and procedure identity; it never substitutes current defaults for an older run.
- Scope permission includes durable gates/expiry, separately from approval-record state and latest reserved attempt outcome. Local access availability/configuration drift is separate from current credential health, which remains unknown. Known setup-permission expiry and historical preflight timestamps are explicitly named; no JWT/session validity is inferred.
- The UI removes hardcoded assessment allowances, labels registrations/approval/latest attempts separately, shows stored attempt policy, and marks failed refreshes stale. Executor-check attempts and actual worker/exit observations remain distinct. Missing/disabled metadata or unavailable access disables Start without disabling Stop; backend admission still decides.
- Focused HTTP/race tests verify policy projection, no target traffic/state mutation, unknown health/expiry, unavailable/changed bindings, unsupported definitions, tenant/run-token rejection, failed/completed attempts, durable gates, and failed executor checks without invented worker observations.
- Fresh `make verify` passes Go race tests, vet, 20 Python tests and the existing historical integrity gate. `make verify-ui-metadata` passes four frontend presentation tests, including nondefault limits, unknown metadata, failed-read staleness and independent Stop. These isolated fixtures never enter product data. JavaScript syntax and documentation checks pass.
- No cluster/fixture mutation, additional engagement slots, customer targets or external model calls were used. No new browser/runtime deployment qualification is claimed. The selected D2 build/evidence remain historical; matching rebuilt server/assets need qualification before presentation.

Request-level activity is not supplied by D3 metadata; D4 provides the separate journal below.

### D4 durable safe activity journal

Implemented and source-verified on 2026-11-07, uncommitted and not redeployed. The [event contract](design.md#activity-events-d4) specifies the schema, safe fields, limits and incomplete-history semantics.

- New runs receive a typed activity journal in their existing bbolt record. Trusted control-plane/controller/broker/validator call sites assign event kinds and sources; legacy free-text events remain compatible and are excluded from the feed. There is no new service, worker capability or arbitrary telemetry ingestion API.
- `GET /v1/runs/{id}/events?after=0&limit=50` requires tenant-authorized operator access. Monotonic per-run cursors, strict pagination validation, 100-event page ceiling, 256-event retention and a 2,048-byte event ceiling bound the feed. Responses identify pagination, retention gaps and whether tracking began at creation. Historical runs are never backfilled.
- Creation, dispatch intent, request admission, persisted capture/rejection, completed validation, durable revocation and lifecycle observations share their state transactions. Successful HTTP write observations are separately recorded as `request.sent`; they never mean target-side processing or successful response. The frozen private forwarding callback has no such observation and produces no invented send event.
- Events contain references, closed codes, timestamps, reservation/status counts and trusted validation summaries. They exclude account aliases, arbitrary URLs/errors, headers, credentials, raw bodies, model conversations and tool results. Raw evidence remains behind explicit existing authorized reads. New run snapshots may include the additive journal; historical stored evidence is unchanged.
- Local owned HTTP tests exercise BOLA/property leaks, fixed/filtering and ambiguous results, correlated admission/send/capture facts, scope rejection before DNS, redirect escape, unsafe DNS with no send, pagination, secret exclusion, tenant/run-token rejection, transaction rollback, retention under Stop, restart persistence and interrupted-operation uncertainty.
- Held-request tests cover run/engagement/tenant/global Stop, target-observed cancellation and separate local drain without invented validation/worker-exit proof. A blocked event-response consumer does not block Stop. Controller tests distinguish dispatch, worker/exit observations, failed checks, deletion acceptance and resource absence.
- Fresh `make verify` passes all Go race tests, vet, 20 Python tests and the existing 32-bundle/240-artifact historical gate. `make verify-ui-metadata` passes five tests, including the new explicit bounded activity client. JavaScript syntax, whitespace and documentation checks pass.
- The frontend has only the client helper for D5; no new activity polling or timeline panel is claimed. No cluster/fixture changes, engagement-slot consumption, external target/model requests or fresh deployment/browser qualification occurred. D2's recorded build remains unchanged.

The next section records D5's subsequent presentation and checks.

### D5 observed activity and independent Stop

Implemented and source-verified on 2026-11-07, uncommitted and not redeployed. See [presentation and concurrency behavior](design.md#activity-presentation-and-independent-stop-d5) and [local UI operation/testing](local-ui.md).

- **Observed run activity** renders D4's actual records with source, sequence, timestamps and request/evidence references. Reservations, observed writes, captures, rejected intents, validation and lifecycle transitions have distinct explanations. Displayed scope-rejection count and navigation derive from received records. Planned reads never become activity; bodies, credentials and arbitrary target URLs/errors are excluded.
- One shared in-flight refresh handles manual/polled observation. Status/feed errors remain separate; cursor validation, bounded pages/display, retention gaps and unknown historical tracking are explicit. Pause/navigation/selection/disconnect cancel viewer reads and reject obsolete responses without revoking work. Restart uncertainty is preserved; resource absence never becomes worker-exit proof.
- Stop stays visible across assessment/evidence screens, independent of pending Start/export/read actions. A separate receipt immediately distinguishes pending/acknowledged/unacknowledged requests without waiting for follow-up reads. Later durable-state recovery does not fabricate a lost acknowledgement. Mutations never retry automatically; old Start/status/export responses cannot clear the receipt. Lifecycle timestamps and uncertain-operation counts remain separately inspectable.
- `make verify-ui-activity` passes **18 tests** (five metadata/client regressions plus thirteen activity/concurrency/failure cases). Coverage includes cursor ordering/gaps/limits, incompatible pages, independent errors, slow reads, late responses, Stop during pending Start/export, lost Start/Stop recovery, no automatic replay, pause/navigation/disconnect and restart uncertainty. Test fixtures never enter product data.
- `make verify-ui-activity-browser` passes a **disposable local Chrome campaign** using the real control API/broker/validator and scripted Mecatl `LocalScheduler`: one vulnerable `confirmed`, one fixed `no_issue_observed` with lost Start recovery, and three active held-request Stops during pending export, pending Start response and lost Stop acknowledgement. Displayed event sequences match stored observations. Go-side assertions verify exactly five attempts, three target-side holds/cancellations, no unexpected target paths and no remaining active work. Browser checks include narrow layout, same-origin requests and no ordinary-DOM proof/token leakage.
- The local browser record is private under `artifacts/ui-activity/local-20261107T122920060Z/`, with summary, source asset hash and screenshots. It uses a temporary database and loopback targets, **not Kubernetes/gVisor/Cilium**. Local dispatch intent is not relabeled as isolated worker observation. This source check does not replace D9 or the selected deployment qualification.
- Fresh `make verify` passes Go race tests, vet, 20 Python tests and retained historical integrity checks. JavaScript syntax, whitespace and documentation checks pass. No production backend API/security-boundary change, cluster/fixture mutation, selected-history deletion, engagement-slot consumption, external target or model call was introduced. D2's build/records remain unchanged; matching deployment/readiness qualification is still required.

The next section records D6's subsequent implementation.

### D6 Dashboard, Add Application and readable Scope Review

Implemented and source-verified on 2026-11-08, uncommitted and not redeployed. The normal path requires no JSON editing or Kubernetes knowledge; it reuses the existing onboarding, approval and execution APIs.

- Dashboard shows registered integrations/applications and saved assessments, keeping approval state separate from the latest attempt's stored state/verdict. The additive engagement-list projection includes observation time and explicit `recorded` / `not_started` / `unavailable` attempt metadata. It reads only the last durably reserved child, validates its tenant/parent identity, never downloads capture bodies or revalidates historical results, and changes no stored scope/digest/export. Unknown metadata remains unknown; failed list refreshes retain a stale snapshot.
- Add Application labels reuse of a reviewed lab integration and its exact API/addresses/access rule. A frontend URL does not select a new API target. Checked questions ask only for missing facts; password/SSO remains unsupported. Protected access remains in the separate masked form with request bodies omitted from inspection. Setup readiness, expiry and the next permitted step are explicit.
- Preflight presents the exact four legitimate account/GET reads and limits as a readable table. It requires a separately reviewed revision/digest; changing setup revision/digest clears that checkbox. Old revisions are not promoted. Prepare still atomically creates a registration and unapproved draft; it does not approve or Start.
- Scope Review presents exact origin/addresses, account/action matrix, intended access rule, protected proof-field names, limits/rate/expiry, execution and exclusions/data handling. Technical scope/assertions/fingerprint and API/architecture inspection are collapsed. The review checkbox binds to the displayed fingerprint without typing; changed scope clears review. The existing empty-body approval API is unchanged. Approval and Start remain separate, with unsupported procedures disabled.
- `make verify-ui-onboarding` passes **23 frontend tests** (18 earlier regressions plus five Dashboard/scope/setup cases). New Go projection tests verify stored observations, missing/foreign child handling, tenant/run-token rejection, sensitive-data exclusion and no target traffic/state mutation. Fresh `make verify` passes Go race tests, vet, 20 Python tests and historical integrity checks.
- `make verify-ui-onboarding-browser` passes the real Add -> four-read validation -> unapproved draft -> explicit scope approval -> Start path with Chrome, Mecatl and the existing hash-checked fixed curl adapter. Go-side target counters establish **four legitimate reads and zero probes before assessment permission, then seven assessment reads and one probe**; one confirmed completed attempt is recorded. It also checks invalid operator credentials, unsupported password/SSO, missing API questions, stale grant rejection without traffic, protected-form omission, exact scope-table correspondence, text-safe injection, narrow layout and Dashboard approval/result separation.
- Latest private onboarding record: `artifacts/ui-onboarding/local-20261108T060035976Z/` (summary, source asset hash and screenshots). The five-case D5 browser regression also passes on this source at `artifacts/ui-activity/local-20261108T060035486Z/`, including active Stop and lost response cases.
- These are disposable loopback fixtures with temporary state. The opt-in onboarding launcher uses `bwrap` only to place the exact reviewed script at its fixed path in a test filesystem view; it installs nothing under host `/opt`, adds no configurable worker script path, and keeps Chrome's normal sandbox outside that view. This is **not Kubernetes/gVisor/Cilium qualification**. No selected-cluster change, credential reset, history deletion, engagement-slot consumption, customer target or external model call occurred. D2's recorded build remains unchanged and needs fresh deployment/readiness qualification for these changes.

The next section records D7's subsequent implementation.

### D7 findings, supporting evidence and remediation

Implemented and source-verified on 2026-11-09, uncommitted and not redeployed. The [read contract](engagements.md#finding-and-evidence-reads-d7) and [UI guide](local-ui.md) describe the additive routes and explicit inspection boundary.

- Tenant-authorized `GET /v1/engagements/{id}/findings` returns body-free candidate, assessment/check and capture metadata. It reuses deterministic validation, candidate grouping/digests, exact artifact serialization and matching existing review verification. It sends no target/model requests and writes no state. Original stored verdict and current read-time interpretation are separate; stopped/failed/unfinished work produces no completed finding.
- The Evidence screen shows failure-specific titles, expected versus observed access, owner baseline/peer probe comparison, identities/rechecks, missing or unresolved evidence, and procedure-specific remediation. D1's earlier supported exposure stays linked even after a later denial. No-issue/inconclusive outcomes remain distinct; confirmed observation does not invent human review or severity.
- Raw bytes require an explicit `GET /v1/engagements/{id}/evidence/{E-sha256}` inspection. Browser hash/provenance checks precede display, and downloads preserve the original bytes used by full exports. Normal navigation/refresh transfers no raw bodies/full exports; API inspection summarizes raw responses. Stop remains independent and cancels obsolete viewer reads. Hash agreement detects byte changes, not a compromised evidence authority.
- Backend regressions cover both profiles, wrong owner/proof, expired identity, missing rechecks, unresolved/repeated exposure, stopped work, strict tenant/parent/procedure bindings, rejected-argument redaction, existing review provenance, artifact/export identity, and reads without state mutation or target traffic. `make verify-ui-findings` passes **30 frontend tests**, including tampered bytes, stale/late responses, safe text rendering, Stop and disconnect.
- `make verify-ui-findings-browser` passes five real scripted Mecatl controls: object leak/denial and property leak/filter/ambiguity. Go counters establish **35 target reads, five probes and no unexpected paths**. Chrome checks metadata-only navigation, selected owner/probe inspection, original-byte downloads, export ID/hash correspondence, no invented review and narrow layout. Private record: `artifacts/ui-findings/local-20261109T130742215Z/`.
- Fresh `make verify` passes all Go race tests, vet, 20 Python tests and the historical 32-bundle/240-artifact/hash gate. D5's active-Stop browser regression passes at `artifacts/ui-activity/local-20261109T130742166Z/`; D6's real onboarding/fixed-curl browser regression passes at `artifacts/ui-onboarding/local-20261109T130751934Z/`.
- These campaigns use disposable loopback fixtures and temporary state, **not Kubernetes/gVisor/Cilium qualification**. No selected deployment, fixture credentials, history, capacity or export format changed; no customer target or external model was used. Existing candidate titles/digests and historical evidence retain their identity; the failure-specific display title is a separate projection.

The next section records D8's subsequent implementation.

### D8 linked fixed-app retest with fresh approval

Implemented and source-verified on 2026-11-09, uncommitted and not redeployed. See the [D8 read/create contract](engagements.md#linked-fixed-app-retest-d8) and [UI guide](local-ui.md).

- Existing configured-application draft creation accepts optional immutable `source_engagement_id`. Tenant/source validation and insertion share one transaction. The new scope binds that reference and the explicitly selected application's own origin/access/procedure/executor settings; it inherits no approval, child runs or credentials. Ordinary/older records omit the field and retain their scope digests. New native views/exports carry the additive link; original candidate/artifact identities and toolkit projections remain intact.
- `GET /v1/engagements/{retest-id}/retest` returns one body-free read snapshot of both scopes, changed public bindings, conservative procedure compatibility, latest reserved attempt states, original stored verdicts, current deterministic results only for completed work, and exact selected artifact IDs. Reads send no target/model traffic, write no state and create no review. Stopped/expired/incomplete source history grants no permission. Different profiles/rules/protected-field/denial semantics are labeled incompatible; no patch or application-wide assurance is inferred.
- Evidence offers explicit destination/name/authorization/executor selection, creates only an unapproved draft, and opens changed-scope review with a fresh unchecked checkbox. The client waits for a valid comparison before linked approval; the backend approval body remains unchanged until D10. Approval and Start stay separate. Source/retest evidence navigation and Dashboard recovery preserve separate IDs/results/approvals; lost draft responses never replay automatically. Stale/late reads and Stop remain independent.
- Backend race regressions use distinct owned origins and **different credential sets**, proving no credential inheritance, no draft/read traffic, separate approval, confirmed/no-issue outcomes, exact evidence/export correspondence, original-history preservation and restart persistence. Negative checks cover cross-tenant/unknown links and rollback, worker/unauthed reads, changed-source digest rejection, stopped/expired sources, incompatible profiles/rules, missing rechecks, stopped retests and a failed latest attempt after an earlier clean result.
- `make verify-ui-retest` passes **36 frontend tests**. `make verify-ui-retest-browser` passes a real Chrome/native Mecatl journey across two owned loopback origins: confirmed source -> revoked source permission -> linked changed-origin draft -> fresh approval/Start -> `no_issue_observed` -> preserved original evidence -> incompatible unexecuted draft -> Dashboard recovery. Go observes **14 reads/two probes and no unexpected paths**. Private record: `artifacts/ui-retest/local-20261109T141621012Z/`.
- Fresh `make verify` passes all Go race tests, vet, 20 Python tests and the historical 32-bundle/240-artifact/hash gate. D7's five-control browser regression passes at `artifacts/ui-findings/local-20261109T141735935Z/`; D5's active-Stop regression passes at `artifacts/ui-activity/local-20261109T141736151Z/`; D6's real onboarding/fixed-curl regression passes at `artifacts/ui-onboarding/local-20261109T141736541Z/`.
- These are disposable local source/UI checks, **not Kubernetes/gVisor/Cilium or selected-deployment qualification**. No selected cluster, credential reset, history deletion, capacity consumption, customer target, external model or new executor capability was involved. D2's recorded deployment is unchanged; D9 still requires a matching build, current readiness/runtime/browser qualification and a timed rehearsal.

The next section records D9's subsequent deployment and qualification.

### D9 selected deployment, complete browser journey and timed rehearsal

Completed for the recorded build on 2026-11-09 and included in the demo completion commit. Qualification ran against an uncommitted working-tree build, as its retained provenance records state. Start with [the retained D9 dossier](../examples/demo-journey/README.md), [actual browser timings/results](../examples/demo-journey/browser/summary.json), [build identities](../examples/demo-journey/build.json) and [the exact presenter script](demo-walkthrough.md#five-minute-presentation).

- Initial read-only readiness found expired VAmPI credentials. Explicit owned preparation rebuilt/deployed matching native/tools images and rotated disposable VAmPI/Stop/property fixture data/access. It preserved all original IDs, the operator Secret, onboarding key, state PVC and unrelated control environment; no Stop gate or history was cleared. Old scopes were never silently rebound.
- Selected tags: `tsunami:showcase-20261109t144510z` and `tsunami-tools:showcase-20261109t144510z`. Both inspected unstarted images contain binary SHA-256 `013faa595eac50ccd2fa45e0e87285c5e4fce84314cf29224ee68d91066c0c1b`, matching the CLI. Four served UI assets matched local bytes. The actual onboarded worker's CRI image/runtime matched the selected tools image and `runsc` / `io.containerd.runsc.v1`. This is a fingerprinted working-tree build, not a clean-commit reproduction.
- Fresh native runtime/TCP/UDP DNS boundary checks passed. All **54** selected curl literal-IP/`--resolve`/`--connect-to` attempts were denied across startup/settled/repeat phases, with reachable before/after controls and broker HTTP. These diagnostic pods use no assessment slots and establish only their recorded selected paths.
- `make demo-rehearse` exercised Dashboard -> Add Application -> approved four-read preflight -> unapproved draft -> exact matrix/review -> approval -> actual Mecatl activity -> supported BOLA finding -> selected owner/probe artifacts/download hashes -> remediation -> separate active-request Stop -> fresh linked fixed-app retest. Activity sequences matched persisted records and contained a real scope rejection. Raw reads/exports were explicit. A hidden committed Start response recovered its existing run without replay; the original finding/artifacts survived retest unchanged. Default candidates stayed unrated and unreviewed.
- The six scheduled core stages finished within their deadlines; the paced dry run elapsed **300.000 seconds**. This was automated browser operation with the supplied narration script, **not recorded human narration**. Property leak/filter/ambiguity were separately assessed after the five-minute presentation and returned `confirmed` / `no_issue_observed` / `inconclusive`. Actual target counters prove seven reads/one probe per property control.
- During the held request and pending viewer export, Stop remained usable. The fixture independently recorded active -> cancelled, unchanged five-request count and zero probe completions; revocation, local drain and API cleanup were separately observed. A Stop receipt does not establish remote rollback, a production latency bound or worker exit during partitions.
- Final preserved history: **351 runs, 92 engagements and 13 setups**, including all original 345/86/12 IDs. Exactly six assessments and one setup were added; no run remained nonterminal. **Eight engagement slots remained** at final readiness. Plan capacity before a live three-assessment presentation or another six-assessment qualification; never silently delete history. Credential/JWT hints remain time-limited and require a fresh pre-presentation check.
- Fresh verification covers Go race tests, vet, **23 Python tests**, **36 frontend tests**, six browser assessments and retained integrity for **38 bundles/286 artifacts plus 20 historical trial hashes**. D9 adds a read-only record checker and corruption regressions for late rehearsals, changed builds, foreign links and discarded history. Exact assigned credential/key bytes and base64 forms were checked in memory against retained files and decoded artifacts without matches; this is not general redaction. Existing D2/historical records remain unchanged. External toolkit schemas were not requested.
- Private full campaign: `artifacts/d9/20261109T143951Z/` and `artifacts/ui/browser-20261109T144950912Z/`. A reviewed selection of source/build/readiness/preservation, six materialized bundles, safe journals, exact linked comparison and six screenshots is retained in `examples/demo-journey/`, with checksums. No customer target, external inference, new executor or broader platform capability was introduced.

Delivery stop rule: **present D1–D9 and collect feedback**. D10/D11 remain optional hardening. Customer identity, secret/evidence lifecycle, independent procedure review, wider isolation/failure/scale qualification and an external emergency cutoff remain separate release work.

## Active implementation queue

D1/D3–D8 are source-verified and deployed/qualified in D9's selected journey; D5–D8 retain their separate local browser checks. D2 remains an older recorded build. D9 is complete for its recorded build; D10/D11 remain **Open** optional hardening. Full task contracts are in [the roadmap](roadmap.md#ordered-implementation-tasks).

| ID | Priority | Deliverable | Depends on |
|---|---|---|---|
| D1 | P0 | **Implemented, source-verified and deployed in D2/D9:** earlier exposure preserved; permanent regression coverage added. Included in demo completion commit. | Existing validator |
| D2 | P0 | **Complete for recorded build:** owned configuration frozen, fixtures explicitly refreshed, source/runtime/network/showcase/browser/readiness checks passed. | D1; recheck time-limited readiness |
| D3 | P0 | **Implemented, source-verified and deployed in D9:** authoritative limits, procedure/execution/access metadata and UI presentation; no target traffic from metadata reads. Included in demo completion commit. | Existing APIs; matching server/assets |
| D4 | P0 | **Implemented, source-verified and deployed in D9:** bounded typed activity, tenant-authorized cursor reads and API client support. Included in demo completion commit. | Existing lifecycle/broker |
| D5 | P0 | **Implemented, source/local-browser verified and deployed in D9:** actual activity/rejection, safe cursor polling, independent Stop receipts and recovery. Included in demo completion commit. | D4 |
| D6 | P0 | **Implemented, source/local-browser verified and deployed in D9:** Dashboard/history, readable onboarding/scope, separate review/approval and real Add-to-Start path. Included in demo completion commit. | D3 |
| D7 | P0 | **Implemented, source/local-browser verified and deployed in D9:** metadata-only findings, exact validator evidence links, explicit raw inspection/download and remediation. Included in demo completion commit. | D1, existing evidence |
| D8 | P0 | **Implemented, source/local-browser verified and deployed in D9:** immutable tenant-checked source link, changed-scope review, fresh approval/Start and actual outcome/evidence comparison. Included in demo completion commit. | Existing scope APIs; D7 presentation |
| D9 | P0 | **Complete for recorded build:** matching deployment/runtime/network, six real browser controls, retained provenance and a paced 300-second dry run. Included in demo completion commit. | D1-D8; time-limited readiness |
| D10 | P1 | Require reviewed scope digest at the approval API. | Existing approval; coordinate clients |
| D11 | P1 | Persist Start idempotency and recover the same attempt after response loss. | Existing atomic child creation |

Next: **present the qualified slice and collect feedback**. D10/D11 remain optional hardening, not automatic follow-on platform work.

## Demo acceptance checklist

This gate is satisfied by the recorded D9 selected-deployment campaign and paced browser rehearsal, with the stated limits. Readiness/build/capacity must be checked again for a later presentation; source/local checks alone do not qualify a changed deployment.

- [x] Repeated-probe regression fixed without weakening prerequisites or uncertainty; selected D1 build deployed and qualified in D2.
- [x] Add owned application -> approved preflight -> exact final scope -> explicit approval/Start works in the connected UI.
- [x] Actual isolated worker execution is observed; dispatch intent is not labeled process startup.
- [x] Sanitized live activity and an actual broker scope rejection are visible without loading raw bodies.
- [x] BOLA finding links legitimate baselines, peer exposure and follow-up checks; remediation is clear and severity remains unrated.
- [x] Separate property assessment distinguishes leak/filter/ambiguity without implying a combined application or broad coverage.
- [x] Stop during an active held request and a pending viewer read shows revocation, drain/uncertainty, execution and cleanup separately.
- [x] Fixed-app retest uses fresh approved scope, returns `no_issue_observed`, and preserves the original finding/evidence.
- [x] Lost responses recover durable IDs without automatic replay; no fabricated frontend state, review, metric or outcome.
- [x] Fresh source/browser/runtime checks and a timed rehearsal identify the actual build and credentials/readiness window.

## Retained evidence and frozen capabilities

Keep historical qualification intact: [onboarding](../examples/onboarding/README.md), [local UI](../examples/local-ui/README.md), [showcase](../examples/showcase/README.md), [property controls](../examples/property-access/README.md), [Stop](../examples/stop-lifecycle/README.md), and [validation history](validation-results.md). Passing archived hashes checks integrity, not current deployment health.

Existing [private connector](private-connectivity.md), [connector TLS](connector-tls.md), [model gateway](model-path.md), [inference diagnostics](inference-diagnostics.md), [synthetic catalog](assessment-catalog.md), and [toolkit mappings](toolkit-mapping.md) remain documented and tested. Freeze their expansion; they are not blockers for this demo. Preserve the CLI and fixed adapter while avoiding new transport wrappers as substitute product progress.

## Deferred work

P2: VM/MCP, network/mobile/cloud/crypto executors, scanners/arbitrary shell, specialist coordination, generic discovery/authentication, live-model application planning, private-connector expansion, toolkit-wide coverage, customer portal/billing, multi-replica storage, distributed budgets, and scale engineering.

Before real customer use, independently review procedures and qualify tenant isolation, identity/roles, secret/evidence protection and lifecycle, remaining TLS/key management, data/model permissions, recovery and relevant failure behavior. These remain release requirements, not requirements to expand the local demo. No production Stop-time, hostile-process containment, or capacity claim is established.

## Updating this tracker

Mark a task done only when its roadmap definition of done has evidence. Record the commit/build and exact checks; distinguish code existence, source checks, retained campaigns and fresh deployment qualification. Keep current work here and task contracts in the roadmap. Do not append another competing stage plan.
