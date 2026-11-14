# Stop lifecycle and failure testing

This milestone makes Stop observable without treating a successful API call as proof that everything has ended. It adds a held-request fixture and a repeatable campaign using the real `curl-http/v1` worker under gVisor. It does not add a new assessment procedure or arbitrary commands.

## Separate the facts

An authenticated operator can read `GET /v1/runs/{id}/status`. The tenant check is the same as for the run itself; worker credentials cannot read this endpoint.

| Field | Meaning |
|---|---|
| `permission` | Whether the durable run state allows work, has expired, is closed, or has been revoked. |
| `execution` | The last recorded dispatch intent, running container observation or terminated container observation. Each observation has a timestamp. This is not a continuously refreshed process check. |
| `activity` | Whether this broker has active work, observed a local drain, or has unfinished records from another process or an unpersisted completion. |
| `cleanup` | Cleanup pending, DELETE accepted, or Job and Pod objects observed absent after accepted cleanup. |
| `assessment` | The existing trusted validation result, when one exists. Stopping is not a successful assessment. |

Before executing a target/model operation, the broker persists its operation ID and process identity. Completion removes that record. Target request/model reservations record `last_admitted_at`. Stop persists revocation before cancelling local contexts and acknowledging the caller. A later status request can separately observe local drain and Kubernetes cleanup.

If a process dies before persisting completion, the next process keeps the record as **uncertain**. Zero local operations after a restart does not prove the previous request finished. Older runs without lifecycle tracking also remain unknown where evidence is missing. These observation fields never grant permission or cause an action to be replayed.

The journal bounds unfinished operations at 32 per run. Existing action, request, model and concurrency limits still apply. This is additional local durable bookkeeping, not a distributed worker lease protocol. It adds database writes, and the controller still scans stored history; neither is qualified at production scale.

## Local UI revocation

The source UI keeps the existing engagement Stop endpoint separate from Start/export/observation reads and visible across assessment/evidence screens. Pending, acknowledged and unacknowledged Stop receipts do not wait for follow-up reads. Later stored revocation can be observed after a lost response without pretending that response was acknowledged. Old viewer responses cannot clear that receipt, and no mutation is automatically replayed.

The run activity panel consumes the [typed event journal](design.md#activity-events), separately from the permission/activity/execution/cleanup status cards and lifecycle timestamps. Missing history, retention gaps, read errors and prior-process uncertainty stay explicit. Pause, navigation and disconnect cancel only viewer reads; they never send Stop. See [source and optional browser checks](local-ui.md#qualification). loopback Mecatl test does not replace the recorded gVisor campaign or qualify a new deployed build.

## Keep revocation available when optional routes fail

- `GET /livez` (and compatibility `/healthz`) checks that the HTTP process responds.
- `GET /readyz` checks the local state store and shutdown gate. Its scope is **the state and revocation API**, not all execution dependencies.
- Authenticated `GET /v1/status` reports startup configuration for public fixtures, configured applications, private connectors, model, scheduler and curl execution.

An invalid optional connector/model configuration disables that route instead of preventing the administrative server from starting. The broker rejects new work on unavailable routes. There is no direct-network, plaintext or paid-model fallback. Startup status is not an ongoing dependency probe: a configured Kubernetes API or model provider can still become unreachable later. Request errors and run observations report those failures separately.

The cluster campaign ran while the private broker certificate was expired. Public/configured application testing and Stop stayed available; the private route stayed unavailable. Certificate renewal is still an explicit separate operation. Failure of the operator credentials, state store or the entire broker can still make normal Stop unavailable. An external emergency egress cutoff remains unfinished.

HTTP shutdown closes admission, cancels request contexts and waits for admitted handlers before the caller closes the store. Reconciliation also exits before store closure. Shutdown is not a durable user Stop. A forced kill or a handler that ignores cancellation remains an operational failure case; the server does not invent completion for it.

## Reproduce in the owned lab

Use the already provisioned dedicated gVisor/Cilium Kind cluster. These commands do not configure a customer target or call a model:

```sh
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
make tools-deploy CONTEXT=kind-tsunami-lab IMAGE=tsunami:stop-lab TOOL_IMAGE=tsunami-tools:stop-lab
make stop-lab-setup CONTEXT=kind-tsunami-lab
make stop-lab-verify CONTEXT=kind-tsunami-lab
```

Setup refuses active/pending workers, creates only an owned synthetic target and four bounded read plans, and preserves other application/model/private configuration. It rotates the Stop fixture's synthetic credentials and resets that fixture's counters. Those credentials stay in ignored local configuration and Kubernetes Secrets. The fixture's diagnostic counters are not agent tools.

The verifier holds the fifth approved request (the cross-user probe), checks the actual CRI sandbox runtime and records target counters. It then measures Stop acknowledgement, target cancellation and worker exit separately. It rejects a request already ended or too close to the ten-second request deadline to attribute cancellation to Stop. A parent Stop also prevents another child from being created.

Cases:

1. Stop one run during the held request.
2. Stop its engagement during the held request.
3. Deny the controller's Kubernetes API access, observe an actual observation failure while the request is active, then Stop. Cleanup must remain pending until API access returns.
4. Repeat the API fault, commit Stop, then restart the control plane while that fault remains. Permission must stay revoked; after recovery, cleanup must finish without more target requests.

The fault uses an owned Cilium deny policy selecting only this lab's control pod. [Cilium deny policy](https://docs.cilium.io/en/stable/security/policy/deny/) takes precedence over allow policy. This does not shut down the cluster API or block the host's management connection. The verifier removes its policy in `finally`, refuses to overwrite an existing fault policy, and attempts parent revocation on failure. If the verifier is forcibly killed, inspect `tsunami-stop-api-outage` before another campaign and remove that owned diagnostic policy only after checking its purpose.

Native verification (`make verify`) also covers active request revocation, cross-tenant status access, incomplete operation records after process replacement, optional-route failure, graceful handler/store lifetime, and the difference between missing API objects and terminated container status. Toolkit export integrity/schema checks remain separate and unchanged.

## What the results establish

See [the recorded campaign](../examples/stop-lifecycle/README.md) for exact builds and observations. All timing samples include this host's client/polling overhead. The owned target cooperates with HTTP cancellation; another application could continue a job after its connection closes. Stop cannot undo a request already accepted by the target, or prove that a model provider stopped computing or billing.

CRI exit, Pod termination status and API resource absence are recorded separately. Missing Job/Pod objects alone cannot prove exit on a partitioned node. The API fault case leaves the broker, local state and target connection available. It is not a broker crash, disk failure, arbitrary node partition, compromised worker test, remote executor Stop test or load test. The restart case restarts **after committed revocation**, not during an uncertain acknowledgement. No production 60-second or 1,000-agent guarantee follows from this campaign.

## Auxiliary setup failure during group Stop

Tenant/global Stop first commits its gate and assessment revocations, then cancels all affected admitted assessment contexts before the separate onboarding-state transaction. A malformed/unavailable setup record can still make the API report a partial failure; it cannot skip cancellation of already-revoked assessments. Do not interpret that error as successful setup revocation or completed worker cleanup. Fault-injection regression tests use an actual held HTTP request, verify target-observed cancellation and later admission denial, preserve the error, and check the tenant/global cancellation boundary.
