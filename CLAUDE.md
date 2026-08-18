# Dex Operator

Ein Kubernetes Operator (Go 1.26, controller-runtime) der die Konfiguration von Dex dynamisch aus Custom Resources zusammenbaut.

## Dokumentation
- `README.md` — User-Doku: Key Features, Naming-Konventionen, TL;DR-Quickstart, vollständige CR-Referenz (alle 18 CRDs maximal befüllt, Connectors in `<details>`-Blöcken). Bei API-Änderungen die betroffenen Beispiele mitpflegen.
- `DEVELOPER.md` — Entwickler-Doku: Repo-Layout, Paket-Verantwortlichkeiten, Reconcile-Flow, Checkliste "neuen Connector-Typ hinzufügen", Test-/Release-Prozess.
- `SECURITY_ARCHITECTURE.md` — Sicherheitsarchitektur: Trust-Boundaries, Secret-Flow (env-Indirektion, MountedSecrets), Namespace-Isolation, RBAC-Footprint, Hardening-Checkliste.
Dex wird weiterhin über das offizielle Dex Helm Chart installiert. Der Operator erzeugt zwei Secrets im Namespace der Dex-Installation:
1. **Config-Secret** — Enthält die vollständige Dex-Konfiguration als YAML (Issuer, Storage, Web, gRPC, Logger, Expiry, Connectors, Static Clients)
2. **Env-Secret** — Enthält alle Client-Secrets als Env-Variablen (z.B. `GRAFANA_CLIENT_SECRET`), wird per `envFrom` an den Dex-Container gehängt und in der Config per `secretEnv` referenziert

## API Group
`dex.gtrfc.com/v1`

## CRDs (alle namespace-scoped)

### DexInstallation
Vollständige globale Dex-Konfiguration: Issuer, Storage, Web (inkl. CORS `allowedOrigins`/`allowedHeaders`), gRPC, Logger, Expiry.
Zusätzlich: `configSecretName`, `envSecretName`, `allowedNamespaces` (Whitelist, `"*"` = alle), optionaler Auto-Restart (`rolloutRestart.enabled`, `rolloutRestart.deploymentName`).

### DexStaticClient
Referenziert eine DexInstallation per Name+Namespace. Enthält `redirectURIs`, `trustedPeers`, `displayName`, `public`.
Die Client-Credentials kommen entweder aus einem bestehenden Secret im gleichen Namespace (`secretRef` mit Keys für `client-id` und `client-secret`) oder — bei public/secretless Clients — inline über `clientID`.

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

### Static clients: confidential vs. public (secretless)

`DexStaticClientSpec` sources the client ID either from `secretRef` (confidential) or
from the inline `clientID` field (public/secretless, PKCE — supported by Dex since
v2.24.0). The two are mutually exclusive; `buildOneStaticClient` branches on
`SecretRef == nil` and then skips secret resolution, the env-key collision check and
the `EnvSecretData` entry entirely, leaving `secretEnv` unset in the config.

### Derived CORS origins (`spec.cors`)

`DexStaticClientSpec.CORS` (bool, `json:"cors,omitempty"`) opts a client into
contributing the origins of its own **https** `redirectURIs` to the
installation's `web.allowedOrigins`. `deriveCORSOrigins` runs in `Build` over
the spec-level clients; `assembleWebConfig` merges the result. Nothing lands in
the rendered `staticClients` entry — dex has no per-client CORS.

Decisions (asked and settled, do not silently revert):

- **Installation order preserved**, derived origins appended sorted
  (`appendDerivedOrigins`). Sorting the whole union would rewrite existing
  authored lists on an operator upgrade alone → config diff → spurious dex
  rollout.
- **https only.** Loopback/http, custom schemes, OOB URN skipped; unparsable
  URIs skipped instead of failing the whole render.
- **Host lowercased, redundant `:443` dropped** — dex matches the `Origin`
  header literally via `gorilla/handlers.AllowedOrigins`.
- **Not gated on `public`** — a confidential client with the flag derives too.
- `spec.web == nil` + derived origins creates the `web:` block (listener
  addresses come from the dex chart's CLI flags, applied after config load).

Consumer follow-ups (k8s-flux-base chart bump, k8s-flux-mgmt `cors: true`) are
documented in `local_web_origins.md` §6 and deliberately **not** done here —
they need a released operator first.

There is no admission webhook in this repo. All conditional validation is done with
**CEL markers** (`+kubebuilder:validation:XValidation`) on the spec struct:

1. `has(self.clientID) != has(self.secretRef)` — exactly one of the two.
2. `(has(self.public) && self.public) || has(self.secretRef)` — confidential needs `secretRef`.
3. `(has(self.public) && self.public) || (has(self.redirectURIs) && self.redirectURIs.size() > 0)` —
   confidential needs `redirectURIs`; public clients may omit them and get Dex's
   loopback/OOB/device-flow defaults.

`public` carries `omitempty` and no default, so the rules must guard with
`has(self.public)` — a bare `self.public` would error on objects that never set it.
The rules are covered by envtest integration tests (the envtest apiserver enforces CEL).

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
