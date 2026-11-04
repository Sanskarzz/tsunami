# Observed local results

Recorded on 2026-11-01. These are development observations, not production guarantees.

## Original lab environment

- Local context: `kind-kind`, Kubernetes v1.34.0, single node.
- Namespace: `tsunami`.
- Kindnet image: `kindest/kindnetd:v20250512-df8de77b`.
- Mecatl: published engine v0.18.1, with a scripted provider.
- No gVisor RuntimeClass installed. No arbitrary shell tool enabled.

## Verification

`go test -race ./...`, `go vet ./...`, and Kubernetes server-side manifest validation passed.

The local workflow and Kubernetes Job workflow both completed:

| Scenario | Observation |
|---|---|
| Vulnerable fixture | Confirmed cross-user read; 3 forwarded requests and 1 scope denial |
| Fixed fixture | No issue observed for the tested known-owner reads |
| Missing object | Inconclusive, not a claimed authorization pass |
| Redirect | Unapproved redirect blocked; tripwire saw zero requests |
| Stop | Active request cancelled; request count did not grow afterward |

Kubernetes verification Job `tsunami-verify-q8mgt` completed successfully after resetting the tripwire counter used by the intentional bypass probe. Its observed stop-call-to-broker-drain time was 28 ms; the target separately observed cancellation. This is one local sample including client/polling overhead, not a 60-second service guarantee or a measurement of all target-side effects.

## Original isolation check

The first runner probe was blocked from a fixture that also had an ingress restriction. That result was insufficient: it did not independently prove runner egress enforcement.

The strengthened probe, Job `tsunami-network-probe-sfcv7`, reached the broker **and directly reached an ingress-unrestricted tripwire**. `make isolation-check` correctly returned failure. At that stage the cause had not been diagnosed. The later checks below separate startup behavior from persistent node access. The original cluster remains unqualified for untrusted execution.

The gVisor runtime prerequisite is also missing. A RuntimeClass object's existence alone would not prove the runtime actually works; full qualification is future work. Functional workflow passes do not override these failed isolation prerequisites.

The cluster CNI/runtime were not replaced or modified. The application was deployed only into its own namespace. No external customer applications or model providers were contacted by the testing workflow.

## Dedicated gVisor/Cilium lab

A separate Kind cluster, `kind-tsunami-lab`, was created without changing the original cluster's CNI/runtime or the user's normal kubeconfig context. It uses Kubernetes v1.34.0, Cilium 1.20.4 in `always` policy mode, and gVisor `release-20261026.0` with systrap. The node image and runtime archive checksum are pinned in the setup.

The improved probes use fresh connections at process startup and after two delays. Trusted positive controls run before and after the runner. They distinguish a denied path from a dead target. These observations were recorded:

| Path | Original Kindnet runner | Dedicated Cilium runner |
|---|---|---|
| Direct tripwire Service and pod TCP | Reachable at startup; timed out afterward | Timed out in all three phases |
| Kubernetes API Service TCP | Reachable in all phases | Timed out in all phases |
| Node API TCP | Reachable in all phases | Timed out in all phases |
| TCP and UDP DNS | Reachable at startup; timed out afterward | Timed out in all phases |
| Broker HTTP | Reachable | Reachable |

The dedicated cluster passed with both runc and gVisor runner probes. That separates tested CNI behavior from the runtime choice. All matching trusted positive controls succeeded. The original failure remains a failure, not a result erased by switching clusters.

The gVisor runtime diagnostic checked actual CRI handler `runsc` and type `io.containerd.runsc.v1`, not only the RuntimeClass object. API token access, raw socket creation, and root filesystem writes were denied. The recorded runtime probe Job was `tsunami-runtime-probe-nsw4t`; the gVisor network runner Job was `tsunami-network-runner-mrbzs`.

The same five assessment scenarios passed with gVisor workers in Job `tsunami-verify-n28r7`. Its observed stop-call-to-broker-drain time was 40 ms; target cancellation was separately observed. This is another local sample, not a production stop objective or a performance comparison with runc.

See [recorded runtime/network output](../examples/isolation/README.md), [gVisor assessment reports](../examples/sandbox-kubernetes/README.md), and [reproduction steps](sandbox-lab.md).

