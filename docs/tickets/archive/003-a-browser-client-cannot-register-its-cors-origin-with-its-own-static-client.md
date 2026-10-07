---
id: T3
title: a browser client cannot register its CORS origin with its own DexStaticClient
state: done
severity: medium
security: none
threat:
urgency: later        # rule 4: a decided fix
effort: S
blocked-by:
filed-from: the first browser client (Dependency-Track's frontend) of a platform installation
opened: 2026-08-18
decided: 2026-08-18
done: 2026-08-18
shipped: 2.2.0 — spec.cors on a DexStaticClient derives the https origins of its own redirectURIs into web.allowedOrigins; the consumer follow-ups of §6 belong to other repositories and are not part of this repository's work
---

# CORS Origins from Static Clients

Plan for aggregating `web.allowedOrigins` dynamically from `DexStaticClient`
resources, so SPA clients can self-register their CORS origin instead of a
cluster operator maintaining a static list on the `DexInstallation`.

## 1. Motivation

Dex gates its browser-facing endpoints (discovery, token, keys) behind CORS.
Server-side OIDC clients never hit this; browser-XHR clients (SPAs doing
code+PKCE in the browser, first consumer: Dependency-Track's frontend) do.
Today the only knob is `DexInstallation.spec.web.allowedOrigins` — a static
list in the installation, i.e. platform-repo territory. But the origin is
*client* data: clients self-register via `DexStaticClient` from their own
namespace, and their CORS origin should travel with them. Without this,
every new SPA client needs a paired edit in the platform repo.

## 2. Research (verified)

- **Dex CORS matching is exact-string.** Dex v2.44.0 wraps the CORS-gated
  handlers with `gorilla/handlers.AllowedOrigins` (`server/server.go`,
  `handleWithCORS`, ~line 442). gorilla/handlers matches origins literally;
  the only wildcard is the full `"*"`. No subdomain globs
  (`https://*.example.com` would never match). Derived origins must therefore
  be exact `scheme://host[:port]` strings.
- **The listener is safe.** Deployments driven by the dexidp helm chart pass
  `--web-http-addr`/`--web-https-addr` flags, which the dex binary applies
  after config load. A rendered `web:` block that carries only
  `allowedOrigins` cannot break serving.
- **Builder gap.** `assembleDexConfig` (internal/builder/builder.go:143)
  only emits `cfg.Web` when `spec.Web != nil`. Derived origins must create
  the block when it is absent.
- **Reconcile path already exists.** `DexStaticClient` changes re-render the
  installation config (clients are part of the rendered `staticClients`), and
  `rolloutRestart` restarts dex on config change. Origin aggregation rides
  the same path — no new watches.

## 3. Design

Per-client **opt-in flag**; origins are **derived from the client's own
`redirectURIs`**, never free-form:

```yaml
apiVersion: dex.gtrfc.com/v1
kind: DexStaticClient
spec:
  public: true
  clientID: "mgmt-p-dependency-track"
  # opt-in: register this client's redirect-URI origins for CORS
  # (browser/SPA clients only)
  cors: true
  redirectURIs:
    - 'https://dtrack.example.com/static/oidc-callback.html'
```

Rendered result:

```yaml
web:
  allowedOrigins:
    - https://dtrack.example.com
```

Why a flag and not a free-form origin list: with `allowedNamespaces: "*"`
client registration is tenant self-service. A free-form field would need
validation against `"*"`, wildcards and foreign origins — and the safe rule
("must be the origin of one of your own redirectURIs") makes the field
redundant. The flag adds **no authority beyond `redirectURIs`**, which is
already the security-critical, RBAC-gated field. A tenant can only ever
allow-list origins it already controls as redirect targets.

Aggregation rules:

1. For every `DexStaticClient` with `cors: true`: parse each redirect URI,
   keep `scheme://host[:port]`.
2. Only `https` origins are taken. `http://localhost`/loopback, custom
   schemes and the OOB URN (native/CLI clients) are skipped silently — they
   are redirect conveniences, not browser origins.
3. Union with `DexInstallation.spec.web.allowedOrigins` (kept as base +
   escape hatch), **deduplicate, sort** — deterministic config, no spurious
   dex rollouts from map ordering.
4. Client deleted or flag removed → origin disappears on the next render
   (stateless derivation, self-healing).
5. A derived set with `spec.web == nil` creates the `web:` block (see
   builder gap above); static `"*"` in the installation list stays allowed
   (platform operator's explicit choice), derived origins can never produce
   it.

## 4. Implementation checklist

Follow the CRD-field checklist in `DEVELOPER.md`.

1. `api/v1/dexstaticclient_types.go` — add to `DexStaticClientSpec`:
   ```go
   // CORS registers this client's redirect-URI origins (https only) in the
   // installation's web.allowedOrigins. Enable for browser/SPA clients that
   // perform the code+PKCE flow via XHR. No effect on confidential
   // server-side clients.
   // +optional
   CORS bool `json:"cors,omitempty"`
   ```
   No CEL needed — boolean, derivation is constrained by design.
2. `make manifests generate` — CRD bases + deepcopy; sync the helm-chart CRD
   copy under `deploy/helm/dex-operator/files/crds/`.
3. `internal/builder/builder.go` — in `Build`, derive the origin set from
   `in.StaticClients` (spec-level, before `buildStaticClients` flattening);
   pass it into `assembleDexConfig` and union/sort/dedupe with
   `spec.Web.AllowedOrigins`. Create `cfg.Web` when absent but origins exist.
4. `internal/builder/clients.go` — no change (the flag is not part of the
   rendered `staticClients` entry; dex has no per-client CORS).
5. Tests (`internal/builder/builder_test.go`):
   - client with `cors: true` + https redirect URI → origin rendered, path
     and duplicates stripped, sorted;
   - `cors: true` with `http://localhost:8000/cb` + OOB URN → skipped;
   - union with installation-level list, overlap deduped;
   - `spec.web == nil` + derived origins → `web:` block present, no
     `http`/`https` keys (listener untouched);
   - `cors: false`/unset → nothing derived (regression guard for existing
     clients);
   - determinism: two builds over the same input render byte-identical YAML.
6. Docs per project standard: README naming/reference section for the new
   field; SECURITY_ARCHITECTURE.md — one paragraph: flag grants nothing
   beyond redirectURIs; CORS remains defense-in-depth (the token endpoint
   has no ambient credentials), exact-match only.

## 5. Out of scope

- Subdomain wildcards — dex/gorilla cannot match them (see Research).
- A free-form per-client origin list (rejected, see Design).
- `userinfo` CORS specifics and dex `master`'s router rework — revisit when
  the deployed dex moves past v2.44.

## 6. Consumer follow-ups (after release)

- k8s-flux-base: bump the dex-operator chart (currently 2.1.1); drop the
  `https://dtrack.${internal_domain_name}` entry from
  `apps/iam/dex/app/dexInstallation.yml` `spec.web.allowedOrigins` once the
  client below carries the flag (the block itself can stay as escape hatch).
- k8s-flux-mgmt: set `cors: true` in
  `components/dependency-track/dependency-track/oidc-credentials-dependency-track.yml`.
