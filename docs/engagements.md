# Engagement foundation

An engagement groups an approved target and procedure selection, child runs, evidence and checklist progress. This is the first step toward using more of the toolkit workflow. The later [application adapter](application-adapters.md) adds registered owned applications and reviewed exports. Arbitrary customer URLs, shell commands, scanners, MCP tools and a multi-agent scheduler are still not enabled.

## Authority and lifecycle

An authenticated tenant operator creates a draft with a name, authorization reference, lab scenario, connectivity and selected procedure IDs. The service selects the configured target, resolves the procedure versions and fixes their digests. A separate operator approval pins the scope digest before any child run can be created. A reference is an operator attestation, not automated verification of a legal authorization document.

The digest covers tenant, target, connectivity, scenario, procedure versions, expiry and limits. It is a consistency check in the trusted database, not a signature against a compromised server. The current plan is immutable. A new scope needs a new engagement.

The API does not accept caller-selected tenant, target URL, `authorized: true`, limits, capture IDs or completed percentages. Run tokens cannot access these operator routes. Reads and exports check the authenticated tenant; knowing another engagement ID is insufficient.

Current fixed limits are 20 child runs, two concurrent children, one active run per procedure, a one-hour approval window, and 100 retained engagements per demo tenant. Run deadlines are clipped to engagement expiry. Failed runs keep their reservations. These are lab admission limits, not customer-wide monetary budgets or demonstrated capacity.

Child creation and its engagement reservation commit in one bbolt transaction. Engagement Stop marks the parent stopped and all nonterminal children stopped in one transaction, then cancels their tracked HTTP/model contexts. New children are refused afterward, including after restart. Policy/version drift may block execution/export, but it must not prevent Stop. Existing per-run and tenant/global stops remain in place. Tenant/global gates also deny new engagement children.

The current process still handles the broker, store and controller together. These transactions do not establish distributed revocation across future executors. The existing connector expiry/cutoff limits still apply; Stop cannot undo a target's accepted action.

## Progress and findings

Progress is derived from linked runs and trusted validation of broker captures. An agent cannot upload a finding or mark a checklist item complete. The latest attempt determines procedure progress. Failed/stopped work and missing prerequisites remain separate from completed work. Skipping requires an operator reason and is allowed only before the procedure has been attempted; it does not count as completed.

Completed means the selected procedure captured its legitimate baselines and probe. The probe can still be inconclusive, for example the intentional missing-object control. Completion never means the whole application passed a security assessment. Only the selected Tsunami procedures or configured application plan contribute to this progress; importing 26 checklist files does not execute their tests.

The export recomputes finding candidates using the synthetic validator. Repeated confirmed observations of one pinned procedure are grouped and retain their evidence links. A later clean run does not silently erase an earlier candidate. Candidates remain `needs_human_review`, with no invented severity or CVSS score. The subsequent application milestone adds an explicit review API and finding/evidence schema export. Individual reviewer identity, remediation workflows and customer impact assessment still need further work.

## Evidence and toolkit compatibility

An export contains the engagement view, a toolkit-shaped engagement projection, finding candidates, a report draft and capture artifacts. Each artifact includes its run/engagement/procedure provenance. SHA-256 names its exact serialized bytes; base64 transport preserves those bytes when JSON is reformatted. Artifact paths are generated from hashes. They never instruct the server to read arbitrary files.

Hashes are not signatures, immutable storage or proof against a compromised broker. Current evidence includes synthetic response bodies; general customer redaction, storage authorization, encryption, retention and deletion remain product work. Provider/operator/run credentials are omitted from public run representations and exports.

The optional local checker validates the engagement projection against the friend's actual schema, checks artifact hashes and candidate evidence references, and records the schema hash and validator version. It does not execute toolkit code. Remote schema references are refused. The subsequent application milestone extends this checker to finding/evidence contracts, review consistency and materialized files. Legacy records here retain their original format and qualification claims. See [jsonschema validation behavior](https://python-jsonschema.readthedocs.io/en/stable/validate/).

## Routes

All routes require the authenticated tenant operator, never a worker token.

