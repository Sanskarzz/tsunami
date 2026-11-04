# Engagement foundation

An engagement groups an approved target and procedure selection, child runs, evidence and checklist progress. This is the first step toward using more of the toolkit workflow. The later [application adapter](application-adapters.md) adds registered owned applications and reviewed exports. Arbitrary customer URLs, shell commands, scanners, MCP tools and a multi-agent scheduler are still not enabled.

## Authority and lifecycle

An authenticated tenant operator creates a draft with a name, authorization reference, lab scenario, connectivity and selected procedure IDs. The service selects the configured target, resolves the procedure versions and fixes their digests. A separate operator approval pins the scope digest before any child run can be created. A reference is an operator attestation, not automated verification of a legal authorization document.

The digest covers tenant, target, connectivity, scenario, procedure versions, expiry and limits. It is a consistency check in the trusted database, not a signature against a compromised server. The current plan is immutable. A new scope needs a new engagement.

The API does not accept caller-selected tenant, target URL, `authorized: true`, limits, capture IDs or completed percentages. Run tokens cannot access these operator routes. Reads and exports check the authenticated tenant; knowing another engagement ID is insufficient.

Current fixed limits are 20 child runs, two concurrent children, one active run per procedure, a one-hour approval window, and 100 retained engagements per demo tenant. Run deadlines are clipped to engagement expiry. Failed runs keep their reservations. These are lab admission limits, not customer-wide monetary budgets or demonstrated capacity.

Child creation and its engagement reservation commit in one bbolt transaction. Engagement Stop marks the parent stopped and all nonterminal children stopped in one transaction, then cancels their tracked HTTP/model contexts. New children are refused afterward, including after restart. Policy/version drift may block execution/export, but it must not prevent Stop. Existing per-run and tenant/global stops remain in place. Tenant/global gates also deny new engagement children.

The current process still handles the broker, store and controller together. These transactions do not establish distributed revocation across future executors. The existing connector expiry/cutoff limits still apply; Stop cannot undo a target's accepted action.

## Progress and findings

Progress is derived from linked runs and trusted validation of broker captures. An agent cannot upload a finding or mark a checklist item complete. The latest attempt determines procedure progress. Failed/stopped work and missing prerequisites remain separate from completed work. Skipping requires an operator reason and is allowed only before the procedure has been attempted; it does not count as completed.

Completed means the selected procedure captured its legitimate baselines and probe. The probe can still be inconclusive, for example the intentional missing-object control. Completion never means the whole application passed a security assessment. Only the selected Tsunami procedures or configured application plan contribute to this progress; importing 26 checklist files does not execute their tests.

The export recomputes finding candidates using the synthetic validator. Repeated confirmed observations of one pinned procedure are grouped and retain their evidence links. A later clean run does not silently erase an earlier candidate. Candidates remain `needs_human_review`, with no invented severity or CVSS score. The subsequent application milestone adds an explicit review API and finding/evidence schema export. Individual reviewer identity, remediation workflows and customer impact assessment still need further work.

## Evidence and toolkit compatibility

An export contains the engagement view, a toolkit-shaped engagement projection, finding candidates, a report draft and capture artifacts. Each artifact includes its run/engagement/procedure provenance. SHA-256 names its exact serialized bytes; base64 transport preserves those bytes when JSON is reformatted. Artifact paths are generated from hashes. They never instruct the server to read arbitrary files.

Hashes are not signatures, immutable storage or proof against a compromised broker. Current evidence includes synthetic response bodies; general customer redaction, storage authorization, encryption, retention and deletion remain product work. Provider/operator/run credentials are omitted from public run representations and exports.

The optional local checker validates the engagement projection against the friend's actual schema, checks artifact hashes and candidate evidence references, and records the schema hash and validator version. It does not execute toolkit code. Remote schema references are refused. The subsequent application milestone extends this checker to finding/evidence contracts, review consistency and materialized files. Legacy records here retain their original format and qualification claims. See [jsonschema validation behavior](https://python-jsonschema.readthedocs.io/en/stable/validate/).

## Routes

All routes require the authenticated tenant operator, never a worker token.

| Route | Operation |
|---|---|
| `POST /v1/engagements` | Create a draft for configured lab targets |
| `GET /v1/engagements` | List this tenant's engagements |
| `GET /v1/engagements/{id}` | Scope, run links and derived progress |
| `POST /v1/engagements/{id}/approve` | Approve the pinned scope; body `{}` |
| `POST /v1/engagements/{id}/runs` | Reserve/create an approved procedure run |
| `POST /v1/engagements/{id}/skip` | Record a reason for an unattempted procedure |
| `POST /v1/engagements/{id}/stop` | Revoke the engagement and active children; body `{}` |
| `GET /v1/engagements/{id}/export` | Tenant-authorized snapshot and artifact bytes |
| `POST /v1/engagements/{id}/reviews` | Later review API; stopped engagement and matching evidence required |
| `GET /v1/engagements/{id}/findings.csv` | Reviewed findings with CSV output guards |

Run creation defaults to scripted. Live engagement runs require the exact `fixture` or `openrouter-free` provider policy; the latter additionally requires the existing free price policy. Private model execution is still disabled. Existing lab run routes remain for regression tests and are outside the engagement allowance; that allowance is not an account-wide budget.

## Reproduce

```sh
make engagement-fixture-local
make toolkit-inventory TOOLKIT_PATH=../vapt-toolkit

# Use the actual campaign directory printed by the first command.
python3 scripts/toolkit.py validate-export \
  --toolkit ../vapt-toolkit \
  --bundle artifacts/engagements/offline/campaign-EXAMPLE/vulnerable/bundle.json \
  --out artifacts/engagements/schema-validation.json

export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
make private-deploy CONTEXT=kind-tsunami-lab
make engagement-verify CONTEXT=kind-tsunami-lab
make engagement-private-verify CONTEXT=kind-tsunami-lab
```

Local execution uses real Mecatl/Responses code with loopback fixtures; it does not establish process isolation. Kubernetes verification uses gVisor worker Jobs; private verification uses the mTLS connector. Both Kubernetes commands select scripted execution even if a free model is configured. No external model calls are needed.

Each campaign preserves a fresh output directory. The local demo exports before Stop; the Kubernetes collector retrieves the later stopped snapshot. Their authorization/status fields reflect those different snapshot times. The verifier checks draft/after-Stop dispatch refusal, all five procedures on vulnerable/fixed apps, progress, and three versus zero candidates. Active-request Stop, racing creation, tenant isolation and restart behavior are covered by native tests. Neither campaign measures production scale or independent discovery.

See [the expanded product plan](product-roadmap.md) for the remaining work and required evidence.
