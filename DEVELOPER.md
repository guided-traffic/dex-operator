# Developer Guide

Technical walkthrough of the dex-operator codebase: where things live, what each package does, and how the pieces interact. For user-facing documentation see the [README](README.md); for the security design see [SECURITY_ARCHITECTURE.md](SECURITY_ARCHITECTURE.md).

## Repository Layout

```
.
├── cmd/
│   └── main.go                  # manager entry point: scheme, manager, controller setup
├── api/v1/                      # CRD types (API group dex.gtrfc.com/v1)
│   ├── groupversion_info.go     # GroupVersion, SchemeBuilder
│   ├── common_types.go          # InstallationRef, SecretKeyRef, CommonStatus, condition types
│   ├── dexinstallation_types.go # DexInstallation + storage/web/grpc/logger/expiry/oauth2/frontend specs
│   ├── dexstaticclient_types.go # DexStaticClient + CEL validation rules
│   ├── dex<type>connector_types.go  # one file per connector CRD (16 total)
│   ├── connector_helpers.go     # ChildObject implementations for every child CRD
│   └── zz_generated.deepcopy.go # generated — never edit, run `make generate-all`
├── internal/
│   ├── builder/                 # pure config assembly: CRs → config.yaml + env data
│   └── controller/              # reconcilers, collection, secret writing, watches
├── config/
│   ├── crd/bases/               # generated CRD manifests (source of truth)
│   ├── rbac/                    # generated ClusterRole
│   ├── manager/ + default/      # kustomize deployment (used by `make deploy`)
├── deploy/helm/dex-operator/    # Helm chart shipped to users
│   ├── files/crds/              # CRDs synced from config/crd/bases (make helm-sync-crds)
│   └── templates/               # deployment, RBAC, CRD upgrade job (pre-install/upgrade hook)
├── test/
│   ├── integration/             # envtest-based integration suite (build tag: integration)
│   └── e2e/                     # Kind-based end-to-end suite (build tag: e2e)
├── Makefile                     # single entry point for build/test/lint/generate
├── Containerfile                # manager image build
└── .github/workflows/           # CI (build.yml), release (release.yml, semantic-release)
```

## Architecture

Two controller kinds cooperate:

1. **`DexInstallationReconciler`** ([internal/controller/dexinstallation_controller.go](internal/controller/dexinstallation_controller.go)) — owns the actual output. On every reconcile it collects all child resources, calls the builder, and applies the config/env Secrets.
2. **`GenericChildReconciler[T, U]`** ([internal/controller/child_reconciler.go](internal/controller/child_reconciler.go)) — one generic instance per child CRD (16 connectors + `DexStaticClient`, registered in [connector_controller.go](internal/controller/connector_controller.go)). It only validates (installation exists, namespace allowed) and maintains the child's `Ready` condition. It does **not** build anything.

Config regeneration is always funneled through the installation reconciler via watches:

- every child CRD type is watched and mapped to its `spec.installationRef` (`mapChildToInstallation`)
- referenced Secrets are watched via a field index (`SecretRefIndexField`); create/update events map back to the owning installations (`mapSecretToInstallation` in [secret_watch.go](internal/controller/secret_watch.go)). Secret **deletes are intentionally ignored** — during credential rotation the replacement Secret triggers the reconcile; reacting to the delete would only produce a failing loop.

### Reconciliation flow (DexInstallation)

```
Reconcile
 ├─ collectConnectors / collectStaticClients   (collect.go)
 │    ├─ List via field index .spec.installationRef == "<ns>/<name>"
 │    ├─ filterItems: namespace allowlist (empty = deny all, "*" = all)
 │    └─ sortByNamespaceName: deterministic order (see below)
 ├─ builder.Build(Input)                        (internal/builder)
 │    └─ resolves SecretKeyRefs via injected SecretResolver
 ├─ applySecret(configSecretName, config.yaml)  (secret.go)
 ├─ applySecret(envSecretName, env map)
 ├─ if config changed → triggerRolloutRestart   (rollout.go)
 └─ status: connectorCount, staticClientCount, Ready condition (status.go)
```

