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

// DexAtlassianCrowdConnectorSpec defines configuration for the Dex Atlassian
// Crowd connector.
type DexAtlassianCrowdConnectorSpec struct {
	// InstallationRef references the DexInstallation this connector belongs to.
	// +kubebuilder:validation:Required
	InstallationRef InstallationRef `json:"installationRef"`

	// ID is the connector ID used internally by Dex. Defaults to metadata.name.
	// +optional
	ID string `json:"id,omitempty"`

	// Name is the human-readable connector name shown on the Dex login page.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// BaseURL is the base URL of the Crowd server
	// (e.g. https://crowd.example.com/crowd).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	BaseURL string `json:"baseURL"`

	// ClientIDRef references the Secret key holding the Crowd application name.
	// +kubebuilder:validation:Required
	ClientIDRef SecretKeyRef `json:"clientIDRef"`

	// ClientSecretRef references the Secret key holding the Crowd application
	// password.
	// +kubebuilder:validation:Required
	ClientSecretRef SecretKeyRef `json:"clientSecretRef"`

	// RedirectURI is the callback URL for the Crowd connector.
	// +optional
	RedirectURI string `json:"redirectURI,omitempty"`

	// Groups restricts login to users who are members of at least one of the
	// listed Crowd groups.
	// +optional
	Groups []string `json:"groups,omitempty"`

	// AdminUser is the Crowd admin username for directory lookups.
	// +optional
	AdminUser string `json:"adminUser,omitempty"`

	// AdminPasswordRef references the Secret key holding the Crowd admin
	// password.
	// +optional
	AdminPasswordRef *SecretKeyRef `json:"adminPasswordRef,omitempty"`
}

// DexAtlassianCrowdConnectorStatus defines the observed state of
// DexAtlassianCrowdConnector.
type DexAtlassianCrowdConnectorStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dexcrowd
// +kubebuilder:printcolumn:name="Installation",type=string,JSONPath=`.spec.installationRef.name`
// +kubebuilder:printcolumn:name="Base URL",type=string,JSONPath=`.spec.baseURL`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DexAtlassianCrowdConnector is the Schema for the
// dexatlassiancrowdconnectors API.
type DexAtlassianCrowdConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexAtlassianCrowdConnectorSpec   `json:"spec,omitempty"`
	Status DexAtlassianCrowdConnectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DexAtlassianCrowdConnectorList contains a list of DexAtlassianCrowdConnector.
type DexAtlassianCrowdConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexAtlassianCrowdConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DexAtlassianCrowdConnector{}, &DexAtlassianCrowdConnectorList{})
}
