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

package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// DexInstallationSpec defines the desired state of DexInstallation.
type DexInstallationSpec struct {
	// Issuer is the base URL for the OpenID Connect issuer.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	Issuer string `json:"issuer"`

	// Storage configures the Dex storage backend.
	// +kubebuilder:validation:Required
	Storage DexStorageSpec `json:"storage"`

	// Web configures the HTTP/HTTPS endpoints of Dex.
	// +optional
	Web *DexWebSpec `json:"web,omitempty"`

	// CORS configures Cross-Origin Resource Sharing.
	// +optional
	CORS *DexCORSSpec `json:"cors,omitempty"`

	// GRPC configures the optional gRPC API server.
	// +optional
	GRPC *DexGRPCSpec `json:"grpc,omitempty"`

	// Logger configures Dex logging.
	// +optional
	Logger *DexLoggerSpec `json:"logger,omitempty"`

	// Expiry configures token expiration.
	// +optional
	Expiry *DexExpirySpec `json:"expiry,omitempty"`

	// OAuth2 configures the OAuth2 behaviour of Dex.
	// +optional
	OAuth2 *DexOAuth2ConfigSpec `json:"oauth2,omitempty"`

	// ConfigSecretName is the name of the Secret written to the Dex namespace
	// that holds the rendered Dex config.yaml.
	// +kubebuilder:validation:Required
	ConfigSecretName string `json:"configSecretName"`

	// EnvSecretName is the name of the Secret written to the Dex namespace
	// that holds all client secrets as env variables.
	// +kubebuilder:validation:Required
	EnvSecretName string `json:"envSecretName"`

	// AllowedNamespaces is a list of namespaces from which Connectors and
	// StaticClients can reference this installation.
	// Use ["*"] to allow all namespaces.
	// +optional
	AllowedNamespaces []string `json:"allowedNamespaces,omitempty"`

	// RolloutRestart configures an optional automated rollout restart of the
	// Dex Deployment when the config changes.
	// +optional
	RolloutRestart *RolloutRestartSpec `json:"rolloutRestart,omitempty"`
}

// StorageType enumerates supported Dex storage backends.
// +kubebuilder:validation:Enum=kubernetes;memory;postgres;sqlite3;etcd;mysql
type StorageType string

const (
	StorageKubernetes StorageType = "kubernetes"
	StorageMemory     StorageType = "memory"
	StoragePostgres   StorageType = "postgres"
	StorageSQLite3    StorageType = "sqlite3"
	StorageEtcd       StorageType = "etcd"
	StorageMySQL      StorageType = "mysql"
)

// DexStorageSpec configures the Dex storage backend.
type DexStorageSpec struct {
	// Type selects the storage backend.
	// +kubebuilder:validation:Required
	Type StorageType `json:"type"`

	// Postgres configures a PostgreSQL storage backend.
	// +optional
	Postgres *DexPostgresStorageSpec `json:"postgres,omitempty"`

	// SQLite3 configures a SQLite3 file storage backend.
	// +optional
	SQLite3 *DexSQLite3StorageSpec `json:"sqlite3,omitempty"`

	// Etcd configures an etcd storage backend.
	// +optional
	Etcd *DexEtcdStorageSpec `json:"etcd,omitempty"`

	// MySQL configures a MySQL storage backend.
	// +optional
	MySQL *DexMySQLStorageSpec `json:"mysql,omitempty"`
}

// DexPostgresStorageSpec configures the PostgreSQL storage backend.
type DexPostgresStorageSpec struct {
	// Host is the PostgreSQL host (host:port).
	// +kubebuilder:validation:Required
	Host string `json:"host"`

	// Database is the PostgreSQL database name.
	// +kubebuilder:validation:Required
	Database string `json:"database"`

	// User is the PostgreSQL user.
	// +kubebuilder:validation:Required
	User string `json:"user"`

	// PasswordRef references the Secret key holding the PostgreSQL password.
	// +optional
	PasswordRef *SecretKeyRef `json:"passwordRef,omitempty"`

	// SSL configures PostgreSQL TLS.
	// +optional
	SSL *DexPostgresSSLSpec `json:"ssl,omitempty"`

	// ConnectionTimeout in seconds.
	// +optional
	ConnectionTimeout *int `json:"connectionTimeout,omitempty"`

	// MaxOpenConns limits the number of open connections.
	// +optional
	MaxOpenConns *int `json:"maxOpenConns,omitempty"`

	// MaxIdleConns limits the number of idle connections.
	// +optional
	MaxIdleConns *int `json:"maxIdleConns,omitempty"`

	// ConnMaxLifetime in seconds.
	// +optional
	ConnMaxLifetime *int `json:"connMaxLifetime,omitempty"`
}

