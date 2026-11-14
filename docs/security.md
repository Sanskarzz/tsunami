# Current trust boundaries

This document describes enforced boundaries and trusted assumptions. [Current functionality](current-status.md), [verification](verification.md) and [security reporting](../SECURITY.md) are separate references. The default path supports owned synthetic data only.

## Trusted components

The operator API, broker/validator/model-gateway process, its persistent database, Kubernetes control plane, private connector and its database, and lab fixtures. The worker binary is known code; model output and target content are untrusted. We do not yet claim complete containment of a hostile or compromised worker process.

The local operator API maps random bearer credentials to two demo tenants. Run credentials are distinct and checked against the requested run. Worker-supplied tenant IDs and targets are rejected. Workers and trusted control services still share the original namespace; optional private apps/connectors have separate customer namespaces. These checks do not establish a complete tenant isolation platform.

## Broker enforcement

- Only configured fixture or owned-application origins and explicit account/path pairs are accepted.
- Hostname, scheme, port, userinfo, fragments, query strings, and encoded alternative paths are checked before resolving.
- Operator-configured fixture DNS answers are pinned at startup for this lab. This bootstrap is not authorization of arbitrary customer DNS.
- The chosen dial address must be approved, including IPv4-mapped IPv6 checks. The dial uses that numeric address rather than resolving again.
- Redirects are inspected manually and checked before another request.
- Environment proxies, arbitrary headers, and worker-supplied Authorization/Host overrides are not used.
- Per-run action, concurrency, request-count, and time bounds apply. General per-customer spend/rate enforcement is future work.
- Request contexts are tracked independently of worker cooperation and cancelled on durable stop.
- Fresh transport state is used per operation; no cookies or upstream connection pools are shared between runs.

Synthetic fixture account credentials stay at the broker. Operator and run credentials are not sent to the target or written into reports. The database is a local prototype store, not an encrypted external secret manager.

## Optional model boundary

The default remains scripted. Explicit `execution: live` uses a run-authenticated gateway; its configured backend may be the offline `fixture` or a Responses service. The operator owns provider/model/endpoint selection. The worker supplies conversation data only, and remote media/resource references are rejected. The gateway supplies the fixed system prompt and two function schemas.

Provider keys are mounted only into the trusted control-plane pod, loaded at startup, and sent only to the configured upstream HTTPS endpoint. SDK retries are disabled; actual upstream HTTP attempts pass a durable quota reservation. Redirects and environment proxies are refused. Incomplete streams release no tool calls. Model responses cannot create a finding or expand HTTP scope.

Per-call diagnostics store gateway-owned timings, counters, and closed outcome codes, not upstream text, tool arguments, headers, or request IDs. A zero-valued adapter usage chunk can mean missing accounting; it stays uncertain. Pending records after a crash have unknown outcomes. The local failure harness is not evidence about an external provider's availability or billing. See [the diagnostic limits](inference-diagnostics.md).

An agent compromise can consume its run's remaining model allowance or produce useless results. It cannot select another model route, use another run's token, or reach the model service directly through the tested runner paths. Call, byte, and time caps are not monetary or distributed spend limits. Closing an in-flight model request is not proof that a provider stopped billing.

One actual external free-model trial has been evaluated using synthetic data. Earlier model-path evidence uses a local Responses fixture. See [configuration and limits](model-path.md) and [the separate free-model record](../examples/free-model/README.md).

OpenRouter has a dedicated `openrouter-free` profile. It requires the official endpoint and an explicit `:free` model. A last-hop transport guard sets zero price ceilings, disables fallback and hosted tools/plugins, and checks before reserving/forwarding an attempt. Free-only evaluators require the same price policy atomically when creating each run; a concurrent operator reconfiguration cannot silently route them through the generic profile. OpenRouter is trusted to honor those controls. Account metadata reported $0 usage after this trial; this is not a vendor billing audit or a dollar-budget implementation for paid providers.

## What is not enforced yet

The original `kind-kind` network probe failed: a fresh runner reached the unrestricted tripwire directly at startup. After a delay the target and DNS paths were blocked, but node API and Kubernetes API Service connections remained possible. An earlier apparent pass was masked by fixture ingress policy and did not establish runner egress enforcement. The observed behavior is consistent with policy startup timing and local-node exceptions; the probe logs are the evidence, not a claim to have proved the exact CNI implementation cause.

The separate `tsunami-lab` cluster uses Cilium's `always` enforcement mode and gVisor. Positive controls bracket runner checks of target Service/pod TCP, node API TCP, Kubernetes API Service TCP, and TCP/UDP DNS. The tested runner paths are denied while broker HTTP succeeds. This is evidence for these paths and this setup, not every address or protocol. Broad allow rules for trusted lab roles are deliberately outside the runner policy; runners have no API permission to change their labels or policy.

gVisor is installed only inside the dedicated Kind node. A diagnostic checks a running pod's CRI handler/type, no mounted API token, denied raw socket creation, and denied root filesystem writes. The actual assessment Jobs also select gVisor. Non-root containers, read-only filesystems, dropped capabilities, and Pod Security admission remain useful but do not establish escape resistance. The trusted services and worker share one node and one namespace in this lab.

