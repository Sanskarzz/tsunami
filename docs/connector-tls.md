# Connector transport and identity

This milestone secures the broker-to-connector link. It does not make every lab connection encrypted or connect us to a real customer's network.

## What changes

The private connector now listens on HTTPS only, with TLS 1.3 and required client certificates. The broker checks the customer-specific CA, service hostname, and exact connector URI SAN. The connector checks the CA and exact broker URI SAN. Common Names and HTTP identity headers are not used.

The identities use URI SANs under `spiffe://tsunami.lab/tenants/demo-a/`. This is a naming convention for this prototype, not an installed SPIFFE/SPIRE system.

- `broker`: may send signed request grants.
- `connectors/customer-a`: identifies the expected server.
- `emergency-operator`: may use the status/cutoff endpoints, with the separate emergency bearer credential.

A broker certificate cannot use the emergency endpoints, even with the emergency bearer key. An emergency certificate cannot execute grants. A certificate for another tenant, a different connector identity, an unknown CA, or the wrong server name is rejected. A valid certificate alone is insufficient to send a target request: the signed, scoped, single-use grant is still required.

This protects request grants and returned evidence while they cross this link. A compromised connector can still forge evidence before sending it. The worker-to-broker link and connector-to-fixture links remain synthetic HTTP. This is not ready for real customer credentials or evidence.

## Expiry and open connections

TLS normally checks certificate validity at handshake time. It does not automatically cancel a long response or re-authorize every request on an existing connection.

The connector therefore checks client role and certificate validity for each HTTP request. It bounds target execution by the earliest of the grant expiry, client/CA certificate expiry, server/CA certificate expiry, and incoming request cancellation. Native tests cover an already-open target response and a later request on the same TLS connection. A partial body after expiry is not released as usable evidence.

The broker uses fresh transports without upstream keepalive and does not use environment proxies or follow connector redirects. Connector connection attempts have a two-second dial timeout and a three-second TLS handshake timeout, within the existing operation deadline. A failed handshake never falls back to HTTP or direct target access.

The same trust assumptions remain: correct clocks, an honest CA/provisioner, trusted broker/connector processes, and a functioning runtime. These checks do not establish a production Stop bound under frozen processes or arbitrary partitions. Target-side work already accepted cannot be undone.

## Small lab PKI

I'd keep certificate provisioning outside the agent. The setup uses Go's standard certificate libraries and an offline CA in the ignored `.local/private-pki/` directory. It proposes a 30-day CA and 24-hour leaf certificates. These are lab defaults, not an automated rotation service.

The CA key and emergency client's TLS key remain on the operator's machine. Kubernetes receives only the CA certificate and the leaf key needed by each component: broker key in `tsunami`, connector key in `tsunami-customer-a`. Workers receive neither. Private files use mode `0600` inside a `0700` directory. The public record contains identities, fingerprints, and expiry dates only.

Ordinary setup preserves valid certificates. Expired or mismatched material fails setup/startup; it is not silently replaced. To renew leaves explicitly, between assessments:

```sh
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
export CONTEXT=kind-tsunami-lab RUNTIME_CLASS=gvisor
RENEW_PRIVATE_TLS=true make private-deploy CONTEXT=kind-tsunami-lab
```

This keeps the CA, replaces leaf material, updates the component Secrets, and restarts the connector/control plane. It preserves grant signing keys, replay state, and the emergency cutoff. Existing assessments can be interrupted; this is a maintenance operation, not Stop. There is no hot reload or automatic renewal controller.

Renewal does **not revoke** an old, still-valid certificate. An attacker with a stolen leaf can still authenticate until it expires, subject to its role and the grant/cutoff checks. Emergency cutoff blocks target execution but does not revoke TLS identity. Fast certificate revocation, safe CA rollover, production key storage, and unattended rotation remain future work. If the offline CA expires, setup refuses to replace it automatically. If installed trust exists but local CA files are missing or different, setup refuses to replace that trust too; restore the correct material or perform a deliberate rollover outside this helper.

## Check the deployed endpoint

```sh
make private-deploy CONTEXT=kind-tsunami-lab
make private-verify CONTEXT=kind-tsunami-lab
# Run after the outage check restores the connector.
make connector-tls-check CONTEXT=kind-tsunami-lab
```

The TLS check uses a loopback Kubernetes port-forward while still verifying the real connector DNS/URI identity. It checks broker access, operator status, role separation, missing certificates, untrusted CA, incorrect server name/audience, old TLS versions, and plaintext refusal. Successful authenticated checks bracket the negative checks. Refused connections and timeouts do not count as TLS rejection. It sends no target requests and does not latch emergency cutoff.

The existing emergency helper now uses the operator's certificate plus its bearer key over this verified TLS connection. It still needs Kubernetes port-forward and leaves cutoff persisted if invoked. The normal connector readiness probe checks only that the TCP listener is available; functional and TLS verification are separate tests.

See [recorded transport and workflow results](../examples/connector-tls/README.md). There are no model calls in these checks. Real customer-side tunnelling, full cross-node partition testing, other internal TLS links, external emergency network cutoff, and production isolation are still unfinished.
