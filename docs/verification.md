# Verification and evidence

Source tests, runtime/network diagnostics and assessment campaigns establish different facts. A passing evidence checker verifies retained bytes/indexes; it does not rerun those historical assessments. No result here proves production isolation, general discovery or a distributed Stop objective.

## Reproduce source checks

Install the [source prerequisites and Python requirements](getting-started.md#path-a--verify-the-source-locally), then run `make verify`, `make verify-ui-retest` and `make build`. The optional four local Chrome targets are documented there. These checks need no Kubernetes, provider key, customer data or persistent lab slots.

`make verify` runs Go race tests with `-count=1`, vet, Python tests and `scripts/check-examples.py`. The evidence gate checks 38 materialized bundles, 286 artifacts and 20 original trial hashes, plus retained browser/showcase indexes. Optional `make verify-toolkit TOOLKIT_PATH=...` additionally validates separately obtained local schemas with remote retrieval disabled. The toolkit is not downloaded, vendored or executed.

## Release preparation

The documentation cleanup starts from public commit `44d20cb63cde7d3c5f0fde20113404222f185d5d`. Before cleanup, a temporary clean clone passed Go race/vet, 23 Python tests, 38 frontend tests, build, module verification/tidy checks, four real local Chrome campaigns, both historical SHA manifests and Markdown links/anchors. A fresh Python venv also passed the source gate. It inherited toolchains/module caches, but no `.local/`, kubeconfig, credentials, workloads or databases. This was not a fresh OS or fresh Kubernetes bootstrap.

Post-cleanup release checks and dedicated scanner findings are recorded below after verification. No existing-lab assessment, credential rotation, state reset, Stop-gate change or cluster rebuild is authorized by source verification.

### Security scan status

An initial Govulncheck run reports Go 1.27.3 affected by **GO-2026-6859 / CVE-2026-97054**, fixed in Go 1.27.4. The advisory describes an HTTP/1 MaxBytesReader header race when body reading and response-header writing occur concurrently. The project uses MaxBytesReader; the broad call-graph report is not a demonstrated exploit. The pinned toolchain remains unchanged pending a separately reviewed update. See the [Go vulnerability record](https://pkg.go.dev/vuln/GO-2026-6859).

The initial Trivy source scan reports no listed Go-module vulnerabilities; it does not replace the Go standard-library finding or container scans. Dedicated secret-scanner detections require triage rather than suppression or a fabricated PASS. Final scanner versions, outcomes and limits belong in this section.

## Recorded deployment and qualification

| Record | Actual scope and provenance |
|---|---|
| [Complete browser dossier](../examples/demo-journey/README.md) | 2026-11-09, scripted Mecatl, six real owned assessments and paced 300-second automated browser rehearsal. Runtime/network controls, onboarding, activity/rejection, finding/evidence, held Stop and linked fixed-app retest. Automated approvals, not human narration/review. Its working-tree source fingerprints and image identities are authoritative; do not relabel it a clean later build. |
| Stop-patch qualification | Build `tsunami[:tools]:showcase-20261109t182656z`, binary `fd9f5870a1f1d06106e42d1e77cd9243ffe03c427bfdcc41cd74eafffea187f0`, built from `92576c2` with a dirty documentation/tooling tree. Actual runtime/native/curl checks and a separate three-assessment 300.000881-second core rehearsal passed. Outcomes: confirmed, stopped, linked fixed no_issue_observed. Property controls passed locally; full Kubernetes property campaign was not repeated. [Original dated report](https://github.com/Sanskarzz/tsunami/blob/44d20cb63cde7d3c5f0fde20113404222f185d5d/docs/archive/stabilization-2026-11-09.md). |
| Reviewed approval / keyed Start deployment | Observed 2026-11-09T19:26:32Z. Clean source `81198026d8337361559df4e2a35aee6f8b3e31ef`; native/tools tags `showcase-20261109t191623z`. Both binaries SHA-256 `fba9a6660759cce42125da8fc4909b52b26aa06b68dc0a6fd1dfbc9f4f089533`. Actual runsc/native/curl checks and matching assets passed; zero assessments created. A fresh Kubernetes assessment/rehearsal was NOT RUN. Private records remain under ignored `artifacts/hardening/d10-d11/`; they are not included in a clone. [Original public summary](https://github.com/Sanskarzz/tsunami/blob/44d20cb63cde7d3c5f0fde20113404222f185d5d/docs/track.md#d10d11-matching-deployment). |

Read-only inspection on 2026-11-14 found the recorded control image ID, both local image IDs, all four served assets and 123 source fingerprints matching. RuntimeClass/runsc/Cilium configuration remained present. The lab retained 354 runs, 95 engagements, 14 setups, five available slots and no nonterminal runs. VAmPI expiry hints failed; the optional private route was unavailable. These observations do not constitute authenticated credential checks, new CRI/network qualification or readiness for a live demo. No lab mutations were made.

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
