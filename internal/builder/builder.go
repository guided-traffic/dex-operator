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

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
	"gopkg.in/yaml.v3"
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

	cfg := assembleDexConfig(in.Installation.Spec, storage, connEntries, clients, len(in.Connectors.Local) > 0)

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
) DexConfig {
	cfg := DexConfig{
		Issuer:           spec.Issuer,
		Storage:          storage,
		Connectors:       connectors,
		StaticClients:    clients,
		EnablePasswordDB: enablePasswordDB,
	}

	if spec.Web != nil {
		cfg.Web = &WebConfig{
			HTTP:           spec.Web.HTTP,
			HTTPS:          spec.Web.HTTPS,
			TLSCert:        spec.Web.TLSCert,
			TLSKey:         spec.Web.TLSKey,
			AllowedOrigins: spec.Web.AllowedOrigins,
		}
	}

	if spec.CORS != nil {
		cfg.CORS = &CORSConfig{
			AllowedOrigins: spec.CORS.AllowedOrigins,
			AllowedHeaders: spec.CORS.AllowedHeaders,
		}
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

	return cfg
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
