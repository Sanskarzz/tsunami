# Tsunami engagement draft

Owned lab authorization assessment

Engagement: `8496b6ec8702439e111eb101f150542c`; state: stopped. Synthetic targets only.

Selected procedures: 5; completed: 5; skipped: 0; pending: 0; running: 0; failed: 0; stopped: 0; incomplete evidence: 0.

Completion means the selected procedure captured its prerequisites and probe. It does not mean the application is secure.

| Procedure | Progress | Verdict |
|---|---|---|
| anonymous-access/v1 | completed | confirmed |
| missing-object/v1 | completed | inconclusive |
| opaque-ownership/v1 | completed | confirmed |
| role-boundary/v1 | completed | confirmed |
| shared-access/v1 | completed | no_issue_observed |

## Finding candidates

Candidates are observations from the synthetic validator. Impact, severity and CVSS need human review.

### F-1a9c70682f5eddd1108e816fc1a252e3

Member access to an administrator summary

Status: needs_human_review; severity: unrated.

The probe returned the exact protected fixture object despite the reviewed deny rule; legitimate baselines also succeeded.

Remediation: Enforce the administrator role on the server before returning the summary. A valid member session is insufficient.

Evidence IDs: E-1edb86671a6ab9a68889577b1cc28ce9301424dc41b258f88e4359fa4c82a223, E-436a5a819c1ee81e6928e19a23b2bab6d1ebb404fde3d34a8180d72f7a2cc71c, E-cb1dddee673fcef96c4454bdfa39f6ab4b17caeba68d3a418e83afb333391a47

### F-47ab41882bf6b143bd395df03e0b963e

Cross-user access to an opaque object ID

Status: needs_human_review; severity: unrated.

The probe returned the exact protected fixture object despite the reviewed deny rule; legitimate baselines also succeeded.

Remediation: Check object ownership against the authenticated account on every object read. Opaque IDs alone do not authorize access.

Evidence IDs: E-00a04fa5c9d1a1606f0dc65ac0f3be11a5c69b53644a2ee7813228ed866757a5, E-27c98c563dcc13a33ed9a4dd4154a76caa2377e8fd135e0e19efe4b57375c222, E-68f1c100d8488ebed9fa7719a656c5f0a20f4cf96a7cd2b432515bd8dd039236

### F-65dd6290df37d70e6df9d77e928b5df9

Unauthenticated access to a known private object

Status: needs_human_review; severity: unrated.

The probe returned the exact protected fixture object despite the reviewed deny rule; legitimate baselines also succeeded.

Remediation: Require authentication and object authorization before returning private data.

Evidence IDs: E-25197810b83ae7c307364826922557c3063817c2b882e2ae69a0472929c0561b, E-e8ff1ec746ee5801447677619f8505bcced4e0577699db50fae9ac5a7bf4ef45, E-4f467872c205d7541440ac8bdc64e75fca0fbc8d26b58c2d3e2de616edd2f395

## Evidence index

