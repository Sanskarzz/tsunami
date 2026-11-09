# Run the owned-lab showcase

This is a short systems demo: a real owned VAmPI deployment, constrained agents, evidence-backed outcomes, scope rejection, direct-network checks and explicit Stop. Execution is scripted Mecatl. It is not blind discovery, customer authorization or a production readiness claim.

For the current presentation, use the [canonical demo guide](demo-walkthrough.md) and [selected configuration](demo-configuration.md). This CLI showcase is supporting qualification tooling. D1–D9 product delivery is recorded in [the tracker](track.md); older D2 results below retain their original build scope.

[The fresh D2 record](../examples/demo-configuration/README.md) qualifies the D1 working-tree build with seven showcase cases and four browser runs. It preserves earlier records; its credential window and history capacity must be rechecked before presentation.

## Before the call

Use the repository root. The dedicated `kind-tsunami-lab` cluster must already be provisioned with gVisor/Cilium and the base Tsunami deployments. These commands select `.local/sandbox.kubeconfig` internally; they do not change the normal kubeconfig context, install a new CNI or create a cluster.

```sh
make verify
make showcase-check
```

The check reads local/Kubernetes metadata and the state/operator API. It makes no target requests. It requires the dedicated Kind node/ownership marker, ready Cilium with `always` enforcement, configured gVisor/curl execution, the fixed owned target origins and Service addresses, ready deployments, sufficient lab history capacity and matching registry credentials. It refuses active/pending assessments and diagnostic Jobs. The permanent offline model fixture is a trusted service, not an active assessment.

VAmPI JWT expiry is inspected without verifying a signature. The check requires at least fifteen minutes of hinted lifetime by default, but does not certify account validity. Actual assessments still verify identities. Opaque Stop/property credentials have no local expiry hint. Private/model routes can be unavailable because this demo does not use them.

If data or credentials need refresh, explicitly run:

```sh
make showcase-prepare
make showcase-check
```

**Prepare resets the disposable owned VAmPI databases/accounts, Stop counters and property fixture.** It rotates their generated credentials/proof data. It builds and loads new uniquely tagged native/tools images, avoiding stale mutable-tag caches. It preserves run/evidence history, unrelated configured plans and operator/model/private settings. It does not renew private certificates, call a model, clear tenant/global stop gates or delete history to make room.

Preparation refuses active Pods and nonterminal run permissions before changing the lab. Use this single-operator lab exclusively during setup: there is no distributed maintenance lock preventing another operator from launching immediately after the check. Keep sensitive raw setup inputs in ignored `.local/`, not in a share bundle.

Run and rehearse:

```sh
make showcase

# Optional: include the four property-level controls.
make showcase-extended

# Same extended run with the separately obtained local toolkit schemas:
python3 scripts/showcase.py run --include-property --toolkit ../vapt-toolkit
```

`showcase` does not reset data or silently recover a failed prerequisite. It runs credential-free curl network diagnostics with reachable positive controls before/after, then the compiled CLI's VAmPI vulnerable/fixed and held-request Stop controls. The extended version adds known private-field exposure, correct omission, an inconclusive redacted value and property Stop.

Every run prints **AUTOMATED OWNED-LAB APPROVAL**. The driver supplies the known fixture scope/digest for these controlled integration checks; that is not a human/customer approval or finding review. There is no arbitrary target or model flag. All paths go through the existing enforcement and evidence APIs.

## Five-minute presenter path

Have one terminal and the resulting `artifacts/showcase/run-.../index.md` open. Rehearse first; keep the most recent successful index available if the laptop/cluster is slow during the call. Be clear when showing retained evidence rather than a live run.

