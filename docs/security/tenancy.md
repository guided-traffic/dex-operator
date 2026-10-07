# Tenancy: which namespaces may contribute what, and which IDs they may hold

How a `DexInstallation` decides which namespaces may add static clients and which may add
connectors, which child holds a client or connector ID when several claim it, where those
decisions are enforced, and what they leave to Kubernetes RBAC. What an admitted connector can do
once it is in is [connectors.md](connectors.md); what an admitted static client can do, and whom
it may trust, is [clients.md](clients.md). The decisions and their alternatives are
[ADR 0006](../adr/0006-two-namespace-allowlists-one-per-child-category-exact-names-only.md) and
[ADR 0008](../adr/0008-an-id-renders-for-one-child-only-and-a-failing-child-is-left-out-instead-of-failing-the-render.md).

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

## One child per ID

The allowlists decide which namespaces may register, not which IDs a registration may use. Dex
keys static clients and connectors by ID and serves one entry per ID (`storage.WithStaticClients`,
`storage.WithStaticConnectors`, dex v2.44.0), so the operator decides who holds an ID before it
renders ([internal/builder/render.go](../../internal/builder/render.go) `contest`):

- Every admitted static client claims its client ID — `spec.clientID`, or the `client-id` from its
  Secret — and every admitted connector its effective ID, across all sixteen kinds. Client IDs and
  connector IDs are separate spaces.
- An ID renders when exactly one child claims it, or when exactly one of its claimants sits in the
  installation's namespace. Otherwise it renders for nobody, and every claimant is `Ready=False`
  with reason `DuplicateID`.
- Nobody outside the installation's namespace can win a contest, so a later claim never replaces
  an existing registration — it can only contest it. A client or connector in the installation's
  namespace cannot be outbid at all.
