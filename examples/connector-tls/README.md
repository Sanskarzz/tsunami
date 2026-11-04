# Connector TLS results

Recorded on 2026-11-02 local time in `kind-tsunami-lab`; JSON timestamps use UTC. All targets and account data were synthetic. Mecatl workers used scripted execution. No model requests were made.

- [Transport record after renewal](transport.json) and [before renewal](transport-before-renewal.json): 12 checks each. Valid broker and operator access bracket missing-certificate, trust/hostname/audience, TLS-version, plaintext, and role-denial checks. These use a loopback port-forward but verify the actual service DNS/URI identities. Neither check sends target requests or changes cutoff.
- [Workflow record](record.json): exact source Jobs, image IDs, and Stop samples. All five private scenarios passed before and after leaf renewal, alongside tenant denial and an offline connector with no target traffic.
- [Confirmed private finding](after-renewal/vulnerable.md), [fixed case](after-renewal/fixed.md), and [stopped run](after-renewal/stop.md): captured evidence after renewal. Missing-object and redirect cases are in the same directory, with original JSON records and [verifier log](after-renewal/verification.log).
- [Before-renewal verifier log](before-renewal/verification.log) and [outage record](after-renewal/outage.json): Stop drained at the broker in 28 ms before renewal and 27 ms afterward; each target separately observed cancellation. These are individual samples, not a benchmark or service guarantee.
- [Public certificate metadata before](pki-before-renewal.json) and [after](pki-after-renewal.json), plus [renewal checks](renewal.json): all leaf fingerprints changed, the CA stayed the same, grant-signing/emergency credentials stayed the same, and both persistent volumes were preserved. An old, still-valid certificate was not revoked.
- [CA continuity checks](ca-continuity.json): setup refused missing/different offline CA files while leaving the installed broker TLS Secret unchanged. The checks used temporary local directories; actual CA files were not removed.
- [Native race-test log](native-tests.log): unknown/missing/wrong-tenant/expired client certificates; wrong server identity/name/CA; broker/operator separation; open-body expiry; certificate expiry on the same established TLS connection; explicit renewal; replay/cutoff/Stop regressions; and refusal to count a dead endpoint as TLS rejection.
- [Public regression log](public-regression.log): the original public workflow still passed without model opt-in.

An initial outage check waited for a retained diagnostic pod with the same broad connector-role label. Its five assessment cases passed, but the outage driver failed before creating an outage run. The selector was narrowed to the main connector instance; the complete successful checks above came afterward. The preliminary records remain under ignored `artifacts/private/tsunami-private-verify-kjbdc/`, and are not counted as a complete pass.

The main connector remains available and cutoff is unlatched. The deployed application image is recorded separately from the local read-only TLS diagnostic. Public records contain no CA/leaf private keys, grant-signing keys, emergency tokens, model keys, or run/operator credentials.

This secures one link: broker to connector. Worker/broker and connector/fixture links remain HTTP with synthetic data. Certificate renewal is manual, not revocation. There is no production stop or partition guarantee. See [design, commands, and remaining trust assumptions](../../docs/connector-tls.md).