1. **Goal, about 30 seconds.** “I wanted to make the execution boundary concrete. The agent is allowed to request a small set of reads; the service owns scope, credentials, evidence and revocation.” Point to the Go broker/controller and Mecatl separation. The gateway is trusted infrastructure, not code the agent can edit.
2. **Inputs and permission, about 45 seconds.** Open the core transcript's plan preview. Explain the two supplied identities/legitimate records, exact paths and addresses, limits and scope digest. Preparation generates bindings; it does not discover intent or approve execution. Show the rejected draft/wrong-digest actions. For a manual demonstration, use the existing [CLI guide](operator-cli.md) and actually review the digest before approving.
3. **Real owned application, about one minute.** Open the vulnerable VAmPI report and candidate evidence, then the fixed control. “The same declared access rule gets different evidence. This is a known deliberately vulnerable app; I am not claiming a new zero-day.” The candidate is unrated and needs impact/severity review.
4. **Enforcement, about 45 seconds.** Show a blocked capture in the index and the network record. The blocked intent proves broker rejection. Separate curl tests attempt direct Service/Pod/API/DNS/app-tripwire connections, including `--resolve` and `--connect-to`; the reachable controls prevent a dead target from looking like a denial. These selected IPv4/TCP checks are not a general sandbox-escape or metadata/IPv6 qualification.
5. **Stop, about one minute.** Show the held target's counters, watch timeout, explicit Stop and status. “Ending observation doesn't revoke permission. Stop persists revocation and cancels broker work; Kubernetes cleanup is separate.” The held worker gets a CRI runtime/container image observation. API object absence alone is not process-exit proof, and remote target effects cannot be undone.
6. **Limits, about 30 seconds.** “This is a working owned-lab slice. I would next focus on independent procedure review, customer identity/isolation, secret/evidence lifecycle and broader failure testing.” Do not claim production sixty-second Stop, 1,000 agents, arbitrary shell safety, all toolkit features or a live-model accuracy result.

The property controls are optional follow-up material: an allowed shared object may still expose forbidden fields. The ambiguous example is useful for explaining why a generic 200 or masking policy is not automatically a clean result.

## Results and evidence

Each rehearsal writes a fresh private directory under `artifacts/showcase/`:

- `index.md`: concise results with links.
- `showcase.json`: exact scenario/run/engagement IDs, blocked capture IDs, evidence paths, observed outcomes, approval mode and actual client/control build identities.
- `prerequisites.json`: configured runtime/CNI, target metadata and unsigned expiry hints, without credentials.
- `network/`: positive-before, runner, positive-after logs and runtime/image observations.
- `object/` and optional `property/`: full CLI transcripts and materialized per-case exports.

Do not send the entire `artifacts/` or `.local/` tree. A successful local result is not an automatic publication/redaction step. Review the selected synthetic demo directory for credentials/data before sharing it. Evidence contains raw synthetic private values deliberately; hashes provide integrity, not authenticity against a compromised trusted broker. The repository's [retained rehearsal](../examples/showcase/README.md) identifies the sanitized subset and actual build tested.

## Recovery and reset

| Failure | Response |
|---|---|
| Missing dedicated cluster/CNI/runtime/base deployment | Follow the sandbox/base deployment guide first. The showcase refuses another cluster rather than altering it. |
| Expired/near-expiry credentials or stale registry fingerprint | Use explicit `showcase-prepare`, then rerun the check. This invalidates old setup bindings; keep historical evidence and create new approved attempts. |
| Active/pending work | Inspect its exact run/engagement and explicitly Stop if appropriate. Preparation must not silently kill unrelated work. Watch timeouts do not Stop it. |
| Start response/connection lost | Read the engagement's stored child IDs before another start. The CLI does not retry uncertain starts automatically. |
| Rehearsal step fails | Keep its private logs and `showcase.json` with `passed: false`. Fix the cause, check for unfinished work, then create a fresh rehearsal. Do not edit the old result into a pass. |
| Control API unavailable | Acknowledge that normal Stop is unavailable; inspect trusted platform controls. There is no implemented independent external network cutoff in this package. Do not declare revocation/exit just because a request failed. |
| Lab engagement history near its 100-record limit | Preserve/export history and plan explicit state maintenance or a fresh owned instance. The demo never wipes its state database automatically. |
| Network diagnostic fails | Treat direct-network containment as unqualified. Inspect both positive controls and the runner paths; do not run the assessment to obtain a nicer result. Diagnostic Jobs have bounded deadlines. |

Child verifiers attempt to Stop only their own created engagements in cleanup. A forcibly killed driver or a platform failure can still leave uncertain work; inspect it explicitly before reset. Inactive Job history and failed records are not silently removed.
