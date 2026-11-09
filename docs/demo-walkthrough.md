# Explain Tsunami by walking through its APIs

The demo should answer: who approves actions, what the worker can actually do, where enforcement happens, what supports the result and what Stop means. Use the running [local UI](local-ui.md) beside a terminal. The current supported application path is scripted Mecatl and synthetic read-only tests.

An external application is a separate qualification step. `app.dev.accuknox.com` is not integrated or assessed by this UI milestone. Written testing scope, disposable identities/records, intended access rules, supported authentication and a reviewed external network path are prerequisites. Two usernames/passwords alone do not provide those facts or permission. Do not promise a finding before testing.

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

## Set up your view

Use a loopback control API port-forward and private operator-token file as described in the UI guide. Open `/ui/`, connect privately before screen sharing, and leave the Kubernetes terminal visible beside it.

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
tsunami_api POST "/v1/engagements/$TSUNAMI_ENGAGEMENT_ID/approve" --data-raw '{}'
```

D6's UI binds the explicit review checkbox to the displayed scope fingerprint without typing. Changing the displayed scope clears review; raw scope and fingerprint remain under optional technical inspection. The CLI still requires the exact reviewed digest. Direct API approval has the existing empty-body contract and approves the saved immutable scope; do not invent a `scope_digest` field. Requiring that digest at the API remains D10. The server owns permission, not the checkbox. An authorization reference is an operator record, not automated verification of legal ownership.

Show that an unapproved draft cannot start. Approval itself still launches no agent.

## 4. Start one run and show its Job

Copy the procedure from `engagement.items[0].procedure`:

```bash
tsunami_api POST "/v1/engagements/$TSUNAMI_ENGAGEMENT_ID/runs" --data-raw \
  '{"assessment":"vampi-vulnerable/object-access/v2","execution":"scripted"}'

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

## Finish with the useful result and boundaries

Open the candidate/report from the vulnerable run and the fixed control beside it. Download the exact evidence bundle if useful. Briefly state what remains: general onboarding, supported customer authentication, secret/evidence lifecycle, independent procedure review, adaptive planning and broader failure/scale testing. A fresh external owned-app deployment would add useful DNS/TLS/network qualification; it would not by itself add autonomous discovery.

Your ToolHive controller/configuration experience explains the lifecycle choice. Your Go external-authorization work explains enforcement outside the agent. Your MCP/runtime-observability experience explains why model statements and untrusted tool content are not evidence authority. Describe Tsunami as this project's implemented lab slice; do not turn proposals into past production experience.
