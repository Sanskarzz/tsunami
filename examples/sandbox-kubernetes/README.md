# Synthetic assessments with gVisor workers

Recorded on 2026-11-01 in the separate `kind-tsunami-lab` cluster. The control plane configured `runtimeClassName: gvisor` for assessment Jobs. Mecatl used a scripted provider, not a live model.

- [Vulnerable](vulnerable.md): a confirmed synthetic cross-user document read.
- [Fixed](fixed.md): no issue observed for the known-owner checks.
- [Inconclusive](inconclusive.md): a missing object is not an authorization pass.
- [Redirect](redirect.md): an unapproved redirect is blocked.
- [Stop](stop.md): an active broker request is cancelled and the run stays stopped.

The matching JSON files contain run state and broker-captured evidence. Account aliases and documents are synthetic; operator and run credentials are excluded. Internal addresses and timestamps are retained.

These are evidence of a constrained workflow, not autonomous discovery or a production security guarantee. See [runtime/network checks](../../docs/sandbox-lab.md) and [observed results](../../docs/validation-results.md).
