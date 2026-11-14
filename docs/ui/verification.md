# Source UI redesign verification

The redesign preserves embedded HTML/CSS and vanilla JavaScript. It changes no Go backend, API/schema, authentication configuration, stored scope, broker policy, worker, Kubernetes deployment, fixture credentials, evidence record or Stop gate. No runtime dependency, external asset, framework, skill package or generated research dataset is added.

## Before and after

| Before | After |
|---|---|
| Repeated target cards with similar content | Searchable application table with exact IDs/origins, registration filtering, actual saved-attempt projection and resume actions; history is never merged/deleted |
| Long flat onboarding form | Application and integration stages, server-checked resource questions, explicit four-read baseline review, scope handoff; lab reuse is separate from new supported HTTPS bearer/JSON integrations |
| Empty unselected destinations | Context explanations and application/history navigation; selected state survives navigation |
| Small type and weak labels | 16px body, 14px supporting text, 44px controls, semantic colors, responsive panels and wrapping IDs |
| Dense scope details | Exact target/identity/action matrix, limits and expiry, within-scope/blocked comparison; digest and assertions remain available |
| Internal event names dominate activity | Readable persisted-event titles; exact kind/source/references expand and stay open across refreshes; no invented activity |
| Result buried among metadata | Trusted verdict, expected/observed comparison, review status and selected baseline/probe artifacts first |
| Retest provenance dominates outcome | Actual source/retest verdicts and reasons first, with scope/attempt provenance expandable |

## Security invariants reviewed

- Same-origin operator API allowlist and credential omission from cookies remain unchanged. Operator tokens and explicitly requested raw evidence remain in page memory; no browser storage or URL credential transfer is introduced.
- No new API or target-fetch capability. Server authorization, tenant/run checks, method/path/address restrictions, credential custody, budgets and deterministic verdict decisions remain authoritative.
- Preflight still requires explicit current revision/digest/confirmation. Saving, DNS approval, baseline permission, assessment approval and Start remain separate. Changing facts or digests clears review.
- Assessment approval still submits the exact reviewed `scope_digest`. Retests remain linked new drafts with their own destination bindings, review, approval and Start.
- Start still uses the original idempotency key for explicit recovery. Lost responses never trigger automatic mutation retries or replacement work.
- Durable Stop remains independent of pending Start, export, finding/status/comparison reads and observation. Acknowledgement, stored revocation, drain, exit and cleanup remain separate. Setup and assessment Stop controls stack without overlapping when both contexts are open.
- Activity is persisted metadata only. Rejection arguments, bodies and credentials are not added to polling. Raw capture bytes are individually requested and checked against artifact/run/capture/procedure/engagement identity and SHA-256 before display. Full export remains a separate action.
- Dynamic application/evidence text is inserted through text nodes. No `innerHTML`, executable Markdown or new external content source is introduced. Loopback exposure guards and CSP are unchanged.

## Accessibility and visual checks

Chrome checks keyboard skip navigation, Tab/Enter activation, route focus, `aria-current`, unselected-context recovery, linked/focused form error summaries and visible protected-access fields. Semantic text/surface pairs meet 4.5:1. Native buttons, selects, labels and disclosures retain keyboard behavior. Status meaning is textual as well as colored. Reduced motion is respected.

Desktop captures/checks cover 1920×1080, 1440×900 and 1280×720. Narrow checks cover 390×844 and 375×667, plus 844×390 landscape. Checks cover viewport overflow, wrapped exact references and Stop visibility after scrolling. The onboarding campaign also verifies both revocation controls together, search focus, exact-ID/registration filtering, safe injected text and unchanged history/target-request counts.

These are targeted source/browser checks, not a full WCAG audit, manual screen-reader certification, production failure test or Kubernetes/runtime/network qualification. A final demonstration against the existing Kind lab still requires explicit approval and readiness checks. The existing persistent lab was not used or modified.

## Exact first-party files changed

- `internal/webui/static/index.html`
- `internal/webui/static/styles.css`
- `internal/webui/static/app.js`
- `internal/webui/static/onboarding.js`
- `scripts/ui-test-support.mjs`
- `scripts/test_ui_metadata.mjs`
- `scripts/test_ui_onboarding.mjs`
- `scripts/test_ui_findings.mjs`
- `scripts/verify-ui-activity.mjs`
- `docs/local-ui.md`
- `docs/ui/redesign.md`
- `docs/ui/demo-guide.md`
- `docs/ui/verification.md`

## Remaining limits

The UI retains the existing constrained synthetic application contracts and required registration URL. It provides no arbitrary discovery, SSO/password adapter, autonomous security assessment or patch-deployment verification. A result applies only to the configured check. Human impact/severity submission, scalable evidence delivery/redaction/retention, customer login and production isolation/failure guarantees remain outside this implementation. The application list filters locally after the existing API snapshot; large customer-scale pagination is not added. Approval fingerprints remain readable in technical details; no backend permission logic is moved into the client.

