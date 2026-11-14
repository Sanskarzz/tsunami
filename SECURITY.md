# Security policy

Tsunami is an owned synthetic-lab prototype, unsupported for production customer data or unrestricted targets. Read the [security model](docs/security.md), [limitations](docs/current-status.md#unimplemented-or-unqualified) and [qualification record](docs/verification.md).

## Reporting

No private reporting channel, dedicated security contact, supported-version schedule or response SLA is currently documented. Do not assume public issues, PRs or commit comments are confidential.

Do not publish credentials, customer information or exploits against third parties. If a private channel is needed, first request a confidential contact method from the repository owner without sensitive details. A non-sensitive public report can identify the component/version, expected versus observed behavior and minimal owned-synthetic reproduction. Separate observations from uncertainty.

An exposed credential needs revocation/rotation through its issuer. Deleting a file or rewriting history does not invalidate it or erase copies. Preserve an appropriate private incident record; do not paste secrets into reports.

## Assessment boundaries

Use only explicitly authorized owned/local/intentionally vulnerable applications with disposable data. An authorization reference does not prove legal ownership. The control plane, broker, store, infrastructure and local operator workstation remain trusted assumptions.

Stop persists revocation and cancels tracked local work. It cannot undo remote effects, prove exit on a disconnected node or remain available after every trusted-service failure. See [Stop lifecycle](docs/stop-lifecycle.md). Closing a browser, pausing observation or stopping a port-forward does not revoke an assessment.
