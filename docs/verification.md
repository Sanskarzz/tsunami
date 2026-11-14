# Verification and evidence

Source tests, runtime/network diagnostics and assessment campaigns establish different facts. A passing evidence checker verifies retained bytes/indexes; it does not rerun those historical assessments. No result here proves production isolation, general discovery or a distributed Stop objective.

## Reproduce source checks

Install the [source prerequisites and Python requirements](getting-started.md#path-a--verify-the-source-locally), then run `make verify`, `make verify-ui-retest` and `make build`. The optional four local Chrome targets are documented there. These checks need no Kubernetes, provider key, customer data or persistent lab slots.

`make verify` runs Go race tests with `-count=1`, vet, Python tests and `scripts/check-examples.py`. The evidence gate checks 38 materialized bundles, 286 artifacts and 20 original trial hashes, plus retained browser/showcase indexes. Optional `make verify-toolkit TOOLKIT_PATH=...` additionally validates separately obtained local schemas with remote retrieval disabled. The toolkit is not downloaded, vendored or executed.

## Release preparation

The documentation cleanup starts from public commit `44d20cb63cde7d3c5f0fde20113404222f185d5d`. Before cleanup, a temporary clean clone passed Go race/vet, 23 Python tests, 38 frontend tests, build, module verification/tidy checks, four real local Chrome campaigns, both historical SHA manifests and Markdown links/anchors. A fresh Python venv also passed the source gate. It inherited toolchains/module caches, but no `.local/`, kubeconfig, credentials, workloads or databases. This was not a fresh OS or fresh Kubernetes bootstrap.

Post-cleanup checks used a separate clean clone at `4c29ffe6c8f7737855cb16dfb0483792166d0b88`, after documentation commit `bdade1b` and the isolated 17-line unused `Server.reserve` removal. The helper had no caller/interface/generated-code reference; the actual request path retains `reserveTarget`. No test was weakened or historical artifact changed.

| Check on 2026-11-14 | Observed result |
|---|---|
| Path A dependency setup | PASS: new `.local/test-venv`, requirements installed from the documented file; no copied local configuration |
| `KUBECONFIG=/dev/null make verify` | PASS: all Go race packages, vet, 23 Python tests and retained evidence gate (38 bundles / 286 artifacts / 20 trial hashes) |
| `make verify-ui-retest` | PASS: 38 frontend tests |
| Four local Chrome targets | PASS sequentially: activity/Stop, onboarding, findings/property controls, linked retest; real local backend/Mecatl, no Kubernetes |
| `make build` | PASS: clean source revision `4c29ffe`, Go 1.27.3, `vcs.modified=false`; binary SHA-256 `481d34d628ac7e70e04bcff27befd232a0e59dc6e3d58e98830b1ed14ac2b2c2` |
| `go mod verify`, `go mod tidy -diff` | PASS: checksums verified; no module-file changes |
| Markdown checks | PASS: 167 Markdown files, 587 local links and 33 anchors; no missing targets at this checkpoint |
| External archive | PASS: all 15 originals match source commit `44d20cb`, sizes and SHA-256; no overwrites |
| Original evidence | PASS: entire examples Git tree identical to `44d20cb`; original 171-entry and 125-entry SHA manifests pass |
| Fresh Kubernetes/runtime/network/assessment campaigns | NOT RUN; no lab adoption, new engagement, credential rotation, state reset or Stop-gate change |
| External toolkit schemas / provider inference | NOT RUN; optional external inputs are not needed for Path A |

Later verification-documentation edits do not change the qualified application source. The binary above is a source build, not a new deployed image or a byte-reproducibility guarantee. The clean clone reused installed system toolchains/module caches but created its own test venv/state. Local raw logs and redacted scanner reports are in a private temporary audit directory, not committed evidence. Only the reviewed summaries are public.

### Security scan status

Scans completed with default detections intact; no broad allowlist, skipped security test or edited historical result was used to obtain a clean appearance.

| Scanner / scope | Outcome |
|---|---|
| Gitleaks module v8.30.1, Git `--all`, redaction 100%, through `4c29ffe` | Raw exit 1: **308 generic-key detections**, all triaged. 113 public credential-binding hashes, 153 synthetic proof hashes, 2 key-data hashes, 38 source/artifact hashes and 2 natural-language matches. No credential value established. |
| Gitleaks tracked/new publication-tree snapshot | Raw exit 1: **307 detections**, same hash categories plus one prose match. Ignored runtime secrets/artifacts are intentionally excluded from publication, not copied into the scanned public tree. |
| Govulncheck v1.9.0, Go 1.27.3, symbol-level source scan | **FINDING:** GO-2026-6859 / CVE-2026-97054 in stdlib `net/http`, fixed in Go 1.27.4. One advisory with multiple call traces, not multiple independent defects. |
| Trivy v0.75.0 source manifest scan | No listed Go-module vulnerabilities; this does not include the standard-library result or prove vulnerability-free code. |
| pip-audit v2.10.1, resolved test requirements | No reported vulnerabilities across five resolved Python test packages. This is separate from VAmPI's dependencies. |
| Trivy native image, ID `53467dc1e2689f57791f9f84345954b4762ba89aa0e4015a9411a164a1b4ca5b` | One standard-library finding, CVE-2026-97054. |
| Trivy tools image, ID `3504e567e16465b625a38d8b0289e570ece62c4c205d45f862d19d384243582f` | 395 OS package/advisory entries (110 high, 133 medium, 147 low, 5 unknown), plus the Go finding. |
| Trivy VAmPI image, ID `8ffa773d316c0e2095e428118c856662815a4aae1e497d1385d20d7d3eb3213a` | 211 OS entries (51 high, 90 medium, 65 low, 5 unknown) and 33 Python entries (1 critical, 9 high, 19 medium, 4 low). The critical entry is PyJWT 2.6.0 / CVE-2026-102268, scanner fix version 2.14.0. |

The Go advisory describes an HTTP/1 MaxBytesReader header race when body reading and response-header writing occur concurrently. The project uses MaxBytesReader; the broad call-graph result is not a demonstrated exploit. The pinned toolchain remains unchanged pending a separately reviewed update. See the [Go vulnerability record](https://pkg.go.dev/vuln/GO-2026-6859).

Trivy's DB snapshot was updated 2026-11-14T07:01:20Z. Counts are package/advisory entries, including duplicates across related packages, not proven reachable exploits. Container reachability/impact has not been fully triaged. The existing images were scanned read-only, not rebuilt or deployed. VAmPI deliberately contains application defects, but that does not excuse or hide dependency findings. Keep it confined to the owned lab. Do not distribute these images as hardened release artifacts.

The source/module/test-dependency scans do not cover every host tool, Kind/Cilium/gVisor binary, upstream component or dependency-license obligation. Repeated scans can differ as databases change. SHA checks establish byte identity, not absence of secrets or vulnerabilities.

**Release hold:** the final project license/publication-rights decision, Go security update and container-finding disposition remain open. The unsafe legacy `make deploy` defaults remain documented; a separate proposed guard has not been applied. No clean security-release or fresh-cluster qualification is claimed.

## Recorded deployment and qualification

| Record | Actual scope and provenance |
|---|---|
| [Complete browser dossier](../examples/demo-journey/README.md) | 2026-11-09, scripted Mecatl, six real owned assessments and paced 300-second automated browser rehearsal. Runtime/network controls, onboarding, activity/rejection, finding/evidence, held Stop and linked fixed-app retest. Automated approvals, not human narration/review. Its working-tree source fingerprints and image identities are authoritative; do not relabel it a clean later build. |
| Stop-patch qualification | Build `tsunami[:tools]:showcase-20261109t182656z`, binary `fd9f5870a1f1d06106e42d1e77cd9243ffe03c427bfdcc41cd74eafffea187f0`, built from `92576c2` with a dirty documentation/tooling tree. Actual runtime/native/curl checks and a separate three-assessment 300.000881-second core rehearsal passed. Outcomes: confirmed, stopped, linked fixed no_issue_observed. Property controls passed locally; full Kubernetes property campaign was not repeated. [Original dated report](https://github.com/Sanskarzz/tsunami/blob/44d20cb63cde7d3c5f0fde20113404222f185d5d/docs/archive/stabilization-2026-11-09.md). |
| Reviewed approval / keyed Start deployment | Observed 2026-11-09T19:26:32Z. Clean source `81198026d8337361559df4e2a35aee6f8b3e31ef`; native/tools tags `showcase-20261109t191623z`. Both binaries SHA-256 `fba9a6660759cce42125da8fc4909b52b26aa06b68dc0a6fd1dfbc9f4f089533`. Actual runsc/native/curl checks and matching assets passed; zero assessments created. A fresh Kubernetes assessment/rehearsal was NOT RUN. Private records remain under ignored `artifacts/hardening/d10-d11/`; they are not included in a clone. [Original public summary](https://github.com/Sanskarzz/tsunami/blob/44d20cb63cde7d3c5f0fde20113404222f185d5d/docs/track.md#d10d11-matching-deployment). |

Pre-cleanup read-only inspection on 2026-11-14 found the recorded control image ID, both local image IDs, all four served assets and 123 source fingerprints matching. RuntimeClass/runsc/Cilium configuration remained present. The lab retained 354 runs, 95 engagements, 14 setups, five available slots and no nonterminal runs. VAmPI expiry hints failed; the optional private route was unavailable. These observations do not constitute authenticated credential checks, new CRI/network qualification or readiness for a live demo. No lab mutations were made.

Core rehearsal consumes three engagements. Full browser qualification creates six and requires seven free slots. The five-slot recorded lab cannot repeat the full campaign without explicit capacity maintenance. Never delete history, raise limits or clear gates to make qualification pass. Credentials/readiness are time-limited; source and historical records are not current account certification.

## Historical failures and corrections

### Original isolation failure

The original `kind-kind` runner reached the direct target at startup; node/Kubernetes API paths remained reachable after settling. An earlier apparent pass was masked by target ingress policy. The original runtime also lacked gVisor. This was a failed containment qualification, even though the functional workflow passed.

Keep [original runner logs and negative network result](../examples/isolation/README.md), [original Kubernetes samples](../examples/kubernetes/README.md) and [the original validation narrative](https://github.com/Sanskarzz/tsunami/blob/44d20cb63cde7d3c5f0fde20113404222f185d5d/docs/validation-results.md#original-isolation-check) accessible. The separate dedicated Cilium `always`/gVisor lab later passed selected paths with reachable before/after controls. It did not retroactively repair the earlier experiment or prove every protocol, address or escape boundary. [Sandbox decisions and limits](sandbox-lab.md) remain maintained.

### Previous correctness and reliability defects

| Earlier defect | Current correction and regression location |
|---|---|
| Operator redirect could forward authority | Redirect rejection and proxy-independent operator transport; [demo tests](../internal/demo/demo_test.go). |
| Rejected Kubernetes DELETE counted as accepted cleanup | Method-specific statuses and retryable cleanup errors; [scheduler tests](../internal/kube/scheduler_test.go). |
| Mixed application history broke legacy collection | Paged metadata filtering and exact-ID collection; [collector tests](../internal/demo/collect_test.go). |
| Legal inline export exceeded client read limit | Bounded larger export contract and explicit oversize errors; [demo tests](../internal/demo/demo_test.go). Maximum memory/load remains unqualified. |
| Selected image was not consistently deployed | Native/tool image rendering and preserved third-party images; [renderer tests](../scripts/test_deploy_images.py). |
| Later denial could erase earlier supported object exposure | Evaluate each exposure with ordered prerequisites/rechecks; preserve unresolved evidence; [application tests](../internal/platform/application_test.go). No historical result migration. |
| Auxiliary setup persistence failure skipped cancellation after group revocation | Cancel already-revoked assessment contexts before auxiliary setup transaction; preserve partial failure reporting; [group Stop tests](../internal/platform/group_stop_test.go). |

The [original review](https://github.com/Sanskarzz/tsunami/blob/44d20cb63cde7d3c5f0fde20113404222f185d5d/docs/review-report.md) and [follow-up](https://github.com/Sanskarzz/tsunami/blob/44d20cb63cde7d3c5f0fde20113404222f185d5d/docs/review-follow-up.md) remain accessible in preserved public history. Failed records were not converted into passes.

### Model and failure-test limits

The [catalog trials](../examples/assessment-catalog/README.md) include an initial 3/4 matched sample and a follow-up 2/4 sample with operational failures. The follow-up did not establish improved overall accuracy. Historical generic model errors retain unknown causes. [Inference diagnostics](inference-diagnostics.md) separate later controlled local failure observations from actual provider behavior. A historical free-model/account snapshot is not a current pricing/availability or billing guarantee.

The [Stop failure campaign](../examples/stop-lifecycle/README.md) includes controller API denial and restart after committed revocation. It does not cover all broker/store failures, disconnected nodes or uncertain acknowledgements. Revocation, target cancellation, CRI exit and API cleanup are distinct observations.

## Evidence preservation and historical navigation

All existing `examples/` bytes and paths remain unchanged, including failed checks, downloaded duplicates, screenshots, source inventories and SHA manifests. They are validation inputs, not disposable build output. Local raw artifacts and credentials are not publication candidates by default.

Fifteen historical documentation originals were copied from `44d20cb` to an external sibling archive before upstream consolidation. The external manifest records source commit, original relative path, SHA-256 and size; copied bytes were verified. The archive is outside Git and contains no bulk copy of ignored runtime data. Public equivalents preserve security decisions, failures and provenance; [the complete original document tree](https://github.com/Sanskarzz/tsunami/tree/44d20cb63cde7d3c5f0fde20113404222f185d5d/docs) remains available without history rewriting.

## Fresh-environment qualification

The complete [fresh Kubernetes bootstrap](getting-started.md#path-b--run-the-controlled-kubernetes-demo) is **NOT RUN** in this release preparation. Fixed cluster names prohibit silently adopting an existing lab from a clean checkout. It needs an explicitly approved separate environment. No one-command installation, fully reproducible container bytes or production isolation claim is made.