See [design decisions](redesign.md) and the [five-minute walkthrough and reproducible screenshot instructions](demo-guide.md).

## Final verification record — 14 November 2026

Implementation source: `bd59fa7` (preceded by `90391e9`, `42bee64` and `f09d1c5`). Environment: Go 1.27.4, Node 22.23.3, Python 3.13.7, Chrome 154.0.8037.97 on Linux. Browser campaigns ran sequentially.

| Required check | Result |
|---|---|
| `make verify` | PASS: all Go race tests, vet, 34 Python tests and historical example/evidence integrity checks |
| `make verify-ui-retest` | PASS: 44 frontend tests (6 metadata, 14 activity, 10 onboarding, 8 findings, 6 retest) |
| `make verify-ui-activity-browser` | PASS: five real local Mecatl attempts; vulnerable/fixed results, scope rejection, same-key recovery and three active Stop failure cases |
| `make verify-ui-onboarding-browser` | PASS: explicit four-read baseline grant, fresh scope approval, one real curl-adapter assessment, stale grant/auth/visibility/filter/history controls |
| `make verify-ui-findings-browser` | PASS: five real object/property cases, metadata-only navigation, selected raw-artifact hashes and original-byte downloads/exports |
| `make verify-ui-retest-browser` | PASS: distinct-target fresh approval/Start, confirmed/no-issue comparison, preserved source artifacts and incompatible unexecuted draft |
| `make build` | PASS: local executable with embedded final assets; no image build or deployment |

Historical example integrity checks only verify retained records; they do not constitute a new Kubernetes assessment. Intermediate failed visual checks were retained in ignored artifacts. The final campaigns pass after fixing narrow table overflow, protected-editor visibility, refresh disclosure preservation and the short-landscape sticky-container boundary.

## Screenshots and private campaign records

All screenshots below contain disposable synthetic local fixtures, not the existing VAmPI/Kind deployment. Operator tokens, bearer credentials and raw response bodies are excluded from these screenshots. Each final summary reports `passed: true`, `kubernetes_qualified: false` and the matching final `app.js` hash.

- [Activity campaign summary](../../artifacts/ui-activity/local-20261114T142424183Z/summary.json)
- [Onboarding campaign summary](../../artifacts/ui-onboarding/local-20261114T142434021Z/summary.json)
- [Findings campaign summary](../../artifacts/ui-findings/local-20261114T142449181Z/summary.json)
- [Retest campaign summary](../../artifacts/ui-retest/local-20261114T142502335Z/summary.json)

| View | Screenshot |
|---|---|
| Before: original dashboard | [Open image](../../artifacts/ui-onboarding/local-20261114T135641087Z/onboarding-dashboard.png) |
| After: applications | [Open image](../../artifacts/ui-onboarding/local-20261114T142434021Z/applications-1440x900.png) |
| Baseline permission review | [Open image](../../artifacts/ui-onboarding/local-20261114T142434021Z/preflight-review.png) |
| Exact scope review | [Open image](../../artifacts/ui-onboarding/local-20261114T142434021Z/scope-review.png) |
| Confirmed result at 1920×1080 | [Open image](../../artifacts/ui-findings/local-20261114T142449181Z/confirmed-finding-1920x1080.png) |
| Selected captured proof | [Open image](../../artifacts/ui-findings/local-20261114T142449181Z/selected-proof.png) |
| Separate retest comparison | [Open image](../../artifacts/ui-retest/local-20261114T142502335Z/retest-desktop.png) |
| Narrow findings | [Open image](../../artifacts/ui-findings/local-20261114T142449181Z/confirmed-finding-375x667.png) |
| Stored activity and Stop | [Open image](../../artifacts/ui-activity/local-20261114T142424183Z/activity-desktop.png) |

The full matrix is in those campaign directories. Regenerate it using the source-only commands in the [walkthrough](demo-guide.md#review-the-source-visually-without-the-cluster). These ignored files are local review artifacts, not committed product/demo data.

## Embedded asset identity

| Asset | SHA-256 |
|---|---|
| `index.html` | `583472a73a396bfe1ae6289ad0e895f1bbea6688013587120db7fb45ff26bc06` |
| `styles.css` | `8b6bd96725c4b1367b28b99e637837147c3e0609c89c44b41276463ee4bb9b11` |
| `app.js` | `406524e4f733b8147074fdf782e9be9aa25d16da73c0ebaceaff53e1e07df257` |
| `onboarding.js` | `9d71b8a05f7ebc9cef6e5e1fc0d88ed66365651ba4541f8b853c6da8d10334c1` |
