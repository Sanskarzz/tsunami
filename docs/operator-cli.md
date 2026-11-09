# Guided operator CLI

`tsunami assess` connects the existing application, engagement, execution, Stop and export APIs. It does not ask the user to invent an attack prompt or specify which bug should be found. The current built-in test is cross-user object access; the application configuration supplies its prerequisites.

This is an owned synthetic HTTP lab. A trusted operator still supplies two test identities, legitimate records, identity/ownership assertions, proof hashes and the intended access rule through startup configuration. A URL plus “I authorize testing” is not enough. The CLI also supports the new [property-access profile](property-access.md), whose preview explicitly lists the private fields to omit from an allowed shared-object response. General onboarding, discovery, browser login and private configured applications are still unfinished. The subsequent [local UI](local-ui.md) connects this same lab journey without adding those capabilities. The obsolete disconnected HTML previews have been removed; use the implemented local UI.

The [onboarding UI](application-onboarding.md) can now derive supported bindings from four approved live baselines and register an immutable synthetic application. The CLI can select that returned application ID; it still provides no URL registration or login command.

The separate [prepare-plan command](binding-preparation.md) now derives the mechanical assertion/path/hash bindings from supplied baseline documents and explicit selectors/rules. It does not contact the target or approve work. Both owned setup helpers use it before trusted registration; raw source documents remain private locally.

## Walk through an assessment

Use the dedicated cluster and its already configured gVisor/Cilium execution boundary. Build and deploy matching native/tools images, then refresh only the disposable owned application if its one-hour test credentials have expired:

```sh
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
make tools-deploy CONTEXT=kind-tsunami-lab IMAGE=tsunami:journey-lab TOOL_IMAGE=tsunami-tools:journey-lab
make application-setup CONTEXT=kind-tsunami-lab
```

Application setup resets the disposable VAmPI databases and creates new test accounts/records. It refreshes only those tenant/application pairs, preserving other configured plans. Existing approved plans referencing the old credentials/configuration cannot silently use the replacement; create and review a new plan. This is lab provisioning, not a customer credential-refresh service.

On the trusted operator workstation, save the tenant operator token to an ignored private file and open a loopback port-forward. The key itself is not passed as a command argument:

```sh
mkdir -p .local
umask 077
install -m 600 /dev/null .local/operator-token
kubectl --context kind-tsunami-lab -n tsunami get secret tsunami-operator -o jsonpath='{.data.admin-a}' | base64 --decode > .local/operator-token
kubectl --context kind-tsunami-lab -n tsunami port-forward --address=127.0.0.1 service/tsunami-control 8080:8080
```

In another terminal at the repository root:

```sh
tsunami_assess() {
  ./bin/tsunami assess "$@" --url http://127.0.0.1:8080 --token-file .local/operator-token
}

tsunami_assess applications
tsunami_assess plan --application vampi-vulnerable --authorization-ref owned-lab/demo --executor curl-http/v1
```

Plan creation saves a **draft** and performs no target requests. Its preview shows the approved origin and destination addresses, credential aliases, ordered GETs, expected access rule, executor, request allowance, engagement expiry, child/concurrency limits and scope digest. Credentials and raw target response bodies are not printed. Account validity is established during execution, not by listing or creating the plan. The authorization reference records the operator's permission context; the software does not independently verify legal ownership.

Review that preview. Set these variables to the actual values from it, then approve:

```sh
TSUNAMI_ENGAGEMENT_ID='copy-id-from-plan'
TSUNAMI_SCOPE_DIGEST='copy-scope-digest-from-plan'
tsunami_assess approve --engagement "$TSUNAMI_ENGAGEMENT_ID" --scope-digest "$TSUNAMI_SCOPE_DIGEST"
TSUNAMI_START_REQUEST_ID="$(python3 -c 'import secrets; print(secrets.token_hex(16))')"
# Keep this action identifier; reuse it only to recover the same Start.
tsunami_assess start --engagement "$TSUNAMI_ENGAGEMENT_ID" --request-id "$TSUNAMI_START_REQUEST_ID"

TSUNAMI_RUN_ID='copy-id-from-start'
tsunami_assess watch --run "$TSUNAMI_RUN_ID"
```

The CLI submits the explicitly reviewed digest to the approval API. The server compares it with the saved canonical scope inside the same transaction that records approval; a missing or mismatched digest is refused even when bypassing the CLI. There is no blanket `--yes` flag. Matching the digest is not proof that a person understood the plan.

Start creates **one attempt** of the saved built-in procedure. Execution is explicitly **scripted Mecatl**, not a live LLM or autonomous discovery. There is no model selection/fallback flag. `native` is the default executor; `curl-http/v1` selects the already reviewed fixed shell/curl adapter and requires its configured gVisor path. It does not enable arbitrary curl or shell commands. The start response includes the run deadline; dispatch intent is separate from observed worker execution.

Watch polls the small status endpoint every 500 ms and prints changes. It ends when permission is closed/revoked/expired, which does not prove drain or cleanup. It does not download capture bodies. Executor observations include their code and recorded running/exit timestamps; they may be stale or unavailable. `target_requests_reserved` counts durable reservations, not independently observed target arrivals.

