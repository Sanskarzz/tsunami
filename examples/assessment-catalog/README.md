# Authorization catalog results

Recorded on 2026-11-02. All applications, accounts, and data are synthetic. Procedure definitions are initial engineering work; security-engineer review is pending.

| Evaluation | Runs | Matched expected result | Missed planted cases | False positives | Inconclusive | Incomplete evidence |
|---|---:|---:|---:|---:|---:|---:|
| Local scripted, two repetitions | 20 | 20 | 0 | 0 | 4 | 0 |
| Local offline Responses fixture, two repetitions | 20 | 20 | 0 | 0 | 4 | 0 |
| Kubernetes public, two repetitions | 20 | 20 | 0 | 0 | 4 | 0 |
| Kubernetes private mTLS connector, two repetitions | 20 | 20 | 0 | 0 | 4 | 0 |
| Initial free-model trial | 4 | 3 | 1 | 0 | 1 | 1 |
| Follow-up after missing-evidence feedback | 4 | 2 | 1 | 0 | 2 | 2 |

The four intentional inconclusive cases in each full suite are the nonexistent-object procedure on both fixtures, repeated twice. They do not represent a passed authorization check. Each full suite also confirms six planted forbidden reads and records ten expected no-issue observations, including approved sharing.

## Deterministic records

- [Public summary](public/evaluation.json) and [private summary](private/evaluation.json) contain all 20 exact run IDs each. The matching directories include the first repetition's ten JSON/Markdown reports, with original IDs and timestamps. Full exports remain in local artifacts.
- [Opaque ownership finding](public/opaque-ownership-v1-vulnerable.md), [known-object 404 denial](public/opaque-ownership-v1-fixed.md), [approved share](public/shared-access-v1-vulnerable.md), [administrator boundary](public/role-boundary-v1-vulnerable.md), [anonymous finding](private/anonymous-access-v1-vulnerable.md), and [missing-object control](public/missing-object-v1-vulnerable.md) show the different evidence rules.
- [Local scripted summary](local-scripted/evaluation.json) and [offline adapter summary](local-responses-fixture/evaluation.json) retain repeated local results. The offline model emits deterministic tool calls and synthetic token usage; these numbers are not real inference measurements.
- [Job/image record](deterministic-record.json), per-directory verifier logs, and [race-test log](safety-tests.log) identify the source. Tests cover legacy-scope preservation, wrong account/path denial, no header on anonymous reads, procedure digest checks, paid-provider rejection, ambiguous evidence, durable/bounded feedback, Stop during feedback, and real Mecatl/Responses recovery from premature completion.
- [Legacy private regression](legacy-private-regression.log) passed the original five scenarios, including Stop, tenant denial, and connector outage. Those runs did not opt into a model.

## Actual free-model outcomes

The provider was `nvidia/nemotron-3.5-lightning:free` through the zero-price, no-fallback profile. Only opaque ownership and shared access were selected, with one repetition each. Fixture hostnames and access rules were visible; these are guided samples, not a blind benchmark.

The [initial trial](initial-live/evaluation.json) matched three cases. On the vulnerable opaque object, the model sent the forbidden cross-user read but finished without either legitimate baseline. The response contained the protected fixture object; the validator still returned inconclusive because its prerequisites were missing. The evaluator counts that incomplete assessment as a missed planted case. The [original report](initial-live/opaque-ownership-v1-vulnerable.md) is retained.

The implementation then added up to two trusted missing-evidence feedback responses without changing scope, budgets, deadlines, or validation. Native tests deliberately reproduce a probe-first/finish-early model and verify recovery through the actual adapter and Mecatl loop.

In the single [follow-up trial](feedback-live/evaluation.json), both shared cases used one feedback response, collected their legitimate baselines, and completed with the approved-sharing result. The [shared report](feedback-live/shared-access-v1-vulnerable.md) and JSON show that recovery. Both opaque cases failed during their first model turn: each reserved one upstream attempt, opened a stream, returned no usable completed turn, and sent no target request. They took roughly the configured 20-second model-call window, but the retained error category is generic `model_error`. The exact upstream failure cause is unconfirmed. Token usage stays uncertain, not recorded as verified zero usage.

No further trials were run to seek a passing result. These two samples do not establish that the prompt or feedback improved model accuracy. They show bounded recovery in two cases and unresolved provider/model reliability in others. Better error classification and repeated evaluation remain work to do.

[Live Job/image records](live-record.json) distinguish both versions. Sanitized account metadata is recorded [before](account-before.json), [after the initial trial](account-after-initial.json), [before the follow-up](account-before-followup.json), and [afterward](account-final.json). OpenRouter reported $0 usage/daily usage and 28 free requests used for these two trials. That is a vendor snapshot, not an independent billing audit.

There are no provider, operator, run, signing, emergency, or TLS private credentials in these samples. See [procedure design and limits](../../docs/assessment-catalog.md) and [the toolkit mapping](../../docs/toolkit-mapping.md).
