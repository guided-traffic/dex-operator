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
