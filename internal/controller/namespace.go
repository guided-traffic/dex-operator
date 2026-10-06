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

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// isNamespaceAllowed reports whether the given namespace is permitted to
// reference a DexInstallation.  allowedNamespaces follows the same semantics
// as DexInstallationSpec.AllowedNamespaces:
//   - an empty list denies all namespaces
//   - the wildcard entry "*" allows every namespace
//   - any other entry is a literal namespace name
func isNamespaceAllowed(namespace string, allowedNamespaces []string) bool {
	for _, allowed := range allowedNamespaces {
		if allowed == "*" || allowed == namespace {
			return true
		}
	}
	return false
}

// connectorNamespaces returns the effective connector allowlist of an
// installation: spec.allowedConnectorNamespaces when set, otherwise only the
// installation's own namespace.
//
// An empty list is treated like an omitted one. The CRD rejects [] at
// admission (MinItems=1), and omitempty drops it on any Go round trip, so
// "empty" can only mean "not set" by the time it reaches the operator.
func connectorNamespaces(installation *dexv1.DexInstallation) []string {
	if len(installation.Spec.AllowedConnectorNamespaces) == 0 {
		return []string{installation.Namespace}
	}
	return installation.Spec.AllowedConnectorNamespaces
}

// checkChildNamespace returns a configError when the namespace of child is
// not admitted by installation. DexStaticClients are governed by
// spec.allowedNamespaces, every connector kind by the effective connector
// allowlist (see [connectorNamespaces]).
func checkChildNamespace(child client.Object, installation *dexv1.DexInstallation) error {
	ns := child.GetNamespace()

	if _, isClient := child.(*dexv1.DexStaticClient); isClient {
		if isNamespaceAllowed(ns, installation.Spec.AllowedNamespaces) {
			return nil
		}
		return newConfigError(fmt.Sprintf(
			"namespace %q is not in DexInstallation %s/%s allowedNamespaces",
			ns, installation.Namespace, installation.Name))
	}

	if isNamespaceAllowed(ns, connectorNamespaces(installation)) {
		return nil
	}
	msg := fmt.Sprintf(
		"namespace %q is not in DexInstallation %s/%s allowedConnectorNamespaces",
		ns, installation.Namespace, installation.Name)
	if len(installation.Spec.AllowedConnectorNamespaces) == 0 {
		msg += fmt.Sprintf(" (omitted: only %q is allowed)", installation.Namespace)
	}
	return newConfigError(msg)
}
