# Architecture

What runs, which controller owns which output, how a change of any input reaches the rendered
config, and why the render must be deterministic. Where each function lives is
[package-map.md](package-map.md); what the builder does inside `Build` is
[builder.md](builder.md).

## Two controller kinds

1. **`DexInstallationReconciler`**
   ([dexinstallation_controller.go](../../internal/controller/dexinstallation_controller.go)) owns
   the output. On every reconcile it collects all children of the installation, calls the builder
   and applies the config and env Secrets.
2. **`GenericChildReconciler[T, U]`** ([child_reconciler.go](../../internal/controller/child_reconciler.go)) —
   one generic instance per child kind (16 connectors + `DexStaticClient`, registered by
   `SetupConnectorControllers` in [connector_controller.go](../../internal/controller/connector_controller.go)).
   It only validates — the installation exists, the child's namespace is admitted — and maintains
   the child's `Ready` condition. It does **not** build anything. Which allowlist applies depends
   on the kind: `DexStaticClient` uses `allowedNamespaces`, every connector kind the effective
   connector list (`connectorNamespaces`: `allowedConnectorNamespaces`, or only the installation's
   own namespace when omitted). Both live as pure functions in
   [namespace.go](../../internal/controller/namespace.go) (`checkChildNamespace`); the rule is
   [ADR 0006](../adr/0006-two-namespace-allowlists-one-per-child-category-exact-names-only.md).

## How a change reaches the render

Config regeneration always goes through the installation reconciler, via watches:

- **Every child kind** is watched and mapped to its `spec.installationRef`
  (`mapChildToInstallation`).
- **Referenced Secrets** are watched through a field index (`SecretRefIndexField`, fed by
  `GetReferencedSecretNames`); create and update events map back to the installations whose
  children reference the Secret (`mapSecretToInstallation` in
  [secret_watch.go](../../internal/controller/secret_watch.go)). Secret **deletes are ignored on
  purpose** — during a rotation the replacement Secret triggers the reconcile; reacting to the
  delete would only produce a failing loop.

The reverse direction exists for child status only: every `GenericChildReconciler` also watches
`DexInstallation` and enqueues the children of its kind that reference it
(`mapInstallationToChildren`, through the `InstallationRefIndexField` index). Without it, a child
keeps a stale `Ready` condition after an allowlist edit until its own next event. The watch passes
only generation changes (spec edits), creates and deletes (`GenerationChangedPredicate`), so the
installation's status updates do not fan out to all children. The index is registered by
`DexInstallationReconciler.SetupWithManager`, so that controller must be set up first —
[cmd/main.go](../../cmd/main.go) and the integration suite do.

## The reconcile of an installation

```
Reconcile
 ├─ collectConnectors / collectStaticClients    (collect.go)
 │    ├─ List via field index .spec.installationRef == "<ns>/<name>"
 │    ├─ filterItems: namespace allowlist
 │    │    connectors:     connectorNamespaces (omitted = own namespace only)
 │    │    static clients: allowedNamespaces   (empty = deny all, "*" = all)
 │    └─ sortByNamespaceName: deterministic order (see below)
 ├─ builder.Build(Input)                         (internal/builder)
 │    └─ resolves SecretKeyRefs via the injected SecretResolver
 ├─ applySecret(configSecretName, config.yaml)   (secret.go, compared as parsed YAML)
 ├─ applySecret(envSecretName, env map)          (compared byte-wise)
 ├─ if the config changed → triggerRolloutRestart (rollout.go)
 └─ status: connectorCount, staticClientCount, Ready condition (status.go)
```

Both Secrets are written into the installation's own namespace with the labels
`app.kubernetes.io/managed-by: dex-operator` and `dex.gtrfc.com/installation: <name>`, and without
an owner reference. A build error lands on the installation's `Ready` condition.

## Determinism matters

The cache-backed `List` returns items in map order. Without the stable sort, the rendered YAML
would reorder `connectors` and `staticClients` on every pass, the change detection would fire every
time, and `rolloutRestart` would bounce the Dex Deployment in a loop. The same reasoning drives
`yamlSecretDataEqual` in [secret.go](../../internal/controller/secret.go): the config Secret is
compared *semantically* (parsed YAML, `reflect.DeepEqual`), not byte-wise. There is no periodic
requeue of the installation. The rules are
[ADR 0003](../adr/0003-the-render-is-deterministic-and-change-driven-no-periodic-requeue.md);
[collect_determinism_test.go](../../internal/controller/collect_determinism_test.go) and
[render_compat_test.go](../../internal/controller/render_compat_test.go) hold them.

**A change to the builder must keep existing output byte-identical** unless the change is meant
to alter it: any reordering or reformatting is a config diff in every installation on upgrade,
and with `rolloutRestart` a Dex restart.

## Child errors

Child resources with user-caused problems — a missing installation, a forbidden namespace —
return a `configError`, which is reported through the `Ready` condition and requeued after five
minutes (`configRequeueInterval`) instead of entering exponential backoff
([child_reconciler.go](../../internal/controller/child_reconciler.go)).

## The ChildObject interface

[interfaces.go](../../internal/controller/interfaces.go) defines the contract every child CRD
implements (in [api/v1/connector_helpers.go](../../api/v1/connector_helpers.go)):

```go
type ChildObject interface {
    GetInstallationRef() dexv1.InstallationRef
    GetCommonStatus() *dexv1.CommonStatus
    GetReferencedSecretNames() []string   // feeds the Secret watch index
}
```

This is what lets one generic reconciler, one indexer pair and one secret-watch mapper serve
seventeen CRDs. `GetReferencedSecretNames` must list every Secret the spec can reference, or a
rotation of the missing one goes unnoticed.

## What runs where

One Deployment, the manager, in the release namespace (`dex-operator-system` in the README),
`replicaCount: 1` with leader election on by default. It watches all namespaces. The health
probes listen on `:8081` (`/healthz`, `/readyz`); the metrics endpoint is off unless
`--metrics-bind-address` is passed. Dex runs elsewhere, installed by its own chart; the operator
touches only its two Secrets and, optionally, its Deployment's restart annotation.
