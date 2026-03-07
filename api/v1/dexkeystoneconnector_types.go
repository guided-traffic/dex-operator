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

// DexKeystoneConnectorSpec defines configuration for the Dex Keystone
// (OpenStack Identity) connector.
type DexKeystoneConnectorSpec struct {
	// InstallationRef references the DexInstallation this connector belongs to.
	// +kubebuilder:validation:Required
	InstallationRef InstallationRef `json:"installationRef"`

	// ID is the connector ID used internally by Dex. Defaults to metadata.name.
	// +optional
	ID string `json:"id,omitempty"`

	// Name is the human-readable connector name shown on the Dex login page.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// KeystoneHost is the Keystone public endpoint
	// (e.g. https://keystone.example.com:5000).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	KeystoneHost string `json:"keystoneHost"`

	// KeystoneAdminURL is the Keystone admin endpoint used for group/role
	// lookups (e.g. https://keystone.example.com:35357).
	// +optional
	KeystoneAdminURL string `json:"keystoneAdminURL,omitempty"`

	// Domain restricts authentication to a specific Keystone domain name.
	// +optional
	Domain string `json:"domain,omitempty"`

	// AdminUsername is the Keystone admin username for directory lookups.
	// +optional
	AdminUsername string `json:"adminUsername,omitempty"`

	// AdminPasswordRef references the Secret key holding the Keystone admin
	// password.
	// +optional
	AdminPasswordRef *SecretKeyRef `json:"adminPasswordRef,omitempty"`

	// Groups restricts login to users who have at least one of the listed roles
	// in Keystone.
	// +optional
	Groups []string `json:"groups,omitempty"`
}

// DexKeystoneConnectorStatus defines the observed state of
// DexKeystoneConnector.
type DexKeystoneConnectorStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dexkeystone
// +kubebuilder:printcolumn:name="Installation",type=string,JSONPath=`.spec.installationRef.name`
// +kubebuilder:printcolumn:name="Keystone Host",type=string,JSONPath=`.spec.keystoneHost`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DexKeystoneConnector is the Schema for the dexkeystoneconnectors API.
type DexKeystoneConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexKeystoneConnectorSpec   `json:"spec,omitempty"`
	Status DexKeystoneConnectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DexKeystoneConnectorList contains a list of DexKeystoneConnector.
type DexKeystoneConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexKeystoneConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DexKeystoneConnector{}, &DexKeystoneConnectorList{})
}
