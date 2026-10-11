# The builder

`internal/builder` turns custom resources into Dex's `config.yaml`, the env Secret's data and the
list of files Dex expects mounted. The file-by-file responsibilities are in
[package-map.md](package-map.md#internalbuilder); where the output goes is
[architecture.md](architecture.md).

## One entry point, no cluster

`Build(ctx, Input) (Output, error)` in [builder.go](../../internal/builder/builder.go) is the
single entry point. It is deliberately **free of Kubernetes client code** — Secrets are resolved
through the caller-provided `SecretResolver` func, which makes the whole package unit-testable
without a cluster.

**Input:** the `DexInstallation`, a `ConnectorSet` (all 16 connector slices), all
`DexStaticClient`s (in render order, namespace/name — the controller sorts), and the resolver.

**Output:**

| Field | Meaning |
|---|---|
| `ConfigYAML` | the rendered `config.yaml` for the config Secret |
| `EnvSecretData` | env var name → value for the env Secret |
| `MountedSecrets` | Secret keys that must be projected as files (TLS certificates, service-account JSON), with the namespace of the resource that referenced them. **Informational** — no code outside the builder reads it; operators add the volumes in their Dex Helm values ([docs/operations/runtime.md](../operations/runtime.md#file-material-is-yours-to-mount)) |
| `Rejected` | every child of the input left out of the config, as `dexv1.RejectedChild` (kind, namespace, name, claimed ID, `DuplicateID` or `BuildFailed`, message), sorted by kind, namespace, name |
| `DroppedTrustedPeers` | every `trustedPeers` entry left out, sorted by namespace, name, peer ID |
| `ClientIDs` | namespace/name → the ID resolved from the spec or the Secret, for every client whose ID resolved, rendered or not — the controller records it in `status.clientID` |
| `ConnectorCount`, `StaticClientCount` | what was rendered; local connectors count |

`Build` fails only when the storage credentials cannot be resolved, when the YAML cannot be
marshalled, and with `ErrNoConnector` (the connector guard). With `ErrNoConnector` the returned
`Output` still carries `Rejected` and `DroppedTrustedPeers`, but no config.

## Claims, the contest and skip-and-report

Every child goes through [render.go](../../internal/builder/render.go) as a `renderUnit`: what it
claims, how to build it, what became of it. The decisions are
[ADR 0008](../adr/0008-an-id-renders-for-one-child-only-and-a-failing-child-is-left-out-instead-of-failing-the-render.md)
and [ADR 0009](../adr/0009-a-client-trusts-only-peers-held-in-its-own-namespace.md).

1. **Storage** builds first into the env and fails the render on error.
2. **Claims.** `connectorUnits` creates one unit per connector in render order, claiming the
   effective ID (`connectorID`); `clientUnits` one per client, claiming `spec.clientID` or the
   `client-id` from the Secret (`resolveClientID`), falling back to `status.clientID` when the
   Secret does not resolve. An empty ID is a build failure, never a claim — Dex refuses to start
   on a client without one.
3. **`reserveLocal`** marks connectors with ID `local` while a `DexLocalConnector` exists.
4. **`decide`** runs `contest` per category: one claimant wins; among several, the sole claimant in
   the installation's namespace wins; otherwise nobody. `contest` is pure and order-independent.
5. **`buildUnits`** builds every unit in env priority order (`byEnvPriority`: installation's
   namespace, oldest, kind, namespace, name). A winner builds into the render's env and renders
   on success; every other unit builds **dry** (a `childEnv` without the render's env), only to
   learn whether its own build fails — a loser that fails is `BuildFailed`, not `DuplicateID`.
6. **`filterTrustedPeers`** narrows each rendered client's `trustedPeers` to IDs whose holder —
   the contest winner, rendered or failed — sits in the client's own namespace.
7. **`noConnectorError`** is the connector guard; otherwise the rendered connectors and clients
   are assembled in render order, and CORS origins come from rendered clients only.

A build function never fails the render: it returns its error, and the unit is reported. Adding a
connector kind means adding one `addConnectorUnits` line, nothing else in this flow.

## Conventions encoded here

Documented for users in the README's naming conventions; the rule is
[ADR 0002](../adr/0002-custom-resources-carry-no-secret-values-credentials-reach-dex-through-an-env-secret.md).

- Connector client secrets: `<TYPE>_<ID>_CLIENT_SECRET`; special cases `LDAP_<ID>_BIND_PW`,
  `KEYSTONE_<ID>_PASSWORD`.
- Static client secrets: `<RESOURCE_NAME>_CLIENT_SECRET`.
- Storage passwords: `STORAGE_POSTGRES_PASSWORD`, `STORAGE_MYSQL_PASSWORD`.
- Every name goes through `sanitizeEnvKey`: upper case, every non-alphanumeric character → `_`.
- Keys are assigned through a `childEnv` per child ([envvar.go](../../internal/builder/envvar.go)):
  `set(base, field, value)` returns the plain key `<BASE>_<FIELD>` when no earlier child (in env
  priority order) holds it, else the fallback key `<BASE>_<HASH>_<FIELD>` (`childHash`: 8 hex
  characters of SHA-256 over kind/namespace/name), and fails only when both are taken.
  `commit` publishes the child's keys after a successful build. The base is `connectorEnvBase`
  (`<type>_<id>`), the client's `metadata.name`, or `storageEnvBase`. Never write into the env map
  directly — `resolveEnvSecret` and `childEnv.set` are the only ways in.
- Mount paths: `/etc/dex/certs/<id>-<field>.pem`; Google service account
  `/etc/dex/secrets/<id>-service-account.json`; storage TLS `/etc/dex/certs/<storage>-<field>.pem`.
- The connector ID is `spec.id`, else `metadata.name` (`connectorID`).
- A `DexLocalConnector` does not render a connector entry — it flips `enablePasswordDB: true`.
- The LDAP root CA is inlined as base64 `rootCAData`; client IDs are inline; client secrets go to
  env.

## Static clients: confidential vs. public

`DexStaticClientSpec` sources the client ID either from `secretRef` (confidential) or from the
inline `clientID` field (public/secretless, PKCE — supported by Dex since v2.24.0).
`resolveClientID` and `buildOneStaticClient` ([clients.go](../../internal/builder/clients.go))
branch on `SecretRef == nil`; a secretless client skips secret resolution, the env key and the
`EnvSecretData` entry entirely, leaving `secretEnv` unset in the config.

There is **no admission webhook** in this repo. All conditional validation is done with CEL
markers (`+kubebuilder:validation:XValidation`) on the spec struct:

1. `has(self.clientID) != has(self.secretRef)` — exactly one of the two.
2. `(has(self.public) && self.public) || has(self.secretRef)` — confidential needs `secretRef`.
3. `(has(self.public) && self.public) || (has(self.redirectURIs) && self.redirectURIs.size() > 0)`
   — confidential needs `redirectURIs`; public clients may omit them and get Dex's
   loopback/OOB/device-flow defaults.

`public` carries `omitempty` and no default, so the rules must guard with `has(self.public)` — a
bare `self.public` would error on objects that never set it. The rules are covered by envtest
integration tests (the envtest API server enforces CEL). The decision is
[ADR 0004](../adr/0004-a-static-client-is-confidential-through-a-secret-or-public-through-an-inline-id-validated-by-cel.md).

## Derived CORS origins

`spec.cors` on a `DexStaticClient` opts that client into contributing the origins of its own https
`redirectURIs` to the installation's `web.allowedOrigins`. `deriveCORSOrigins` runs in `Build` over
the *spec-level* objects of the rendered clients (`renderedClients` — Dex has no per-client CORS,
so nothing lands in the rendered `staticClients` entry) and `assembleWebConfig` merges the result.
A rejected client contributes no origin.

Decisions worth knowing before touching this — settled with the owner, recorded in
[ADR 0005](../adr/0005-a-static-client-opts-in-to-derive-its-cors-origins-from-its-own-https-redirect-uris.md),
not to be reverted silently:

- **Only https, and only the client's own redirect URIs.** Loopback/http targets, custom schemes
  and the OOB URN belong to native clients and are skipped; unparsable URIs are skipped rather than
  failing the build, so one malformed tenant resource cannot break the render for every other
  client. Why the flag grants no authority beyond `redirectURIs` is
  [docs/security/clients.md](../security/clients.md#tenant-registered-cors-origins-cors-true).
- **The authored list keeps its order**, derived origins are appended sorted
  (`appendDerivedOrigins`). Sorting the whole union would rewrite existing
  `spec.web.allowedOrigins` lists on an operator upgrade alone → config diff → spurious Dex
  rollout. The sorted tail is what makes the output independent of client iteration order.
- **`spec.web == nil` + derived origins creates the `web:` block** with only `allowedOrigins`.
  Safe because Helm chart deployments pass `--web-http-addr`/`--web-https-addr` as CLI flags,
  applied after config load.
- **Hosts are lowercased and a redundant `:443` dropped.** Dex matches the `Origin` header
  literally (`gorilla/handlers.AllowedOrigins`); without normalization a mixed-case host or an
  explicit default port renders an entry that can never match.
- **The flag is not gated on `public`.** A confidential client that sets it gets its origins
  derived too — no hidden conditional to debug.

## Tests

The builder's unit tests are in [builder_test.go](../../internal/builder/builder_test.go) (helpers
`minimalInstallation`, `mockResolver`, `parseYAML`),
[contest_test.go](../../internal/builder/contest_test.go) (claims, contest, skip-and-report, the
connector guard, env key priority, trusted peers; helpers `confClient`, `pubClient`,
`clientSecrets`, `oidcConn`, `build`) and
[builder_dexschema_test.go](../../internal/builder/builder_dexschema_test.go), which pins the
rendered keys to Dex's schema (`assertNoKeys`). More in [testing.md](testing.md).
