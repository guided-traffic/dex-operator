# Dex Operator

Ein Kubernetes Operator (Go 1.26, controller-runtime) der die Konfiguration von Dex dynamisch aus Custom Resources zusammenbaut.
Dex wird weiterhin über das offizielle Dex Helm Chart installiert. Der Operator erzeugt zwei Secrets im Namespace der Dex-Installation:
1. **Config-Secret** — Enthält die vollständige Dex-Konfiguration als YAML (Issuer, Storage, Web, CORS, gRPC, Logger, Expiry, Connectors, Static Clients)
2. **Env-Secret** — Enthält alle Client-Secrets als Env-Variablen (z.B. `GRAFANA_CLIENT_SECRET`), wird per `envFrom` an den Dex-Container gehängt und in der Config per `$ENV_VAR` referenziert

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

---

## Projektplan

### Phase 1: Projekt-Setup
- [x] Go-Modul initialisieren (`go mod init github.com/guided-traffic/dex-operator`)
- [x] Kubebuilder-Projektstruktur anlegen (cmd/, internal/, api/, config/)
- [x] Boilerplate-Dateien (hack/boilerplate.go.txt, main.go)
- [x] Kopierte CI-Dateien anpassen (Makefile, package.json, Containerfile, workflows, .releaserc.json, renovate.json — valkey-operator → dex-operator)
- [x] README.md aktualisieren

### Phase 2: CRD-Typen definieren (api/v1/)
- [x] DexInstallation types + deepcopy
- [x] DexStaticClient types + deepcopy
- [x] DexLDAPConnector types
- [x] DexGitHubConnector types
- [x] DexSAMLConnector types
- [x] DexGitLabConnector types
- [x] DexOIDCConnector types
- [x] DexOAuth2Connector types
- [x] DexGoogleConnector types
- [x] DexLinkedInConnector types
- [x] DexMicrosoftConnector types
- [x] DexAuthProxyConnector types
- [x] DexBitbucketConnector types
- [x] DexLocalConnector types (BuiltIn)
- [x] DexOpenShiftConnector types
- [x] DexAtlassianCrowdConnector types
- [x] DexGiteaConnector types
- [x] DexKeystoneConnector types
- [x] Gemeinsame Typen: InstallationRef, SecretKeyRef, Status-Conditions
- [x] CRD-Manifeste generieren (`make manifests`)

### Phase 3: Config Builder
- [x] Dex Config YAML Struct (interne Repräsentation, nicht CRD) → `internal/builder/config_types.go`
- [x] Config-Builder: DexInstallation + Connectors + Clients → Dex YAML → `internal/builder/builder.go`, `connectors.go`, `connectors_oauth.go`, `clients.go`, `storage.go`
- [x] Env-Secret-Builder: Client-Secrets aus referenzierten Secrets sammeln → Env-Secret Map (in `Build` via `SecretResolver`)
- [x] Unit-Tests für Config-Builder → `internal/builder/builder_test.go`

### Phase 4: Controller
- [x] DexInstallation Controller (Reconciler): Config + Env Secret schreiben
- [x] DexStaticClient Controller: DexInstallation re-reconcile triggern
- [x] Connector Controller (generisch oder pro Typ): DexInstallation re-reconcile triggern
- [x] Namespace-Whitelist-Validierung
- [x] Optionaler Rollout-Restart Logik
- [x] RBAC-Konfiguration (Secrets lesen/schreiben, Deployments patchen)
- [x] Unit-Tests für Controller

### Phase 5: Integration & E2E Tests
- [ ] Integration-Tests mit envtest
- [ ] E2E-Tests mit Kind-Cluster + Dex Helm Chart
- [ ] CI-Pipeline testen (build, lint, test, release)

### Phase 6: Dokumentation & Helm
- [ ] Operator Helm Chart (deploy/helm/dex-operator)
- [ ] CRD-Sync in Helm Chart
- [ ] Beispiel-Manifeste (examples/)
- [ ] README mit Architektur, Quickstart, Beispielen