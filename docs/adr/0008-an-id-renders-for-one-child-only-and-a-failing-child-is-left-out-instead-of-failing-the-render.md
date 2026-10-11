# ADR 0008: An ID Renders for One Child Only — the Sole Claimant or the Sole Claimant in the Installation's Namespace — and a Failing Child Is Left Out Instead of Failing the Render

## Status

Accepted. Date: 2026-10-07.

**Implemented**, in the release after 2.3.0.

## Context

One `DexInstallation` renders static clients and connectors from every namespace its allowlists
admit into one `config.yaml` ([ADR 0006](0006-two-namespace-allowlists-one-per-child-category-exact-names-only.md)).
Dex keys both by ID: `storage.WithStaticClients` and `storage.WithStaticConnectors` build a map by
ID, so of two entries with one ID the last one is served, and `cmd/dex/serve.go` rejects neither
(dex v2.44.0, `storage/static.go`). Client IDs and connector IDs are mutable — `spec.clientID`,
every connector's `spec.id`, and the `client-id` in the Secret behind `secretRef`, which is the
normal path of a confidential client — so neither `creationTimestamp` nor the order of the render
says who registered an ID first. The allowlists decide which namespaces may register, not which
IDs a registration may use.

Separately, until this decision one child whose Secret could not be resolved, or two confidential
clients whose env var names collided, failed the whole build, and the installation kept its last
config for every relying party until that one child was fixed.

## Decision

**D1 — Every child claims an ID, per category.** A static client claims its client ID —
`spec.clientID`, or the `client-id` from its Secret — and a connector its effective ID
(`spec.id`, else `metadata.name`), across all sixteen kinds. Client IDs and connector IDs are
separate ID spaces, as in Dex. Only children that pass the namespace allowlists claim
([internal/builder/clients.go](../../internal/builder/clients.go) `clientUnits`,
[connectors.go](../../internal/builder/connectors.go) `connectorUnits`).

**D2 — The contest rule.** Per ID ([internal/builder/render.go](../../internal/builder/render.go)
`contest`):

1. one claimant: it renders;
2. several claimants, exactly one of them in the installation's namespace: that one renders;
3. otherwise the ID renders for nobody.

Every claimant that does not render is reported as `DuplicateID`. The outcome depends on the
specs, the Secrets and each client's own `status.clientID` (D4) only — not on age, render order or
earlier renders. A later claim never replaces an existing registration; it can only contest it,
and only the installation's namespace wins a contest. Two claimants in one namespace, the
installation's included, are both left out: both ends are one RBAC domain, and a silent pick
would hide the mistake.

**D3 — `local` belongs to Dex while a `DexLocalConnector` renders.** Dex appends its password
database as a connector with ID `local` after the configured ones (`cmd/dex/serve.go`, dex
v2.44.0). While a `DexLocalConnector` renders, a connector whose effective ID is `local` is
`DuplicateID` in any namespace (`reserveLocal`).

**D4 — A confidential client keeps claiming its last resolved ID while its Secret is missing.**
After every render the installation reconciler patches `status.clientID` on every collected
client whose resolved ID differs
([internal/controller/dexinstallation_controller.go](../../internal/controller/dexinstallation_controller.go)
`recordClientIDs`). A client whose Secret does not resolve claims that ID, renders nothing and is
`BuildFailed`; its claim still counts in the contest, so a gap in a Secret frees nothing. The
field counts only while the Secret does not resolve.

**D5 — A failing child is left out and reported, in every namespace.** A child whose build fails —
a missing Secret or key, an empty client ID, both env var keys taken (ADR 0002 D3) — is skipped
as `BuildFailed` and the render continues. A losing claimant whose own build also fails is
reported as `BuildFailed`, because its own error is what its owner can act on. Only a failing
storage credential still fails the render: storage belongs to the installation, not to a child.

**D6 — The connector guard.** When at least one connector was collected, none renders and no
`DexLocalConnector` renders, the render fails with `ErrNoConnector` and the last written config
stays; the installation is `Ready=False` and still reports its rejections. Dex does not start
without a connector (`server: no connectors specified`, `server/server.go`, dex v2.44.0), so
skipping would turn broken identity sources into a Dex that is down for every relying party at its
next restart. A contested connector counts as not rendering.

**D7 — Rejections are reported on both sides.** The installation carries
`status.rejectedChildren` (kind, namespace, name, claimed ID, reason, message, sorted) and the
condition `ChildrenRejected`; the child carries `Ready=False` with reason `DuplicateID` or
`BuildFailed` and the same message. A `DuplicateID` message is identical under rules 2 and 3 and
names no namespace but the installation's. `Ready` of the installation keeps its meaning: the
config was rendered and written. CORS origins and the status counts come from rendered children
only.

