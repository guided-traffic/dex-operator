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
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// GenericChildReconciler is a generic controller for all child resources
// (connectors and static clients).  It validates namespace access against
// the referenced DexInstallation and updates the child's status.
//
// Type parameters:
//   - T: pointer to the concrete resource type (e.g. *DexOIDCConnector)
//   - U: the underlying struct type (e.g. DexOIDCConnector)
type GenericChildReconciler[T interface {
	*U
	client.Object
	ChildObject
}, U any] struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile fetches the resource, validates namespace access, and updates status.
func (r *GenericChildReconciler[T, U]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var obj U
	ptr := T(&obj)
	if err := r.Get(ctx, req.NamespacedName, ptr); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	reconcileErr := r.reconcileChild(ctx, ptr)

	setReadyCondition(ptr.GetCommonStatus(), ptr.GetGeneration(), reconcileErr)
	ptr.GetCommonStatus().ObservedGeneration = ptr.GetGeneration()

	if statusErr := r.Status().Update(ctx, ptr); statusErr != nil {
		if !errors.IsConflict(statusErr) {
			logger.Error(statusErr, "failed to update child resource status")
		}
	}

	return ctrl.Result{}, reconcileErr
}

// SetupWithManager registers this reconciler as a controller for type T.
func (r *GenericChildReconciler[T, U]) SetupWithManager(mgr ctrl.Manager) error {
	var zero U
	return ctrl.NewControllerManagedBy(mgr).
		For(T(&zero)).
		Complete(r)
}

// reconcileChild validates that the resource's namespace is allowed by its
// referenced DexInstallation.
func (r *GenericChildReconciler[T, U]) reconcileChild(ctx context.Context, obj T) error {
	ref := obj.GetInstallationRef()

	var installation dexv1.DexInstallation
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: ref.Namespace,
		Name:      ref.Name,
	}, &installation); err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("referenced DexInstallation %s/%s not found", ref.Namespace, ref.Name)
		}
		return fmt.Errorf("fetching DexInstallation %s/%s: %w", ref.Namespace, ref.Name, err)
	}

	sourceNS := obj.GetNamespace()
	if !isNamespaceAllowed(sourceNS, installation.Spec.AllowedNamespaces) {
		return fmt.Errorf("namespace %q is not in DexInstallation %s/%s allowedNamespaces",
			sourceNS, ref.Namespace, ref.Name)
	}

	return nil
}
