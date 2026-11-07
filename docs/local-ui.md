# Local operator interface

This is a small running frontend for the existing configured-application APIs. It replaces neither the CLI nor the static product mockups. Its data comes from the control plane: no sample finding, fictional usage counter or simulated running state is inserted.

The [application onboarding flow](application-onboarding.md) now adds an initial form, checked questions for missing facts, protected bearer access, bounded prerequisite validation and a saved assessment draft. The setup advisor is rule-checked rather than a live model; wider agent planning remains in [the product direction](onboarding-readiness.md).

The interface supports registered synthetic object/property read plans, scope review/approval, one scripted attempt per engagement in this viewer, status, engagement Stop, recovery through saved assessments, explicit evidence loading and downloads. The new onboarding path can register supported synthetic bearer/JSON integrations after validation. Username/password login, SSO/MFA, arbitrary requests, live application inference, customer accounts and finding review submission are not implemented.

## Enable and open

The dedicated `kind-tsunami-lab` cluster, gVisor/Cilium and configured applications must already exist. Run from the repository root:

```sh
make ui-deploy
```

This checks for idle owned workloads and nonterminal permissions, builds/loads matching uniquely tagged native/tools images, provisions/preserves a control-only onboarding encryption key and enables `TSUNAMI_ENABLE_UI=true` on the existing control Deployment. The worker image setting is updated with the native image; the tools image is updated with the same source build. Other environment/Secret configuration and durable history are preserved. There is no new cluster, Ingress, NodePort, model call, target reset or automatic assessment. Use the lab exclusively during this maintenance; the check is not a distributed maintenance lock.

If disposable lab credentials have expired, explicitly refresh them between assessments:

```sh
make showcase-prepare
make showcase-check
```

Preparation resets only the documented owned fixture data and preserves the UI switch. See [the reset guide](showcase.md). Rebuild after source changes; the browser verifier checks deployed asset bytes against the local source.

In one terminal:

```sh
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
kubectl --context kind-tsunami-lab -n tsunami port-forward \
  --address=127.0.0.1 service/tsunami-control 8080:8080
```

Open **http://127.0.0.1:8080/ui/**. The handler refuses non-loopback Host values and credentials/query parameters in UI URLs. The API remains a ClusterIP service in this lab. This Host check is an exposure guard, not operator authentication or protection from a compromised workstation/local browser.

Connect with the tenant operator token from a private local file. This is not a target account credential. To create that file without putting its value into shell arguments/history:

```sh
mkdir -p .local
umask 077
kubectl --context kind-tsunami-lab -n tsunami get secret tsunami-operator \
  -o jsonpath='{.data.admin-a}' | base64 --decode > .local/operator-token
chmod 600 .local/operator-token
```

Paste it privately into the masked field, then Connect before sharing your screen. Keep the private file out of Git and recordings. There is no implemented desktop keychain/clipboard cleanup integration. Use your local editor/clipboard tools deliberately; do not paste the value into chats, screenshots or frontend configuration.

## Screens and API calls

| Screen/action | Existing operator API | Decision |
|---|---|---|
| Connect / refresh lists | `GET /v1/applications`, `GET /v1/engagements`; Connect also reads `GET /v1/status` | Tenant-authenticated registered plans and history only. Route status is startup configuration, not a live dependency probe. |
| Choose application | None | Local selection; no target requests. |
| Create draft | `POST /v1/engagements`, then `GET /v1/engagements/{id}` | Server snapshots the registered application and limits. |
| Approve | `POST /v1/engagements/{id}/approve` | Server records approval of immutable scope. The UI additionally requires the reviewed digest and checkbox. |
| Start | `POST /v1/engagements/{id}/runs` | Creates one scripted attempt; reconciliation creates/observes its Job. |
| Observe | `GET /v1/engagements/{id}` and `GET /v1/runs/{run}/status` | Read-only metadata. No capture bodies are polled. |
| Stop assessment | `POST /v1/engagements/{id}/stop` | Durable engagement revocation and cancellation; drain/exit/cleanup are separate facts. |
| Load evidence | `GET /v1/engagements/{id}/export` | Explicit snapshot including raw synthetic responses and existing candidates/reviews. |
| Download | No new request after loading a snapshot | Browser creates a local bundle JSON or draft Markdown download. |