## Consequences

- No child outside the installation's namespace can replace or inherit a client or connector
  registration of the same installation, in any sequence of creates, updates, deletes and
  restores. The installation's namespace cannot be outbid.
- Any admitted principal can take a tenant client out of the config by claiming its ID — loudly:
  both claimants are `Ready=False` with the same message and the installation names both. Nobody
  gains the ID. Keeping security-relevant clients in the installation's namespace makes them
  immune.
- Moving a client between namespaces needs no order: created first, the new object contests the
  old one; deleted first, the old one leaves a gap. Either way one render goes without it.
- A client or connector in the installation's namespace whose Secret is missing leaves the config
  on the next render instead of freezing it; with `rolloutRestart`, Dex restarts twice around the
  gap.
- Every collected client costs one status write on the first render after the upgrade, and one
  whenever its ID changes.
- Child reconcilers follow `status.rejectedChildren` of the installation, so its watch passes
  that change ([ADR 0006](0006-two-namespace-allowlists-one-per-child-category-exact-names-only.md) D6).

## Alternatives Considered

- **A recorded holder in the installation status** that keeps its ID against every later claim.
  Protects a running tenant client against a co-tenant, but the render then depends on state that
  a restore without status, a failed status write or a delete-and-recreate loop loses, each time
  handing the ID to the oldest claimant again; it also needs a four-tier winner rule and an order
  for moving a client. Lost: it weakens the property this record exists for.
- **First in namespace/name order wins.** Deterministic, but a namespace that sorts earlier takes
  over an existing ID. Lost.
- **Drop every claimant, without the installation-namespace rule.** Lets any admitted principal
  knock out the installation's own clients. Lost.
- **The older object wins.** IDs are mutable, so an old object that changes its ID onto a held one
  outranks the holder. Lost.
- **Immutable IDs** (`self == oldSelf` on `spec.clientID`, `spec.id`). Leaves the client ID from
  the Secret mutable. Lost.
- **Namespace-prefixed tenant IDs.** No collisions between tenants, but changes the audience
  applications and resource servers see. Lost.
- **Admission-time uniqueness.** Needs other objects, which CEL cannot see; there is no webhook.
  Lost.
- **No `status.clientID`.** Leaves the gap of a missing Secret open for every confidential client
  on every render. Lost.
- **A failing child in the installation's namespace stays fatal.** Keeps platform clients through
  a transient error, but keeps the freeze. Lost to D5.
- **Keep the last good entry of a failing child** from the written Secrets. Makes the written
  config a second source of state, and a credential whose Secret was removed stays live. Lost.
- **No connector guard.** Simpler; an installation whose connectors all fail renders a config Dex
  refuses at its next restart. Lost.

## Residual risks

The gaps the rule leaves are kept with their mechanism in
[docs/security/tenancy.md](../security/tenancy.md): a co-tenant taking a tenant client out of the
config, ID squatting before the legitimate client exists, a client that never resolved having no
claim while its Secret is missing, a failing platform child being dropped, the connector guard
triggered by contest, the size of the config Secret and the installation status, and static
clients shadowing gRPC-created clients of the same ID.

- Not verified: Dex's behaviour when the config Secret exceeds the 1 MiB Secret limit beyond the
  write failing at the API server.

## References

- [internal/builder/render.go](../../internal/builder/render.go),
  [clients.go](../../internal/builder/clients.go),
  [connectors.go](../../internal/builder/connectors.go),
  [connectors_oauth.go](../../internal/builder/connectors_oauth.go),
  [builder.go](../../internal/builder/builder.go) `Build`
- [internal/controller/dexinstallation_controller.go](../../internal/controller/dexinstallation_controller.go),
  [status.go](../../internal/controller/status.go) `setChildReport`,
  [child_reconciler.go](../../internal/controller/child_reconciler.go) `findRejectedChild`
- [internal/builder/contest_test.go](../../internal/builder/contest_test.go),
  [internal/controller/claims_test.go](../../internal/controller/claims_test.go),
  [test/integration/claims_test.go](../../test/integration/claims_test.go),
  [test/e2e/claims_e2e_test.go](../../test/e2e/claims_e2e_test.go)
- [ADR 0002](0002-custom-resources-carry-no-secret-values-credentials-reach-dex-through-an-env-secret.md) D3,
  [ADR 0009](0009-a-client-trusts-only-peers-held-in-its-own-namespace.md)
