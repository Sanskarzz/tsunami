# Contributing

Start with [Getting Started](docs/getting-started.md), [current scope](docs/current-status.md), [architecture](docs/design.md) and [security boundaries](docs/security.md). Changes should preserve this owned-lab prototype's enforcement and evidence contracts.

## Verify a change

Install the source prerequisites and Python test requirements from Getting Started, then run:

```sh
make verify
make verify-ui-retest
make build
```

For UI changes use the relevant disposable local browser target: `make verify-ui-activity-browser`, `make verify-ui-onboarding-browser`, `make verify-ui-findings-browser` or `make verify-ui-retest-browser`. They require Node 22 and Chrome; no Kubernetes or provider credentials. Test fixtures are not product data.

Do not run deployment/assessment campaigns against an existing lab without operator authorization. They can rotate access, consume persistent history capacity or interrupt work. Source and local browser tests are the default checks.

## Changes and review

Use a small branch and pull request describing the problem, resulting behavior, tests and limits. Reproduce only against owned synthetic targets. No response SLA, CLA or signoff requirement is established. Repository licensing is pending; resolve licensing before submitting material intended for reuse.

Keep approval, credentials, request admission, durable Stop, evidence and verdict authority outside the worker. Do not retry uncertain security actions automatically, broaden scope, trust model prose as confirmation or turn missing evidence into a pass. Procedure/API changes need matching negative tests and documentation.

Keep historical examples and failed outcomes intact. `make verify` checks evidence hashes/indexes; do not regenerate them to hide failures. Explain version transitions and preserve older records. Never commit `.local/`, runtime databases, credentials, customer data or unreviewed artifact dumps.

Use [SECURITY.md](SECURITY.md) for suspected vulnerabilities. Public issues/PRs suit non-sensitive defects and documentation improvements. A passing local check implies no deployment, push or release.
