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
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IsNamespaceAllowed exposes [isNamespaceAllowed] for white-box testing.
var IsNamespaceAllowed = isNamespaceAllowed

// RolloutEnabled exposes [rolloutEnabled] for white-box testing.
var RolloutEnabled = rolloutEnabled

// SecretDataEqual exposes [secretDataEqual] for white-box testing.
var SecretDataEqual = secretDataEqual

// YamlSecretDataEqual exposes [yamlSecretDataEqual] for white-box testing.
var YamlSecretDataEqual = yamlSecretDataEqual

// CountConnectors exposes [countConnectors] for white-box testing.
var CountConnectors = countConnectors

// IsConfigError exposes [isConfigError] for white-box testing.
var IsConfigError = isConfigError

// NewConfigError exposes [newConfigError] for white-box testing.
var NewConfigError = newConfigError

// LookupChildSecretRefs exposes [lookupChildSecretRefs] for white-box testing.
var LookupChildSecretRefs = lookupChildSecretRefs

// MapSecretToInstallation exposes [DexInstallationReconciler.mapSecretToInstallation]
// for white-box testing. Usage: r.MapSecretToInstallation(ctx, obj).
func (r *DexInstallationReconciler) MapSecretToInstallation(ctx context.Context, obj client.Object) []ctrl.Request {
	return r.mapSecretToInstallation(ctx, obj)
}
