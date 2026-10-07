---
id: T2
title: a public static client cannot be created without a client secret
state: done
severity: medium
security: none
threat:
urgency: later        # rule 4: a decided fix
effort: M
blocked-by:
filed-from: the support of dex for secretless public clients since v2.24.0
opened: 2026-07-26
decided: 2026-07-26
done: 2026-07-26
shipped: 2.1.0 — an inline clientID for public clients, secretRef optional, three CEL rules on DexStaticClientSpec
---

# Secretless (Public) Static Clients

Plan for supporting Dex static clients without a client secret (public clients, PKCE flow).

## 1. Research: Does Dex support secretless clients?

**Yes.** Dex supports static clients with `public: true` and no secret since **v2.24.0**
(commit `c830d498` "allow no secret for static public clients", closing
[dexidp/dex#1894](https://github.com/dexidp/dex/issues/1894)).

Verified against current Dex `master`:

- Config validation in `cmd/dex/serve.go` only requires a secret for confidential clients:

  ```go
  if client.Secret == "" && client.SecretEnv == "" && !client.Public {
      return fmt.Errorf("invalid config: Secret or SecretEnv field is required for client %q", client.ID)
  }
  ```

- Redirect URI handling (`server/authflow/request.go`, `validateRedirectURI`):
  - Explicitly listed `redirectURIs` are always honored, for public and confidential
    clients alike ("required to make PKCE-enabled web apps work when configured as
    public clients").
  - A public client with an **empty** `redirectURIs` list additionally accepts
    `http://localhost:<any-port>` / loopback hosts, the OOB URN
    (`urn:ietf:wg:oauth:2.0:oob`), and the device-flow callback (`/device/callback`).
    This is the native-app / CLI use case.

- PKCE (`S256`) is supported for the authorization code flow; public clients are the
  intended consumers. Dex `master` (post-v2.45.1, unreleased) additionally adds an
  `oauth2.pkce` config block (`enforce`, `codeChallengeMethodsSupported`) via
  [dexidp/dex#4638](https://github.com/dexidp/dex/pull/4638) — see "Out of scope".

Example of what the generated `config.yaml` must contain:

```yaml
staticClients:
  - id: my-cli
    public: true
    name: 'My CLI'
    redirectURIs:            # may be empty for public clients (loopback/device defaults)
      - 'http://127.0.0.1:8085/callback'
```

**Caveat:** the device flow token exchange currently still demands a client secret for
confidential clients and has known compliance gaps
([dexidp/dex#3983](https://github.com/dexidp/dex/issues/3983)); public clients are the
supported path for device flow.

## 2. Current state in this operator

- `DexStaticClientSpec` **already has** `Public bool` ([api/v1/dexstaticclient_types.go:48](../../../api/v1/dexstaticclient_types.go#L48)),
  and the builder already copies it into the config
  ([internal/builder/clients.go:96-105](../../../internal/builder/clients.go#L96-L105),
  `StaticClient.Public` in [internal/builder/config_types.go:110-118](../../../internal/builder/config_types.go#L110-L118)).
- But the feature is unusable secretless, because:
  1. `secretRef` is `+kubebuilder:validation:Required` and a non-pointer struct
     ([api/v1/dexstaticclient_types.go:30](../../../api/v1/dexstaticclient_types.go#L30)); the CRD
     enforces `required: [... secretRef]`
     ([config/crd/bases/dex.gtrfc.com_dexstaticclients.yaml:119-123](../../../config/crd/bases/dex.gtrfc.com_dexstaticclients.yaml#L119-L123)).
  2. The **client ID** is only obtainable from the referenced Secret (`client-id` key) —
     there is no inline ID field.
  3. `buildOneStaticClient` resolves the `client-secret` key **unconditionally**
     ([internal/builder/clients.go:87-94](../../../internal/builder/clients.go#L87-L94)); a missing
     secret fails the whole DexInstallation build.
  4. `redirectURIs` has `MinItems=1`, so the "public client with Dex loopback defaults"
     mode cannot be expressed.

## 3. API design (CRD changes)

File: [api/v1/dexstaticclient_types.go](../../../api/v1/dexstaticclient_types.go)

```go
// ClientID is the OAuth2 client ID, set inline. Mutually exclusive with
// secretRef. Intended for public (secretless) clients whose ID is not
// confidential.
// +optional
ClientID string `json:"clientID,omitempty"`

// SecretRef references a Secret holding client-id and client-secret.
// Required for confidential clients. Becomes a pointer:
// +optional
SecretRef *StaticClientSecretRef `json:"secretRef,omitempty"`

// RedirectURIs: drop Required + MinItems=1, keep +optional.
// Public clients with an empty list get Dex's loopback/OOB/device defaults.
```

Conditional rules via **CEL validation markers** on the spec struct (no webhook — the
repo has none, and CEL needs no new infrastructure; requires K8s ≥ 1.25, project deps
are on v0.36 so fine):

```go
// +kubebuilder:validation:XValidation:rule="has(self.clientID) != has(self.secretRef)",message="exactly one of clientID or secretRef must be set"
// +kubebuilder:validation:XValidation:rule="self.public || has(self.secretRef)",message="secretRef is required for confidential (non-public) clients"
// +kubebuilder:validation:XValidation:rule="self.public || (has(self.redirectURIs) && self.redirectURIs.size() > 0)",message="redirectURIs is required for confidential (non-public) clients"
```

Note: `public` has `omitempty`, so the rules must read `self.public` defensively — either
give `Public` a `+kubebuilder:default=false` or use `(has(self.public) && self.public)`
in the rules. Decide during implementation; default marker is cleaner.

Semantics matrix:

| public | clientID | secretRef | valid? | behavior |
|--------|----------|-----------|--------|----------|
| false  | —        | set       | yes    | today's behavior (id + secret from Secret) |
| false  | set      | —         | no     | CEL rejects: confidential needs secretRef |
| true   | set      | —         | yes    | **new: secretless client**, no env var emitted |
| true   | —        | set       | yes    | public client with id (+ secret) from Secret — Dex allows a secret on public clients (e.g. PKCE + secret) |
| any    | set      | set       | no     | CEL rejects: ambiguous |

Backward compatibility: all existing CRs set `secretRef` and non-empty `redirectURIs`, so
loosening `required` and switching to a pointer is non-breaking. Same API version stays.

Also update:
- `GetReferencedSecretNames` ([api/v1/connector_helpers.go:246-251](../../../api/v1/connector_helpers.go#L246-L251)):
  nil-check the pointer (already returns nil for empty name — extend to `SecretRef == nil`).
- `zz_generated.deepcopy.go` via `make generate-all` (pointer deepcopy).

## 4. Builder changes

File: [internal/builder/clients.go](../../../internal/builder/clients.go), `buildOneStaticClient`:

- `SecretRef == nil` path: `id = spec.ClientID`; skip client-secret resolution, skip
  `clientEnvKey` collision registration, emit no `EnvSecretData` entry, leave
  `SecretEnv` empty. Output: `{id, name, redirectURIs, trustedPeers?, public: true}`.
- `SecretRef != nil` path: unchanged (both keys resolved, `secretEnv` emitted) — also
  when `public: true`.
- No changes to `config_types.go` (`Public` and `omitempty` on `SecretEnv`/`RedirectURIs`
  already correct — a nil `RedirectURIs` is simply omitted from the YAML, which is
  exactly Dex's "defaults" mode).

Keep cyclomatic complexity < 15: extract the two id/secret-sourcing paths into a small
helper if needed.

## 5. Controller changes

Minimal:
- [internal/controller/child_reconciler.go](../../../internal/controller/child_reconciler.go)
  does no secretRef validation today — nothing to change.
- Secret-watch indexers ([internal/controller/dexinstallation_controller.go:206](../../../internal/controller/dexinstallation_controller.go#L206),
  [internal/controller/secret_watch.go](../../../internal/controller/secret_watch.go)) go through
  `GetReferencedSecretNames`, which becomes nil-safe in step 3 — verify, no logic change
  expected.
- `StaticClientCount`, collect/filter logic: unchanged.

## 6. Manifests

- `make manifests` regenerates
  [config/crd/bases/dex.gtrfc.com_dexstaticclients.yaml](../../../config/crd/bases/dex.gtrfc.com_dexstaticclients.yaml)
  (drops `secretRef`/`redirectURIs` from `required`, adds `clientID` +
  `x-kubernetes-validations`) and syncs the Helm copy under
  [deploy/helm/dex-operator/files/crds/](../../../deploy/helm/dex-operator/files/crds/) via
  `helm-sync-crds`.

## 7. Tests

Unit ([internal/builder/builder_test.go](../../../internal/builder/builder_test.go), patterns:
`minimalInstallation`, `mockResolver`, `parseYAML`):
- Public secretless client → YAML has `id` (from `clientID`), `public: true`, **no**
  `secretEnv`; `EnvSecretData` has no entry. Use the `assertNoKeys` dex-schema pinning
  pattern from [internal/builder/builder_dexschema_test.go:54](../../../internal/builder/builder_dexschema_test.go#L54).
- Public secretless client with empty `redirectURIs` → key omitted from YAML.
- Public client **with** secretRef → behaves exactly like today plus `public: true`.
- Mixed set: one confidential + one secretless client → env secret contains only the
  confidential client's env var; no collision interference.
- Helpers test: `GetReferencedSecretNames` with nil `SecretRef`
  ([internal/controller/helpers_test.go:278](../../../internal/controller/helpers_test.go#L278) pattern).

Integration ([test/integration/staticclient_test.go](../../../test/integration/staticclient_test.go),
envtest — the envtest apiserver enforces CEL, so validation rules get real coverage):
- Create secretless public client → installation config secret contains the client with
  `public: true`; env secret unchanged.
- Apply invalid CRs (confidential without secretRef; clientID + secretRef both set;
  confidential without redirectURIs) → apiserver rejects with the CEL messages.

E2E ([test/e2e/dexinstallation_e2e_test.go:142](../../../test/e2e/dexinstallation_e2e_test.go#L142)):
- Extend `TestE2E_StaticClient` (or add one test) with a secretless public client and
  assert Dex accepts the rendered config (Dex ≥ v2.24 required — any current chart
  version qualifies).

Run everything via Makefile targets only: `make lint test-unit test-integration`
(`test-e2e` on demand).

## 8. Docs

- README: add a secretless/public client example next to the existing DexStaticClient
  example ([README.md:149-175](../../../README.md#L149-L175)). Note: that example is stale
  (`name:` instead of `displayName:`) — fix in the same pass.
- CLAUDE.md: document the clientID/secretRef XOR + CEL approach under the builder notes.

## 9. Out of scope / follow-ups

- **`oauth2.pkce` enforcement** (`enforce`, `codeChallengeMethodsSupported`) in
  `DexOAuth2ConfigSpec` ([api/v1/dexinstallation_types.go:389](../../../api/v1/dexinstallation_types.go#L389)):
  merged upstream after Dex v2.45.1, not in any release yet. Add once a Dex release
  ships it; separate small ticket.
- Device-flow specifics (dexidp/dex#3983) — Dex-side limitation, nothing actionable here.
- No admission webhook — CEL covers all conditional validation.

## 10. Success criteria

1. `kubectl apply` of a DexStaticClient with `public: true` + `clientID` + no
   `secretRef` succeeds; the rendered config secret contains the client with
   `public: true` and no `secretEnv`; the env secret gets no entry.
2. Invalid combinations from the matrix are rejected at admission with clear messages.
3. Existing CRs (confidential, secretRef) reconcile byte-identically to before.
4. `make lint test-unit test-integration` green; e2e covers the happy path against real
   Dex.

## Suggested commit message

```
feat(staticclient): support secretless public clients via inline clientID and CEL validation
```