- A confidential client whose Secret is missing keeps claiming the ID it last resolved, recorded
  in its own `status.clientID` by the installation reconciler; a gap in a Secret frees nothing
  ([H-20](#h-20) is the case without a record). Whoever may write that status gains nothing by
  forging it: it counts only while the Secret does not resolve, and the Secret can name any ID
  already.
- While a `DexLocalConnector` renders, the connector ID `local` belongs to Dex's password database,
  which Dex appends after the configured connectors (`cmd/dex/serve.go`, dex v2.44.0): a connector
  with that ID is `DuplicateID` in every namespace.

The outcome depends on the specs, the Secrets and each client's own `status.clientID` — not on
age, render order or earlier renders. The installation lists every child it left out in
`status.rejectedChildren`, with kind, namespace, name, claimed ID and reason, and sets
`ChildrenRejected=True`. The message a rejected child shows its owner never names another
namespace ([H-24](#h-24)).

Env var keys cannot be used to displace another child either: they are assigned across storage,
connectors and clients in one priority order, and a child whose plain key is taken gets a
fallback key ([secret-flow.md](secret-flow.md)).

## A failing child stays its owner's problem

A child whose build fails — a missing Secret or key, an empty client ID — is left out and
reported as `BuildFailed`, in every namespace, and the render goes on for everybody else. Only a
failing storage credential, which belongs to the installation, still fails the render. One
exception keeps Dex startable: when connectors were collected and none of them renders and no
`DexLocalConnector` does, the render fails and the last written config stays, because Dex does not
start without a connector (`server: no connectors specified`, dex v2.44.0) ([H-22](#h-22)).

## What the allowlists defend against

A tenant in an arbitrary namespace registering an OAuth2 client or an identity-providing
connector with the company SSO. A rogue client with an attacker-controlled redirect URI would
receive authorization codes for real users; a rogue connector could mint identities. With the
allowlists, only namespaces the platform explicitly trusts can do either, and trusting a
namespace with connectors is a separate, explicit decision. Inside the admitted namespaces, the
contest rule keeps one registration from replacing another.

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

<a id="h-18"></a>
### H-18 — A co-tenant can take a tenant client out of the config

Live wherever `allowedNamespaces` admits more than one namespace besides the installation's. Any
principal who may create a `DexStaticClient` in an admitted namespace can claim the ID of a client
in another admitted namespace — deliberately, or by accident when two teams deploy one chart with
its default client ID — and both leave the config until one is removed or renamed. The outage is
loud and attributable: both clients are `Ready=False` with the same `DuplicateID` message, and the
installation lists both with namespace, name and ID. Nobody gains tokens. The same holds for
connectors among the namespaces admitted to connectors. Mitigation: keep security-relevant clients
in the installation's namespace — they always win — or narrow `allowedNamespaces`.

<a id="h-19"></a>
### H-19 — An ID can be squatted before its client exists

Live under the same condition as [H-18](#h-18). A tenant registers an ID before the legitimate
client exists — an application not deployed yet. When the legitimate client appears in another
tenant namespace, both are rejected: the squatter holds nothing from then on, and the legitimate
client does not render until the squatter is removed. A legitimate client in the installation's
namespace simply wins. Mitigation: as for H-18.

<a id="h-20"></a>
### H-20 — A confidential client that never resolved has no claim while its Secret is missing

Dormant unless all of these meet: the client has no `status.clientID` yet (a new object, the
first render after upgrading to the claim rules, a restore without status), its Secret is
missing, and another namespace claims its ID. That claimant renders until the Secret resolves;
then a client in the installation's namespace takes the ID back, a tenant client contests it. The
first render after the upgrade closes the gap for every client whose Secret is present.
Mitigation: create the Secret before the `DexStaticClient`.

<a id="h-21"></a>
### H-21 — A failing child in the installation's namespace is dropped

Live by design. A client or connector in the installation's namespace whose Secret is missing
leaves the config on the next render; every login through it fails until the Secret is back, and
with `rolloutRestart` Dex restarts twice around the gap. Accepted in exchange for an installation
that one failing child can no longer freeze. Mitigation: watch `ChildrenRejected` on the
installation.

<a id="h-22"></a>
### H-22 — The connector guard can be triggered by contest

Live when `allowedConnectorNamespaces` admits tenant namespaces and no connector lives in the
installation's namespace. A tenant in an admitted namespace claims the ID of every connector, none
renders, and the installation stays on its last config with `Ready=False` until the claims are
removed. Mitigation: keep at least one connector — or a `DexLocalConnector` — in the
installation's namespace.

<a id="h-23"></a>
### H-23 — Enough children push the config Secret past the Secret size limit

Live wherever tenants may create clients without limit. Every rendered child grows the config
Secret, every rejected child and every dropped peer the installation's status. Past the 1 MiB
limit of a Secret the write fails and the installation stays on its last config. The operator
cannot prevent it. Mitigation: a `ResourceQuota` with `count/dexstaticclients.dex.gtrfc.com` —
and the connector kinds where admitted — in every tenant namespace.

<a id="h-24"></a>
### H-24 — The installation's status names the namespaces of rejected children

Live by design. A rejected child's own message names no other namespace, but
`status.rejectedChildren` and `status.droppedTrustedPeers` list namespace, name and ID of every
child concerned, readable by whoever can read the `DexInstallation`. Client IDs are not secret —
they appear in every authorization URL — namespace names may be. Mitigation: grant `get` on
`dexinstallations` only to whoever may see the tenants of the installation.

<a id="h-25"></a>
### H-25 — A static client shadows a gRPC-created client of the same ID

Live when Dex's gRPC API creates clients. Dex looks static clients up before its storage
(`GetClient`, `storage/static.go`, dex v2.44.0), and the operator cannot see clients created
through the API, so a `DexStaticClient` with the ID of such a client replaces it for every flow.
Mitigation: do not mix gRPC-created and static clients in one installation, or keep the gRPC API
closed ([clients.md](clients.md)).
