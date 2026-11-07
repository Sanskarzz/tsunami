# Tsunami implementation tracker

Updated: 2026-11-06. Includes native review fixes, Stop/lifecycle qualification, the operator CLI, property-access and local binding preparation. This is an implementation status, not a promise of current health for every lab route. Private certificates and application credentials may need renewal before those demos.

**Current position:** a working Kubernetes/Mecatl lab for bounded HTTP authorization tests, with evidence, approvals, Stop, a synthetic private connector, and one controlled shell/curl adapter. It is not yet a general VAPT platform or ready for real customer data.

Status meanings:

- **Implemented:** code exists, with the stated lab tests. Production qualification is separate.
- **Partial:** a narrow version exists; important capabilities or validation are missing.
- **Not implemented:** design or roadmap work only.

The five P2 defects in the independent review are fixed in source, with regression tests and a `make verify` gate. See [the review follow-up](review-follow-up.md) for API compatibility and the revised work order. The [Stop lifecycle campaign](stop-lifecycle.md) now qualifies selected Stop and cleanup paths on the updated build; it does not repeat every older application/network campaign.

## 1. What is implemented

| Area | What exists | Evidence and boundary |
|---|---|---|
| Mecatl execution | Real pinned Mecatl engine, `http_request` and `finish_assessment` tools, scripted execution by default. | [Worker](../internal/worker/worker.go). Scripted execution is repeatable; it is not evidence of autonomous discovery. |
| Kubernetes lifecycle | Namespace-scoped Job reconciliation, separate run credentials, resource/security settings, durable dispatch state and cleanup. Dispatched runs are not automatically replayed. | [Scheduler](../internal/kube/scheduler.go), [security notes](security.md). This is a small Job reconciler, not an implemented CRD/operator product. |
| Sandbox lab | Dedicated Kind cluster with gVisor and Cilium enforcement; actual CRI inspection, no API token, raw-socket/root-write smoke checks, selected network bypass tests. | [Sandbox results](../examples/sandbox-kubernetes/README.md). The original cluster's failed network results remain recorded. |
| HTTP scope enforcement | Trusted broker checks configured origin, scheme, port, approved addresses, account/path pairs and redirects; dials a validated numeric address and keeps target credentials outside worker configuration. | [Scope code](../internal/platform/scope.go), [security notes](security.md). A compromised broker remains outside the demonstrated protection. |
| Authorization procedures | Five versioned procedures: object ownership, intentional sharing, administrator access, anonymous access and missing-object control. Broker captures and prerequisites determine the verdict. | [Catalog](assessment-catalog.md), [results](../examples/assessment-catalog/README.md). Initial procedures still need independent security review. |
| Configured applications | Tenant-bound startup registry with `http-object-access/v2` and `http-property-access/v1`: verify identities/private baselines, probe, then owner/session rechecks. | [Object design](application-adapters.md), [property design](property-access.md). Object access and field filtering have separate versioned meanings. Known inventory, private proofs and intended access rules are supplied. |
| Engagements | Draft/approve immutable scope, child-run reservations and limits, derived progress, skip reasons, engagement-wide Stop. | [Engagements](engagements.md). Child creation, credential and reservation commit together in bbolt. |
| Operator journey | `tsunami assess`: list configured applications, create/review a draft, approve its digest, launch one scripted procedure, inspect/watch metadata, Stop, recover IDs and export evidence privately. | [Guide](operator-cli.md), [compiled CLI records](../examples/operator-journey/README.md). Existing APIs remain authoritative. This is operator access to preconfigured synthetic applications, not general onboarding. A separate local UI now connects these same APIs. |
| Local operator UI | Opt-in embedded frontend for configured applications, scope/digest review, one scripted attempt, metadata observation, independent engagement Stop, recovery, evidence snapshots/downloads and a credential-free API explorer. | [Guide](local-ui.md), [presenter path](demo-walkthrough.md). Loopback port-forward only; operator token in page memory; no URL registration, target password login, model application execution or customer portal. |
| Application onboarding | Initial form, durable typed questions/revisions, protected origin-bound bearer credentials, explicit four-read preflight, inferred supported bindings, atomic application/draft registration and setup/assessment Stop. | [Guide](application-onboarding.md). Checked-rule advisor, synthetic bearer/JSON only; no general login, discovery, model planning or customer-readiness claim. |
| Binding preparation | `prepare-plan` derives public assertions, literal object paths and protected-value hashes from supplied synthetic baselines; emits a private unapproved package. Both owned setup helpers use it. | [Guide](binding-preparation.md), [linked runtime records](../examples/binding-preparation/README.md). No network requests or authority from preparation; server credential binding, approval and live validation remain required. |
| Stop foundation | Durable run, engagement, tenant and global revocation; cancellation of tracked work; explicit permission/activity/execution/cleanup observations, unfinished-operation records and graceful shutdown. | [Lifecycle and qualification](stop-lifecycle.md). Actual gVisor/curl Stop, controller API denial and restart after revocation are tested. API resource absence does not prove a disconnected process has exited. |
| Private connector lab | Tenant-bound signed single-use grants, local scope checks, durable replay protection, expiry and persisted emergency cutoff. Broker-to-connector mTLS with role separation. | [Private connectivity](private-connectivity.md), [TLS lifecycle](connector-tls.md). This connects synthetic lab namespaces, not a real customer network. |
| Model gateway and diagnostics | Optional broker-mediated inference, provider key outside workers, durable call/attempt reservations, free-only routing controls, bounded feedback and per-call diagnostics. | [Model path](model-path.md), [failure diagnostics](inference-diagnostics.md). Live trials have mixed results; missing usage remains uncertain. |
| Finding review and evidence exports | Unrated candidates, review revisions, evidence fingerprints, CVSS 3.1 calculation from an explicit vector, JSON/CSV/Markdown exports and artifact hashes. | [Application contracts](application-adapters.md). Example confirmations are labeled `automated_lab_test`; they are not human signoff. |
| Toolkit contracts and progress references | Local validation against all three actual engagement/finding/evidence schemas, relational/file checks, inventory and partial checklist-ID mappings. | [Toolkit mapping](toolkit-mapping.md). Original schemas/scripts/prompts are not vendored or executed. |
| First command adapter | `curl-http/v1`: fixed shell script and actual curl, private Unix bridge, approved executor selection, gVisor-only tool Jobs, time/output/invocation limits and cancellation. | [Execution design](tool-execution.md), [results](../examples/tool-execution/README.md). It adds a command-to-broker path for the same read test, not a new vulnerability class or general curl compatibility. |

