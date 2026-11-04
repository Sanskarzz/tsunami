# Reviewed curl adapter qualification

Recorded on 2026-11-03, UTC. This campaign uses scripted Mecatl, real `/bin/sh` and curl in gVisor workloads, the trusted broker, and an owned VAmPI deployment with synthetic accounts/data. It made no external model calls and executed no toolkit source scripts.

Application Job: `tsunami-application-znls5`. Executor: `curl-http/v1`. Approved plan: `http-object-access/v2`.

| Owned application | Result | Approved target requests | Candidates | Rated test exports |
|---|---|---|---|---|
| Vulnerable object-access implementation | `confirmed` | 7 | 1 | 1 |
| Fixed object-access implementation | `no_issue_observed` | 7 | 0 | 0 |

Each also has one deliberately blocked out-of-scope attempt, giving eight capture artifacts. The adapter submits URL/account JSON over a Unix socket; the broker supplies target authentication, enforces scope and captures responses. Curl output cannot confirm a finding. The rated export is explicitly `automated_lab_test`, with example CVSS 3.1 score 6.5/Medium; it is not human signoff or a customer impact assessment.

## Records

- [Campaign summary](summary.json) and [verification log](verification.log).
- Vulnerable [bundle](vampi-vulnerable/bundle.json), [draft report](vampi-vulnerable/report-draft.md) and [schema/file validation](vampi-vulnerable/schema-validation.json).
- Fixed [bundle](vampi-fixed/bundle.json), [draft report](vampi-fixed/report-draft.md) and [schema/file validation](vampi-fixed/schema-validation.json).
- [Image/script metadata](build.json) and [curl version](curl-version.txt). The default scratch image and tools image remain separate. The Debian base is pinned; apt package selection still depends on the build-time repository.
- [Race-enabled Go tests](go-test-race.log), [Python tests](python-tests.log) and [vet output](go-vet.log). Native tests include actual curl injection/configuration guards, strict bridge capabilities, output/invocation limits, timeouts, ordinary descendant cleanup, Stop and later-request denial, TLS verification and redirects.
- [Curl network record](curl-network/network.json), [runner observations](curl-network/runner.log), and trusted controls [before](curl-network/positive-before.log) / [after](curl-network/positive-after.log).
- [Separate Go network record](go-network/network.json) includes UDP DNS and the configured private/application paths. It does not prove a raw-network execution profile.

Both bundles validate against the three actual local toolkit schemas. The checker verifies eight materialized capture files per variant and the vulnerable finding plus seven reviewed evidence metadata files. Schema hashes are recorded without copying the source schemas.

## What the network test establishes

Runner Job `tsunami-curl-runner-26kdv` timed out on six selected endpoints using literal IPs, `--resolve` and `--connect-to`, at startup, settled and repeat phases: 54 denied connection attempts. They cover target Service/pod tripwires, an unrestricted-ingress application-namespace tripwire, Kubernetes API Service/node addresses and TCP DNS. The broker was reachable. Matching before/after controls connected using the same tools image and gVisor runtime. The harness independently inspected CRI: `runsc` / `io.containerd.runsc.v1`. Runtime smoke checks denied raw sockets/root writes and found no API token.

Credential-free diagnostics ignore the Kubernetes API certificate only to avoid confusing TLS rejection with network denial. The supported curl adapter and broker never disable upstream TLS verification.

The [initial failed control](initial-rollout-failure/network.json) and [log](initial-rollout-failure/positive-before.log) are retained. It started while I was rolling out broker configuration: its first broker attempt was unproven, while later broker attempts and all target controls connected. The harness now checks infrastructure rollout readiness before creating probes. It still tests each pod's first connection immediately, without a pod-side delay. The failed record is not a passing qualification result.

## Limits

The actual gVisor application campaign tests supported execution and finding validation. Active curl/broker/target cancellation is tested natively, not measured as a Kubernetes end-to-end Stop latency. Malicious processes that create new sessions, arbitrary shell execution, sandbox escapes, IPv6, metadata, alternate interfaces, CNI failure and cross-node partitions remain unqualified. No production 60-second Stop or measured agent-scale guarantee follows from this campaign.

The current adapter supports one configured GET workflow, not arbitrary scripts, headers/bodies, browser sessions, raw HTTP, scanners or the toolkit's VM/MCP/mobile execution. General customer data, redaction and independent procedure review remain open. See [design and reproduction](../../docs/tool-execution.md) and [the remaining product stages](../../docs/product-roadmap.md).
