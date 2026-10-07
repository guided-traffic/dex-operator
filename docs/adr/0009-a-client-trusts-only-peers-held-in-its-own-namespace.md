# ADR 0009: A Client Trusts Only Peers Held in Its Own Namespace

## Status

Accepted. Date: 2026-10-07.

**Implemented**, in the release after 2.3.0.

## Context

`trustedPeers` on a static client B lists client IDs. A client A that requests the scope
`audience:server:client_id:B` gets an ID token with audience B when B's `trustedPeers` names A's
ID (`validateCrossClientTrust`, `server/oauth2.go`, dex v2.44.0). The trust is granted to an ID,
not to an object: whoever registers that ID in the installation holds it.
[ADR 0008](0008-an-id-renders-for-one-child-only-and-a-failing-child-is-left-out-instead-of-failing-the-render.md)
decides who holds an ID, but nobody contests an ID that a client trusts while no client holds it
yet — a peer that is not deployed yet, or was removed.

## Decision

**D1 — A rendered `trustedPeers` entry names an ID held in the trusting client's own namespace.**
An ID is held by the single client the contest rule leaves it to — the one that renders, or the
sole claimant that failed to build, so a gap in a peer's Secret does not rewrite the trusting
client's entry. Every other entry — held in another namespace, contested, or held by nobody — is
left out of the config ([internal/builder/clients.go](../../internal/builder/clients.go)
`filterTrustedPeers`). It comes back on the render in which a client of that namespace holds the
ID.

**D2 — Holders and the filter come from one render.** Dex never loads a config that trusts a peer
whose holder sits outside the trusting client's namespace, in any order of changes.

**D3 — The rule is symmetric.** It holds in the installation's namespace too: a platform client
trusts only platform peers. Within one namespace, RBAC on `dexstaticclients` governs both ends of
the trust, as it already governs both registrations.

**D4 — Dropped entries are reported.** The installation carries `status.droppedTrustedPeers`
(namespace, name, peer ID, sorted) and the condition `TrustedPeersDropped`; the trusting client
carries `TrustedPeersDropped=True` with its dropped peer IDs and never the namespace of their
holder, and stays `Ready=True`.

## Consequences

- Cross-namespace trusted peers stop working on upgrade: the entry is dropped, the config changes
  and Dex rolls out; token requests from that peer for the trusting client's audience fail until
  both clients live in one namespace. Moving a confidential client moves its `secretRef` Secret
  with it.
- A peer that is not deployed yet costs a `TrustedPeersDropped=True` condition, not a trust that
  whoever deploys first receives.

## Alternatives Considered

- **Reserve the IDs that platform clients trust** for the installation's namespace. Protects
  platform clients only, checks at the claim instead of the trust, and breaks tenant clients that
  are deliberate peers of a platform client. Lost.
- **Document the gap only.** Holds until an administrator forgets an entry, and nothing reports
  it. Lost.
- **Pinned cross-namespace peers** (`namespace/id` entries, or a `trustedPeerRefs` field). Keeps
  every use case, but extends the API now, and `namespace/id` is ambiguous for IDs containing `/`.
  Not built; it loosens the rule additively, so nothing breaks when it is added on a concrete
  need.

## Residual risks

- A client that trusts itself is redundant: Dex grants a client its own audience without
  `trustedPeers` (`validateCrossClientTrust` returns early for `peerID == clientID`).
- Within one namespace the trust is as strong as RBAC on `dexstaticclients` there.

## References

- [internal/builder/clients.go](../../internal/builder/clients.go) `filterTrustedPeers`,
  [internal/controller/child_reconciler.go](../../internal/controller/child_reconciler.go)
  `setTrustedPeersDroppedCondition`
- [internal/builder/contest_test.go](../../internal/builder/contest_test.go) `TestBuild_TrustedPeers*`,
  [test/integration/claims_test.go](../../test/integration/claims_test.go)
  `TestIntegration_TrustedPeerInOtherNamespace`,
  [test/e2e/claims_e2e_test.go](../../test/e2e/claims_e2e_test.go)
  `TestE2E_PlatformClientTrustsTenantHeldPeer`
- [docs/security/clients.md](../security/clients.md),
  [ADR 0008](0008-an-id-renders-for-one-child-only-and-a-failing-child-is-left-out-instead-of-failing-the-render.md)
