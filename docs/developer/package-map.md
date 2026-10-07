# Package map

Where something lives and what each file is responsible for. The tree itself is
[repository-layout.md](repository-layout.md); how the pieces interact is
[architecture.md](architecture.md).

## `api/v1`

| File | Responsibility |
|---|---|
| [groupversion_info.go](../../api/v1/groupversion_info.go) | `GroupVersion` `dex.gtrfc.com/v1`, `SchemeBuilder` |
| [common_types.go](../../api/v1/common_types.go) | `InstallationRef`, `SecretKeyRef` (name + key, no namespace), `CommonStatus`, condition types (`Ready`, `ChildrenRejected`, `TrustedPeersDropped`) |
| [dexinstallation_types.go](../../api/v1/dexinstallation_types.go) | `DexInstallation`: issuer, storage, web (CORS), gRPC, logger, expiry, oauth2, frontend, `configSecretName`, `envSecretName`, `allowedNamespaces`, `allowedConnectorNamespaces`, `rolloutRestart`; status with the counts, `rejectedChildren` (`RejectedChild`, the `RejectionReason*` constants) and `droppedTrustedPeers` (`DroppedTrustedPeer`) |
| [dexstaticclient_types.go](../../api/v1/dexstaticclient_types.go) | `DexStaticClient`: `secretRef` or `clientID`, `redirectURIs`, `trustedPeers`, `public`, `cors`; the three CEL rules; `status.clientID` |
| `dex<type>connector_types.go` | one connector CRD each (16), every spec with `InstallationRef`, `ID`, `DisplayName` and the kind's options |
| [connector_helpers.go](../../api/v1/connector_helpers.go) | the `ChildObject` methods of every child kind — `GetInstallationRef`, `GetCommonStatus`, `GetReferencedSecretNames` |

## `internal/builder`

Pure: no Kubernetes client, Secrets resolved through the injected `SecretResolver`
([builder.md](builder.md)).

| File | Responsibility |
|---|---|
| [builder.go](../../internal/builder/builder.go) | `Build`, `Input`, `Output`, `ConnectorSet`; `assembleDexConfig`, `assembleWebConfig`, CORS origin derivation (`deriveCORSOrigins`, `appendDerivedOrigins`), the Secret and mount helpers (`resolveEnvSecret`, `resolveSecret`, `mountCertFile`, `mountSecretAsFile`), `connectorID` (`spec.id`, else `metadata.name`) |
| [render.go](../../internal/builder/render.go) | `renderUnit`, the contest rule (`contest`, `decide`), env priority (`byEnvPriority`), `buildUnits` (winners build into the env, losers dry), `rejectedChildren`, the connector guard (`noConnectorError`, `ErrNoConnector`) |
| [config_types.go](../../internal/builder/config_types.go) | YAML-tagged structs mirroring Dex's `config.yaml` |
| [envvar.go](../../internal/builder/envvar.go) | env var naming: `sanitizeEnvKey`, `connectorEnvBase`, `storageEnvBase`; `childEnv` — plain or fallback key per child, committed on success (`childHash`) |
| [clients.go](../../internal/builder/clients.go) | static clients: `clientUnits` (the claim, `status.clientID` fallback), `resolveClientID`, `buildOneStaticClient` — the confidential and public branches, `secretEnv` —, `filterTrustedPeers`, `renderedClients`, `resolvedClientIDs` |
| [connectors.go](../../internal/builder/connectors.go) | `connectorUnits`, `addConnectorUnits`, `reserveLocal`, `renderedConnectors`; LDAP (inline base64 `rootCAData`, mounted client certificate and key), SAML (mounted `ca`), AuthProxy; local connectors excluded |
| [connectors_oauth.go](../../internal/builder/connectors_oauth.go) | `oauthConnectorUnits` and every OAuth2-style connector; the shared `resolveOAuthCreds` (client ID inline, client secret → env) |
| [storage.go](../../internal/builder/storage.go) | kubernetes, memory, postgres, sqlite3, etcd, mysql; passwords → `STORAGE_*` env, TLS material → fixed mount paths |
| [doc.go](../../internal/builder/doc.go) | package documentation |
| [export_test.go](../../internal/builder/export_test.go) | exports internals to the external test package |

## `internal/controller`

| File | Responsibility |
|---|---|
| [dexinstallation_controller.go](../../internal/controller/dexinstallation_controller.go) | `DexInstallationReconciler`: collect, build, write both Secrets, restart, status, `recordClientIDs`; `makeSecretResolver`; `registerIndexers`, the watches of every child kind and of Secrets |
| [child_reconciler.go](../../internal/controller/child_reconciler.go) | `GenericChildReconciler[T, U]`: the child's `Ready` condition (`findRejectedChild`, `newRejectionError`), a static client's `TrustedPeersDropped` (`setTrustedPeersDroppedCondition`), `configError` with its five-minute requeue, the `DexInstallation` watch (`mapInstallationToChildren`, `childReportChangedPredicate`) |
| [connector_controller.go](../../internal/controller/connector_controller.go) | registers one generic reconciler per child kind (16 connectors + `DexStaticClient`) |
| [interfaces.go](../../internal/controller/interfaces.go) | the `ChildObject` interface |
| [collect.go](../../internal/controller/collect.go) | the field indexes (`InstallationRefIndexField`, `SecretRefIndexField`), `collectConnectors`, `collectStaticClients`, `filterItems` (allowlist + sort), `sortByNamespaceName` |
| [namespace.go](../../internal/controller/namespace.go) | `isNamespaceAllowed`, `connectorNamespaces`, `checkChildNamespace` — pure functions |
| [secret.go](../../internal/controller/secret.go) | `applySecret` with a pluggable comparison: `secretDataEqual` (bytes), `yamlSecretDataEqual` (parsed YAML) |
| [secret_watch.go](../../internal/controller/secret_watch.go) | `secretWatchPredicate` (deletes dropped), `mapSecretToInstallation`, `lookupChildSecretRefs` |
| [rollout.go](../../internal/controller/rollout.go) | `triggerRolloutRestart`: the `kubectl.kubernetes.io/restartedAt` pod-template annotation |
| [status.go](../../internal/controller/status.go) | `setReadyCondition`, `setOrReplaceCondition`, `setChildReport` (the installation's rejected children, dropped peers and their conditions) |
| [rbac.go](../../internal/controller/rbac.go) | every kubebuilder RBAC marker of the operator |
| [doc.go](../../internal/controller/doc.go), [export_test.go](../../internal/controller/export_test.go) | package documentation; exports for the external test package |

## `cmd`

[cmd/main.go](../../cmd/main.go) parses the flags (`--metrics-bind-address`, default `0` = off;
`--health-probe-bind-address`; `--leader-elect`), builds the manager and sets up the
`DexInstallationReconciler` **before** the child reconcilers, because it registers the
`InstallationRefIndexField` index they use.
