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

// DexMicrosoftConnectorSpec defines configuration for the Dex Microsoft
// connector (Azure AD / Entra ID).
type DexMicrosoftConnectorSpec struct {
	// InstallationRef references the DexInstallation this connector belongs to.
	// +kubebuilder:validation:Required
	InstallationRef InstallationRef `json:"installationRef"`

	// ID is the connector ID used internally by Dex. Defaults to metadata.name.
	// +optional
	ID string `json:"id,omitempty"`

	// Name is the human-readable connector name shown on the Dex login page.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// ClientIDRef references the Secret key holding the Azure AD app client ID.
	// +kubebuilder:validation:Required
	ClientIDRef SecretKeyRef `json:"clientIDRef"`

	// ClientSecretRef references the Secret key holding the Azure AD app client
	// secret.
	// +kubebuilder:validation:Required
	ClientSecretRef SecretKeyRef `json:"clientSecretRef"`

	// RedirectURI is the callback URL registered with the Azure AD application.
	// +optional
	RedirectURI string `json:"redirectURI,omitempty"`

	// Tenant specifies the Azure AD tenant to authenticate against.
	// Use "common" for multi-tenant apps, "consumers" for personal Microsoft
	// accounts, or a specific tenant ID / name.
	// +optional
	Tenant string `json:"tenant,omitempty"`

	// OnlySecurityGroups limits group claims to security groups only.
	// +optional
	OnlySecurityGroups bool `json:"onlySecurityGroups,omitempty"`

	// Groups restricts login to users who are members of at least one of the
	// listed Azure AD groups.
	// +optional
	Groups []string `json:"groups,omitempty"`

	// GroupNameFormat controls whether group names are resolved as ObjectIDs or
	// display names. Allowed values: id, name.
	// +kubebuilder:validation:Enum=id;name
	// +optional
	GroupNameFormat string `json:"groupNameFormat,omitempty"`

	// DomainHint is appended as the domain_hint query parameter to the auth
	// request to pre-select a tenant.
	// +optional
	DomainHint string `json:"domainHint,omitempty"`
}

// DexMicrosoftConnectorStatus defines the observed state of
// DexMicrosoftConnector.
type DexMicrosoftConnectorStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dexmicrosoft
// +kubebuilder:printcolumn:name="Installation",type=string,JSONPath=`.spec.installationRef.name`
// +kubebuilder:printcolumn:name="Tenant",type=string,JSONPath=`.spec.tenant`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DexMicrosoftConnector is the Schema for the dexmicrosoftconnectors API.
type DexMicrosoftConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexMicrosoftConnectorSpec   `json:"spec,omitempty"`
	Status DexMicrosoftConnectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DexMicrosoftConnectorList contains a list of DexMicrosoftConnector.
type DexMicrosoftConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexMicrosoftConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DexMicrosoftConnector{}, &DexMicrosoftConnectorList{})
}
