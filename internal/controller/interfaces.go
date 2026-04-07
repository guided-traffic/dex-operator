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

package controller

import dexv1 "github.com/guided-traffic/dex-operator/api/v1"

// ChildObject is implemented by all child resources of a DexInstallation
// (connectors and static clients). It allows controllers to use shared
// reconciliation logic regardless of the concrete type.
//
// This interface is intentionally defined in the controller package (not api/v1)
// so that controller-gen does not attempt to process it during code generation.
type ChildObject interface {
	// GetInstallationRef returns the reference to the owning DexInstallation.
	GetInstallationRef() dexv1.InstallationRef
	// GetCommonStatus returns a pointer to the shared status block so
	// controllers can read / write conditions and observedGeneration.
	GetCommonStatus() *dexv1.CommonStatus
	// GetReferencedSecretNames returns the names of all Kubernetes Secrets
	// referenced by this resource (e.g. clientSecretRef, bindPWRef).
	// Used by the secret-watch indexer to trigger reconciliation when a
	// referenced Secret changes.
	GetReferencedSecretNames() []string
}
