# Configured application qualification

Recorded on 2026-11-03 local time (machine records use UTC on 2026-11-02). This campaign uses an existing application, VAmPI, in our own Kubernetes lab with synthetic accounts/data. Mecatl uses its scripted provider. No external model calls or toolkit scripts were executed.

The v2 campaign Job was `tsunami-application-j22b5`.

| Application mode | Trusted result | Approved target requests | Candidates | Rated test exports |
|---|---|---|---|---|
| Vulnerable object-access implementation | `confirmed` | 7 | 1 | 1 |
| Fixed object-access implementation | `no_issue_observed` | 7 | 0 | 0 |

Each also records a blocked out-of-scope attempt, giving eight capture artifacts. The seven approved reads verify both identities, both own objects, the cross-user probe, owner continuity, and the peer session afterward. The stronger profile is `http-object-access/v2`; missing/deleted/expired prerequisites cannot be replaced by an earlier successful capture.

The rated export is explicitly `automated_lab_test`, produced to exercise the review/export contract. Its example CVSS 3.1 base vector calculates to 6.5/Medium. This is not human signoff, a customer impact rating, or model-generated severity. The candidate fingerprint and review revision are linked in the bundle. Raw observations remain separate from review dispositions.

## Records

- [Campaign summary](summary.json) and [verification log](verification.log).
- Vulnerable: [bundle](vampi-vulnerable/bundle.json), [draft report](vampi-vulnerable/report-draft.md), [CSV](vampi-vulnerable/findings.csv), and [schema/file validation](vampi-vulnerable/schema-validation.json). The bundle references the materialized finding and seven evidence metadata records in that directory.
- Fixed: [bundle](vampi-fixed/bundle.json), [draft report](vampi-fixed/report-draft.md), and [schema/file validation](vampi-fixed/schema-validation.json).
- [Vulnerable partial mapping](vampi-vulnerable/mapping.json) and [coverage projection](vampi-vulnerable/checklist-coverage.json); [fixed coverage](vampi-fixed/checklist-coverage.json). Ten API checklist IDs were imported. API1 is sampled by one procedure; fully completed standard items remain zero. Runtime scope was unchanged.
- [Go race tests](go-test-race.log), [targeted tests](targeted-tests.log), and [Python tests](python-tests.log). `go vet ./...` passed with an empty diagnostic log.
- Legacy catalog [public summary](catalog-public-evaluation.json) / [log](catalog-public-verification.log), and [private summary](catalog-private-evaluation.json) / [log](catalog-private-verification.log): both 20/20 matched, with expected inconclusive controls retained.

The local checker validated the actual engagement, finding and evidence schemas, review/rating consistency, membership, hashes, generated paths, timestamps and indexes. It checked all eight materialized artifact files per variant, plus the vulnerable finding file and seven reviewed evidence metadata files. Schema hashes and installed validator version are in the validation records. Original toolkit schemas/content were not copied into this repository.

## Network and source

[Network record](network/network.json), [runner log](network/runner.log), and trusted controls [before](network/positive-before.log) / [after](network/positive-after.log) qualify selected paths. gVisor runner Job `tsunami-network-runner-qhn8f` could reach the broker but timed out on both application Services and an ingress-unrestricted tripwire in the application namespace, plus existing API/DNS/model paths. Matching trusted probes connected. This avoids crediting the application ingress policy as runner egress enforcement. IPv6, metadata, alternate interfaces, escape resistance and production partitions are not established.

VAmPI source commit: `f16052dce83f05847133ec98f01c5193a41de7d8`; [upstream](https://github.com/erev0s/VAmPI) is MIT licensed. [Source/build metadata](source.json) records application image `sha256:8ffa773d316c0e2095e428118c856662815a4aae1e497d1385d20d7d3eb3213a`. Upstream source remains ignored/local. The packaging disables debug/reloader and adds a compatibility dependency pin; it is still deliberately vulnerable lab software.

Control/worker/diagnostic image: `sha256:dd9e373f202379acff22c5fee49c062f8e626ec57fdd0f3f618881ae9b3003ad`. Public catalog Job `tsunami-catalog-kftln` and private Job `tsunami-catalog-8fvgd` used that current build. The configured free provider and default `kind-kind` kubectl context were preserved; all deployment/testing here used `kind-tsunami-lab`.

A later historical-report guard was tested natively and deployed as `sha256:f502357874f61f5ec013332b7a0fcd21e06f934336497746b5b47416fbf89713`. The restart encountered an expired lab connector certificate and refused startup. Explicit leaf renewal under the same CA restored it; test accounts were reseeded separately. [Maintenance log](maintenance.log), [application smoke summary](maintenance-smoke-summary.json), [smoke log](maintenance-smoke.log), and [transport checks](maintenance-tls.log) retain that follow-up. Application Job `tsunami-application-p6rlp` again observed the expected two outcomes with seven requests each. These are manual maintenance observations, not automated rotation or an availability guarantee. Original qualification records above were not replaced.

Earlier local-only v1 campaigns, `tsunami-application-w22gx` and `tsunami-application-qxmf8`, used six reads and also observed the expected app behaviors. Their ignored records are retained. V2 adds the peer-session recheck after identifying the expiry ambiguity; old plans are refused rather than silently reinterpreted. Those earlier records do not establish the stronger v2 check. No free-model trial was repeated or replaced.

## Limits and reproduction

[Application config template](application-config.template.json) shows the shape without real credentials or private proof values. Its placeholders deliberately fail validation. Supply reviewed owned-lab facts and valid synthetic credentials through a protected startup file, not through an agent or a public API.

See [reproduction and design limits](../../docs/application-adapters.md). Current support is a bounded read-only HTTP profile with configured inventory/rules and synthetic data. Independent security review, broader authentication/response formats, real customer data handling, configurable private connectors, shell/scanners, VM/device/MCP executors, specialist coordination and scale qualification remain required work. This is not autonomous discovery, a compliance assessment or a customer-ready product.
