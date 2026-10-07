# ADR 0002: Custom Resources Carry No Secret Values; Credentials Reach Dex Through an Env Secret, File Material Through a Mount Path

## Status

Accepted. Date: 2026-10-07, recording a design in force since the first release. D2's
`secretEnv` for static clients since 1.0 (`fix(builder): use secretEnv instead of secret for
static clients`, 2026-03-08).

**Partly built:** D1–D4 and D6 hold. D5's mounts are computed and returned as
`Output.MountedSecrets`, but no code outside the builder reads them: the controller creates no
volume and no volume mount, and the operator of the Dex installation mounts the files
([docs/operations/runtime.md](../operations/runtime.md#file-material-is-yours-to-mount)).

## Context

Dex reads one `config.yaml` and expands `$VAR` references in it from its environment; static
clients take `secretEnv: VAR` instead of an inline `secret`. The operator assembles that file from
up to eighteen kinds of custom resources, many of them written by tenants in their own
namespaces. Three things pull against each other: credentials must reach Dex, the rendered config
is the artifact people read, diff and paste into tickets, and a tenant must never be able to
expose material it could not already read.

## Decision

**D1 — A custom resource carries no secret value.** Every credential field is a reference —
`SecretKeyRef{name, key}`, or `StaticClientSecretRef` with its two key names — and has no
namespace field: the operator resolves it in the namespace of the resource that holds it
([api/v1/common_types.go](../../api/v1/common_types.go) `SecretKeyRef`;
[internal/builder/builder.go](../../internal/builder/builder.go) `resolveEnvSecret`,
`resolveSecret`). A tenant can expose only material it can already create in its own namespace.

**D2 — A credential reaches Dex through the env Secret, never through `config.yaml`.** The
builder writes the value into `Output.EnvSecretData` under a deterministic name and puts only the
reference into the config: `$VAR` for connector and storage credentials, `secretEnv: VAR` for a
static client ([internal/builder/envvar.go](../../internal/builder/envvar.go)). The env Secret is
attached to the Dex container with `envFrom` in the Dex chart's values, which the operator does
not write.

**D3 — The env var names are deterministic and collisions are refused.**
`<TYPE>_<ID>_<FIELD>` for a connector, `<RESOURCE_NAME>_CLIENT_SECRET` for a static client,
`STORAGE_<FIELD>` for storage, every name sanitized to upper case with every other character
replaced by `_`. Two static clients whose names sanitize to the same key fail the build instead of
overwriting each other's secret ([internal/builder/clients.go](../../internal/builder/clients.go)).

**D4 — Identifiers are inline.** A client ID, a host name, a base URL is not a secret and is
rendered into the config. The LDAP root CA (`rootCARef`) is inlined as base64 `rootCAData`: a CA
certificate is public material, and inlining removes a mount.

**D5 — File material is a mount path.** A key that Dex reads from a file — a SAML CA, a TLS client
certificate and key, a Google service-account JSON, storage TLS material — is rendered as a fixed
path (`/etc/dex/certs/<id>-<field>.pem`, `/etc/dex/secrets/<id>-service-account.json`,
`/etc/dex/certs/<storage>-<field>.pem`) and reported in `Output.MountedSecrets` with the Secret's
namespace, name and key.

**D6 — The operator never deletes a Secret.** Its ClusterRole has no `delete` on `secrets`;
generated Secrets left behind by a renamed `configSecretName` or `envSecretName` are removed by a
person.

## Consequences

- Rotating a credential changes only the env Secret: the config stays byte-identical and no
  rollout restart follows ([ADR 0003](0003-the-render-is-deterministic-and-change-driven-no-periodic-requeue.md)) —
  which also means Dex reads the new value only when its pods restart.
- The env Secret aggregates every client secret and connector credential of the installation.
  Read access to it, or to the Dex namespace's Secrets in general, is read access to all of them.
- A forgotten file mount surfaces as a Dex start or connector error, not as a reconcile error.
- A Secret that a tenant references for file material lives in the tenant's namespace; a pod
  mounts only Secrets of its own namespace, so that material has to be made available in the Dex
  namespace by whoever runs Dex.
- Stale generated Secrets accumulate after a rename until a person removes them.

## Alternatives Considered

- **Inline credentials in `config.yaml`.** One Secret instead of two, but every `kubectl get` and
  every diff of the config shows them. Lost.
- **A namespace field on `SecretKeyRef`.** Lets a resource point at a Secret elsewhere, and so
  lets a tenant make the operator read material the tenant cannot read. Lost.
- **Mount every file in the controller.** Needs write access to the Dex Deployment beyond the
  restart annotation and copying tenant Secrets into the Dex namespace. Not built; D5's report is
  the interface such a change would consume.
- **Give the operator `delete` on Secrets** to clean up after renames. Makes the operator able to
  destroy credentials it only needs to write. Lost.

## Residual risks

- The env indirection keeps credentials out of the config artifact, not out of the namespace:
  config and env Secret sit side by side, readable by the same principals.
- Not verified: how Dex treats a `$VAR` whose variable is unset (an empty value or a start
  error); the operator always writes the variable it references.

## References

- [internal/builder/builder.go](../../internal/builder/builder.go), [envvar.go](../../internal/builder/envvar.go),
  [clients.go](../../internal/builder/clients.go), [connectors.go](../../internal/builder/connectors.go),
  [storage.go](../../internal/builder/storage.go)
- [internal/controller/rbac.go](../../internal/controller/rbac.go),
  [deploy/helm/dex-operator/templates/clusterrole.yaml](../../deploy/helm/dex-operator/templates/clusterrole.yaml)
- [docs/security/secret-flow.md](../security/secret-flow.md)
