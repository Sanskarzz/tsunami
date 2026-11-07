# Tsunami engagement draft

UI qualification: vampi-vulnerable

Engagement: `f1ddf3f4625d712fdf7c24644bd7ab25`; state: active. Synthetic targets only.

Selected procedures: 1; completed: 1; skipped: 0; pending: 0; running: 0; failed: 0; stopped: 0; incomplete evidence: 0.

Completion means the selected procedure captured its prerequisites and probe. It does not mean the application is secure.

| Procedure | Progress | Verdict |
|---|---|---|
| vampi-vulnerable/object-access/v2 | completed | confirmed |

## Finding candidates

Candidates come from broker-captured evidence on owned lab data. Explicit review records are separate from the observation.

### F-54ad48aa227e0a03af6928c3b49a2d4b

Cross-user book secret access

Status: needs_human_review; severity: unrated.

The authenticated peer received the owner&\#39;s approved proof data; account identities and owner access were independently checked.

Remediation: Authorize every object read against the authenticated principal and intended sharing rules on the server.

Evidence IDs: E-ad749f9fa1e0a6ae576fe156778371a241b70bb02f0306de8ff99ceaf484cf77, E-a83a076f813dd6f7b66cb9a2468305476d10143ecf50005b7dd9df5a6d0e67ea, E-56b77b2819305fb408f1157696835c9b16b536baf6f8e64fc4d3677c0b8d9421, E-f420e016480fd7703bd6446358c2753e353fd1a0599dd4650ad3e29ba513a6f1, E-cb493edd8cb027cfcf15bbd1bc55f390d7c7b92a898bd127ca4285ea42e6d13c, E-d4997874e26f5c67aade5c9e030573087ed46671b1cebb8270fc27b99bd88c2b, E-ffb3059b832ae292cff6ad3c3ae068e11d40bedf40a391beaab7b6ccea4b27d6

## Evidence index

| Evidence | Run | Artifact path |
|---|---|---|
| E-68c24ab731924f32f4dcdaf316f72455adba37fa6ec88de4a8cfb7c4cb48a01e | bfd6e2d0f3e3808606341cc9b633a3c9 | evidence/68c24ab731924f32f4dcdaf316f72455adba37fa6ec88de4a8cfb7c4cb48a01e.json |
| E-ad749f9fa1e0a6ae576fe156778371a241b70bb02f0306de8ff99ceaf484cf77 | bfd6e2d0f3e3808606341cc9b633a3c9 | evidence/ad749f9fa1e0a6ae576fe156778371a241b70bb02f0306de8ff99ceaf484cf77.json |
| E-a83a076f813dd6f7b66cb9a2468305476d10143ecf50005b7dd9df5a6d0e67ea | bfd6e2d0f3e3808606341cc9b633a3c9 | evidence/a83a076f813dd6f7b66cb9a2468305476d10143ecf50005b7dd9df5a6d0e67ea.json |
| E-56b77b2819305fb408f1157696835c9b16b536baf6f8e64fc4d3677c0b8d9421 | bfd6e2d0f3e3808606341cc9b633a3c9 | evidence/56b77b2819305fb408f1157696835c9b16b536baf6f8e64fc4d3677c0b8d9421.json |
| E-f420e016480fd7703bd6446358c2753e353fd1a0599dd4650ad3e29ba513a6f1 | bfd6e2d0f3e3808606341cc9b633a3c9 | evidence/f420e016480fd7703bd6446358c2753e353fd1a0599dd4650ad3e29ba513a6f1.json |
| E-cb493edd8cb027cfcf15bbd1bc55f390d7c7b92a898bd127ca4285ea42e6d13c | bfd6e2d0f3e3808606341cc9b633a3c9 | evidence/cb493edd8cb027cfcf15bbd1bc55f390d7c7b92a898bd127ca4285ea42e6d13c.json |
| E-d4997874e26f5c67aade5c9e030573087ed46671b1cebb8270fc27b99bd88c2b | bfd6e2d0f3e3808606341cc9b633a3c9 | evidence/d4997874e26f5c67aade5c9e030573087ed46671b1cebb8270fc27b99bd88c2b.json |
| E-ffb3059b832ae292cff6ad3c3ae068e11d40bedf40a391beaab7b6ccea4b27d6 | bfd6e2d0f3e3808606341cc9b633a3c9 | evidence/ffb3059b832ae292cff6ad3c3ae068e11d40bedf40a391beaab7b6ccea4b27d6.json |

This is a draft for human review. Export hashes describe bytes, not independent evidence authenticity.
