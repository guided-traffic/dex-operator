/*
Copyright 2025 Guided Traffic GmbH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package builder

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// SecretResolver resolves a [dexv1.SecretKeyRef] to its plaintext string value.
// The namespace parameter identifies where the Secret lives.
type SecretResolver func(ctx context.Context, namespace string, ref dexv1.SecretKeyRef) (string, error)

// ConnectorSet groups all connector resources by type.
type ConnectorSet struct {
	LDAP           []dexv1.DexLDAPConnector
	GitHub         []dexv1.DexGitHubConnector
	SAML           []dexv1.DexSAMLConnector
	GitLab         []dexv1.DexGitLabConnector
	OIDC           []dexv1.DexOIDCConnector
	OAuth2         []dexv1.DexOAuth2Connector
	Google         []dexv1.DexGoogleConnector
	LinkedIn       []dexv1.DexLinkedInConnector
	Microsoft      []dexv1.DexMicrosoftConnector
	AuthProxy      []dexv1.DexAuthProxyConnector
	Bitbucket      []dexv1.DexBitbucketConnector
	Local          []dexv1.DexLocalConnector
	OpenShift      []dexv1.DexOpenShiftConnector
	AtlassianCrowd []dexv1.DexAtlassianCrowdConnector
	Gitea          []dexv1.DexGiteaConnector
	Keystone       []dexv1.DexKeystoneConnector
}

// Input holds every resource required to produce one complete Dex config.
type Input struct {
	// Installation is the DexInstallation that owns this config.
	Installation *dexv1.DexInstallation
	// Connectors contains all connector resources grouped by type.
	Connectors ConnectorSet
	// StaticClients contains all DexStaticClient resources for this installation.
	StaticClients []dexv1.DexStaticClient
	// Secrets is called to resolve every SecretKeyRef encountered during build.
	Secrets SecretResolver
}

// MountedSecret describes a Kubernetes Secret key that must be projected as a
// file into the Dex container (e.g. a TLS certificate or CA bundle).
type MountedSecret struct {
	// Namespace is the namespace that contains the Secret.
	Namespace string
	// SecretName is the name of the Secret.
	SecretName string
	// SecretKey is the key within the Secret.
	SecretKey string
	// MountPath is the absolute file path inside the Dex container.
	MountPath string
}

// Output is the result of a successful [Build] call.
type Output struct {
	// ConfigYAML is the rendered Dex config.yaml ready to be stored in a Secret.
	ConfigYAML []byte
	// EnvSecretData maps environment variable names to their values for the
	// companion env Secret (mounted via envFrom on the Dex container).
	EnvSecretData map[string][]byte
	// MountedSecrets lists Secret keys that must be projected as files.
	MountedSecrets []MountedSecret
}

// Build constructs the Dex config YAML and companion env Secret data from the
// provided [Input].  It calls [Input.Secrets] for every referenced Secret key.
func Build(ctx context.Context, in Input) (Output, error) {
	envs := make(map[string][]byte)
	var mounts []MountedSecret

	storage, storageMounts, err := buildStorage(ctx, in.Installation.Spec.Storage, in.Secrets, in.Installation.Namespace, envs)
	if err != nil {
		return Output{}, fmt.Errorf("building storage config: %w", err)
	}
	mounts = append(mounts, storageMounts...)

	connEntries, connMounts, err := buildAllConnectors(ctx, in.Connectors, in.Secrets, envs)
	if err != nil {
		return Output{}, fmt.Errorf("building connector configs: %w", err)
	}
	mounts = append(mounts, connMounts...)

	clients, err := buildStaticClients(ctx, in.StaticClients, in.Secrets, envs)
	if err != nil {
		return Output{}, fmt.Errorf("building static client configs: %w", err)
	}

	corsOrigins := deriveCORSOrigins(in.StaticClients)

	cfg := assembleDexConfig(in.Installation.Spec, storage, connEntries, clients, len(in.Connectors.Local) > 0, corsOrigins)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return Output{}, fmt.Errorf("marshaling dex config to YAML: %w", err)
	}

	return Output{
		ConfigYAML:     data,
		EnvSecretData:  envs,
		MountedSecrets: mounts,
	}, nil
}

// assembleDexConfig builds the [DexConfig] from an installation spec and the
// already-built sub-components.
func assembleDexConfig(
	spec dexv1.DexInstallationSpec,
	storage StorageConfig,
	connectors []ConnectorEntry,
	clients []StaticClient,
	enablePasswordDB bool,
	corsOrigins []string,
) DexConfig {
	cfg := DexConfig{
		Issuer:           spec.Issuer,
		Storage:          storage,
		Connectors:       connectors,
		StaticClients:    clients,
		EnablePasswordDB: enablePasswordDB,
		Web:              assembleWebConfig(spec.Web, corsOrigins),
	}

	if spec.GRPC != nil {
		cfg.GRPC = assembleGRPCConfig(spec.GRPC)
	}

	if spec.Logger != nil {
		cfg.Logger = &LoggerConfig{
			Level:  spec.Logger.Level,
			Format: spec.Logger.Format,
		}
	}

	if spec.Expiry != nil {
		cfg.Expiry = assembleExpiryConfig(spec.Expiry)
	}

	if spec.OAuth2 != nil {
		cfg.OAuth2 = assembleOAuth2Config(spec.OAuth2)
	}

	if spec.Frontend != nil {
		cfg.Frontend = &FrontendConfig{
			Dir:     spec.Frontend.Dir,
			Theme:   spec.Frontend.Theme,
			Issuer:  spec.Frontend.Issuer,
			LogoURL: spec.Frontend.LogoURL,
		}
	}

	return cfg
}

// assembleWebConfig renders the web block from the installation spec plus the
// CORS origins derived from static clients.  It returns nil when neither
// contributes anything, so an installation without web configuration keeps a
// config free of an empty web: block.  Origins alone create the block: the
// listener addresses come from the dex binary's --web-http-addr/--web-https-addr
// flags (set by the dexidp helm chart), which are applied after config load.
func assembleWebConfig(spec *dexv1.DexWebSpec, corsOrigins []string) *WebConfig {
	if spec == nil && len(corsOrigins) == 0 {
		return nil
	}

	web := &WebConfig{}
	if spec != nil {
		web.HTTP = spec.HTTP
		web.HTTPS = spec.HTTPS
		web.TLSCert = spec.TLSCert
		web.TLSKey = spec.TLSKey
		web.AllowedOrigins = spec.AllowedOrigins
		web.AllowedHeaders = spec.AllowedHeaders
	}
	web.AllowedOrigins = appendDerivedOrigins(web.AllowedOrigins, corsOrigins)

	return web
}

// appendDerivedOrigins appends every derived origin that is not already present
// to base and returns the result, leaving base itself untouched.  The
// installation's authored order is preserved so that an operator upgrade alone
// never rewrites an existing allowedOrigins list (which would diff the config
// and trigger a spurious dex rollout); the derived tail is sorted, making the
// rendered YAML independent of client iteration order.
func appendDerivedOrigins(base, derived []string) []string {
	if len(derived) == 0 {
		return base
	}

	seen := make(map[string]struct{}, len(base)+len(derived))
	for _, o := range base {
		seen[o] = struct{}{}
	}

	extra := make([]string, 0, len(derived))
	for _, o := range derived {
		if _, dup := seen[o]; dup {
			continue
		}
		seen[o] = struct{}{}
		extra = append(extra, o)
	}
	if len(extra) == 0 {
		return base
	}
	sort.Strings(extra)

	out := make([]string, 0, len(base)+len(extra))
	out = append(out, base...)
	return append(out, extra...)
}

// deriveCORSOrigins collects the browser origins of every static client that
// opted in via spec.cors.  Origins are derived from the client's own
// redirectURIs, so the flag grants no authority beyond that already
// RBAC-gated field.  Only https URLs contribute: loopback/http targets, custom
// schemes and the OOB URN are redirect conveniences for native clients, not
// browser origins.  Unparsable entries are skipped instead of failing the
// build — one malformed tenant resource must not break the render for every
// other client of the installation.
func deriveCORSOrigins(clients []dexv1.DexStaticClient) []string {
	var origins []string
	seen := make(map[string]struct{})

	for i := range clients {
		if !clients[i].Spec.CORS {
			continue
		}
		for _, uri := range clients[i].Spec.RedirectURIs {
			origin, ok := originFromRedirectURI(uri)
			if !ok {
				continue
			}
			if _, dup := seen[origin]; dup {
				continue
			}
			seen[origin] = struct{}{}
			origins = append(origins, origin)
		}
	}

	return origins
}

// originFromRedirectURI reduces an https redirect URI to the exact
// scheme://host[:port] string a browser sends in its Origin header.  The host
// is lowercased and a redundant :443 dropped, because Dex matches origins
// literally (gorilla/handlers) and would otherwise silently never match.
func originFromRedirectURI(uri string) (string, bool) {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return "", false
	}
	host := strings.TrimSuffix(strings.ToLower(u.Host), ":443")
	return "https://" + host, true
}

func assembleGRPCConfig(s *dexv1.DexGRPCSpec) *GRPCConfig {
	return &GRPCConfig{
		Addr:        s.Addr,
		TLSCert:     s.TLSCert,
		TLSKey:      s.TLSKey,
		TLSClientCA: s.TLSClientCA,
		Reflection:  s.Reflection,
	}
}

func assembleExpiryConfig(s *dexv1.DexExpirySpec) *ExpiryConfig {
	e := &ExpiryConfig{
		SigningKeys:    s.SigningKeys,
		IDTokens:       s.IDTokens,
		AuthRequests:   s.AuthRequests,
		DeviceRequests: s.DeviceRequests,
	}
	if s.RefreshTokens != nil {
		e.RefreshTokens = &RefreshTokensConfig{
			DisableRotation:   s.RefreshTokens.DisableRotation,
			ReuseInterval:     s.RefreshTokens.ReuseInterval,
			ValidIfNotUsedFor: s.RefreshTokens.ValidIfNotUsedFor,
			AbsoluteLifetime:  s.RefreshTokens.AbsoluteLifetime,
		}
	}
	return e
}

func assembleOAuth2Config(s *dexv1.DexOAuth2ConfigSpec) *OAuth2Config {
	return &OAuth2Config{
		ResponseTypes:         s.ResponseTypes,
		SkipApprovalScreen:    s.SkipApprovalScreen,
		AlwaysShowLoginScreen: s.AlwaysShowLoginScreen,
		GrantTypes:            s.GrantTypes,
		PasswordConnector:     s.PasswordConnector,
	}
}

// resolveEnvSecret resolves a SecretKeyRef, stores the value under envKey in
// envs, and returns the "$envKey" substitution reference.
func resolveEnvSecret(ctx context.Context, namespace string, ref dexv1.SecretKeyRef, envKey string, sr SecretResolver, envs map[string][]byte) (string, error) {
	val, err := sr(ctx, namespace, ref)
	if err != nil {
		return "", fmt.Errorf("secret %s/%s[%s]: %w", namespace, ref.Name, ref.Key, err)
	}
	return envRef(envKey, val, envs), nil
}

// resolveSecret resolves a SecretKeyRef and returns the plaintext value
// without creating an env var entry.
func resolveSecret(ctx context.Context, namespace string, ref dexv1.SecretKeyRef, sr SecretResolver) (string, error) {
	val, err := sr(ctx, namespace, ref)
	if err != nil {
		return "", fmt.Errorf("secret %s/%s[%s]: %w", namespace, ref.Name, ref.Key, err)
	}
	return val, nil
}

// mountCertFile registers a PEM certificate Secret key as a mounted file and
// returns the deterministic file path used in the Dex config.
func mountCertFile(ref dexv1.SecretKeyRef, namespace, connectorID, fieldName string, mounts *[]MountedSecret) string {
	path := fmt.Sprintf("/etc/dex/certs/%s-%s.pem", connectorID, fieldName)
	return mountSecretAsFile(ref, namespace, path, mounts)
}

// mountSecretAsFile registers an arbitrary Secret key as a mounted file at
// the given absolute path and returns that path.
func mountSecretAsFile(ref dexv1.SecretKeyRef, namespace, path string, mounts *[]MountedSecret) string {
	*mounts = append(*mounts, MountedSecret{
		Namespace:  namespace,
		SecretName: ref.Name,
		SecretKey:  ref.Key,
		MountPath:  path,
	})
	return path
}

// connectorID returns the effective connector ID, falling back to metaName if
// the spec ID is empty.
func connectorID(metaName, specID string) string {
	if specID != "" {
		return specID
	}
	return metaName
}