These checks do not qualify IPv6, metadata routes, arbitrary UDP/ICMP, alternate interfaces, cross-node traffic, sandbox escapes, CNI failure, or production multi-tenant isolation. At this stage shell and live-model execution were disabled. The next section records the optional model-path work.

The laptop hit an fsnotify limit while following some Job logs. The verifier now waits for Job completion and reads logs afterward, avoiding a truncated log stream being mistaken for a complete result. No host sysctl settings were changed.

## Optional model gateway

Implemented and tested on 2026-11-01. The provider path uses Mecatl's published Responses adapter v0.3.0 with engine v0.18.1. All inference tests used local deterministic API fixtures. No real provider key or paid model call was used.

Kubernetes evaluation Job `tsunami-model-eval-z28rk` passed the vulnerable/fixed pair using gVisor worker Jobs. Both runs made 5 gateway calls and 5 upstream HTTP attempts. Each recorded 3 forwarded target requests and a denied unapproved-origin attempt. The fixture emitted synthetic usage of 200 input / 100 output tokens per run; those are test data, not tokenizer measurements or real billing.

The evaluator exported the exact run IDs into [recorded model-path examples](../examples/model-path/README.md). The vulnerable result was `confirmed`; the fixed result was `no_issue_observed`. These deterministic results do not measure live-model accuracy or autonomous discovery.

The original five-case suite also passed in Job `tsunami-verify-rcm54` while the model fixture was configured. Those runs remained scripted. Its observed target stop/drain sample was 27 ms, not a production guarantee.

The expanded gVisor network probe, Job `tsunami-network-runner-zbjl5`, also denied direct access to the new model fixture Service at startup and after settling. Trusted before/after probes could reach it. Existing target/API/DNS denials and broker access still passed.

Native race-enabled tests exercise provider redirects, untrusted TLS certificates, truncated streams, SDK retry disabling, generic error responses, explicit live opt-in, cross-run credential rejection, quota persistence across restart, retained reservations after failure, model-route pinning, remote media rejection, and active inference cancellation on run/tenant/global stop or deadline. Stop also succeeds while both model-body admission slots are held by incomplete requests. No Kubernetes reconciler is needed for these cancellation tests.

The real-provider configuration branch was smoke-tested with a synthetic unused key and the placeholder `https://model.invalid/v1/` endpoint, without creating a model run. The key was mounted as a Secret in the control plane rather than an environment value. Switching to fixture mode removed that Secret. This is configuration evidence, not proof of live-provider compatibility.

Reconfiguration with unchanged settings was checked to restart the control-plane pod, so a Secret-only key change can be loaded at process startup. A subsequent application deployment retained the explicitly configured fixture. The paid evaluation target refused to run without its explicit opt-in flag.

At that point real model behavior and useful output were unverified. The next section records the subsequent opt-in trial; monetary budgets, vendor compute cancellation, and production failure behavior remain unverified. See [the model-path notes](model-path.md).

## Actual OpenRouter free-model trial

The user selected OpenRouter and required free models only. On 2026-11-01, `nvidia/nemotron-3.5-lightning:free` passed a connection check, one vulnerable/fixed assessment pair, and a separate active-inference stop test. This used the published Mecatl Responses adapter and gVisor worker Jobs, not the offline API fixture.

The dedicated `openrouter-free` transport profile pins the endpoint and explicit `:free` model, disables fallback/hosted features, and sends zero prompt/completion/request price ceilings. The evaluators also require that policy atomically at run creation. The provider key stays in a local ignored file and the control-plane Secret, outside worker pods and exported reports.

| Check | Recorded observation |
|---|---|
| Pair Job `tsunami-model-eval-khk9p` | Vulnerable `confirmed`; fixed `no_issue_observed` |
| Each assessment | 5 completed model calls / 5 wire attempts; 4 approved target requests |
| Vulnerable reported tokens | 5,679 input / 228 output |
| Fixed reported tokens | 5,537 input / 216 output |
| Stop Job `tsunami-model-eval-wwlmj` | Active provider stream headers observed; durable stop; broker drain 29 ms; no counter growth for one second |
| Stopped inference | 1 attempt, 0 completed calls, uncertain usage, 0 target requests |
| Account metadata after trial | Usage and daily usage $0; free requests used 12 of 50 |

The 12 requests include the single connection check, 10 assessment calls, and one stopped call. These are vendor-reported usage observations, not a billing audit. The stopped call's zero recorded tokens means missing usage, not proof that no vendor work happened.

