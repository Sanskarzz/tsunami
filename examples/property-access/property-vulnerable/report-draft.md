# Tsunami engagement draft

Web/API authorization assessment

Engagement: `4032d0302ef4c739a92b3c147ccc893a`; state: stopped. Synthetic targets only.

Selected procedures: 1; completed: 1; skipped: 0; pending: 0; running: 0; failed: 0; stopped: 0; incomplete evidence: 0.

Completion means the selected procedure captured its prerequisites and probe. It does not mean the application is secure.

| Procedure | Progress | Verdict |
|---|---|---|
| property-vulnerable/property-access/v1 | completed | confirmed |

## Finding candidates

Candidates come from broker-captured evidence on owned lab data. Explicit review records are separate from the observation.

### F-fe08dbba4bb4925a2b89d24d403c7291

Owner-only note exposed through a shared record

Status: needs_human_review; severity: unrated.

The peer&\#39;s permitted object read included a known owner-only property value; identities, legitimate private views and follow-up access were verified.

Remediation: Authorize each response property for the requesting principal. Use an explicit public response projection and omit the protected fields from the peer&\#39;s view.

Evidence IDs: E-7980dbf05239387b6d95bf921b2af365707b150018e4c4a47429ac6b729325b9, E-1bdcee7b975f7101c40862a7f791a640c72fc6e3cacdf10dff4cd0740ecb04ce, E-0569445301052200297e920312ad7000e1a8a1850af3d58ea07881863fb51d71, E-32fa09211fd527cd40054ec50351c0288d159d2a6c4d624c6c0fbc898cc1ab1b, E-2459e8534adbc861dd7620b74493d6225443320fad6f6f5e926eac4ffe946aff, E-ada8c6a76672e9aae6a48db18b8680728756f13ab68e8abd08eefa93bd68d4ea, E-3b41535885b919ee55561bcb1c3a6c82c73a07f5991abc79aca58af19507dc24

## Evidence index

| Evidence | Run | Artifact path |
|---|---|---|
| E-e4d3db4bbaf6d0457dcfffbfd5810ebf8a2a3de5df296c2fa5a327b37dfbe2db | 711b29185127e834ff8c8cf036b50cd9 | evidence/e4d3db4bbaf6d0457dcfffbfd5810ebf8a2a3de5df296c2fa5a327b37dfbe2db.json |
| E-7980dbf05239387b6d95bf921b2af365707b150018e4c4a47429ac6b729325b9 | 711b29185127e834ff8c8cf036b50cd9 | evidence/7980dbf05239387b6d95bf921b2af365707b150018e4c4a47429ac6b729325b9.json |
| E-1bdcee7b975f7101c40862a7f791a640c72fc6e3cacdf10dff4cd0740ecb04ce | 711b29185127e834ff8c8cf036b50cd9 | evidence/1bdcee7b975f7101c40862a7f791a640c72fc6e3cacdf10dff4cd0740ecb04ce.json |
| E-0569445301052200297e920312ad7000e1a8a1850af3d58ea07881863fb51d71 | 711b29185127e834ff8c8cf036b50cd9 | evidence/0569445301052200297e920312ad7000e1a8a1850af3d58ea07881863fb51d71.json |
| E-32fa09211fd527cd40054ec50351c0288d159d2a6c4d624c6c0fbc898cc1ab1b | 711b29185127e834ff8c8cf036b50cd9 | evidence/32fa09211fd527cd40054ec50351c0288d159d2a6c4d624c6c0fbc898cc1ab1b.json |
| E-2459e8534adbc861dd7620b74493d6225443320fad6f6f5e926eac4ffe946aff | 711b29185127e834ff8c8cf036b50cd9 | evidence/2459e8534adbc861dd7620b74493d6225443320fad6f6f5e926eac4ffe946aff.json |
| E-ada8c6a76672e9aae6a48db18b8680728756f13ab68e8abd08eefa93bd68d4ea | 711b29185127e834ff8c8cf036b50cd9 | evidence/ada8c6a76672e9aae6a48db18b8680728756f13ab68e8abd08eefa93bd68d4ea.json |
| E-3b41535885b919ee55561bcb1c3a6c82c73a07f5991abc79aca58af19507dc24 | 711b29185127e834ff8c8cf036b50cd9 | evidence/3b41535885b919ee55561bcb1c3a6c82c73a07f5991abc79aca58af19507dc24.json |

This is a draft for human review. Export hashes describe bytes, not independent evidence authenticity.
