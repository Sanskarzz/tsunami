# Tsunami engagement draft

Web/API authorization assessment

Engagement: `17db847636ff157eac3e3d7a41692349`; state: stopped. Synthetic targets only.

Selected procedures: 1; completed: 1; skipped: 0; pending: 0; running: 0; failed: 0; stopped: 0; incomplete evidence: 0.

Completion means the selected procedure captured its prerequisites and probe. It does not mean the application is secure.

| Procedure | Progress | Verdict |
|---|---|---|
| vampi-vulnerable/object-access/v2 | completed | confirmed |

## Finding candidates

Candidates come from broker-captured evidence on owned lab data. Explicit review records are separate from the observation.

### F-57409751904ff21c2075a32d669cfb40

Cross-user book secret access

Status: needs_human_review; severity: unrated.

The authenticated peer received the owner&\#39;s approved proof data; account identities and owner access were independently checked.

Remediation: Authorize every object read against the authenticated principal and intended sharing rules on the server.

Evidence IDs: E-0db607285514a68504c2bf392081542301e96de1af945a76162530e8a900c383, E-6c3cad23a35ff5371bc870514de88ebd0ca9a1d6edb4347db6975f141d056e64, E-0f76aa61afe381a1eb27bdb74103e0c9248ce3e0a980a30600ce16b114ae28e2, E-46681b01dc368ef3f424a9cfa66c6dce0e99b9900603d7523f569d9917d977ea, E-22876e068c6102782c218dade655efd29a95c99ef0b470241632bd4696e4bf59, E-96ba35a562c12d2c2ed274cc12f4a4d784638c8035d26fd702eda92b2b5bb4ac, E-47ecc2ce9abb894c2f56ce3d3dd8c2491f5634834ea3279a347dca2c2119a316

## Evidence index

| Evidence | Run | Artifact path |
|---|---|---|
| E-e53db72ccc7920968c9190fcdb43e94348eb37ca8bdc4a85a8321db4a4b69801 | 0a6bc7741a5b0e26b286925585ed7164 | evidence/e53db72ccc7920968c9190fcdb43e94348eb37ca8bdc4a85a8321db4a4b69801.json |
| E-0db607285514a68504c2bf392081542301e96de1af945a76162530e8a900c383 | 0a6bc7741a5b0e26b286925585ed7164 | evidence/0db607285514a68504c2bf392081542301e96de1af945a76162530e8a900c383.json |
| E-6c3cad23a35ff5371bc870514de88ebd0ca9a1d6edb4347db6975f141d056e64 | 0a6bc7741a5b0e26b286925585ed7164 | evidence/6c3cad23a35ff5371bc870514de88ebd0ca9a1d6edb4347db6975f141d056e64.json |
| E-0f76aa61afe381a1eb27bdb74103e0c9248ce3e0a980a30600ce16b114ae28e2 | 0a6bc7741a5b0e26b286925585ed7164 | evidence/0f76aa61afe381a1eb27bdb74103e0c9248ce3e0a980a30600ce16b114ae28e2.json |
| E-46681b01dc368ef3f424a9cfa66c6dce0e99b9900603d7523f569d9917d977ea | 0a6bc7741a5b0e26b286925585ed7164 | evidence/46681b01dc368ef3f424a9cfa66c6dce0e99b9900603d7523f569d9917d977ea.json |
| E-22876e068c6102782c218dade655efd29a95c99ef0b470241632bd4696e4bf59 | 0a6bc7741a5b0e26b286925585ed7164 | evidence/22876e068c6102782c218dade655efd29a95c99ef0b470241632bd4696e4bf59.json |
| E-96ba35a562c12d2c2ed274cc12f4a4d784638c8035d26fd702eda92b2b5bb4ac | 0a6bc7741a5b0e26b286925585ed7164 | evidence/96ba35a562c12d2c2ed274cc12f4a4d784638c8035d26fd702eda92b2b5bb4ac.json |
| E-47ecc2ce9abb894c2f56ce3d3dd8c2491f5634834ea3279a347dca2c2119a316 | 0a6bc7741a5b0e26b286925585ed7164 | evidence/47ecc2ce9abb894c2f56ce3d3dd8c2491f5634834ea3279a347dca2c2119a316.json |

This is a draft for human review. Export hashes describe bytes, not independent evidence authenticity.
