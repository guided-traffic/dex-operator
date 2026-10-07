# The Helm chart and the CRD lifecycle

What the chart in [deploy/helm/dex-operator/](../../deploy/helm/dex-operator/) installs, how its
CRDs stay current, and which of its files are generated. What an operator does with it is
[docs/operations/](../operations/README.md); the permissions it grants are
[docs/security/privilege-footprint.md](../security/privilege-footprint.md).

## CRDs ship through a hook, not `crds/`

The chart does **not** use Helm's `crds/` directory, which Helm installs once and never upgrades
([ADR 0007](../adr/0007-crds-ship-through-a-helm-hook-job-not-the-crds-directory.md)). Instead:

- The CRD manifests live in `files/crds/`, copied from `config/crd/bases/` by
  `make helm-sync-crds` (part of `make manifests` and `make generate-all`).
- `crd-configmap.yaml` renders every file of `files/crds/` into a ConfigMap — a hook of weight
  `-10`, like the ServiceAccount, ClusterRole and binding of `crd-upgrade-rbac.yaml`.
- `crd-upgrade-job.yaml`, a `pre-install,pre-upgrade` hook of weight `0`, mounts that ConfigMap and
  runs `kubectl apply --server-side --force-conflicts` on each file, with the image from
  `crdUpgradeJob.image.*` (`guidedtraffic/kubectl`). Every hook resource is deleted before the next
  hook run and after success.

So CRD updates ship automatically with every `helm upgrade`, before the new operator pod starts.
The Makefile reads the kubectl image from `values.yaml` for `make e2e-local`, so a Renovate bump of
the tag reaches the local e2e run too.

## The templates

| Template | What it is |
|---|---|
| `deployment.yaml` | the manager: `--health-probe-bind-address`, `--leader-elect` when `leaderElect`, the pod and container security contexts from the values |
| `serviceaccount.yaml` | the operator's ServiceAccount |
| `clusterrole.yaml`, `clusterrolebinding.yaml` | the operator's cluster permissions — **written by hand**, kept in step with the markers in [internal/controller/rbac.go](../../internal/controller/rbac.go) |
| `leader-election-role.yaml`, `leader-election-rolebinding.yaml` | ConfigMaps, Leases and Events in the release namespace |
| `crd-configmap.yaml`, `crd-upgrade-rbac.yaml`, `crd-upgrade-job.yaml` | the CRD hook above |
| `_helpers.tpl` | names and labels |

The chart's `version` is set from the release tag by the release workflow
([ci-and-release.md](ci-and-release.md)).

## Generated vs. hand-written

| Path | Source |
|---|---|
| `files/crds/*.yaml` | generated: `api/v1` → `config/crd/bases/` → `make helm-sync-crds` |
| `templates/clusterrole.yaml` | hand-written; `config/rbac/role.yaml` is generated from the same markers and serves only the kustomize deployment |
| everything else | hand-written |

A new CRD or a new verb therefore needs an edit of `templates/clusterrole.yaml` beside the marker
([adding-things.md](adding-things.md)); the e2e suite, which installs the chart, is what catches a
missing one.
