# Selected owned-lab configuration

This is the target/procedure contract. [Getting Started](getting-started.md) owns setup/recovery, [verification](verification.md#recorded-deployment-and-qualification) owns dated build results, and [the walkthrough](demo-walkthrough.md) owns the prepared presentation.

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

## Operation and qualification

Use [Getting Started](getting-started.md) for readiness, explicit maintenance, loopback access and qualification commands. The [UI reference](local-ui.md) explains authentication and observations. [Verification](verification.md) preserves historical build identities, failed controls and unqualified paths. Readiness is time-limited; do not infer current credential health from a historical pass.