| Route | Operation |
|---|---|
| `POST /v1/engagements` | Create a draft for configured lab targets |
| `GET /v1/engagements` | List this tenant's engagements |
| `GET /v1/engagements/{id}` | Scope, run links and derived progress |
| `POST /v1/engagements/{id}/approve` | Approve the pinned scope; body `{}` |
| `POST /v1/engagements/{id}/runs` | Reserve/create an approved procedure run |
| `POST /v1/engagements/{id}/skip` | Record a reason for an unattempted procedure |
| `POST /v1/engagements/{id}/stop` | Revoke the engagement and active children; body `{}` |
| `GET /v1/engagements/{id}/findings` | Body-free finding, validator-check and capture metadata |
| `GET /v1/engagements/{id}/retest` | Read-only linked source/retest scope and outcome comparison |
| `GET /v1/engagements/{id}/evidence/{E-sha256}` | Explicit read of one exact export artifact |
| `GET /v1/engagements/{id}/export` | Tenant-authorized snapshot and artifact bytes |
| `POST /v1/engagements/{id}/reviews` | Later review API; stopped engagement and matching evidence required |
| `GET /v1/engagements/{id}/findings.csv` | Reviewed findings with CSV output guards |

Run creation defaults to scripted. Live engagement runs require the exact `fixture` or `openrouter-free` provider policy; the latter additionally requires the existing free price policy. Private model execution is still disabled. Existing lab run routes remain for regression tests and are outside the engagement allowance; that allowance is not an account-wide budget.

## Read-only assessment metadata (D3)

D3 adds projections to existing operator responses, without changing stored plans, scope digests, approval bodies, execution authority or export snapshots:

| Response | Added metadata |
|---|---|
| `GET /v1/applications` | Each existing plan-shaped array item gains a sibling `metadata`: registration source, observation time, supported procedure ID/version/digest, planned reads/request allowance/interval, prospective run-duration ceiling, execution mode and local access availability. The metadata is not part of `application.Plan` or its digest. |
| `GET /v1/engagements/{id}` and successful approval | Sibling `metadata` contains observed scope permission, executor/execution mode, prospective duration ceiling, pinned procedure metadata, local access availability and the latest reserved attempt's ID/state/verdict/deadline. |
| `GET /v1/runs/{id}/status` | Adds `state`, `created_at`, `target_request_limit`, procedure ID/version/digest and `observed_at`. Existing `expires_at` is the stored actual run deadline. `executor_observation_attempted_at` plus `observation_code` describes the last executor-check attempt; existing worker/exit timestamps remain actual observations. |

An engagement's observed permission may be `unapproved`, `active`, `revoked`, `expired`, `disabled` by a durable tenant/global gate, or `unavailable` for inconsistent scope. A completed/failed child does not change an active approval record into a completed engagement. `latest_attempt` follows durable reservation order, not random run-ID sorting. Metadata is an observation and never replaces admission checks or grants execution.

For supported application plans, the prospective request allowance and duration come from the same backend policy used for new runs. Duration is capped by engagement expiry; it is not a promised actual deadline before Start. Existing run status reports its stored allowance/deadline even when they differ from the current prospective defaults. Unsupported stored procedures retain their pinned identifiers/digests but do not acquire invented current allowances.

Access `availability` is `configured`, `unavailable`, or `configuration_changed` relative to the selected saved plan. It describes locally loadable credential/configuration bindings, not authentication success. `credential_health` remains `unknown`; these reads perform no DNS lookup, target connection, login, baseline validation or model call. They do not decode JWT claims into verified expiry. Known onboarding timestamps are separately named `setup_permission_expires_at` and `baselines_validated_at`: setup permission expiry and historical four-read validation are not current credential health.

Application arrays retain their existing shape with additive fields. Go clients decoding `application.Plan` ignore the sibling metadata; clients needing it can decode `ApplicationSummary`. Normal engagement reads/approval populate `EngagementView.Metadata`, while store snapshots and evidence exports leave it omitted, preserving the existing export contract. Unknown metadata remains unknown in the UI; use matching server/frontend assets after rebuilding.

