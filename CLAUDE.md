# Dex Operator

Ein Kubernetes Operator (Go 1.26, controller-runtime) der die Konfiguration von Dex dynamisch aus Custom Resources zusammenbaut.
Dex wird weiterhin über das offizielle Dex Helm Chart installiert. Der Operator erzeugt zwei Secrets im Namespace der Dex-Installation:
1. **Config-Secret** — Enthält die vollständige Dex-Konfiguration als YAML (Issuer, Storage, Web, CORS, gRPC, Logger, Expiry, Connectors, Static Clients)
2. **Env-Secret** — Enthält alle Client-Secrets als Env-Variablen (z.B. `GRAFANA_CLIENT_SECRET`), wird per `envFrom` an den Dex-Container gehängt und in der Config per `secretEnv` referenziert

## API Group
`dex.gtrfc.com/v1`

## CRDs (alle namespace-scoped)

### DexInstallation
Vollständige globale Dex-Konfiguration: Issuer, Storage, Web, CORS, gRPC, Logger, Expiry.
Zusätzlich: `configSecretName`, `envSecretName`, `allowedNamespaces` (Whitelist, `"*"` = alle), optionaler Auto-Restart (`rolloutRestart.enabled`, `rolloutRestart.deploymentName`).

### DexStaticClient
Referenziert eine DexInstallation per Name+Namespace. Enthält `redirectURIs`, `allowedScopes`, `trustedPeers`, `name`.
Referenziert ein bestehendes Secret im gleichen Namespace per `secretRef` mit Keys für `client-id` und `client-secret`.

### Connector CRDs (je eine eigene CRD pro Typ)
`DexLDAPConnector`, `DexGitHubConnector`, `DexSAMLConnector`, `DexGitLabConnector`, `DexOIDCConnector`, `DexOAuth2Connector`, `DexGoogleConnector`, `DexLinkedInConnector`, `DexMicrosoftConnector`, `DexAuthProxyConnector`, `DexBitbucketConnector`, `DexLocalConnector`, `DexOpenShiftConnector`, `DexAtlassianCrowdConnector`, `DexGiteaConnector`, `DexKeystoneConnector`
Jede referenziert eine DexInstallation per Name+Namespace und enthält die typ-spezifische Konfiguration.

## Architektur-Überblick
- Operator watched alle Namespaces
- Bei Reconciliation einer DexInstallation: Alle zugehörigen Connectors und Static Clients aus erlaubten Namespaces sammeln, Config-YAML + Env-Secret bauen, Secrets im Dex-Namespace schreiben
- Bei Reconciliation eines Clients/Connectors: Die referenzierte DexInstallation triggern
- Namespace-Whitelist-Validierung bei jedem Client/Connector
- Optionaler Rollout-Restart des Dex-Deployments bei Config-Änderung

## Repository & Registry
`guidedtraffic/dex-operator`

## Builder Package (`internal/builder`)

The `Build(ctx, Input) (Output, error)` function is the single entry point.

**Input** carries:
- `*DexInstallation` (issuer, storage, web, gRPC, logger, expiry, oauth2)
- `ConnectorSet` — all 16 connector types grouped by CRD kind
- `[]DexStaticClient`
- `SecretResolver` — caller-provided func to resolve `SecretKeyRef` → value

**Output** carries:
- `ConfigYAML []byte` — ready-to-store Dex `config.yaml`
- `EnvSecretData map[string][]byte` — env vars for the Dex env Secret (`$VAR` refs in config)
- `MountedSecrets []MountedSecret` — Secret keys that the controller must mount as files (TLS certs, service-account JSON, etc.)

Env-var naming convention:
- Connector credential: `<TYPE>_<ID>_<FIELD>` (e.g. `OIDC_OKTA_CLIENT_SECRET`)
- Static client secret: `<RESOURCE_NAME>_CLIENT_SECRET` (e.g. `GRAFANA_CLIENT_SECRET`)
- Storage credential: `STORAGE_<FIELD>` (e.g. `STORAGE_POSTGRES_PASSWORD`)

CA cert data (LDAP `rootCAData`) is base64-encoded and inlined in config.
File-path-only certs (SAML `ca`, client TLS, service accounts) are added to `MountedSecrets`; the controller picks these up in Phase 4.

---

# Important Notes

- Remember Cyclomatic Complexity: Keep it under 15 for all functions. Refactor if it exceeds this threshold.
- Check Code linting and formatting before reporing task done
- We have Unit-Tests, Integration-Tests and E2E-Tests. Always write tests for new features and bug fixes. Aim for high coverage, especially for critical reconciliation logic.
- Use the Makefile targets for all testing, linting, and analysis tasks. Do not run Go test commands or tools directly. This ensures consistency between local development and CI pipelines.
- For E2E tests, focus on real-world scenarios like rolling updates, failover, and recovery. Use actual Valkey instances to verify behavior.
- Do not commit to git, ask the user for a review and let the user commit to git. This ensures that the user is aware of all changes and can provide feedback before they are finalized.
- if you need to write temporary files, write them to local tmp-folder. Do not use the system tmp folder at /tmp
- persist important information about the project and implementation in this file
- if you are done with your task, always report a conventional commit message to the user, but do not commit to git. Let the user review and commit to git. This ensures that the user is aware of all changes and can provide feedback before they are finalized.
