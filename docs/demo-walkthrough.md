# Explain Tsunami by walking through its APIs

The demo should answer: who approves actions, what the worker can actually do, where enforcement happens, what supports the result and what Stop means. Use the running [local UI](local-ui.md) beside a terminal. The current supported application path is scripted Mecatl and synthetic read-only tests.

An external application is a separate qualification step. `app.dev.accuknox.com` is not integrated or assessed by this UI milestone. Written testing scope, disposable identities/records, intended access rules, supported authentication and a reviewed external network path are prerequisites. Two usernames/passwords alone do not provide those facts or permission. Do not promise a finding before testing.

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

UI: **Applications**. Explain: “The operator selects a registered application. The trusted setup already supplied two identities, their legitimate records and the intended access rule. The agent doesn't choose an arbitrary host or receive the target credentials.”

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

The UI/CLI additionally require the exact digest you reviewed. Direct API approval has the existing empty-body contract and approves the saved immutable scope; do not invent a `scope_digest` field. The server owns permission, not the checkbox. An authorization reference is an operator record, not automated verification of legal ownership.

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
tsunami_api GET "/v1/engagements/$TSUNAMI_ENGAGEMENT_ID/export"
```

This can print raw synthetic evidence; prefer the UI's explicit **Load evidence snapshot** and controlled capture viewer during the call. Show the two identity checks, legitimate owner response and peer response that exposes the same known private value. Also show the rejected off-scope intent: it was captured as blocked, not sent to that destination.

“The broker observed the responses. The validator has evidence supporting this candidate. Human review still decides business impact/severity. A generic HTTP 200 is not enough, and hashes detect changed bytes rather than authenticating a compromised broker.”

Repeat with `vampi-fixed` in a new engagement and its actual procedure ID. The expected result is `no_issue_observed` for this completed test. Do not say the whole application is secure. The property controls are optional follow-up material, including the inconclusive redacted-value case.

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
