# Rotation and change propagation: when a change reaches Dex

What happens between a changed credential, allowlist or child and the moment Dex acts on it, and
why the operator restarts Dex only when the config changes. Where credentials go is
[secret-flow.md](secret-flow.md); the decision is
[ADR 0003](../adr/0003-the-render-is-deterministic-and-change-driven-no-periodic-requeue.md).

## A rotation re-renders at once

Every child kind is indexed by the Secrets it references (`SecretRefIndexField`, fed by
`GetReferencedSecretNames`); a created or updated Secret maps back to the installations whose
children reference it and re-renders the config and env Secrets
([internal/controller/secret_watch.go](../../internal/controller/secret_watch.go)
`mapSecretToInstallation`).

A Secret **deletion is deliberately not acted on** (`secretWatchPredicate`): during a rotation by
delete and recreate, reacting to the delete would produce a failing render between the two events;
the replacement's create event triggers the update.

## Dex restarts only on a config change

With `rolloutRestart.enabled` and a `deploymentName`, a *semantic* change of the config Secret
patches the Dex Deployment's pod-template annotation `kubectl.kubernetes.io/restartedAt`, forcing a
rolling restart so Dex reads the new `config.yaml`
([internal/controller/rollout.go](../../internal/controller/rollout.go),
[dexinstallation_controller.go](../../internal/controller/dexinstallation_controller.go)
`reconcileInstallation`). A change of the env Secret alone triggers no restart. Dex reads its
environment at start, so a rotated value that leaves `config.yaml` unchanged becomes active at the
next restart ([H-17](#h-17)).

## Determinism guards availability

Children are sorted by namespace and name before rendering, and the config Secret is compared as
parsed YAML, not bytes ([internal/controller/collect.go](../../internal/controller/collect.go),
[secret.go](../../internal/controller/secret.go)). Without this, list-order jitter or map-order
marshalling would make every reconcile look like a change and restart Dex in a loop — a
self-inflicted denial of service of the login of every relying party.

## What this does not cover

<a id="h-17"></a>
### H-17 — A rotated credential that changes only the env Secret is not active until Dex restarts

Live today, by decision (ADR 0003 D5). Rotating a static client's secret or a connector's client
secret updates the env Secret within one reconcile, but the running Dex pods keep the old value in
their environment: Dex keeps accepting the **old** client secret and keeps using the old upstream
credential until its pods restart. Rotating a leaked secret therefore does not revoke it by
itself. The same holds for every change while `rolloutRestart` is disabled
([H-5](tenancy.md#h-5)). Mitigation: after a rotation that must take effect, restart Dex
(`kubectl -n <dex-namespace> rollout restart deployment/<dex>`); see
[docs/operations/runtime.md](../operations/runtime.md#a-rotation-needs-a-restart).
