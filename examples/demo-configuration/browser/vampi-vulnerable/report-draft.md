# Tsunami engagement draft

UI qualification: vampi-vulnerable

Engagement: `bc1250e37e3407474abf9953c252b766`; state: active. Synthetic targets only.

Selected procedures: 1; completed: 1; skipped: 0; pending: 0; running: 0; failed: 0; stopped: 0; incomplete evidence: 0.

Completion means the selected procedure captured its prerequisites and probe. It does not mean the application is secure.

| Procedure | Progress | Verdict |
|---|---|---|
| vampi-vulnerable/object-access/v2 | completed | confirmed |

## Finding candidates

Candidates come from broker-captured evidence on owned lab data. Explicit review records are separate from the observation.

### F-cba0ec7c7fefc169a53ca5c46b0b4ff7

Cross-user book secret access

Status: needs_human_review; severity: unrated.

The authenticated peer received the owner&\#39;s approved proof data; account identities and owner access were independently checked.

Remediation: Authorize every object read against the authenticated principal and intended sharing rules on the server.

Evidence IDs: E-627a06828903700a15db969fb4d93e4f5454ab9bb289be6cc49e6eed572a0534, E-ddbb3756b5e4ba9636a3202a4c5f8de3b18242bfa81508a88206a43f8080db7e, E-09a8ab6f1c208d2254eb557e13c2bf375141fd291888ebf0d1f8d674952be6b8, E-67a0bdff0153f8ba4862209481947f75828aa847dbabdaa27db44b91d88f6f78, E-37ce66ca35a1b364af027067ba3bf202fe016bbba8ab012c33576136f598e40c, E-7d6bbbfa540d2f809e8dc8404613bf79e350c379aefc0a4af69db137ee7971c9, E-d8f0fa098b9c9622d4c2c66d8b2165c9d7aa6b777d89a774c6af0c252df155d9

## Evidence index

| Evidence | Run | Artifact path |
|---|---|---|
| E-d11d3b189b72ce851ea17b468189762cff738a2f942918584d19393de87a0e96 | 91acfe803d52e3b719114d82493797ee | evidence/d11d3b189b72ce851ea17b468189762cff738a2f942918584d19393de87a0e96.json |
| E-627a06828903700a15db969fb4d93e4f5454ab9bb289be6cc49e6eed572a0534 | 91acfe803d52e3b719114d82493797ee | evidence/627a06828903700a15db969fb4d93e4f5454ab9bb289be6cc49e6eed572a0534.json |
| E-ddbb3756b5e4ba9636a3202a4c5f8de3b18242bfa81508a88206a43f8080db7e | 91acfe803d52e3b719114d82493797ee | evidence/ddbb3756b5e4ba9636a3202a4c5f8de3b18242bfa81508a88206a43f8080db7e.json |
| E-09a8ab6f1c208d2254eb557e13c2bf375141fd291888ebf0d1f8d674952be6b8 | 91acfe803d52e3b719114d82493797ee | evidence/09a8ab6f1c208d2254eb557e13c2bf375141fd291888ebf0d1f8d674952be6b8.json |
| E-67a0bdff0153f8ba4862209481947f75828aa847dbabdaa27db44b91d88f6f78 | 91acfe803d52e3b719114d82493797ee | evidence/67a0bdff0153f8ba4862209481947f75828aa847dbabdaa27db44b91d88f6f78.json |
| E-37ce66ca35a1b364af027067ba3bf202fe016bbba8ab012c33576136f598e40c | 91acfe803d52e3b719114d82493797ee | evidence/37ce66ca35a1b364af027067ba3bf202fe016bbba8ab012c33576136f598e40c.json |
| E-7d6bbbfa540d2f809e8dc8404613bf79e350c379aefc0a4af69db137ee7971c9 | 91acfe803d52e3b719114d82493797ee | evidence/7d6bbbfa540d2f809e8dc8404613bf79e350c379aefc0a4af69db137ee7971c9.json |
| E-d8f0fa098b9c9622d4c2c66d8b2165c9d7aa6b777d89a774c6af0c252df155d9 | 91acfe803d52e3b719114d82493797ee | evidence/d8f0fa098b9c9622d4c2c66d8b2165c9d7aa6b777d89a774c6af0c252df155d9.json |

This is a draft for human review. Export hashes describe bytes, not independent evidence authenticity.
