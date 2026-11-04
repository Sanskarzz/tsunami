# Recorded synthetic examples

These JSON captures and Markdown reports came from actual Mecatl worker Jobs in the local `tsunami` namespace on 2026-11-01. The model provider is scripted; these are workflow and enforcement examples, not autonomous discovery results.

- [Vulnerable](vulnerable.md): both ownership baselines pass, then Bob receives Alice's private fixture document.
- [Fixed](fixed.md): the cross-user read is forbidden.
- [Inconclusive](inconclusive.md): a missing object does not prove authorization works.
- [Redirect](redirect.md): the broker rejects an unapproved destination.
- [Stop](stop.md): an active broker request is cancelled and the run remains stopped.

The data and account aliases are synthetic. Operator and run credentials are excluded. Internal lab addresses and timestamps are retained so the examples can be inspected. The JSON files hold the corresponding run state and captures.

These samples do not prove hostile-container isolation or a production stop guarantee. The independent network check failed, and gVisor is not installed. See [the validation record](../../docs/validation-results.md).