The model made no out-of-scope attempt in this trial. Scope denial is demonstrated by the scripted/offline-fixture suites and native tests, not inferred from the live model behaving correctly. Each live assessment tested both cross-user directions; the trusted validator used the expected baselines and Bob-to-Alice response.

The five-case scripted suite also passed with the real provider configured, Job `tsunami-verify-knplq`; it did not opt into inference. Its observed target stop/drain sample was 28 ms.

A subsequent isolation check added a fixed-dependency public TCP probe. A trusted operator resolved `openrouter.ai` to `104.18.3.115`; gVisor runner Job `tsunami-network-runner-ljxm6` timed out on TCP/443 at startup, settled, and repeat phases. Both matching trusted controls connected. Existing target, API, model-fixture, TCP/UDP DNS denials and broker access passed too. This tests one IPv4 dependency address, not all public egress, DNS answers, or IPv6. The actual CRI gVisor smoke check also passed again.

The assessment/stop image was `sha256:88f13c5619aea7e1913ebe306bdef9846f6d252571998530739a3e777d3be215`. The later diagnostic image was `sha256:098a2336d64d0e7fcbceff57893453dc1e13d9aca7dcf46682aaf61c6767f5e7`; neither model inference nor assessment behavior changed for that diagnostic. Native race tests and `go vet` passed for that source.

See [original reports, logs, account snapshots, and network output](../examples/free-model/README.md), and [free-only controls/reproduction](openrouter-free.md). This is one guided synthetic trial, not autonomous discovery, measured accuracy, general hostile-worker containment, or a 60-second production stop guarantee.

## Private connector and failure cases

Implemented and tested on 2026-11-01, without model calls. A connector in `tsunami-customer-a` opens requests to explicitly configured private fixture addresses. The broker in `tsunami` holds the signing key; the connector has its verification key and a separate emergency credential. The customer namespace has its own replay/cutoff database PVC. `tsunami-customer-b` supplies a separate private fixture for negative tests.

Job `tsunami-private-verify-n882m` passed all five scenarios through gVisor worker Jobs: vulnerable confirmed, fixed no issue observed, missing object inconclusive, redirect denied, and active-request Stop. The Stop sample was 29 ms from operator request through broker drain; the private target separately observed cancellation. The tripwire received zero assessment requests.

The same verifier script confirmed that Customer B cannot create a run using Customer A's private route. It then scaled the connector to zero and created another private run. No target requests arrived during that outage, no successful response was captured, and the connector was restored afterward. This is a controlled outage, not an arbitrary network partition.

The stable runner network probe was `tsunami-network-runner-npv45`. Direct private app, connector, Customer B, and unrestricted-ingress private tripwire TCP paths timed out at startup, settled, and repeat checks, alongside existing API/DNS denials. The broker was reachable, and trusted before/after controls connected to all listed destinations.

An earlier probe, `tsunami-network-runner-ghfbj`, overlapped the connector outage. Its connector connection was refused rather than timed out. The harness returned failure because this was unproven, and that record is retained separately. The later healthy-endpoint run supplies the qualified result; the earlier result was not counted as a pass.

Connector-identity probe `tsunami-connector-probe-2fkvp` ran in Customer A's namespace under the actual connector network policy, without credentials. Its own private app's health endpoint worked; Customer B, Kubernetes API Service, and node API connections timed out in all phases. Trusted probes in `tsunami` bracketed it and reached those destinations. This establishes selected paths and one local cluster, not complete cross-tenant containment.

Native race tests cover signed tenant/audience/target/account/lifetime checks; replay rejection after reopening the connector database; expiry while reading an already-open HTTP response body without any control-plane callback; emergency cutoff cancelling active traffic and surviving restart; run/tenant/global Stop; out-of-scope redirects; and no direct fallback during connector outage. Expiry/cutoff/replay failure tests have not been repeated across separate Kubernetes nodes. The main Kubernetes connector's emergency cutoff remains unlatched.

The public five-case regression suite also passed after private setup. The application image was `sha256:d7cc9d71a46b9095b2e7c047762c0e476d6ce2ff432f35c16215a5b2f4298f69`. Race-enabled tests and `go vet` passed. Source and examples were checked for actual provider/operator/signing/emergency/run credential values before sharing.

