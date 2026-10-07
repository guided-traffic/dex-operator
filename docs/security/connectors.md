# Identity sources: what an admitted connector can make every client believe

What a connector adds to an installation, why every connector kind is an identity source for every
client, and the kinds whose risk does not depend only on who wrote them. Which namespaces may add
connectors at all is [tenancy.md](tenancy.md). Claims about Dex were read at dex **v2.44.0**
(`connector/authproxy/authproxy.go`, `server/server.go`, `cmd/dex/serve.go`,
`storage/storage.go`).

## Every connector reaches every client

Dex has no per-client connector restriction: `storage.Client` has no field that limits a client
to some connectors, so every connector of the installation logs users in to every client. For any
connector whose upstream the connector's author runs or picks — OIDC, OAuth2, SAML, LDAP,
Keystone, AtlassianCrowd, Gitea, OpenShift, self-hosted GitLab — the upstream returns whatever
`email`, `email_verified` and `groups` its operator wants. Dex passes these claims through and
encodes the connector ID only into `sub`, so a client that authorizes by email or groups — most of
them — accepts the forged identity. SaaS-only kinds (Google, LinkedIn, Microsoft, Bitbucket,
GitHub) limit what can be claimed, but the connector's author still chooses which upstream
accounts, organisations and groups are trusted.

That is why connectors have their own allowlist with a restrictive default
([tenancy.md](tenancy.md),
[ADR 0006](../adr/0006-two-namespace-allowlists-one-per-child-category-exact-names-only.md)):
admitting a namespace to connectors is trusting it with identity issuance for the whole
installation, and a per-kind split would not reduce that for OIDC and its like.

## The authproxy connector trusts request headers

`DexAuthProxyConnector` is a production integration: a front proxy performs a login Dex does not
support natively and Dex takes over the result. Its `HandleCallback` takes the user, email, user
ID and groups **directly from request headers** and returns `EmailVerified: true`; `staticGroups`
are added to every identity. The login URL is `/callback/<connector-id>`. Dex strips `X-Remote-*`
headers only on the plain `/callback` route; the connector's own `/callback/<connector-id>` route
strips nothing, and custom header names (`userHeader`, `groupHeader`, …) are never stripped. The
Dex documentation requires the proxy to remove these headers for any path.

For every other kind, only the connector's author can abuse the connector. For this one, an honest
mistake by its author is enough for an outsider ([H-7](#h-7)).

## The local connector is an installation-wide switch

A `DexLocalConnector` renders no connector entry. Its presence only sets `enablePasswordDB: true`
([internal/builder/connectors.go](../../internal/builder/connectors.go) `buildAllConnectors`,
[builder.go](../../internal/builder/builder.go) `assembleDexConfig`), and Dex then registers a
fixed connector with ID `local` and name `Email` for every client of the installation. The
operator renders no `staticPasswords`. Whoever creates the resource cannot create users with it;
the risk lies in who can write password entries ([H-8](#h-8)).

## Upstream accounts

The bind DN of an LDAP connector, the admin account of a Keystone connector and the
service-account JSON of a Google connector are credentials the operator passes on as they are
([secret-flow.md](secret-flow.md)). The operator cannot tell how much they may do upstream; give
them read-only, minimal scope.

## What this does not cover

<a id="h-6"></a>
### H-6 — Every admitted connector issues identities every client accepts

Live by design of Dex. A connector in a namespace admitted to connectors can log any user in to
any client of the installation with any email and groups its upstream returns. The allowlist
decides which namespaces are trusted with that; it does not narrow what a trusted namespace can
do. Mitigation: keep `allowedConnectorNamespaces` omitted unless a tenant must run its own
identity provider, and have relying parties authorize on `sub` (which carries the connector ID)
where that matters.

<a id="h-7"></a>
### H-7 — An authproxy connector without a protecting proxy logs anyone in as anyone

Live wherever a `DexAuthProxyConnector` is admitted and Dex is reachable on a path that does not
go through a proxy that authenticates `/callback/<connector-id>` and overwrites the identity
headers. Then anyone who reaches Dex logs in as any user with any groups. The operator cannot see
the network path. Mitigation: deploy it only behind a proxy that is the exclusive network path to
Dex and overwrites `X-Remote-*` and every custom header name on every path; keep it in the
installation's own namespace — the protecting proxy is part of the ingress in front of Dex, owned
by whoever runs the installation, not by a tenant.

<a id="h-8"></a>
### H-8 — With the password database on, whoever writes password entries controls logins

Live wherever a `DexLocalConnector` is admitted. Password entries live in Dex's storage — with
`kubernetes` storage, the `passwords.dex.coreos.com` objects in the Dex namespace — and can be
written through Dex's gRPC API. Dex requires client certificates on gRPC only when
`grpc.tlsClientCA` is set; without it, anyone who reaches the gRPC port can call `CreatePassword`
and log in with that account to every client. Mitigation: set `grpc.tlsClientCA` (or leave gRPC
off), restrict who can write `passwords.dex.coreos.com`, and keep the gRPC port off every network
path a tenant reaches.

<a id="h-9"></a>
### H-9 — `insecure*` options pass through unflagged

Live wherever they are set. Every connector option prefixed `insecure` — skip TLS verification,
skip signature validation, skip the email-verified check, `insecureNoSSL`, `insecureCA` — removes
a verification step. The operator renders them as given and reports nothing. Mitigation: allow
them in development installations only and look for them:
`kubectl get dex<type>connectors -A -o yaml | grep -n insecure`.

<a id="h-10"></a>
### H-10 — A connector without upstream filters accepts any account of its provider

Live wherever a connector omits them. Without `orgs`, `groups`, `hostedDomains`, `teams` or
`tenant` (whichever its kind offers), a GitHub, Google, Microsoft or similar connector accepts
every account of that provider. The operator enforces no filter. Mitigation: filter at the
connector, not only in the relying party.