// DexPostgresSSLSpec configures TLS for the PostgreSQL backend.
type DexPostgresSSLSpec struct {
	// Mode is the SSL mode (disable, require, verify-ca, verify-full).
	// +kubebuilder:validation:Enum=disable;require;verify-ca;verify-full
	// +optional
	Mode string `json:"mode,omitempty"`

	// CARef references the CA certificate secret key.
	// +optional
	CARef *SecretKeyRef `json:"caRef,omitempty"`

	// CertRef references the client certificate secret key.
	// +optional
	CertRef *SecretKeyRef `json:"certRef,omitempty"`

	// KeyRef references the client key secret key.
	// +optional
	KeyRef *SecretKeyRef `json:"keyRef,omitempty"`
}

// DexSQLite3StorageSpec configures a SQLite3 file backend.
type DexSQLite3StorageSpec struct {
	// File is the path to the SQLite3 database file.
	// +kubebuilder:validation:Required
	File string `json:"file"`
}

// DexEtcdStorageSpec configures the etcd storage backend.
type DexEtcdStorageSpec struct {
	// Endpoints is a list of etcd endpoints.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Endpoints []string `json:"endpoints"`

	// Namespace is the etcd key prefix.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Username is the etcd basic auth username.
	// +optional
	Username string `json:"username,omitempty"`

	// PasswordRef references the etcd password secret key.
	// +optional
	PasswordRef *SecretKeyRef `json:"passwordRef,omitempty"`

	// SSL configures etcd TLS.
	// +optional
	SSL *DexEtcdSSLSpec `json:"ssl,omitempty"`
}

// DexEtcdSSLSpec configures TLS for the etcd backend.
type DexEtcdSSLSpec struct {
	// ServerName overrides the TLS server name.
	// +optional
	ServerName string `json:"serverName,omitempty"`

	// CARef references the CA certificate secret key.
	// +optional
	CARef *SecretKeyRef `json:"caRef,omitempty"`

	// CertRef references the client certificate secret key.
	// +optional
	CertRef *SecretKeyRef `json:"certRef,omitempty"`

	// KeyRef references the client key secret key.
	// +optional
	KeyRef *SecretKeyRef `json:"keyRef,omitempty"`
}

// DexMySQLStorageSpec configures the MySQL storage backend.
type DexMySQLStorageSpec struct {
	// DSN is the MySQL data source name.
	// +optional
	DSN string `json:"dsn,omitempty"`

	// DSNRef references the Secret key holding the MySQL DSN.
	// +optional
	DSNRef *SecretKeyRef `json:"dsnRef,omitempty"`

	// MaxOpenConns limits the number of open connections.
	// +optional
	MaxOpenConns *int `json:"maxOpenConns,omitempty"`

	// MaxIdleConns limits the number of idle connections.
	// +optional
	MaxIdleConns *int `json:"maxIdleConns,omitempty"`

	// ConnMaxLifetime in seconds.
	// +optional
	ConnMaxLifetime *int `json:"connMaxLifetime,omitempty"`
}

// DexWebSpec configures the Dex HTTP/HTTPS endpoints.
type DexWebSpec struct {
	// HTTP is the address for the HTTP endpoint (e.g. 0.0.0.0:5556).
	// +optional
	HTTP string `json:"http,omitempty"`

	// HTTPS is the address for the HTTPS endpoint (e.g. 0.0.0.0:5554).
	// +optional
	HTTPS string `json:"https,omitempty"`

	// TLSCert is the path to the TLS certificate file.
	// +optional
	TLSCert string `json:"tlsCert,omitempty"`

	// TLSKey is the path to the TLS key file.
	// +optional
	TLSKey string `json:"tlsKey,omitempty"`

	// AllowedOrigins lists origins from which cross-origin requests are allowed.
	// +optional
	AllowedOrigins []string `json:"allowedOrigins,omitempty"`
}

// DexCORSSpec configures CORS for the Dex web server.
type DexCORSSpec struct {
	// AllowedOrigins lists origins that may send cross-origin requests.
	// +optional
	AllowedOrigins []string `json:"allowedOrigins,omitempty"`

	// AllowedHeaders lists additional headers to allow in CORS requests.
	// +optional
	AllowedHeaders []string `json:"allowedHeaders,omitempty"`
}

// DexGRPCSpec configures the optional Dex gRPC API.
type DexGRPCSpec struct {
	// Addr is the gRPC address (e.g. 0.0.0.0:5557).
	// +optional
	Addr string `json:"addr,omitempty"`

	// TLSCert is the path to the TLS certificate file.
	// +optional
	TLSCert string `json:"tlsCert,omitempty"`

	// TLSKey is the path to the TLS key file.
	// +optional
	TLSKey string `json:"tlsKey,omitempty"`

	// TLSClientCA is the path to the client CA certificate file.
	// +optional
	TLSClientCA string `json:"tlsClientCA,omitempty"`

	// Reflection enables gRPC server reflection.
	// +optional
	Reflection bool `json:"reflection,omitempty"`
}

