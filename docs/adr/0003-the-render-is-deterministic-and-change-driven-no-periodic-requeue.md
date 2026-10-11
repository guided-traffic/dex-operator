# ADR 0003: The Render Is Deterministic and Change-Driven — Sorted Children, a Semantic Config Comparison, Watched Secrets, No Periodic Requeue

## Status

Accepted, amended 2026-10-07 (a consequence: the claim of a client whose Secret does not resolve
comes from its own status). Date: 2026-10-07, recording decisions built in three steps: the Secret
watch and the semantic comparison of the config Secret in 1.1.4 (2026-04-07), the deletions
ignored in 1.2.0 (2026-04-07), the sorted children in 2.0.3 (2026-07-21).

**Implemented.**

## Context

A rotated credential did not reach Dex: the operator watched only its own kinds, so a change to a
referenced Secret produced no reconcile and a GitLab login failed with "invalid client
credentials" until something else touched the installation. At the same time every reconcile
restarted Dex: `map[string]any` parts of the config marshalled in a different key order each
time, the byte comparison saw a change, and the rollout restart fired. Later the same symptom came
back from the other side: the cache-backed `List` returns children in map order, so the rendered
`connectors` and `staticClients` reordered from pass to pass.

A restart loop is a self-inflicted outage of the login of every relying party; a missed
rotation is an outage of one. Both come from the operator not knowing when something actually
changed.

## Decision

**D1 — Children are sorted before rendering.** `filterItems` sorts every collected list by
namespace, then name (`sortByNamespaceName`,
[internal/controller/collect.go](../../internal/controller/collect.go)). The render of the same
input is byte-identical, whatever order the cache returns.

**D2 — The config Secret is compared as parsed YAML.** `applySecret` takes the comparison as a
function; the config Secret uses `yamlSecretDataEqual` (parse, `reflect.DeepEqual`, byte
comparison only when a value does not parse), the env Secret `secretDataEqual`
([internal/controller/secret.go](../../internal/controller/secret.go)). `applySecret` reports a
change only when the data differs under that comparison.

**D3 — Referenced Secrets are watched.** Every child kind is indexed by the names of the Secrets
it references (`SecretRefIndexField`, fed by `GetReferencedSecretNames`); a created or updated
Secret maps to the installations whose children reference it (`mapSecretToInstallation`,
[internal/controller/secret_watch.go](../../internal/controller/secret_watch.go)).

**D4 — A Secret deletion triggers nothing.** `secretWatchPredicate` passes create, update and
generic events and drops deletes. During a rotation by delete and recreate, the replacement's
create event triggers the render; reacting to the delete would only produce a failing reconcile
between the two events.

**D5 — A rollout restart follows a config change and nothing else.** When `applySecret` reports a
changed config Secret and `rolloutRestart.enabled` names a Deployment, the operator patches the
Deployment's pod-template annotation `kubectl.kubernetes.io/restartedAt`
([internal/controller/rollout.go](../../internal/controller/rollout.go)). A change of the env
Secret alone does not restart Dex.

**D6 — There is no periodic requeue of the installation.** Renders are triggered by changes to the
installation, its children and the Secrets they reference, never by a timer. (A child with a
configuration error is requeued after five minutes to refresh its own status; that renders
nothing.)

## Consequences

- Rotating a credential re-renders the env Secret within one reconcile, but Dex reads its
  environment at start: a rotated value that changes nothing in `config.yaml` becomes active at
  the next Dex restart, which D5 does not trigger.
- An operator upgrade that changes nothing about the rendered config causes no Dex rollout; a
  change to the rendering that reorders existing output would cause exactly one. Changes to the
  builder have to keep existing output stable (ADR 0005 D3 is an instance).
- A drifted Secret that nobody touches — edited back by hand to an old value without an event the
  watch sees — stays drifted until the next event. D6 accepts that.
- The render reads one piece of state it wrote itself: the claim of a confidential client whose
  Secret does not resolve comes from that client's `status.clientID`
  ([ADR 0008](0008-an-id-renders-for-one-child-only-and-a-failing-child-is-left-out-instead-of-failing-the-render.md)
  D4). It counts only while the Secret does not resolve, so every other render stays a function of
  specs and Secrets; D1 is unchanged — which child holds an ID never depends on the order of the
  children.

## Alternatives Considered

- **A periodic `RequeueAfter` as a safety net.** Would also heal missed events, but restarts must
  come from actual changes to Secrets or the config, not from polling. Lost.
- **Typed structs instead of `map[string]any` for connector and storage config.** Makes the bytes
  stable at the source. Offered as the alternative and not chosen; the reason is not recorded —
  the semantic comparison fixes the restart loop at one point, typed structs would model every
  connector kind's schema. Still compatible with D2.
- **React to Secret deletions.** Produces a failing render in the middle of every delete-and-
  recreate rotation. Lost.
- **Restart Dex on env Secret changes too.** Would make rotations take effect at once; not built.
  It is the missing half of D5, recorded as a gap in
  [docs/security/rotation.md](../security/rotation.md).

## Residual risks

- D5 leaves a rotated credential inactive until the next restart; with `rolloutRestart` disabled,
  every change waits for one.
- Not verified: the behaviour under a Secret watch event storm (many Secrets referenced by many
  children); each event enqueues the affected installations, deduplicated per event only.

## References

- [internal/controller/collect.go](../../internal/controller/collect.go),
  [secret.go](../../internal/controller/secret.go),
  [secret_watch.go](../../internal/controller/secret_watch.go),
  [rollout.go](../../internal/controller/rollout.go),
  [child_reconciler.go](../../internal/controller/child_reconciler.go) `configRequeueInterval`
- [internal/controller/collect_determinism_test.go](../../internal/controller/collect_determinism_test.go),
  [render_compat_test.go](../../internal/controller/render_compat_test.go)
- [ADR 0002](0002-custom-resources-carry-no-secret-values-credentials-reach-dex-through-an-env-secret.md)
