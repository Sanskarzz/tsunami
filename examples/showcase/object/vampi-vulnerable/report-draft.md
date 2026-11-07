# Tsunami engagement draft

Web/API authorization assessment

Engagement: `f27939d9985805c7afee5352e6ba4e51`; state: stopped. Synthetic targets only.

Selected procedures: 1; completed: 1; skipped: 0; pending: 0; running: 0; failed: 0; stopped: 0; incomplete evidence: 0.

Completion means the selected procedure captured its prerequisites and probe. It does not mean the application is secure.

| Procedure | Progress | Verdict |
|---|---|---|
| vampi-vulnerable/object-access/v2 | completed | confirmed |

## Finding candidates

Candidates come from broker-captured evidence on owned lab data. Explicit review records are separate from the observation.

### F-d674f014ce09177716e6a8cc87be3906

Cross-user book secret access

Status: needs_human_review; severity: unrated.

The authenticated peer received the owner&\#39;s approved proof data; account identities and owner access were independently checked.

Remediation: Authorize every object read against the authenticated principal and intended sharing rules on the server.

Evidence IDs: E-c9a2bfbd5b722ce9f7e0519f29873779908f5d7599f2a00fe6efeb084fa9e445, E-83a7be951ef0358c1f74295ea024d970ff3e678a79f1a29e41fc9b9778c952a1, E-dbe238b2dd53ab8a02c22b5065e04a367f067b39e81815c2baa175fca80efaa0, E-fd691a567fff97cad0d00aa3f4647fdd2dffb30e32a23587af7f68ca29a6ea6b, E-9e63e17ef92bd4e7838151258eba35af8f9d60d6045c13c48484a409ebdad3d8, E-df3cda5311df26c13b93d59302dd2cd39012af84f0be26221d679b83eec8e60e, E-4c78abcda20578ffc618122e02cb75f8d4f66434dcc93a78a599bc2d3a7ea373

## Evidence index

| Evidence | Run | Artifact path |
|---|---|---|
| E-981b067f260ebf9103338fdeec6b7001e200f78bbc7489e5fcfae595bdca3529 | fc479a57ea17510c976e1bbd3799d663 | evidence/981b067f260ebf9103338fdeec6b7001e200f78bbc7489e5fcfae595bdca3529.json |
| E-c9a2bfbd5b722ce9f7e0519f29873779908f5d7599f2a00fe6efeb084fa9e445 | fc479a57ea17510c976e1bbd3799d663 | evidence/c9a2bfbd5b722ce9f7e0519f29873779908f5d7599f2a00fe6efeb084fa9e445.json |
| E-83a7be951ef0358c1f74295ea024d970ff3e678a79f1a29e41fc9b9778c952a1 | fc479a57ea17510c976e1bbd3799d663 | evidence/83a7be951ef0358c1f74295ea024d970ff3e678a79f1a29e41fc9b9778c952a1.json |
| E-dbe238b2dd53ab8a02c22b5065e04a367f067b39e81815c2baa175fca80efaa0 | fc479a57ea17510c976e1bbd3799d663 | evidence/dbe238b2dd53ab8a02c22b5065e04a367f067b39e81815c2baa175fca80efaa0.json |
| E-fd691a567fff97cad0d00aa3f4647fdd2dffb30e32a23587af7f68ca29a6ea6b | fc479a57ea17510c976e1bbd3799d663 | evidence/fd691a567fff97cad0d00aa3f4647fdd2dffb30e32a23587af7f68ca29a6ea6b.json |
| E-9e63e17ef92bd4e7838151258eba35af8f9d60d6045c13c48484a409ebdad3d8 | fc479a57ea17510c976e1bbd3799d663 | evidence/9e63e17ef92bd4e7838151258eba35af8f9d60d6045c13c48484a409ebdad3d8.json |
| E-df3cda5311df26c13b93d59302dd2cd39012af84f0be26221d679b83eec8e60e | fc479a57ea17510c976e1bbd3799d663 | evidence/df3cda5311df26c13b93d59302dd2cd39012af84f0be26221d679b83eec8e60e.json |
| E-4c78abcda20578ffc618122e02cb75f8d4f66434dcc93a78a599bc2d3a7ea373 | fc479a57ea17510c976e1bbd3799d663 | evidence/4c78abcda20578ffc618122e02cb75f8d4f66434dcc93a78a599bc2d3a7ea373.json |

This is a draft for human review. Export hashes describe bytes, not independent evidence authenticity.