See [reports, original logs, and network records](../examples/private-connectivity/README.md) and [design and trust assumptions](private-connectivity.md). Internal HTTP supports synthetic data only. Authenticated TLS, real customer-side connectivity, full partition testing, and an independent external network cutoff remain future work.

## Connector mTLS and certificate renewal

Recorded on 2026-11-02 local time; machine records use UTC. Broker-to-connector transport now uses TLS 1.3, required client certificates, exact tenant/role URI SANs, and normal CA/DNS verification. Signed request grants remain required. The worker/broker and connector/fixture links still use synthetic HTTP.

Private workflow Job `tsunami-private-verify-zd82v` passed all five cases over mTLS, plus tenant binding and a controlled connector outage with no private target traffic. Its observed Stop/drain sample was 28 ms, with cancellation separately observed at the private fixture.

All leaf certificates were then explicitly renewed under the same offline CA. Public fingerprints changed for broker, connector, and emergency operator leaves. Grant-signing/emergency bearer credentials and both persistent-volume identities stayed unchanged. Job `tsunami-private-verify-pps7h` passed the complete private/outage suite again afterward; its Stop/drain sample was 27 ms. These samples are not a performance comparison or a production objective.

The deployed-endpoint diagnostic passed 12 checks both before and after renewal: authenticated broker health and operator status, broker denial from the emergency role, operator denial from the grant path, missing client certificate, untrusted server CA, incorrect server hostname/audience, obsolete TLS version, plaintext refusal, and successful authenticated checks afterward. The diagnostic did not send target requests or latch cutoff. Connection refusal/timeouts are treated as unproven, not accepted as TLS rejection.

Native race-enabled tests additionally cover wrong-tenant and expired client certificates, expiry while reading an active target response, expiry checked on a later request over the same TLS connection, identity/grant separation, replay protection across restart, emergency-cutoff persistence, and all Stop levels. Native renewal tests check explicit versus implicit renewal and private-file permissions. TLS/certificate-failure tests are local HTTP/TLS tests, not multi-node partition qualification.

Setup was also invoked from temporary local directories with missing and mismatched CA files. Both attempts were refused with the specific CA-continuity error; the installed broker TLS Secret stayed unchanged. Actual CA files were not moved/deleted. The original public five-case regression suite passed and remained scripted. No model calls were made.

The preliminary Job `tsunami-private-verify-kjbdc` passed its five assessment cases, but its outage driver waited on a retained connector diagnostic pod and failed. The driver now selects only `tsunami-instance=main`. Only the later complete runs above count as passes; preliminary reports remain in local artifacts.

The assessment image was `sha256:8256ae1c3e57801de3eada4bf36bbcc4037b6657611203f6f90eb498ba23059e`; the read-only transport diagnostic ran from the local CLI. The main connector is available with cutoff unlatched. Race tests, `go vet`, public-manifest validation, and checks for actual private key/credential leakage passed.

See [reports, transport checks, expiry tests, and public renewal metadata](../examples/connector-tls/README.md), and [lifecycle limitations](connector-tls.md). Manual renewal does not revoke an old valid leaf. Other internal TLS links, real private tunnels, automated identity/key lifecycle, external network cutoff, and broader failure testing remain unfinished.

## Versioned authorization catalog

Recorded on 2026-11-02. Five original procedures cover opaque object ownership, explicitly shared access, administrator role boundaries, anonymous reads, and a missing-object negative control. They are embedded definitions with pinned run digests, exact account/path permissions, and a six-request allowance. Their security-engineer review status is pending. The fixture implementation uses independent data; it does not generate responses from the catalog's expectations.

The local scripted and local Responses-fixture suites each passed 20/20 cases across two repetitions. Final Kubernetes public Job `tsunami-catalog-6sl52` and private-connector Job `tsunami-catalog-z57l6` each passed 20/20. Each suite recorded six confirmed planted cases, ten expected no-issue observations, four intentional missing-object inconclusive results, zero false positives/missed planted cases, and complete required evidence. Scripted/offline scores test integration and validation, not model accuracy.

The first actual free-model catalog Job was `tsunami-catalog-8gz87`, using `nvidia/nemotron-3.5-lightning:free`. Its opaque/shared pair sample matched 3/4 expected results. The vulnerable opaque read returned the protected object, but the model skipped both legitimate baselines and finished. The validator returned inconclusive and the evaluator counted incomplete evidence/a missed planted case. No false finding was promoted.

