# Add an application and resolve missing prerequisites

The local UI now implements a form-first setup flow with a saved, typed conversation, protected bearer access, bounded baseline validation and dynamic registration. The setup assistant uses **checked rules**, not a live LLM. The actual assessment still runs the real scripted Mecatl engine in a Kubernetes Job. This does not add autonomous discovery, a general chatbot or password/SSO login.

## Try the complete owned example

Use the existing dedicated gVisor/Cilium lab:

```sh
make ui-deploy
```

This rebuilds matching native/tools images, preserves state/configuration, and provisions the onboarding encryption key once as an immutable Kubernetes Secret mounted only in the trusted control pod. It preserves an existing key and refuses to replace a missing previously configured key. It creates no target account, resets no target data and calls no model. If owned VAmPI credentials expired, refresh only its disposable setup between assessments with the existing helper, then check readiness. See [the local UI guide](local-ui.md).

Open the loopback UI and connect privately with the operator token. Choose **Add application**:

1. Enter a name, environment, frontend URL and authorization reference. Unknown API/authentication details may be omitted. Saving starts no target traffic.
2. For a repeatable owned demo, select **vampi-vulnerable** under existing reviewed integration and confirm disposable synthetic data. The trusted backend supplies its known paths and credential aliases; no token is exposed to the browser.
3. Read the next prerequisite and readiness facts; the checked conversation is expandable. Known information is reused. The reuse preview names the exact API target/addresses and explains that the frontend URL does not change them. A frontend-only setup asks for its backend API. Unsupported password/SSO is identified without guessing a login endpoint or collecting a password.
4. Review the API origin, exact destination addresses, identities and four legitimate GETs. Explicitly approve the preflight grant and choose **Validate four legitimate reads**. These are two identity checks and each account's own-resource read, not a cross-user probe.
5. Successful strict JSON baselines produce a compiled plan. The compiler infers an unambiguous identity/owner relationship and resource IDs from the actual responses, then derives assertions and hashes. Ambiguous or unsupported contracts stay unavailable; an HTTP 200 alone is not readiness.
6. Choose **Open the assessment scope for approval**. Registration and an unapproved engagement draft are saved atomically. Review the readable target/account/GET/action matrix, rule, proof-field names, limits and expiry; check permission and approve, then explicitly Start. No JSON editing or digest typing is needed. The checkbox binds to the displayed fingerprint, which remains available with raw assertions under technical inspection. The worker then executes the seven-read authorization procedure through the broker.
7. Observe permission, execution, broker activity and cleanup. Load the evidence snapshot and inspect the proposed candidate. Stop the assessment, or use **Stop setup / assessment** to revoke the setup and its linked assessment together.

**Update protected access** opens masked bearer-token fields. Saving them invalidates prior validation and sends no target request. The values are not added to conversation messages or API explorer bodies. They are assigned to one API origin; changing that origin invalidates access and destination bindings. Completed registrations are immutable: use a new setup for changed access/scope.