Focused checks: `go test -race ./internal/platform -run 'TestMetadata|TestOnboardedMetadata'` and `make verify-ui-metadata` (Node). The latter exercises actual frontend presentation code with isolated fixtures; those fixtures never appear in the running product.

## Dashboard list metadata (D6)

`GET /v1/engagements` preserves its array of scope-shaped objects and adds `observed_at`, `latest_attempt_status` and optional `latest_attempt`. These are read-only response projections, not part of stored engagements, approval digests or exports. Existing clients decoding `Engagement` ignore the additions.

`latest_attempt_status` is `not_started` when no child is reserved, `recorded` when the last reserved child's identity/tenant/parent matches, or `unavailable` when that child cannot be projected safely. `latest_attempt` uses D3's ID, stored state, creation/deadline timestamps and optional **stored** verdict. Reservation order comes from the engagement's durable child list; it is not inferred from random run IDs. Missing/foreign/invalid child metadata does not become “not started” or a clean result.

The list reads metadata only from the child JSON, excluding captures, result reasons, legacy event text and model call contents. It does not contact targets, validate credentials, revalidate historical findings, mutate scopes or close an active approval because an attempt completed. The Dashboard labels approval separately from the latest attempt and retains an explicitly stale list if refreshing fails. Focused checks: `go test -race ./internal/platform -run TestDashboard` and `make verify-ui-onboarding`.

D6's UI approval checkbox binds to the displayed immutable scope digest without manual typing; changed scope clears review. The current approval endpoint still accepts `{}`. Requiring an expected digest at the API remains D10, not an implemented server-side guarantee.

## Durable activity reads (D4)

`GET /v1/runs/{id}/events?after=0&limit=50` returns a tenant-authorized, bounded metadata page from the run's typed activity journal. It exposes broker admission/send/capture/rejection, trusted validation and durable lifecycle transitions without raw bodies, credentials, account aliases or arbitrary URL/error text. It performs no target or executor probe. See the [complete schema, cursor/retention contract and observation limits](design.md#activity-events-d4).

The journal is additive to new run records and raw run snapshots; it is separate from legacy free-text `events` and D3's ephemeral metadata. Older runs are not backfilled. An admitted reservation without a send/completion observation remains uncertain. Reading or pausing observation never revokes permission; existing Stop endpoints remain authoritative.

