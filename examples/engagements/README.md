# Engagement workflow records

Recorded on 2026-11-02. No external model calls or toolkit scripts were executed. The existing free provider configuration was preserved. These results demonstrate the engagement foundation with synthetic applications, not customer readiness, general discovery, full toolkit support or production scale.

| Flow | Procedure runs | Progress per app variant | Candidates: vulnerable / fixed |
|---|---|---|---|
| Local Mecatl + Responses fixture | 10 | 5 completed | 3 / 0 |
| Public Kubernetes Job `tsunami-engagement-cp7jz` | 10 | 5 completed | 3 / 0 |
| Private mTLS Job `tsunami-engagement-g4vk8` | 10 | 5 completed | 3 / 0 |

Each flow refused dispatch before approval and after engagement Stop. Completed means the selected procedure captured its baselines and probe; the intentional missing-object result still remains inconclusive. Candidates have no invented severity/CVSS. Human impact triage and final finding/evidence schema adapters are pending.

Records:

- [Local summary](offline-summary.json) and [local Responses log](offline-verification.log).
- Public [summary](kubernetes-public/summary.json), [log](kubernetes-public/verification.log), [vulnerable bundle](kubernetes-public/bundle.json), [draft report](kubernetes-public/report-draft.md), and [schema/hash validation](kubernetes-public/schema-validation.json).
- Private [summary](kubernetes-private/summary.json), [log](kubernetes-private/verification.log), [vulnerable bundle](kubernetes-private/bundle.json), [draft report](kubernetes-private/report-draft.md), and [schema/hash validation](kubernetes-private/schema-validation.json).
- [Full native Go test log](go-test-race.log), [additional engagement tests](engagement-tests.log), and [Python boundary tests](python-tests.log). `go vet ./...` also passed.

The Kubernetes collector retrieved the stopped snapshots: authorization is false, children remain linked, and earlier finding candidates/evidence are retained. Artifact bytes survive JSON formatting through base64 encoding and match their SHA-256 paths. The local engagement-schema checker used the friend's local file and installed `jsonschema` 4.19.2, with remote references disabled. Passing that one schema is not integration of all three toolkit contracts.

Native tests cover tenant/worker credential substitution, child limits under contention, failed-run reservation retention, racing Stop/creation, restart, target-observed active cancellation, policy drift not blocking Stop, skip restrictions, free-price policy, private-route binding, tenant gates and deadline clipping. The Kubernetes verifier checks dispatch refusal around completed campaigns; it does not measure active-request Stop latency under load.

[Toolkit inventory summary](toolkit-inventory-summary.json) records 44 scripts, 26 checklist files, three schemas, eight documented roles and four prompt files. Its source content was not copied into this repository. The full local inventory keeps hashes and checklist IDs under ignored artifacts; all script adapters remain marked unimplemented. Source hashes were unchanged afterward, and no license file was detected.

The Kubernetes image was `sha256:09d6b599beb8fd4476da210ee6b991bf0a9fa52542aef417a3cdbdd8b390581b`. Workers used the existing gVisor Job path. The default kubectl context stayed `kind-kind`; these deployments/checks used the dedicated `kind-tsunami-lab` context.

See [implementation/reproduction](../../docs/engagements.md) and [the customer-use roadmap](../../docs/product-roadmap.md). VM/MCP execution, network/mobile profiles, general application adapters, specialist coordination, customer infrastructure and capacity qualification are still required work.