**Determinism matters:** the cache-backed `List` returns items in map order. Without the stable sort, the rendered YAML would reorder `connectors`/`staticClients` on every pass, the change detection would fire every time, and `rolloutRestart` would bounce the Dex Deployment in a loop. The same reasoning drives `yamlSecretDataEqual` in [secret.go](internal/controller/secret.go): the config Secret is compared *semantically* (parsed YAML, `reflect.DeepEqual`), not byte-wise.

Child resources with user-caused problems (missing installation, forbidden namespace) return a `configError`, which is reported via the `Ready` condition and requeued after 5 minutes instead of entering exponential backoff ([child_reconciler.go](internal/controller/child_reconciler.go)).

### The ChildObject interface

[interfaces.go](internal/controller/interfaces.go) defines the contract every child CRD implements (in [api/v1/connector_helpers.go](api/v1/connector_helpers.go)):

```go
type ChildObject interface {
    GetInstallationRef() dexv1.InstallationRef
    GetCommonStatus() *dexv1.CommonStatus
    GetReferencedSecretNames() []string   // feeds the Secret watch index
}
```

This is what lets one generic reconciler, one indexer pair, and one secret-watch mapper serve 17 CRDs.

## Builder Package (`internal/builder`)

`Build(ctx, Input) (Output, error)` in [builder.go](internal/builder/builder.go) is the single entry point. It is deliberately **free of Kubernetes client code** — Secrets are resolved through the caller-provided `SecretResolver` func, which makes the whole package unit-testable without a cluster.

**Input:** the `DexInstallation`, a `ConnectorSet` (all 16 connector slices), all `DexStaticClient`s, and the resolver.

**Output:**

| Field | Meaning |
|---|---|
| `ConfigYAML` | rendered `config.yaml` for the config Secret |
| `EnvSecretData` | env var name → value for the env Secret |
| `MountedSecrets` | Secret keys that must be projected as files (TLS certs, SA JSON). **Currently informational** — the controller does not yet mount them; users add volumes in their Dex Helm values |

File map:

| File | Responsibility |
|---|---|
| [builder.go](internal/builder/builder.go) | orchestration, `assembleDexConfig`/`assembleWebConfig`, CORS origin derivation (`deriveCORSOrigins`), secret/mount helpers, `connectorID` (spec.id → metadata.name fallback) |
| [config_types.go](internal/builder/config_types.go) | YAML-tagged structs mirroring Dex's `config.yaml` schema |
| [envvar.go](internal/builder/envvar.go) | env var naming: `sanitizeEnvKey` (uppercase, non-alnum → `_`), `connectorEnvKey`, `clientEnvKey`, `storageEnvKey` |
| [clients.go](internal/builder/clients.go) | static clients: confidential vs. public branch, env-key collision detection, `secretEnv` wiring |
| [connectors.go](internal/builder/connectors.go) | LDAP (inline base64 `rootCAData`, mounted client cert/key), SAML (mounted `ca`), AuthProxy |
| [connectors_oauth.go](internal/builder/connectors_oauth.go) | all OAuth2-style connectors; shared `resolveOAuthCreds` (client ID inline, client secret → env) |
| [storage.go](internal/builder/storage.go) | kubernetes/memory/postgres/sqlite3/etcd/mysql; passwords → `STORAGE_*` env, TLS material → fixed mount paths |

Conventions encoded here (documented user-facing in the README):

- connector client secrets: `<TYPE>_<ID>_CLIENT_SECRET`; special cases `LDAP_<ID>_BIND_PW`, `KEYSTONE_<ID>_PASSWORD`
- static client secrets: `<RESOURCE_NAME>_CLIENT_SECRET`, with an explicit collision check across clients
- cert mount paths: `/etc/dex/certs/<id>-<field>.pem`, Google SA: `/etc/dex/secrets/<id>-service-account.json`
- a `DexLocalConnector` does not render a connector entry — it flips `enablePasswordDB: true`

## Static Clients: Confidential vs. Public

`DexStaticClientSpec` sources the client ID either from `secretRef` (confidential) or from the inline `clientID` field (public/secretless, PKCE — supported by Dex since v2.24.0). `buildOneStaticClient` branches on `SecretRef == nil` and then skips secret resolution, the env-key collision check and the `EnvSecretData` entry entirely, leaving `secretEnv` unset in the config.

