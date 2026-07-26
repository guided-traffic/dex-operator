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

// DexStaticClientSpec defines the desired state of a Dex static OAuth2 client.
//
// A client is either confidential (client-id and client-secret sourced from a
// Kubernetes Secret via secretRef) or public/secretless (client-id set inline
// via clientID, no secret at all).  Public clients may still use secretRef if
// they need a secret in addition to PKCE.
//
// +kubebuilder:validation:XValidation:rule="has(self.clientID) != has(self.secretRef)",message="exactly one of clientID or secretRef must be set"
// +kubebuilder:validation:XValidation:rule="(has(self.public) && self.public) || has(self.secretRef)",message="secretRef is required for confidential (non-public) clients"
// +kubebuilder:validation:XValidation:rule="(has(self.public) && self.public) || (has(self.redirectURIs) && self.redirectURIs.size() > 0)",message="redirectURIs is required for confidential (non-public) clients"
type DexStaticClientSpec struct {
	// InstallationRef references the DexInstallation this client belongs to.
	// +kubebuilder:validation:Required
	InstallationRef InstallationRef `json:"installationRef"`

	// ClientID is the OAuth2 client ID, set inline.  Mutually exclusive with
	// secretRef and intended for public (secretless) clients, whose ID is not
	// confidential.
	// +optional
	ClientID string `json:"clientID,omitempty"`

	// SecretRef references the existing Secret that contains the client-id
	// and client-secret for this client.  Required for confidential clients,
	// mutually exclusive with clientID.
	// +optional
	SecretRef *StaticClientSecretRef `json:"secretRef,omitempty"`

	// DisplayName is the human-readable display name of the client shown on the
	// Dex approval screen.
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// RedirectURIs is the list of allowed redirect URIs.  Required for
	// confidential clients.  A public client may leave it empty, in which case
	// Dex accepts its loopback, OOB and device-flow defaults.
	// +optional
	RedirectURIs []string `json:"redirectURIs,omitempty"`

	// TrustedPeers lists other client IDs whose ID tokens this client trusts.
	// +optional
	TrustedPeers []string `json:"trustedPeers,omitempty"`

	// Public marks the client as a public application (no client secret required).
	// +optional
	Public bool `json:"public,omitempty"`
}

// StaticClientSecretRef references the keys inside a Kubernetes Secret that
// hold the client-id and client-secret.
type StaticClientSecretRef struct {
	// Name of the Secret resource in the same namespace as this DexStaticClient.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// ClientIDKey is the key within the Secret that holds the client-id.
	// Defaults to "client-id".
	// +kubebuilder:default=client-id
	// +optional
	ClientIDKey string `json:"clientIDKey,omitempty"`

	// ClientSecretKey is the key within the Secret that holds the client-secret.
	// Defaults to "client-secret".
	// +kubebuilder:default=client-secret
	// +optional
	ClientSecretKey string `json:"clientSecretKey,omitempty"`
}

// DexStaticClientStatus defines the observed state of DexStaticClient.
type DexStaticClientStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dexsc
// +kubebuilder:printcolumn:name="Installation",type=string,JSONPath=`.spec.installationRef.name`
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DexStaticClient is the Schema for the dexstaticclients API.
// It represents a static OAuth2 client registered with Dex.
type DexStaticClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexStaticClientSpec   `json:"spec,omitempty"`
	Status DexStaticClientStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DexStaticClientList contains a list of DexStaticClient.
type DexStaticClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexStaticClient `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DexStaticClient{}, &DexStaticClientList{})
}
