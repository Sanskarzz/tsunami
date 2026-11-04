# Inference diagnostics milestone

Recorded on 2026-11-02. No external model calls were made for these checks. The configured free provider was preserved; the Kubernetes evaluations explicitly used scripted execution.

## Controlled failures

[Verification log](verification.log) and [call metadata](diagnostics.json) record 14 passing cases using the real Mecatl Responses adapter, gateway, and loopback fixtures. Normal and omitted-usage completions return a usable turn; all 12 failure cases return no reply/tool calls. Every case makes one fixture request and zero target requests. Failed calls retain reservations and uncertain usage. The omitted-usage success is also uncertain, despite the adapter's zero usage chunk.

The stall cases use short deadlines only in the trusted local harness. These are not real-provider reliability measurements, Kubernetes failure tests, or provider compute timings. Raw provider text and IDs are excluded; the tests include synthetic private-data canaries.

## Archived trials, unchanged

- [Initial analysis](initial-live-analysis.json): still 3/4 matched, one incomplete assessment and one raw missed planted case.
- [Feedback analysis](feedback-live-analysis.json): still 2/4 matched, two unclassified model failures and one raw missed planted case. There were no completed assessment misses in this sample because the two other runs failed operationally.
- [Archive checksums](archive-sha256.json): SHA-256 inventory of the original records. Analysis writes separate files; it does not change the old verdicts or infer a missing historical cause.

An operational failure still prevents an expected result. The extra categories do not raise the end-to-end score. See [original live records](../assessment-catalog/README.md).

## Regression checks

| Check | Result |
|---|---|
| Local Responses-fixture catalog, one repetition | 10/10 matched |
| Public Kubernetes catalog, Job `tsunami-catalog-25klx` | 20/20 matched |
| Private mTLS catalog, Job `tsunami-catalog-jzwlq` | 20/20 matched |
| Native race-enabled Go tests | Passed |
| `go vet ./...` | Passed |

The catalog scores above are deterministic integration checks. They do not measure live-model accuracy. Public/private suites each include four expected missing-object inconclusive results. Both have zero operational failures, assessment misses, and incomplete evidence.

Records: [offline summary](offline-catalog-evaluation.json), [public summary](kubernetes-public-evaluation.json), [private summary](kubernetes-private-evaluation.json), [public log](kubernetes-public-verification.log), [private log](kubernetes-private-verification.log), and [native test log](go-test-race.log).

[Example report](offline-example.md) and [run JSON](offline-example.json) show the new per-call table and metadata on an actual completed offline assessment. Its token counts are synthetic fixture values, not provider billing. Released tools count replies passed by the gateway, not proof of tool execution.

The Kubernetes regression image was `sha256:b46d89e07e88d7a76b31913d1fda0aeebb2f5f1970311f13413a21c8e9a7e2a4`. The later native archive-output link safeguard does not affect the deployed worker/gateway path. No timeouts, scope permissions, request/model budgets, provider selection, or retry settings were expanded. The current kubectl context remains `kind-kind`; checks used the separate `kind-tsunami-lab` context.

See [reproduction and diagnostic limits](../../docs/inference-diagnostics.md). Procedure review, a held-out dataset, repeated live evaluation, real customer networks, arbitrary shell/scanner tools, and production security qualification remain separate work.
