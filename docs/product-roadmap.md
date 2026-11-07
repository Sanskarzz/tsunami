# Customer-use roadmap

The project goal has expanded beyond the take-home: a platform for authorized assessments using the toolkit's web/API, network, mobile, reporting, and specialist-agent ideas. Kubernetes coordinates execution; Mecatl runs agent loops. The original assignment response stays unchanged.

The current release is still a synthetic lab. The HTTP-only boundary is its first execution profile, not the limit of the planned product. Adding a tool to a container does not make it supported or safe. Each adapter needs an execution boundary, explicit scope, quotas, cancellation, captured evidence, and tests.

The [independent review](review-report.md) changes the immediate priority: its five source defects are addressed in [the follow-up](review-follow-up.md). The [Stop lifecycle milestone](stop-lifecycle.md) covers selected failure cases; the [operator CLI](operator-cli.md) connects the configured-application journey. The new [property-access test](property-access.md) adds field filtering beyond object access, with owned vulnerable/filtered/ambiguous controls. Independent procedure review, less manual customer prerequisite binding and the pilot's customer controls remain next. A small [local operator UI](local-ui.md) now connects the lab APIs; the customer portal and general authentication/onboarding remain planned. Stage numbers below identify capability areas; they do not require VM/mobile delivery before the HTTP profile's customer controls.

[Local preparation](binding-preparation.md) now derives bindings from supplied synthetic baseline facts while retaining registration, approval and runtime verification. It is a partial setup improvement, not general onboarding. The [Axel showcase plan](showcase-readiness.md) is a separate two-milestone slice; both preparation and repeatable packaging/rehearsal are now complete. The [demo guide](showcase.md) and retained seven-case run support a controlled CLI demonstration. This does not establish production or full-toolkit readiness.

## Current inventory

The local toolkit contains 44 scripts: 14 web/API, 19 mobile, four network, two cloud, two crypto, and three reporting. It has 26 checklists, three JSON schemas, eight documented roles, and four role-prompt files. The inventory records filenames, hashes, and checklist IDs without running scripts or copying their contents. It is not a standards compliance audit.

The friend's VM/MCP workflow is additional context. It is not wired into Tsunami. We still need its connection contract, tool interfaces, isolation assumptions, and a sanitized assessment example. No license file was found; redistribution terms are pending. Local references and original implementations can continue while that is clarified.

## Implementation order and release gates

| Stage | Work | Required evidence before marking it done |
|---|---|---|
| 8: engagement foundation | Approved scope snapshots, child-run limits, evidence links, checklist progress, engagement Stop | Implemented for synthetic targets. Native race/restart/cancellation tests and end-to-end lab checks; see [engagement notes](engagements.md). |
| 9: application adapters and data contracts | Implemented a bounded configured HTTP profile, review API, finding/evidence exports, local partial checklist mappings and output guards. | VAmPI is deployed in our own lab with positive/negative controls. See [application boundaries](application-adapters.md). Broader authentication/response shapes, real customer data, full checklist mappings and human procedure review remain work. |
| 10: web/API tool execution | First original `curl-http/v1` read adapter implemented as a command-to-broker boundary prototype; it adds no new vulnerability class. Broader reviewed adapters, bounded methods/headers/bodies and useful tool behavior remain. | Selected curl bypass and owned-application checks pass; see [records](../examples/tool-execution/README.md). [Active-request Stop records](../examples/stop-lifecycle/README.md) now include gVisor workers, controller API denial and restart after revocation. Broader failure cases, modes, side effects and independent procedure review still need qualification. |
| 11: VM, network and mobile profiles | Authenticated MCP interfaces, isolated VM/network executors, offline APK analysis, then dedicated Android/iOS device execution. | Review and qualify each profile separately. Test target boundaries, device/package binding, indirect requests, credential exposure, partition/expiry, process cleanup and emergency cutoff. Unsupported modes must refuse execution. |
| 12: specialist-agent coordination | Service-owned task graph for recon, web/API/mobile/network testing, validation and reporting; bounded grants and artifact handoffs. | Specialists cannot expand scope, spawn privileged children, forge progress or confirm each other's unsupported findings. Test failed/duplicate/late handoffs, aggregate budgets and engagement-wide Stop. |
| 13: customer infrastructure | Tenant identity and roles, durable concurrent backend, protected evidence storage, secret delivery, TLS, retention/deletion, audit and key lifecycle. | Independent cross-tenant tests, secret/redaction tests, restore/migration tests, revocation and rotation, customer data/LLM handling review. Current demo tokens and bbolt process are not the final customer architecture. |
| 14: capacity and operational qualification | Distributed rate/concurrency/budget reservations, metrics, load generator, multi-node deployment and failure testing. | Publish the tested hardware, actual active agent count, throughput, resource use, rejected work and Stop latency distribution. Test failures while loaded; a single-node Kind run or 1,000 API requests is not 1,000 isolated agents. |
| 15: supervised customer pilot | Written scope, customer test identities/data, operating window, approved test profiles, human review and incident/cleanup procedures. | Run a limited agreed assessment, review findings with the customer/security engineer, and fix gaps before broader use. |