### Latest recorded checks

- Onboarding browser campaign: frontend-only question, unsupported-login guard, protected input without token exposure, four approved own-account reads, inferred plan, atomic registration/draft, explicit final approval and real gVisor/Mecatl execution. The new app produced one unreviewed candidate with six runtime baselines/rechecks. [Records](../examples/onboarding/README.md). No model/customer calls.

- Local UI browser qualification: actual vulnerable/fixed/held-Stop cases, one Start per engagement, wrong-digest and invalid-token rejection, committed Start response loss/recovery, Stop during a held viewer export, evidence downloads/hashes, display injection and narrow layout. [Records](../examples/local-ui/README.md). No external model/customer calls or human reviews.

- Repeatable showcase: a fresh seven-case extended CLI rehearsal passed, including two unreviewed finding candidates, fixed/filtered controls, an inconclusive response and two held-request Stop cases. Zero external model calls; [records and limits](../examples/showcase/README.md).

- Curl-backed VAmPI campaign: vulnerable variant `confirmed`; fixed variant `no_issue_observed`; seven approved target requests each, plus a blocked scope attempt.
- Curl network campaign: 54 selected connection attempts denied, with reachable controls before/after and the broker reachable. Covers literal IP, `--resolve` and `--connect-to` across six IPv4/TCP endpoints and three phases.
- Separate Go network checks include TCP/UDP DNS and selected private/application paths.
- Fresh Go race tests, vet, twenty Python tests and materialized export/toolkit-schema validation pass. The integrity gate now checks 32 bundles and 240 artifacts, six prepared/runtime matches and the 20 archived trial hashes.
- Four owned gVisor/curl Stop cases separately record acknowledgement, target cancellation, CRI exit and API cleanup. Controller API denial keeps cleanup pending; restart after committed revocation does not restore permission. See [records and limits](../examples/stop-lifecycle/README.md).
- Compiled CLI campaign: owned VAmPI vulnerable/fixed outcomes, explicit digest approval, rejected unapproved/stopped launch, private unreviewed exports, and engagement Stop after a read-only watch timeout. Native tests additionally cover expired-session inconclusive results, cross-tenant denial and a lost start response without automatic retry.
- Property-access campaign: known private-value exposure `confirmed`, correct omission `no_issue_observed`, redaction `inconclusive`, and held-request Stop. Target counters confirm seven approved reads per completed case; the held gVisor/curl worker has a separate actual runtime inspection. Proof bindings are generated for this owned fixture, not discovered in customer applications.
- Binding-preparation campaign: two VAmPI and four property plans derived by the compiler match their actual approved runtime snapshots, apart from server-assigned credential fingerprints. Expected outcomes and Stop controls passed; raw source facts remain private and are not authenticated by their hashes.

