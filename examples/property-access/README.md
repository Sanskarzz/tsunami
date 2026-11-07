# Property-access campaign

Recorded on 2026-11-06 at 05:12 UTC in the dedicated `kind-tsunami-lab` cluster. This ran the compiled operator CLI, real scripted Mecatl loop and fixed curl adapter against an original owned fixture. No external customer targets or model providers were used. Independent security-engineer review remains pending.

| Variant | Observed result | Actual target reads | Evidence artifacts | Finding candidates / reviewed findings |
|---|---|---:|---:|---:|
| `property-vulnerable` | `confirmed`: peer's allowed shared-record read included the known owner-only note | 7 | 8 | 1 / 0 |
| `property-filtered` | `no_issue_observed`: correct public object, selected private field absent | 7 | 8 | 0 / 0 |
| `property-ambiguous` | `inconclusive`: field remained present as `[redacted]`, which did not establish the approved omission rule | 7 | 8 | 0 / 0 |
| `property-stop` | Explicit engagement Stop cancelled the held peer request | 5 | 6 | 0 / 0 |

Each worker also sent the existing intentional off-scope intent. The broker rejected it without contacting that origin; its capture explains the extra artifact. All three completed cases verified two identities, both legitimate private reads, the public-object probe, owner continuity and peer-session continuity. Target-side counters independently confirmed the approved reads. A qualifying `passed` flag means the driver observed the expected control outcome; the ambiguous assessment remains inconclusive.

The CLI refused draft launch, wrong-digest approval and new launch after engagement Stop. Generated proof values were not printed in the CLI transcript. Their raw synthetic values are deliberately retained in protected evidence artifacts to make the leak reproducible; this is not a general redaction or production data-handling guarantee. No review or severity/CVSS decision was submitted.

For the held case, a read-only watch timeout left permission active. Explicit Stop then revoked permission, and the target recorded cancellation rather than completion. Its request count remained stable through API cleanup. Actual CRI inspection confirmed `runsc` / `io.containerd.runsc.v1` and the tools image below. The three fast workers did not get an independently retained CRI inspection. API Job/Pod resource absence in all four cases does not itself prove exit on a partitioned node. Earlier [Stop failure records](../stop-lifecycle/README.md) retain their own builds and limits.

Builds:

- Native image `tsunami:property-v4`: `sha256:903c37c78b01c49c412a7228cb34ad0876768425b8255618b7550812b98594ac`.
- Tools image `tsunami-tools:property-v4`: `sha256:3bb141f16d0be15d8cdf08d4134379eb128e0cb4f651633a239bb0145e8b9a99`.
- Compiled operator binary SHA-256: `0d03d0324bb6cf9138e4edc96b7a00917a94c7bfe2cd224e2dbc8e882829324b`.
- Dedicated single-node Kubernetes v1.34.0, Cilium 1.20.4 and gVisor `release-20261026.0` with systrap. The normal kubeconfig context stayed `kind-kind`.
- Source: the property-access working tree after baseline `985d57a9a9df3eb0876b3a01c2d56e4b81e2b913`. The image/binary identities identify the actual tested build; the baseline commit alone does not contain this milestone.

Files:

- [summary.json](summary.json): actual status, counts, runtime observations and cleanup facts.
- [transcript.txt](transcript.txt): real CLI scope previews, rejected actions, approval, start, watch, Stop and export output. Common connection/token-file arguments are omitted from labels.
- [Leaked-note report](property-vulnerable/report-draft.md), [filtered-note report](property-filtered/report-draft.md), [ambiguous report](property-ambiguous/report-draft.md), [stopped attempt](property-stop/report-draft.md): materialized bundles and linked synthetic evidence.
- [native-verification.txt](native-verification.txt): fresh Go race tests, vet, fourteen Python tests, eleven-bundle/84-artifact integrity checks and the twenty historical trial hashes.
- [toolkit-verification.txt](toolkit-verification.txt): actual local reference-schema validation and artifact relations for the retained bundles.

Before copying, all 52 generated campaign files, including decoded inline artifacts, were checked against current lab credential values. No credential matches were found. The temporary CLI operator-token file was removed by cleanup; the lab credentials/configuration remain in ignored private files and Kubernetes Secrets.

Native tests additionally cover copied private values under different JSON fields, duplicate/malformed JSON, wrong public identity, expired/denied sessions, missing follow-up evidence, unsupported policies, unchanged old profiles and property-specific reviewed-finding classification. Review tests are explicitly `automated_lab_test`; they do not rate this campaign's candidate.

This adds a separate read-only property-authorization capability, not complete API3 coverage, generic secret discovery, mass assignment, a new toolkit script adapter, general onboarding or customer readiness. Exact-string comparisons can miss transformed/encoded data; source selectors are limited to top-level members. See [the design, assumptions and reproduction steps](../../docs/property-access.md).

The final native regressions also prove that a later filtered reply cannot erase an earlier supported property leak. Exposure recorded before complete ordered prerequisites keeps a later filtered result inconclusive and retains the earlier evidence reference.