| Evidence | Run | Artifact path |
|---|---|---|
| E-da64e030765d80afa75b2bf03774b435f8a058359b0e247d7f415d2d901bd50c | 33b53bb6a11290df89d4a9ee3e88b3af | evidence/da64e030765d80afa75b2bf03774b435f8a058359b0e247d7f415d2d901bd50c.json |
| E-25197810b83ae7c307364826922557c3063817c2b882e2ae69a0472929c0561b | 33b53bb6a11290df89d4a9ee3e88b3af | evidence/25197810b83ae7c307364826922557c3063817c2b882e2ae69a0472929c0561b.json |
| E-e8ff1ec746ee5801447677619f8505bcced4e0577699db50fae9ac5a7bf4ef45 | 33b53bb6a11290df89d4a9ee3e88b3af | evidence/e8ff1ec746ee5801447677619f8505bcced4e0577699db50fae9ac5a7bf4ef45.json |
| E-4f467872c205d7541440ac8bdc64e75fca0fbc8d26b58c2d3e2de616edd2f395 | 33b53bb6a11290df89d4a9ee3e88b3af | evidence/4f467872c205d7541440ac8bdc64e75fca0fbc8d26b58c2d3e2de616edd2f395.json |
| E-c740120fbf09c29970b92e33f8646008af2c66b761c9d7c510f9bde10fa309ef | 728738d5cf8e43a4e7265433518c945d | evidence/c740120fbf09c29970b92e33f8646008af2c66b761c9d7c510f9bde10fa309ef.json |
| E-c9aaa62f6c9558157a02f4adbea9c36e22f82da0c344553edac4bf873382b065 | 728738d5cf8e43a4e7265433518c945d | evidence/c9aaa62f6c9558157a02f4adbea9c36e22f82da0c344553edac4bf873382b065.json |
| E-82b012f24d6b6774b0f17fc9003c38f96030ecd26f6355adaeb84a5a0de4e1ab | 728738d5cf8e43a4e7265433518c945d | evidence/82b012f24d6b6774b0f17fc9003c38f96030ecd26f6355adaeb84a5a0de4e1ab.json |
| E-e19ad72ea11aa55797b6757222cbcb528060b9f95c3a16b6da0ed90698340eb1 | 728738d5cf8e43a4e7265433518c945d | evidence/e19ad72ea11aa55797b6757222cbcb528060b9f95c3a16b6da0ed90698340eb1.json |
| E-e889c8b0461ae398b38bcaac9c142bda696eef5b27502b503190bed537279b37 | 45029f0e695c989134c0f74c88582283 | evidence/e889c8b0461ae398b38bcaac9c142bda696eef5b27502b503190bed537279b37.json |
| E-00a04fa5c9d1a1606f0dc65ac0f3be11a5c69b53644a2ee7813228ed866757a5 | 45029f0e695c989134c0f74c88582283 | evidence/00a04fa5c9d1a1606f0dc65ac0f3be11a5c69b53644a2ee7813228ed866757a5.json |
| E-27c98c563dcc13a33ed9a4dd4154a76caa2377e8fd135e0e19efe4b57375c222 | 45029f0e695c989134c0f74c88582283 | evidence/27c98c563dcc13a33ed9a4dd4154a76caa2377e8fd135e0e19efe4b57375c222.json |
| E-68f1c100d8488ebed9fa7719a656c5f0a20f4cf96a7cd2b432515bd8dd039236 | 45029f0e695c989134c0f74c88582283 | evidence/68f1c100d8488ebed9fa7719a656c5f0a20f4cf96a7cd2b432515bd8dd039236.json |
| E-5ad49b1169be0eb6e0bce23d1c4507c6c009b06194efab0a7c0c26ff40048531 | eb5f22afc9f6cf1e03c1b1d91687fb0e | evidence/5ad49b1169be0eb6e0bce23d1c4507c6c009b06194efab0a7c0c26ff40048531.json |
| E-1edb86671a6ab9a68889577b1cc28ce9301424dc41b258f88e4359fa4c82a223 | eb5f22afc9f6cf1e03c1b1d91687fb0e | evidence/1edb86671a6ab9a68889577b1cc28ce9301424dc41b258f88e4359fa4c82a223.json |
| E-436a5a819c1ee81e6928e19a23b2bab6d1ebb404fde3d34a8180d72f7a2cc71c | eb5f22afc9f6cf1e03c1b1d91687fb0e | evidence/436a5a819c1ee81e6928e19a23b2bab6d1ebb404fde3d34a8180d72f7a2cc71c.json |
| E-cb1dddee673fcef96c4454bdfa39f6ab4b17caeba68d3a418e83afb333391a47 | eb5f22afc9f6cf1e03c1b1d91687fb0e | evidence/cb1dddee673fcef96c4454bdfa39f6ab4b17caeba68d3a418e83afb333391a47.json |
| E-20a95a01c5c8f6100edd62623ecc9215524f776e86e0473f535616c10b3d2df6 | b47fbe96c21606dc273dd0eeff68a7f4 | evidence/20a95a01c5c8f6100edd62623ecc9215524f776e86e0473f535616c10b3d2df6.json |
| E-86368e4a6500a07ad5bb84620abf719f311bef7907ba9f3d8218ee705e3173e3 | b47fbe96c21606dc273dd0eeff68a7f4 | evidence/86368e4a6500a07ad5bb84620abf719f311bef7907ba9f3d8218ee705e3173e3.json |
| E-d30d0fbc2eeb66edcd0caed5f224c468704d08efb904fce66486b0cb7d09fedd | b47fbe96c21606dc273dd0eeff68a7f4 | evidence/d30d0fbc2eeb66edcd0caed5f224c468704d08efb904fce66486b0cb7d09fedd.json |
| E-8249a0f7267f7af6697d9a3a1c16aedf9c5ece8056639202a098c294b8f7bf72 | b47fbe96c21606dc273dd0eeff68a7f4 | evidence/8249a0f7267f7af6697d9a3a1c16aedf9c5ece8056639202a098c294b8f7bf72.json |

This is a draft for human review. Export hashes describe bytes, not independent evidence authenticity.
