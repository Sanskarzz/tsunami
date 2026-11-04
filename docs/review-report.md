# Tsunami architecture and implementation review

Reviewed: 2026-11-04. Target: the current working tree, including untracked implementation files. Reviewer used the requested `review-agent` workflow without delegating. Production code, deployments, original assignment answers, and the two reference repositories were left unchanged.

## Actionable findings

These are defects in the reviewed implementation. The architectural recommendations below are separate; documented future capabilities are not being reported as regressions. All five findings are P2. No P0/P1 defect was established within the reviewed lab scope; this is not an isolation certification.

### [P2] Reject redirects in the Go operator client — internal/demo/demo.go:80

`Operator.Call` attaches the tenant operator bearer token but constructs a client without `CheckRedirect`. A redirect to a different port on the same hostname retains that Authorization header. An isolated transport reproduction redirected `operator.example:8080` to `operator.example:9090` and received the token at the second endpoint. This affects the Go verification/collection clients; the Python engagement collector already refuses redirects. The current control API does not deliberately emit redirects, so this requires a redirected or misrouted operator connection, rather than an ordinary target response. Refuse redirects before forwarding any credential and use an explicit transport that ignores ambient proxies for the loopback collector. Add the equivalent regression test for the Go client.

### [P2] Retry rejected cleanup instead of recording success — internal/kube/scheduler.go:68

The common Kubernetes response helper treats every `409 Conflict` as non-error, including DELETE. `Scheduler.Stop` ignores response status, so two rejected deletions return success. `Server.Reconcile` then sets `deletion_requested_at` and permanently skips later cleanup attempts. A fake Kubernetes transport returning 409 for both DELETE requests reproduced this. Make accepted status codes operation-specific: an already-absent resource can satisfy deletion, while a conflict must leave cleanup pending and retryable. This does not reopen broker permission, but can retain a workload or its credential Secret and makes the cleanup record inaccurate.

### [P2] Filter application runs before legacy scenario collection — internal/demo/collect.go:122

Configured application runs are stored with `scenario: application`, and the shared `/v1/runs` endpoint returns them alongside the original five scenarios. The default `collect-k8s` path selects every scripted scenario, then rejects `application` as an unknown stored scenario. Consequently, running the application workflow can break later basic report collection from the same database; map iteration can also leave a partially written collection. A stubbed kubectl/client reproduction returned one application run and reproduced `unknown stored scenario`. Filter the original scenario family before selection, or add an explicit application collection path. Test a mixed history containing both legacy and application runs.

### [P2] Make the export read limit compatible with supported engagements — internal/demo/demo.go:92

Every decoded operator response is truncated at 4 MiB, while engagements allow 20 children and target responses can reach 64 KiB. The exporter places captured evidence into base64 artifacts. A real store/export reproduction with only 12 permitted child attempts, six 48 KiB captures each, produced a valid approximately 4.80 MB bundle that `Operator.Call` could not decode (`unexpected EOF`). The full run-list endpoint can hit the same limit before an exact-ID collection even starts. Use a bounded export-specific limit consistent with the export contract, and paginate/project run listings instead of returning every historical response body. Detect oversized responses explicitly. Test a permitted engagement larger than 4 MiB.

### [P2] Render the selected image into deployments and worker configuration — Makefile:29

`make deploy IMAGE=tsunami:review` builds and loads `tsunami:review`, but applies `deploy/lab.yaml`, whose images remain `tsunami:dev`. The renderer changes only the worker runtime; it does not set the selected image or `TSUNAMI_IMAGE`, whose server fallback is also `tsunami:dev`. This can execute an older cached build or fail to start on a fresh cluster while appearing to deploy the newly built image. The Make dry run and an isolated renderer check confirmed the mismatch. Propagate the selected image through control/fixture manifests and worker configuration; keep verifier/probe images consistent with that build too. Assert the rendered image values before deploying.

## Overall assessment

**The core architecture is moving in the right direction. The next development phase should consolidate one useful HTTP assessment workflow before expanding into VM/mobile execution and specialist coordination.**

The strongest part is the separation of authority from agent reasoning: the service owns scope, credentials, reservations, revocation, evidence, and validation. Mecatl selects and dispatches permitted tool calls. Kubernetes runs the workloads. This directly addresses the original assignment's two concerns: off-scope actions and cross-customer disclosure.

