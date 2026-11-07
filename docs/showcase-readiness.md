# Axel showcase: a bounded delivery plan

Updated: 2026-11-06.

There is already a working example to show: Tsunami tests an owned deployment of VAmPI, a pinned intentionally vulnerable open-source API. The vulnerable/fixed controls run through real Kubernetes Jobs, Mecatl and the controlled HTTP path, with captured evidence. That is a real application implementation in a controlled lab. It is not a new finding on an unknown customer application, a production deployment or autonomous discovery.

The two planned milestones for a controlled CLI showcase are now complete:

| Showcase milestone | State | Deliverable |
|---|---|---|
| S1: guided binding preparation | Implemented | Derive assertions, paths and proof hashes from supplied baseline facts; validate prerequisites; keep preparation separate from registration/approval; rerun owned examples. |
| S2: repeatable demo package and rehearsal | Implemented | Prerequisite/expiry checks, explicit owned-data refresh, core/extended commands, exact scenario/evidence index, presenter guide, recovery steps and a fresh seven-case rehearsal with network controls. |

**No further planned milestone is required for this scoped CLI showcase.** Use [the demo guide](showcase.md) and [retained rehearsal](../examples/showcase/README.md). This is not customer readiness or a guarantee of Axel's response. The optional [local operator UI](local-ui.md) is now connected to the existing lab APIs, with an [API-by-API presenter guide](demo-walkthrough.md). It adds no general onboarding or customer-readiness claim. The original 14 HTML product templates remain static previews.

## What the showcase should demonstrate

1. Explain the goal and trust boundaries: the untrusted agent proposes permitted reads, while the service owns scope, credentials, evidence and revocation.
2. Show a plan compiled from known owned-app facts. Review and explicitly approve its actual registered scope before execution.
3. Run the owned VAmPI vulnerable/fixed pair. Show the difference between a supported finding candidate and no issue observed for this test.
4. Show a rejected off-scope intent and link it to broker evidence. Use the recorded network/runtime qualifications with their original build/path limits.
5. Hold a request, end observation without pretending that stops it, then explicitly Stop. Show permission, target cancellation, execution observations and cleanup separately.
6. Optionally show property-level exposure, correct filtering and an inconclusive redacted response. Keep scripted execution and known prerequisites clearly labelled.
7. Open the evidence/report bundle. Explain that candidates require human impact/severity review; no invented rating or signoff.

The live path should be short. Detailed tests, failure records and the roadmap are supporting material, not a long tour through every feature. The demo guide states what is reset, what needs renewal, which build was rehearsed and how to recover without replaying uncertain work.

## Claims that remain outside this showcase

The showcase does not establish 1,000-agent capacity, production Stop within sixty seconds, arbitrary hostile-shell containment, all toolkit scripts/roles, VM/MCP or mobile compatibility, blind vulnerability discovery, general onboarding or real-customer readiness. Recorded controlled failure tests remain useful evidence for the conditions they actually exercise.

If “real example” means a genuine customer assessment, there is no honest fixed milestone count yet. That requires an actually authorized target, suitable test identities/data, independent security review, customer identity/isolation, protected secret/evidence lifecycle, operating/incident procedures and relevant failure qualification. A new real-world vulnerability also cannot be promised before testing.

The broader product roadmap remains in [track.md](track.md) and [product-roadmap.md](product-roadmap.md). The two showcase milestones do not replace its customer and execution-profile gates. The original take-home response remains a separate proposed design; implemented claims must come from the code and recorded results.
