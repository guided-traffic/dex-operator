# Upgrading

What happens when the operator is upgraded and what to check. The commands, and the behaviour
changes of individual releases that ask something of an installation, are in the README's
*Upgrade / Uninstall* section of the fast start.

## Before

1. **Read the release's notes and the README's upgrade notes.** A release that changes what is
   admitted or rendered says so there — `allowedConnectorNamespaces`, for example, changed which
   connectors are rendered on upgrade.
2. **Know whether `rolloutRestart` is on.** A release that changes the rendered config restarts
   Dex once if it is; if it is off, the new config waits for your restart.

## What `helm upgrade` does

1. **The CRDs are applied first.** A `pre-upgrade` hook Job applies every CRD of the new chart
   with `kubectl apply --server-side --force-conflicts`, before the new operator pod starts
   ([ADR 0007](../adr/0007-crds-ship-through-a-helm-hook-job-not-the-crds-directory.md)). Its
   ServiceAccount, ClusterRole, binding and CRD ConfigMap are hooks too and are deleted when the
   Job succeeds. The kubectl image is `crdUpgradeJob.image.*`.
2. **The operator pod is replaced.** On start it renders every installation; a release whose
   rendering is unchanged writes nothing and restarts nothing.
3. **New validation holds for new writes.** A CEL rule or schema change in the new CRDs is checked
   on the next create or update of an object; objects already stored are not re-validated.

## After

- `kubectl get dexinstallations -A` — every installation `READY` `True`.
- The children's `Ready` conditions — a release that changes admission shows refused children
  there, with the reason ([runtime.md](runtime.md#reading-the-conditions)).
- After a **failed** upgrade: the hook resources stay when the Job fails. Check that the
  `<release>-crd-upgrade` ServiceAccount, ClusterRole and ClusterRoleBinding are gone once the
  upgrade succeeds — the ClusterRole may write every CRD of the cluster
  ([docs/security/privilege-footprint.md](../security/privilege-footprint.md#h-15)).

## Without hooks

The hook cannot be switched off in the chart. An installation that must not run hook Jobs uses
`helm upgrade --no-hooks` and applies the CRDs of the new chart itself, from the chart's
`files/crds/`, before the upgrade.

## Uninstalling

`helm uninstall` removes the operator and its RBAC. The CRDs were applied by the hook, not
installed by Helm, so they stay — and with them every custom resource and the generated Secrets.
Dex keeps running on its last config.