These are recorded observations for specific builds and paths. They do not establish general shell containment, blind vulnerability discovery, 1,000-agent capacity or a production 60-second Stop guarantee. See [validation history](validation-results.md).

## 2. What is only partially implemented

| Capability | Existing part | Still missing |
|---|---|---|
| Customer isolation | Run/tenant authorization, restricted worker Jobs and selected network tests. | Customer execution boundaries across namespaces/nodes, isolation from trusted services, independent adversarial tests and wider network/runtime qualification. Workers and trusted services currently share a lab namespace/node. |
| Web/API testing | Five embedded synthetic procedures and configured object-access/property-access read profiles. | Independent review, customer prerequisite binding, authentication/session workflows, more useful tests, scoped methods/headers/bodies and an agreed approach to discovery. Arbitrary customer URLs are not accepted. |
| Shell/tool integration | One fixed shell/curl adapter. | Other command adapters and process/resource failure tests. No general shell tool, unmodified toolkit scripts, browser automation or scanners. |
| Private targets | Signed-grant connector between synthetic namespaces. | Customer-side deployment/connectivity, private configured-application plans, operational recovery and real network-partition tests. Private live-model execution is disabled. |
| Secrets | Broker-held synthetic target credentials, Kubernetes-mounted component secrets and distinct run tokens. | Secret-manager/ESO integration, customer-issued run-scoped target credentials, rotation/revocation and safe expiry/refresh. VAmPI's one-hour credentials are reused across allowed runs. |
| Evidence and reports | Tenant-checked API exports, linked artifacts, review API and tested output guards. | Protected object storage, encryption, general redaction, retention/deletion, individual reviewer identity, remediation/retest workflow and audit suitable for customers. |
| Checklist coverage | Durable procedure progress and export-side sampled mappings. | Reviewed test coverage for the broader checklist items. One procedure completing must not mark an entire standard item complete. |
| Cost controls | Per-run request/model/time bounds and engagement child/concurrency limits. | Distributed rate/concurrency limits, customer-wide monetary reservations, unknown-cost reconciliation, usage metering and scheduling fairness. |
| Agent quality | Repeatable scripted/offline evaluations and bounded real free-model trials. | Independent procedure review, less guided datasets, repeated quality measurements and customer data/LLM handling qualification. Configured applications currently use scripted execution only. |
| Certificate lifecycle | Explicit leaf renewal and expiry/role tests. | Automated renewal, safe CA rollover, rapid revocation of stolen valid leaves and TLS/identity on the remaining HTTP links. |

## 3. Remaining work, in recommended order

### A. Finish the first useful web/API execution slice — stage 10

- [ ] Get the current procedures and evidence rules reviewed by a security engineer.
- [x] Add an original field-level exposure test beyond the curl transport wrapper, with explicit prerequisites, read-only scope, evidence and ambiguous outcomes. The owned lab is qualified; independent security-engineer review is still pending.
- [ ] Extend the broker contract only as needed for that test: bounded methods, headers and bodies, with explicit approval for state-changing actions. GET alone does not mean harmless.
- [ ] Add each supported tool as its own adapter with scope, quota, output, cancellation and evidence tests. Installing a tool does not make it supported.
- [x] Test Stop while a real gVisor curl worker has an active target request. Measure API acceptance, target cancellation, last accepted request and workload cleanup separately.
- [ ] Exercise broader broker/controller outages, partitions, delayed cleanup and restart during active work. Controller API denial and restart after committed revocation are tested; broker crash, lost acknowledgement, node partitions and production load remain. Validate the 60-second objective under stated conditions.
- [ ] Qualify additional network/runtime paths needed by the deployment: metadata and platform endpoints, IPv6 where enabled, alternate interfaces, cross-node traffic and CNI failure behavior.
- [ ] Keep arbitrary shell execution disabled until its separate containment/resource/process tests justify enabling it. Process groups alone do not stop a malicious process that creates a new session.
- [x] Refresh the engagement demo's stale finding-schema export message. Preserve archived results with their original meaning.
- [x] Expose permission, execution and cleanup facts separately; cover graceful shutdown and administrative revocation when optional routes are unhealthy. Readiness is for state/revocation; route status describes startup configuration.
- [x] Connect an operator CLI to the configured application journey, with explicit scope review/approval, metadata observation, Stop, recovery and private export. A small local operator UI also connects these APIs; general onboarding and the customer portal remain planned.