// DexLoggerSpec configures Dex logging.
type DexLoggerSpec struct {
	// Level sets the log level (debug, info, warning, error).
	// +kubebuilder:validation:Enum=debug;info;warning;error
	// +optional
	Level string `json:"level,omitempty"`

	// Format sets the log format (json, text).
	// +kubebuilder:validation:Enum=json;text
	// +optional
	Format string `json:"format,omitempty"`
}

// DexExpirySpec configures token expiry durations.
// Durations are expressed as Go duration strings (e.g. "24h", "30m").
type DexExpirySpec struct {
	// SigningKeys is the period after which Dex rotates signing keys.
	// +optional
	SigningKeys string `json:"signingKeys,omitempty"`

	// IDTokens is the lifetime of issued ID tokens.
	// +optional
	IDTokens string `json:"idTokens,omitempty"`

	// AuthRequests is the lifetime of pending auth requests.
	// +optional
	AuthRequests string `json:"authRequests,omitempty"`

	// DeviceRequests is the lifetime of pending device requests.
	// +optional
	DeviceRequests string `json:"deviceRequests,omitempty"`

	// RefreshTokens configures refresh token expiry.
	// +optional
	RefreshTokens *DexRefreshTokenExpirySpec `json:"refreshTokens,omitempty"`
}

// DexRefreshTokenExpirySpec configures refresh token lifetime.
type DexRefreshTokenExpirySpec struct {
	// DisableRotation disables refresh token rotation.
	// +optional
	DisableRotation bool `json:"disableRotation,omitempty"`

	// ReuseInterval is the minimum time before a new refresh token is issued.
	// +optional
	ReuseInterval string `json:"reuseInterval,omitempty"`

	// ValidIfNotUsedFor invalidates the token if not used for this duration.
	// +optional
	ValidIfNotUsedFor string `json:"validIfNotUsedFor,omitempty"`

	// AbsoluteLifetime is the absolute maximum lifetime of a refresh token.
	// +optional
	AbsoluteLifetime string `json:"absoluteLifetime,omitempty"`
}

// DexOAuth2ConfigSpec configures Dex's OAuth2 behaviour.
type DexOAuth2ConfigSpec struct {
	// ResponseTypes restricts which response_type values are supported.
	// +optional
	ResponseTypes []string `json:"responseTypes,omitempty"`

	// SkipApprovalScreen skips the UI approval screen on every auth request.
	// +optional
	SkipApprovalScreen bool `json:"skipApprovalScreen,omitempty"`

	// AlwaysShowLoginScreen always shows the login screen even for single connectors.
	// +optional
	AlwaysShowLoginScreen bool `json:"alwaysShowLoginScreen,omitempty"`

	// GrantTypes lists allowed OAuth2 grant types.
	// +optional
	GrantTypes []string `json:"grantTypes,omitempty"`

	// PasswordConnector sets the connector used for the Resource Owner Password
	// Credentials grant type.
	// +optional
	PasswordConnector string `json:"passwordConnector,omitempty"`
}

// RolloutRestartSpec configures optional automated rollout restart.
type RolloutRestartSpec struct {
	// Enabled toggles the automatic rollout restart.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// DeploymentName is the name of the Dex Deployment to restart.
	// +optional
	DeploymentName string `json:"deploymentName,omitempty"`
}

// DexInstallationStatus defines the observed state of DexInstallation.
type DexInstallationStatus struct {
	CommonStatus `json:",inline"`

	// ConnectorCount is the number of connectors currently reconciled.
	// +optional
	ConnectorCount int `json:"connectorCount,omitempty"`

	// StaticClientCount is the number of static clients currently reconciled.
	// +optional
	StaticClientCount int `json:"staticClientCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dexinst
// +kubebuilder:printcolumn:name="Issuer",type=string,JSONPath=`.spec.issuer`
// +kubebuilder:printcolumn:name="Storage",type=string,JSONPath=`.spec.storage.type`
// +kubebuilder:printcolumn:name="Connectors",type=integer,JSONPath=`.status.connectorCount`
// +kubebuilder:printcolumn:name="Clients",type=integer,JSONPath=`.status.staticClientCount`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DexInstallation is the Schema for the dexinstallations API.
// It describes a complete Dex OIDC provider installation including issuer,
// storage, web, gRPC, logger, and expiry configuration.
type DexInstallation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexInstallationSpec   `json:"spec,omitempty"`
	Status DexInstallationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DexInstallationList contains a list of DexInstallation.
type DexInstallationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexInstallation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DexInstallation{}, &DexInstallationList{})
}
