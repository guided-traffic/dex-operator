# Secret flow: where a credential goes from a tenant's Secret to Dex

How a credential referenced by a custom resource is resolved, which of three paths it takes, and
why the rendered `config.yaml` carries none. Where all of it ends up and who can read it there is
[trust-boundaries.md](trust-boundaries.md); how a rotated credential propagates is
[rotation.md](rotation.md). The decision is
[ADR 0002](../adr/0002-custom-resources-carry-no-secret-values-credentials-reach-dex-through-an-env-secret.md).

## Custom resources carry references, never values

Every credential field is a reference: `SecretKeyRef{name, key}`
([api/v1/common_types.go](../../api/v1/common_types.go)), or `StaticClientSecretRef` with the two
key names for a static client. Neither has a namespace field: the builder resolves every reference
in the namespace of the resource that holds it, through the `SecretResolver` the controller
injects (`makeSecretResolver`,
[internal/controller/dexinstallation_controller.go](../../internal/controller/dexinstallation_controller.go);
`resolveEnvSecret`, `resolveSecret`, [internal/builder/builder.go](../../internal/builder/builder.go)).
A tenant can expose to the operator only material it can already create in its own namespace.

## Three paths

| Class | Handling | Examples |
|---|---|---|
| **Secret credentials** | Written to the **env Secret** under a deterministic name; the rendered config holds only the reference (`$VAR`, or `secretEnv: VAR` for a static client), never the value | connector client secrets, the LDAP bind password, the Keystone password, storage passwords, static-client secrets |
| **Non-secret identifiers** | Inline in `config.yaml` | client IDs, host names, base URLs; the LDAP root CA as base64 `rootCAData` |
| **File material** | Rendered as a fixed mount path in the config and reported in `Output.MountedSecrets`; not copied, not mounted by the operator ([H-13](#h-13)) | the SAML CA, TLS client certificates and keys, the Google service-account JSON, storage TLS material |

The env var names ([internal/builder/envvar.go](../../internal/builder/envvar.go)):
`<TYPE>_<ID>_<FIELD>` for a connector (special cases `LDAP_<ID>_BIND_PW`,
`KEYSTONE_<ID>_PASSWORD`), `<RESOURCE_NAME>_CLIENT_SECRET` for a static client, `STORAGE_<FIELD>`
for storage — each upper-cased, every other character replaced by `_`. Two static clients whose
names sanitize to the same key fail the build instead of silently overwriting each other's secret
([internal/builder/clients.go](../../internal/builder/clients.go)). The mount paths:
`/etc/dex/certs/<id>-<field>.pem` for connector certificates, `/etc/dex/secrets/<id>-service-account.json`
for a Google service account, `/etc/dex/certs/<postgres|etcd|mysql>-<ca|client-cert|client-key>.pem`
for storage TLS.

## Why the env indirection, with both Secrets in one namespace

- `config.yaml` is the artifact people `kubectl get`, attach to tickets and diff while debugging.
  Keeping credentials out of it prevents routine, accidental disclosure.
- The names are deterministic, and colliding static-client names are refused at build time
  instead of overwriting each other.
- A rotation changes only the env Secret; the config stays byte-identical, which avoids a Dex
  restart — and also means the rotated value waits for one ([rotation.md](rotation.md)).

The one deliberate exception to "no credential material in the config" is the LDAP root CA, which
is public material; inlining it removes a file mount.

## What this does not cover

<a id="h-13"></a>
### H-13 — File material is not mounted by the operator, and a missing mount fails only at Dex runtime

Live today. The builder computes `MountedSecrets` — namespace, Secret, key, path — but no code
outside the builder reads them: the controller creates no volume and no volume mount on the Dex
Deployment. A config that names `/etc/dex/certs/okta-ca.pem` without the file in the pod renders,
reconciles `Ready=True`, and fails when Dex starts or when the connector is first used. The Secret
named in a `MountedSecret` lives in the namespace of the resource that references it; a pod
mounts only Secrets of its own namespace, so file material of a connector outside the Dex
namespace has to be made available there by whoever runs Dex — a copy that is then readable by
every principal who can read Secrets in the Dex namespace ([H-1](trust-boundaries.md#h-1)).
Mitigation: add the volumes and mounts in the Dex Helm values at the documented paths
([docs/operations/runtime.md](../operations/runtime.md#file-material-is-yours-to-mount)), and check
Dex's log after adding a file-based connector.
