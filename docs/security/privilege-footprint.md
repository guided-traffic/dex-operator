# Privilege footprint: what the operator and its chart may do in the cluster

Every Kubernetes permission the Helm chart grants — to the operator, to its leader election, and
to the hook Job that applies the CRDs — and what each one means if the holder is compromised. Why
those permissions are needed in the first place is [trust-boundaries.md](trust-boundaries.md) and
[secret-flow.md](secret-flow.md).

## The operator's ClusterRole

Granted by [deploy/helm/dex-operator/templates/clusterrole.yaml](../../deploy/helm/dex-operator/templates/clusterrole.yaml)
to the operator's ServiceAccount through a ClusterRoleBinding. The chart's file is written by hand;
the kubebuilder markers in [internal/controller/rbac.go](../../internal/controller/rbac.go)
generate `config/rbac/role.yaml` for the kustomize deployment and say the same.

| Resource | Verbs | Why |
|---|---|---|
| `secrets` (cluster-wide) | get, list, watch, **create, update, patch** — **no delete** | read the credentials children reference in their own namespaces, write the config and env Secrets |
| `deployments` (apps) | get, list, watch, **patch** | the rollout-restart annotation only |
| the sixteen connector kinds and `dexstaticclients` | get, list, watch, update, patch | collect children; the operator writes only their status |
| their `/status` and `dexinstallations/status` | get, update, patch | the `Ready` condition, the counts |
| `dexinstallations` | get, list, watch, **create, update, patch, delete** | get, list and watch are used; the operator never creates, updates or deletes an installation ([H-14](#h-14)) |
| `dexinstallations/finalizers` | update | unused — the operator sets no finalizer |

The operator's own writes are, in full: the status update of an installation and of a child, the
create and patch of a generated Secret, and the patch of the Dex Deployment
([internal/controller/](../../internal/controller/) `Status().Update`, `applySecret`,
`triggerRolloutRestart`).

**Consequences:**

- Cluster-wide Secret read is the operator's most sensitive permission. It is inherent to the
  design — tenants reference Secrets in their own namespaces — and it reaches every Secret of the
  cluster, not only referenced ones ([H-2](trust-boundaries.md#h-2)).
- Cluster-wide Secret **write** means a compromised operator can overwrite any Secret it can name,
  not only the two it generates.
- The operator never deletes a Secret. That makes it incapable of destroying credentials, and
  leaves generated Secrets behind after a rename ([H-16](#h-16)).
- It has no permission on `namespaces`, so it cannot look one up
  ([tenancy.md](tenancy.md#exact-names-and-why)).

## Leader election

A Role in the release namespace
([leader-election-role.yaml](../../deploy/helm/dex-operator/templates/leader-election-role.yaml))
grants `configmaps` and `coordination.k8s.io/leases` (all verbs including delete) and `events`
(create, patch) — used by controller-runtime's leader election (`leaderElect: true` by default).
It reaches nothing outside the operator's own namespace.

## The CRD upgrade hook

For the pre-install and pre-upgrade hook Job
([ADR 0007](../adr/0007-crds-ship-through-a-helm-hook-job-not-the-crds-directory.md)), the chart
creates a ServiceAccount, a ClusterRole and a binding as hooks
([crd-upgrade-rbac.yaml](../../deploy/helm/dex-operator/templates/crd-upgrade-rbac.yaml)):
`customresourcedefinitions` — create, get, list, patch, update, watch — cluster-wide. They are
deleted when the hook succeeds (`hook-delete-policy: before-hook-creation,hook-succeeded`)
([H-15](#h-15)).

## How the pods run

Chart defaults ([deploy/helm/dex-operator/values.yaml](../../deploy/helm/dex-operator/values.yaml)):
the operator pod runs as non-root with seccomp `RuntimeDefault`, its container drops all
capabilities and disallows privilege escalation. The hook Job runs as user `65534`, non-root,
seccomp `RuntimeDefault`, all capabilities dropped, its CRD files mounted read-only. The operator's
metrics endpoint is off unless `--metrics-bind-address` is passed, which the chart does not do
([cmd/main.go](../../cmd/main.go)); the health probes listen on `healthProbeBindAddress`.

## What this does not cover

<a id="h-14"></a>
### H-14 — The ClusterRole grants create and delete on `dexinstallations`, which the operator never uses

Live today, hardening. A stolen operator token can additionally create and delete
`DexInstallation`s in every namespace; deleting one stops every further render for it (its
generated Secrets carry no owner reference and stay, so Dex keeps its last config). Neither verb
is needed. Mitigation until the chart narrows them: an admission policy that denies these verbs to
the operator's ServiceAccount.

<a id="h-15"></a>
### H-15 — The CRD hook holds cluster-wide write on every CRD while it runs

Live during every `helm install` and `helm upgrade`. The hook's ClusterRole is not limited to this
operator's CRDs (`resourceNames` is not set), so for the hook's lifetime — and longer if it fails,
because the delete policy removes it only on success — a ServiceAccount in the release namespace
may create and rewrite any CRD of the cluster. Whoever can run a pod with that ServiceAccount in
the meantime holds that power. Mitigation: keep the release namespace closed to tenants, and check
after a failed upgrade that the `-crd-upgrade` ServiceAccount, ClusterRole and binding are gone.

<a id="h-16"></a>
### H-16 — A renamed config or env Secret stays behind with its credentials

Live after every change of `configSecretName` or `envSecretName`. The operator writes the new
Secret and never deletes the old one, so the old env Secret keeps every credential it held — never
rotated again, since the operator no longer writes it. Mitigation: delete the old Secrets by hand
after a rename; they carry the labels `app.kubernetes.io/managed-by: dex-operator` and
`dex.gtrfc.com/installation: <name>`.
