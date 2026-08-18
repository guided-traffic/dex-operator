# Security Architecture

How the dex-operator handles credentials, isolates tenants, and what its trust model assumes. Written for platform operators who run the operator and for security reviewers who audit it. Code references point at the enforcing implementation.

## System Roles and Trust Boundaries

| Role | Trust level |
|---|---|
| **dex-operator** | High privilege: cluster-wide read of all 18 CRDs, read/write of Secrets, patch of Deployments. Compromise of the operator ≈ compromise of the SSO configuration. |
| **Dex namespace** (e.g. `dex`) | Sink for all rendered material. Anyone who can read Secrets here can read the full config **and every client secret**. |
| **App namespaces** | Semi-trusted tenants. They may contribute connectors/clients *only if* allowlisted, and they expose credentials to the operator only by referencing their own Secrets. |
| **Kubernetes API server** | Trusted enforcement point: RBAC, CEL validation, etcd storage. |

```
app namespace (tenant)                dex namespace
┌───────────────────────┐            ┌─────────────────────────────┐
│ DexStaticClient       │            │ DexInstallation             │
│ Dex*Connector         │──ref──┐    │  allowedNamespaces ✓ gate   │
│ Secret (credentials)  │       │    │                             │
└───────────────────────┘       ▼    │ config Secret (config.yaml) │
                          dex-operator ──► env Secret (env vars)    │
                                     │        │ envFrom            │
                                     │        ▼                    │
                                     │ Dex Deployment              │
                                     └─────────────────────────────┘
```

## Namespace Isolation (`allowedNamespaces`)

The central multi-tenancy control. Each `DexInstallation` declares which namespaces may contribute connectors and static clients:

- **empty / omitted → deny all** (fail-closed default)
- `"*"` → every namespace
- anything else → literal namespace names

It is enforced twice, independently:

1. **At build time** — `collectConnectors` / `collectStaticClients` filter every listed child through the allowlist before the config is rendered ([internal/controller/collect.go](internal/controller/collect.go), `filterItems` → `isNamespaceAllowed`). A non-allowlisted resource can never influence the rendered config, regardless of what its own status claims.
2. **At child reconciliation** — the generic child reconciler marks resources from forbidden namespaces with a `Ready=False` condition and a clear reason ([internal/controller/child_reconciler.go](internal/controller/child_reconciler.go)), giving tenants feedback instead of silent exclusion.

**What the allowlist defends against:** a tenant in an arbitrary namespace registering an OAuth2 client (or an identity-providing connector!) with your company SSO. A rogue client with an attacker-controlled redirect URI would receive authorization codes for real users; a rogue connector could mint identities. With the allowlist, only namespaces you explicitly trust can do either.

**What it does not defend against:** principals who already have `create` rights on Dex CRDs *inside* an allowed namespace. That boundary is Kubernetes RBAC — treat `dexstaticclients`/`dex*connectors` create/update as privileged verbs and grant them accordingly.

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
- [ ] **Treat CRD verbs as privileged.** `create`/`update` on `dexstaticclients` and connectors in an allowed namespace equals the power to register SSO clients. Scope RBAC per namespace/team.
- [ ] **Prefer explicit `allowedNamespaces`.** `"*"` shifts the entire tenancy boundary onto CRD RBAC.
- [ ] **Audit `insecure*` flags.** Every connector option prefixed `insecure` (skip TLS verify, skip signature validation, skip email-verified, `insecureNoSSL`, `insecureCA`) removes a verification step and belongs in dev environments only. They are greppable in cluster: `kubectl get dex<type>connectors -A -o yaml | grep -n insecure`.
- [ ] **Scope upstream restrictions.** Connectors without `orgs`/`groups`/`hostedDomains`/`teams`/`tenant` filters accept *any* account of that provider. Filter at the connector, not only in the app.
- [ ] **AuthProxy connector:** only deploy behind a proxy that is the exclusive network path to Dex and strips inbound identity headers — otherwise identity forgery is trivial.
- [ ] **Mount file-based material.** `MountedSecrets` are not auto-mounted yet; a forgotten mount surfaces as a Dex startup/connector error, not a reconcile error.
- [ ] **Least-privilege upstream accounts.** LDAP bind DN, Keystone admin, Google service account: read-only, minimally scoped.

## Reporting

Security issues: please use GitHub private vulnerability reporting on [guided-traffic/dex-operator](https://github.com/guided-traffic/dex-operator) instead of public issues.
