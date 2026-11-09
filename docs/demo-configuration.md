# Selected owned-lab demo configuration

Updated: 2026-11-09. This operating contract began in D2 and now selects the D9-qualified [five-minute product demo](project-goal.md). Status belongs in [the tracker](track.md). [D9's retained record](../examples/demo-journey/README.md) qualifies the connected D1–D8 journey; [D2](../examples/demo-configuration/README.md) remains a historical build record.

Selected D9 images are `tsunami:showcase-20261109t144510z` and `tsunami-tools:showcase-20261109t144510z`. The 2026-11-09 campaign passed actual runtime/native/curl network checks, six browser assessments and a paced 300.000-second dry run. It preserved all original history and ended with eight engagement slots. Readiness is time-limited; recheck before presenting. This was automated browser operation with a supplied script, not recorded human narration.

## Fixed environment and entry point

- Dedicated single-node Kind cluster `tsunami-lab`, context `kind-tsunami-lab`, kubeconfig `.local/sandbox.kubeconfig`.
- Cilium `always` policy enforcement and gVisor RuntimeClass `gvisor` mapped to `runsc`. Require actual CRI inspection and selected network checks; configuration objects alone do not qualify isolation.
- Existing Go control plane/bbolt PVC, separate worker Jobs with no Kubernetes token, and the fixed `curl-http/v1` adapter through the trusted broker.
- Scripted Mecatl for every configured application assessment; zero external model calls. Optional private/model routes are outside this demonstration and may be unavailable.
- Running frontend: loopback-only `http://127.0.0.1:8080/ui/`, using tenant operator access. No public ingress or customer sign-in is added.

Do not substitute the original `kind-kind` cluster or fall back to ordinary-container execution. Do not silently change an origin, address, account/path pair, tool or Stop gate to make qualification pass.

## Selected applications and procedures

| Application ID | Exact origin | Procedure | Expected observation |
|---|---|---|---|
| `vampi-vulnerable` | `http://vampi-vulnerable.tsunami-app-lab.svc.cluster.local:5000` | `vampi-vulnerable/object-access/v2` | `confirmed`; one unrated candidate. |
| `vampi-fixed` | `http://vampi-fixed.tsunami-app-lab.svc.cluster.local:5000` | `vampi-fixed/object-access/v2` | `no_issue_observed` for this rule. |
| `stop-lab-run` | `http://tsunami-stop-fixture.tsunami-stop-lab.svc.cluster.local:8080` | `stop-lab-run/object-access/v2` | Held probe, durable revocation and independently observed cancellation. |
| `property-vulnerable` | `http://tsunami-property-fixture.tsunami-property-lab.svc.cluster.local:8080` | `property-vulnerable/property-access/v1` | Known owner-only value exposed; `confirmed`. |
| `property-filtered` | Same exact property origin above | `property-filtered/property-access/v1` | Expected public object with private field omitted; `no_issue_observed`. |
| `property-ambiguous` | Same exact property origin above | `property-ambiguous/property-access/v1` | Redacted/unknown value; `inconclusive`. |
| `property-stop` | Same exact property origin above | `property-stop/property-access/v1` | Held property probe, revocation and cancellation. |

Each registry plan pins the current exact Service address and account/path pairs. Service addresses, object IDs, tokens, proof values, plan digests and credential fingerprints are generated facts, not constants to copy between deployments. Property cases use distinct approved `/cases/...` paths despite sharing an origin.

The normal assessment makes seven approved GETs within an eight-request allowance and a 90-second run deadline. Two identity checks and two own-object reads precede the peer probe; owner-object and peer-identity rechecks follow it. The worker also submits a fixed out-of-scope intent, rejected before contacting its destination. Onboarding preflight separately allows four legitimate reads, at most 15 seconds, with no cross-user probe. A held probe must be stopped while it is actually active and before the broker request timeout; explain Stop before starting it.

VAmPI is a pinned upstream owned lab deployment with its vulnerable/fixed switch. The property and Stop applications are separate synthetic fixtures. Do not present them as a combined unknown customer application. Existing catalog/private/model experiments are not part of this selection.

## Prepare without discarding history

From the repository root, inspect readiness first:

```sh
make showcase-check
# Include all four property controls in the read-only prerequisite check:
python3 scripts/showcase.py check --include-property
```

These checks make no assessment-target requests. They inspect owned infrastructure, registration, startup route status, history capacity and unsigned VAmPI JWT expiry hints. They do not prove that a session is valid; assessments recheck identities. Opaque property/Stop credentials have no local expiry hint.

If the disposable credentials need refreshing, explicitly run:

```sh
make showcase-prepare
python3 scripts/showcase.py check --include-property
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
python3 scripts/showcase.py check --include-property
```

Run infrastructure/assessment/browser qualification sequentially so guards and target counters do not interfere. `make demo-rehearse` checks matching deployed asset bytes, then runs the real Add -> four-read preflight -> exact approval -> actual activity -> finding/evidence/remediation -> active Stop -> linked fixed-app retest path. It records six scheduled presentation stages and fails late stages; property leak/filter/ambiguity run afterward outside presentation time. `make verify-ui-demo` performs the same checks without pacing and does not establish a five-minute rehearsal. Approvals are automated owned-lab actions, never customer authorization or human finding review. The existing seven-case `make showcase-extended` remains an optional separate CLI campaign that consumes seven additional engagement slots; it was not repeated in D9. Plan capacity before every campaign and never silently discard history.

Retain actual source/binary and image identities, CRI observations, target cancellation/counters, exact run/engagement IDs, result exports and prerequisite timestamps. A dirty working-tree build must be labeled and fingerprinted rather than described as a clean-commit reproduction. Historical exports remain untouched. Keep failed attempts with their failure status.

## Open the running UI

In a separate terminal:

```sh
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
kubectl --context kind-tsunami-lab -n tsunami port-forward \
  --address=127.0.0.1 service/tsunami-control 8080:8080
```

Open `http://127.0.0.1:8080/ui/` and connect privately as described in [the UI guide](local-ui.md). Do this before screen sharing; do not put tokens in commands, URLs, screenshots or chats.

Under **Add application**, select the existing reviewed `vampi-vulnerable` integration, confirm disposable synthetic data, approve the four legitimate reads, prepare the new draft, then review/approve its exact scope and explicitly Start. The backend reuses trusted lab access without revealing stored tokens to the page. From Evidence, explicitly select `vampi-fixed` in linked retest, enter a fresh authorization reference, review changed bindings, approve its own scope and Start separately. Property and Stop controls use their own registered plans and approvals. Follow [the exact five-minute script](demo-walkthrough.md#five-minute-presentation). The historical D2 build predates this presentation and remains labeled separately.

## Readiness window and recovery

VAmPI setup issues one-hour credentials. Preflight requires at least 900 seconds of hinted remaining lifetime by default. This is a readiness threshold, not permanent certification or verified session lifetime. Recheck immediately before D9/presentation; if it fails, explicitly refresh between assessments, create new bindings/drafts and repeat the relevant qualification. Do not lower the threshold to hide expired access.

Keep the preflight timestamp and build identities with every qualification record. A source/image mismatch, changed Service address, invalid identity, failed positive network control or unavailable observation invalidates the corresponding claim. Investigate and record the failure; do not replace it with a simulated success.

A failed/lost Start response requires reading saved child IDs before another attempt. An unacknowledged Stop remains uncertain. If the control API is unavailable, use trusted platform inspection and acknowledge that normal Stop is unavailable; there is no independent external egress cutoff in this package. Preserve/export history before separately planned capacity maintenance; never wipe the lab database to free engagement slots.

Local artifacts may contain raw synthetic evidence and private setup material. Share only a reviewed selected subset. General redaction, production identity/isolation, hostile-process containment, broader partitions/load, capacity and a production Stop guarantee remain outside this qualification.
