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

// InstallationRef references a DexInstallation by name and namespace.
type InstallationRef struct {
	// Name of the DexInstallation resource.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace where the DexInstallation resides.
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`
}

// SecretKeyRef selects a key from a Kubernetes Secret.
type SecretKeyRef struct {
	// Name of the Secret resource.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Key within the Secret.
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

// Condition types for all Dex operator resources.
const (
	// ConditionTypeReady indicates the resource has been reconciled successfully.
	ConditionTypeReady = "Ready"
	// ConditionTypeError indicates the resource is in an error state.
	ConditionTypeError = "Error"
)

// CommonStatus contains the shared status fields for all Dex CRDs.
type CommonStatus struct {
	// Conditions represent the latest available observations of the object's state.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the generation last processed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}
