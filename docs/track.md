# Tsunami implementation tracker

Updated: 2026-11-05. Includes native review fixes; Kubernetes results remain the recorded campaigns through 2026-11-03. This is an implementation status, not a fresh check of cluster health. Lab credentials and certificates may need renewal before another demo.

**Current position:** a working Kubernetes/Mecatl lab for bounded HTTP authorization tests, with evidence, approvals, Stop, a synthetic private connector, and one controlled shell/curl adapter. It is not yet a general VAPT platform or ready for real customer data.

Status meanings:

- **Implemented:** code exists, with the stated lab tests. Production qualification is separate.
- **Partial:** a narrow version exists; important capabilities or validation are missing.
- **Not implemented:** design or roadmap work only.

The five P2 defects in the independent review are fixed in source, with regression tests and a `make verify` gate. See [the review follow-up](review-follow-up.md) for API compatibility, verification and the revised work order. These changes have not yet received a fresh Kubernetes qualification campaign.

## 1. What is implemented

| Area | What exists | Evidence and boundary |
|---|---|---|
| Mecatl execution | Real pinned Mecatl engine, `http_request` and `finish_assessment` tools, scripted execution by default. | [Worker](../internal/worker/worker.go). Scripted execution is repeatable; it is not evidence of autonomous discovery. |
| Kubernetes lifecycle | Namespace-scoped Job reconciliation, separate run credentials, resource/security settings, durable dispatch state and cleanup. Dispatched runs are not automatically replayed. | [Scheduler](../internal/kube/scheduler.go), [security notes](security.md). This is a small Job reconciler, not an implemented CRD/operator product. |
| Sandbox lab | Dedicated Kind cluster with gVisor and Cilium enforcement; actual CRI inspection, no API token, raw-socket/root-write smoke checks, selected network bypass tests. | [Sandbox results](../examples/sandbox-kubernetes/README.md). The original cluster's failed network results remain recorded. |
| HTTP scope enforcement | Trusted broker checks configured origin, scheme, port, approved addresses, account/path pairs and redirects; dials a validated numeric address and keeps target credentials outside worker configuration. | [Scope code](../internal/platform/scope.go), [security notes](security.md). A compromised broker remains outside the demonstrated protection. |
| Authorization procedures | Five versioned procedures: object ownership, intentional sharing, administrator access, anonymous access and missing-object control. Broker captures and prerequisites determine the verdict. | [Catalog](assessment-catalog.md), [results](../examples/assessment-catalog/README.md). Initial procedures still need independent security review. |
| Configured applications | Tenant-bound startup registry and `http-object-access/v2`: verify two identities, legitimate reads, cross-user probe, then owner/session rechecks. | [Application design](application-adapters.md). Owned VAmPI vulnerable/fixed variants pass expected outcomes. Known inventory and access rules are supplied. |
| Engagements | Draft/approve immutable scope, child-run reservations and limits, derived progress, skip reasons, engagement-wide Stop. | [Engagements](engagements.md). Child creation, credential and reservation commit together in bbolt. |
| Stop foundation | Durable run, engagement, tenant and global revocation; cancellation of tracked HTTP/model work; reconciliation cannot reopen stopped permissions. | [Stop semantics](security.md#stop-semantics). Native tests cover races, restart and active cancellation. Cleanup acceptance does not prove a disconnected process has exited. |
| Private connector lab | Tenant-bound signed single-use grants, local scope checks, durable replay protection, expiry and persisted emergency cutoff. Broker-to-connector mTLS with role separation. | [Private connectivity](private-connectivity.md), [TLS lifecycle](connector-tls.md). This connects synthetic lab namespaces, not a real customer network. |
| Model gateway and diagnostics | Optional broker-mediated inference, provider key outside workers, durable call/attempt reservations, free-only routing controls, bounded feedback and per-call diagnostics. | [Model path](model-path.md), [failure diagnostics](inference-diagnostics.md). Live trials have mixed results; missing usage remains uncertain. |
| Finding review and evidence exports | Unrated candidates, review revisions, evidence fingerprints, CVSS 3.1 calculation from an explicit vector, JSON/CSV/Markdown exports and artifact hashes. | [Application contracts](application-adapters.md). Example confirmations are labeled `automated_lab_test`; they are not human signoff. |
| Toolkit contracts and progress references | Local validation against all three actual engagement/finding/evidence schemas, relational/file checks, inventory and partial checklist-ID mappings. | [Toolkit mapping](toolkit-mapping.md). Original schemas/scripts/prompts are not vendored or executed. |
| First command adapter | `curl-http/v1`: fixed shell script and actual curl, private Unix bridge, approved executor selection, gVisor-only tool Jobs, time/output/invocation limits and cancellation. | [Execution design](tool-execution.md), [results](../examples/tool-execution/README.md). It adds a command-to-broker path for the same read test, not a new vulnerability class or general curl compatibility. |

### Latest recorded checks

- Curl-backed VAmPI campaign: vulnerable variant `confirmed`; fixed variant `no_issue_observed`; seven approved target requests each, plus a blocked scope attempt.
- Curl network campaign: 54 selected connection attempts denied, with reachable controls before/after and the broker reachable. Covers literal IP, `--resolve` and `--connect-to` across six IPv4/TCP endpoints and three phases.
- Separate Go network checks include TCP/UDP DNS and selected private/application paths.
- Go race tests, vet, seven Python tests and materialized export/schema validation passed for the recorded build.
- Active curl/broker/target cancellation is tested natively. The latest Kubernetes campaign does not measure active curl Stop latency.

These are recorded observations for specific builds and paths. They do not establish general shell containment, blind vulnerability discovery, 1,000-agent capacity or a production 60-second Stop guarantee. See [validation history](validation-results.md).

## 2. What is only partially implemented

| Capability | Existing part | Still missing |
|---|---|---|
| Customer isolation | Run/tenant authorization, restricted worker Jobs and selected network tests. | Customer execution boundaries across namespaces/nodes, isolation from trusted services, independent adversarial tests and wider network/runtime qualification. Workers and trusted services currently share a lab namespace/node. |
| Web/API testing | Five synthetic procedures and one configured object-access read profile. | More useful reviewed tests, authentication/session workflows, scoped methods/headers/bodies and an agreed approach to discovery. Arbitrary customer URLs are not accepted. |
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
- [ ] Add a reviewed HTTP test that provides useful behavior beyond the current curl transport wrapper. Define prerequisites, permitted requests, evidence and ambiguous outcomes before implementation.
- [ ] Extend the broker contract only as needed for that test: bounded methods, headers and bodies, with explicit approval for state-changing actions. GET alone does not mean harmless.
- [ ] Add each supported tool as its own adapter with scope, quota, output, cancellation and evidence tests. Installing a tool does not make it supported.
- [ ] Test Stop while a real gVisor curl worker has an active target request. Measure API acceptance, target cancellation, last accepted request and workload cleanup separately.
- [ ] Exercise broker/controller/Kubernetes API outages, partitions, delayed cleanup and restart during active work. Validate the 60-second objective under stated conditions.
- [ ] Qualify additional network/runtime paths needed by the deployment: metadata and platform endpoints, IPv6 where enabled, alternate interfaces, cross-node traffic and CNI failure behavior.
- [ ] Keep arbitrary shell execution disabled until its separate containment/resource/process tests justify enabling it. Process groups alone do not stop a malicious process that creates a new session.
- [x] Refresh the engagement demo's stale finding-schema export message. Preserve archived results with their original meaning.
- [ ] Expose permission, execution and cleanup facts separately; cover graceful shutdown and administrative revocation when optional routes are unhealthy.

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

**First:** close the Kubernetes active-request Stop gap and add one useful reviewed web/API test beyond the transport wrapper. Keep its scope small and record vulnerable, fixed and ambiguous outcomes.

**Then:** connect a thin owned-application UI/CLI, improve useful HTTP testing and complete the first pilot profile's customer controls. VM/MCP contract discovery can continue as design work. Broader execution profiles and specialists follow a useful, qualified HTTP journey; see [the review's ordered plan](review-follow-up.md#next-implementation-order).

Keep this tracker aligned with the code and test records. [Milestone history](roadmap.md) records completed lab work; [the product roadmap](product-roadmap.md) defines the broader stages and release checks. Older example documents describe their historical builds and may list capabilities that were implemented in later milestones.
