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

// DexBitbucketConnectorSpec defines configuration for the Dex Bitbucket
// Cloud connector.
type DexBitbucketConnectorSpec struct {
	// InstallationRef references the DexInstallation this connector belongs to.
	// +kubebuilder:validation:Required
	InstallationRef InstallationRef `json:"installationRef"`

	// ID is the connector ID used internally by Dex. Defaults to metadata.name.
	// +optional
	ID string `json:"id,omitempty"`

	// Name is the human-readable connector name shown on the Dex login page.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// ClientIDRef references the Secret key holding the Bitbucket OAuth2
	// consumer key.
	// +kubebuilder:validation:Required
	ClientIDRef SecretKeyRef `json:"clientIDRef"`

	// ClientSecretRef references the Secret key holding the Bitbucket OAuth2
	// consumer secret.
	// +kubebuilder:validation:Required
	ClientSecretRef SecretKeyRef `json:"clientSecretRef"`

	// RedirectURI is the callback URL registered with the Bitbucket OAuth2
	// consumer.
	// +optional
	RedirectURI string `json:"redirectURI,omitempty"`

	// Teams restricts login to users who are members of at least one of the
	// listed Bitbucket teams (workspaces).
	// +optional
	Teams []string `json:"teams,omitempty"`
}

// DexBitbucketConnectorStatus defines the observed state of
// DexBitbucketConnector.
type DexBitbucketConnectorStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dexbitbucket
// +kubebuilder:printcolumn:name="Installation",type=string,JSONPath=`.spec.installationRef.name`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DexBitbucketConnector is the Schema for the dexbitbucketconnectors API.
type DexBitbucketConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexBitbucketConnectorSpec   `json:"spec,omitempty"`
	Status DexBitbucketConnectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DexBitbucketConnectorList contains a list of DexBitbucketConnector.
type DexBitbucketConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexBitbucketConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DexBitbucketConnector{}, &DexBitbucketConnectorList{})
}
