# Tsunami engagement draft

Owned application access-control test

Engagement: `88bbc4ec7190f5203c124724c3443514`; state: stopped. Synthetic targets only.

Selected procedures: 1; completed: 1; skipped: 0; pending: 0; running: 0; failed: 0; stopped: 0; incomplete evidence: 0.

Completion means the selected procedure captured its prerequisites and probe. It does not mean the application is secure.

| Procedure | Progress | Verdict |
|---|---|---|
| vampi-vulnerable/object-access/v2 | completed | confirmed |

## Finding candidates

Candidates come from broker-captured evidence on owned lab data. Explicit review records are separate from the observation.

### F-335c0d397cc2410c480a36a69399c44f

Cross-user book secret access

Status: reviewed_confirmed; severity: Medium (see review provenance below).

The authenticated peer received the owner&\#39;s approved proof data; account identities and owner access were independently checked.

Remediation: Authorize every object read against the authenticated principal and intended sharing rules on the server.

Evidence IDs: E-6288d6355b7c2bf46e8ed3fc28797d9f57e489900af78a2faa0ff710320fb618, E-c5b02f7521b8d6b8a33ba1dc72f62ba7af88b7bf6c3327ec576c8221d9b18bc1, E-7cb5b932b4dcc61bbffa8d00fa4924003bfe3cadf716adde300a3ac1721b50d1, E-95fb6e3752c5b563c994d918604e020448df74ef1018640ba6f69b7895eb57f8, E-5833187eac3360eed5d9f25066f17506d9e8b15e9702c317b93384124e18e4a0, E-f9e6a5152683c9759871520e58edff0ff221cd140cc569c04925d62da796e05e, E-ce7630a1e37fbacfe0d1cd143171bd60cbeb636a2bd459a3bfb7a779c50b6192

## Reviewed findings

### F-335c0d397cc2410c480a36a69399c44f: Cross-user book secret access

Severity: Medium; CVSS 3.1 base score: 6.5; vector: `CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N`.

Review purpose: owned-lab, automated_lab_test. An automated lab review is a test, not human signoff.

Impact supplied by reviewer: An authenticated synthetic peer retrieved another synthetic test account&\#39;s seeded private object proof.

## Evidence index

| Evidence | Run | Artifact path |
|---|---|---|
| E-038dccc95459a4ea0643ce11eff043a8c5030917d39d53edd117caef89e46bf3 | a9b4685d4b85ed1f9cf82fd17c856732 | evidence/038dccc95459a4ea0643ce11eff043a8c5030917d39d53edd117caef89e46bf3.json |
| E-6288d6355b7c2bf46e8ed3fc28797d9f57e489900af78a2faa0ff710320fb618 | a9b4685d4b85ed1f9cf82fd17c856732 | evidence/6288d6355b7c2bf46e8ed3fc28797d9f57e489900af78a2faa0ff710320fb618.json |
| E-c5b02f7521b8d6b8a33ba1dc72f62ba7af88b7bf6c3327ec576c8221d9b18bc1 | a9b4685d4b85ed1f9cf82fd17c856732 | evidence/c5b02f7521b8d6b8a33ba1dc72f62ba7af88b7bf6c3327ec576c8221d9b18bc1.json |
| E-7cb5b932b4dcc61bbffa8d00fa4924003bfe3cadf716adde300a3ac1721b50d1 | a9b4685d4b85ed1f9cf82fd17c856732 | evidence/7cb5b932b4dcc61bbffa8d00fa4924003bfe3cadf716adde300a3ac1721b50d1.json |
| E-95fb6e3752c5b563c994d918604e020448df74ef1018640ba6f69b7895eb57f8 | a9b4685d4b85ed1f9cf82fd17c856732 | evidence/95fb6e3752c5b563c994d918604e020448df74ef1018640ba6f69b7895eb57f8.json |
| E-5833187eac3360eed5d9f25066f17506d9e8b15e9702c317b93384124e18e4a0 | a9b4685d4b85ed1f9cf82fd17c856732 | evidence/5833187eac3360eed5d9f25066f17506d9e8b15e9702c317b93384124e18e4a0.json |
| E-f9e6a5152683c9759871520e58edff0ff221cd140cc569c04925d62da796e05e | a9b4685d4b85ed1f9cf82fd17c856732 | evidence/f9e6a5152683c9759871520e58edff0ff221cd140cc569c04925d62da796e05e.json |
| E-ce7630a1e37fbacfe0d1cd143171bd60cbeb636a2bd459a3bfb7a779c50b6192 | a9b4685d4b85ed1f9cf82fd17c856732 | evidence/ce7630a1e37fbacfe0d1cd143171bd60cbeb636a2bd459a3bfb7a779c50b6192.json |

This is a draft for human review. Export hashes describe bytes, not independent evidence authenticity.
