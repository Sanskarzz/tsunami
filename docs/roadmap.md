# Next steps

The product goal now includes the toolkit's additional execution profiles and workflows. The customer-readiness stages and gates are in [the product roadmap](product-roadmap.md). Milestones below record completed lab work; they do not establish customer readiness.

## Milestone 1: reproducible workflow

- Real Mecatl engine with a scripted provider.
- Synthetic vulnerable/fixed API and known-owner validation.
- Trusted HTTP broker, evidence, durable stops, request quotas.
- Local and Kubernetes Job verification.
- Independent network-boundary check and an explicit failure for the missing sandbox runtime.

## Milestone 2: qualify the execution boundary

Implemented in a separate development cluster: pinned gVisor, Cilium default deny, actual CRI runtime inspection, raw-socket/filesystem/token smoke checks, and positive-control network tests at startup and after settling. The original cluster's failed result is retained. See [the experiment](sandbox-lab.md).

The listed IPv4 target, API, and TCP/UDP DNS paths are tested. IPv6, metadata, alternate protocols/interfaces, runtime failure, and network/controller failure testing remain. Test deliberate shell/client bypasses before enabling untrusted shell code; the current runtime smoke test is not a complete qualification.

Separate customer execution namespaces and keep trusted services off runner nodes. Add platform-service authorization and network tests so an allowed service cannot become a relay.

## Milestone 3: live agent evaluation

Implemented: an opt-in Responses model gateway, provider keys kept outside workers, a versioned known-owner workflow, durable call/wire-attempt reservations, bounded turns, usage records, and cancellation of inference. Both the offline fixture and one actual free-model paired trial passed with gVisor workers. A separate live inference stop check passed. See [the model path](model-path.md) and [recorded free trial](../examples/free-model/README.md).

Next: evaluate repeatedly with less guided cases. Measure misses, false positives, inconclusive results, evidence completeness, actual usage, and time to result. The current guided fixture names and supplied ownership rules do not form a blind benchmark. The free-only profile is implemented; general customer monetary budgets are not.

Mecatl's loop is not the scope boundary. No model decision can override the broker or promote a finding without evidence. Aggregate monetary spend reservations, account budgets, and billing reconciliation remain future work. The current per-run call/attempt limits include adapter retries; delegation is disabled.

## Milestone 4: private connectivity and failure cases

Implemented a synthetic customer connector in a separate Kubernetes namespace. The broker issues Ed25519-signed, single-use request grants; the connector rechecks target scope and actual dial addresses, consumes replay state durably, and enforces expiry locally. An emergency key latches the connector closed independently of the normal Stop API. The public lab path is unchanged.

Kubernetes checks cover all five scenarios, tenant binding, a disconnected connector with no direct fallback, and selected runner/connector network paths. Native tests cover active-request expiry without a control-plane callback, replay after restart, emergency cutoff persistence, and run/tenant/global cancellation. See [the private lab](private-connectivity.md).

The following milestone secures this connector link. Real customer-side connectivity and cross-node partition tests remain. The connector-local cutoff still trusts that connector and the Kubernetes management path; an external network cutoff is not implemented. No production stop guarantee is claimed.

## Milestone 5: connector transport and certificate lifecycle

Implemented TLS 1.3 with required client certificates on the connector, exact tenant/role URI identities, broker verification of the server hostname/identity, and a separate emergency operator certificate. Request grants remain required. Plaintext and direct-target fallback are refused.

Offline lab provisioning keeps the CA and operator private key outside Kubernetes. Normal setup preserves certificates; explicit leaf renewal updates Secrets and restarts the affected components without resetting run/replay/cutoff state. Tests cover wrong/missing/expired identities, role separation, expiry during a response and on an existing connection, and credential renewal. See [transport and lifecycle boundaries](connector-tls.md).

Next: the other internal TLS links, production identity/key management and revocation, real customer-side connectivity, and multi-node failure tests. Leaf renewal does not revoke a still-valid old certificate. The fixture workflow still supports synthetic data only; measured general vulnerability discovery remains separate work.

## Milestone 6: authorization procedures and repeatable evaluation