The current result is a substantial owned-lab prototype. It demonstrates controlled assessment execution and evidence handling. It does not yet demonstrate a general autonomous security assessor, production customer isolation, or the assignment's production Stop/scale objectives. The README and recent design documents mostly make those distinctions correctly.

The main development risk is breadth: more execution profiles and elaborate infrastructure can accumulate while application setup remains manual and assessment usefulness is still measured on supplied facts. The curl milestone was useful for testing a command boundary, but wrapping the existing Go request in curl adds no testing intelligence. Another transport wrapper should not be the next product milestone.

## What the project is trying to build

The intended product is an assessment control plane:

1. A customer supplies authorization, target boundaries, dedicated access, and relevant business rules.
2. Tsunami selects supported built-in procedures and produces a bounded plan.
3. The customer approves that plan.
4. Agents and reviewed tools execute within permissions enforced outside their process.
5. The service captures observations, checks evidence, records uncertainty, and supports review, reporting, Stop, and retest.

The customer should choose a testing capability, rather than write an attack prompt or identify the vulnerability in advance. Reliable authorization testing still requires suitable accounts, legitimate objects, and intended access rules. A URL alone does not supply those facts.

The three projects have distinct responsibilities:

| Project | Appropriate responsibility | Current integration |
|---|---|---|
| Tsunami | Assessment lifecycle, permission compilation, execution admission, target enforcement, evidence, review and reporting | Implemented narrowly for the lab and configured HTTP read plans |
| Mecatl | Agent loop, tool dispatch, permission-policy integration and provider contracts | Published engine `v0.18.1` and Responses provider `v0.3.0`; no local checkout replacement |
| vapt-toolkit | Security methodology, prerequisites, test ideas, specialist workflows and export conventions | Local inventory, procedure inspiration, schema validation and partial checklist mappings |

Importing Mecatl's engine does not also provide the checkout's hosted service, Redis persistence, Kubernetes session leases, or complete tenant isolation. Tsunami currently creates an in-memory Mecatl session per worker and persists assessment state separately. That is a reasonable choice for bounded attempts that deliberately do not resume uncertain target actions.

The toolkit is methodology and potential adapter input. Its shared writable engagement directory, agent-written audit log, proposed agent retries, and agent-updated finding statuses should not become Tsunami's authority model. Preserve the useful test knowledge while keeping state transitions and final validation service-owned. The fresh inventory confirmed 44 scripts, 26 checklists, three schemas, eight documented roles and four prompt files; script entries remain unimplemented adapters.

## Architectural decisions to preserve

| Decision | Assessment and code evidence |
|---|---|
| Enforce at the component opening the connection | `internal/platform/scope.go` validates the origin and actual addresses and dials a validated numeric address; `Server.Execute` validates redirect hops and account/path pairs. Keep enforcement here when tools change. |
| Keep target/provider credentials outside workers | Registered application credentials stay in `platform/application.go`; workers get a run token and aliases. The gateway supplies the provider key. This reduces the authority available to a compromised worker. |
| Reserve before forwarding | Target requests, model calls, upstream attempts, and engagement child creation have server-owned reservations. Ambiguous failures retain allowance. Preserve this when storage or execution becomes distributed. |
| Separate Stop from Kubernetes deletion | Stop persists revocation and cancels tracked requests; cleanup follows. Missing dispatched Jobs are not recreated automatically. These are stronger semantics than treating pod existence as permission. |
| Validate observations independently | The application validator checks identities, legitimate reads, proof content and post-probe owner/session checks. Worker text cannot supply a confirmed verdict. Explicit sharing and missing-object controls are valuable. |
| Keep review separate from observation | Candidates are unrated until an explicit review pins evidence and a revision. Automated lab reviews are labeled. The CVSS calculation does not pretend to determine business impact. |
| Test boundaries with reachable controls | Retaining the original failed network experiment and bracketing later probes with positive controls is useful evidence. RuntimeClass existence alone is correctly treated as insufficient. |
| Use a small trusted deployment now | One Go process and bbolt make current transactions and cancellation comprehensible. The diagrams describe responsibilities; they do not require a microservice per box. |

## Architectural changes, in priority order

### 1. Complete an end-to-end assessment experience

Build one thin UI or CLI over the existing application, engagement, run, Stop, review and export APIs. Start with registered owned applications. Show prerequisites, proposed reads, expected access policy, execution mode, unavailable tests, evidence and uncertainty. The HTML templates are a useful design preview, but their screens, findings and metrics are fictional and have no backend connection.

