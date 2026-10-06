# Security Architecture

How the dex-operator handles credentials, isolates tenants, and what its trust model assumes. Written for platform operators who run the operator and for security reviewers who audit it. Code references point at the enforcing implementation.

## System Roles and Trust Boundaries

| Role | Trust level |
|---|---|
| **dex-operator** | High privilege: cluster-wide read of all 18 CRDs, read/write of Secrets, patch of Deployments. Compromise of the operator ≈ compromise of the SSO configuration. |
| **Dex namespace** (e.g. `dex`) | Sink for all rendered material. Anyone who can read Secrets here can read the full config **and every client secret**. |
| **App namespaces** | Semi-trusted tenants. They may contribute static clients *only if* listed in `allowedNamespaces`, and connectors *only if* listed in `allowedConnectorNamespaces` (by default no tenant namespace is). They expose credentials to the operator only by referencing their own Secrets. |
| **Kubernetes API server** | Trusted enforcement point: RBAC, CEL validation, etcd storage. |

```
app namespace (tenant)                dex namespace
┌───────────────────────┐            ┌─────────────────────────────────────┐
│ DexStaticClient       │            │ DexInstallation                     │
│ Dex*Connector         │──ref──┐    │  allowedNamespaces          ✓ gate  │
│ Secret (credentials)  │       │    │    (static clients)                 │
└───────────────────────┘       │    │  allowedConnectorNamespaces ✓ gate  │
                                │    │    (connectors; default: dex only)  │
                                ▼    │ Dex*Connector (default home)        │
                          dex-operator ──► config Secret (config.yaml)     │
                                     │    └─► env Secret (env vars)        │
                                     │         │ envFrom                   │
                                     │         ▼                           │
                                     │ Dex Deployment                      │
                                     └─────────────────────────────────────┘
```

## Namespace Isolation (`allowedNamespaces`, `allowedConnectorNamespaces`)

The central multi-tenancy control. Each `DexInstallation` carries two allowlists, one per child category, because the two categories carry different trust:

| Field | Governs | Omitted / empty | `"*"` | Other entries |
|---|---|---|---|---|
| `allowedNamespaces` | `DexStaticClient` | deny all (fail-closed; the own namespace is not implied) | every namespace | literal namespace names |
| `allowedConnectorNamespaces` | all 16 connector kinds | only the installation's own namespace | every namespace (must be the only entry) | literal names; exhaustive, so list the own namespace too if connectors live there |

`allowedConnectorNamespaces: []` is rejected at admission (`MinItems=1`, [api/v1/dexinstallation_types.go](api/v1/dexinstallation_types.go)). With `omitempty`, any Go client that writes the object back would drop `[]` and silently turn "deny all" into "own namespace"; the schema makes that third state unrepresentable instead.

**Why connectors get their own, restrictive list.** A static client registers one relying party. A connector adds an identity source, and every client of the installation accepts the identities it produces: dex has no per-client connector restriction (`storage.Client` has no such field, dex v2.44.0). For any connector whose upstream the tenant runs or picks (OIDC, OAuth2, SAML, LDAP, Keystone, AtlassianCrowd, Gitea, OpenShift, self-hosted GitLab), the upstream returns whatever `email`, `email_verified` and `groups` the tenant wants; dex encodes the connector ID only into `sub`, so clients that authorize by email or groups accept the forged identity. Admitting a namespace to `allowedConnectorNamespaces` therefore means trusting it with identity issuance for the whole installation. The default keeps connectors in the installation's own namespace, which already holds the config and env Secrets and is the most trusted namespace of the installation.

The allowlists are enforced twice, independently:

1. **At build time** — `collectStaticClients` filters through `allowedNamespaces`, `collectConnectors` through the effective connector list (`connectorNamespaces`), both via `filterItems` → `isNamespaceAllowed` ([internal/controller/collect.go](internal/controller/collect.go), [namespace.go](internal/controller/namespace.go)). A non-allowlisted resource can never influence the rendered config, regardless of what its own status claims.
2. **At child reconciliation** — the generic child reconciler marks resources from forbidden namespaces with a `Ready=False` condition that names the governing field; for a connector with `allowedConnectorNamespaces` omitted it adds `(omitted: only "<ns>" is allowed)` ([internal/controller/child_reconciler.go](internal/controller/child_reconciler.go)). The child reconcilers also watch `DexInstallation`, so the condition follows allowlist edits instead of staying stale until the child's own next event.

