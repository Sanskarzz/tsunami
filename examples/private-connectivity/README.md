# Private connector lab results

Recorded on 2026-11-01 in the dedicated `kind-tsunami-lab` cluster. The upstream agent was Mecatl's scripted adapter; no model requests were made. The applications, accounts, and evidence are synthetic.

- [Vulnerable report](vulnerable.md): confirmed cross-user document access through the connector.
- [Fixed report](fixed.md): no issue observed for the tested known-owner reads.
- [Missing object](inconclusive.md): stays inconclusive.
- [Redirect](redirect.md): an unapproved hop was denied; the private tripwire received no assessment requests.
- [Stopped run](stop.md): observed broker drain of 29 ms; the target observed cancellation.
- [Verification log](verification.log) and [record](record.json): source Job, image ID, exact run IDs, and counters. Matching JSON evidence files retain the original records.
- [Outage record](outage.json): with the connector scaled to zero, the private target request count stayed unchanged. Customer B's attempt to create a private run was also denied. The connector was restored afterward.
- [Runner network record](runner-network/network.json) and [runner log](runner-network/runner.log): startup/settled/repeat denials to tested private apps, private connector, unrestricted-ingress private tripwire, platform API, and DNS paths. The broker remained reachable. Trusted positive controls are in the same directory.
- [Connector network record](connector-network/record.json) and [connector log](connector-network/connector.log): a pod with the actual connector network identity in Customer A could reach its own app's health endpoint, but the tested Customer B and Kubernetes API paths timed out. Before/after controls could reach those destinations. This probe mounted no credentials.
- [Native race-test log](native-tests.log): grant identity checks, durable replay rejection, expiry of an open response body without a control-plane callback, persisted emergency cutoff, run/tenant/global cancellation, outage, and redirect denial.
- [Public regression log](public-regression.log): the original public five-case suite still passed with the model configuration present; it remained scripted.

The first runner network probe overlapped the intentional connector outage. It saw `connection refused`, which the harness correctly treated as unproven. That [failed record](network-during-outage/network.json) and [log](network-during-outage/runner.log) are retained. The qualified result above came from a separate run with the connector healthy. A dead endpoint never counts as evidence of policy denial.

The Kubernetes suite exercises real Jobs, a connector process in a separate namespace, persistent state, and controlled outages. The expiry/replay/emergency tests are native HTTP tests; they do not demonstrate every cross-node partition or a production stop objective. The connector's emergency cutoff was not latched on the main Kubernetes deployment; it remains available for lab use.

Internal links are HTTP. Signatures authenticate grants, but do not encrypt traffic or authenticate the response. This setup is for synthetic data only. No signing, emergency, model, operator, or run credential is included in these files. See [design and reproduction](../../docs/private-connectivity.md).
