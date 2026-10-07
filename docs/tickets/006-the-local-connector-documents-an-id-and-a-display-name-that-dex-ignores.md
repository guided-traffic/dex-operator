---
id: T6
title: the DexLocalConnector documents an id and a display name that dex ignores
state: filed
severity: low
security: none
threat:
urgency: now          # rule 1: a measured-false statement in tracked files
effort: XS
blocked-by:
filed-from: the analysis of the connector kinds behind allowedConnectorNamespaces (T4)
opened: 2026-10-07
decided:
done:
shipped:
dropped-reason:
publication-accepted:
---

## Current state

A `DexLocalConnector` renders no connector entry: its presence only sets
`enablePasswordDB: true` (`buildAllConnectors` leaves local connectors out,
[internal/builder/connectors.go:31-32](../../internal/builder/connectors.go#L31-L32); `Build`
passes `len(in.Connectors.Local) > 0` to `assembleDexConfig`,
[internal/builder/builder.go:116](../../internal/builder/builder.go#L116)). Dex then registers a
fixed connector with the ID `local` and the name `Email` (dex v2.44.0, `cmd/dex/serve.go`, as read
for T4; not re-read for this ticket). The resource's `id` and `displayName` have no effect.

Three tracked places say otherwise:

- [README.md:1167-1168](../../README.md#L1167-L1168), the fully populated example:
  `id: local # default: metadata.name` and `displayName: Email # required — label on the login form`.
- [api/v1/dexlocalconnector_types.go:28](../../api/v1/dexlocalconnector_types.go#L28):
  "ID is the connector ID used internally by Dex. Defaults to metadata.name." — this doc comment
  is the field description in the generated CRD, so `kubectl explain` repeats it.
- [api/v1/dexlocalconnector_types.go:21-22](../../api/v1/dexlocalconnector_types.go#L21-L22):
  "User credentials are stored in a Kubernetes Secret." — the passwords live in dex's storage
  (with `kubernetes` storage, the `passwords.dex.coreos.com` objects in the dex namespace), not in
  a Secret the operator reads.

Impact: an operator who sets `id` or `displayName` expects a login-form label or a connector ID
that never appears, and looks for the user list in a Secret that does not exist.

## Required changes

1. The doc comments of `DexLocalConnectorSpec`, `ID` and `DisplayName` say that dex fixes the
   connector to ID `local` and name `Email`, that both fields are accepted and ignored, and where
   the passwords live. `make generate-all` carries the text into the CRDs and the Helm chart.
2. The README example drops the misleading comments (`id` and `displayName` marked as without
   effect; `displayName` stays because the CRD requires it).
3. No behaviour change, no test change: the builder test that pins `enablePasswordDB` stays as
   it is.

## Open questions

### Q1: Should `displayName` stop being required?

It is `+kubebuilder:validation:Required` but has no effect. Options: keep it required and document
it as ignored (no API change, recommended: a relaxed requirement is a schema change for a field
nobody needs, and nothing breaks today); or make it optional (one fewer meaningless field to fill
in, a CRD change that every installation picks up on its next upgrade).

**Answer:** _open_
