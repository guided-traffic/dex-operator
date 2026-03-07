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

// DexGitLabConnectorSpec defines configuration for the Dex GitLab connector.
type DexGitLabConnectorSpec struct {
	// InstallationRef references the DexInstallation this connector belongs to.
	// +kubebuilder:validation:Required
	InstallationRef InstallationRef `json:"installationRef"`

	// ID is the connector ID used internally by Dex. Defaults to metadata.name.
	// +optional
	ID string `json:"id,omitempty"`

	// DisplayName is the human-readable connector name shown on the Dex login page.
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// BaseURL is the base URL of the GitLab instance.
	// Defaults to https://gitlab.com.
	// +optional
	BaseURL string `json:"baseURL,omitempty"`

	// ClientIDRef references the Secret key holding the GitLab application
	// client ID.
	// +kubebuilder:validation:Required
	ClientIDRef SecretKeyRef `json:"clientIDRef"`

	// ClientSecretRef references the Secret key holding the GitLab application
	// client secret.
	// +kubebuilder:validation:Required
	ClientSecretRef SecretKeyRef `json:"clientSecretRef"`

	// RedirectURI is the callback URL registered with the GitLab application.
	// +optional
	RedirectURI string `json:"redirectURI,omitempty"`

	// Groups restricts login to users that belong to at least one of the
	// listed GitLab groups.
	// +optional
	Groups []string `json:"groups,omitempty"`

	// UseLoginAsID uses the GitLab login username as the user ID.
	// +optional
	UseLoginAsID bool `json:"useLoginAsID,omitempty"`
}

// DexGitLabConnectorStatus defines the observed state of DexGitLabConnector.
type DexGitLabConnectorStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dexgitlab
// +kubebuilder:printcolumn:name="Installation",type=string,JSONPath=`.spec.installationRef.name`
// +kubebuilder:printcolumn:name="Base URL",type=string,JSONPath=`.spec.baseURL`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DexGitLabConnector is the Schema for the dexgitlabconnectors API.
type DexGitLabConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexGitLabConnectorSpec   `json:"spec,omitempty"`
	Status DexGitLabConnectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DexGitLabConnectorList contains a list of DexGitLabConnector.
type DexGitLabConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexGitLabConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DexGitLabConnector{}, &DexGitLabConnectorList{})
}
