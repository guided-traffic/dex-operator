# Trust boundaries of an installation

Who takes part in a Dex installation run by the operator, how far each is trusted, and where the
rendered material ends up. Which namespaces may contribute what is [tenancy.md](tenancy.md); what
a connector can make every client believe is [connectors.md](connectors.md); how a credential
travels from a tenant's Secret into Dex is [secret-flow.md](secret-flow.md); what the operator may
do in the cluster is [privilege-footprint.md](privilege-footprint.md).

## The parties

| Party | Trust | What it holds or decides |
|---|---|---|
| **The operator** (its ServiceAccount) | High. Compromising it is compromising the single sign-on configuration | Cluster-wide read of every dex.gtrfc.com kind and of every Secret, create/update/patch of Secrets, patch of Deployments ([privilege-footprint.md](privilege-footprint.md)) |
| **The Dex namespace** (the installation's namespace, `dex` in the README) | The sink of everything rendered | The `DexInstallation`, the config Secret (`configSecretName`), the env Secret (`envSecretName`) with **every** client secret and connector credential of the installation, the Dex Deployment, and by default every connector ([tenancy.md](tenancy.md)) |
| **Tenant namespaces** | Semi-trusted | May contribute `DexStaticClient`s only if listed in `allowedNamespaces`, connectors only if listed in `allowedConnectorNamespaces` (by default no tenant namespace is). Expose credentials to the operator only by referencing Secrets in their own namespace ([secret-flow.md](secret-flow.md)) |
| **Whoever writes the `DexInstallation`** | Platform | Decides the issuer, the storage, both allowlists, the authored CORS origins and whether Dex is restarted on change |
| **The Kubernetes API server** | Trusted enforcement point | RBAC on the CRDs and Secrets, CEL and schema validation at admission ([validation.md](validation.md)), storage in etcd |
| **Dex** | Trusted to do what its config says | Reads `config.yaml` from the config Secret and the `$VAR` values from its environment, which the Dex chart fills from the env Secret with `envFrom` |
| **Upstream identity providers** | As trusted as the connector that names them | Return the identities Dex passes on to every client ([connectors.md](connectors.md)) |

```
tenant namespace                         Dex namespace (installation's own)
┌────────────────────────┐              ┌─────────────────────────────────────┐
│ DexStaticClient        │              │ DexInstallation                     │
│ Dex*Connector          │──ref────┐    │  allowedNamespaces         ✓ gate   │
│ Secret (credentials)   │         │    │    (static clients)                 │
└────────────────────────┘         │    │  allowedConnectorNamespaces ✓ gate  │
                                   │    │    (connectors; default: own only)  │
                                   ▼    │ Dex*Connector (default home)        │
                             dex-operator ──► config Secret (config.yaml)     │
                                        │    └─► env Secret (every credential)│
                                        │          │ envFrom                  │
                                        │          ▼                          │
                                        │ Dex Deployment ◄── restart patch    │
                                        └─────────────────────────────────────┘
```

The operator writes both Secrets into the namespace of the `DexInstallation`
([internal/controller/dexinstallation_controller.go](../../internal/controller/dexinstallation_controller.go)
`reconcileInstallation`) and, with `rolloutRestart.enabled`, patches the Deployment named in
`rolloutRestart.deploymentName` in that same namespace
([internal/controller/rollout.go](../../internal/controller/rollout.go)). Dex itself is installed
by the official Dex Helm chart; the operator neither creates nor configures its Deployment beyond
the restart annotation.

## What crosses a boundary

- **From a tenant namespace to the operator:** custom resources and the Secrets they reference by
  name. Nothing a tenant writes is evaluated as code; the builder renders it as data into a
  YAML-tagged struct and marshals it ([internal/builder/config_types.go](../../internal/builder/config_types.go)).
- **From the operator to the Dex namespace:** two Secrets and a pod-template annotation. The
  operator never copies a tenant's Secret into the Dex namespace — values travel inside the env
  Secret, file material does not travel at all ([secret-flow.md](secret-flow.md)).
- **From Dex to every relying party:** identities from every admitted connector, issued for every
  client of the installation.

## What this does not cover

<a id="h-1"></a>
### H-1 — Everything sensitive converges in the Dex namespace

Live by design ([ADR 0002](../adr/0002-custom-resources-carry-no-secret-values-credentials-reach-dex-through-an-env-secret.md)).
The env Secret aggregates every client secret, connector credential and storage password of the
installation in one object, next to the config Secret. A principal who can `get` Secrets in the
Dex namespace — a namespace admin, a too-wide RoleBinding, a backup tool, a person with etcd
access — reads all of them at once, whatever namespaces the credentials came from. The env
indirection keeps them out of `config.yaml`, not out of the namespace. Mitigation: grant Secret
read in the Dex namespace to nobody but Dex and the operator, keep connectors in that namespace
only where their owners may read each other's credentials anyway, and encrypt Secrets at rest in
etcd.

<a id="h-2"></a>
### H-2 — The operator's ServiceAccount is a single point of compromise

Live by design. Its cluster-wide Secret read is inherent to tenants referencing Secrets in their
own namespaces ([privilege-footprint.md](privilege-footprint.md)); a token of that ServiceAccount
reads every Secret of the cluster, not only those the Dex installation needs. Mitigation: run the
operator in a namespace of its own that no tenant can exec into or read Secrets in, and alert on
Secret access by its ServiceAccount from anywhere but its own pod — the Kubernetes audit log
records it cheaply and the signal is high.