IPv6, metadata routes, arbitrary UDP/ICMP, alternate interfaces, CNI failure, and cross-node behavior remain unqualified. The trusted broker is still the HTTP scope boundary. A broker compromise or a sandbox/network escape is outside the demonstrated guarantee. See [the sandbox experiment](sandbox-lab.md) and [the retained original failure logs](../examples/isolation/README.md).

The default scratch image contains no shell. The optional [reviewed curl adapter](tool-execution.md) uses a separate gVisor image, one fixed command and the existing broker boundary. No general shell tool is registered. Installing tooling or setting a proxy variable alone does not enforce scope.

## Stop semantics

Run state and disabled tenant/global gates survive control-plane restart. Workload creation is separate from permission. Stale Job reconciliation can create a pod that immediately fails its run authorization; it must not reopen a stopped run.

The control plane and broker are co-located and consult the same store. A Kubernetes API outage can delay cleanup without preventing that process from cancelling its broker requests. An unavailable store denies new actions. The optional private connector enforces single-request grant expiry, but renewable session leases and an external emergency network cutoff are not implemented.

Only the durable pending-to-running transition grants an initial Job creation attempt. Later reconciliation can adopt that Job but cannot recreate a missing one. A crash between persisting the dispatch intent and creating the Job leaves an incomplete run rather than replaying uncertain actions. A new run requires an explicit operator request.

Successful Kubernetes deletion requests are recorded once instead of resending them for every historical run on every tick. `deletion_requested_at` means the API accepted cleanup, not that a process on a disconnected node has exited.

The operator [lifecycle status endpoint](stop-lifecycle.md) separates durable permission, observed execution, active/uncertain operations and cleanup. Unfinished operation records survive process replacement without claiming a drain. Optional startup configuration failures disable their execution route while leaving the state/revocation API available. This does not provide availability after state-store or broker failure.

The ten-second operation timeout must not be described as a distributed permission lease. The private lab adds a separate signed request expiry at the connector; its native tests cover expiry without a control-plane callback. A disconnected real customer-side connector and cross-node partitions are not qualified.

## Private connector boundary

Private run creation is bound to the authenticated tenant and requires an explicitly configured connector; there is no public/direct fallback. The current private route supports scripted lab assessments only. The broker validates scope and reserves quota before issuing a grant, and pins the run's target URL. The connector verifies the tenant/audience/signature/expiry, rechecks its own URL/address policy, and consumes the request ID durably before dialing. Redirects are not followed at the connector; a next hop requires another broker scope check and grant.

The signing key is mounted only in the broker. The connector receives a public key and a separate emergency credential. Cutoff persists before acknowledgement and cancels active requests; replay and cutoff state survive restart. No request grant is sent to workers. The connector cannot independently query durable run state: stale valid grants can remain usable until expiry. Correct clocks and trusted infrastructure remain assumptions.

The broker-to-connector link now uses TLS 1.3 with required client certificates and exact tenant/role URI identities. The broker verifies the connector CA, DNS name, and URI; the connector separates the broker role from emergency operator role. Grant signatures remain the per-request authorization. Plaintext fallback is refused. Per-request transport checks and execution deadlines also bound work by certificate validity; a handshake alone is insufficient for an existing connection.

The CA key and emergency TLS key stay in ignored local files. Only component leaf credentials are mounted in Kubernetes. Explicit renewal does not revoke old valid leaves, does not hot-reload active processes, and can interrupt assessments. Fast revocation and CA rollover remain unfinished. See [certificate lifecycle and test scope](connector-tls.md).

Other internal links remain HTTP with synthetic data only. A compromised connector can access its own allowed private apps and falsify returned evidence before encrypting it. Customer-specific network policy limits its tested reach into other namespaces and API paths, but does not make it safe for real credentials. The emergency cutoff is independent of the broker, not of a compromised connector or unavailable Kubernetes management path. See [the exact private-lab boundaries](private-connectivity.md).

## Procedure boundary

New catalog runs accept only a known procedure ID, not custom steps or caller-supplied definitions. Definition/digest, normative access rule, and six-request allowance are pinned in durable state. The broker installs the procedure's exact paths under the existing origin/address approval and checks each account/path pair. It does not broaden legacy runs. The connector rechecks the signed procedure version/digest and pair before opening a private connection.

The `anonymous` alias is available only to the procedure that approves it and causes the target Authorization header to be omitted. It cannot be used to read another procedure's paths or supply arbitrary headers. Evidence is still broker-owned, and the validator checks known synthetic bodies and legitimate baselines. Shared access is not automatically a finding; fabricated-ID rejection and ambiguous bodies remain inconclusive.

When prerequisite captures are missing, completion can return up to two persisted feedback responses containing only approved reads. It grants no new account/path capability, does not execute those reads, and does not reset budgets or deadlines. After the cap or spent target budget, incomplete completion remains inconclusive. Stop revokes a run waiting for this feedback as it revokes any other run.

