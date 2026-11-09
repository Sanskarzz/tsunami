# Tsunami: goal and scope

Updated: 2026-11-07.

**Immediate goal: deliver a polished five-minute SaaS-style demo of one complete, bounded HTTP authorization-assessment journey for experienced infrastructure and security engineers.**

The deployment remains a local owned-lab prototype. The demo must show real backend behavior and evidence. Building a complete VAPT platform or accepting real customer data is outside this delivery.

## Planning documents

- This document defines the goal, boundaries, and acceptance criteria.
- [track.md](track.md) records implementation status, known defects, and task completion.
- [roadmap.md](roadmap.md) defines ordered, independently committable tasks and their definitions of done.
- [design.md](design.md) maps the journey, backend states, and activity event contract.
- Operating guides and `examples/` retain their specific reproduction instructions, builds, and historical results.

These documents supersede the earlier broad execution-profile/stage roadmap. Deferred ideas are not scheduled commitments. Keep status in the tracker and task requirements in the roadmap rather than maintaining competing work orders.

## The journey to demonstrate

1. An operator adds an owned test application through the connected UI.
2. The system shows the exact target, addresses, account/action pairs, intended access rule, limits, and expiry.
3. The operator separately approves bounded prerequisite validation and final assessment scope.
4. A Kubernetes Job starts the isolated Mecatl worker.
5. The UI shows actual broker activity and a visibly rejected out-of-scope intent.
6. A bounded authorization workflow runs against the intentionally vulnerable application.
7. The trusted validator produces a supported finding with reproducible broker evidence.
8. The UI explains expected versus observed access and procedure-specific remediation.
9. A separately approved held-request fixture demonstrates Stop during active work.
10. A fresh assessment against the fixed application returns `no_issue_observed`, preserving the original finding and evidence.

Stop must also remain available during ordinary active assessments. The held fixture makes cancellation observable; stopping an already completed run is not an active-request demonstration. Changed targets require fresh scope approval.

## Included security workflows

| Workflow | Demo target and purpose |
|---|---|
| BOLA / cross-user object access | Owned VAmPI vulnerable/fixed deployments; verify identities, legitimate objects, peer probe, and follow-up checks. |
| Sensitive-field exposure | Separate owned property fixture; distinguish a known private-value leak, correct omission, and an ambiguous response. |
| Rejected scope escape | A fixed worker intent is rejected by the broker before contacting the unapproved destination. This is a boundary test, not autonomous discovery. |
| Stop during an active request | Dedicated owned held-request fixture; observe durable revocation, cancellation/drain, execution, and cleanup separately. |

Use existing registered plans and separate approved assessments. Do not build a general planner or combine the fixtures into a fictional single application. Keep ambiguous controls in qualification and available for inspection.

The configured application path currently uses a scripted provider driving the real Mecatl engine. Label it explicitly. The default demo makes no external model calls. Existing optional model experiments are separate from this delivery.

## Security and evidence invariants

- Test only owned/local/intentionally vulnerable targets with disposable synthetic accounts and records.
- Preserve the trusted broker as the target-network boundary. The worker/model receives no unrestricted networking or arbitrary shell capability.
- Preserve origin, port, account/path, redirect, DNS, actual-IP, and TLS validation. Discovery or model output never expands permission.
- Keep credentials and evidence tenant-bound. Target/provider secrets stay outside worker configuration, ordinary UI activity, prompts, and reports.
- Keep preflight permission, assessment approval, and Start separate. A saved URL or credential is not permission to test.
- Only trusted captured evidence and prerequisite validation determine results. Model output cannot confirm a finding or assign reviewed severity.
- Preserve `confirmed`, `no_issue_observed`, and `inconclusive`. A later denial must not erase earlier established exposure within the same run.
- Persist Stop before acknowledgement; deny later admissions and cancel tracked work. Do not restore permission on restart or silently replay uncertain operations.
- Display permission, request activity, worker observations, and cleanup as separate facts. Kubernetes deletion acceptance or object absence is not process-exit proof.
- Render untrusted text safely. Ordinary polling/events contain metadata; raw evidence requires explicit authorized inspection.
- Show no fabricated findings, progress, usage, coverage, customer identities, reviews, or timing guarantees.

## Product and implementation choices

Reuse the existing Go control plane, bbolt state, embedded frontend, Kubernetes Jobs, gVisor/Cilium lab, broker, validators, and exports. The controller, broker, validator, and optional model gateway can continue sharing one trusted process.

Keep the normal journey understandable without JSON editing or Kubernetes knowledge. Put the API explorer, raw scope, hashes, and infrastructure details behind optional inspection. Use bounded polling before introducing streaming infrastructure. Add abstractions only when a concrete demo requirement needs them.

## Definition of demo complete

- The connected browser journey above passes on an identified build with real Kubernetes execution and appropriate runtime/network qualification.
- BOLA and property controls retain their intended positive, negative, and ambiguous outcomes; missing prerequisites never become clean results.
- The reproduced repeated-object-probe defect is fixed and covered by a committed regression test.
- Live activity and scope rejection come from durable backend observations, with no secret/raw-body stream.
- Stop works while a target request is active and while the viewer has a pending read; lost acknowledgements remain visibly uncertain.
- Fixed-app retest has a fresh approved scope, linked comparison, and separate evidence. It never overwrites the original result.
- Fresh source checks and browser qualification pass; selected fixture credentials and build prerequisites are checked before a timed five-minute rehearsal.
- The presentation distinguishes scripted execution, validator confirmation, pending human review, retained evidence, and current limitations.

Historical CLI/browser successes are useful foundations, not completion of these remaining requirements. [The tracker](track.md) records the current position.

## Deferred work and release boundary

Defer additional executors/scanners, arbitrary shell, VM/MCP, network/mobile/cloud/crypto testing, specialist agents, generic discovery/login, live-model application planning, private-connector expansion, and toolkit-wide coverage.

Also defer customer login/billing, multi-replica storage, distributed monetary budgets, and scale engineering from this local demo. Before any real customer pilot, independently review procedures and qualify tenant isolation, identity/roles, secret/evidence lifecycle, TLS/key management, data/model handling, recovery, and relevant failures. Deferral does not waive those requirements.

There is no production sixty-second Stop guarantee, measured 1,000-agent capacity, general hostile-process containment, or autonomous vulnerability-discovery claim. Further product investment follows feedback on this demonstrated journey.
