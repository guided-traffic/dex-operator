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

// DexLDAPConnectorSpec defines the configuration for the Dex LDAP connector.
type DexLDAPConnectorSpec struct {
	// InstallationRef references the DexInstallation this connector belongs to.
	// +kubebuilder:validation:Required
	InstallationRef InstallationRef `json:"installationRef"`

	// ID is the connector ID used internally by Dex. Defaults to metadata.name.
	// +optional
	ID string `json:"id,omitempty"`

	// Name is the human-readable connector name shown on the Dex login page.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Host is the LDAP host in the format host:port.
	// +kubebuilder:validation:Required
	Host string `json:"host"`

	// InsecureNoSSL disables all TLS (plain LDAP on port 389). Not recommended.
	// +optional
	InsecureNoSSL bool `json:"insecureNoSSL,omitempty"`

	// InsecureSkipVerify skips TLS verification of the server certificate.
	// +optional
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`

	// StartTLS upgrades the connection using the STARTTLS command.
	// +optional
	StartTLS bool `json:"startTLS,omitempty"`

	// RootCARef references the Secret key holding the PEM-encoded root CA.
	// +optional
	RootCARef *SecretKeyRef `json:"rootCARef,omitempty"`

	// ClientCertRef references the Secret key holding the client TLS certificate.
	// +optional
	ClientCertRef *SecretKeyRef `json:"clientCertRef,omitempty"`

	// ClientKeyRef references the Secret key holding the client TLS key.
	// +optional
	ClientKeyRef *SecretKeyRef `json:"clientKeyRef,omitempty"`

	// BindDN is the DN used to bind to the LDAP directory.
	// +optional
	BindDN string `json:"bindDN,omitempty"`

	// BindPWRef references the Secret key holding the bind password.
	// +optional
	BindPWRef *SecretKeyRef `json:"bindPWRef,omitempty"`

	// UsernamePrompt overrides the username input label on the login form.
	// +optional
	UsernamePrompt string `json:"usernamePrompt,omitempty"`

	// UserSearch configures user search parameters.
	// +kubebuilder:validation:Required
	UserSearch LDAPUserSearch `json:"userSearch"`

	// GroupSearch configures optional group membership lookup.
	// +optional
	GroupSearch *LDAPGroupSearch `json:"groupSearch,omitempty"`
}

// LDAPUserSearch defines how users are searched in the LDAP directory.
type LDAPUserSearch struct {
	// BaseDN is the base distinguished name to search from.
	// +kubebuilder:validation:Required
	BaseDN string `json:"baseDN"`

	// Filter is an optional LDAP filter applied to user entries.
	// +optional
	Filter string `json:"filter,omitempty"`

	// Username is the attribute mapping to the user-provided username field.
	// +kubebuilder:validation:Required
	Username string `json:"username"`

	// Scope is the LDAP search scope (sub or one). Defaults to sub.
	// +kubebuilder:validation:Enum=sub;one
	// +optional
	Scope string `json:"scope,omitempty"`

	// IDAttr is the LDAP attribute mapped to the user ID claim.
	// +kubebuilder:default=DN
	// +optional
	IDAttr string `json:"idAttr,omitempty"`

	// EmailAttr is the LDAP attribute mapped to the email claim.
	// +kubebuilder:default=mail
	// +optional
	EmailAttr string `json:"emailAttr,omitempty"`

	// NameAttr is the LDAP attribute mapped to the name claim.
	// +optional
	NameAttr string `json:"nameAttr,omitempty"`

	// PreferredUsernameAttr is the LDAP attribute mapped to the
	// preferred_username claim.
	// +optional
	PreferredUsernameAttr string `json:"preferredUsernameAttr,omitempty"`

	// EmailSuffix appends a suffix to users that do not have an email attribute.
	// +optional
	EmailSuffix string `json:"emailSuffix,omitempty"`
}

// LDAPGroupSearch defines how groups are searched in the LDAP directory.
type LDAPGroupSearch struct {
	// BaseDN is the base distinguished name for group searches.
	// +kubebuilder:validation:Required
	BaseDN string `json:"baseDN"`

	// Filter is an optional LDAP filter applied to group entries.
	// +optional
	Filter string `json:"filter,omitempty"`

	// Scope is the LDAP search scope (sub or one). Defaults to sub.
	// +kubebuilder:validation:Enum=sub;one
	// +optional
	Scope string `json:"scope,omitempty"`

	// UserAttr is the attribute on the user entry that is matched against
	// GroupAttr.
	// +kubebuilder:default=DN
	// +optional
	UserAttr string `json:"userAttr,omitempty"`

	// GroupAttr is the attribute on the group entry that is matched against
	// UserAttr.
	// +kubebuilder:default=member
	// +optional
	GroupAttr string `json:"groupAttr,omitempty"`

	// NameAttr is the LDAP attribute for the group name claim.
	// +kubebuilder:default=cn
	// +optional
	NameAttr string `json:"nameAttr,omitempty"`
}

// DexLDAPConnectorStatus defines the observed state of DexLDAPConnector.
type DexLDAPConnectorStatus struct {
	CommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=dexldap
// +kubebuilder:printcolumn:name="Installation",type=string,JSONPath=`.spec.installationRef.name`
// +kubebuilder:printcolumn:name="Host",type=string,JSONPath=`.spec.host`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DexLDAPConnector is the Schema for the dexldapconnectors API.
type DexLDAPConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DexLDAPConnectorSpec   `json:"spec,omitempty"`
	Status DexLDAPConnectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DexLDAPConnectorList contains a list of DexLDAPConnector.
type DexLDAPConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DexLDAPConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DexLDAPConnector{}, &DexLDAPConnectorList{})
}
