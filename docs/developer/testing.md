# Testing

Three tiers, each with its own build tag and its own target ([build-test-lint.md](build-test-lint.md)).
New features and bug fixes come with tests; reconciliation logic is the coverage priority. A fix
comes with the test that failed without it.

| Tier | Where | Target | Needs |
|---|---|---|---|
| Unit | `internal/...` | `make test-unit` | nothing but the envtest binaries the target downloads |
| Integration | `test/integration/` (tag `integration`) | `make test-integration` | the same; envtest starts an API server and etcd |
| E2E | `test/e2e/` (tag `e2e`) | `make e2e-local`, or `make test-e2e` against a prepared cluster | Docker, Kind, Helm |

## Unit

Builder rendering, env naming, determinism, the namespace rules — no cluster.

- [internal/builder/builder_test.go](../../internal/builder/builder_test.go) — the rendering of
  every kind. Helpers: `minimalInstallation(namespace)`, `mockResolver(secrets)`, `parseYAML`.
- [internal/builder/builder_dexschema_test.go](../../internal/builder/builder_dexschema_test.go) —
  pins the rendered keys to Dex's config schema; `assertNoKeys` asserts a key is **absent**, the
  pattern for "this field must not be rendered".
- [internal/controller/namespace_test.go](../../internal/controller/namespace_test.go) — both
  allowlists, their defaults and `"*"`.
- [internal/controller/collect_determinism_test.go](../../internal/controller/collect_determinism_test.go) —
  `filterItems` sorts by namespace and name, stays stable across shuffled input, denies all on an
  empty `allowedNamespaces`, and sorts connectors admitted by the default list.
- [internal/controller/render_compat_test.go](../../internal/controller/render_compat_test.go) —
  an installation whose connectors live in its own namespace renders byte-identical config to the
  unfiltered render for several `allowedNamespaces` values: the upgrade promise of
  `allowedConnectorNamespaces`. The pattern for any change that must not alter existing output —
  an altered render is a config diff, and a Dex restart, in every installation on upgrade.
- [internal/controller/dexinstallation_controller_test.go](../../internal/controller/dexinstallation_controller_test.go),
  [child_reconciler_test.go](../../internal/controller/child_reconciler_test.go),
  [helpers_test.go](../../internal/controller/helpers_test.go) — controller logic against a fake
  client.

`export_test.go` in both packages exports internals to the external test packages.

## Integration (envtest)

A real API server, so the CRD schema, CEL rules and `MinItems` are enforced. Reconciler behaviour,
the namespace allowlists, Secret generation, the CEL validation of `DexStaticClient`
([suite_test.go](../../test/integration/suite_test.go) sets up the manager, the
`DexInstallationReconciler` before the child reconcilers):

- [dexinstallation_test.go](../../test/integration/dexinstallation_test.go) — Secrets, status,
  `allowedNamespaces`
- [connector_test.go](../../test/integration/connector_test.go),
  [connector_namespaces_test.go](../../test/integration/connector_namespaces_test.go) — connectors
  and `allowedConnectorNamespaces`, including the child status following an allowlist edit
- [staticclient_test.go](../../test/integration/staticclient_test.go) — confidential, public and
  invalid clients, derived CORS origins

envtest runs no RBAC: a missing verb in the chart's ClusterRole is not caught here.

## E2E (Kind)

The chart installed into a Kind cluster with [test/e2e/helm-values.yaml](../../test/e2e/helm-values.yaml),
the operator running under the chart's real RBAC, the pre-install hook applying the CRDs. Tests in
[dexinstallation_e2e_test.go](../../test/e2e/dexinstallation_e2e_test.go):
`TestE2E_MinimalInstallation`, `TestE2E_OIDCConnector`, `TestE2E_StaticClient`,
`TestE2E_NamespaceIsolation`, `TestE2E_ConnectorNamespaceDefault`, `TestE2E_ConnectorLifecycle`.

`make test-e2e-helm` runs `-run TestE2E_Migrate` with the tags `e2e,e2e_helm`, but no file in the
tree carries the `e2e_helm` tag or defines `TestE2E_Migrate`: the target runs no test. There is no
Helm migration test today.

In CI the e2e job builds the image, loads it and the kubectl image into a Kind cluster and installs
the chart the same way ([ci-and-release.md](ci-and-release.md)).
