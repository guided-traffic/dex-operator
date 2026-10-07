# Tenancy: which namespaces may contribute what

How a `DexInstallation` decides which namespaces may add static clients and which may add
connectors, where that decision is enforced, and what it leaves to Kubernetes RBAC. What an
admitted connector can do once it is in is [connectors.md](connectors.md); what an admitted static
client can do is [clients.md](clients.md). The decision and its alternatives are
[ADR 0006](../adr/0006-two-namespace-allowlists-one-per-child-category-exact-names-only.md).

## Two allowlists, one per category

Exactly two categories of resources reference an installation — sixteen connector kinds and
`DexStaticClient` — and they carry different trust, so each has its own list
([api/v1/dexinstallation_types.go](../../api/v1/dexinstallation_types.go)):

| Field | Governs | Omitted / empty | `"*"` | Other entries |
|---|---|---|---|---|
| `allowedNamespaces` | `DexStaticClient` | deny all — fail-closed; the own namespace is not implied | every namespace | literal namespace names |
| `allowedConnectorNamespaces` | all sixteen connector kinds | only the installation's own namespace | every namespace (must be the only entry) | literal names; exhaustive, so list the own namespace too if connectors live there |

`allowedConnectorNamespaces: []` is rejected at admission (`MinItems=1`), and a CEL rule refuses
`"*"` next to another entry. With `omitempty`, any Go client that writes the object back would
drop `[]` and silently turn "deny all" into "own namespace"; the schema makes that third state
unrepresentable instead, and a list can never look restrictive while admitting everything.

**Why connectors get their own, restrictive list.** A static client registers one relying party.
A connector adds an identity source, and every client of the installation accepts the identities
it produces ([connectors.md](connectors.md)). Admitting a namespace to
`allowedConnectorNamespaces` is therefore trusting it with identity issuance for the whole
installation. The default keeps connectors in the installation's own namespace, which already
holds the config and env Secrets and is the most trusted namespace of the installation.

## Enforced twice, independently

1. **At build time.** `collectStaticClients` filters through `allowedNamespaces`,
   `collectConnectors` through the effective connector list (`connectorNamespaces`), both via
   `filterItems` → `isNamespaceAllowed`
   ([internal/controller/collect.go](../../internal/controller/collect.go),
   [namespace.go](../../internal/controller/namespace.go)). A resource outside its list never
   influences the rendered config, whatever its own status claims.
2. **At child reconciliation.** `checkChildNamespace` sets `Ready=False` on a resource from a
   namespace its list does not admit, with a message that names the governing field — for a
   connector with `allowedConnectorNamespaces` omitted it adds
   `(omitted: only "<ns>" is allowed)` ([namespace.go](../../internal/controller/namespace.go)).
   The child reconcilers also watch `DexInstallation`, so the condition follows an allowlist edit
   instead of staying stale until the child's own next event
   ([internal/controller/child_reconciler.go](../../internal/controller/child_reconciler.go)
   `mapInstallationToChildren`).

## Exact names, and why

Both fields take exact names only. Label selectors would need cluster-wide `get/list/watch` on
`namespaces` and move the admission decision to whoever can label a namespace; name patterns
would move it to whoever can create a matching namespace. Namespace names are immutable, so the
decision stays with whoever can write the `DexInstallation`. No lookup of `Namespace` objects is
needed, and the operator has no permission on them.

## Choosing the lists

- **`allowedNamespaces`:** prefer explicit names. `"*"` makes every namespace a place to register
  SSO clients and moves the static-client boundary entirely onto RBAC on `dexstaticclients`
  ([H-3](#h-3)).
- **`allowedConnectorNamespaces`:** keep it omitted unless a tenant must run its own identity
  provider. Every extra entry trusts that namespace with identity issuance for every client of
  the installation; `"*"` makes every namespace an identity source.

## What the allowlists defend against

A tenant in an arbitrary namespace registering an OAuth2 client or an identity-providing
connector with the company SSO. A rogue client with an attacker-controlled redirect URI would
receive authorization codes for real users; a rogue connector could mint identities. With the
allowlists, only namespaces the platform explicitly trusts can do either, and trusting a
namespace with connectors is a separate, explicit decision.

## What this does not cover

<a id="h-3"></a>
### H-3 — Inside an admitted namespace, the boundary is RBAC on the Dex CRDs

Live by design. The allowlists decide which namespaces may register, not who inside them may.
`create`/`update` on `dexstaticclients` in an admitted namespace is the power to register SSO
clients; on any `dex*connectors` kind in a namespace admitted to connectors, the power to issue
identities for every client of the installation. The chart ships no ClusterRole that aggregates
these kinds into `edit` or `admin`, so nobody holds the verbs until a role grants them — and a
wildcard role (`resources: ["*"]`) grants them silently. Mitigation: treat those verbs as
privileged and grant them per namespace and team.

<a id="h-4"></a>
### H-4 — A listed namespace that does not exist yet admits whoever creates it first

Live wherever tenants can create namespaces (for example Capsule tenant owners). The operator
never checks that a listed name exists — it has no permission on `namespaces`. Mitigation: list
only names that exist and that the platform owns.

<a id="h-5"></a>
### H-5 — Revocation waits for a Dex restart when `rolloutRestart` is off

Live with `rolloutRestart.enabled: false` (or no `deploymentName`). Removing a namespace from a
list drops its children from the next render at once, but Dex keeps serving the old config —
those clients and connectors included — until its pods restart
([rotation.md](rotation.md)). Mitigation: enable `rolloutRestart`, or restart Dex by hand after
narrowing a list.