Stages can overlap, but their gates cannot be skipped by enabling a feature flag. Customer readiness includes operational and assessment validation, not only passing unit tests. There is no measured 1,000-agent capacity or production 60-second Stop guarantee yet.

## Execution profiles

- **HTTP/API:** keep request enforcement outside the agent. Adapt supported clients to the broker, or build a reviewed proxy integration that inspects HTTPS requests. Proxy environment variables alone do not prevent direct sockets. Malformed HTTP/smuggling tests may need a different, explicitly approved profile because an HTTP broker normalizes requests.
- **Network:** use a separately isolated execution network with approved addresses, ports, rates and protocols. An HTTP gateway cannot enforce raw TCP/UDP scope. Start by qualifying unprivileged connection-based probes; raw sockets/NSE scripts need a separate review. Do not add host networking or general privileges to the existing worker.
- **VM/MCP:** Kubernetes can coordinate an executor outside its container network. Each remote operation still needs a run identity, narrow permission, expiry, evidence and Stop enforcement at the actual execution boundary. An MCP connection is not itself a sandbox. Avoid a generic remote shell as the first interface.
- **Mobile static:** parse customer-provided artifacts in a bounded sandbox without target-network access. Treat APKs and analysis output as untrusted; control extraction paths, parser resources and evidence access.
- **Mobile dynamic:** use a dedicated device/emulator bridge with device/package binding. Android virtualization and iOS device needs differ. We cannot promise these run inside the current gVisor container. A privileged device bridge must be separate from the untrusted agent and from other customers.
- **Reporting/crypto:** offline, limited to approved engagement artifacts. No target credentials or network capability. Review input/output paths, untrusted markup, CSV formula risks and sensitive data before using existing utilities.

Some scripts attempt login guessing, high-rate scans, state-changing actions, metadata/OOB requests or device protection bypasses. They need explicit engagement permission and reviewed defaults. A scope-approved HTTP request can still cause harmful application-side effects; the gateway alone cannot promise non-destructive testing. Use staging/synthetic data and appropriately limited customer accounts for these workflows.

One concrete issue in the local toolkit: its authentication test disables TLS verification. That behavior will not be inherited silently by an adapter. Its scanner defaults and severity suggestions also need review rather than automatic adoption.

## Data and coordination decisions

The service owns scope, approvals, budgets and task state. Agent prompts propose work; they do not grant authority. Discovered hosts and instructions inside tool output do not become approved targets. Handoffs reference tenant-bound artifacts instead of giving every agent a shared writable engagement directory.

The toolkit's `authorized` boolean and evidence `path` are export fields, not authorization or filesystem permissions. Its role document mentions validation labels that differ from the finding schema's status enum. Keep internal validation/review state separate and use an explicit export adapter. Do not invent CVSS or copy a checklist's default severity into a customer finding.

Real evidence can include sessions, credentials and personal data. Redaction, storage permissions, provider data handling and customer approval must precede sending it to an external model. Short-lived credentials still expose whatever a compromised recipient is permitted to read. Current exports and models use synthetic data only.

The current engagement allowance bounds child runs, not account-wide spend. Production work needs durable monetary reservations, unknown-cost reconciliation, distributed rate limits and scheduling fairness. Redis rate limits and usage metering do not replace a durable budget ledger. The user's free-only model restriction remains in force for development; no paid fallback is authorized.

## Open inputs

Toolkit redistribution terms; the VM/MCP interfaces used by the friend; an owned or customer-approved application; which mobile platforms/devices are required first; deployment infrastructure and an approved load-test budget; customer identity, evidence retention and model-data requirements. These are inputs to implementation and qualification, not details to invent.

Stage 10 now has one reviewed curl read adapter; see [its execution boundary](tool-execution.md). More web/API profiles still need individual contracts, qualification and security-engineer review. VM/MCP and offline mobile foundations in stage 11 can follow, while general shell, scanners and additional agent roles stay disabled until their controls and tests exist. The repository should only be described as customer-ready after the relevant gates above have evidence.
