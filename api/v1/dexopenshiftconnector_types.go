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

// DexOpenShiftConnectorSpec defines configuration for the Dex OpenShift
// connector.
type DexOpenShiftConnectorSpec struct {
	// InstallationRef references the DexInstallation this connector belongs to.
	// +kubebuilder:validation:Required
	InstallationRef InstallationRef `json:"installationRef"`

	// ID is the connector ID used internally by Dex. Defaults to metadata.name.
	// +optional
	ID string `json:"id,omitempty"`

	// Name is the human-readable connector name shown on the Dex login page.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Issuer is the base URL of the OpenShift API server.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uri
	Issuer string `json:"issuer"`

	// ClientIDRef references the Secret key holding the OpenShift OAuth client
	// ID.
	// +kubebuilder:validation:Required
	ClientIDRef SecretKeyRef `json:"clientIDRef"`

	// ClientSecretRef references the Secret key holding the OpenShift OAuth
	// client secret.
	// +kubebuilder:validation:Required
	ClientSecretRef SecretKeyRef `json:"clientSecretRef"`

	// RedirectURI is the callback URL registered with the OpenShift OAuth
	// client.
	// +optional
	RedirectURI string `json:"redirectURI,omitempty"`

	// Groups restricts login to users who are members of at least one of the
	// listed OpenShift groups.
	// +optional
	Groups []string `json:"groups,omitempty"`

	// InsecureCA skips CA verification for the OpenShift API server.
	// WARNING: only for development/testing.
	// +optional
	InsecureCA bool `json:"insecureCA,omitempty"`

	// RootCARef references the Secret key holding the CA certificate for the
	// OpenShift API server.
	// +optional
	RootCARef *SecretKeyRef `json:"rootCARef,omitempty"`
}

// DexOpenShiftConnectorStatus defines the observed state of
// DexOpenShiftConnector.
type DexOpenShiftConnectorStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dexopenshift
// +kubebuilder:printcolumn:name="Installation",type=string,JSONPath=`.spec.installationRef.name`
// +kubebuilder:printcolumn:name="Issuer",type=string,JSONPath=`.spec.issuer`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DexOpenShiftConnector is the Schema for the dexopenshiftconnectors API.
type DexOpenShiftConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexOpenShiftConnectorSpec   `json:"spec,omitempty"`
	Status DexOpenShiftConnectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DexOpenShiftConnectorList contains a list of DexOpenShiftConnector.
type DexOpenShiftConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexOpenShiftConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DexOpenShiftConnector{}, &DexOpenShiftConnectorList{})
}
