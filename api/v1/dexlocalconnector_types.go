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

// DexLocalConnectorSpec defines configuration for the Dex local (builtin)
// password connector. User credentials are stored in a Kubernetes Secret.
type DexLocalConnectorSpec struct {
	// InstallationRef references the DexInstallation this connector belongs to.
	// +kubebuilder:validation:Required
	InstallationRef InstallationRef `json:"installationRef"`

	// ID is the connector ID used internally by Dex. Defaults to metadata.name.
	// +optional
	ID string `json:"id,omitempty"`

	// DisplayName is the human-readable connector name shown on the Dex login page.
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`
}

// DexLocalConnectorStatus defines the observed state of DexLocalConnector.
type DexLocalConnectorStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dexlocal
// +kubebuilder:printcolumn:name="Installation",type=string,JSONPath=`.spec.installationRef.name`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DexLocalConnector is the Schema for the dexlocalconnectors API.
// It enables the Dex built-in local password database connector.
type DexLocalConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexLocalConnectorSpec   `json:"spec,omitempty"`
	Status DexLocalConnectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DexLocalConnectorList contains a list of DexLocalConnector.
type DexLocalConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexLocalConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DexLocalConnector{}, &DexLocalConnectorList{})
}
