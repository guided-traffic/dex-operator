# ADR 0004: A Static Client Is Confidential Through a Secret or Public Through an Inline ID, Validated by CEL Without a Webhook

## Status

Accepted. Date: 2026-10-07, recording the decision built in 2.1.0 (2026-07-26).

**Implemented.**

## Context

Dex has accepted static clients with `public: true` and no secret since v2.24.0; their
authorization code is bound by PKCE, and with an empty `redirectURIs` list Dex accepts loopback
addresses, the OOB URN and the device-flow callback — the native-app and CLI case. The operator
could not express such a client: `secretRef` was required, the client ID came only from the
referenced Secret, the builder resolved `client-secret` unconditionally, and `redirectURIs` had
`MinItems=1`.

The rules that make a client valid are conditional (which field is required depends on
`public`), and the repository has no admission webhook.

## Decision

**D1 — The client ID comes from exactly one source.** Either `secretRef` (the Secret holds
`client-id` and `client-secret`) or the inline `clientID`, never both and never neither.

**D2 — A confidential client needs `secretRef` and at least one redirect URI.** A public client
may omit `redirectURIs` and gets Dex's loopback, OOB and device-flow defaults.

**D3 — A public client may still carry a secret.** `public: true` with `secretRef` is valid (PKCE
plus a secret); the builder then behaves as for a confidential client and sets `public: true`.

**D4 — A secretless client leaves no trace in the env Secret.** With `SecretRef == nil`,
`buildOneStaticClient` skips secret resolution, the env-key collision check and the
`EnvSecretData` entry, and leaves `secretEnv` unset
([internal/builder/clients.go](../../internal/builder/clients.go)).

**D5 — The conditional rules are CEL on the spec; there is no webhook.**
On `DexStaticClientSpec` ([api/v1/dexstaticclient_types.go](../../api/v1/dexstaticclient_types.go)):

1. `has(self.clientID) != has(self.secretRef)`
2. `(has(self.public) && self.public) || has(self.secretRef)`
3. `(has(self.public) && self.public) || (has(self.redirectURIs) && self.redirectURIs.size() > 0)`

`public` carries `omitempty` and no default, so every rule guards with `has(self.public)`; a bare
`self.public` errors on an object that never set it.

## Consequences

- Invalid combinations are rejected at `kubectl apply` with the rule's message; the envtest
  apiserver enforces CEL, so the integration suite covers the rules.
- Validation that needs another object — whether the referenced Secret exists, whether two
  clients' env keys collide — stays at build time and shows on the installation's status, not at
  admission.
- Marking a server-side application `public` removes client authentication from its token
  requests; the schema cannot tell a server from a CLI, so it does not stop it.

## Alternatives Considered

- **An admission webhook.** Would allow cross-object checks, but brings certificate management
  and couples every apply to the operator's availability. Lost; CEL covers the static rules.
- **`+kubebuilder:default=false` on `public`** so the rules can read `self.public`. Cleaner rules;
  the work list preferred it, the implementation chose the `has()` guard, and why is not recorded.
  A default would write the field into every stored object. Either is correct.
- **A separate CRD for public clients.** Doubles the reconciler wiring for one boolean. Lost.

## Residual risks

- Device-flow token exchange in Dex still has known gaps for confidential clients
  (dexidp/dex#3983); public clients are the supported path there. Not something this operator
  can change.
- PKCE enforcement (`oauth2.pkce`) was merged upstream after Dex v2.45.1 and is not offered by
  `DexOAuth2ConfigSpec`. Not verified whether a Dex release ships it yet.

## References

- [api/v1/dexstaticclient_types.go](../../api/v1/dexstaticclient_types.go),
  [api/v1/connector_helpers.go](../../api/v1/connector_helpers.go) (`GetReferencedSecretNames`, nil-safe)
- [internal/builder/clients.go](../../internal/builder/clients.go)
- [test/integration/staticclient_test.go](../../test/integration/staticclient_test.go)
- [docs/security/clients.md](../security/clients.md), [docs/security/validation.md](../security/validation.md)
