# Controlled demo: prepare, qualify and present

This is the single operating guide for the five-minute owned-lab presentation. The demo uses real backend state, scripted Mecatl and synthetic read-only tests. Never substitute a simulated outcome when readiness or evidence is missing.

## Prerequisites

Run commands from the repository root. A clone does not create the prepared environment. Bootstrap the dedicated Kind/gVisor/Cilium lab using [sandbox-lab.md](sandbox-lab.md), then provision the owned targets using [showcase.md](showcase.md). Confirm the exact environment and seven registered application plans in [demo-configuration.md](demo-configuration.md). Do not use arbitrary external targets or real customer records.

Source checks require the Go toolchain pinned in `go.mod`, Python 3 and Node 22. Browser checks require Google Chrome. Lab preparation also requires Docker, Kind, kubectl, Helm and the documented runtime/network installation. This is a prepared-environment demonstration, not a one-command fresh-cluster install or a five-minute provisioning claim.

## Prepare without discarding history

Choose the campaign before reserving history capacity: the core presentation needs three free engagement slots; the full campaign needs seven for six assessments plus headroom. From the repository root, inspect readiness first:

```sh
make showcase-check
# Full campaign only: include property controls and reserve seven free slots:
python3 scripts/showcase.py check --include-property
```

These checks make no assessment-target requests. They inspect owned infrastructure, registration, startup route status, history capacity and unsigned VAmPI JWT expiry hints. They do not prove that a session is valid; assessments recheck identities. Opaque property/Stop credentials have no local expiry hint.

If disposable credentials need refreshing, choose one preparation matching the campaign. Full selection (requires seven free slots):

```sh
make showcase-prepare
python3 scripts/showcase.py check --include-property
```

Core-only selection (requires three free slots; preserves existing property fixture data):

```sh
python3 scripts/showcase.py prepare
make showcase-check
```

Preparation builds matching uniquely tagged native/tools images from the current source, deploys them, resets only disposable VAmPI/Stop/property data and rotates their credentials/proofs. The UI setting and existing onboarding key/configuration are preserved. It does not renew optional private certificates, call a model, delete run/evidence history or clear tenant/global Stop gates. Old approved bindings are not silently updated: create fresh drafts/approvals after rotation.

Preparation requires idle owned work. If active work or nonterminal permissions are found, inspect exact IDs and explicitly resolve them before maintenance; do not kill unrelated work or remove state automatically. This is a single-operator lab, not a distributed maintenance lock. Preserve the database/key together for any separately planned backup/restore.

If the UI is absent/disabled, use `make ui-deploy` after confirming idle work. It preserves target data and the existing onboarding key but selects a new build, so subsequent qualification must refer to that new build. Do not rebuild between qualification and presentation without qualifying the replacement.

## Qualify the selected build

Use explicit kubeconfig/context for commands that do not set them internally:

```sh
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
make verify
make verify-ui-retest
make isolation-check CONTEXT=kind-tsunami-lab
CONTEXT=kind-tsunami-lab python3 scripts/tool-network-check.py
# Six fresh assessments: full connected journey, then separate property controls:
make demo-rehearse
# Or just the three-assessment timed core, with property checks qualified separately:
# make demo-rehearse-core
```

After a campaign, inspect its recorded capacity before scheduling another: the preflight reserves slots for a future campaign and may correctly refuse another full run. Preparation does not consume engagement slots, but its guards require seven free slots for the full selection, or three without `--include-property`. Never change those limits or discard history implicitly.

Run infrastructure/assessment/browser qualification sequentially so guards and target counters do not interfere. `make demo-rehearse` checks matching deployed asset bytes, then runs the real Add -> four-read preflight -> exact approval -> actual activity -> finding/evidence/remediation -> active Stop -> linked fixed-app retest path. It records six scheduled presentation stages and fails late stages; property leak/filter/ambiguity run afterward outside presentation time. `make verify-ui-demo` performs the same checks without pacing and does not establish a five-minute rehearsal. Approvals are automated owned-lab actions, never customer authorization or human finding review. The existing seven-case `make showcase-extended` remains an optional separate CLI campaign that consumes seven additional engagement slots; it was not repeated in D9. Plan capacity before every campaign and never silently discard history. Full qualification reserves at least seven free slots for six new assessments; `make demo-rehearse-core` reserves three and runs only the timed onboarding/finding, Stop and linked retest journey. Its summary explicitly excludes the property campaign. Run the disposable local property/browser checks separately; do not label the reduced campaign a full six-case D9 qualification.

