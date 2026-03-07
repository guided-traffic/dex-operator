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

// DexOAuth2ConnectorSpec defines configuration for the generic Dex OAuth2
// connector.
type DexOAuth2ConnectorSpec struct {
	// InstallationRef references the DexInstallation this connector belongs to.
	// +kubebuilder:validation:Required
	InstallationRef InstallationRef `json:"installationRef"`

	// ID is the connector ID used internally by Dex. Defaults to metadata.name.
	// +optional
	ID string `json:"id,omitempty"`

	// DisplayName is the human-readable connector name shown on the Dex login page.
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// ClientIDRef references the Secret key holding the OAuth2 client ID.
	// +kubebuilder:validation:Required
	ClientIDRef SecretKeyRef `json:"clientIDRef"`

	// ClientSecretRef references the Secret key holding the OAuth2 client
	// secret.
	// +kubebuilder:validation:Required
	ClientSecretRef SecretKeyRef `json:"clientSecretRef"`

	// RedirectURI is the callback URL registered with the OAuth2 provider.
	// +optional
	RedirectURI string `json:"redirectURI,omitempty"`

	// AuthorizationURL is the provider's authorization endpoint.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	AuthorizationURL string `json:"authorizationURL"`

	// TokenURL is the provider's token endpoint.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	TokenURL string `json:"tokenURL"`

	// UserInfoURL is the optional provider's user info endpoint.
	// +optional
	UserInfoURL string `json:"userInfoURL,omitempty"`

	// Scopes lists the OAuth2 scopes to request.
	// +optional
	Scopes []string `json:"scopes,omitempty"`

	// RootCARef references the Secret key holding a CA bundle for the provider.
	// +optional
	RootCARef *SecretKeyRef `json:"rootCARef,omitempty"`

	// HeaderPrefix is prepended to the Authorization header value.
	// +optional
	HeaderPrefix string `json:"headerPrefix,omitempty"`

	// InsecureTrustEmail trusts the email claim without verifying email_verified.
	// +optional
	InsecureTrustEmail bool `json:"insecureTrustEmail,omitempty"`

	// ClaimMapping maps provider claim fields to Dex claim fields.
	// +optional
	ClaimMapping *OAuth2ClaimMapping `json:"claimMapping,omitempty"`
}

// OAuth2ClaimMapping maps provider claims to Dex's standard claims.
type OAuth2ClaimMapping struct {
	// PreferredUsername maps a provider claim to preferred_username.
	// +optional
	PreferredUsername string `json:"preferredUsername,omitempty"`

	// Email maps a provider claim to the email claim.
	// +optional
	Email string `json:"email,omitempty"`

	// Groups maps a provider claim to the groups claim.
	// +optional
	Groups string `json:"groups,omitempty"`
}

// DexOAuth2ConnectorStatus defines the observed state of DexOAuth2Connector.
type DexOAuth2ConnectorStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dexoauth2
// +kubebuilder:printcolumn:name="Installation",type=string,JSONPath=`.spec.installationRef.name`
// +kubebuilder:printcolumn:name="Auth URL",type=string,JSONPath=`.spec.authorizationURL`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DexOAuth2Connector is the Schema for the dexoauth2connectors API.
type DexOAuth2Connector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexOAuth2ConnectorSpec   `json:"spec,omitempty"`
	Status DexOAuth2ConnectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DexOAuth2ConnectorList contains a list of DexOAuth2Connector.
type DexOAuth2ConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexOAuth2Connector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DexOAuth2Connector{}, &DexOAuth2ConnectorList{})
}