The completion path then gained up to two persisted missing-evidence feedback responses. It returns only already-approved reads, never executes them, and does not widen scope or increase budgets/deadlines. Native tests reproduce premature model completion through the actual Responses adapter and Mecatl engine, verify recovery, and check the cap and Stop behavior across restart.

Follow-up free Job `tsunami-catalog-chj44` matched 2/4 cases. Both approved-sharing cases used one feedback response and recovered to complete evidence/no issue observed. The two opaque cases failed during their first model turn, with no completed call or target request. Their events record a generic `model_error` near the existing model-call window; exact upstream cause is unconfirmed. Missing usage stays uncertain. There were no false positives. No additional trial was run to obtain a passing score.

The vendor metadata snapshot reported $0 usage and daily usage, with 28 free requests used across the initial/follow-up trials. The zero-price transport guard and per-run attempt caps remained active. These observations are not a billing audit or a claim of model reliability. The application images were `sha256:7a38744ec38b16371b3db7dabbc26ea6befe100e1b471515a5b378325bb396a3` initially and `sha256:b33ed07d43043a9e4f0fb82224df0e6d100b71f92a1c9104bdec70946174a270` after feedback was added.

The original private five-case/Stop/outage suite also passed, Job `tsunami-private-verify-pg66t`. Native race tests and `go vet` passed, including procedure isolation, anonymous-header omission, private signed-digest checks, ambiguous HTTP 200 bodies, paid-provider refusal, bounded feedback, and feedback recovery. Existing report footers and sandbox notes were corrected to reflect the implemented lab connector and live-model path; historical example files were retained unchanged.

See [summaries, selected original reports, failed live trials, test logs, and vendor snapshots](../examples/assessment-catalog/README.md), [procedure rules](assessment-catalog.md), and [toolkit provenance](toolkit-mapping.md). The catalog is a guided synthetic evaluation; arbitrary customer configuration, private model execution, general scanners, and security-engineer-reviewed production procedures remain unfinished.

## Inference failure diagnostics

Recorded on 2026-11-02, without external model calls. New calls persist gateway-owned status, timing, attempts, buffered/released tool counts, usage state, and closed error codes. Raw upstream text, headers, tool arguments, and provider IDs are excluded from this diagnostic record. Initial reservations and observed HTTP 200 headers persist before completion; a crash can leave a pending call with an unknown outcome.

The local fault campaign passed 14/14 cases through the actual Responses adapter and gateway. Cases cover normal completion, omitted usage, HTTP 401/429/503, absent headers, headers without a completed turn, Stop after headers, a truncated tool turn, output exhaustion, unknown incomplete response, an in-stream rate limit, invalid usage, and an oversized reply. Each made one upstream fixture attempt and zero target requests. All 12 failure cases returned no tools or partial reply. Stall/Stop fixtures observed connection cancellation. Synthetic deadlines belong only to the harness; deployed call/run deadlines are unchanged.

The adapter can emit zero usage even when the upstream omitted usage. The omitted-usage success now retains uncertain accounting rather than claiming verified zero usage. Native tests cover persistence across reopening the database, bounded diagnostic codes/private canaries, classification after a recovered call, and archive immutability, including linked output paths. Race-enabled tests and `go vet` passed.

Offline analysis of the archived free trials keeps 3/4 and 2/4 matched. The initial trial has one incomplete assessment; the follow-up has two `model_failure_unclassified` outcomes. Each still has one raw missed planted case. Those old records cannot identify the exact failure cause, and no rerun was used to replace them. The analyzer does not revalidate old evidence or edit source records.

The local Responses-fixture catalog passed 10/10. Public Kubernetes Job `tsunami-catalog-25klx` and private mTLS Job `tsunami-catalog-jzwlq` each passed 20/20, with zero operational failures and complete required evidence. Both remained scripted even though the free provider was still configured. These scores test integration, not model accuracy. The deployed image was `sha256:b46d89e07e88d7a76b31913d1fda0aeebb2f5f1970311f13413a21c8e9a7e2a4`; the subsequent offline analyzer output-link safeguard was tested natively.

