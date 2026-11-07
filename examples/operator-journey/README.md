# Compiled operator CLI campaign

Recorded on 2026-11-05 at 14:02 UTC in the dedicated `kind-tsunami-lab` cluster. This campaign ran the actual `tsunami assess` binary against the existing APIs and real Kubernetes worker Jobs. It used owned synthetic data and scripted Mecatl, with zero external model calls. It is not a customer assessment or an autonomous discovery result.

| Application | Observed result | Target request reservations | Evidence artifacts | Finding candidates / reviewed findings |
|---|---|---:|---:|---:|
| Owned VAmPI vulnerable variant | `confirmed` | 7 | 8 | 1 / 0 |
| Owned VAmPI fixed variant | `no_issue_observed` | 7 | 8 | 0 / 0 |
| Held-request fixture | Permission revoked after read-only watch timed out; target observed cancellation | 5 | 6 | 0 / 0 |

Each worker also submitted the existing scripted off-scope intent. The broker rejected it without contacting that destination; its blocked capture explains the extra evidence artifact. The CLI does not select an expected finding. Only this trusted qualification driver knows the expected behavior of the owned vulnerable/fixed controls.

For each application the compiled CLI listed configured applications, created a draft, showed its scope, refused launch before approval, refused an incorrect scope digest, approved the actual digest, launched one scripted attempt, observed status, stopped the engagement and exported a new private directory. Launch after engagement Stop was refused. No finding review, severity score or CVSS vector was supplied or invented.

The Stop case independently checked the target's held/cancelled counters. A one-second watch timeout left permission active; explicit engagement Stop cancelled the held request. The target saw no additional requests through cleanup. CRI inspection of this held worker confirmed `runsc` / `io.containerd.runsc.v1` and the recorded tools image. The two fast VAmPI workers used the selected executor, but this campaign did not independently retain their CRI inspection. All three cases reached API Job/Pod resource absence; that alone is not proof of process exit on a partitioned node. The earlier [Stop campaign](../stop-lifecycle/README.md) separately records CRI exit under its stated failure conditions.

Builds:

- Native image `tsunami:journey-v2`: `sha256:d6167d4848636712a890358a20b2d6bee63789c6f6a165e7fed1249bfc73a1d6`.
- Tools image `tsunami-tools:journey-v2`: `sha256:1d32641d77d7f946358dca070655111a4cf47f0b5d35cfd724c8f5d8abe935e3`.
- Compiled operator binary SHA-256: `89d31bc34721db8e9db3ef44ebeb6aa7f20cb2493cfa5d7091e4380b6de702f9`.
- VAmPI source remains pinned at `f16052dce83f05847133ec98f01c5193a41de7d8`; owned image `sha256:8ffa773d316c0e2095e428118c856662815a4aae1e497d1385d20d7d3eb3213a`. Setup refreshed only its disposable accounts and data; other tenant/application plans were preserved.
- The dedicated single-node setup uses Kubernetes v1.34.0, Cilium 1.20.4 and gVisor `release-20261026.0` with systrap. The normal kubeconfig context remained `kind-kind`.
- Source is the operator-journey working tree after baseline `985d57a9a9df3eb0876b3a01c2d56e4b81e2b913`. The actual image/binary identities above identify the tested build; the baseline commit alone does not contain this milestone.

Files:

- [summary.json](summary.json): observed outcomes, IDs, status/cleanup facts, held-target counters and build identities.
- [transcript.txt](transcript.txt): actual command output, including scope previews, approval errors, monitoring and export messages. Common token-file/connection arguments are omitted from command labels; no credential values or target response bodies are printed.
- [vampi-vulnerable](vampi-vulnerable/report-draft.md), [vampi-fixed](vampi-fixed/report-draft.md), [stop-lab-run](stop-lab-run/report-draft.md): materialized CLI exports with linked synthetic evidence. The stopped attempt is incomplete, not an authorization pass.
- [native-verification.txt](native-verification.txt): fresh Go race tests, vet, thirteen Python tests and seven-bundle integrity checks.
- [toolkit-verification.txt](toolkit-verification.txt): all seven retained bundles checked against the actual separately obtained local toolkit schemas, including materialized artifact relations. Empty reviewed-finding lists in these three exports are intentional.

All 39 generated campaign files, including decoded inline artifacts, were checked against current lab credential values before copying. No credential matches were found. Historical examples and live-model trial hashes remain unchanged. Native tests additionally cover expired-session inconclusive results, a committed start with its response lost, cross-tenant denial, invalid target input, private exports and watch/Stop behavior.

This result establishes a configured operator workflow. It does not add URL-only onboarding, discovery, new vulnerability classes, arbitrary shell, general curl compatibility, customer identity, private application connectors, a connected UI or production load/Stop guarantees. See [the guide and recovery rules](../../docs/operator-cli.md).
