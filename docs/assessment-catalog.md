# Authorization procedure catalog

This milestone turns the single hardcoded check into five small, versioned procedures. They are original engineering definitions informed by the VAPT toolkit. A security engineer has not reviewed these definitions yet; each one records that limitation.

All requests still go through the broker. There is no shell, scanner, browser, or external MCP tool. The procedures exercise developer-owned synthetic fixtures only.

## Procedures

| Procedure | Approved rule | Vulnerable fixture | Fixed fixture |
|---|---|---|---|
| `opaque-ownership/v1` | Bob must not read Alice's real private object, even with its opaque ID | Confirmed forbidden read | No issue observed; target conceals it with 404 |
| `shared-access/v1` | Alice explicitly shares one object with Bob | Approved access; no finding | Approved access; no finding |
| `role-boundary/v1` | Alice is an administrator; Bob may not read the admin summary | Confirmed forbidden read | No issue observed; 403 |
| `anonymous-access/v1` | A private object requires an authenticated, authorized account | Confirmed read with no Authorization header | No issue observed; 401 |
| `missing-object/v1` | The test object has no established existence | Inconclusive | Inconclusive |

The anonymous alias is a specific capability. The agent cannot supply arbitrary Authorization headers or remove authentication from other procedures.

Each definition under `internal/assessment/procedures/` contains the purpose, review status, allowed GET paths and account pairs, legitimate baselines, probe, expected access rule, request budget, and remediation. The expected access rule describes what should be allowed. It does not tell the worker whether the application contains the bug.

## Where enforcement happens

An authenticated operator selects a known procedure ID when creating a run. The API rejects custom definitions and unknown IDs. The server snapshots the definition and its digest into durable run state; the worker cannot replace it. Definitions are embedded into the binary, not downloaded from target responses or loaded from an agent-writable directory.

The broker keeps the existing approved origin, port, and dial-address policy. It uses the selected procedure's exact path catalog and checks each account/path pair. Selecting one procedure does not grant the other procedures' paths or accounts. Legacy runs retain their original path catalog. Procedure runs reserve at most six target requests; denied actions still count toward the existing bounded action/evidence limits.

Private grants include the procedure ID and digest. The connector independently loads that version, verifies the digest and account/path pair, then checks the destination at the actual connection point. A stale or unknown procedure fails closed. The existing signature, replay store, mTLS identity, expiry, and Stop controls still apply. Changing a procedure without matching its stored digest does not silently reinterpret an active run.

The worker's two tools remain `http_request` and `finish_assessment`. Its account enum is built from the server-owned procedure. In scripted mode it follows the procedure's reads. In model mode the gateway builds the prompt and tool schemas from the same pinned procedure. Model text cannot confirm a finding.

## Evidence rules

The validator requires successful legitimate baselines before drawing a conclusion from the probe. It compares the returned object ID, owner, and content against the approved fixture facts. An error page with HTTP 200, the wrong object, an interrupted response, or missing baselines stays inconclusive. A later failed probe does not fall back to an earlier successful capture.

Approved sharing returning another owner's object is not a vulnerability. A nonexistent object's rejection remains inconclusive even when both legitimate baselines work. A 404 for a known existing object counts as denial here because these synthetic fixture objects are immutable and their existence was established. That assumption must be reviewed for a real application where an object could disappear between requests.

The fixture application has its own data and authorization implementation. It does not construct responses from the procedure definitions. Evaluation expectations are also kept in the operator harness, outside worker configuration. This allows a mistaken definition or a changed fixture response to fail evaluation.

Reports include procedure version/digest, review status, evidence IDs, reproduction requests, and remediation. They are drafts for human review. They do not assign a real-world severity automatically.

Premature completion returns trusted `required_reads` feedback while permission and request budget remain. The broker lists only already-approved account/path pairs; it does not execute them or accept the model's verdict. At most two feedback responses are persisted per run. After that, or after the target budget is spent, incomplete validation terminates as inconclusive. These counters survive restart and do not refund or increase target/model budgets. Stop still revokes the run during this process.

## Repeatable evaluation

```sh
# Two repetitions: five procedures x two fixture variants x two = 20 runs.
make catalog-local
make catalog-fixture-local

export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
export CONTEXT=kind-tsunami-lab RUNTIME_CLASS=gvisor
make private-deploy CONTEXT=kind-tsunami-lab
make catalog-verify CONTEXT=kind-tsunami-lab
make catalog-private-verify CONTEXT=kind-tsunami-lab
```

The local fixture mode uses Mecatl's actual Responses adapter against a deterministic API fixture. It is not a real model. Kubernetes evaluations run real gVisor worker Jobs; private evaluation uses the mTLS connector. They default to scripted execution even when OpenRouter is configured.

The evaluator records each run's expected/actual result, terminal state, evidence completeness, duration, reserved target requests, and model usage when applicable. It reports matched cases, confirmed planted bugs, missed planted bugs, false positives, and inconclusive results. Incomplete evidence or a failed worker cannot pass merely because the expected result is inconclusive. Failures and partial results are retained, and exports use exact run IDs.

For each full 20-run suite, the expected totals are six confirmed planted bugs, ten no-issue observations, and four intentional inconclusive results. A perfect scripted/fixture result tests integration and validation; it is not LLM accuracy.

## Optional free-model sample

After configuring and checking the free provider as described in `openrouter-free.md`:

```sh
python3 scripts/evaluate-catalog.py --mode free --repeat 1 \
  --assessments opaque-ownership/v1,shared-access/v1 --allow-model-calls
```

This is an explicit external-inference action. It requires `openrouter-free` and the free price policy atomically at run creation, in addition to the existing zero-price transport guard. There is no paid catalog mode. Free trials are limited to one repeat and at most two procedures: four runs, each with at most eight model attempts. Vendor allowance/availability may still stop the trial; there is no paid fallback.

Private catalog model execution remains disabled. Less guided, held-out datasets, repeated live-model evaluation, additional applications, and external security review remain future work. The current prompts provide known inventory and access rules, and fixture hostnames are visible. These are guided samples, not blind vulnerability discovery. See [recorded results](../examples/assessment-catalog/README.md).

The initial live trial matched 3/4 cases; one incomplete-baseline assessment stayed inconclusive. A single follow-up after adding bounded feedback matched 2/4: both shared cases recovered using feedback, while two model turns failed without usable completion or target traffic. The generic historical error code does not establish their precise upstream cause. Both trials are retained; this is not evidence of improved overall model accuracy. OpenRouter reported $0 usage after 28 free requests across these trials.
