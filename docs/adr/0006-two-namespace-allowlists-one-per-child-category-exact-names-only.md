# ADR 0006: Two Namespace Allowlists, One per Child Category, Exact Names Only

## Status

Accepted. Date: 2026-10-07, recording the decision built in 2.3.0 (2026-10-06). The single list
`allowedNamespaces` it split had governed both categories since the first release.

**Implemented.**

## Context

Exactly two kinds of resources reference a `DexInstallation`: sixteen connector kinds and
`DexStaticClient` (the `registerIndexers` list,
[internal/controller/dexinstallation_controller.go](../../internal/controller/dexinstallation_controller.go)).
They carry different trust. A static client registers one relying party. A connector adds an
identity source, and every client of the installation accepts the identities it produces: Dex has
no per-client connector restriction (`storage.Client` has no such field, dex v2.44.0). For any
connector whose upstream the tenant runs or picks — OIDC, OAuth2, SAML, LDAP, Keystone,
AtlassianCrowd, Gitea, OpenShift, self-hosted GitLab — the upstream returns whatever `email`,
`email_verified` and `groups` the tenant wants, and Dex encodes the connector ID only into `sub`,
so clients that authorize by email or groups accept the forged identity.

One list decided for both at once: admitting a namespace for client registration — the normal
case of tenant self-service — admitted it for identity issuance as well. The operator has no
permission on `namespaces` and should not need one.

## Decision

**D1 — `allowedNamespaces` governs static clients only.** Empty or omitted denies all (the
installation's own namespace is not implied), `"*"` admits every namespace, any other entry is a
literal namespace name.

**D2 — `allowedConnectorNamespaces` governs every connector kind.** Omitted, it admits only the
installation's own namespace (`connectorNamespaces`,
[internal/controller/namespace.go](../../internal/controller/namespace.go)). Set, it is
exhaustive — the own namespace only if listed. `"*"` admits all and must be the only entry.

**D3 — `allowedConnectorNamespaces: []` is unrepresentable.** `MinItems=1`, `listType=set`, and a
CEL rule `!self.exists(n, n == '*') || self.size() == 1`
([api/v1/dexinstallation_types.go](../../api/v1/dexinstallation_types.go)). With `omitempty`, a Go
client writing the object back drops `[]`, and "deny all" would silently turn into "own
namespace". An installation that wants no connectors has none.

**D4 — Exact names only.** No label selectors, no name patterns, in either list.

**D5 — The lists are enforced twice.** At build time `collectStaticClients` and
`collectConnectors` filter through `filterItems` → `isNamespaceAllowed`, so a resource outside its
list never reaches the render whatever its own status says
([internal/controller/collect.go](../../internal/controller/collect.go)). At child reconciliation
`checkChildNamespace` sets `Ready=False` with a message naming the governing field, and for a
connector with the field omitted adds `(omitted: only "<ns>" is allowed)`.

**D6 — Child status follows the installation.** Every child reconciler watches `DexInstallation`
(generation changes, creates and deletes only) and enqueues the children of its kind that
reference it (`mapInstallationToChildren`, through `InstallationRefIndexField`). The
DexInstallation controller registers that index, so it is set up first
([cmd/main.go](../../cmd/main.go)).

**D7 — No per-kind connector rule and no separate category for `cors: true`.** With the
restrictive default, every connector kind comes from the installation's namespace unless a
namespace is admitted explicitly. A derived CORS origin grants less than the redirect URI it is
derived from, which `allowedNamespaces` already gates
([ADR 0005](0005-a-static-client-opts-in-to-derive-its-cors-origins-from-its-own-https-redirect-uris.md)).

## Consequences

- Upgrading to 2.3.0 changed what is admitted: connectors outside the installation's namespace
  were dropped until listed in `allowedConnectorNamespaces`, and connectors in the installation's
  own namespace that the old `allowedNamespaces` excluded became active. Installations whose
  connectors all lived in the own namespace rendered byte-identical config. The README's upgrade
  note says so.
- Admitting a namespace to `allowedConnectorNamespaces` is trusting it with identity issuance for
  every client of the installation; the field makes that a separate, visible decision.
- The admission decision stays with whoever can write the `DexInstallation`: namespace names are
  immutable, and no lookup of `Namespace` objects is needed.

## Alternatives Considered

- **Label selectors.** Need cluster-wide `get/list/watch` on `namespaces` and move the decision to
  whoever can label a namespace. Lost.
- **Name patterns (`team-*`).** Need no RBAC, but move the decision to whoever can create a
  namespace with a matching name. Not built; a later extension would be a separate field, so the
  meaning of an existing list never changes.
- **A per-kind rule** (for example `DexAuthProxyConnector` only from the own namespace). Matters
  only once tenant namespaces are admitted to connectors, which the allowlist makes the visible
  exception. If it becomes the rule, that fixed AuthProxy rule is the smallest extension, because
  an AuthProxy connector is the one kind where an honest mistake opens the installation to
  outsiders ([docs/security/connectors.md](../security/connectors.md)).
- **An installation-level switch for derived origins** (`web.deriveClientOrigins`). Not built
  without a concrete need.

## Residual risks

- A listed name that does not exist yet admits whoever creates that namespace first.
- Removing a namespace drops its children on the next render; with `rolloutRestart.enabled:
  false`, Dex serves the old config until it restarts.
- Within an admitted namespace the boundary is Kubernetes RBAC on the Dex CRDs; the allowlists
  decide which namespaces may register, not what a registration inside them may claim.

## References

- [api/v1/dexinstallation_types.go](../../api/v1/dexinstallation_types.go),
  [internal/controller/namespace.go](../../internal/controller/namespace.go),
  [collect.go](../../internal/controller/collect.go),
  [child_reconciler.go](../../internal/controller/child_reconciler.go)
- [internal/controller/namespace_test.go](../../internal/controller/namespace_test.go),
  [test/integration/connector_namespaces_test.go](../../test/integration/connector_namespaces_test.go),
  [test/e2e/dexinstallation_e2e_test.go](../../test/e2e/dexinstallation_e2e_test.go)
  `TestE2E_NamespaceIsolation`, `TestE2E_ConnectorNamespaceDefault`
- [docs/security/tenancy.md](../security/tenancy.md)
