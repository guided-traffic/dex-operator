---
id: T1
title: a rotated Secret does not reach dex and every reconcile restarts dex
state: done
severity: high
security: none
threat:
urgency: next         # rule 3: severity high, trigger live in every installation that rotates a credential
effort: S
blocked-by:
filed-from: a failed GitLab login after a credential rotation
opened: 2026-04-07
decided: 2026-04-07
done: 2026-04-07
shipped: 1.1.4 (referenced Secrets watched through a field index, the config Secret compared as parsed YAML) and 1.2.0 (Secret deletions ignored)
---

## Current state

After the Secret `oidc-credentials-gitlab` in the namespace `gitlab` was rotated, the env Secret
`dex-env` in the namespace `iam` was not updated in time. The OIDC login to GitLab failed with:

```
Could not authenticate you from OpenIDConnect because "Invalid client :: invalid client credentials.".
```

The analysis found two causes:

1. The operator watches no `corev1.Secret` resources — a Secret rotation triggers no reconcile.
2. The generated config YAML is not deterministic (`map[string]any` → random key order), so every
   reconcile sees a false `configChanged` diff and triggers a needless rollout restart.

## Required changes

### Measure 1: watch the referenced Secrets

**Problem:** the DexInstallation controller watches only CRDs (`DexStaticClient`, every connector
kind). Changes to the Kubernetes Secrets referenced by `secretRef`, `clientSecretRef`, `bindPWRef`
and the like produce no event.

**Solution:**

- Add `Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(mapSecretToInstallation))` to
  the controller builder.
- `mapSecretToInstallation` must:
  1. find every `DexStaticClient` in the Secret's namespace that references this Secret through
     `secretRef.name`;
  2. find every connector in the Secret's namespace that references this Secret through
     `clientSecretRef.name`, `bindPWRef.name`, `rootCARef.name` and the like;
  3. resolve each one's `installationRef` and return it as a `ctrl.Request`.
- Register field indexers on the Secret reference fields of the CRDs for that (for example
  `.spec.secretRef.name` for `DexStaticClient`).

**Result:** a Secret rotation reconciles the affected DexInstallation at once → `dex-env` is
updated, and a rollout restart follows when the data actually changed.

### Measure 2: a deterministic config YAML

**Problem:** `ConnectorEntry.Config` and `StorageConfig.Config` are `map[string]any`. Go maps have
no guaranteed iteration order, so `yaml.Marshal` can produce different bytes on every call. As a
result:

- `secretDataEqual()` sees a "change" on every reconcile
- `configChanged` is almost always `true`
- every reconcile triggers a rollout restart (phantom restart)

**Solution:** compare the existing and the new config semantically (deserialise the YAML →
deep-equal) before the byte comparison in `applySecret`, instead of a raw `bytes.Equal`.
Alternatively, replace the `map[string]any` fields with typed structs that guarantee a stable
serialisation order.

**Result:** rollout restarts happen only on actual config changes. No needless pod restarts.

### Deliberately not done: a periodic requeue

A `RequeueAfter` as a safety net is deliberately **not** implemented. Restarts are to be triggered
only by actual changes to Secrets or to the config, not by periodic polling.