### B. Add the other execution profiles — stage 11

- [ ] **VM/MCP:** define authenticated typed operations, run identity, scope/expiry, request budgets, evidence return and Stop at the actual remote executor. Obtain the friend's interface details before claiming compatibility.
- [ ] **Network:** add a separate executor/network boundary for approved IPs, ports, protocols and rates. Start with bounded unprivileged connection probes; raw sockets and scanner scripts need separate review. The HTTP broker cannot enforce their scope.
- [ ] **Mobile static:** accept approved artifacts, bind them to the engagement by hash, and run bounded offline analysis. Test archive traversal, malicious files, parser limits and output access.
- [ ] **Mobile dynamic:** design device/emulator and package binding, tenant isolation, credentials, cleanup and Stop. Determine Android/iOS needs separately; do not assume gVisor containers replace the required devices or VMs.
- [ ] **Cloud and crypto:** review those toolkit ideas separately. Define account/resource permissions for cloud operations and bounded offline inputs for suitable crypto analysis.
- [ ] **Reporting utilities:** review or replace useful toolkit utilities with operations restricted to approved engagement artifacts. Keep network access, path handling, untrusted markup and sensitive output controlled.

### C. Build specialist coordination — stage 12

- [ ] Implement a service-owned task graph for selected recon, testing, validation and reporting roles.
- [ ] Issue bounded child permissions and artifact references. A specialist or remote MCP tool must not expand scope or act as an indirect network relay.
- [ ] Apply aggregate engagement budgets and Stop across local and remote executors.
- [ ] Test duplicate, failed and late handoffs, untrusted evidence, scope-expansion requests and partial completion. Agent agreement alone must not confirm a finding.

The toolkit's eight documented roles and four prompt files are inventory/reference material today. A general multi-agent scheduler and delegation are not implemented.

### D. Add customer infrastructure — stage 13

- [ ] Replace demo tenant bearer tokens with customer identity and roles for approval, execution, review and administration.
- [ ] Choose and implement a durable backend that safely supports the intended number of control-plane replicas; add migrations, backups and restore tests. The current bbolt process is the lab backend.
- [ ] Deliver appropriately scoped, short-lived customer secrets through a secret manager. ESO is a possible delivery option, not an existing integration.
- [ ] Add tenant-authorized evidence storage, encryption, redaction and customer-agreed retention/deletion, including exported copies and backups.
- [ ] Add TLS and service identity on remaining internal links; automate certificate/key lifecycle and revocation.
- [ ] Deploy and qualify customer-specific private connectivity and infrastructure exclusions. There must be no public/direct fallback or cross-customer route.
- [ ] Provide an emergency egress cutoff that remains usable when the normal Stop API is unavailable. The existing connector-local cutoff still trusts that connector and its management path.
- [ ] Define customer data disclosure to models and obtain the required approval before enabling application inference. A compromised credential recipient may still read or misuse its own assigned access.
- [ ] Add audit and individual reviewer identity, customer review/remediation/retest flows, and practical operator/customer access to the workflow.
- [ ] Use immutable images and a reviewed dependency update process. The current tools image pins its base but installs apt packages from the build-time repository.

### E. Demonstrate scale and operations — stage 14

- [ ] Add distributed rate and concurrency enforcement, bounded queues and tenant scheduling fairness. Redis-backed limiting is not implemented in Tsunami yet.
- [ ] Reserve token/monetary budgets before calls; reconcile actual and unknown usage without silently refunding uncertain work.
- [ ] Add usage metering where useful. OpenMeter is a proposed option, not implemented; metering does not enforce the budget.
- [ ] Add production metrics, tracing, alerts and operator dashboards for dispatch, tool/model calls, denials, evidence failures, spend and Stop. OpenTelemetry and KubeArmor/eBPF integration are not implemented here.
- [ ] Build an owned-lab load generator and multi-node tests. Increase real active isolated workloads gradually and record hardware, throughput, resource use, failures and rejected work.
- [ ] Measure Stop under load and failure, including remote executors. Test recovery, lost acknowledgements and ambiguous target/model outcomes.
- [ ] Review isolation and failure behavior independently before claiming production capacity. No 1,000-concurrent-agent result exists.

