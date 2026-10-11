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
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// configRequeueInterval is the delay before a child resource with a
// configuration error (e.g. missing DexInstallation) is re-checked.
const configRequeueInterval = 5 * time.Minute

// configError signals a user-caused configuration problem that will not
// resolve by simple retrying. The controller logs these at WARN level
// and requeues after a long interval instead of using the default
// exponential back-off. A non-empty reason replaces the default reason of
// the Ready condition.
type configError struct {
	reason string
	msg    string
}

func (e *configError) Error() string { return e.msg }

// newConfigError creates a configError with the given message.
func newConfigError(msg string) *configError {
	return &configError{msg: msg}
}

// newRejectionError creates a configError for a child the installation left
// out of its rendered config, carrying the installation's reason.
func newRejectionError(rc dexv1.RejectedChild) *configError {
	return &configError{reason: rc.Reason, msg: rc.Message}
}

// isConfigError returns true when err (or any wrapped cause) is a configError.
func isConfigError(err error) bool {
	var ce *configError
	return errors.As(err, &ce)
}

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

	installation, reconcileErr := r.reconcileChild(ctx, ptr)

	setReadyCondition(ptr.GetCommonStatus(), ptr.GetGeneration(), reconcileErr)
	ptr.GetCommonStatus().ObservedGeneration = ptr.GetGeneration()
	if sc, isClient := any(ptr).(*dexv1.DexStaticClient); isClient {
		setTrustedPeersDroppedCondition(sc, installation)
	}

	if statusErr := r.Status().Update(ctx, ptr); statusErr != nil {
		if !apierrors.IsConflict(statusErr) {
			logger.Error(statusErr, "failed to update child resource status")
		}
	}

	// Configuration errors are caused by the user (wrong installationRef,
	// forbidden namespace, …). Log a clear warning and requeue after a long
	// interval instead of returning an error, which would trigger
	// controller-runtime's exponential back-off with noisy stack traces.
	if isConfigError(reconcileErr) {
		logger.Info("configuration error on child resource, will retry",
			"reason", reconcileErr.Error(),
			"requeueAfter", configRequeueInterval,
		)
		return ctrl.Result{RequeueAfter: configRequeueInterval}, nil
	}

	return ctrl.Result{}, reconcileErr
}

// SetupWithManager registers this reconciler as a controller for type T.
//
// Besides its own kind it watches DexInstallation, so that a change to the
// installation's allowlists, or to the children it left out of its rendered
// config, re-evaluates the conditions of every child referencing it. Only
// generation changes (spec edits), changes of status.rejectedChildren or
// status.droppedTrustedPeers, creates and deletes pass; the installation's
// other status updates would otherwise fan out to all its children. The
// DexInstallation controller must be set up first: it registers the
// InstallationRefIndexField index used by the mapping.
func (r *GenericChildReconciler[T, U]) SetupWithManager(mgr ctrl.Manager) error {
	var zero U
	listGVK, err := listGVKFor(T(&zero), r.Scheme)
	if err != nil {
		return fmt.Errorf("resolving list kind for %T: %w", T(&zero), err)
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(T(&zero)).
		Watches(
			&dexv1.DexInstallation{},
			handler.EnqueueRequestsFromMapFunc(r.mapInstallationToChildren(listGVK)),
			builder.WithPredicates(predicate.Or(predicate.GenerationChangedPredicate{}, childReportChangedPredicate())),
		).
		Complete(r)
}

// childReportChangedPredicate passes an installation update that changes
// status.rejectedChildren or status.droppedTrustedPeers, the two status
// fields a child's conditions are derived from.
func childReportChangedPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldInst, okOld := e.ObjectOld.(*dexv1.DexInstallation)
			newInst, okNew := e.ObjectNew.(*dexv1.DexInstallation)
			if !okOld || !okNew {
				return false
			}
			return !equality.Semantic.DeepEqual(oldInst.Status.RejectedChildren, newInst.Status.RejectedChildren) ||
				!equality.Semantic.DeepEqual(oldInst.Status.DroppedTrustedPeers, newInst.Status.DroppedTrustedPeers)
		},
	}
}