Implemented five embedded, versioned procedures: opaque object ownership, approved shared access, administrator role checks, unauthenticated access, and a nonexistent-object control. Runs snapshot each definition/digest; the broker and private connector enforce its exact account/path pairs. Procedures are original engineering definitions informed by the VAPT toolkit; external security review is still pending.

Repeated scripted and offline-provider evaluations cover evidence completeness and expected findings, missed cases, false positives, and inconclusive outcomes. Real-model trials are separate, explicit, free-only, and bounded. A premature model finish is retained as inconclusive rather than treated as success. Missing-evidence feedback is capped and leaves scope, request/model budgets, and Stop unchanged. See [the catalog](assessment-catalog.md) and [recorded outcomes](../examples/assessment-catalog/README.md).

Next: have the security engineer review prerequisites and access rules, add a separate less guided dataset, measure repeated live behavior across additional procedures, and support reviewed application-specific configuration. The initial/follow-up free trials matched 3/4 and 2/4 respectively; failures are retained. This is not a blind model benchmark or a general customer assessment engine. Private catalog inference remains disabled; its scripted path is tested through the connector.

## Milestone 7: inference failure diagnostics

Implemented durable per-call metadata, closed error codes, a 14-case local Responses failure harness, and separate catalog categories for operational failures versus incomplete or incorrect assessments. Partial failed turns release no tools. Missing provider usage stays uncertain even when the adapter emits zero counts. Timeouts, scope, budgets, and retry behavior are unchanged.

The archived live trials keep their original 3/4 and 2/4 scores. Offline analysis identifies one initial incomplete assessment and two follow-up model failures without assigning those old failures a cause. No external model calls were made for this milestone. See [diagnostic boundaries and reproduction](inference-diagnostics.md).

Next: independent procedure review, a held-out fixture dataset, then explicitly bounded live evaluation. The new traces support investigation; they do not establish provider reliability or solve discovery quality. Shell/scanner integration and real customer connectivity remain separate work.

## Milestone 8: engagement foundation

Implemented [engagements](engagements.md): pinned operator-approved scope, bounded child creation, engagement-wide Stop, derived checklist progress, evidence-linked candidates and an optional local toolkit inventory/schema check. Local Responses and Kubernetes public/private workflows exercise all five procedures on both fixture variants. Native tests cover racing creation/Stop, restart, active cancellation, tenant isolation and limits. Full toolkit execution and real customer assessments remain staged work in the product roadmap.

## Milestone 9: configured applications and data contracts

Implemented a versioned configured HTTP read plan, tenant-bound startup registry and broker-only bearer delivery. The v2 validator checks identities, legitimate objects, the cross-user probe, owner continuity and peer-session validity afterward. It has an explicit review API, CVSS 3.1 calculation, finding/evidence projections, materialized exports, CSV/Markdown guards and local partial checklist mappings. See [application implementation and qualification](application-adapters.md).

VAmPI is an existing open-source application deployed only in our lab; no external customer or model is involved in this campaign. The following milestone adds the first reviewed web/API executor. General authentication, private configured-app connectors, real customer data and full toolkit execution are still unfinished.

## Milestone 10: first reviewed web/API executor

Implemented original `curl-http/v1` for configured HTTP read plans: an actual fixed shell/curl command, private Unix bridge, broker-owned credentials and target requests, gVisor-only tools Jobs, approval-bound executor selection, bounded process I/O and cancellation. General commands and unmodified toolkit scripts remain disabled. See [implementation and qualification boundaries](tool-execution.md). Broader web/API modes and security-engineer review remain open; the next execution-profile work is VM/MCP contracts and offline mobile foundations.

## Later

Curated scanner adapters, shell integration, authenticated browser flows, human approvals for higher-risk actions, secret-manager delivery, retention/deletion, and richer reports. Mobile and raw network/AD are planned execution profiles in the expanded product scope. Higher-impact tests need separate explicit permissions and qualification; they are not enabled by the HTTP lab.

The immediate value is a small, explainable test with real evidence and explicit safety limits. A large multi-agent catalog comes after those pieces are demonstrated.
