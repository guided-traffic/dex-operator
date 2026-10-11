# Relying parties: confidential and public static clients, their trusted peers and CORS origins

What a `DexStaticClient` registers, what protects a confidential client and what protects a public
one, whom a client may trust with its audience, and what `cors: true` adds to the installation.
Which namespaces may register clients at all, and which client holds an ID several claim, is
[tenancy.md](tenancy.md); how the client secret reaches Dex is [secret-flow.md](secret-flow.md).
The decisions are
[ADR 0004](../adr/0004-a-static-client-is-confidential-through-a-secret-or-public-through-an-inline-id-validated-by-cel.md),
[ADR 0005](../adr/0005-a-static-client-opts-in-to-derive-its-cors-origins-from-its-own-https-redirect-uris.md)
and [ADR 0009](../adr/0009-a-client-trusts-only-peers-held-in-its-own-namespace.md).

## What a registration is

A static client brings its client ID, its `redirectURIs`, its `trustedPeers` and, when
confidential, its secret. The redirect URIs are the security-critical part: an authorization code
for a user who logs in to that client goes to them. Who may set them is decided by
`allowedNamespaces` and by RBAC on `dexstaticclients` ([tenancy.md](tenancy.md)). The client ID is
the key of the registration: it renders for one client of the installation only, and a client
outside the installation's namespace can never take it over from another
([tenancy.md](tenancy.md#one-child-per-id)).

## Trusted peers: who may obtain tokens for this client's audience

`trustedPeers` on a client B lists client IDs. A client A that requests the scope
`audience:server:client_id:B` gets an ID token with audience B when B's `trustedPeers` names A's
ID (`validateCrossClientTrust`, `server/oauth2.go`, dex v2.44.0). Every resource server that
accepts audience B accepts that token. The trust is granted to an **ID**, not to an object: Dex
trusts whoever holds the listed ID.

The operator therefore renders a `trustedPeers` entry only when the ID is held by a static client
in the trusting client's **own namespace**
([internal/builder/clients.go](../../internal/builder/clients.go) `filterTrustedPeers`). An ID is
held by the one client the contest rule leaves it to — the one that renders, or the sole claimant
whose build failed, so a gap in a peer's Secret does not rewrite the trusting client's entry. An
entry naming an ID held in another namespace, contested, or held by nobody — a peer that is not
deployed yet, or was removed — is left out of the config, listed in the installation's
`status.droppedTrustedPeers`, and shown on the trusting client as `TrustedPeersDropped=True` with
the peer IDs, never the namespace of their holder. It comes back on the render in which a client
of that namespace holds the ID; holders and the filter always come from the same render. The rule
holds for the installation's namespace too: a platform client trusts only platform peers.

Within one namespace, RBAC on `dexstaticclients` governs both ends of the trust, as it governs
both registrations. A client never needs to list itself: Dex grants a client its own audience.

## Confidential and public clients

A **confidential** client (`secretRef`) authenticates with a secret at the token endpoint; the
redirect-URI allowlist is its second control. A **public** client (`public: true`, inline
`clientID`) has **no** secret — by definition its binary runs on user devices where any embedded
secret is extractable. Its security rests on:

1. **PKCE**, which binds the authorization code to the party that started the flow and defeats
   code interception.
2. **The redirect-URI policy.** With `redirectURIs` omitted, Dex accepts loopback addresses
   (`http://localhost:<port>`), the OOB URN and the device-flow callback — correct for CLIs and
   native apps, useless for servers.

A hybrid exists too: `public: true` with `secretRef` — PKCE plus a secret. The schema guarantees
that a client is one of these and never an ambiguous mix ([validation.md](validation.md)); a
secretless client leaves no entry in the env Secret
([internal/builder/clients.go](../../internal/builder/clients.go) `buildOneStaticClient`).

**Operational rule:** model server-side applications as confidential, interactive tools as public
([H-11](#h-11)).

## Tenant-registered CORS origins (`cors: true`)

A `DexStaticClient` with `cors: true` contributes the origins of its own **https** `redirectURIs`
to the installation's `web.allowedOrigins`
([internal/builder/builder.go](../../internal/builder/builder.go) `deriveCORSOrigins`). It is a
boolean on purpose and not a free-form list: **it grants no authority beyond `redirectURIs`**. A
tenant can only allowlist origins it already controls as redirect targets — and a tenant able to
set an arbitrary redirect URI holds a far stronger primitive than a CORS entry. Derivation never
produces `"*"`; that value stays reachable only through the platform's own
`spec.web.allowedOrigins`.

What CORS does here: it is **defense in depth, not an authorization boundary**. Dex wraps six
routes with CORS — discovery, `/`, `/token`, `/keys`, `/userinfo`, `/token/introspect` — and sets
only `AllowedOrigins` and `AllowedHeaders`, not `AllowCredentials` (dex v2.44.0 `server/server.go`).
Each of those routes needs either no credential (public metadata) or one the calling script must
already hold — a code plus PKCE verifier or client secret, an access token. No ambient credential
is involved, so an extra origin gains nothing its server could not do with plain HTTP, except
reading those routes through a victim's browser where Dex is reachable only internally — which
without a credential yields public metadata. Matching is **literal**
(`gorilla/handlers.AllowedOrigins`): no subdomain wildcards, so a compromised sibling host
inherits nothing, and the operator lowercases the host and drops a redundant `:443` so a derived
entry can match at all.

Dex has no per-client CORS, so an origin derived from client A applies to the flows of client B as
well; it still needs B's code and verifier or secret to do anything, and B's codes go to B's
redirect URIs.

## What this does not cover

<a id="h-11"></a>
### H-11 — Marking a server application public removes its client authentication

Live wherever it is done. `public: true` with an inline `clientID` passes validation for any
client; the schema cannot tell a server from a CLI. The token endpoint then asks that client for
no secret, so an authorization code that leaks — from a log, a `Referer` header, a proxy — is
redeemed by whoever holds it, unless the application uses PKCE, which a server-side application
written for a confidential client usually does not. Mitigation: review `public: true` on every
client whose redirect URIs are https URLs of a server.

<a id="h-12"></a>
### H-12 — A tenant in an admitted namespace grows the installation's origin list

Live by design (ADR 0005). With `cors: true`, the installation's `web.allowedOrigins` depends on
tenant resources. The bound is each client's own https redirect URIs; the control is the one that
already gates client registration — `allowedNamespaces` plus RBAC on `dexstaticclients`. There is
no installation-level switch to refuse derived origins. Mitigation: narrow `allowedNamespaces`
where the authored list must be the whole list.
