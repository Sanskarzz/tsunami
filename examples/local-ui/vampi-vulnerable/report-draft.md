# Tsunami engagement draft

UI qualification: vampi-vulnerable

Engagement: `8d38b0cee8dbd24b7984beb7fd11328e`; state: active. Synthetic targets only.

Selected procedures: 1; completed: 1; skipped: 0; pending: 0; running: 0; failed: 0; stopped: 0; incomplete evidence: 0.

Completion means the selected procedure captured its prerequisites and probe. It does not mean the application is secure.

| Procedure | Progress | Verdict |
|---|---|---|
| vampi-vulnerable/object-access/v2 | completed | confirmed |

## Finding candidates

Candidates come from broker-captured evidence on owned lab data. Explicit review records are separate from the observation.

### F-ab0803e2850f855a369db4ef3aebe654

Cross-user book secret access

Status: needs_human_review; severity: unrated.

The authenticated peer received the owner&\#39;s approved proof data; account identities and owner access were independently checked.

Remediation: Authorize every object read against the authenticated principal and intended sharing rules on the server.

Evidence IDs: E-04d56464f72ffdbb6cd7b7cfc7e4bb427f8aad9a6d351f041b56de1afbdffe61, E-5ea919b374ab4259ef9f373cb7667baf1714035510427653c9778feccaccfce3, E-2fbc196b75230c996356cfa6041a68bcbb27acf8337d7bcedcbcd350263e0107, E-b10d375ceca820232cd42e4ab18dd88c8292ef4634662a9798a6356c35b7c946, E-f8e109b04f828042d93c563bd99d872de05f814503ad9c40df6004508627d409, E-cc72a1de3489fbaf05218df0edf06a734a86ae0d69466fd06f84b1b09a4d9489, E-20c9557f9349342c9326680ca9bfbc3b1c9a674b44e5296656bc1b76dd02457b

## Evidence index

| Evidence | Run | Artifact path |
|---|---|---|
| E-1dedb18f939e92d91f0a4fb88d70f1361cd9255ade3f000a670a629269df6ce9 | 75634fff085f8f6ee98ce32c1d1ffd20 | evidence/1dedb18f939e92d91f0a4fb88d70f1361cd9255ade3f000a670a629269df6ce9.json |
| E-04d56464f72ffdbb6cd7b7cfc7e4bb427f8aad9a6d351f041b56de1afbdffe61 | 75634fff085f8f6ee98ce32c1d1ffd20 | evidence/04d56464f72ffdbb6cd7b7cfc7e4bb427f8aad9a6d351f041b56de1afbdffe61.json |
| E-5ea919b374ab4259ef9f373cb7667baf1714035510427653c9778feccaccfce3 | 75634fff085f8f6ee98ce32c1d1ffd20 | evidence/5ea919b374ab4259ef9f373cb7667baf1714035510427653c9778feccaccfce3.json |
| E-2fbc196b75230c996356cfa6041a68bcbb27acf8337d7bcedcbcd350263e0107 | 75634fff085f8f6ee98ce32c1d1ffd20 | evidence/2fbc196b75230c996356cfa6041a68bcbb27acf8337d7bcedcbcd350263e0107.json |
| E-b10d375ceca820232cd42e4ab18dd88c8292ef4634662a9798a6356c35b7c946 | 75634fff085f8f6ee98ce32c1d1ffd20 | evidence/b10d375ceca820232cd42e4ab18dd88c8292ef4634662a9798a6356c35b7c946.json |
| E-f8e109b04f828042d93c563bd99d872de05f814503ad9c40df6004508627d409 | 75634fff085f8f6ee98ce32c1d1ffd20 | evidence/f8e109b04f828042d93c563bd99d872de05f814503ad9c40df6004508627d409.json |
| E-cc72a1de3489fbaf05218df0edf06a734a86ae0d69466fd06f84b1b09a4d9489 | 75634fff085f8f6ee98ce32c1d1ffd20 | evidence/cc72a1de3489fbaf05218df0edf06a734a86ae0d69466fd06f84b1b09a4d9489.json |
| E-20c9557f9349342c9326680ca9bfbc3b1c9a674b44e5296656bc1b76dd02457b | 75634fff085f8f6ee98ce32c1d1ffd20 | evidence/20c9557f9349342c9326680ca9bfbc3b1c9a674b44e5296656bc1b76dd02457b.json |

This is a draft for human review. Export hashes describe bytes, not independent evidence authenticity.
