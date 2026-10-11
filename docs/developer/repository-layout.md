# Repository layout

The tree, annotated. What each file of the Go packages is responsible for is
[package-map.md](package-map.md).

```
.
├── cmd/
│   └── main.go                      # manager entry point: scheme, flags, manager, controller setup order
├── api/v1/                          # CRD types, API group dex.gtrfc.com/v1
│   ├── groupversion_info.go         # GroupVersion, SchemeBuilder
│   ├── common_types.go              # InstallationRef, SecretKeyRef, CommonStatus, condition types
│   ├── dexinstallation_types.go     # DexInstallation + storage/web/grpc/logger/expiry/oauth2/frontend specs, both allowlists, rejected children
│   ├── dexstaticclient_types.go     # DexStaticClient + its CEL rules, status.clientID
│   ├── dex<type>connector_types.go  # one file per connector CRD (16)
│   ├── connector_helpers.go         # ChildObject implementations for every child CRD
│   └── zz_generated.deepcopy.go     # generated — never edit, run `make generate-all`
├── internal/
│   ├── builder/                     # pure config assembly: custom resources → config.yaml + env data
│   └── controller/                  # reconcilers, collection, Secret writing, watches, RBAC markers
│       └── testdata/                # golden renders of earlier releases for render_compat_test.go
├── config/                          # kustomize deployment, used by `make install`/`make deploy`
│   ├── crd/bases/                   # generated CRD manifests — the source of truth for every copy
│   ├── rbac/                        # generated ClusterRole (role.yaml) + leader election + bindings
│   ├── manager/                     # the manager Deployment
│   └── default/                     # the kustomization that ties them together
├── deploy/helm/dex-operator/        # the Helm chart shipped to users
│   ├── files/crds/                  # CRDs copied from config/crd/bases by `make helm-sync-crds`
│   ├── templates/                   # Deployment, hand-written RBAC, the CRD ConfigMap and upgrade hook Job
│   └── values.yaml
├── test/
│   ├── integration/                 # envtest suite (build tag `integration`)
│   └── e2e/                         # Kind suite (build tag `e2e`), helm-values.yaml for the chart install
├── hack/boilerplate.go.txt          # license header for generated code
├── docs/
│   ├── adr/                         # decisions
│   ├── developer/                   # this directory
│   ├── operations/                  # running the operator
│   ├── security/                    # the security architecture, one page per perspective
│   └── tickets/                     # work lists; archive/ for finished ones
├── .github/
│   ├── workflows/                   # release.yml (test and release), build.yml (images and chart), renovate.yml
│   └── badges/coverage.json         # the README's coverage badge, written by semantic-release
├── Makefile                         # the single entry point for build, generate, test, lint
├── Containerfile                    # the manager image
├── .golangci.yml                    # linter configuration
├── .releaserc.json, package.json    # semantic-release and its plugins
└── renovate.json                    # dependency updates
```

Generated paths — never edited by hand, changed by editing `api/v1` or the RBAC markers and
running `make generate-all`: `api/v1/zz_generated.deepcopy.go`, `config/crd/bases/`,
`config/rbac/role.yaml`, `deploy/helm/dex-operator/files/crds/`. The chart's
`templates/clusterrole.yaml` is **not** generated; it is kept in step with
[internal/controller/rbac.go](../../internal/controller/rbac.go) by hand.

`cover.out`, `coverage/`, `bin/` and `tmp/` are local output and ignored.
