# D9: qualified browser journey and five-minute dry run

Recorded 2026-11-09 on the dedicated owned `kind-tsunami-lab` cluster. This record qualifies the D1–D8 product slice on the build below. The presentation is an **automated paced browser dry run**, not recorded human narration. All six assessments used scripted Mecatl and automated owned-lab approvals, with zero external model calls or human finding reviews.

Start with [qualification.json](qualification.json), [browser results and actual timings](browser/summary.json), [build identities](build.json), [before](before.json)/[after](after.json) preservation records and [final readiness](final-prerequisites.json). The presenter follows [the exact five-minute script](../../docs/demo-walkthrough.md#five-minute-presentation); the operating contract is [demo-configuration.md](../../docs/demo-configuration.md).

## Selected build

- Base commit `b0ede2a9357aa58566a99ff502b2aa0d44cff71e`, plus uncommitted D1–D8 source and D9 qualification tooling. This is a fingerprinted working tree, not a clean-commit reproduction.
- Native/control image `tsunami:showcase-20261109t144510z`; ID `sha256:2581f5408e99616b93209799c4af6adee37e0108b87bd2ad684b91a3ffb687f1`.
- Tools image `tsunami-tools:showcase-20261109t144510z`; ID `sha256:b771f2c236fb42a4e68b8485b354db24714973cd00255c5892f927b90f9263ae`.
- Both inspected, unstarted images contain binary SHA-256 `013faa595eac50ccd2fa45e0e87285c5e4fce84314cf29224ee68d91066c0c1b`, matching the compiled CLI. The onboarded worker's CRI image reference matches the tools image.
- All four served UI assets matched source bytes; hashes are recorded in both build/browser records. [source.json](source.json) fingerprints the build-time tree. The later qualification driver's exact hash is recorded separately in `build.json` and the browser summary; tooling/documentation continued after the image build.

Initial readiness found expired VAmPI credentials. Explicit preparation rebuilt/deployed matching images and rotated only disposable VAmPI/property/Stop fixture data and credentials. It preserved the operator Secret, onboarding key, state PVC, unrelated control settings, and every original history ID. Old scopes were not rebound. The previous [D2 record](../demo-configuration/README.md) remains unchanged.

## Actual journey and timing

| Scheduled stage | Observed completion from presentation start |
|---|---:|
| 0:00 problem/architecture | 0.086 s |
| 0:30 Add Application, four-read preflight, scope approval, actual worker/finding | 45.378 s |
| 1:30 finding, selected owner/probe artifacts and remediation | 90.629 s |
| 2:30 active-request Stop during pending viewer export | 156.350 s |
| 3:20 changed-origin linked retest and lost Start recovery | 206.241 s |
| 4:15 limitations and preserved original finding | 255.498 s |

The clock continued to **300.000 seconds**. Every stage finished before its deadline; intervals leave time for the supplied narration. These timings do not measure a human presenter or establish service latency. Preparation, runtime/network checks and the following property controls are outside the five-minute presentation.

| Assessment | Observed outcome | Evidence |
|---|---|---|
| Newly onboarded owned VAmPI | `confirmed`, one unrated candidate | [Bundle](browser/onboarded-vampi/bundle.json) |
| Separate held-request fixture | `stopped`, no completed validation | [Bundle](browser/stop-lab-run/bundle.json) |
| Linked fixed VAmPI | `no_issue_observed`, fresh scope/approval | [Comparison](browser/linked-retest.json), [bundle](browser/linked-vampi-fixed/bundle.json) |
| Separate property leak | `confirmed`, one unrated candidate | [Bundle](browser/property-vulnerable/bundle.json) |
| Separate property filtering | `no_issue_observed` | [Bundle](browser/property-filtered/bundle.json) |
| Separate ambiguous property response | `inconclusive` | [Bundle](browser/property-ambiguous/bundle.json) |

The onboarded scope used four approved legitimate preflight reads, followed by a separately approved seven-read assessment. Actual CRI inspection observed `runsc` / `io.containerd.runsc.v1`. Persisted activity matched the displayed sequences and contained an actual rejected off-scope intent. Normal finding navigation fetched metadata only; explicit owner/probe reads and downloads checked exact artifact bytes. Original candidate/artifact identities remained intact after the linked retest. A deliberately hidden committed Start response was recovered by reading its stored run, with one Start POST and no replay.

For Stop, target counters independently changed from five requests, one active probe, zero cancellations to the same five requests, no active probe and one cancellation. Probe-completion count remained zero. Durable revocation, local drain and API resource absence were observed separately; these are not equivalent to process-exit proof during a partition. Holding a real export response was a controlled client failure injection.

Six selected screenshots remain here: [scope review](browser/onboarded-scope-review.png), [finding inspection](browser/finding-selected-proof.png), [held request](browser/held-before-stop.png), [linked comparison](browser/linked-retest-results.png), the actual activity screenshot indexed in the browser directory, and [narrow layout](browser/mobile.png). Other original screenshots remain in the private campaign named in `qualification.json`. They are actual UI captures, not mockups.

## Runtime, network, preservation and integrity

[Runtime inspection](isolation/runtime.json) and [native TCP/UDP DNS checks](isolation/gvisor/network.json) passed, including denied direct paths with reachable trusted before/after controls. [Curl checks](curl-network/network.json) passed all 54 selected literal-IP/`--resolve`/`--connect-to` attempts across startup/settled/repeat phases. The actual tools diagnostic pods used gVisor and the selected image; positive controls reached those destinations before and after. Broker HTTP remained reachable.

All original **345 runs, 86 engagements and 12 setups** remained. The campaign ended at **351 runs, 92 engagements and 13 setups**, with no nonterminal run and **eight engagement slots remaining** at the recorded observation. Reserve capacity before repeating campaigns or presenting; never delete history or reset Stop gates to free slots. Each live Add → retest → held-Stop presentation consumes three fresh engagements. The full six-assessment verifier consumes six.

[Source verification](source-verification.log) covers Go race tests, vet, Python regressions and archived/materialized evidence integrity. D9 adds six bundles/46 artifacts to the fixed gate, for 38 bundles/286 artifacts plus the original 20 trial hashes. The D9 checker also rejects late rehearsal claims, changed worker builds, foreign source links and missing original history. External toolkit-schema validation was not requested.

Verify the retained bytes without contacting targets:

```sh
sha256sum --check examples/demo-journey/SHA256SUMS
python3 -B scripts/check-examples.py
```

Assigned operator/target/onboarding-key bytes and base64 forms were compared in memory against selected files and decoded artifacts, with no matches. This is a specific comparison, not general redaction. Raw synthetic proof values remain intentionally present. No Secret dump, credential file, encryption key, kubeconfig or browser profile is retained.

Readiness and unsigned JWT expiry hints are valid only at their recorded timestamps. Recheck before showing the demo; refresh expired fixtures explicitly and qualify changed builds/bindings. This single-node, selected-path result establishes neither autonomous discovery, general hostile-process containment, all network protocols, customer isolation, scale nor a production Stop guarantee. Human procedure/impact review and customer secret/evidence lifecycle remain unfinished.
