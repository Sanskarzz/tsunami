# Form → setup conversation → validated application → assessment

Recorded on 2026-11-06, starting at 13:56 UTC. This is a real browser/owned-Kubernetes qualification with automatic **owned-lab** approvals, not a customer assessment or human finding review. The setup advisor uses checked rules; the assessment uses scripted Mecatl. There were zero external model calls.

Open [the summary](summary.json), [browser pass log](browser-verification.txt), [operator request metadata](operator-requests.json), [build identities](build.json) or [the validated setup snapshot](onboarding-session.json).

| Control | Observation |
|---|---|
| Frontend URL without backend details | Saved metadata, asked for API origin, zero preflight requests. [Screenshot](onboarding-missing-api.png). |
| Password login without an adapter | Marked unsupported; no guessed login request or credential collection. |
| Owned bearer/JSON integration | Known facts reused, protected access updated, explicit four-read validation completed. [Setup ready](onboarding-ready.png). |
| Registration and final approval | Compiled plan became a new immutable application plus an unapproved assessment draft. No security run until its scope was approved and Start pressed. |
| New application's actual assessment | Seven target reservations, six verified baselines/rechecks, one blocked off-scope intent, one unreviewed finding candidate. [Bundle](onboarded-vampi/bundle.json), [draft](onboarded-vampi/report-draft.md), [screenshot](onboarded-finding.png). |
| Actual isolation runtime | The running onboarding-created worker was inspected through CRI: `runsc` / `io.containerd.runsc.v1`. Its observed tools image ID is in the summary. This is not a sandbox-escape test. |

The same run repeated the original [vulnerable](vampi-vulnerable/bundle.json), [fixed](vampi-fixed/bundle.json) and [held-request Stop](stop-lab-run/bundle.json) controls. The fixed case includes a client-side lost committed Start response; the Stop case holds a browser export response while revoking actual run permission. Those client injections are not production network-partition qualifications. The API explorer's calls are tab-local operator requests, not an internal controller/packet trace.

The setup snapshot was taken **before registration**, at `ready_for_approval`: four legitimate reads and a compiled public plan, with no engagement yet. The summary links it to the later registered application's approved runtime snapshot and actual evidence. The known lab integration supplied paths and protected access references; raw tokens were never sent to the conversation. Updating the protected input exercised the encrypted credential API without displaying or retaining those values in browser storage/trace output.

Downloaded exports are retained under `downloads/` and materialized through the existing Go exporter. The integrity gate checks all four bundle copies, evidence hashes, IDs, the preflight/runtime plan match, baseline counts and actual runtime observation. Raw preflight responses remain encrypted in the state database; they are not copied here. Final assessment captures are collected separately and intentionally contain raw synthetic proof values.

The native/control and tools builds are pinned by the recorded image IDs/tags and client/asset hashes. The successful driver checked deployed assets against local source before starting. Source was the working tree after baseline commit `985d57a9a9df3eb0876b3a01c2d56e4b81e2b913`; this is not a clean-commit reproduction claim. A preceding handoff check read a previous hidden plan too early and failed before starting the new assessment. Its failed local record remains unchanged. The test now waits for the actual new application scope, and the loaded-scope view hides the unrelated draft-creation form.

Current owned credential values and the onboarding key were compared against the selected files and decoded artifact contents before copying; no matches were found. Screenshots were captured after protected fields were cleared. This is a concrete comparison, not general redaction or permission to include customer data. No kubeconfig, encryption key, browser profile, `.local/` file or raw Secret dump is included.

API cleanup observations do not prove process exit. Some status snapshots correctly retain `unknown_after_resource_removal`; no CRI exit observation is invented. Prior network/Stop campaigns keep their own builds and limits. This run does not establish every CNI path, arbitrary hostile shell safety, customer readiness, a production 60-second Stop bound or 1,000-agent capacity.

Use [the onboarding guide](../../docs/application-onboarding.md) for the actual supported contract. Password/SSO, automatic frontend/API discovery, document ingestion and live-model setup planning remain unimplemented. AccuKnox and Target were not contacted; their auth/endpoint behavior or findings were not inferred from this controlled result.

Fresh [Go race/vet/Python/integrity checks](verification.txt) and [actual local toolkit-schema validation](toolkit-validation.txt) passed. The gate checks 32 bundles/240 artifacts, six prepared/runtime matches and 20 unchanged historical trial hashes; twenty Python tests pass.
