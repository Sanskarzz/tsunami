# Five-minute visual walkthrough

This guide describes the redesigned source UI. It does not qualify or update the existing Kind deployment. A recording against that lab still requires separate approval, matching assets/server, readiness checks, idle work, valid disposable access and sufficient engagement capacity. Connect privately before recording. Never display the operator token.

| Time | Visual sequence | Main point |
|---|---|---|
| 0:00–0:25 | **Applications**: use the reviewed VAmPI shortcut, or filter reviewed integrations and identify `vampi-vulnerable` by its exact ID and origin. | The product performs bounded assessments of owned synthetic applications. Configuration availability is not credential validity. History retains separate scopes and attempts. |
| 0:25–1:15 | **Add application**: application metadata → exact lab integration → test resources → baseline review. Point to the four identity/own-object GET rows, destinations and limits. Explicitly grant validation. | Saving facts sends no target traffic. Baseline permission excludes the cross-user probe. Credentials remain trusted-side. |
| 1:15–1:45 | **Scope & approval**: review origin, addresses, identities, rule, GET matrix, budget, expiry and allowed/blocked explanation. Approve the exact saved scope; continue to Assessment and separately Start. | Approval binds the server-required digest. The worker cannot choose additional authority. |
| 1:45–2:20 | **Assessment**: worker → broker → target strip; actual permission/execution/activity/cleanup facts; timeline; **Show scope rejection**. | Requested dispatch, admitted request, observed HTTP write, capture and validation are distinct observations. The escape intent was rejected; it did not broaden scope. |
| 2:20–3:10 | **Findings & retest**: confirmed verdict, expected/observed comparison, selected owner baseline and peer-probe references, identity/rechecks and remediation. Explicitly inspect the individual proof artifact(s). | Confirmation comes from captured evidence. The candidate remains unrated pending impact review. Hash checks identify bytes; they do not prove independent authenticity. |
| 3:10–4:00 | Choose the exact registered fixed app. Save a linked retest, review changed bindings, approve separately, Start and compare actual `confirmed` / `no_issue_observed` outcomes. | Original evidence remains intact. A new target needs new permission. The check does not independently verify a deployed patch. |
| 4:00–4:35 | Open the separate held-request Stop assessment. Explain Stop before starting it; observe the active held request, then Stop, including during a pending viewer read. | Pause ends observation only. Stop revokes permission. Receipt, local drain, target cancellation, worker exit and cleanup require their own evidence. |
| 4:35–5:00 | Return to retained evidence and state limits. | Scripted Mecatl; supplied synthetic facts; no arbitrary discovery/login/model calls; no application-wide clean verdict or production containment/scale guarantee. |

These segments may be joined from real captures with their build/date clearly identified. If readiness, execution or evidence is missing, leave the uncertainty visible. Do not substitute disposable browser fixtures for Kubernetes results. The disposable fixture uses native Mecatl with a LocalScheduler and supplies no runtime/network isolation qualification.

## Review the source visually without the cluster

Run the disposable browser campaigns **sequentially** from the repository root:

```sh
make verify-ui-activity-browser
make verify-ui-onboarding-browser
make verify-ui-findings-browser
make verify-ui-retest-browser
```

They launch temporary loopback servers, a temporary database, owned local synthetic targets and a sandboxed Chrome profile; the onboarding launcher uses its existing temporary adapter filesystem. They make no Kind/API/model calls and leave no preview server running. Private screenshots and summaries are saved under `artifacts/ui-{activity,onboarding,findings,retest}/local-*/`. Inspect the latest `summary.json`: require `passed: true` and `kubernetes_qualified: false`.

The matrix captures 1920×1080, 1440×900, 1280×720, 390×844 375×667 and 844×390 landscape under reduced motion. The finding campaign includes the confirmed object-access screen before any raw artifact is opened. The retest campaign captures the separately approved comparison. Artifacts are ignored by Git and are not shipped product fixtures.
