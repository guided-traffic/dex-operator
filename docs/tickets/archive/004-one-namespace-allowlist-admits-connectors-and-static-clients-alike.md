---
id: T4
title: one namespace allowlist admits connectors and static clients alike
state: done
severity: high
security: boundary
threat: a principal who may create connectors in a namespace admitted only for static-client registration adds an identity source whose identities every client of the installation accepts
urgency: next         # rule 3: severity high, trigger live wherever a tenant namespace is admitted
effort: M
blocked-by:
filed-from: the analysis of which child kinds a tenant namespace should contribute
opened: 2026-10-06
decided: 2026-10-06
done: 2026-10-06
shipped: 2.3.0 — allowedConnectorNamespaces governs connectors (omitted = the installation's own namespace), allowedNamespaces static clients only; child reconcilers watch DexInstallation
---

# Separate Namespace Allowlist for Connectors

Add `DexInstallation.spec.allowedConnectorNamespaces`, a namespace allowlist
that applies **only to connectors**. When it is not set, connectors are
admitted only from the DexInstallation's own namespace.
`spec.allowedNamespaces` keeps its semantics, but from now on applies **only
to static clients**. No new operator RBAC permission.

## 1. Motivation

Today one list decides for every child kind at once
([api/v1/dexinstallation_types.go:66-70](../../../api/v1/dexinstallation_types.go#L66-L70)).
The two categories carry different trust:

- A **connector** adds an identity source. Every client of the installation
  accepts the identities it produces (§5.1).
- A **static client** registers a relying party: its redirect URIs, its
  `trustedPeers`, and with `cors: true` entries in the installation-wide
  `web.allowedOrigins` (§5.2).

Connectors should, as a rule, not come from tenant namespaces. Client
registration from app namespaces is the normal case. The current model cannot
express this: admitting a namespace for client registration also admits it
for connectors.

## 2. Current state (verified in this repo)

- **Field:** `AllowedNamespaces []string`, `omitempty`, no CEL rule. An empty
  list denies all, `"*"` allows all, any other entry is a literal namespace
  name ([internal/controller/namespace.go](../../../internal/controller/namespace.go)).
- **Enforcement point 1, build time:** `filterItems` in
  [internal/controller/collect.go](../../../internal/controller/collect.go) filters
  every listed child before rendering. `collectConnectors` (`allowed:
  installation.Spec.AllowedNamespaces`) and `collectStaticClients` both pass
  the **same** list. `filterItems` also sorts deterministically; that sort must
  stay (see its doc comment: without it an endless rollout loop happens).
- **Enforcement point 2, child status:** `reconcileChild` in
  [internal/controller/child_reconciler.go](../../../internal/controller/child_reconciler.go)
  sets `Ready=False` with `namespace %q is not in DexInstallation %s/%s
  allowedNamespaces`.
- **Stale child status (existing gap):** the child reconciler watches only its
  own kind (`For(T(&zero)).Complete(r)`), not `DexInstallation`. When a
  namespace is removed from the allowlist, the build-time filter drops the
  child immediately, but the child keeps `Ready=True` until its own next event.
- **RBAC:** the operator has no permission on `namespaces`. This ticket keeps
  it that way. The namespace of a child is on the child object, the namespace
  of the installation on the installation object. No lookup is needed.
- **Layout in the docs:** the operator runs in `dex-operator-system`; Dex, the
  `DexInstallation` and the connectors live in `dex`
  ([README.md](../../../README.md), TL;DR and Keycloak example).

### Categories

Verified: exactly two kinds of resources reference a `DexInstallation`
(the `registerIndexers` list in
[internal/controller/dexinstallation_controller.go](../../../internal/controller/dexinstallation_controller.go)):

| Category | Kinds | Governed by |
|---|---|---|
| Connectors | `DexLDAPConnector`, `DexGitHubConnector`, `DexSAMLConnector`, `DexGitLabConnector`, `DexOIDCConnector`, `DexOAuth2Connector`, `DexGoogleConnector`, `DexLinkedInConnector`, `DexMicrosoftConnector`, `DexAuthProxyConnector`, `DexBitbucketConnector`, `DexLocalConnector`, `DexOpenShiftConnector`, `DexAtlassianCrowdConnector`, `DexGiteaConnector`, `DexKeystoneConnector` | `allowedConnectorNamespaces` (new) |
| Static clients | `DexStaticClient` | `allowedNamespaces` (existing) |

There is no third CRD category. Finer splits (per connector kind, static
clients with `cors: true`) were analysed in §5 and are not part of this
ticket.

## 3. Design

### 3.1 API

```yaml
apiVersion: dex.gtrfc.com/v1
kind: DexInstallation
metadata:
  name: main
  namespace: dex
spec:
  # ...
  # Static clients only. Empty/omitted = deny all.
  allowedNamespaces: ["*"]                      # example
  # Connectors only. Omitted = only this installation's namespace ("dex").
  # Once set, the list is exhaustive: list "dex" too if connectors live there.
  allowedConnectorNamespaces: [dex, platform-idp]   # example
```

Change in `api/v1/dexinstallation_types.go`:

```go
	// AllowedNamespaces is a list of namespaces from which DexStaticClients
	// can reference this installation. Use ["*"] to allow all namespaces.
	// An empty or omitted list denies all. Connectors are governed by
	// AllowedConnectorNamespaces.
	// +optional
	AllowedNamespaces []string `json:"allowedNamespaces,omitempty"`

	// AllowedConnectorNamespaces is a list of namespaces from which
	// connectors can reference this installation. When omitted, only the
	// installation's own namespace is allowed. When set, the list is
	// exhaustive: the installation's own namespace is allowed only if it is
	// listed. Use ["*"] to allow all namespaces.
	// +optional
	// +listType=set
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:XValidation:rule="!self.exists(n, n == '*') || self.size() == 1",message="\"*\" must be the only entry"
	AllowedConnectorNamespaces []string `json:"allowedConnectorNamespaces,omitempty"`
```

Verify the CEL rule and `MinItems` against the envtest apiserver.

### 3.2 Semantics

Connectors (`allowedConnectorNamespaces`):

| Spec | Admitted namespaces |
|---|---|
| field omitted | the DexInstallation's own namespace only |
| `["*"]` | all |
| `[a, b]` | exactly `a` and `b`; the own namespace only if listed |
| `[]` | rejected at admission (`MinItems=1`) |

`MinItems=1` exists because an empty list would be a third state that cannot
survive a round trip: with `omitempty`, any Go client that writes the object
back drops `[]`, and "deny all" would silently turn into "own namespace".
"No connectors at all" is therefore not expressible. It is not needed: an
installation without connectors simply has none.

Static clients (`allowedNamespaces`): unchanged. Empty or omitted denies all,
`"*"` admits all, otherwise exact names.

Both fields: exact names only, no patterns (see §4.2).

### 3.3 Compatibility

Additive within `dex.gtrfc.com/v1`. No conversion webhook, no API version
bump, released as `feat` (minor).

Behavior change on upgrade, which must be stated in the release notes and in
the README: `allowedNamespaces` no longer admits connectors. An installation
whose connectors live in a namespace other than the installation's own loses
them on upgrade, until `allowedConnectorNamespaces` lists that namespace. The
change goes in the safe direction (less is admitted, not more). The affected
connectors get `Ready=False` with a message that names the new field.

Installations whose connectors live in the installation's own namespace, as
in all README examples, render byte-identical config after the upgrade: no
config diff, no dex rollout.

### 3.4 Controller changes

1. **Admission helpers** in
   [internal/controller/namespace.go](../../../internal/controller/namespace.go):
   keep `isNamespaceAllowed(ns, list)` for the list semantics, and add a helper
   that returns the effective connector list:
   `connectorNamespaces(inst) = inst.Spec.AllowedConnectorNamespaces`, or
   `[inst.Namespace]` when the field is nil. Pure functions, no client.
2. **Build-time filter:** `collectConnectors` passes
   `connectorNamespaces(installation)` instead of `AllowedNamespaces`.
   `collectStaticClients` keeps `AllowedNamespaces`. `filterItems` and
   `sortByNamespaceName` stay unchanged.
3. **Child status:** `reconcileChild` picks the list by its own kind:
   `DexStaticClient` uses `AllowedNamespaces`, every connector kind uses
   `connectorNamespaces(installation)`. Derive the category in the generic
   reconciler or via a method on `ChildObject`
   ([internal/controller/interfaces.go](../../../internal/controller/interfaces.go)),
   whichever keeps the 16 connector types free of boilerplate. Messages:
   - static client: `namespace %q is not in DexInstallation %s/%s allowedNamespaces`
     (unchanged)
   - connector: `namespace %q is not in DexInstallation %s/%s allowedConnectorNamespaces`,
     with the suffix ` (omitted: only %q is allowed)` when the field is not set,
     so an operator who upgraded sees the reason.
4. **Child status re-evaluation:** the child reconciler additionally watches
   `DexInstallation` and enqueues the children referencing it, via the
   existing `InstallationRefIndexField`. This closes the stale `Ready=True` gap
   from §2. The operator already has `watch` on `dexinstallations`, so no new
   permission.

Keep cyclomatic complexity under 15 per function (`make cyclo`).

## 4. Security considerations

### 4.1 Secure default for connectors

Without configuration, only the installation's own namespace can add
connectors. That namespace already holds the rendered config Secret and the
env Secret with every client secret, so it is the most trusted namespace of
the installation. Admitting tenant namespaces to connectors becomes an
explicit entry in `allowedConnectorNamespaces`.

Namespace names are immutable, so the decision stays entirely with whoever can
write the `DexInstallation`. Edge case (unchanged from today): a listed name
that does not exist yet admits whoever creates that namespace first. In
clusters where tenants can create namespaces (e.g. Capsule tenant owners),
list only names that exist.

### 4.2 Why no patterns and no label selectors

- **Label selectors** would need cluster-wide `get/list/watch` on
  `namespaces`, and they move the admission decision to whoever can label a
  namespace. Rejected.
- **Name patterns** (`team-*`) would need no RBAC, but they move the decision
  to whoever can create namespaces with a matching name. Not part of this
  ticket. A later extension would be a separate field, not new syntax inside
  the existing lists, so the meaning of an existing list never changes.

### 4.3 Revocation

When a namespace loses admission, the next render drops its children, and
the rollout restart (if enabled) applies it. With `rolloutRestart.enabled:
false`, dex keeps the old config until it restarts. Unchanged from today.

## 5. Analysis: finer categories (not implemented)

Sources: this repo and dex **v2.44.0** source (`connector/authproxy/authproxy.go`,
`server/server.go`, `cmd/dex/serve.go`, `storage/storage.go`).

### 5.1 Per connector kind

#### 5.1.1 Every connector can forge identities

The question for a namespace policy is: what can a **malicious tenant** do with
one connector of this kind? For most kinds, the answer is the same: the tenant
controls the upstream, or picks it. A self-hosted upstream (OIDC, OAuth2, SAML,
LDAP, Keystone, AtlassianCrowd, Gitea, OpenShift, self-hosted GitLab) returns
whatever `email`, `email_verified` and `groups` the tenant wants. Dex passes
these claims through. Dex encodes the connector ID only into `sub`, so clients
that authorize by `email` or `groups` (most of them) accept the forged
identity. SaaS-only kinds (Google, LinkedIn, Microsoft, Bitbucket, GitHub)
limit what the tenant can claim, but the tenant still chooses which upstream
accounts and groups are trusted.

Verified: dex v2.44.0 has no per-client connector restriction. `storage.Client`
has no `allowedConnectors` field. So any connector reaches every client.

Consequence: admitting a namespace to connectors means trusting it with
identity issuance for the whole installation. That is the reason for the
separate allowlist and its restrictive default. A per-kind split does not
reduce this risk for OIDC and the like.

#### 5.1.2 `DexAuthProxyConnector`: exploitable by third parties

Purpose: a production integration, not a development aid. A front proxy
performs a login that dex does not support natively, and dex takes over the
result (dex docs: "used by proxies to implement login strategies not supported
by dex"; package comment: "e.g. mod_auth in Apache2"). The dex docs mark it
experimental and state it does not support refresh tokens. Dex's development
connectors are `mockCallback` and `mockPassword`; the operator offers no CRD
for them.

Verified in `connector/authproxy/authproxy.go`: `HandleCallback` takes the
user, email, user ID and groups **directly from request headers** and returns
`EmailVerified: true`. `staticGroups` are added to every identity. The login
URL is `/callback/<connector-id>`.

Verified in `server/server.go`: dex strips `X-Remote-*` headers only on the
plain `/callback` route. `/callback/{connector}`, the authproxy route, does
**not** strip anything. Custom header names (`userHeader` etc.) are never
stripped. The dex docs require the proxy to remove these headers "for any URL
path".

Consequence: unless a proxy in front of dex authenticates `/callback/<id>`
and overwrites these headers, **anyone who can reach dex** logs in as any user
with any groups. The other connector kinds differ here: there, only the
connector author can abuse the connector. Here, an honest mistake by the
author is enough for an outsider. The protecting proxy is part of the ingress
in front of dex, so it is owned by whoever runs the dex installation, not by a
tenant.

#### 5.1.3 `DexLocalConnector`: an installation-wide switch, no identity forging

Verified in [internal/builder/connectors.go](../../../internal/builder/connectors.go)
and `cmd/dex/serve.go`: the CR renders no connector entry. It only sets
`enablePasswordDB: true`. Dex then registers a fixed connector with ID `local`
and name `Email`. The CR's `id` and `displayName` have no effect. The operator
renders no `staticPasswords`.

Consequence: a tenant with a `DexLocalConnector` cannot create users. It turns
on email/password login for every client of the installation. The risk lies in
who can write password entries:

- dex storage (with `kubernetes` storage: the `passwords.dex.coreos.com`
  objects in the dex namespace)
- the dex gRPC API. Verified in `cmd/dex/serve.go`: client certificates are
  required only when `grpc.tlsClientCA` is set. Without it, anyone who reaches
  the gRPC port can call `CreatePassword`, and with the password DB on, log in
  with that account.

So `DexLocalConnector` is less dangerous than an OIDC connector from the same
tenant. It does not need its own rule.

Side finding: [README.md](../../../README.md) (DexLocalConnector example) shows
`id: local # default: metadata.name`. The value has no effect, because dex
hard-codes the ID. To be fixed separately.

#### 5.1.4 Decision

No per-kind rule. With the restrictive default (§3.2), all connector kinds,
AuthProxy included, come from the installation's namespace unless a namespace
is admitted explicitly. A per-kind rule would only matter when tenant
namespaces are admitted to connectors, and that is the exception the
allowlist makes visible. If it ever becomes the rule, the smallest extension is
a fixed rule "`DexAuthProxyConnector` only from the installation's own
namespace", because §5.1.2 is the one case where an honest mistake opens the
installation to outsiders.

### 5.2 Static clients with `cors: true`

#### 5.2.1 What an added origin enables

Verified in `server/server.go`: dex wraps exactly six routes with CORS:
`/.well-known/openid-configuration`, `/`, `/token`, `/keys`, `/userinfo`,
`/token/introspect`. The gorilla `handlers.CORS` call sets only
`AllowedOrigins` and `AllowedHeaders`, **not** `AllowCredentials`. `/auth`,
`/callback/*`, `/approval` and the login pages are not CORS-wrapped.

An extra allowed origin lets JavaScript on that origin **read responses** of
these six routes. Each of them needs either no credential (discovery, keys,
`/`: public data) or a credential the script must already hold: an
authorization code plus PKCE verifier or client secret for `/token`, an access
token for `/userinfo` and `/token/introspect`. Dex uses no cookies on them.
Everything the origin can do from a browser, its server can already do with
plain HTTP. CORS does not authenticate anything.

The only extra power is the network position: a victim's browser can read
responses of a dex that is reachable only internally. Without a credential that
yields only public metadata.

#### 5.2.2 The installation-wide effect

Dex has no per-client CORS (`storage.Client` has no origins field), so an
origin derived from client A also applies to flows of client B. The origin
still needs B's code and verifier or secret to do anything. B's code goes to
B's redirect URI, not to the origin of A.

#### 5.2.3 Relation to redirect URIs

An origin is derived only from the client's own https `redirectURIs`. Whoever
owns such a redirect URI already receives authorization codes for users who
log in to that client. That is a much bigger power than CORS, and it is gated
by `allowedNamespaces`. A separate allowlist for `cors: true` would gate the
smaller power behind a second list while the bigger one stays with the first.

#### 5.2.4 Decision

No separate category. `allowedNamespaces` is the effective gate. If a platform
still wants to forbid derived origins for one installation, the smallest
control is an installation-level boolean (e.g. `web.deriveClientOrigins`), not
a namespace list. Not built without a concrete need.

Not verified: the behavior of `gorilla/handlers` for requests from origins not
in the list (whether disallowed preflights get 403 or an empty 200). This does
not change the conclusion, because no route depends on CORS for
authentication.

## 6. Implementation checklist

1. Field + markers (§3.1) in `api/v1/dexinstallation_types.go`, updated doc
   comment on `AllowedNamespaces`. `make generate-all` (regenerates CRDs,
   DeepCopy, syncs Helm CRDs).
2. `connectorNamespaces` helper (§3.4.1).
3. `collectConnectors` switch to the connector list (§3.4.2).
4. Category-aware `reconcileChild` + messages (§3.4.3).
5. DexInstallation watch on the child reconcilers (§3.4.4).
6. Tests (all via Makefile targets):
   - **Unit** (`make test-unit`), extending
     [internal/controller/namespace_test.go](../../../internal/controller/namespace_test.go):
     field omitted → own namespace only; `["*"]` → all; list → exactly the
     list; list without own namespace → own namespace denied.
   - **Determinism:** existing
     [internal/controller/collect_determinism_test.go](../../../internal/controller/collect_determinism_test.go)
     must stay green; add a case with connectors admitted via the default.
   - **Render equality:** an installation with connectors in its own
     namespace and only `allowedNamespaces` set renders byte-identical config
     before and after.
   - **Integration / envtest** (`make test-integration`): CEL and `MinItems`
     (`[]` rejected, `["*", "a"]` rejected, `["*"]` and `["a","b"]`
     accepted); a connector and a static client in the same tenant namespace
     with `allowedNamespaces: [tenant]` and the connector field omitted →
     client rendered, connector not rendered and `Ready=False` with the
     "omitted" message; editing `allowedConnectorNamespaces` → connector status
     follows (stale-status regression for §2). Adjust
     [test/integration/dexinstallation_test.go](../../../test/integration/dexinstallation_test.go)
     (`TestIntegration_AllowedNamespacesFilter`) and
     [test/integration/connector_test.go](../../../test/integration/connector_test.go)
     to the split.
   - **E2E:** adjust `TestE2E_NamespaceIsolation`
     ([test/e2e/dexinstallation_e2e_test.go](../../../test/e2e/dexinstallation_e2e_test.go)):
     it admits a connector from a foreign namespace via `allowedNamespaces`,
     which must now go through `allowedConnectorNamespaces`. Add one case:
     connector from a foreign namespace with the field omitted is not
     rendered.
7. `make lint`, `make cyclo`, `make test`.
8. Docs (same change):
   - [README.md](../../../README.md): DexInstallation reference block
     (`allowedNamespaces` around line 406): both fields, each with its
     category, default and a security note. Update the `installationRef`
     paragraph (around line 276). Add an upgrade note for §3.3.
   - `SECURITY_ARCHITECTURE.md`: section
     "Namespace Isolation" (two lists, connector default, §4.1 edge case);
     the connector trust statement from §5.1.1; the AuthProxy item in the
     hardening checklist extended with the `/callback/<id>` detail from
     §5.1.2; the password-DB/gRPC dependency from §5.1.3; checklist item
     "Prefer explicit `allowedNamespaces`" split per field.
   - `DEVELOPER.md`: reconcile flow (new watch).
   - [CLAUDE.md](../../../CLAUDE.md): DexInstallation section.
9. Commit as `feat(api): ...` with the behavior change of §3.3 in the commit
   body, so it reaches the release notes.

## 7. Out of scope

- Per-kind connector rules (§5.1.4).
- A separate allowlist or switch for `cors: true` (§5.2.4).
- Name patterns and label selectors (§4.2).
- The README `id: local` inaccuracy (§5.1.3).
- An admission webhook. The repo has none and CEL covers the static rules.
