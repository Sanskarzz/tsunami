# Retained owned-lab showcase

Recorded on 2026-11-06, starting at 09:09 UTC. Open [the result index](index.md), [exact run IDs](showcase.json), or the [five-minute presenter guide](../../docs/showcase.md). These are retained results, not a live customer assessment.

The extended driver passed seven cases through the compiled operator CLI, Kubernetes Jobs, the real scripted Mecatl engine, and the fixed curl/broker path. The two known vulnerable controls produced unreviewed candidates; the fixed/filtered controls returned `no_issue_observed`; redaction returned `inconclusive`. Both held-request controls were stopped. No severity rating or human review was added. Approvals were automated owned-lab actions. There were **zero external model calls**.

[Network records](network/network.json) and logs show 54 selected runner connection timeouts, with the same 54 destinations/modes/phases reachable in each trusted before/after control. The broker remained reachable. All three diagnostic workloads had actual CRI `runsc` / `io.containerd.runsc.v1` observations and passed the no-token, denied raw-socket and read-only-root smoke checks. This qualifies selected single-node IPv4/TCP paths, not every possible network route or sandbox escape.

The [object](object/summary.json) and [property](property/summary.json) summaries retain held-target counters before/after Stop, durable revocation, local drain, execution observations and API cleanup separately. Counters showed no extra target requests or completed held probes after cancellation during the observation window. Both held workers have actual CRI runtime/container image observations. Fast workers are not individually CRI-inspected by this driver. API object absence alone does not prove process exit. This is not a production 60-second Stop qualification or a promise that remote effects can be undone.

## Build and environment

- Dedicated context `kind-tsunami-lab`, one Kubernetes v1.34.0 Kind node; the normal `kind-kind` context was preserved.
- gVisor `release-20261026.0`, Cilium v1.20.4 with policy mode `always`. [Prerequisites](prerequisites.json) retain the configured image digest and exact owned target origins. These internal lab Services use the broker's configured application route, not the private-customer connector.
- Native tag `tsunami:showcase-20261106t090350z`; deployed control image ID `sha256:7ec58b2ec6dd87853c4fc812515ce26aa9d9a9526d91aac8de711002df6f4a51`.
- Tools tag `tsunami-tools:showcase-20261106t090350z`; actual diagnostic and held-worker image ID `sha256:d3ab4c2bd07ff0b655bef23e368221e58cc66d68529e4ac3081fa0a1c89b16f3`.
- Client binary SHA-256 `eb02258dfddb42472a9ce74ab62c1a29b4255098ae38cd5d84f3c6e2e362ae4b`, built from the working tree after baseline commit `985d57a9a9df3eb0876b3a01c2d56e4b81e2b913`. This is not an exact clean-commit reproduction claim.

The prerequisite check first refused expired VAmPI credentials; [the rejection](prerequisite-expiry.txt) is retained. Explicit preparation refreshed only disposable owned fixtures and preserved durable history, operator credentials and optional model/private configuration. The expired optional private route was not repaired or used for this rehearsal.

The first rehearsal completed the network and object cases but failed while indexing an omitted `blocked: false` field. Its private record remains `passed: false` under ignored local artifacts. After correcting the indexer, a **fresh** extended rehearsal passed; this directory contains that later run. A regression test covers the omitted field. Later packaging checks/helper extraction did not change the Go binary or target path.

## Sharing and checking

This directory is a copy of `artifacts/showcase/run-20261106T090905Z` plus this explanation, the expiry rejection and fresh [verification](verification.txt)/[schema-check](toolkit-validation.txt) logs. Transcripts keep their original local output paths; links in `index.md` resolve here. It contains raw synthetic evidence values deliberately, but comparison against current lab credential values, including decoded bundle artifacts, found no matches. Do not copy all of `.local/` or `artifacts/` into a share bundle.

`make verify` checks indexed IDs, verdicts, blocked captures, Stop counters, network controls, and all materialized evidence hashes. `make verify-toolkit TOOLKIT_PATH=../vapt-toolkit` also checks the separately obtained local schemas. Hashes detect changed bytes; they do not authenticate evidence against a compromised trusted broker. Toolkit source was not copied into Tsunami.

Follow [the showcase guide](../../docs/showcase.md) for a fresh run and explicit reset/recovery. Customer identity, secret/evidence lifecycle, independent procedure review, broader failure testing, scale and general onboarding remain separate product work. Static UI previews are not a connected application.
