# Validation: what is refused at admission and what only at build time

What the API server refuses before a custom resource is stored, and what the operator can only
find out when it renders. Who may write the resources at all is [tenancy.md](tenancy.md) and
RBAC; what a valid client or connector can then do is [clients.md](clients.md) and
[connectors.md](connectors.md).

## No webhook

The repository has no admission webhook — no certificate management, no coupling of every apply
to the operator's availability. Everything enforced at admission is enforced by the API server
itself, from the CRD schema
([ADR 0004](../adr/0004-a-static-client-is-confidential-through-a-secret-or-public-through-an-inline-id-validated-by-cel.md) D5).

## Refused at admission

- **CEL rules on `DexStaticClientSpec`**
  ([api/v1/dexstaticclient_types.go](../../api/v1/dexstaticclient_types.go)) guarantee that a
  client is either confidential (`secretRef` set, at least one redirect URI) or public
  (`public: true`, inline `clientID`, redirect URIs optional) — never both `clientID` and
  `secretRef`, never neither. Invalid objects are rejected at `kubectl apply` with the rule's
  message. The README's DexStaticClient reference has the full truth table.
- **`DexInstallation.spec.allowedConnectorNamespaces`** is a set (`listType=set`, `MinItems=1`)
  and a CEL rule requires `"*"` to be the only entry, so a list can never look restrictive while
  admitting everything, and "deny all" cannot be written as `[]`
  ([api/v1/dexinstallation_types.go](../../api/v1/dexinstallation_types.go),
  [tenancy.md](tenancy.md)).
- **Enums and formats** on fields Dex interprets: the storage type, the PostgreSQL SSL mode, the
  log level and format, a few connector fields, and `format: uri` on the issuer and on connector
  URLs (OIDC issuer, OAuth2 endpoints, SAML SSO URL, OpenShift, Gitea, Keystone and
  AtlassianCrowd hosts). They refuse values Dex would not understand before they reach a render.
  They are not what keeps tenant input out of the YAML structure: the builder fills YAML-tagged
  structs and marshals them, so a field value cannot become a key
  ([internal/builder/config_types.go](../../internal/builder/config_types.go)).

The CEL rules and `MinItems` are covered by the integration suite — the envtest API server
enforces CEL ([test/integration/staticclient_test.go](../../test/integration/staticclient_test.go),
[connector_namespaces_test.go](../../test/integration/connector_namespaces_test.go)).

## Found only at build time

Validation that needs knowledge of other objects cannot be expressed in CEL: whether a referenced
Secret and key exist, whether a client or connector ID is claimed by more than one child, whether
an env var key is taken. It happens at build time. A child that fails it is left out of the
config and reported — on the installation in `status.rejectedChildren` with the condition
`ChildrenRejected`, on the child as `Ready=False` with reason `DuplicateID` or `BuildFailed` — while
every other child renders
([ADR 0008](../adr/0008-an-id-renders-for-one-child-only-and-a-failing-child-is-left-out-instead-of-failing-the-render.md)).
A colliding env var key is not an error at all: the later child gets a fallback key
([secret-flow.md](secret-flow.md)). Only a failing storage credential and the connector guard fail
the render; they surface through the `DexInstallation`'s `Ready` condition
([internal/controller/status.go](../../internal/controller/status.go) `setReadyCondition`). The
operator records no Kubernetes events — it has no event recorder, and `events` appear only in the
leader-election Role.

## What this does not cover

- **Cross-object rules at admission.** Nothing that depends on another object — a Secret, another
  client, another connector — is checked when a resource is applied; there is no webhook to do it.
  A duplicate ID is accepted by the API server and only rejected by the render.
- **Semantic checks of upstream settings.** A syntactically valid issuer URL, bind DN or scope list
  is accepted whether or not the upstream answers to it; Dex finds out when it starts or when a
  user logs in.
