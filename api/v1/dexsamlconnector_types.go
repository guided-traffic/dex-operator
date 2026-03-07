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

// DexSAMLConnectorSpec defines configuration for the Dex SAML 2.0 connector.
type DexSAMLConnectorSpec struct {
	// InstallationRef references the DexInstallation this connector belongs to.
	// +kubebuilder:validation:Required
	InstallationRef InstallationRef `json:"installationRef"`

	// ID is the connector ID used internally by Dex. Defaults to metadata.name.
	// +optional
	ID string `json:"id,omitempty"`

	// DisplayName is the human-readable connector name shown on the Dex login page.
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// SSOURL is the IdP's SSO endpoint URL.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	SSOURL string `json:"ssoURL"`

	// CARef references the Secret key holding the PEM-encoded CA certificate
	// used to verify the IdP's signature.
	// +optional
	CARef *SecretKeyRef `json:"caRef,omitempty"`

	// CABundleRef references the Secret key holding a PEM CA bundle.
	// +optional
	CABundleRef *SecretKeyRef `json:"caBundleRef,omitempty"`

	// SSOIssuer is the entity issuer of the IdP. Required when the IdP sends
	// a different issuer than is configured.
	// +optional
	SSOIssuer string `json:"ssoIssuer,omitempty"`

	// EntityIssuer overrides the entity issuer Dex sends in SAML requests.
	// +optional
	EntityIssuer string `json:"entityIssuer,omitempty"`

	// RedirectURI is the ACS (Assertion Consumer Service) URL.
	// +optional
	RedirectURI string `json:"redirectURI,omitempty"`

	// NameIDPolicyFormat controls the NameID format requested from the IdP.
	// +optional
	NameIDPolicyFormat string `json:"nameIDPolicyFormat,omitempty"`

	// UsernameAttr is the SAML attribute mapped to the username claim.
	// +optional
	UsernameAttr string `json:"usernameAttr,omitempty"`

	// EmailAttr is the SAML attribute mapped to the email claim.
	// +optional
	EmailAttr string `json:"emailAttr,omitempty"`

	// GroupsAttr is the SAML attribute mapped to the groups claim.
	// +optional
	GroupsAttr string `json:"groupsAttr,omitempty"`

	// AllowedGroups restricts login to users who are members of at least one
	// of the listed groups.
	// +optional
	AllowedGroups []string `json:"allowedGroups,omitempty"`

	// InsecureSkipSignatureValidation disables assertion signature verification.
	// WARNING: only for development/testing.
	// +optional
	InsecureSkipSignatureValidation bool `json:"insecureSkipSignatureValidation,omitempty"`
}

// DexSAMLConnectorStatus defines the observed state of DexSAMLConnector.
type DexSAMLConnectorStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dexsaml
// +kubebuilder:printcolumn:name="Installation",type=string,JSONPath=`.spec.installationRef.name`
// +kubebuilder:printcolumn:name="SSO URL",type=string,JSONPath=`.spec.ssoURL`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DexSAMLConnector is the Schema for the dexsamlconnectors API.
type DexSAMLConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexSAMLConnectorSpec   `json:"spec,omitempty"`
	Status DexSAMLConnectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DexSAMLConnectorList contains a list of DexSAMLConnector.
type DexSAMLConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexSAMLConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DexSAMLConnector{}, &DexSAMLConnectorList{})
}