See [logs, diagnostics, archived analyses, and example report](../examples/inference-diagnostics/README.md), and [meaning and limits of the fields](inference-diagnostics.md). Header/first-adapter-chunk timings are not provider compute time. Usage acceptance is not billing verification, and closing a request does not prove vendor compute stopped. No retry, scope, quota, or timeout expansion was added.

## Engagement foundation and toolkit inventory

Recorded on 2026-11-02. The new engagement API pins a configured lab target, procedure digests, approval and expiry. Run/credential creation and the parent reservation share a transaction. Engagement Stop atomically revokes nonterminal children and cancels local contexts. Progress and finding candidates derive from broker captures; they are not accepted from agent completion messages.

The local Responses-fixture campaign completed ten procedure runs: five each on vulnerable/fixed apps. It produced three unrated candidates on the vulnerable app and none on the fixed app, with 20 exported capture artifacts per variant. Draft and after-Stop dispatch were refused. The export hash checks and local engagement-schema validation passed. Final finding/evidence schema adapters and human impact triage are still pending.

Kubernetes public Job `tsunami-engagement-cp7jz` and private mTLS Job `tsunami-engagement-g4vk8` each passed the same ten-run scripted flow through gVisor workers. Each had five completed procedures per variant and three versus zero candidates. The collected stopped snapshots showed `authorized: false`, matching hashes, and valid engagement-schema projections. No external model calls were made; the free provider configuration was preserved. These are integration checks, not independent discovery or scale measurements.

Native race tests additionally cover cross-tenant/worker-token rejection, competing run reservations, retained failed-run allowance, Stop racing child creation, restart persistence, target-observed active cancellation, changed policy not blocking Stop, skipped-test restrictions, and exported-byte preservation after JSON reformatting. Race-enabled Go tests and `go vet` passed. Python tests cover inventory as data, symlink/duplicate-JSON refusal and remote schema-reference rejection; a later collector test covers refusal to forward the operator header on redirects.

The toolkit inventory found 44 scripts, 26 checklist files, three schemas, eight documented roles and four role-prompt files. It records hashes/identifiers without running scripts or copying their contents. It does not establish script safety, normative checklist completeness, licensing permission, or implemented adapters. No license file was detected. The local schema checker used installed `jsonschema` 4.19.2 and recorded the schema hash; no remote schema references were fetched.

The Kubernetes image was `sha256:09d6b599beb8fd4476da210ee6b991bf0a9fa52542aef417a3cdbdd8b390581b`. The default kubectl context remains `kind-kind`; deployment and checks used the dedicated `kind-tsunami-lab` context. See [engagement records](../examples/engagements/README.md), [implementation limits](engagements.md), and [the expanded customer-use roadmap](product-roadmap.md).

## Configured application and reviewed exports

Recorded on 2026-11-03 local time (UTC records on 2026-11-02). A configured, tenant-bound HTTP profile now supports identities, exact account/path pairs, JSON-pointer assertions and proof hashes without editing fixture code. Credentials remain in the broker. Configuration changes invalidate older execution. The profile is restricted to owned synthetic data and scripted execution; private configured-application connectors and live-model data handling are not qualified.

VAmPI source `f16052dce83f05847133ec98f01c5193a41de7d8` was built and deployed in `tsunami-app-lab`, with debugging disabled, restricted ingress and no egress. Trusted setup created synthetic accounts and books. Job `tsunami-application-j22b5` ran gVisor Mecatl workers: the vulnerable variant returned `confirmed`, the fixed variant returned `no_issue_observed`. Each made seven approved target requests and captured one blocked out-of-scope attempt.

The v2 evidence sequence verifies both identities and legitimate objects before the probe, then owner access and peer identity afterward. A peer-session recheck was added after identifying that an expired token could otherwise look like an authorization denial. Native tests reproduce expiry, deletion, wrong identity, generic bodies, duplicate-key JSON, missing prerequisites and stale successful probes. V1 internal plans are refused, with original local records retained rather than reinterpreted.

The vulnerable case generated one explicit automated lab review to exercise contracts, with example CVSS 3.1 score 6.5/Medium; it is not human signoff or a customer severity assessment. The review API requires a stopped engagement, matching evidence fingerprint and optimistic revision. Worker/cross-tenant attempts, conflicting/stale reviews and fabricated numeric scores are refused. False-positive dispositions create no rated finding or zero-score placeholder.