Keep manual configuration available as an operator tool. Reduce the normal user's need to write selectors and proof hashes through a guided prerequisite-binding flow. Any network preflight or discovery needs a bounded permission before it connects; application registration must not itself become an unrestricted fetch endpoint.

Then add one useful reviewed procedure or a second independently implemented application. Define its positive, negative and ambiguous controls with the security engineer first. General session login, cookie refresh and arbitrary application discovery should each be explicit capabilities, not implicit extensions of the current two-bearer profile.

### 2. Make revocation, execution and assessment outcomes separately observable

The current run state combines dispatch, permission and result concepts. `running` can mean that dispatch intent was persisted before a pod starts; deadline reconciliation uses Stop; accepted deletion does not prove termination. The intended UI already acknowledges this, so make it an explicit API contract before connecting the UI.

Represent at least:

| Concern | Facts to expose |
|---|---|
| Permission | Approved, revoked or expired; reason, effective time and policy version |
| Execution | Queued/dispatched/started/ended; executor identity and attempt ID |
| Controlled activity | Last admitted operation, active request count, cancellation/drain observations |
| Cleanup | Pending, deletion accepted, termination observed or unknown |
| Assessment | Confirmed observation, no issue observed, inconclusive, and missing prerequisites |

Retain the conservative no-replay rule. A safe retry creates a new authorized attempt; a dispatch recovery protocol must not silently repeat an uncertain target operation. Add graceful shutdown coverage that waits for handler/controller exit before closing their store; the current `serve` helper starts `Shutdown` in a goroutine but returns when the listener closes.

The next qualification campaign should hold a request open behind a real gVisor curl worker, press run/engagement Stop, and measure revocation, target cancellation, last admitted activity and cleanup separately. Repeat through controller/API outages, restart and a connector partition. Use per-test counter deltas or request IDs: a lifetime `cancelled > 0` statistic can describe an earlier verification run.

### 3. Reduce optional-feature failure coupling without losing fail-closed execution

Keep the modular monolith, but make dependency health explicit. Startup currently loads fixture DNS, application configuration, connector TLS and provider configuration before serving any API. A connector certificate problem can therefore prevent the public control API and normal Stop interface from starting. The validation history records an actual restart failure after connector certificate expiry.

Design a degraded administrative mode: a failed private route denies private dispatch, while authenticated status and revocation remain available when the state backend is healthy. Distinguish API liveness, state/revocation readiness and each executor's admission readiness. Never turn failed private/model configuration into direct-network or alternate-provider fallback. Test this isolation of failures before a customer pilot.

### 4. Compile one explicit permission contract before adding executors

The current split between embedded catalog definitions and configured application plans is manageable at this size. Avoid creating another independent policy language for each future tool. Introduce a small compiled execution-permission type as a capability genuinely needs it, containing tenant/engagement/run binding, procedure and policy versions, executor identity, target/connector binding, allowed operation/account/resource pairs, expiry and budget references.

The service should issue that permission; each actual execution boundary should enforce it. The current connector's signed single-use grants are a useful narrow example. An MCP server, VM helper or browser service is another executor and potential network relay, even when the worker cannot connect to targets directly.

Before customer inference, also bind model disclosure permission and the allowed provider/price policy to approval. `Engagement.digest()` currently binds scope and executor, but model policy is selected later at child creation. That is acceptable for trusted synthetic operator experiments, and insufficient as the future customer's data-disclosure approval contract.

### 5. Preserve historical results independently of the latest executable catalog

Snapshots and digests protect current runs from accidental policy substitution. However, `ProcedureValid`, `getBoundRun` and engagement exports depend on definitions accepted by the current binary. Unsupported historical versions can make a whole engagement unavailable for normal viewing/export; the docs intentionally choose refusal over reinterpretation.

Keep that refusal for execution. For read-only history, retain the original validated result, evidence, review and validator version, and introduce a versioned renderer/validator registry or explicit archival representation. A procedure upgrade should neither bless old evidence under new rules nor make customers lose access to their old report. Add an upgrade fixture that checks both properties.

### 6. Separate durable metadata from evidence as the storage model grows

Bbolt is adequate for the present single-writer lab. Do not increase control replicas against the same file or treat a PVC as a distributed database. A future backend must preserve atomic approval/child reservations, deny-dominant Stop, optimistic review revisions and retained uncertain usage.

