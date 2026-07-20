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

// DexAuthProxyConnectorSpec defines configuration for the Dex authproxy
// connector. This connector delegates authentication to an HTTP reverse proxy
// that sets a header with the authenticated user's identity.
type DexAuthProxyConnectorSpec struct {
	// InstallationRef references the DexInstallation this connector belongs to.
	// +kubebuilder:validation:Required
	InstallationRef InstallationRef `json:"installationRef"`

	// ID is the connector ID used internally by Dex. Defaults to metadata.name.
	// +optional
	ID string `json:"id,omitempty"`

	// DisplayName is the human-readable connector name shown on the Dex login page.
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// UserIDHeader is the HTTP header carrying the user's unique ID.
	// Defaults to X-Remote-User-ID.
	// +optional
	UserIDHeader string `json:"userIDHeader,omitempty"`

	// UserHeader is the HTTP header carrying the username.
	// Defaults to X-Remote-User.
	// +optional
	UserHeader string `json:"userHeader,omitempty"`

	// UserNameHeader is the HTTP header carrying the user's display name.
	// Defaults to X-Remote-User-Display-Name.
	// +optional
	UserNameHeader string `json:"userNameHeader,omitempty"`

	// EmailHeader is the HTTP header carrying the user's email.
	// Defaults to X-Remote-User-Email.
	// +optional
	EmailHeader string `json:"emailHeader,omitempty"`

	// GroupHeader is the HTTP header carrying the user's groups.
	// Defaults to X-Remote-Group.
	// +optional
	GroupHeader string `json:"groupHeader,omitempty"`

	// GroupHeaderSeparator splits the group header value into multiple groups.
	// +optional
	GroupHeaderSeparator string `json:"groupHeaderSeparator,omitempty"`

	// StaticGroups are groups added to every user authenticated via this
	// connector.
	// +optional
	StaticGroups []string `json:"staticGroups,omitempty"`
}

// DexAuthProxyConnectorStatus defines the observed state of
// DexAuthProxyConnector.
type DexAuthProxyConnectorStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dexauthproxy
// +kubebuilder:printcolumn:name="Installation",type=string,JSONPath=`.spec.installationRef.name`
// +kubebuilder:printcolumn:name="Header",type=string,JSONPath=`.spec.header`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DexAuthProxyConnector is the Schema for the dexauthproxyconnectors API.
type DexAuthProxyConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexAuthProxyConnectorSpec   `json:"spec,omitempty"`
	Status DexAuthProxyConnectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DexAuthProxyConnectorList contains a list of DexAuthProxyConnector.
type DexAuthProxyConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexAuthProxyConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DexAuthProxyConnector{}, &DexAuthProxyConnectorList{})
}
