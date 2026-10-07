# ADR 0005: A Static Client Opts In to Derive Its CORS Origins From Its Own https Redirect URIs

## Status

Accepted. Date: 2026-10-07, recording the decisions taken with the owner and built in 2.2.0
(2026-08-18). The owner asked for them not to be reverted silently.

**Implemented.**

## Context

Dex gates its browser-facing endpoints — discovery, `/`, `/token`, `/keys`, `/userinfo`,
`/token/introspect` — behind CORS. Server-side clients never meet it; a single-page application
doing the code flow with PKCE from the browser does. The only knob was the installation's
`spec.web.allowedOrigins`, a list in platform territory, while static clients register themselves
from their own namespaces: every new browser client needed a paired edit of the installation.

Dex matches the `Origin` header literally (`gorilla/handlers.AllowedOrigins`, dex v2.44.0
`server/server.go`); the only wildcard is the whole `"*"`. Deployments through the Dex Helm chart
pass the listener addresses as `--web-http-addr`/`--web-https-addr` flags, applied after the config
is loaded.

## Decision

**D1 — A flag, not a list.** `DexStaticClientSpec.CORS` (`json:"cors,omitempty"`) opts the client
in; the origins are derived from the client's own `redirectURIs` (`deriveCORSOrigins`,
[internal/builder/builder.go](../../internal/builder/builder.go)). The flag grants no authority
beyond `redirectURIs`, which is already the security-critical, allowlist- and RBAC-gated field.

**D2 — https only.** Loopback and other http URIs, custom schemes and the OOB URN are skipped; an
unparsable URI is skipped instead of failing the render, so one malformed resource cannot break
the config of every other client. Derivation can never produce `"*"`.

**D3 — The authored list keeps its order; derived origins are appended sorted**
(`appendDerivedOrigins`). Sorting the whole union would rewrite existing authored lists on an
operator upgrade alone → a config diff → a spurious Dex rollout
([ADR 0003](0003-the-render-is-deterministic-and-change-driven-no-periodic-requeue.md)). The sorted
tail makes the output independent of client iteration order.

**D4 — The host is lowercased and a redundant `:443` dropped,** because Dex compares literally and
an unnormalised entry would never match.

**D5 — The flag is not gated on `public`.** A confidential client that sets it derives too.

**D6 — Nothing lands in the rendered `staticClients` entry.** Dex has no per-client CORS; the
derived origins go to the installation's `web.allowedOrigins` only (`assembleWebConfig`).

**D7 — `spec.web == nil` plus derived origins creates the `web:` block** with only
`allowedOrigins`; the listener addresses come from the chart's flags.

## Consequences

- A tenant in an admitted namespace can grow the installation's origin list, bounded by its own
  https redirect URIs.
- Removing the flag or the client removes its origins on the next render; the derivation keeps no
  state.
- `"*"` stays reachable, but only through the platform's own `spec.web.allowedOrigins`.

## Alternatives Considered

- **A free-form origin list per client.** Would need validation against `"*"`, wildcards and
  foreign origins, and the safe rule — "must be the origin of one of your own redirect URIs" —
  makes the field redundant. Lost.
- **Sort and deduplicate the whole union.** Deterministic, but rewrites authored lists on upgrade
  and so restarts Dex once for nothing. Lost (D3).
- **A separate namespace allowlist for clients with `cors: true`.** Would gate the smaller power
  (an origin) behind a second list while the bigger one (a redirect target that receives
  authorization codes) stays behind the first. Lost; if a platform must forbid derived origins,
  the smallest control is an installation-level boolean, not built without a need.
- **Subdomain wildcards.** Dex cannot match them. Lost.

## Residual risks

- CORS is defense in depth here, not an authorization boundary: the CORS-wrapped endpoints carry
  no ambient credentials, so an origin gains nothing a server could not do with plain HTTP — except
  reading those endpoints through a victim's browser where Dex is reachable only internally, which
  without a credential yields public metadata.
- Not verified: how `gorilla/handlers` answers a preflight from an origin not on the list (403 or
  an empty 200). No route depends on CORS for authentication, so the answer does not change D1.

## References

- [api/v1/dexstaticclient_types.go](../../api/v1/dexstaticclient_types.go) `CORS`
- [internal/builder/builder.go](../../internal/builder/builder.go) `deriveCORSOrigins`,
  `appendDerivedOrigins`, `assembleWebConfig`
- [docs/security/clients.md](../security/clients.md)
- [ADR 0006](0006-two-namespace-allowlists-one-per-child-category-exact-names-only.md) — the
  allowlist that gates who may set the flag