First separate interfaces and API projections: run listing should return bounded metadata with pagination; evidence should be fetched by authorized references. Today `Store.List` deserializes every historical run including bodies, reconciliation repeats that scan, updates rewrite a run's accumulated captures, and exports assemble full base64 bundles inside a read transaction. The 4 MiB defect is one visible consequence of these coupled contracts.

Before customer data, add tenant-bound artifact authorization, protected storage, retention/deletion including backups and exports, and a tested redaction/model-disclosure policy. Keep original restricted evidence separate from sanitized views. Hashes identify bytes; they do not establish authenticity against a compromised broker or connector. Store/restore and schema migrations need their own checks.

### 7. Treat customer identity and execution isolation as prerequisites for a pilot

Move the relevant customer-infrastructure work ahead of broad VM/mobile/specialist delivery. The current lab's tenant tokens, shared runner/control namespace and node, broad trusted-role network rules, HTTP links and synthetic credentials are documented constraints, rather than newly discovered regressions.

For the first customer profile, define authenticated principals and roles for configuring, approving, running, reviewing and administering; add TLS/service identity, credential rotation and reviewer attribution; isolate untrusted execution from trusted services; and independently test cross-tenant access across APIs, workloads, artifacts, connectors and exports. Keep target-application user identity separate from Tsunami tenant identity.

A complete mobile platform is not required for an HTTP pilot. The identity, data and boundary controls required by that HTTP pilot are required. Namespace separation alone does not replace those checks.

### 8. Keep the executor roadmap driven by useful capabilities

For each new adapter, require a concrete operation contract, supported inputs, scope checks, credentials, resource/output limits, cancellation, evidence interpretation and qualification tests. Prefer the native path when an external process provides no additional testing capability.

| Profile | Appropriate next boundary |
|---|---|
| HTTP/API | Existing broker; extend methods/headers/bodies only for a reviewed procedure. Raw or malformed-HTTP testing needs a separately approved contract. |
| VM/MCP | Typed authenticated operations with local expiry and Stop at the remote executor. Obtain the friend's actual interface before promising compatibility. |
| Network | Separate address/port/protocol/rate enforcement. The HTTP broker cannot authorize raw TCP/UDP scanner activity. |
| Mobile static | Hash-bound artifacts, offline processing and bounded extraction/parsers. |
| Mobile dynamic | Dedicated device/emulator and package identity, with controlled bridge privileges, cleanup and tenant separation. |
| Reporting/crypto | Access only to approved artifacts; no target credential or network capability unless explicitly required and authorized. |

Keep general shell and the toolkit's direct-network scripts disabled until their own boundaries are qualified. Toolkit redistribution terms remain an unresolved input. Inventory and schema compatibility do not establish execution compatibility.

### 9. Introduce specialist coordination only after a single agent is useful

If specialists become necessary, persist service-owned tasks, dependencies and artifact references. Give each child a narrower capability and charge all children to the engagement's aggregate allowances. Handle duplicate/late handoffs and cancellation explicitly. A reporting agent should not inherit target credentials; a validation agent's agreement with another model is not evidence of exploitability.

No separate orchestration framework, CRD or additional queue is currently justified just to execute the existing seven reads. Add infrastructure when its required guarantees and workload are demonstrated.

### 10. Measure assessment quality and operational capacity separately

The scripted and Responses-fixture suites test implementation. The retained free-model samples are guided and mixed: the catalog records 3/4 and 2/4 matches. Keep those original failures visible. Current application execution supplies the complete read plan and is scripted, so its success is not a measurement of discovery.

Create a held-out owned-lab set with renamed origins/objects, intentional sharing, field-level redaction, wrong/stale identities, unavailable objects, ambiguous responses and hostile instructions in response content. Establish expected findings independently of the agent prompt. Report planted-case detection, false positives, inconclusive prerequisites, operational failures, evidence completeness, time and calls; compare deterministic execution with model-assisted selection.

Production capacity is another experiment. Add bounded admission, tenant fairness, target rate/concurrency controls and a durable cost-reservation ledger before claiming large scale. Preserve unknown usage after interrupted inference. Metering reports consumption; admission decides whether work may start. The current per-run/engagement limits and two process-local inference slots do not imply 1,000-agent capacity or an account-wide monetary ceiling. No external model calls are needed for the next correctness and failure tests.

## Maintainability and development workflow

Keep one Go module and the current inward dependency direction. `platform` currently mixes HTTP routing, domain policy, bbolt operations, outbound requests, validation and report formatting. Extract small interfaces and application services when implementing the changes above, rather than doing a broad package rewrite. Keep domain permission/validation code independent of HTTP, Kubernetes and bbolt; keep transport and storage adapters concrete at the composition boundary. Continue using Mecatl's loop instead of copying it.