There is **no admission webhook** in this repo. All conditional validation is done with CEL markers (`+kubebuilder:validation:XValidation`) on the spec struct:

1. `has(self.clientID) != has(self.secretRef)` — exactly one of the two.
2. `(has(self.public) && self.public) || has(self.secretRef)` — confidential needs `secretRef`.
3. `(has(self.public) && self.public) || (has(self.redirectURIs) && self.redirectURIs.size() > 0)` — confidential needs `redirectURIs`; public clients may omit them and get Dex's loopback/OOB/device-flow defaults.

`public` carries `omitempty` and no default, so the rules must guard with `has(self.public)` — a bare `self.public` would error on objects that never set it. The rules are covered by envtest integration tests (the envtest apiserver enforces CEL).

## Derived CORS Origins

`spec.cors` on a `DexStaticClient` opts that client into contributing the origins of its own https `redirectURIs` to the installation's `web.allowedOrigins`. `deriveCORSOrigins` runs in `Build` over the *spec-level* clients (before `buildStaticClients` flattens them — Dex has no per-client CORS, so nothing lands in the rendered `staticClients` entry) and `assembleWebConfig` merges the result.

Design decisions worth knowing before touching this:

- **Only https, and only the client's own redirect URIs.** Loopback/http targets, custom schemes and the OOB URN belong to native clients and are skipped; unparsable URIs are skipped rather than failing the build, so one malformed tenant resource cannot break the render for every other client. See [SECURITY_ARCHITECTURE.md](SECURITY_ARCHITECTURE.md#tenant-registered-cors-origins-cors-true) for why the flag grants no authority beyond `redirectURIs`.
- **The authored list keeps its order**, derived origins are appended sorted (`appendDerivedOrigins`). Sorting the whole union would rewrite existing `spec.web.allowedOrigins` lists on an operator upgrade alone → config diff → spurious dex rollout. The sorted tail is what makes the output independent of client iteration order.
- **`spec.web == nil` + derived origins creates the `web:` block** with only `allowedOrigins`. Safe because helm-chart deployments pass `--web-http-addr`/`--web-https-addr` as CLI flags, applied after config load.
- **Hosts are lowercased and a redundant `:443` dropped.** Dex matches the `Origin` header literally (`gorilla/handlers.AllowedOrigins`); without normalization a mixed-case host or explicit default port renders an entry that can never match.
- **The flag is not gated on `public`.** A confidential client that sets it gets its origins derived too — no hidden conditional to debug.

## Adding a New Connector Type

Checklist, in order:

1. **API type** — `api/v1/dex<type>connector_types.go`: spec struct (with `InstallationRef`, `ID`, `DisplayName`), status embedding `CommonStatus`, kubebuilder markers (`shortName`, printcolumns), `init()` scheme registration.
2. **ChildObject** — add the three methods in [api/v1/connector_helpers.go](api/v1/connector_helpers.go); `GetReferencedSecretNames` must list every Secret the spec can reference (drives rotation reactivity).
3. **Builder** — add the slice to `ConnectorSet` ([builder.go](internal/builder/builder.go)), write `build<Type>Connector`, wire it into `buildAllConnectors` (or the OAuth loop in [connectors_oauth.go](internal/builder/connectors_oauth.go)).
4. **Collection** — add a collector method + `ConnectorSet` field in [collect.go](internal/controller/collect.go), extend `countConnectors`.
5. **Wiring** — extend the type lists in `registerIndexers` and `childWatchSources` ([dexinstallation_controller.go](internal/controller/dexinstallation_controller.go)), `lookupChildSecretRefs` ([secret_watch.go](internal/controller/secret_watch.go)), the reconciler list in [connector_controller.go](internal/controller/connector_controller.go), and the RBAC markers in [rbac.go](internal/controller/rbac.go).
6. **Generate** — `make generate-all` (regenerates DeepCopy + CRDs and syncs them into the Helm chart).
7. **Tests** — builder unit test (config rendering, env vars, mounts) + integration test (reconcile, namespace allowlist, status).

