# ADR 0007: CRDs Ship Through a Helm Hook Job, Not Through the `crds/` Directory

## Status

Accepted. Date: 2026-10-07, recording the chart's design as it stands; the release that
introduced it is not recorded here.

**Implemented.**

## Context

Helm installs the manifests in a chart's `crds/` directory on the first install and never again:
`helm upgrade` neither updates nor deletes them. Every release of this operator that adds a field
— a new connector option, `cors`, `allowedConnectorNamespaces` — needs the new schema in the
cluster before the new operator writes or reads it, and validation rules (CEL, `MinItems`) only
hold once the CRD carries them.

## Decision

**D1 — The CRD manifests live in the chart's `files/crds/`,** copied from
`config/crd/bases/` by `make helm-sync-crds`, which `make manifests` and `make generate-all` run.
`config/crd/bases/` is generated from `api/v1` and is the source of truth.

**D2 — A pre-install and pre-upgrade hook Job applies them.** The chart renders the files into a
ConfigMap; the Job (`crd-upgrade-job.yaml`, hook weight `0`) runs `kubectl apply --server-side
--force-conflicts` on each, with the image from `crdUpgradeJob.image.*`. Its ServiceAccount,
ClusterRole and binding are hooks of weight `-10`, deleted when the hook succeeds
([deploy/helm/dex-operator/templates/](../../deploy/helm/dex-operator/templates/)).

**D3 — The hook cannot be switched off in the chart.** There is no value that disables it;
`helm --no-hooks` skips it together with every other hook.

## Consequences

- CRD updates ship with every `helm upgrade`, before the new operator pod starts.
- During the hook the release holds a ServiceAccount that may create, patch and update every
  `CustomResourceDefinition` in the cluster, not only this operator's
  ([docs/security/privilege-footprint.md](../security/privilege-footprint.md)).
- A cluster where CRDs must be applied by someone else (a GitOps controller with its own CRD
  handling, a policy that forbids hook Jobs) has to skip all hooks and apply the files itself.
- Uninstalling the chart leaves the CRDs, and with them every custom resource, in place.

## Alternatives Considered

- **Helm's `crds/` directory.** Never upgrades. Lost.
- **CRDs as ordinary templates.** Helm would upgrade them, and also delete them — and every
  custom resource with them — on `helm uninstall`. Not chosen; the reason is not recorded.
- **The operator applies its own CRDs at start.** Needs the CRD write permission in the
  long-running operator instead of a short-lived hook. Not chosen; the reason is not recorded.

## Residual risks

- Not verified: what the hook does when a CRD change is not server-side-apply compatible with an
  object in the cluster; `--force-conflicts` takes over field ownership, it does not migrate stored
  objects.

## References

- [deploy/helm/dex-operator/templates/crd-upgrade-job.yaml](../../deploy/helm/dex-operator/templates/crd-upgrade-job.yaml),
  [crd-upgrade-rbac.yaml](../../deploy/helm/dex-operator/templates/crd-upgrade-rbac.yaml),
  [crd-configmap.yaml](../../deploy/helm/dex-operator/templates/crd-configmap.yaml)
- [Makefile](../../Makefile) `manifests`, `helm-sync-crds`, `generate-all`
- [docs/developer/helm-chart.md](../developer/helm-chart.md)