D6 clears preflight review when the saved revision/digest changes and clears assessment review when the displayed scope changes. A stale response cannot replace a newer setup revision. Preflight and final assessment permission stay separate; an earlier access check never silently approves a probe. The Dashboard shows registration and saved scope/attempt state separately. API inspection is collapsed by default. See [source/local browser qualification](local-ui.md#qualification); the selected deployment still needs matching rebuilt assets/server and fresh checks.

## New supported API integrations

A new application need not already exist in the startup file. The supported path is deliberately narrow:

- An explicitly approved HTTPS API origin and reviewed exact destination addresses. The DNS action is a separately confirmed bounded lookup, not an API request. Later connections resolve again, reject unapproved answers and dial the validated numeric address while retaining normal hostname/TLS checks.
- Public destinations only. The private exception is a tenant's exact existing reviewed lab integration; its origin, addresses and legitimate paths cannot be widened. New private/customer connector onboarding is not implemented.
- Two distinct bearer test tokens. Password, browser-cookie, SSO/MFA and automatic refresh adapters are not implemented. Do not paste credentials into the conversation or URL.
- A GET current-user path and two legitimate own-resource GET paths, without queries. Resource IDs must be distinct safe strings in the final segment of the same path template. Arrays/numeric IDs/ambiguous identity relationships need a reviewed adapter.
- An explicit intended sharing rule and one to six private synthetic string fields. The current proof contract requires distinct 24–256 byte values in the two disposable records. The user provides field names, not hashes; the backend derives hashes from validated responses. This is not a general sensitive-data classifier.
- Only disposable synthetic data. Normal customer-data handling and production secret/evidence infrastructure are not qualified by this release.

The form accepts initial information, and the typed conversation asks for missing fields. It is not a free-text LLM interpreting arbitrary instructions. Documentation upload, approved frontend inspection, automatic backend discovery and a model-assisted planner remain future work. A discovered or supplied API host would still need approval; it is never implicit in the frontend URL.

AccuKnox's frontend can be saved as metadata, but no AccuKnox endpoint/login is guessed or tested. Its actual API/authentication contract, allowed routes/actions/limits, disposable records and supported access integration are still needed. Permission for the dev environment does not make every available test applicable or promise a finding.

## Actual API contract

All routes use the existing tenant operator bearer authentication. Workers cannot access them. IDs come from server state; request bodies cannot select another tenant.

| API | Behavior |
|---|---|
| `POST /v1/onboarding` | Save initial application details; no target requests. Optional existing integration is looked up for this tenant only. |
| `GET /v1/onboarding` | List compact setup metadata for recovery. Does not download all conversations/plans. |
| `GET /v1/onboarding/{id}` | Read the saved question, facts, revision, validation status and public plan. |
| `POST …/{id}/answers` | Submit a typed answer matching the pending question ID and current revision. Stale replies are refused. |
| `POST …/{id}/details` | Revise details before registration. Invalidates readiness; an API-origin change clears assigned access/addresses. |
| `POST …/{id}/resolve` | With explicit confirmation, resolve candidate addresses. No API connection or credential forwarding. |
| `POST …/{id}/access` | Assign two protected bearer tokens to this API origin. No target requests. |
| `POST …/{id}/validate` | Require the current revision, preflight digest and explicit confirmation. Run at most four approved baseline GETs; never follow redirects. |
| `POST …/{id}/prepare` | Atomically register the validated application and create an unapproved assessment draft. A retry recovers the same engagement ID. |
| `POST …/{id}/stop` | Revoke setup permission, cancel local preflight and Stop a linked engagement. Does not depend on the encryption key being available. |

The existing engagement approval/start/status/Stop/export APIs are reused afterward. `POST /v1/applications` is not added: dynamic registration happens only after validated onboarding. The API explorer shows real calls from this tab, with **all onboarding request bodies omitted** so protected inputs never appear there.

## Permissions, failures and data

A preflight permit is distinct from assessment approval. It admits only the four shown account/path pairs, four request reservations per attempt, a 15-second deadline, at least one second between that attempt's requests, and at most three explicit attempts. Setups expire after one hour and have revision/message/count limits. No automatic login, account creation, retry or security probe is allowed by adding application details.

The compiled assessment adds a persisted minimum one-second interval between each run's target reservations, retains the eight-request allowance and 90-second run deadline, and is reviewed through its scope digest. These are per-attempt/per-run controls, not a distributed or customer-wide aggregate rate/budget guarantee. Broader limits remain product work.

Public-address filtering excludes private/loopback/link-local and selected reserved/metadata ranges. The deployment helper also records known node and Kubernetes API Service addresses in trusted infrastructure exclusions, outside user/agent input. Operators must review additional deployment/provider infrastructure exclusions before external use. This is not a complete catalog of every provider's infrastructure or a qualified customer network boundary.

Setup access and raw preflight responses use authenticated encryption with a fresh nonce and tenant/session/purpose binding. The key is outside the bbolt database in a control-only Secret. This protects those records at rest and rejects ciphertext substitution; a compromised trusted control process can still decrypt them. Preserve the matching key with database backups. Automated rotation, restore and retention/deletion remain unfinished. Existing startup Secrets and ordinary assessment evidence retain their documented lab storage limitations; this is not a secret-manager integration or complete evidence encryption.

Preflight raw bodies remain encrypted in local state, referenced by validation ID; the conversation receives status and public assertions, not those bodies. The final assessment collects its own evidence for the exported finding. Exact assigned-credential reflection is refused, including decoded JSON escaping. Unknown/encoded/transformed secrets still need general redaction; do not treat this as a complete secret scanner.

Stop persists revocation before acknowledgement and cancels local preflight contexts. Responses arriving after revocation cannot promote readiness. Tenant/global Stop also closes setup work. A restarted process marks unfinished preflight uncertain and does not replay it. A lost validation response is recovered by reading state; another attempt needs a fresh explicit grant. A lost prepare response recovers its atomic saved engagement. The existing Start endpoint still lacks an idempotency key; inspect stored child IDs rather than retrying automatically.

The frontend holds credentials only while submitting protected input, clears the fields, and stores no browser login. The local browser/workstation and control plane remain trusted. No external model call is made by this setup workflow and no supplied document/response is sent to a provider. Metadata such as names/URLs is not a safe place to paste secrets.

## Verify

```sh
make verify
make verify-toolkit TOOLKIT_PATH=../vapt-toolkit
node scripts/verify-ui.mjs --onboarding
```

Native tests cover no-traffic metadata, missing/unsupported prerequisites, stale answers/grants, scope edits, tenant isolation, encrypted access/evidence, wrong-key/ciphertext substitution, redirects, atomic draft recovery, active-preflight Stop, global Stop and restart uncertainty. Browser qualification includes the existing assessment/Stop controls plus form → question → protected access → four-read validation → scope approval → real worker → evidence. It uses owned fixtures, automatic lab approvals and no model/customer calls. Failed records stay failed; exact successful builds/results have their own retained record.
