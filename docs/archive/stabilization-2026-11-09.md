# Stabilization verification — 2026-11-09 UTC

Read this as a dated patch qualification, not permanent readiness or a replacement for the [full D9 dossier](../../examples/demo-journey/README.md). Current operation: [demo guide](../demo-walkthrough.md).

## Changes and regression evidence

`92576c2` fixes tenant/global Stop cancellation ordering. An auxiliary onboarding persistence error previously skipped cancellation after durable assessment revocation. Permanent held-target tests failed for both Stop variants before the fix and passed afterward. They also preserve error reporting, block later admission and check tenant/global boundaries. Normal run, engagement, tenant and global Stop tests and restart gates pass.

`37b8fd5` adds an explicitly labeled core-only verification mode; the full six-assessment campaign remains unchanged by default. Documentation cleanup is separate from these changes. No assessment feature, new agent capability or broader scope was added.

## Verified checks

| Check | Result |
|---|---|
| `make verify` | PASS: Go race tests, vet, 23 Python tests; 38 materialized bundles / 286 artifacts / 20 historical hashes. |
| `make verify-ui-retest` | PASS: 36 frontend tests. |
| Local activity, findings, retest and onboarding browser targets | PASS: real Mecatl/broker fixtures; findings includes property leak/filter/ambiguity. |
| Actual runtime/native network and fixed curl network checks | PASS: inspected runsc; selected startup/repeated direct-path denials with reachable before/after controls. |
| `make demo-rehearse-core` | PASS: three real assessments and six scheduled stages over 300.000881 seconds; automated browser, no recorded human narration. |
| D2 and D9 manifests | PASS: 171 and 125 entries; original evidence unchanged. |
| Markdown links/anchors and whitespace | PASS. |
| Fresh full six-assessment Kubernetes/property campaign | NOT RUN: preserve remaining lab capacity; local property controls rerun, historical D9 full campaign retained. |
| Fresh-cluster provisioning, customer isolation, broad failure/scale, external toolkit schemas | NOT RUN / not qualified by this patch. |

The first manual manifest invocation used the evidence directory rather than the repository root and could not resolve its repository-relative paths. Rerunning from the documented root passed; no evidence bytes changed. The original Stop fault reproduction and expired-credential preflight remain recorded failures, not converted into successful historical results.

## Selected build

Both images were explicitly rebuilt/deployed and their extracted `/tsunami` bytes matched the local binary. The binary records VCS revision `92576c2` with `vcs.modified=true` because documentation/rehearsal-tool edits were present. This is a fingerprinted working-tree build, not a clean final-commit binary. Go/source and driver fingerprints were rechecked after rehearsal.

| Identity | Value |
|---|---|
| Binary SHA-256 | `fd9f5870a1f1d06106e42d1e77cd9243ffe03c427bfdcc41cd74eafffea187f0` |
| Browser driver SHA-256 | `f0493f3b97010408538fd1c088bd2750712d4ce4916962a4b4a902a479efc63b` |
| native_image | `tsunami:showcase-20261109t182656z` / `sha256:242c5c31d0a5c863e03910711c85f41904ca34fab8ce197182912f69c3e977b5` |
| tools_image | `tsunami-tools:showcase-20261109t182656z` / `sha256:36396feceaf4b6f147df6c22bb4305a245415202559bb4166d87006100a596b3` |

## Actual core outcomes

| Application | Run | Observed outcome |
|---|---|---|
| onboard-54eaa360e20a97f469f5fa3ffdbe06d8 | `7a01cc08e3dd9dbf4f23573b0c943a4f` | `confirmed` |
| stop-lab-run | `896652e2f3dcaf21831ec6e2a9d1857b` | `stopped` |
| vampi-fixed | `b15ce5001bd7358f39889b9db1d538d9` | `no_issue_observed` |

| Stage | Actual start → finish (seconds) |
|---|---|
| Problem and architecture | 0.000 → 0.096 |
| Add application, scope review, approval and real agent | 30.000 → 45.152 |
| Finding, reproducible evidence and remediation | 90.000 → 90.633 |
| Stop during an active request and pending viewer export | 150.001 → 156.389 |
| Fresh linked fixed-app retest and lost Start recovery | 200.000 → 207.806 |
| Limitations and original history | 255.001 → 255.491 |

The driver checked real onboarding/preflight/scope approval, matching deployed assets, persisted activity/rejection, selected evidence bytes/hashes, remediation, target-observed active cancellation, fresh fixed-origin approval, lost-Start recovery and original-finding preservation. The core summary explicitly sets `includes_property_campaign=false` and uses a distinct qualification version. It is not full six-case D9 requalification.

## Readiness and retained local records

All 351 pre-existing run IDs, 92 engagement IDs and 13 setup IDs remain present. Three new assessments consumed three slots: 8 → 5. No run is pending/running at the final observation. No history deletion, capacity increase, key replacement or Stop-gate clearing was performed. Only disposable owned fixture data/credentials were refreshed. Credentials remain time-limited; recheck before presenting. Five slots permit one further three-assessment core presentation; another full campaign requires explicit capacity maintenance with preserved state.

Raw local records remain ignored/private, not bundled into this documentation change: `artifacts/stabilization/20261109/` (before/after, build fingerprints, runtime and network records), `artifacts/showcase/prepare-20261109T182656Z/`, and `artifacts/ui/browser-20261109T183109815Z` (browser summary, screenshots, events, actual downloads/materialized evidence). Test command logs are `/tmp/tsunami-stabilization-*.log`; local browser campaigns are dated `20261109T182633` / `20261109T182634`. This note summarizes those executed checks; the historical full D9 public dossier remains byte-identical.

Normal Stop still depends on the trusted API/store. Setup revocation can remain uncertain after its own persistence failure. The selected runtime/network paths do not establish general hostile-agent containment, production tenant isolation, remote undo or a latency/partition guarantee. D10/D11 remain open optional hardening.