Catalog live inference is limited to offline `fixture` or `openrouter-free`; evaluation additionally requires the provider/price policy atomically when creating each run. External trials are opt-in and restricted to at most two procedures and one repetition. Model/provider attempt limits still apply. The initial engineering definitions need independent security review, and these guided fixtures do not establish general discovery quality. See [procedure and evaluation limits](assessment-catalog.md).

## Approval and Start recovery

Approval atomically compares the caller's reviewed `scope_digest` with the canonical immutable saved scope. Empty, wrong/stale and unauthorized approvals grant no permission. Start requires a tenant/engagement-bound action key; its request fingerprint and child/credential record commit together. Conflicting reuse fails. Known-action recovery may identify stopped/expired work after restart without renewing permission, deadlines, budgets or credentials. Recovery responses contain launch metadata/public scope, not captured bodies. No remote exactly-once execution guarantee follows. See [Start contract](engagements.md#start-recovery).

## Engagement boundary

An operator approves a pinned lab target/procedure selection before child-run creation. Parent reservation, child record and credential creation commit atomically. Engagement Stop revokes all active child states in one transaction and then cancels local contexts; restart cannot reopen the engagement. New scopes require new engagements. Current expiry/run/concurrency limits are fixed lab bounds, not customer-wide budgets.

Run tokens cannot read engagement evidence, approve scope, launch siblings, skip tests or change progress. Progress and candidates are recomputed from trusted broker captures; skipped and incomplete work do not become completed coverage. Exports contain synthetic evidence only. Content hashes are not evidence signatures or an immutable audit store. The later review API permits explicit ratings and schema export with matching evidence; it does not invent ratings for unreviewed candidates. See [engagement scope and limitations](engagements.md).

## Configured application and review boundary

Application registration is trusted startup configuration, not a worker/API capability. Tenant-bound public plans pin the origin, exact addresses, account/path pairs, assertions and credential fingerprint. The broker supplies credentials; changing configuration invalidates older execution. The v2 profile requires identities and legitimate reads before the probe, then owner-object and peer-identity rechecks. It does not create a transactional snapshot or infer a customer's intended policy automatically.

Only a stopped engagement and matching candidate fingerprint can receive a review. Worker tokens cannot review findings; stale/repeated revisions are refused. The score is computed from an explicit CVSS 3.1 vector, not chosen by the model. Automated lab reviews are labeled and are not human signoff. The current tenant-level operator credential is not individual reviewer identity. JSON contracts, hashes and output guards do not replace evidence access controls, redaction, retention or an independent audit store. See [application limits](application-adapters.md).

## Tests

The offline suite exercises origin bypasses, path/port/scheme checks, DNS rebinding, numeric-address dialing, TLS verification, cross-tenant/run credential rejection, worker verdict forgery, quota limits, cancellation observed by the target, durable tenant/global gates, and the real Mecatl-driven lab workflow.

Kubernetes verification adds actual worker Jobs, mounted run identities, control-plane state on a PVC, and API-driven cancellation. It does not qualify host escape resistance, metadata/node paths, arbitrary network protocols, distributed failover, or a production stop objective.

## Local operator interface

The optional `/ui/` handler serves embedded assets only for loopback Host values and requires `TSUNAMI_ENABLE_UI=true` at startup. It provides no operator credential, reverse proxy, target registration or application login. The same API authentication/tenant checks apply. The browser holds the operator token in page memory, omits cookies/storage and uses fixed same-origin paths. The wrapper preserves shutdown admission and sets no-store; the UI CSP has no external script dependencies and evidence/Markdown are displayed as text. A compromised local browser, extension or workstation can still read memory or use operator authority. Stop remains independent of viewer requests; a lost mutation response is not automatically retried. See [actual scope and limits](local-ui.md) and [browser records](../examples/local-ui/README.md).

## Protected onboarding preflight

Authenticated operator setup APIs now hold durable typed facts/questions/revisions and an explicit bounded preflight grant. Secrets and raw preflight responses are encrypted with a control-only key and tenant/session/purpose binding; no token goes into conversation or worker configuration. The final assessment evidence retains its documented local lab storage limits. Preflight uses the existing validated-address/hostname/TLS policy, exact own-account GETs, no redirects and persistent reservations; setup/global Stop cancel it and late responses cannot grant readiness. Interrupted attempts stay uncertain, not automatically replayed. New public targets are HTTPS-only with address filtering and trusted infrastructure exclusions; private exceptions remain exact registered lab integrations. These are not complete customer identity, secret/evidence lifecycle, all-provider infrastructure exclusions or production failure/scale qualification. See [the implementation limits](application-onboarding.md).

## Release qualification and dependencies

Source builds require Go 1.27.4, fixing the previously reported GO-2026-6859 advisory. Existing lab images retain their separately recorded build versions; see [scanner findings](verification.md#security-scan-status). Passing race tests and module checksums do not establish vulnerability-free dependencies. Container packages, upstream target dependencies and the host CA bundle need their own release review. A source release does not qualify production deployment or grant third-party redistribution rights.