// reconcileChild validates that the resource's namespace is allowed by its
// referenced DexInstallation (see [checkChildNamespace] for which allowlist
// applies to which kind), and that the installation did not leave it out of
// its rendered config (status.rejectedChildren). It returns the installation
// when it could be read.
func (r *GenericChildReconciler[T, U]) reconcileChild(ctx context.Context, obj T) (*dexv1.DexInstallation, error) {
	ref := obj.GetInstallationRef()

	var installation dexv1.DexInstallation
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: ref.Namespace,
		Name:      ref.Name,
	}, &installation); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, newConfigError(fmt.Sprintf(
				"referenced DexInstallation %s/%s not found", ref.Namespace, ref.Name))
		}
		return nil, fmt.Errorf("fetching DexInstallation %s/%s: %w", ref.Namespace, ref.Name, err)
	}

	if err := checkChildNamespace(obj, &installation); err != nil {
		return &installation, err
	}

	gvk, err := apiutil.GVKForObject(obj, r.Scheme)
	if err != nil {
		return &installation, fmt.Errorf("resolving kind of %T: %w", obj, err)
	}
	if rc := findRejectedChild(&installation, gvk.Kind, obj.GetNamespace(), obj.GetName()); rc != nil {
		return &installation, newRejectionError(*rc)
	}
	return &installation, nil
}

// findRejectedChild returns the installation's report on the child
// kind/namespace/name, or nil when the child is not rejected.
func findRejectedChild(installation *dexv1.DexInstallation, kind, namespace, name string) *dexv1.RejectedChild {
	for i, rc := range installation.Status.RejectedChildren {
		if rc.Kind == kind && rc.Namespace == namespace && rc.Name == name {
			return &installation.Status.RejectedChildren[i]
		}
	}
	return nil
}

// setTrustedPeersDroppedCondition sets the TrustedPeersDropped condition of
// a static client from the installation's status.droppedTrustedPeers: True
// with the client's own dropped peer IDs, False when it has none. The
// message never names the namespace of a peer's holder. installation may be
// nil when it could not be read.
func setTrustedPeersDroppedCondition(sc *dexv1.DexStaticClient, installation *dexv1.DexInstallation) {
	var peers []string
	if installation != nil {
		for _, d := range installation.Status.DroppedTrustedPeers {
			if d.Namespace == sc.Namespace && d.Name == sc.Name {
				peers = append(peers, strconv.Quote(d.PeerID))
			}
		}
	}

	cond := metav1.Condition{
		Type:               dexv1.ConditionTypeTrustedPeersDropped,
		ObservedGeneration: sc.Generation,
		Status:             metav1.ConditionFalse,
		Reason:             "NoneDropped",
	}
	if len(peers) > 0 {
		cond.Status = metav1.ConditionTrue
		cond.Reason = dexv1.ConditionTypeTrustedPeersDropped
		cond.Message = fmt.Sprintf(
			"trustedPeers %s are left out of the rendered config: no DexStaticClient in namespace %q holds these IDs",
			strings.Join(peers, ", "), sc.Namespace)
	}
	setOrReplaceCondition(&sc.Status.CommonStatus, cond)
}

// mapInstallationToChildren returns a handler.MapFunc that maps a
// DexInstallation event to a request for every child of this reconciler's
// kind that references the installation. Without it, a child keeps a stale
// Ready condition after the installation's allowlists change, until its own
// next event.
//
// listGVK is the GroupVersionKind of the list type of T. The lookup uses the
// InstallationRefIndexField index, which DexInstallationReconciler registers
// for every child kind.
func (r *GenericChildReconciler[T, U]) mapInstallationToChildren(listGVK schema.GroupVersionKind) handler.MapFunc {
	return func(ctx context.Context, obj client.Object) []ctrl.Request {
		logger := log.FromContext(ctx)

		newList, err := r.Scheme.New(listGVK)
		if err != nil {
			logger.Error(err, "creating child list", "kind", listGVK.Kind)
			return nil
		}
		list, ok := newList.(client.ObjectList)
		if !ok {
			logger.Error(fmt.Errorf("%T is not a client.ObjectList", newList), "creating child list")
			return nil
		}

		key := installationRefIndexValue(obj.GetNamespace(), obj.GetName())
		if err := r.List(ctx, list, client.MatchingFields{InstallationRefIndexField: key}); err != nil {
			logger.Error(err, "listing children of DexInstallation", "kind", listGVK.Kind, "installation", key)
			return nil
		}

		items, err := meta.ExtractList(list)
		if err != nil {
			logger.Error(err, "extracting children of DexInstallation", "kind", listGVK.Kind)
			return nil
		}
		requests := make([]ctrl.Request, 0, len(items))
		for _, item := range items {
			if child, ok := item.(client.Object); ok {
				requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(child)})
			}
		}
		return requests
	}
}

// listGVKFor returns the GroupVersionKind of the list type that belongs to
// obj (by convention its Kind plus "List").
func listGVKFor(obj runtime.Object, scheme *runtime.Scheme) (schema.GroupVersionKind, error) {
	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	gvk.Kind += "List"
	if !scheme.Recognizes(gvk) {
		return schema.GroupVersionKind{}, fmt.Errorf("list kind %s is not registered in the scheme", gvk)
	}
	return gvk, nil
}
