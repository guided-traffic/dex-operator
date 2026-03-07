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

// DexGoogleConnectorSpec defines configuration for the Dex Google connector.
type DexGoogleConnectorSpec struct {
	// InstallationRef references the DexInstallation this connector belongs to.
	// +kubebuilder:validation:Required
	InstallationRef InstallationRef `json:"installationRef"`

	// ID is the connector ID used internally by Dex. Defaults to metadata.name.
	// +optional
	ID string `json:"id,omitempty"`

	// DisplayName is the human-readable connector name shown on the Dex login page.
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// ClientIDRef references the Secret key holding the Google OAuth2 client ID.
	// +kubebuilder:validation:Required
	ClientIDRef SecretKeyRef `json:"clientIDRef"`

	// ClientSecretRef references the Secret key holding the Google OAuth2 client
	// secret.
	// +kubebuilder:validation:Required
	ClientSecretRef SecretKeyRef `json:"clientSecretRef"`

	// RedirectURI is the callback URL registered with the Google OAuth2 app.
	// +optional
	RedirectURI string `json:"redirectURI,omitempty"`

	// HostedDomains restricts authentication to users from the listed G Suite
	// hosted domains.
	// +optional
	HostedDomains []string `json:"hostedDomains,omitempty"`

	// Groups restricts login to users who are members of at least one of the
	// listed Google Groups.
	// +optional
	Groups []string `json:"groups,omitempty"`

	// ServiceAccountFileRef references the Secret key holding the Google service
	// account JSON used for group lookups via the Admin SDK.
	// +optional
	ServiceAccountFileRef *SecretKeyRef `json:"serviceAccountFileRef,omitempty"`

	// AdminEmail is the G Suite administrator email used for the service account
	// impersonation required by the Admin SDK.
	// +optional
	AdminEmail string `json:"adminEmail,omitempty"`

	// FetchTransitiveMembership fetches transitive group memberships when listing
	// a user's groups.
	// +optional
	FetchTransitiveMembership bool `json:"fetchTransitiveMembership,omitempty"`
}

// DexGoogleConnectorStatus defines the observed state of DexGoogleConnector.
type DexGoogleConnectorStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dexgoogle
// +kubebuilder:printcolumn:name="Installation",type=string,JSONPath=`.spec.installationRef.name`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DexGoogleConnector is the Schema for the dexgoogleconnectors API.
type DexGoogleConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexGoogleConnectorSpec   `json:"spec,omitempty"`
	Status DexGoogleConnectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DexGoogleConnectorList contains a list of DexGoogleConnector.
type DexGoogleConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexGoogleConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DexGoogleConnector{}, &DexGoogleConnectorList{})
}
