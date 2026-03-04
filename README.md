# dex-operator

A Kubernetes Operator (Go, controller-runtime) that dynamically assembles [Dex](https://dexidp.io/) configuration from Custom Resources.

Dex is still installed via the official [Dex Helm Chart](https://github.com/dexidp/helm-charts). The operator generates two Secrets in the Dex namespace:

- **Config-Secret** — Full Dex configuration as YAML (Issuer, Storage, Web, CORS, gRPC, Logger, Expiry, Connectors, Static Clients)
- **Env-Secret** — All client secrets as environment variables (e.g. `GRAFANA_CLIENT_SECRET`), attached to the Dex container via `envFrom` and referenced in the config as `$ENV_VAR`

## API Group

`dex.gtrfc.com/v1`

## CRDs (all namespace-scoped)

| CRD | Description |
|-----|-------------|
| `DexInstallation` | Global Dex config: Issuer, Storage, Web, CORS, gRPC, Logger, Expiry. Defines allowed namespaces and optional auto-restart. |
| `DexStaticClient` | OAuth2 static client referencing a `DexInstallation` and an existing Secret for client-id / client-secret. |
| `DexLDAPConnector` | LDAP identity provider connector |
| `DexGitHubConnector` | GitHub identity provider connector |
| `DexOIDCConnector` | Generic OIDC identity provider connector |
| `DexSAMLConnector` | SAML 2.0 identity provider connector |
| `DexGitLabConnector` | GitLab identity provider connector |
| `DexOAuth2Connector` | Generic OAuth2 identity provider connector |
| `DexGoogleConnector` | Google identity provider connector |
| `DexMicrosoftConnector` | Microsoft identity provider connector |
| `DexLinkedInConnector` | LinkedIn identity provider connector |
| `DexAuthProxyConnector` | AuthProxy connector |
| `DexBitbucketConnector` | Bitbucket identity provider connector |
| `DexLocalConnector` | Local (built-in) connector |
| `DexOpenShiftConnector` | OpenShift identity provider connector |
| `DexAtlassianCrowdConnector` | Atlassian Crowd connector |
| `DexGiteaConnector` | Gitea identity provider connector |
| `DexKeystoneConnector` | OpenStack Keystone connector |

## Architecture

- The operator watches all namespaces.
- On `DexInstallation` reconciliation: all associated connectors and static clients from allowed namespaces are collected, the config YAML and env-secret are built, and the secrets are written to the Dex namespace.
- On reconciliation of a client or connector: the referenced `DexInstallation` is re-triggered.
- Namespace whitelist validation on every client/connector.
- Optional rollout restart of the Dex deployment on config change.

## Development

```bash
# Build the manager binary
make build

# Run locally (requires a valid kubeconfig)
make run

# Generate CRDs and DeepCopy code
make generate-all

# Run linting
make lint

# Run tests
make test
```

## Helm Chart

The operator is deployed via its own Helm chart located at `deploy/helm/dex-operator`.

Dex itself is installed via the official Helm chart:
<https://github.com/dexidp/helm-charts>