Retain actual source/binary and image identities, CRI observations, target cancellation/counters, exact run/engagement IDs, result exports and prerequisite timestamps. A dirty working-tree build must be labeled and fingerprinted rather than described as a clean-commit reproduction. Historical exports remain untouched. Keep failed attempts with their failure status.

## Open the running UI

In a separate terminal:

```sh
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
kubectl --context kind-tsunami-lab -n tsunami port-forward \
  --address=127.0.0.1 service/tsunami-control 8080:8080
```

Open `http://127.0.0.1:8080/ui/`. The UI refuses non-loopback Host values; this exposure guard does not replace operator authentication. Connect privately before sharing your screen.

Under **Add application**, select the existing reviewed `vampi-vulnerable` integration, confirm disposable synthetic data, approve the four legitimate reads, prepare the new draft, then review/approve its exact scope and explicitly Start. The backend reuses trusted lab access without revealing stored tokens to the page. From Evidence, explicitly select `vampi-fixed` in linked retest, enter a fresh authorization reference, review changed bindings, approve its own scope and Start separately. Property and Stop controls use their own registered plans and approvals. Follow [the exact five-minute script](demo-walkthrough.md#five-minute-presentation). The historical D2 build predates this presentation and remains labeled separately.

Connect with the tenant operator token from a private local file. This is not a target account credential. To create that file without putting its value into shell arguments/history:

```sh
mkdir -p .local
umask 077
kubectl --context kind-tsunami-lab -n tsunami get secret tsunami-operator \
  -o jsonpath='{.data.admin-a}' | base64 --decode > .local/operator-token
chmod 600 .local/operator-token
```

Paste it privately into the masked field, then Connect before sharing your screen. Keep the private file out of Git and recordings. There is no implemented desktop keychain/clipboard cleanup integration. Use your local editor/clipboard tools deliberately; do not paste the value into chats, screenshots or frontend configuration.


## Readiness window and recovery

VAmPI setup issues one-hour credentials. Preflight requires at least 900 seconds of hinted remaining lifetime by default. This is a readiness threshold, not permanent certification or verified session lifetime. Recheck immediately before D9/presentation; if it fails, explicitly refresh between assessments, create new bindings/drafts and repeat the relevant qualification. Do not lower the threshold to hide expired access.

Keep the preflight timestamp and build identities with every qualification record. A source/image mismatch, changed Service address, invalid identity, failed positive network control or unavailable observation invalidates the corresponding claim. Investigate and record the failure; do not replace it with a simulated success.

A failed/lost Start response is recovered by reading saved child IDs or explicit **Recover Start** with the original action key. Never generate a replacement key automatically. An unacknowledged Stop remains uncertain. If the control API is unavailable, use trusted platform inspection and acknowledge that normal Stop is unavailable; there is no independent external egress cutoff in this package. Preserve/export history before separately planned capacity maintenance; never wipe the lab database to free engagement slots.

Local artifacts may contain raw synthetic evidence and private setup material. Share only a reviewed selected subset. General redaction, production identity/isolation, hostile-process containment, broader partitions/load, capacity and a production Stop guarantee remain outside this qualification.

## Five-minute presentation

Prepare and qualify the selected owned lab before the clock starts: matching images/assets, current credentials, idle work, history capacity and loopback access. Connect privately before sharing. Use only the exact registered applications. The automated equivalent is `make demo-rehearse`: it records real stage timings, waits for the scheduled boundaries and fails late stages. It is a paced browser dry run, not recorded human narration. Property controls run afterward as separate qualification, outside the five-minute presentation.

| Time | Screen and action | Say |
|---|---|---|
| 0:00–0:30 | Dashboard; point to the existing worker → broker → target boundary. | “The problem is safely running authorized security tests with customer credentials. A worker must not choose its own scope or declare success. The Go service stores approval and Stop, holds credentials, and brokers each request. Mecatl executes the bounded procedure in a Kubernetes Job. This build uses the owned gVisor/Cilium lab; those boundaries have separate runtime and network checks.” |
| 0:30–1:30 | Add Application; explicitly reuse `vampi-vulnerable` lab facts, mark synthetic data, review/approve the four legitimate reads. Prepare, review the exact account/GET/path matrix and limits, approve, then Start. Show actual activity and worker observation. | “This is an owned VAmPI application with two disposable accounts and supplied object facts. Saving is not testing permission. First I permit four legitimate checks; they do not include a cross-user probe. The final scope shows the exact origin, addresses, identities, actions, intended rule and budget. I approve this scope, then separately Start. The timeline contains persisted broker/controller observations. Dispatch intent and observed worker startup are different facts.” |
| 1:30–2:30 | Evidence and findings; show expected versus observed access, identities/rechecks, owner baseline and peer probe. Explicitly inspect the two proof artifacts, their byte check and remediation. Locate the actual rejected intent in activity. | “The authenticated peer received the owner's known private value. The validator also checked both identities, legitimate baselines and follow-up access. These exact captured responses support the result; model text cannot confirm it. Normal navigation reads metadata. Raw evidence is a separate authorized action, and downloaded bytes match their hashes. Hashes are not signatures. The candidate remains unrated and needs human impact review. The correction is server-side object authorization. This attempted scope escape was rejected before a send; approved scope did not expand.” |
| 2:30–3:20 | Open the separately approved `stop-lab-run` fixture. Explain Stop before Start. Wait for its active fifth request; hold a real viewer export and click Stop. Show receipt, revocation, drain and cleanup separately. | “This separate synthetic fixture holds a request so Stop can be tested during real activity. Pausing observation does not revoke permission. Stop stays available while an evidence read is pending. The service durably revokes permission and cancels the broker operation; the fixture independently records cancellation. Worker observations and API cleanup are separate. Stop cannot undo an action already accepted remotely, and this lab does not establish a production latency or partition guarantee.” |
| 3:20–4:15 | Recover the original finding. Select `vampi-fixed` in linked retest, enter a new authorization reference, review changes, approve and Start. Show actual `no_issue_observed` beside the original `confirmed`. | “The fixed application is a different origin with its own bindings. This creates a new unapproved assessment; it copies no credentials or permission. I review the changed scope and approve again. This completed check returned no issue observed. The original result and evidence remain intact. The comparison shows actual versions, runs and artifact references; a different rule is labeled incompatible. A lost Start response is recovered by reading the stored run ID, never by silently replaying work.” |
| 4:15–5:00 | Reopen original evidence/history; state limits and leave the displayed observations unchanged. | “This proves one bounded, reproducible authorization journey. Configured applications use scripted Mecatl, not autonomous discovery or external inference. Object access and property exposure are separate tests; property leak, filtering and ambiguity have their own owned controls. A clean check is not application-wide security, and this system does not independently verify a deployed patch. Customer login, broad authentication/discovery, general redaction/retention, independent procedure review and production failure/isolation qualification remain unfinished. If evidence or readiness is missing, we show uncertainty rather than a fabricated outcome.” |

Do not fill a missed deadline with a simulated result. Leave the live state visible and identify a retained record as historical fallback. The dry run's deliberate lost-Start and delayed-viewer responses affect the client only; they are not production partition tests. Fixture preparation, builds, network diagnostics, downloads/materialization and the separate property campaign are qualification work, not a claim that every setup task fits in five minutes.


## Optional API inspection

The remaining sections explain the same saved state through the API; they are not a second presentation sequence.

## Set up your view

For API inspection, keep the Kubernetes terminal beside the connected UI. The commands below are optional reference; the timed presentation uses the UI.

Choose one client to perform mutations: either the UI buttons or the API terminal. Do not press Start in both. When changing state through the terminal, use Refresh lists/open the saved assessment or Refresh status in the UI; its API explorer records only its own calls. A model, Pod or request moving independently is observed state, not a browser animation.

If using curl, this Bash helper reads the header through stdin so the token is not a command argument. Do not add `--location`, `--verbose`, tracing, credentials or arbitrary origins:

```bash
tsunami_api() {
  local method="$1" api_path="$2"
  shift 2
  { printf 'Authorization: Bearer '; cat .local/operator-token; printf '\n'; } |
    curl --silent --show-error --fail-with-body --proxy '' \
      --header @- --header 'Content-Type: application/json' \
      --request "$method" "http://127.0.0.1:8080$api_path" "$@"
}
```

These are operator calls to your local control API. They do not bypass the worker/broker to test the application directly. Use actual returned IDs; the placeholders below are not executable IDs.

## 1. Select the application

```bash
tsunami_api GET /v1/applications
```

UI: **Dashboard**. Explain: “The operator selects a registered application. The trusted setup already supplied two identities, their legitimate records and the intended access rule. The agent doesn't choose an arbitrary host or receive the target credentials.” Use **Add application** for the [separately approved onboarding path](application-onboarding.md); its four legitimate preflight reads never approve an assessment probe.

For the owned demo choose `vampi-vulnerable`. VAmPI is an independently developed vulnerable API, deployed with synthetic accounts and records. The same configured procedure will later run against its fixed variant. That is a control for our conclusion, not blind discovery.

## 2. Save and review scope

```bash
tsunami_api POST /v1/engagements --data-raw \
  '{"application":"vampi-vulnerable","name":"Owned API demo","authorization_ref":"owned-lab/demo","executor":"curl-http/v1"}'

TSUNAMI_ENGAGEMENT_ID='paste-the-returned-id'
tsunami_api GET "/v1/engagements/$TSUNAMI_ENGAGEMENT_ID"
```

UI: **Scope & approval**. Show the target origin, exact destination addresses, permitted account/path pairs, seven planned GETs, private-field rule where applicable, limits, expiry and digest. Open the full scope assertions if asked. Say: “Saving a plan makes no target requests. This is where the application's facts become a bounded permission record.”

The workflow comes from Tsunami: verify both identities, establish each account's own-object baseline, perform the peer probe, then recheck owner access and peer identity. These checks help distinguish an access-control result from an expired session or missing object. Preparation derives mechanical bindings; it does not discover intent or grant approval.

## 3. Approve that saved plan

```bash
tsunami_api POST "/v1/engagements/$TSUNAMI_ENGAGEMENT_ID/approve" --data-raw '{"scope_digest":"paste-the-reviewed-scope-digest"}'
```

D6's UI binds the explicit review checkbox to the displayed scope fingerprint without typing. Changing the displayed scope clears review; raw scope and fingerprint remain under optional technical inspection. The CLI still requires the exact reviewed digest. Direct API approval requires the reviewed `scope_digest`; empty, wrong or stale values are rejected without changing permission. Copy the digest from the exact saved scope you inspected, not from another assessment. The server owns permission, not the checkbox. An authorization reference is an operator record, not automated verification of legal ownership.

Show that an unapproved draft cannot start. Approval itself still launches no agent.

## 4. Start one run and show its Job

Copy the procedure from `engagement.items[0].procedure`. Generate one action ID (for example `python3 -c 'import secrets; print(secrets.token_hex(16))'`) and retain it for same-action recovery. Replace the placeholder below; it is not a valid key:

```bash
tsunami_api POST "/v1/engagements/$TSUNAMI_ENGAGEMENT_ID/runs" --data-raw \
  '{"idempotency_key":"paste-one-new-32-lowercase-hex-action-id","assessment":"vampi-vulnerable/object-access/v2","execution":"scripted"}'

TSUNAMI_RUN_ID='paste-the-returned-run-id'
tsunami_api GET "/v1/runs/$TSUNAMI_RUN_ID/status"
```

UI: **Run & Stop**. In the other terminal:

```sh
kubectl --context kind-tsunami-lab -n tsunami get jobs,pods -l tsunami-role=runner
```

“Start records work. The Go reconciliation loop creates a Kubernetes Job; this is not a new custom CRD. The worker runs Mecatl under the configured gVisor runtime, without an API token, privileged access or direct target-network permission. The curl adapter submits intents through a private socket to the broker. It isn't unrestricted curl.”

Explain that namespace separation, gVisor and network enforcement solve different problems. The lab has actual runtime and selected bypass-test records, not a general sandbox-escape proof. The current trusted-service lab egress is broader than a production customer deployment should use.

The main procedure is predefined, not a live LLM inventing tests. Mecatl runs the real tool-call loop. The broker checks run permission, origin/address/path/account, limits and redirects before sending a request and capturing its response. The validator decides from the captured evidence; a worker's text cannot declare the finding confirmed.

## 5. Inspect the candidate and its evidence

```bash
tsunami_api GET "/v1/engagements/$TSUNAMI_ENGAGEMENT_ID/findings"
```

On a matching D7 build, **Inspect evidence & candidates** loads body-free metadata. Show the failure-specific title, expected versus observed access, owner baseline/peer probe comparison, both identity checks/rechecks and deterministic remediation. Use **Inspect raw capture** separately on the owner baseline and peer probe to inspect the synthetic proof. Each action reads one stored artifact, verifies its exact hash/provenance, and sends no new target request. The optional inspected-artifact download preserves its original bytes. Full export/report remains a separate explicit action and includes raw synthetic bodies. The older D2 deployed build needs a qualified replacement before presenting this flow.

Show the actual rejected off-scope intent in Observed run activity: it was captured as blocked, not sent to that destination. Finding metadata omits rejected arguments; an explicit raw artifact can contain them and should be inspected deliberately. Show current read-time validation separately from the original stored verdict if they differ.

“The broker observed the responses. The validator has evidence supporting this candidate. Human review still decides business impact/severity. A generic HTTP 200 is not enough, and hashes detect changed bytes rather than authenticating a compromised broker.”

On a matching D8 build, use **Retest with an owned registered application** below remediation: explicitly select `vampi-fixed`, enter a fresh name/authorization reference and create the linked draft. Show the changed origin, account/path/assertion bindings, definition digest and scope/expiry; the original approval grants no permission here. Check the fresh review box and approve, then Start separately. The expected result is `no_issue_observed` for this completed check. Show the source/retest comparison with actual versions, run IDs, verdicts and evidence references; open the original evidence to demonstrate preservation. A different profile/access rule is labeled incompatible, and unfinished/stopped/inconclusive work supplies no clean result. The older D2 deployment needs a qualified matching replacement before this presentation. Do not say the whole application is secure or claim the system independently verified a deployed patch. The property controls are optional follow-up material, including the inconclusive redacted-value case.

## 6. Demonstrate Stop separately

Use a new `stop-lab-run` engagement, approve its actual scope and start its returned object-access procedure. Its fifth target request deliberately waits, so cancellation can be observed. This is a dedicated synthetic Stop fixture, not a discovered vulnerability or slow production target.

Pause observation and explain that permission remains active. Then:

```bash
tsunami_api POST "/v1/engagements/$TSUNAMI_ENGAGEMENT_ID/stop" --data-raw '{}'
tsunami_api GET "/v1/runs/$TSUNAMI_RUN_ID/status"
```

Use the IDs for the Stop engagement/run, not the earlier VAmPI run. Watch durable revocation, broker activity, recorded container execution and cleanup separately. Target fixture counters and older CRI qualification records provide additional observations; the UI does not directly inspect containerd or poll target counters.

“Stop closes permission before cleanup. The broker cancels its active operation and denies later calls; reconciliation removes the workload. A restart must not restore permission. A successful DELETE alone is not exit proof, and we cannot undo a remote effect already performed.”

Broader failure/load qualification and an independent emergency cutoff remain unfinished. Do not claim a production sixty-second guarantee.
