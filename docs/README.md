# Documentation

Start with [Getting Started](getting-started.md), [current functionality](current-status.md), then [architecture](design.md) and [security boundaries](security.md). Those four documents are the default review path.

## Getting Started

- [Getting Started](getting-started.md): source verification, explicitly unqualified fresh-lab bootstrap, UI access, recovery and safe shutdown.
- [Controlled demo walkthrough](demo-walkthrough.md): presentation in an already prepared environment.
- [Selected lab configuration](demo-configuration.md): exact synthetic applications and procedures.

## Architecture and security

- [Architecture, states and activity events](design.md).
- [Trust boundaries and limitations](security.md).
- [Mecatl integration](mecatl-harness.md): engine facilities versus Tsunami enforcement.
- [Stop lifecycle](stop-lifecycle.md): permission, local activity, execution and cleanup.
- [Security reporting policy](../SECURITY.md).

## Developer and operator guides

- [Contributing](../CONTRIBUTING.md), [local UI](local-ui.md), [operator CLI](operator-cli.md).
- [Application onboarding](application-onboarding.md), [configured applications](application-adapters.md), [binding preparation](binding-preparation.md).
- [Engagement/API contracts](engagements.md), [property authorization](property-access.md).
- [Sandbox lab](sandbox-lab.md), [fixed curl adapter](tool-execution.md).

## Verification and evidence

- [Verification](verification.md): commands, build provenance, qualification limits and historical failures.
- [Recorded complete browser journey](../examples/demo-journey/README.md).
- [Original failed isolation checks](../examples/isolation/README.md).
- [Active-request Stop/failure records](../examples/stop-lifecycle/README.md).

## Optional experiments

These remain implemented/tested references. They are unnecessary for the default scripted demo and are not automatically qualified on its latest deployment.

- [Authorization catalog](assessment-catalog.md).
- [Model gateway](model-path.md), [inference diagnostics](inference-diagnostics.md), [historical free-model trial](openrouter-free.md).
- [Private connectivity](private-connectivity.md), [connector TLS](connector-tls.md).
- [Toolkit mapping](toolkit-mapping.md): local read-only compatibility work; no vendored or executed toolkit.

## Historical references

The [historical-results reference](validation-results.md) preserves failed isolation context and links to public history. Original planning/review documents were copied into an external local archive before consolidation. They remain accessible at [public commit `44d20cb`](https://github.com/Sanskarzz/tsunami/tree/44d20cb63cde7d3c5f0fde20113404222f185d5d/docs).

Existing `examples/` records, hashes, failed outcomes and source fingerprints remain unchanged. Compatibility pages preserve incoming evidence links; they are not competing roadmaps.