A useful next verification gate is a documented command that runs Go race tests, vet, the Python suite, and representative materialized export checks. `make test` currently runs only Go tests. Exercise mixed histories, large legal bundles, image selection, conflict cleanup and credential redirects in that gate. Keep live provider evaluations opt-in and separately labeled.

The reviewed repository had only `README.md` tracked at HEAD (`d2d5ea2`); the implementation and docs were untracked. Preserve a deliberate source/test/manifest baseline before further milestones so another session can reproduce a reviewed build. This review did not stage or commit anything. Historical example records should retain their original meaning; current helper messages can be corrected without rewriting archives.

## Recommended work order for the implementation agent

| Order | Deliverable | Evidence needed |
|---|---|---|
| 1 | Fix the five findings above | Go redirect test; DELETE-conflict retry; mixed-history collector; legal export above 4 MiB; rendered image assertions |
| 2 | Finish active-request Stop and dependency-failure behavior | Real gVisor curl request held open; run/engagement revocation; outage/restart observations; optional private failure does not disable administrative revocation |
| 3 | Connect a thin owned-application UI/CLI | Create, inspect plan, approve, start, observe, stop, inspect evidence, review and export through the same APIs; execution mode shown accurately |
| 4 | Improve assessment usefulness | Security-engineer-reviewed procedure or second application; vulnerable/fixed/ambiguous controls; less manual prerequisite setup; held-out evaluation |
| 5 | Complete the first profile's customer controls | Identity/roles, protected secrets/evidence, TLS, retention, historical report compatibility and independent tenant-boundary tests |
| 6 | Add broader profiles and specialists selectively | A separate permission/execution contract and qualification evidence for each; aggregate budgets and Stop |

This changes the emphasis of the roadmap, rather than discarding its long-term goals. VM/MCP contract discovery can proceed as design work while the HTTP slice is completed.

## Verification performed and limits

| Check | Result during this review |
|---|---|
| `go test -race -count=1 ./...` | Passed fresh across all packages; the demo package includes actual Mecatl/Responses-fixture workflows |
| `go vet ./...` | Passed |
| `python3 -B -m unittest discover -s scripts -p 'test_*.py'` | Six passed; one errored because this session denied Python socket creation (`PermissionError: Operation not permitted`). This was an environment restriction, not a demonstrated application assertion failure. |
| Toolkit inventory | Confirmed the documented counts without executing toolkit scripts |
| Actual toolkit-schema validation of `examples/tool-execution/vampi-vulnerable/bundle.json` | Passed; eight artifact files, one reviewed finding and seven evidence metadata files verified |
| Actual toolkit-schema validation of `examples/tool-execution/vampi-fixed/bundle.json` | Passed; eight artifact files verified, no reviewed findings |
| Four isolated Go review probes | Confirmed redirect credential forwarding, rejected cleanup reported as success, mixed-history collection failure, and a permitted export exceeding the client limit |
| Image-selection check | Make dry run and renderer reproduction confirmed `IMAGE=tsunami:review` leaves the deployment at `tsunami:dev` and omits `TSUNAMI_IMAGE` |
| Fresh Kubernetes/runtime/network/partition/load qualification | Not run; checked-in records were reviewed as historical evidence |
| External models or customer targets | Not called |

The review probes used a Go overlay with files under `/tmp/tsunami-review-20261104`, leaving repository test/source files unchanged. They assert the observed defective behavior; they are evidence for the findings, not regression tests asserting that fixes already exist. They can currently be rerun with:

```sh
go test -race -count=1 \
  -overlay /tmp/tsunami-review-20261104/overlay.json \
  ./internal/demo ./internal/kube -run '^TestReview' -v
```

The scratch directory also holds the local inventory, schema-check results and a source/document hash snapshot. It is temporary; implement permanent regression tests with the fixes rather than depending on `/tmp` for future qualification.

Review coverage included intent/design/status documents, the original take-home and response, core broker/store/application/catalog/engagement/review/model/connector/worker/scheduler/tool-runner paths, test call sites, deployment/build boundaries, selected setup/collection helpers, and the local UI preview contract. Mecatl and vapt-toolkit were reference context, not separately audited products. This was a source and architecture review with targeted reproductions; it does not prove absence of other defects, hostile-process containment, production Stop latency, assessment accuracy or customer readiness.
