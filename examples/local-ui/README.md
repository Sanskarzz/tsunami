# Running local UI: retained browser qualification

Recorded on 2026-11-06, starting at 11:43 UTC, using a fresh headless Chrome profile and the existing owned `kind-tsunami-lab` cluster. Open the [browser summary](summary.json), [pass log](browser-verification.txt), [credential-free request metadata](operator-requests.json) or [build identities](build.json). Fresh [Go/race/vet/Python/integrity checks](verification.txt) and [local toolkit-schema validation](toolkit-validation.txt) also passed. These are retained observations, not a live customer assessment.

| Application | Browser-observed result | Evidence |
|---|---|---|
| Owned VAmPI vulnerable | `confirmed`, one unreviewed candidate | [Bundle](vampi-vulnerable/bundle.json), [draft](vampi-vulnerable/report-draft.md), [screenshot](vampi-vulnerable.png) |
| Owned VAmPI fixed | `no_issue_observed`, no candidate | [Bundle](vampi-fixed/bundle.json), [draft](vampi-fixed/report-draft.md), [screenshot](vampi-fixed.png) |
| Held-request fixture | Revoked permission, no local operations, API cleanup observed | [Bundle](stop-lab-run/bundle.json), [status](summary.json), [before Stop](held-before-stop.png) |

The browser selected registered applications, saved drafts, rejected a wrong digest locally, approved the actual scope, and sent one Start POST per engagement despite double clicks. Real Kubernetes Jobs ran the scripted Mecatl/curl/broker procedure. Each exported bundle contains one rejected off-scope intent. There were zero external model calls and no human finding reviews. Automation approved only these owned-lab scopes; this is not customer consent.

The fixed case delivered the real Start POST, then a browser fetch wrapper hid its committed response. The UI showed an uncertain Start, refused retry and recovered the stored child ID by reading the engagement. The Stop case held a real export response in the browser while the target probe was active; the Stop button remained usable and actual revocation/local drain/cleanup were observed. These are controlled **client-side failure injections**, not Kubernetes/API partition or production Stop qualification. The viewer still cannot undo a remote effect or provide an independent emergency cutoff.

Other checks covered invalid operator tokens, no assessment on connection, no operator token in the DOM/storage/export, same-origin browser requests, captured scope denial, plain-text rendering of hostile API display content, download artifact hashes, narrow-screen layout and cleared display state on disconnect. The injection fixture changed only a browser-side application-list title; it did not change the real target responses, validator results or evidence.

See [the applications screen](applications.png), [disconnected screen](disconnected.png), and [narrow layout](mobile.png). Screenshots are of the running interface, not the static templates. Browser downloads are retained under `downloads/` and materialized through the existing Go exporter; the integrity gate verifies they match the linked bundles and evidence bytes.

The native control image, current client binary hash and three served asset hashes are recorded in `build.json`. The browser driver required deployed assets to match local source bytes. It checked the dedicated runtime/network configuration before starting, but did **not** individually inspect these workers through CRI. Prior runtime/network qualification retains its own builds and limits. The tools image is recorded as configured, not given a fabricated observed container image ID.

Before this successful run, a source/image mismatch was rejected before assessments. A separate browser setup attempt failed to navigate under an overly broad diagnostic DNS override; it created no assessment. That override was removed. An earlier three-case browser run also passed before the added lost-response/pending-export checks. All local records remain under ignored `artifacts/ui/`; this directory retains the later successful qualification without changing failed results into passes.

Current lab credential values were compared against the selected files and decoded artifact contents before copying; no matches were found. Raw synthetic private proof values are intentionally present. That comparison is not general redaction or permission to include customer data. Do not share `.local/`, the entire artifacts tree, a browser profile or real tokens/passwords.

Use [the local UI guide](../../docs/local-ui.md) and [API presenter walkthrough](../../docs/demo-walkthrough.md) to explore it. AccuKnox/Target login and testing are not implemented or performed here. General onboarding, customer identity/secrets/evidence lifecycle, adaptive application testing, hostile shell qualification, broader failures and scale remain product work.