Both exports passed the actual three local toolkit schemas and relational checks. Eight materialized artifact files per variant were verified, plus the vulnerable finding file and seven evidence metadata files. CSV common-formula guards and Markdown evidence/metadata guards have native tests. Seven Python tests cover local-only references, content substitution, evidence/index linkage, partial mappings and collector credential redirects. Race-enabled Go tests and `go vet` passed.

The partial checklist adapter imported ten API reference IDs and mapped API1 to the sampled procedure. Fully completed standard items remain zero; no runtime permissions changed. This is an export-side projection, not automated execution of a standard or a compliance result. Toolkit source remains local and redistribution terms remain unresolved.

Network runner Job `tsunami-network-runner-qhn8f` denied direct TCP access to both VAmPI Services and an ingress-unrestricted tripwire in the new namespace at startup, settled and repeat checks. Trusted before/after probes connected; broker access and existing API/model/DNS boundary checks also passed. This qualifies selected IPv4 paths only. Public catalog Job `tsunami-catalog-kftln` and private mTLS Job `tsunami-catalog-8fvgd` each retained 20/20 matched outcomes.

The application image was `sha256:8ffa773d316c0e2095e428118c856662815a4aae1e497d1385d20d7d3eb3213a`; the final Tsunami image was `sha256:dd9e373f202379acff22c5fee49c062f8e626ec57fdd0f3f618881ae9b3003ad`. No external model calls were made; configured free routing, previous live-trial archives and the default kubectl context were preserved. See [materialized records](../examples/application-adapters/README.md) and [application assumptions](application-adapters.md). No production capacity, customer-readiness or autonomous discovery claim is made.

The later native-tested historical-report guard was deployed in image `sha256:f502357874f61f5ec013332b7a0fcd21e06f934336497746b5b47416fbf89713`. That restart failed closed because the short-lived lab connector certificate had expired. Explicit leaf renewal under the existing CA restored the control plane; disposable VAmPI accounts were reseeded separately. Follow-up application Job `tsunami-application-p6rlp` again passed both expected outcomes with seven approved requests each, and the deployed connector transport checks passed without target/model requests. The control plane is available. This records manual recovery, not an automatic renewal or production availability guarantee. Original qualification files remain unchanged.


## First reviewed shell/curl adapter — 2026-11-03 UTC

Executor `curl-http/v1` adds a fixed reviewed script and real curl process to configured HTTP read plans, with a private Unix bridge and the same broker target enforcement. It is a command execution boundary prototype, not general shell or toolkit-script support and not a new vulnerability class. The control/default worker retains its scratch image; only approved tool Jobs use the separate gVisor image.

Application Job `tsunami-application-znls5` observed `confirmed` on owned VAmPI's vulnerable object-access implementation and `no_issue_observed` on its fixed variant, seven approved target requests each. A blocked scope attempt gives eight captures each. Both materialized bundles passed the actual three local toolkit schemas and relational/file checks. The one rated finding is explicitly `automated_lab_test`, not human approval or a customer assessment.

Curl runner Job `tsunami-curl-runner-26kdv` denied 54 selected IPv4/TCP attempts across six destinations, three connection modes (literal IP, `--resolve`, `--connect-to`) and startup/settled/repeat phases. Matching trusted before/after controls connected, and broker access remained reachable. Both sides used the same tools image and independently inspected CRI `runsc` / `io.containerd.runsc.v1`. Runtime smoke checks passed. The separate Go diagnostic also passed its TCP and UDP DNS checks, including configured application/private paths.

The initial positive-control Job `tsunami-curl-positive-before-bp9xk` failed because it overlapped a broker rollout; its first broker attempt was unproven. That record is retained. Infrastructure readiness is now checked before probe creation, without delaying a pod's first network attempt.

Native race tests and vet passed; seven Python tests passed. Actual curl tests cover input as data, strict bridge capabilities, ignored ambient config/proxy settings, output/invocation limits, timeout/ordinary-descendant cleanup, held-target Stop and subsequent denial, upstream TLS rejection and redirect isolation. Active curl Stop is not measured end-to-end in Kubernetes here. Hostile-session cleanup, unsupported HTTP modes, general shell, real customer data and production network/scale/failure qualification remain open. No external model calls were made. See [materialized records and image IDs](../examples/tool-execution/README.md) and [execution boundaries](tool-execution.md).