Both fields take exact names only. Label selectors would need cluster-wide `get/list/watch` on `namespaces` and move the admission decision to whoever can label a namespace; name patterns would move it to whoever can create a matching namespace. Namespace names are immutable, so the decision stays with whoever can write the `DexInstallation`. No lookup of `Namespace` objects is needed, and the operator has no permission on them.

**What the allowlists defend against:** a tenant in an arbitrary namespace registering an OAuth2 client or an identity-providing connector with your company SSO. A rogue client with an attacker-controlled redirect URI would receive authorization codes for real users; a rogue connector could mint identities. With the allowlists, only namespaces you explicitly trust can do either, and connector trust is a separate, explicit decision.

**What they do not defend against:**

- Principals who already have `create` rights on Dex CRDs *inside* an admitted namespace. That boundary is Kubernetes RBAC — treat `dexstaticclients`/`dex*connectors` create/update as privileged verbs and grant them accordingly.
- **Name squatting.** A listed name that does not exist yet admits whoever creates that namespace first. In clusters where tenants can create namespaces (e.g. Capsule tenant owners), list only names that exist.
- **Revocation latency without a restart.** Removing a namespace drops its children on the next render; with `rolloutRestart.enabled: false`, Dex keeps serving the old config until it restarts.

## Secret Flow

Design rule: **CRs never carry secret values.** Specs carry only `{name, key}` references to Secrets in the *same namespace* as the CR — a tenant can only expose material it can already create in its own namespace. The operator resolves references at build time ([makeSecretResolver](internal/controller/dexinstallation_controller.go)).

Resolved material takes one of three paths:

| Class | Handling | Examples |
|---|---|---|
| **Secret credentials** | Written to the **env Secret** under a deterministic name; the rendered config contains only the reference (`$VAR` or `secretEnv: VAR`), never the value | connector client secrets, LDAP bind password, Keystone password, storage passwords, static-client secrets |
| **Non-secret identifiers** | Embedded inline in `config.yaml` | client IDs, hostnames, base URLs |
| **File material** | Registered as a mount path in the config (`/etc/dex/certs/...`, `/etc/dex/secrets/...`) | SAML signing CA, TLS client cert/key, Google service-account JSON, storage TLS |

Why the env indirection, given that config and env Secrets sit in the same namespace?

- `config.yaml` is the artifact people `kubectl get`, attach to tickets, and diff in debugging sessions. Keeping credentials out of it prevents routine, accidental disclosure.
- Env var names are deterministic ([internal/builder/envvar.go](internal/builder/envvar.go)); static-client names that would collide after sanitization are rejected at build time ([internal/builder/clients.go](internal/builder/clients.go)) instead of silently overwriting each other's secret.
- Rotation changes only the env Secret; the config stays byte-identical, avoiding unnecessary Dex restarts.

One deliberate exception: the LDAP root CA is embedded base64-inline (`rootCAData`). A CA certificate is public material; inlining removes a file-mount dependency.

**Current gap — file mounts:** the builder reports required file projections as `MountedSecrets`, but the controller does not yet create volumes/volumeMounts on the Dex Deployment. Until it does, operators must mount those Secrets in their Dex Helm values. The config paths are deterministic, so this is a one-time setup per certificate — but a missing mount fails only at Dex runtime, not at reconcile time.

## Admission-Time Validation

There is no webhook (no cert management, no availability coupling). Everything is enforced by the API server itself:

