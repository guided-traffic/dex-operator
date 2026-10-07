---
id: T7
title: the operator's ClusterRole grants create and delete on dexinstallations, which it never uses
state: filed
severity: low
security: hardening
threat: a compromised operator ServiceAccount could additionally create and delete DexInstallations in every namespace; removing the verbs takes that away and leaves the operator unable to delete an installation's configuration source
urgency: later        # rule 4: a cheap known fix
effort: XS
blocked-by:
filed-from: the verification of the privilege footprint while moving the security architecture to docs/security/
opened: 2026-10-07
decided:
done:
shipped:
dropped-reason:
publication-accepted:
---

## Current state

The Helm chart's ClusterRole grants `create, delete, get, list, patch, update, watch` on
`dexinstallations` and `update` on `dexinstallations/finalizers`
([deploy/helm/dex-operator/templates/clusterrole.yaml](../../deploy/helm/dex-operator/templates/clusterrole.yaml));
the markers in [internal/controller/rbac.go:23-25](../../internal/controller/rbac.go#L23-L25) say
the same. Every other dex.gtrfc.com kind gets `get, list, patch, update, watch`.

The operator never creates, deletes or updates a `DexInstallation` and sets no finalizer: its only
write on the kind is `r.Status().Update`
([internal/controller/dexinstallation_controller.go:59](../../internal/controller/dexinstallation_controller.go#L59)).
The writes of the controller package are that status update, the child status update, the
Secret create and patch, and the Deployment patch (`grep -n '\.Create(\|\.Delete(\|\.Update(\|\.Patch('`
over `internal/controller/*.go` without the tests).

## Required changes

1. [internal/controller/rbac.go](../../internal/controller/rbac.go): `dexinstallations` gets
   `get;list;watch` (the controller reads and watches it; it neither updates nor patches the
   object itself), the `finalizers` marker goes.
2. `make generate-all` regenerates `config/rbac/role.yaml`; the Helm ClusterRole is edited to the
   same verbs (it is not generated).
3. [docs/security/privilege-footprint.md](../security/privilege-footprint.md): the table row and
   the gap close.
4. Verification: the integration suite (`make test-integration`) runs under envtest without RBAC,
   so it proves nothing here; the e2e suite (`make e2e-local`) runs the operator under the chart's
   ClusterRole and must stay green — it reconciles installations and writes their status.

## Not verified

Whether `update` and `patch` on the main resource (not `/status`) are needed for anything the
controller-runtime client does implicitly. Settled by the e2e run of step 4 with `get;list;watch`.