**Ctrl+C or a watch timeout only ends observation.** It does not Stop the assessment. Reconnect with status or use the explicit Stop command:

```sh
tsunami_assess stop --engagement "$TSUNAMI_ENGAGEMENT_ID"
tsunami_assess status --run "$TSUNAMI_RUN_ID"
tsunami_assess show --engagement "$TSUNAMI_ENGAGEMENT_ID"
tsunami_assess export --engagement "$TSUNAMI_ENGAGEMENT_ID" --out artifacts/my-assessment
```

Engagement Stop prevents additional child attempts. `stop --run ID` closes only that run. Stop does not depend on the application/model route being ready, and its acknowledgement does not prove worker exit or undo target-side effects. If Stop is not acknowledged, the client says so; it does not claim success or implement an external emergency cutoff.

The export goes to a new directory with mode 0700 and files with mode 0600. Existing directories are not overwritten. A failed write can leave a partial new directory, which must not be treated as a complete bundle. Export is a snapshot and can include incomplete tests. Review the progress and evidence rather than treating the presence of a report as completion.

`confirmed` means the validator has evidence supporting a finding **candidate**. The CLI does not approve it, assign severity/CVSS or generate a human signoff. `no_issue_observed` covers only the completed procedure. `inconclusive`, including invalid/expired test sessions, is not a pass. The existing review API remains separate; the CLI currently exports its results rather than adding a review command.

## Recover without replaying work

| Situation | Action |
|---|---|
| Draft response lost | Use `engagements`, then `show --engagement ID`; inspect the saved draft before creating another. |
| Start response lost | Use `show --engagement ID` to read child IDs, or explicitly repeat `start` with the same `--request-id`. The client never retries automatically. |
| Watch disconnected or timed out | Reconnect with `status`/`watch`, or explicitly Stop. Disconnection is not revocation. |
| Optional connector/model configuration failed | The state/revocation API can remain available; use explicit Stop and inspect its acknowledgement. |
| Kubernetes cleanup delayed | Read permission, local activity and cleanup separately. Missing API objects alone do not prove remote process exit. |
| Account expired/configuration changed | Retain the inconclusive/failed attempt. A trusted operator must fix prerequisites and create a new approved attempt. |

Start requires `--request-id` (32 lowercase hexadecimal characters), sent as `idempotency_key`. The same key/request identifies the original run across response loss/restart and after Stop; a different request cannot reuse it. A new explicit attempt needs a new key and normal permission/budgets. Returned state can be stopped, failed or completed; recovery does not launch replacement work. Retain the identifier with the assessment ID; it is an action ID, not a credential. Existing engagement limits and durable reservations still apply. This is not exactly-once remote execution.

## API mapping and output

`applications` uses `GET /v1/applications`; `plan` creates an engagement; `show` reads its snapshot; `approve` and `start` call its approval/child endpoints. `status` and `watch` use `GET /v1/runs/{id}/status`. `stop` calls the selected run/engagement Stop endpoint. `export` uses the bounded engagement export and existing artifact materializer. `engagements` lists this tenant's saved engagements for recovery.

Use `--json` for structured output. Watch emits a sequence of JSON objects, not a single array; use `status --json` for one snapshot. A successful watch command means observation ended normally, not “no vulnerability found.” Command exit codes distinguish operation/observation errors, not finding severity. Target response bodies stay out of console output; they are deliberately present in protected evidence exports.

The CLI reuses the existing redirect-rejecting, proxy-independent operator client. It accepts plain HTTP only for loopback/localhost development connections and requires HTTPS for remote API origins, with normal certificate verification. The operator workstation, localhost resolution/port-forward and selected control API are trusted. It is not an authenticated customer portal, does not support an insecure TLS override, and does not add customer identity/secret management. Use a matching server/CLI build; general cross-version negotiation and limits negotiation are not implemented.

## Reproduce the qualification

```sh
# After matching images, owned VAmPI setup, and the held-request fixture setup:
make stop-lab-setup CONTEXT=kind-tsunami-lab
make journey-verify CONTEXT=kind-tsunami-lab

# Optionally validate these fresh exports against the local reference schemas:
python3 scripts/verify-journey.py --context kind-tsunami-lab --toolkit ../vapt-toolkit
```

The verifier uses a temporary 0600 operator-token file, runs the compiled CLI, and removes the file on exit. It records the vulnerable/fixed pair, rejected draft/wrong-digest/stopped launches, unreviewed exports, and explicit Stop after a read-only watch timeout on a real gVisor/curl worker. Its automated lab approvals are not customer authorization. It does not use the standalone client to contact targets; diagnostic target counters belong only to the owned test driver.

Native tests also cover expired credentials producing inconclusive evidence, a committed start whose response is lost without automatic retry, cross-tenant denial, arbitrary target input rejection and private export permissions. The VAmPI setup regression covers preservation of other plans/tenants. A setup rendering fix handles kubectl's consecutive JSON objects for multi-document YAML, including tripwire and model-fixture resources. No model calls are needed for this milestone.

See [the retained CLI campaign](../examples/operator-journey/README.md) for exact builds and results. More useful HTTP procedures, independent security review, real customer onboarding, stronger customer isolation, secret/evidence lifecycle and broader failure/load qualification remain separate work.
