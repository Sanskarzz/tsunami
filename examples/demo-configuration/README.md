# D2: selected demo build and fresh qualification

Recorded 2026-11-07. This qualifies the existing owned HTTP lab on the D1 validator build. It does not implement D3-D9's future UI behavior, a customer portal, general discovery, or production guarantees. Approvals were automated **owned-lab** qualification actions; no human finding reviews or external model calls were made.

Start with [qualification.json](qualification.json), [build identities](build.json), [source fingerprints](source.json), [initial prerequisites](prerequisites.json), and [final prerequisites](final-prerequisites.json). The reproducible operating contract is [demo-configuration.md](../../docs/demo-configuration.md).

## Build and preservation

- Base commit: `b0ede2a9357aa58566a99ff502b2aa0d44cff71e`, plus the uncommitted D1 validator/tests and documentation cleanup. This is a fingerprinted working-tree build, not clean-commit reproduction.
- Native/control image: `tsunami:showcase-20261107t093304z`, observed ID `sha256:100c336063b39d928a65b9516e995e0712ef298e47f9a79d345d7c5c4407a51f`.
- Tools image: `tsunami-tools:showcase-20261107t093304z`, observed worker ID `sha256:42d715d0f828322f673ecb2bc08e893039cd515e0c380831d71b92d9c50430d0`.
- Both images were inspected without starting their containers and contain binary SHA-256 `aad5d0b98d08cedc83334a71cbc85eece2348439e2da2d8bfa7e1825f48327bf`, matching the compiled CLI. Actual held workers and the onboarded worker matched the tools image through CRI.
- All four served UI assets were fetched independently and matched local bytes; their hashes are in `build.json`. The browser driver also checked deployed bytes before testing. The driver itself does not emit a separate build file; this dossier records independently observed build data.

The initial read-only prerequisite check failed because VAmPI credentials were expired or too near expiry. Explicit `make showcase-prepare` rebuilt/deployed the selected images and reset only disposable VAmPI, property and Stop data. [Preservation checks](preservation.json) confirm the original 334 run IDs, 75 engagement IDs and 9 setup IDs remained, along with the operator Secret, onboarding key, state PVC and unrelated control environment. There were no unrelated startup application entries in this ten-entry registry; that comparison is not an independent cross-tenant registry qualification.

Qualification added 11 runs/engagements and three setups. Final history contained 345 runs, 86 engagements and 12 setups; all original IDs remained. No history was deleted or Stop gate cleared. The 100-engagement lab ceiling leaves **14 slots** at this observation. Reserve capacity before D9 and the live presentation; a full seven-case showcase plus four-run browser campaign consumes eleven slots. Do not repeat campaigns until capacity is exhausted or silently wipe state to make room.

## Observed outcomes

| CLI application | Result | Candidates | Evidence |
|---|---|---:|---|
| `vampi-vulnerable` | `confirmed` | 1, unrated | [Bundle](showcase/object/vampi-vulnerable/bundle.json) |
| `vampi-fixed` | `no_issue_observed` | 0 | [Bundle](showcase/object/vampi-fixed/bundle.json) |
| `stop-lab-run` | Stopped during held probe | 0 | [Bundle](showcase/object/stop-lab-run/bundle.json) |
| `property-vulnerable` | `confirmed` | 1, unrated | [Bundle](showcase/property/property-vulnerable/bundle.json) |
| `property-filtered` | `no_issue_observed` | 0 | [Bundle](showcase/property/property-filtered/bundle.json) |
| `property-ambiguous` | `inconclusive` | 0 | [Bundle](showcase/property/property-ambiguous/bundle.json) |
| `property-stop` | Stopped during held probe | 0 | [Bundle](showcase/property/property-stop/bundle.json) |

Each completed assessment reserved seven target requests and captured one rejected off-scope intent. Both held fixtures independently observed active -> cancelled, unchanged probe-completion counts, no additional target requests after Stop, and API cleanup. Worker runtime/image observations are in the [object](showcase/object/summary.json) and [property](showcase/property/summary.json) summaries. API cleanup does not establish process exit on a partitioned node.

The [browser campaign](browser/summary.json) separately passed vulnerable/fixed/held-Stop controls and registered a new application through approved four-read preflight, fresh final approval and actual gVisor/Mecatl execution. It produced an unreviewed finding with six verified runtime prerequisites/rechecks and a blocked scope intent. See [onboarding snapshot](browser/onboarding-session.json), [new application evidence](browser/onboarded-vampi/bundle.json), [actual screenshot](browser/onboarded-finding.png), and [credential-free operator metadata](browser/operator-requests.json).

Browser controls include a lost committed Start response recovered without retry, duplicate-click prevention, Stop while a real export response was held, invalid-token/wrong-digest rejection, same-origin calls, safe rendering, and narrow layout. These are controlled client-side failure injections, not production partition qualification.

## Runtime and network observations

- [Actual runtime smoke test](isolation/runtime.json) observed `runsc` / `io.containerd.runsc.v1`, no API token, denied raw socket creation and denied root-filesystem writes.
- [Native network checks](isolation/gvisor/network.json) denied selected direct target, Kubernetes/node API, model-service and TCP/UDP DNS paths while broker HTTP remained reachable; trusted positive controls passed before/after.
- [Curl network checks](showcase/network/network.json) passed 54 selected denied IPv4/TCP attempts across literal IP, `--resolve` and `--connect-to` in startup/settled/repeat phases. Trusted before/after probes reached the same destinations. These actual tools pods used gVisor and the selected tools image.

This is a single-node, selected-path qualification. It does not establish IPv6/metadata coverage, all protocols, arbitrary hostile-shell containment, customer isolation, production Stop latency or capacity. Local drain, target cancellation, execution observations and API cleanup retain their distinct meanings.

## Verification and retention

[Fresh source verification](source-verification.log) passed Go race tests, vet, 20 Python tests and the existing gate's 32 historical bundles/240 artifacts plus 20 historical trial hashes. [Browser verification](browser-verification.log) passed. [Separate retained-record checks](retained-verification.log) reused the existing showcase/onboarding and artifact validators for these **11 fresh bundles/82 artifacts**. Those additional D2 bundles are separate from the existing gate's fixed historical list.

Check this selected record's bytes from the repository root:

```sh
sha256sum --check examples/demo-configuration/SHA256SUMS
```

The checksum list and artifact hashes identify bytes; they do not authenticate a compromised trusted service. Source fingerprints are captured inputs, not a requirement for every future source tree to match this historical build.

Exact assigned operator/target credential values, onboarding-key bytes and their base64 forms were compared in memory against selected files and decoded artifacts before retention; no matches were found. The screenshot was inspected after protected fields were cleared. This is a concrete comparison, not general redaction. Raw synthetic proof values are intentionally present in evidence. No Secret dump, token file, encryption key, kubeconfig, browser profile or entire artifacts tree is retained here.

Original private preparation/transcripts/screenshots remain in the local artifact paths recorded by `qualification.json`. No failed assessment campaign was converted into a pass. The earlier expired-credential prerequisite failure was resolved by explicit preparation, not hidden by lowered expiry thresholds or changed scope.

VAmPI credentials were issued for one hour. Prerequisite lifetime is an unsigned hint at its timestamp, not a continuing readiness claim. Recheck the owned lab before D9/presentation; if refresh/rebuild is needed, preserve history, create fresh approvals and qualify the replacement build. The UI entry point is loopback `/ui/`; the API explorer still shows tab-local operator calls, not D4's planned request timeline. Finding navigation and linked retest remain later tasks.