### F. Run a supervised customer pilot — stage 15

- [ ] Agree written target scope, test identities/data, permitted profiles/actions, operating window, model/data handling and incident contacts.
- [ ] Complete the customer controls and qualification required by the selected profile.
- [ ] Run a limited assessment with human oversight, validate findings with the customer/security engineer, verify cleanup and address gaps before broader use.

A limited HTTP pilot needs its own completed controls and release checks. It does not require every planned mobile/network feature. Delivering the full toolkit-inspired product requires those additional profiles as well.

## 4. How much of vapt-toolkit is integrated?

The recorded inventory has **44 scripts**: 14 web/API, 19 mobile, four network, two cloud, two crypto and three reporting. It also has 26 checklists, three schemas, eight documented roles and four role-prompt files.

| Toolkit part | Current Tsunami use | Work left |
|---|---|---|
| Testing workflow and playbooks | Informed the original authorization procedures, legitimate baselines, cross-user probes and evidence-first validation. | Independent review, more useful procedures and less guided evaluation. |
| Web/API scripts | Reviewed as reference; original Go procedures and one original curl adapter exist. | Existing source scripts are not executed or declared compatible. Review/adapt or replace useful behavior individually. |
| Network/mobile/cloud/crypto scripts | Inventory/reference only. | Their execution profiles, permissions, validation and tests. |
| VM/MCP setup | Friend's workflow is known at a high level. | Actual interface details, executor implementation and end-to-end qualification. |
| Engagement/finding/evidence schemas | All three are used locally to validate Tsunami's exports; hashes and linked files are checked. | Customer-grade storage/lifecycle and broader workflow compatibility. Schema validity alone is not finding correctness. |
| Checklists | IDs/hashes imported locally; sampled mappings and durable procedure progress. | Full reviewed mappings and execution coverage. All 26 checklists are not implemented. |
| Reporting utilities | Tsunami has original JSON/CSV/Markdown exports. | The toolkit's reporting scripts themselves are not integrated. |
| Specialist roles/prompts | Inventory and architectural reference. | Coordination, scoped delegation, handoffs and aggregate limits. |

All original toolkit script entries remain marked `adapter_not_implemented` in the inventory. The original curl adapter does not change that status. No license file was found during review; permission to redistribute or copy toolkit content remains unresolved. Original implementations and local read-only references can continue while that is clarified.

## 5. Inputs still needed

- Toolkit redistribution/usage terms and security-engineer review of procedures.
- The friend's VM/MCP connection and tool contracts; a sanitized workflow example when available.
- Which additional web/API tests and mobile platforms to prioritize.
- Deployment environment and an approved load-test budget.
- A customer-approved pilot target, data/model rules, retention agreement and intended authorization rules.

These inputs are not reasons to pause the existing lab work. They are required before claiming compatibility, capacity or customer readiness in those areas. Development model use remains free-only, with no paid fallback.

## 6. What to build next

**Showcase complete:** both scoped milestones are implemented. The [demo guide](showcase.md) covers prerequisite/expiry checks, explicit owned-data refresh, core/extended commands, a presenter path and recovery. The [fresh rehearsal](../examples/showcase/README.md) records seven expected outcomes and 54 selected network denials with reachable controls. No further milestone is required for this controlled CLI demonstration; customer readiness has separate gates.

**For the product:** obtain independent procedure review, improve real customer prerequisite/onboarding support and complete the pilot's customer controls. Local preparation reduces manual binding but does not discover facts, authenticate sources or grant permission. Broader failure and isolation qualification remains.

**Confirmed onboarding direction:** initial application/access form → setup-agent review → focused questions only when facts are missing → bounded approved preflight → backend readiness evaluation → final scope confirmation → assessment workflows. See [the readiness design](onboarding-readiness.md). A first checked-rule conversation, protected bearer flow, four-read validation and atomic dynamic registration now exist; see [the implemented contract](application-onboarding.md). Password/SSO adapters, live-model setup planning and automatic discovery remain unimplemented.

**Then:** complete the first pilot profile's customer controls and broader Stop failure tests. VM/MCP contract discovery can continue as design work. Broader execution profiles and specialists follow a useful, qualified HTTP journey; see [the review's ordered plan](review-follow-up.md#next-implementation-order).

Keep this tracker aligned with the code and test records. [Milestone history](roadmap.md) records completed lab work; [the product roadmap](product-roadmap.md) defines the broader stages and release checks. Older example documents describe their historical builds and may list capabilities that were implemented in later milestones.