Focused checks: `go test -race ./internal/platform -run TestActivity` and `make verify-ui-activity`. D5 connects the journal to the UI with independent Stop receipts and bounded cursor polling. The [UI qualification guide](local-ui.md#qualification) separates the optional disposable local browser test from Kubernetes deployment qualification.

## Finding and evidence reads (D7)

`GET /v1/engagements/{id}/findings` returns a `schema_version: 1` snapshot with `engagement_id`, `observed_at`, `scope_digest`, `candidates`, `assessments` and `captures`. This is metadata for normal finding navigation, separate from the existing full inline export. It internally reads stored evidence to validate/hash it, but returns no capture bodies, raw rejected URLs/accounts, arbitrary capture errors or review narratives. It makes no target/model request and writes no state.

- **Candidates:** existing stable ID, candidate digest, original title, separate failure-specific `display_title`, procedure/target, status, trusted reason, deterministic remediation, contributing run IDs and exact export artifact IDs. Grouping and candidate digests use the same helpers as exports. Only completed, deterministically confirmed observations contribute candidates; a later clean attempt never erases an earlier supported observation.
- **Assessments:** each bound run's state, pinned procedure/digest, original `stored_verdict` when present, and a current validation `result` only for completed work. Current validation uses the same logic as export; it does not rewrite historical results. Expected rule, approved account aliases/object path/protected field names and remediation come from the saved application plan. No-issue and inconclusive assessments use a neutral title. Stopped/failed/unfinished attempts have no completed result or candidate.
- **Checks:** configured application profiles map the validator's selected captures to `owner_identity`, `peer_identity`, `owner_baseline`, `peer_baseline`, `peer_probe`, `owner_recheck` and `peer_identity_recheck`. Each has approved account/path, a closed outcome and optional capture/artifact references. Missing, unverified and unevaluated checks remain explicit. Repeated probes retain the actual selected earlier exposure; unresolved earlier exposure can add an `earlier_unresolved_probe` reference without confirmation. Catalog procedures do not acquire invented application roles.
- **Captures:** run/capture/artifact IDs, exact SHA-256/artifact path, timestamps, HTTP status and a closed outcome. Account/path are resolved only for nonblocked exact reviewed application pairs. Blocked arguments are never echoed. A capture's presence is not a confirmed finding.
- **Existing review:** only a review matching the current candidate digest is projected, using the export's existing consistency checks. Decision, purpose, actor, revision, timestamp and actual rating when present are metadata; impact/rationale are omitted. Default candidates remain unrated and need human review. An `automated_lab_test` review is explicitly not human signoff. Reads do not create reviews.

`GET /v1/engagements/{id}/evidence/{E-sha256}` returns one existing `EvidenceArtifact`: its ID, run/capture references, hash/path and `data_base64` of the exact original serialized artifact bytes. It never replays a request. IDs and bytes match the full export, including its engagement/run/procedure provenance. The UI fetches this only after **Inspect**, verifies SHA-256 and metadata/provenance correspondence before display, and downloads those same bytes. Hash matching checks byte integrity, not independent authenticity or redaction. Raw artifacts/full exports can contain sensitive synthetic bodies; ordinary metadata navigation does not fetch them. Full exports remain a separate explicit action and retain their existing schema.

Both routes require the tenant operator and check parent, child-run, tenant, executor, target and pinned-procedure bindings as existing exports do. Worker tokens/unauthed requests are rejected; foreign/unknown engagements or artifacts return 404. Evidence cannot be fetched through another engagement, even within one tenant. Only GET is accepted (otherwise 405), query parameters are rejected (400), and responses use `Cache-Control: no-store`. Inconsistent pinned definitions/bindings leave the read unavailable rather than reinterpreting evidence under a different procedure. Stop remains separately available even if these reads fail.

The UI performs one metadata read on Evidence navigation or explicit refresh, not full-export polling. Navigation, selection, Stop and disconnect cancel obsolete finding/raw reads; late responses cannot populate a new context or clear a Stop receipt. A failed refresh leaves the prior metadata explicitly stale. Protected body bytes appear only in the dedicated explicit viewer, never in ordinary activity or the API inspector.

Checks: `go test -race ./internal/platform -run 'TestFinding'`, `make verify-ui-findings` and the optional `make verify-ui-findings-browser`. See [local qualification boundaries](local-ui.md#qualification).

## Linked fixed-app retest (D8)

Create a retest through the existing `POST /v1/engagements` configured-application API, adding optional `source_engagement_id`. The operator still explicitly selects `application`, `name`, `authorization_ref` and optional reviewed `executor`. The service resolves that registered application and its own access binding. It does not inherit the source's target, credentials, execution settings, approval or child runs.

The source reference is checked against the authenticated tenant in the same bbolt transaction that inserts the new draft. Unknown/foreign/malformed references are rejected without leaving a draft. Both sides must be supported configured-application scopes; catalog-only sources are not supported. Stopped, expired, incomplete and unapproved source assessments can supply history without granting authority. The new assessment always starts as `draft`, with its own ID/expiry/scope digest, empty run list and no approval. Existing approval and Start endpoints remain separate and enforce their normal gates/current configuration. Stop on either assessment affects only that assessment's children; tenant/global gates still apply to both.

`source_engagement_id` is immutable and included in the new scope digest. Changing it invalidates that scope. The field is omitted for ordinary/older assessments, preserving their existing serialized shape and scope digests. New retest native views/exports carry the additive reference; existing candidate IDs, artifact bytes and toolkit projections remain unchanged. There is no history migration, workflow engine, credential copying, automatic replay or reverse-link mutation on the original assessment.

`GET /v1/engagements/{retest-id}/retest` returns `schema_version: 1`, `observed_at`, `source`, `retest`, `compatibility`, `reason` and `differences`. It reads both scopes and their latest durably reserved attempts in one read transaction, validates scope/procedure and child tenant/parent/target/executor bindings, and returns no capture bodies. It sends no target/model traffic and creates no review or stored result.

- Each side includes its saved public engagement, supported procedure ID/profile version/definition digest and `attempt_status: not_started|recorded`. A recorded `latest_attempt` includes run ID/state, optional original `stored_verdict`, current trusted `result` only for a completed attempt, and exact selected export artifact IDs. Pending/running/failed/stopped attempts have no completed result. Missing/corrupt linked state makes the comparison unavailable, never clean. The last reserved attempt is used even if an earlier attempt was clean; original candidates remain inspectable through D7.
- `compatibility: comparable` means the supported profile, intended access rule, ownership pointer, protected proof-field names and denial policy match. Application IDs, origins, record/account bindings and definition digests can differ. This is a comparison of configured checks, not independent verification of a deployed code fix or equivalent application coverage. Any mismatch in those semantic criteria is `incompatible`; outcomes remain individually visible without inferring remediation success.
- `differences` lists changed application/origin/addresses, profile/rule, ownership pointer, reviewed reads/assertions, denial policy, rate, credential binding digest (never credentials), definition digest, executor, authorization reference, expiry, child/concurrency allowances and scope digest. Values are serialized public-scope values rendered as text. The new full scope remains authoritative for approval.

The comparison route requires tenant operator authentication; worker/unauthed access is refused. Unknown/foreign assessments or absent/foreign source links return 404. Only GET is accepted (otherwise 405); queries are rejected (400), and responses use `Cache-Control: no-store`. Invalid stored scope/bindings return an unavailable comparison; existing Stop remains separately available.

In the UI, Evidence offers an explicit destination selector and new authorization reference. Saving opens Scope Review with changed bindings shown and a fresh unchecked approval checkbox. Linked approval waits for a valid current-scope comparison in this client; backend approval still accepts `{}` until D10. Approval never starts work. On Evidence entry/manual refresh the comparison shows actual outcomes, versions and evidence references; source/retest evidence buttons navigate to their own D7 views. No raw body/export is automatically fetched or polled. Dashboard retains the source link for recovery. Failed reads leave an explicitly stale comparison; obsolete reads are cancelled on navigation/selection/Stop/disconnect, and Stop does not wait for them. Lost draft responses require read-only history recovery, not automatic mutation retries.

Checks: `go test -race ./internal/platform -run TestRetest`, `make verify-ui-retest` (36 frontend tests including D3–D7 regressions) and optional `make verify-ui-retest-browser`. The latter uses two owned loopback origins and real native Mecatl, with fourteen target reads/two probes, a separate unexecuted incompatible draft, and original artifact/candidate preservation. It does not qualify Kubernetes isolation or a selected deployment.

## Reproduce

```sh
make engagement-fixture-local
make toolkit-inventory TOOLKIT_PATH=../vapt-toolkit

# Use the actual campaign directory printed by the first command.
python3 scripts/toolkit.py validate-export \
  --toolkit ../vapt-toolkit \
  --bundle artifacts/engagements/offline/campaign-EXAMPLE/vulnerable/bundle.json \
  --out artifacts/engagements/schema-validation.json

export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
make private-deploy CONTEXT=kind-tsunami-lab
make engagement-verify CONTEXT=kind-tsunami-lab
make engagement-private-verify CONTEXT=kind-tsunami-lab
```

Local execution uses real Mecatl/Responses code with loopback fixtures; it does not establish process isolation. Kubernetes verification uses gVisor worker Jobs; private verification uses the mTLS connector. Both Kubernetes commands select scripted execution even if a free model is configured. No external model calls are needed.

Each campaign preserves a fresh output directory. The local demo exports before Stop; the Kubernetes collector retrieves the later stopped snapshot. Their authorization/status fields reflect those different snapshot times. The verifier checks draft/after-Stop dispatch refusal, all five procedures on vulnerable/fixed apps, progress, and three versus zero candidates. Active-request Stop, racing creation, tenant isolation and restart behavior are covered by native tests. Neither campaign measures production scale or independent discovery.

See [the expanded product plan](product-roadmap.md) for the remaining work and required evidence.
