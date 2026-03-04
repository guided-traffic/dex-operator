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
- [ ] DexInstallation types + deepcopy
- [ ] DexStaticClient types + deepcopy
- [ ] DexLDAPConnector types
- [ ] DexGitHubConnector types
- [ ] DexSAMLConnector types
- [ ] DexGitLabConnector types
- [ ] DexOIDCConnector types
- [ ] DexOAuth2Connector types
- [ ] DexGoogleConnector types
- [ ] DexLinkedInConnector types
- [ ] DexMicrosoftConnector types
- [ ] DexAuthProxyConnector types
- [ ] DexBitbucketConnector types
- [ ] DexLocalConnector types (BuiltIn)
- [ ] DexOpenShiftConnector types
- [ ] DexAtlassianCrowdConnector types
- [ ] DexGiteaConnector types
- [ ] DexKeystoneConnector types
- [ ] Gemeinsame Typen: InstallationRef, SecretKeyRef, Status-Conditions
- [ ] CRD-Manifeste generieren (`make manifests`)

### Phase 3: Config Builder
- [ ] Dex Config YAML Struct (interne Repräsentation, nicht CRD)
- [ ] Config-Builder: DexInstallation + Connectors + Clients → Dex YAML
- [ ] Env-Secret-Builder: Client-Secrets aus referenzierten Secrets sammeln → Env-Secret Map
- [ ] Unit-Tests für Config-Builder

### Phase 4: Controller
- [ ] DexInstallation Controller (Reconciler): Config + Env Secret schreiben
- [ ] DexStaticClient Controller: DexInstallation re-reconcile triggern
- [ ] Connector Controller (generisch oder pro Typ): DexInstallation re-reconcile triggern
- [ ] Namespace-Whitelist-Validierung
- [ ] Optionaler Rollout-Restart Logik
- [ ] RBAC-Konfiguration (Secrets lesen/schreiben, Deployments patchen)
- [ ] Unit-Tests für Controller

### Phase 5: Integration & E2E Tests
- [ ] Integration-Tests mit envtest
- [ ] E2E-Tests mit Kind-Cluster + Dex Helm Chart
- [ ] CI-Pipeline testen (build, lint, test, release)

### Phase 6: Dokumentation & Helm
- [ ] Operator Helm Chart (deploy/helm/dex-operator)
- [ ] CRD-Sync in Helm Chart
- [ ] Beispiel-Manifeste (examples/)
- [ ] README mit Architektur, Quickstart, Beispielen