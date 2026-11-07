# Tsunami engagement draft

Web/API authorization assessment

Engagement: `734f2944515f58bf2f978c563b837897`; state: stopped. Synthetic targets only.

Selected procedures: 1; completed: 1; skipped: 0; pending: 0; running: 0; failed: 0; stopped: 0; incomplete evidence: 0.

Completion means the selected procedure captured its prerequisites and probe. It does not mean the application is secure.

| Procedure | Progress | Verdict |
|---|---|---|
| vampi-vulnerable/object-access/v2 | completed | confirmed |

## Finding candidates

Candidates come from broker-captured evidence on owned lab data. Explicit review records are separate from the observation.

### F-8fd869d00f5196b8483fb732905872e0

Cross-user book secret access

Status: needs_human_review; severity: unrated.

The authenticated peer received the owner&\#39;s approved proof data; account identities and owner access were independently checked.

Remediation: Authorize every object read against the authenticated principal and intended sharing rules on the server.

Evidence IDs: E-26d2081b090047b79ccd1c6388698abfc83de866bbb7f169df57be0a55900f93, E-92c9c85d8e47c8d75431b16e1b3adc5ab9dd0721476eba9fe1d0cd3fe6a7c765, E-ec0564ad5f80d03913b040178cac189c832dc1f57630383370f266c4574bc250, E-4cd9e00b8bc32284b0469b3f505c22b07e40a7ac4b78e33e8a3e7a7431e902be, E-6d85a2becacfb82d4b0e443ce5bf7c98fe2901faf410444ad80ad0b53443f0b0, E-7f9fc59186d4ebe1dc7c743c7dff94e9ac032234505c453cfd2ab95784709d0a, E-42e681a8fc7f4737316f99eef0ec5c59fed13b786b606dc0a38d825fa0138025

## Evidence index

| Evidence | Run | Artifact path |
|---|---|---|
| E-65a1343500d3352d44e1c68c744923e83c9c309f3c58566e8be2f253b1454558 | bc0651ea992c1f129ef2efb1424b919a | evidence/65a1343500d3352d44e1c68c744923e83c9c309f3c58566e8be2f253b1454558.json |
| E-26d2081b090047b79ccd1c6388698abfc83de866bbb7f169df57be0a55900f93 | bc0651ea992c1f129ef2efb1424b919a | evidence/26d2081b090047b79ccd1c6388698abfc83de866bbb7f169df57be0a55900f93.json |
| E-92c9c85d8e47c8d75431b16e1b3adc5ab9dd0721476eba9fe1d0cd3fe6a7c765 | bc0651ea992c1f129ef2efb1424b919a | evidence/92c9c85d8e47c8d75431b16e1b3adc5ab9dd0721476eba9fe1d0cd3fe6a7c765.json |
| E-ec0564ad5f80d03913b040178cac189c832dc1f57630383370f266c4574bc250 | bc0651ea992c1f129ef2efb1424b919a | evidence/ec0564ad5f80d03913b040178cac189c832dc1f57630383370f266c4574bc250.json |
| E-4cd9e00b8bc32284b0469b3f505c22b07e40a7ac4b78e33e8a3e7a7431e902be | bc0651ea992c1f129ef2efb1424b919a | evidence/4cd9e00b8bc32284b0469b3f505c22b07e40a7ac4b78e33e8a3e7a7431e902be.json |
| E-6d85a2becacfb82d4b0e443ce5bf7c98fe2901faf410444ad80ad0b53443f0b0 | bc0651ea992c1f129ef2efb1424b919a | evidence/6d85a2becacfb82d4b0e443ce5bf7c98fe2901faf410444ad80ad0b53443f0b0.json |
| E-7f9fc59186d4ebe1dc7c743c7dff94e9ac032234505c453cfd2ab95784709d0a | bc0651ea992c1f129ef2efb1424b919a | evidence/7f9fc59186d4ebe1dc7c743c7dff94e9ac032234505c453cfd2ab95784709d0a.json |
| E-42e681a8fc7f4737316f99eef0ec5c59fed13b786b606dc0a38d825fa0138025 | bc0651ea992c1f129ef2efb1424b919a | evidence/42e681a8fc7f4737316f99eef0ec5c59fed13b786b606dc0a38d825fa0138025.json |

This is a draft for human review. Export hashes describe bytes, not independent evidence authenticity.