## Build, Test, Lint

Always use the Makefile targets — they match CI exactly.

| Target | What it does |
|---|---|
| `make build` / `make run` | build binary / run locally with debug logging (current kubeconfig) |
| `make generate-all` | controller-gen: DeepCopy + CRDs + RBAC, then sync CRDs into the Helm chart. Run after **any** `api/v1` change |
| `make lint` | `go vet`, `gofmt -l`, `golangci-lint` |
| `make cyclo` | cyclomatic complexity gate — keep every function **< 15** |
| `make test` | fmt + vet + all Go tests under envtest |
| `make test-unit` / `-coverage` | `-short` test run (skips integration) |
| `make test-integration` / `-coverage` | envtest suite in `test/integration/` (build tag `integration`); real apiserver, CEL enforced |
| `make test-e2e` | e2e suite against a running Kind cluster (build tag `e2e`) |
| `make e2e-local` | full loop: Kind cluster + image build + Helm install + e2e + teardown |
| `make gosec` / `make vuln` | security static analysis / govulncheck |
| `make docker-build` / `docker-buildx` | image build (multi-arch with buildx), `IMG=` to override the tag |
| `make coverage-merge` / `coverage-json` | merge unit+integration profiles, emit the README badge JSON |

Coverage profiles exclude generated code (`zz_generated.*`, see `COVERAGE_EXCLUDE_RE`
in the [Makefile](Makefile)) — the filter runs directly after every
profile-producing test target, so local reports, the CI merge step and the badge
all measure hand-written code only.

Test layers:

- **Unit** (`internal/...`): builder rendering, env naming, determinism — no cluster.
- **Integration** (`test/integration/`): envtest apiserver; reconciler behavior, namespace allowlist, Secret generation, CEL validation of `DexStaticClient`.
- **E2E** (`test/e2e/`): Kind + Helm chart install (values in [test/e2e/helm-values.yaml](test/e2e/helm-values.yaml)); real-world flows including the CRD upgrade hook and Helm migration.

## Helm Chart & CRD Lifecycle

The chart in [deploy/helm/dex-operator/](deploy/helm/dex-operator/) does **not** use Helm's `crds/` directory (which would never upgrade CRDs). Instead:

- CRD manifests live in `files/crds/`, kept in sync from `config/crd/bases/` by `make helm-sync-crds` (part of `make manifests` / `generate-all`)
- a pre-install/pre-upgrade hook Job (`crd-upgrade-job.yaml`) applies them with `kubectl` (image configurable via `crdUpgradeJob.image.*`)

So CRD updates ship automatically with every `helm upgrade`.

## CI & Releases

- **release.yml** ("Test and Release") — on PRs and pushes to `main`: lint, security scans, unit/integration/E2E tests, combined coverage report; on `main` additionally [semantic-release](https://semantic-release.gitbook.io/) (`.releaserc.json`) derives the version from **Conventional Commits**, tags and creates the GitHub release with generated notes.
- **build.yml** ("Release Docker & Helm") — on `release: published`: builds and pushes the Docker image `guidedtraffic/dex-operator`, packages the Helm chart, publishes it to the Helm repo at `https://guided-traffic.github.io/dex-operator` (gh-pages branch) and attaches only the current chart `.tgz` to the release.
- **renovate.yml / renovate.json** — automated dependency updates. `conventional-changelog-conventionalcommits` is held at v9 (major updates disabled in `renovate.json`): with v10, `@semantic-release/release-notes-generator` silently drops all commits and release notes come out empty.

Because versioning is commit-driven, commit messages must follow Conventional Commits (`feat:`, `fix:`, `chore:` …); `feat!:`/`BREAKING CHANGE:` triggers a major release.

## Conventions

- Go 1.26, controller-runtime; cyclomatic complexity < 15 per function (`make cyclo`)
- code, comments and documentation in English
- never edit generated files (`zz_generated.deepcopy.go`, `config/crd/bases/`, `deploy/helm/dex-operator/files/crds/`) — change the source types and run `make generate-all`
- new features and bug fixes come with tests; reconciliation logic is the coverage priority
