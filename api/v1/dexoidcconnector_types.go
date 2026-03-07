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

// DexOIDCConnectorSpec defines configuration for the Dex OIDC connector.
type DexOIDCConnectorSpec struct {
	// InstallationRef references the DexInstallation this connector belongs to.
	// +kubebuilder:validation:Required
	InstallationRef InstallationRef `json:"installationRef"`

	// ID is the connector ID used internally by Dex. Defaults to metadata.name.
	// +optional
	ID string `json:"id,omitempty"`

	// DisplayName is the human-readable connector name shown on the Dex login page.
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// Issuer is the OIDC issuer URL of the upstream IdP.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	Issuer string `json:"issuer"`

	// ClientIDRef references the Secret key holding the OIDC client ID.
	// +kubebuilder:validation:Required
	ClientIDRef SecretKeyRef `json:"clientIDRef"`

	// ClientSecretRef references the Secret key holding the OIDC client secret.
	// +kubebuilder:validation:Required
	ClientSecretRef SecretKeyRef `json:"clientSecretRef"`

	// RedirectURI is the callback URL registered with the upstream IdP.
	// +optional
	RedirectURI string `json:"redirectURI,omitempty"`

	// Scopes is the list of additional scopes to request from the IdP.
	// openid is always included.
	// +optional
	Scopes []string `json:"scopes,omitempty"`

	// GetUserInfo fetches additional claims from the userinfo endpoint.
	// +optional
	GetUserInfo bool `json:"getUserInfo,omitempty"`

	// UserNameKey is the claim name used as the username.
	// +optional
	UserNameKey string `json:"userNameKey,omitempty"`

	// UserIDKey is the claim name used as the user's stable ID.
	// +optional
	UserIDKey string `json:"userIDKey,omitempty"`

	// PromptType overrides the prompt parameter sent to the upstream IdP.
	// +optional
	PromptType string `json:"promptType,omitempty"`

	// OverrideClaimMapping enables overriding claim mapping with ClaimMapping.
	// +optional
	OverrideClaimMapping bool `json:"overrideClaimMapping,omitempty"`

	// ClaimMapping maps upstream claims to Dex claims.
	// +optional
	ClaimMapping *OIDCClaimMapping `json:"claimMapping,omitempty"`

	// InsecureSkipEmailVerified skips the email_verified claim check.
	// +optional
	InsecureSkipEmailVerified bool `json:"insecureSkipEmailVerified,omitempty"`

	// InsecureEnableGroups enables group claim support even when the IdP does
	// not include it in the discovery document.
	// +optional
	InsecureEnableGroups bool `json:"insecureEnableGroups,omitempty"`

	// BasicAuthUnsupported forces HTTP Basic Auth in token requests.
	// +optional
	BasicAuthUnsupported bool `json:"basicAuthUnsupported,omitempty"`

	// HostedDomains restricts authentication to users from the listed
	// Google-hosted domains (relevant for Google IdPs).
	// +optional
	HostedDomains []string `json:"hostedDomains,omitempty"`

	// ACRValues sets the acr_values parameter in the auth request.
	// +optional
	ACRValues []string `json:"acrValues,omitempty"`

	// DiscoveryPollInterval overrides how often the discovery document is
	// refreshed (Go duration string, e.g. "30m").
	// +optional
	DiscoveryPollInterval string `json:"discoveryPollInterval,omitempty"`

	// RootCARef references the Secret key holding a CA bundle for the upstream
	// IdP's HTTPS endpoint.
	// +optional
	RootCARef *SecretKeyRef `json:"rootCARef,omitempty"`
}

// OIDCClaimMapping maps upstream IdP claims to Dex claims.
type OIDCClaimMapping struct {
	// PreferredUsername is the upstream claim mapped to preferred_username.
	// +optional
	PreferredUsername string `json:"preferredUsername,omitempty"`

	// Email is the upstream claim mapped to the email claim.
	// +optional
	Email string `json:"email,omitempty"`

	// Groups is the upstream claim mapped to the groups claim.
	// +optional
	Groups string `json:"groups,omitempty"`
}

// DexOIDCConnectorStatus defines the observed state of DexOIDCConnector.
type DexOIDCConnectorStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dexoidc
// +kubebuilder:printcolumn:name="Installation",type=string,JSONPath=`.spec.installationRef.name`
// +kubebuilder:printcolumn:name="Issuer",type=string,JSONPath=`.spec.issuer`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DexOIDCConnector is the Schema for the dexoidcconnectors API.
type DexOIDCConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexOIDCConnectorSpec   `json:"spec,omitempty"`
	Status DexOIDCConnectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DexOIDCConnectorList contains a list of DexOIDCConnector.
type DexOIDCConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexOIDCConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DexOIDCConnector{}, &DexOIDCConnectorList{})
}