The API explorer shows actual operator HTTP calls made by **this tab**, with method/path, body, response status and duration. Authorization is omitted. Exports are summarized in that explorer; raw evidence is available in the evidence panel. It is not a trace of internal controller decisions, broker tool calls, target packets or another terminal's API calls. Broker captures and recorded status/events provide those separate observations. Target/tool traffic never comes from the frontend.

The planned seven reads are shown before execution, with their purpose. Full public scope/prerequisite assertions are available for review. The result and baseline verification count come from the trusted validator. `confirmed` produces a candidate requiring review; `no_issue_observed` is limited to the selected test; `inconclusive` and Stop do not mean secure. This UI creates no review or severity rating. Previously recorded reviews remain in the downloaded bundle.

Use [the API-by-API presenter walkthrough](demo-walkthrough.md) for a call. The older [14 templates](templates/README.md) remain disconnected future-product previews; do not present their fictional states as live behavior.

## Failures and browser boundary

- The operator token is held in page memory only. No cookies, local/session storage, arbitrary control origins, proxy service or saved browser login are added. Onboarding has separate masked bearer fields with encrypted backend storage; those inputs are omitted from conversation/API explorer bodies. A page reload/disconnect requires reconnection. Browser extensions, DevTools and the trusted workstation can still access memory; this is not protection against a compromised browser.
- The opt-in wrapper sets `Cache-Control: no-store` and the UI has a same-origin CSP with no external scripts/fonts. Target/API text and Markdown are rendered as text, not HTML. Evidence can still contain sensitive echoed values; general redaction remains unfinished. Treat raw evidence/downloads accordingly.
- Draft/approve/start/Stop are never automatically retried. A lost Start response may already have created work. Refresh/recover its stored child IDs. This viewer blocks duplicate Start clicks and a second attempt for its loaded engagement; another client/operator can still create attempts within backend limits. There is no server start idempotency key.
- Stop is available independently of a pending Start/export/observation request. A failed acknowledgement is shown as unconfirmed. An acknowledged Stop followed by a failed read is not called a cleanup success. No external emergency egress cutoff is added.
- Observation polls after a run starts, without overlap within that loop. It ends on observed API resource cleanup, errors, pause, navigation/disconnect or a bounded observation window. Closing/pausing/reloading the viewer never implies Stop. Observations may be stale and API object absence is not process-exit proof.
- The viewer uses the existing 192 MiB inline export ceiling. Large exports can consume significant browser memory; this is a small lab interface, not scalable evidence storage/delivery. Downloads are JSON/Markdown, not an automatically materialized evidence directory. Use `tsunami export-bundle --input FILE --out NEW_DIRECTORY` to materialize and run the existing verifier.

## Qualification

```sh
make verify
node --check internal/webui/static/app.js
node scripts/verify-ui.mjs
# Include form/question/protected-access/validation/dynamic-registration checks:
node scripts/verify-ui.mjs --onboarding
```

The browser driver requires local Node 22 with WebSocket support and Google Chrome. It uses a fresh temporary headless profile, the owned cluster and a loopback port-forward. It reads the lab token into process/page memory without printing it. It automatically approves **owned-lab** scopes, runs the vulnerable/fixed/held-Stop cases, inspects actual API results, downloads/materializes evidence, checks hashes, and tests invalid tokens, wrong digests, duplicate clicks, display injection and narrow-screen layout. It also simulates a lost response after a committed Start and holds a real export response while Stop is used. These are client-side failure injections, not production partitions. It creates new run history and attempts to Stop only its own engagements. It never contacts a customer application or a model. A forced kill/platform failure can still leave uncertain work; inspect it before maintenance.

It saves a private result under `artifacts/ui/browser-.../`, including screenshots, IDs/status, credential-free operator request metadata, downloads and materialized exports. The display-injection check modifies only a browser-side application-list response; it does not fabricate target evidence. Failed records remain failed. Review output for credentials/data before sharing. This does not qualify arbitrary hostile shell execution, all browsers, accessibility, distributed failures, scale or real customer readiness.

The [retained browser qualification](../examples/local-ui/README.md) includes three actual outcomes, screenshots, downloaded/materialized evidence and exact build identities.
