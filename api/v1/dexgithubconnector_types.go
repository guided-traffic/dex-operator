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

// DexGitHubConnectorSpec defines configuration for the Dex GitHub connector.
type DexGitHubConnectorSpec struct {
	// InstallationRef references the DexInstallation this connector belongs to.
	// +kubebuilder:validation:Required
	InstallationRef InstallationRef `json:"installationRef"`

	// ID is the connector ID used internally by Dex. Defaults to metadata.name.
	// +optional
	ID string `json:"id,omitempty"`

	// Name is the human-readable connector name shown on the Dex login page.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// ClientIDRef references the Secret key holding the GitHub OAuth app client ID.
	// +kubebuilder:validation:Required
	ClientIDRef SecretKeyRef `json:"clientIDRef"`

	// ClientSecretRef references the Secret key holding the GitHub OAuth app
	// client secret.
	// +kubebuilder:validation:Required
	ClientSecretRef SecretKeyRef `json:"clientSecretRef"`

	// RedirectURI is the callback URL registered with the GitHub OAuth app.
	// +optional
	RedirectURI string `json:"redirectURI,omitempty"`

	// HostName allows connecting to a GitHub Enterprise instance.
	// Defaults to github.com.
	// +optional
	HostName string `json:"hostName,omitempty"`

	// RootCARef references the Secret key holding the CA certificate for GitHub
	// Enterprise TLS.
	// +optional
	RootCARef *SecretKeyRef `json:"rootCARef,omitempty"`

	// Orgs filters login to users that belong to one of the listed GitHub
	// organizations (and optionally specific teams).
	// +optional
	Orgs []GitHubOrg `json:"orgs,omitempty"`

	// LoadAllGroups loads all GitHub organizations (and their teams) for the
	// authenticated user, not just the Orgs listed above.
	// +optional
	LoadAllGroups bool `json:"loadAllGroups,omitempty"`

	// TeamNameField controls the format of group names.
	// Allowed values: name, slug, both.
	// +kubebuilder:validation:Enum=name;slug;both
	// +optional
	TeamNameField string `json:"teamNameField,omitempty"`

	// UseLoginAsID uses the GitHub login username as the user ID instead of the
	// numeric user ID.
	// +optional
	UseLoginAsID bool `json:"useLoginAsID,omitempty"`
}

// GitHubOrg restricts authentication to a GitHub organization.
type GitHubOrg struct {
	// Name is the GitHub organization name.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Teams filters to specific team slugs within the organization.
	// +optional
	Teams []string `json:"teams,omitempty"`
}

// DexGitHubConnectorStatus defines the observed state of DexGitHubConnector.
type DexGitHubConnectorStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dexgithub
// +kubebuilder:printcolumn:name="Installation",type=string,JSONPath=`.spec.installationRef.name`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DexGitHubConnector is the Schema for the dexgithubconnectors API.
type DexGitHubConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexGitHubConnectorSpec   `json:"spec,omitempty"`
	Status DexGitHubConnectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DexGitHubConnectorList contains a list of DexGitHubConnector.
type DexGitHubConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexGitHubConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DexGitHubConnector{}, &DexGitHubConnectorList{})
}
