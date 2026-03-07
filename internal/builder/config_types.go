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

// DexConfig is the top-level Dex configuration structure written to the
// config Secret. It mirrors the official Dex config.yaml schema.
type DexConfig struct {
	Issuer           string           `yaml:"issuer"`
	Storage          StorageConfig    `yaml:"storage"`
	Web              *WebConfig       `yaml:"web,omitempty"`
	CORS             *CORSConfig      `yaml:"cors,omitempty"`
	GRPC             *GRPCConfig      `yaml:"grpc,omitempty"`
	Logger           *LoggerConfig    `yaml:"logger,omitempty"`
	Expiry           *ExpiryConfig    `yaml:"expiry,omitempty"`
	OAuth2           *OAuth2Config    `yaml:"oauth2,omitempty"`
	Connectors       []ConnectorEntry `yaml:"connectors,omitempty"`
	StaticClients    []StaticClient   `yaml:"staticClients,omitempty"`
	EnablePasswordDB bool             `yaml:"enablePasswordDB,omitempty"`
}

// StorageConfig describes the Dex storage backend.
type StorageConfig struct {
	Type   string         `yaml:"type"`
	Config map[string]any `yaml:"config,omitempty"`
}

// WebConfig describes the HTTP/HTTPS endpoints.
type WebConfig struct {
	HTTP           string   `yaml:"http,omitempty"`
	HTTPS          string   `yaml:"https,omitempty"`
	TLSCert        string   `yaml:"tlsCert,omitempty"`
	TLSKey         string   `yaml:"tlsKey,omitempty"`
	AllowedOrigins []string `yaml:"allowedOrigins,omitempty"`
}

// CORSConfig configures Cross-Origin Resource Sharing.
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowedOrigins,omitempty"`
	AllowedHeaders []string `yaml:"allowedHeaders,omitempty"`
}

// GRPCConfig describes the optional Dex gRPC API server.
type GRPCConfig struct {
	Addr        string `yaml:"addr,omitempty"`
	TLSCert     string `yaml:"tlsCert,omitempty"`
	TLSKey      string `yaml:"tlsKey,omitempty"`
	TLSClientCA string `yaml:"tlsClientCA,omitempty"`
	Reflection  bool   `yaml:"reflection,omitempty"`
}

// LoggerConfig configures Dex logging.
type LoggerConfig struct {
	Level  string `yaml:"level,omitempty"`
	Format string `yaml:"format,omitempty"`
}

// ExpiryConfig configures token expiry durations.
type ExpiryConfig struct {
	SigningKeys    string               `yaml:"signingKeys,omitempty"`
	IDTokens       string               `yaml:"idTokens,omitempty"`
	AuthRequests   string               `yaml:"authRequests,omitempty"`
	DeviceRequests string               `yaml:"deviceRequests,omitempty"`
	RefreshTokens  *RefreshTokensConfig `yaml:"refreshTokens,omitempty"`
}

// RefreshTokensConfig configures refresh token lifetime.
type RefreshTokensConfig struct {
	DisableRotation   bool   `yaml:"disableRotation,omitempty"`
	ReuseInterval     string `yaml:"reuseInterval,omitempty"`
	ValidIfNotUsedFor string `yaml:"validIfNotUsedFor,omitempty"`
	AbsoluteLifetime  string `yaml:"absoluteLifetime,omitempty"`
}

// OAuth2Config configures Dex's OAuth2 behaviour.
type OAuth2Config struct {
	ResponseTypes         []string `yaml:"responseTypes,omitempty"`
	SkipApprovalScreen    bool     `yaml:"skipApprovalScreen,omitempty"`
	AlwaysShowLoginScreen bool     `yaml:"alwaysShowLoginScreen,omitempty"`
	GrantTypes            []string `yaml:"grantTypes,omitempty"`
	PasswordConnector     string   `yaml:"passwordConnector,omitempty"`
}

// ConnectorEntry is a single entry in the Dex connectors list.
// Config holds connector-type-specific configuration as a generic map so
// that any connector type can be serialized without additional structs.
type ConnectorEntry struct {
	Type   string         `yaml:"type"`
	ID     string         `yaml:"id"`
	Name   string         `yaml:"name"`
	Config map[string]any `yaml:"config,omitempty"`
}

// StaticClient represents a statically configured OAuth2 client in Dex.
type StaticClient struct {
	ID            string   `yaml:"id"`
	Secret        string   `yaml:"secret,omitempty"`
	Name          string   `yaml:"name,omitempty"`
	RedirectURIs  []string `yaml:"redirectURIs,omitempty"`
	AllowedScopes []string `yaml:"allowedScopes,omitempty"`
	TrustedPeers  []string `yaml:"trustedPeers,omitempty"`
	Public        bool     `yaml:"public,omitempty"`
}