- **CEL rules** on `DexStaticClient` guarantee that a client is either confidential (`secretRef`, `redirectURIs` required) or public (`public: true`, inline `clientID`) — never an ambiguous mix. Invalid objects are rejected at `kubectl apply`. See the truth table in the [README](README.md#dexstaticclient).
- **`DexInstallation.spec.allowedConnectorNamespaces`** is a set (`MinItems=1`, no duplicates) and a CEL rule requires `"*"` to be the only entry, so a list can never look restrictive while admitting everything.
- **Enums and formats** on security-relevant fields (storage types, SSL modes, log levels, URI formats) reduce the injection surface into the rendered YAML.

Validation that needs cross-object knowledge (Secret existence, env-key collisions) happens at build time and surfaces via status conditions and events rather than admission errors.

## Confidential vs. Public Clients

Confidential clients authenticate with a secret at the token endpoint; the redirect-URI allowlist is their second control. Public clients (`public: true`) have **no** secret — by definition their binaries run on user devices where any embedded secret is extractable. Their security rests on:

1. **PKCE** — binds the authorization code to the party that started the flow, defeating code interception.
2. **Redirect URI policy** — if `redirectURIs` is omitted, Dex accepts loopback addresses (`http://localhost:<port>`), the OOB URN and the device-flow callback. That is correct for CLIs/native apps and useless for servers.

Operational rule: model server-side applications as confidential, interactive tools as public. Marking a server app `public` silently removes client authentication from your token endpoint. A hybrid also exists (public + `secretRef`): PKCE *plus* a secret.

## Tenant-Registered CORS Origins (`cors: true`)

A `DexStaticClient` with `cors: true` contributes the origins of its own **https** `redirectURIs` to the installation's `web.allowedOrigins` ([internal/builder/builder.go](internal/builder/builder.go), `deriveCORSOrigins`). This is deliberately a boolean flag and not a free-form origin list: **it grants no authority beyond `redirectURIs`**, which is already the security-critical, RBAC-gated field. A tenant can only allowlist origins it already controls as redirect targets — and a tenant able to set an arbitrary redirect URI has a far stronger primitive than a CORS entry. Derivation can never produce `"*"`; that value remains reachable only through the platform operator's own `spec.web.allowedOrigins`.

What CORS does and does not do here: it is **defense in depth, not an authorization boundary**. Dex's token endpoint carries no ambient credentials (no cookies, no session), so a cross-origin request from an unlisted site gains nothing even if it were allowed — the authorization code is bound by PKCE. CORS keeps browser-side probing of the discovery/keys/token endpoints down and prevents an unrelated page from reading responses. Matching is **literal** (Dex wraps these handlers with `gorilla/handlers.AllowedOrigins`): no subdomain wildcards, so a compromised sibling host does not inherit access, and an entry with a wrong case or a redundant `:443` would simply never match — the operator normalizes both when deriving.

Residual exposure: the flag makes the installation's origin list depend on tenant resources, so a tenant in an allowed namespace can grow that list. The bound is its own `redirectURIs`; the control is the same one that already gates client registration — `allowedNamespaces` plus RBAC on `dexstaticclients`.

## Operator RBAC Footprint

Granted by the Helm chart's ClusterRole (markers in [internal/controller/rbac.go](internal/controller/rbac.go)):

| Resource | Verbs | Why |
|---|---|---|
| all `dex.gtrfc.com` CRDs + status | get/list/watch/update/patch | reconciliation and status reporting |
| `secrets` (cluster-wide) | get/list/watch/**create/update/patch** — **no delete** | read referenced credentials, write config/env Secrets |
| `deployments` (apps) | get/list/watch/**patch** | rollout restart annotation only |

Consequences to be aware of:

- Cluster-wide Secret read is the operator's most sensitive permission. It is inherent to the design (tenants reference Secrets in their own namespaces) — protect the operator's ServiceAccount accordingly and monitor its usage (audit logs on `secrets` access by the SA are cheap and high-signal).
- The operator never deletes Secrets. Stale generated Secrets (after renaming `configSecretName`) must be cleaned up manually — deliberate, to make the operator incapable of destroying credentials.
- The operator itself runs hardened by chart defaults: non-root, seccomp `RuntimeDefault`, all capabilities dropped, no privilege escalation ([deploy/helm/dex-operator/values.yaml](deploy/helm/dex-operator/values.yaml)).

## Rotation and Change Propagation

- Referenced Secrets are indexed and watched; a credential rotation (Secret update/create) re-renders config and env Secret automatically ([internal/controller/secret_watch.go](internal/controller/secret_watch.go)).
- Secret **deletion is deliberately not acted on**: during rotation, delete+recreate would otherwise produce a failing reconcile between the two events. The replacement Secret's create event triggers the actual update.
- With `rolloutRestart.enabled`, a *semantic* config change patches the Dex Deployment's pod-template annotation (`kubectl.kubernetes.io/restartedAt`), forcing a rolling restart so file-based config is re-read. Env-only changes do not restart Dex automatically today — a rotated client secret becomes active on the next Dex restart unless the config changed too.
- Determinism guards availability: children are sorted before rendering and the config Secret is compared as parsed YAML, not bytes ([internal/controller/collect.go](internal/controller/collect.go), [secret.go](internal/controller/secret.go)). Without this, list-order jitter would cause endless restart loops — a self-inflicted denial of service.

## Residual Risks & Hardening Checklist

- [ ] **Protect the dex namespace.** Everything sensitive converges there. Restrict Secret read via RBAC; consider etcd encryption at rest — the env Secret aggregates all client secrets.
- [ ] **Treat CRD verbs as privileged.** `create`/`update` on `dexstaticclients` in an admitted namespace equals the power to register SSO clients; on `dex*connectors` in an admitted namespace, the power to issue identities for every client. Scope RBAC per namespace/team.
- [ ] **Prefer explicit `allowedNamespaces`.** `"*"` shifts the static-client tenancy boundary onto CRD RBAC.
- [ ] **Keep `allowedConnectorNamespaces` omitted unless a tenant must run its own IdP.** The default admits only the installation's namespace. Each extra entry trusts that namespace with identity issuance for the whole installation (see [Namespace Isolation](#namespace-isolation-allowednamespaces-allowedconnectornamespaces)); `"*"` makes every namespace an identity source.
- [ ] **Audit `insecure*` flags.** Every connector option prefixed `insecure` (skip TLS verify, skip signature validation, skip email-verified, `insecureNoSSL`, `insecureCA`) removes a verification step and belongs in dev environments only. They are greppable in cluster: `kubectl get dex<type>connectors -A -o yaml | grep -n insecure`.
- [ ] **Scope upstream restrictions.** Connectors without `orgs`/`groups`/`hostedDomains`/`teams`/`tenant` filters accept *any* account of that provider. Filter at the connector, not only in the app.
- [ ] **AuthProxy connector:** only deploy behind a proxy that is the exclusive network path to Dex, authenticates `/callback/<connector-id>` and overwrites the identity headers — otherwise anyone who reaches Dex logs in as any user with any groups. Dex takes user, email, user ID and groups straight from request headers and returns `EmailVerified: true`. It strips `X-Remote-*` only on the plain `/callback` route; the connector's own `/callback/<connector-id>` route strips nothing, and custom header names (`userHeader`, `groupHeader`, …) are never stripped (dex v2.44.0, `server/server.go`, `connector/authproxy/authproxy.go`). Unlike other connector kinds, an honest mistake by the connector author is enough for an outsider; keep it in the installation's namespace.
- [ ] **Password DB and the gRPC API:** a `DexLocalConnector` renders no connector entry; it only sets `enablePasswordDB: true`, which turns on email/password login (connector ID `local`) for every client of the installation. Whoever can write password entries then controls logins: dex storage (with `kubernetes` storage, the `passwords.dex.coreos.com` objects in the dex namespace) and the gRPC API. Dex requires client certificates on gRPC only when `grpc.tlsClientCA` is set; without it, anyone who reaches the gRPC port can call `CreatePassword` (dex v2.44.0, `cmd/dex/serve.go`).
- [ ] **Mount file-based material.** `MountedSecrets` are not auto-mounted yet; a forgotten mount surfaces as a Dex startup/connector error, not a reconcile error.
- [ ] **Least-privilege upstream accounts.** LDAP bind DN, Keystone admin, Google service account: read-only, minimally scoped.

## Reporting

Security issues: please use GitHub private vulnerability reporting on [guided-traffic/dex-operator](https://github.com/guided-traffic/dex-operator) instead of public issues.
