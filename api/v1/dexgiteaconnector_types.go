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

// DexGiteaConnectorSpec defines configuration for the Dex Gitea connector.
type DexGiteaConnectorSpec struct {
	// InstallationRef references the DexInstallation this connector belongs to.
	// +kubebuilder:validation:Required
	InstallationRef InstallationRef `json:"installationRef"`

	// ID is the connector ID used internally by Dex. Defaults to metadata.name.
	// +optional
	ID string `json:"id,omitempty"`

	// Name is the human-readable connector name shown on the Dex login page.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// BaseURL is the base URL of the Gitea instance.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	BaseURL string `json:"baseURL"`

	// ClientIDRef references the Secret key holding the Gitea OAuth2 app client
	// ID.
	// +kubebuilder:validation:Required
	ClientIDRef SecretKeyRef `json:"clientIDRef"`

	// ClientSecretRef references the Secret key holding the Gitea OAuth2 app
	// client secret.
	// +kubebuilder:validation:Required
	ClientSecretRef SecretKeyRef `json:"clientSecretRef"`

	// RedirectURI is the callback URL registered with the Gitea OAuth2 app.
	// +optional
	RedirectURI string `json:"redirectURI,omitempty"`

	// Orgs restricts login to users who are members of at least one of the
	// listed Gitea organizations.
	// +optional
	Orgs []string `json:"orgs,omitempty"`

	// UseLoginAsID uses the Gitea login username as the user ID.
	// +optional
	UseLoginAsID bool `json:"useLoginAsID,omitempty"`

	// InsecureSkipVerify skips TLS verification of the Gitea instance.
	// +optional
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`

	// RootCARef references the Secret key holding the CA certificate for the
	// Gitea HTTPS endpoint.
	// +optional
	RootCARef *SecretKeyRef `json:"rootCARef,omitempty"`
}

// DexGiteaConnectorStatus defines the observed state of DexGiteaConnector.
type DexGiteaConnectorStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dexgitea
// +kubebuilder:printcolumn:name="Installation",type=string,JSONPath=`.spec.installationRef.name`
// +kubebuilder:printcolumn:name="Base URL",type=string,JSONPath=`.spec.baseURL`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DexGiteaConnector is the Schema for the dexgiteaconnectors API.
type DexGiteaConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexGiteaConnectorSpec   `json:"spec,omitempty"`
	Status DexGiteaConnectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DexGiteaConnectorList contains a list of DexGiteaConnector.
type DexGiteaConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexGiteaConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DexGiteaConnector{}, &DexGiteaConnectorList{})
}
