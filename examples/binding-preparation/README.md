# Preparation and runtime qualification

Recorded on 2026-11-06. These examples prove that locally prepared synthetic facts can enter the existing trusted registration/approval path and still produce the expected owned-lab outcomes. Preparation is not authorization, a live account check or an assessment result.

The fictional [input template](input.template.json) uses a reserved example origin and synthetic values. Its [prepared output](template-prepared/plan.json) and [manifest](template-prepared/manifest.json) were generated without network access. Do not treat those addresses, proof strings or hashes as a runnable target configuration.

The fresh owned Kubernetes runs used six compiler-generated application plans:

| Application | Supplied baseline source | Actual assessment outcome |
|---|---|---|
| `vampi-vulnerable` | Four legitimate baseline GETs captured during owned setup | `confirmed` candidate |
| `vampi-fixed` | Four legitimate baseline GETs captured during owned setup | `no_issue_observed` for this test |
| `property-vulnerable` | Documents from the generated owned fixture seed | `confirmed` candidate |
| `property-filtered` | Documents from the generated owned fixture seed | `no_issue_observed` for the filtering rule |
| `property-ambiguous` | Documents from the generated owned fixture seed | `inconclusive` |
| `property-stop` | Documents from the generated owned fixture seed | Explicit Stop cancelled a held request |

The object campaign also repeated the existing Stop control, which was not compiler-generated. All seven runtime cases retained normal draft/digest approval gates, scripted Mecatl, the fixed curl adapter, Stop and unreviewed exports. The actual assessment still captured its own evidence; saved baseline documents did not replace runtime checks. No model calls or customer targets were used.

Builds:

- Native image `tsunami:binding-v1`: `sha256:7ec58b2ec6dd87853c4fc812515ce26aa9d9a9526d91aac8de711002df6f4a51`.
- Tools image `tsunami-tools:binding-v1`: `sha256:d3ab4c2bd07ff0b655bef23e368221e58cc66d68529e4ac3081fa0a1c89b16f3`.
- Operator/compiler binary SHA-256: `eb02258dfddb42472a9ce74ab62c1a29b4255098ae38cd5d84f3c6e2e362ae4b`.
- VAmPI remains the pinned upstream `f16052dce83f05847133ec98f01c5193a41de7d8` owned deployment. Its setup credentials/data were refreshed; unrelated application plans remained present.
- The dedicated single-node Kubernetes v1.34.0/Cilium 1.20.4/gVisor `release-20261026.0` lab was used. The normal context stayed `kind-kind`.
- Source is the binding-preparation working tree after baseline `985d57a9a9df3eb0876b3a01c2d56e4b81e2b913`; the baseline alone does not contain this milestone.

The property Stop case retained the actual tools image ID and CRI `runsc` / `io.containerd.runsc.v1` observation. The object Stop case observed the runtime, but Kubernetes had not populated its image-ID field at sampling; the raw record keeps that field empty. Fast runs have no independently retained CRI inspection. API resource absence is not substituted for process-exit proof.

Files:

- [qualification.json](qualification.json): links all six prepared packages to their actual approved engagement/run IDs. Each plan exactly matches the stored runtime plan after excluding the credential fingerprint that only the server assigns.
- [prepared](prepared): generated plan/manifest pairs. They contain selected public facts and protected-value hashes, with an empty credential fingerprint and pending-registration/approval status.
- [Object campaign](runtime/object/summary.json), [transcript](runtime/object/transcript.txt), [VAmPI evidence/report](runtime/object/vampi-vulnerable/report-draft.md).
- [Property campaign](runtime/property/summary.json), [transcript](runtime/property/transcript.txt), [property evidence/report](runtime/property/property-vulnerable/report-draft.md).
- [native-verification.txt](native-verification.txt) and [toolkit-verification.txt](toolkit-verification.txt): fresh race tests, vet, fourteen Python tests and expanded materialized export/provenance/schema checks.

Before copying, 103 campaign/prepared-package files were checked against current lab credential values, including decoded inline artifacts. No credential matches were found. Raw binding inputs remain in ignored private local directories; they are not copied here. The saved input hashes identify bytes but do not authenticate their source. Raw synthetic proof values intentionally remain in assessment evidence.

Native tests cover ambiguous/oversized JSON, duplicate keys, missing/mismatched owners, duplicate identities/records, numeric/path-injection IDs, invalid profile rules, weak/reused proof values, private output permissions, symlink/FIFO rejection and refusal to overwrite packages. They also verify that server credential binding and explicit scope approval remain necessary.

This is a controlled example, not general onboarding, a connected UI, autonomous discovery or customer readiness. See [the preparation guide](../../docs/binding-preparation.md) and [the remaining showcase milestone](../../docs/showcase-readiness.md).
