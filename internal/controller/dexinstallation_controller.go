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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
	"github.com/guided-traffic/dex-operator/internal/builder"
)

// DexInstallationReconciler reconciles a DexInstallation object.
// RBAC markers are centralised in rbac.go.
type DexInstallationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile implements the reconciliation loop for DexInstallation.
func (r *DexInstallationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var installation dexv1.DexInstallation
	if err := r.Get(ctx, req.NamespacedName, &installation); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	result, reconcileErr := r.reconcileInstallation(ctx, &installation)

	setReadyCondition(&installation.Status.CommonStatus, installation.Generation, reconcileErr)
	if statusErr := r.Status().Update(ctx, &installation); statusErr != nil {
		logger.Error(statusErr, "failed to update DexInstallation status")
	}

	return result, reconcileErr
}

// reconcileInstallation contains the core reconciliation logic.
func (r *DexInstallationReconciler) reconcileInstallation(
	ctx context.Context,
	installation *dexv1.DexInstallation,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	connectors, err := collectConnectors(ctx, r.Client, installation)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("collecting connectors: %w", err)
	}

	clients, err := collectStaticClients(ctx, r.Client, installation)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("collecting static clients: %w", err)
	}

	out, err := builder.Build(ctx, builder.Input{
		Installation:  installation,
		Connectors:    connectors,
		StaticClients: clients,
		Secrets:       r.makeSecretResolver(),
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("building dex config: %w", err)
	}

	labels := map[string]string{
		"app.kubernetes.io/managed-by": "dex-operator",
		"dex.gtrfc.com/installation":   installation.Name,
	}

	configChanged, err := applySecret(
		ctx, r.Client,
		installation.Namespace,
		installation.Spec.ConfigSecretName,
		labels,
		map[string][]byte{"config.yaml": out.ConfigYAML},
	)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("applying config secret: %w", err)
	}

	_, err = applySecret(
		ctx, r.Client,
		installation.Namespace,
		installation.Spec.EnvSecretName,
		labels,
		out.EnvSecretData,
	)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("applying env secret: %w", err)
	}

	if configChanged {
		logger.Info("dex config changed, triggering rollout restart if configured")
		if err := triggerRolloutRestart(ctx, r.Client, installation); err != nil {
			logger.Error(err, "rollout restart failed (non-fatal)")
		}
	}

	installation.Status.ConnectorCount = countConnectors(connectors)
	installation.Status.StaticClientCount = len(clients)
	return ctrl.Result{}, nil
}

// makeSecretResolver returns a [builder.SecretResolver] backed by the API server.
func (r *DexInstallationReconciler) makeSecretResolver() builder.SecretResolver {
	return func(ctx context.Context, namespace string, ref dexv1.SecretKeyRef) (string, error) {
		var secret corev1.Secret
		if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: ref.Name}, &secret); err != nil {
			if errors.IsNotFound(err) {
				return "", fmt.Errorf("secret %s/%s not found", namespace, ref.Name)
			}
			return "", err
		}
		val, ok := secret.Data[ref.Key]
		if !ok {
			return "", fmt.Errorf("key %q not found in secret %s/%s", ref.Key, namespace, ref.Name)
		}
		return string(val), nil
	}
}

// countConnectors returns the total number of connectors across all types.
func countConnectors(cs builder.ConnectorSet) int {
	return len(cs.LDAP) + len(cs.GitHub) + len(cs.SAML) +
		len(cs.GitLab) + len(cs.OIDC) + len(cs.OAuth2) +
		len(cs.Google) + len(cs.LinkedIn) + len(cs.Microsoft) +
		len(cs.AuthProxy) + len(cs.Bitbucket) + len(cs.Local) +
		len(cs.OpenShift) + len(cs.AtlassianCrowd) + len(cs.Gitea) +
		len(cs.Keystone)
}

// mapChildToInstallation is a handler.MapFunc that maps any child object that
// implements [dexv1.ChildObject] to its referenced DexInstallation.
func mapChildToInstallation(_ context.Context, obj client.Object) []ctrl.Request {
	child, ok := obj.(dexv1.ChildObject)
	if !ok {
		return nil
	}
	ref := child.GetInstallationRef()
	return []ctrl.Request{
		{NamespacedName: types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}},
	}
}

// SetupWithManager registers the DexInstallation controller and its secondary
// watches with the controller-runtime manager.
func (r *DexInstallationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.registerIndexers(mgr); err != nil {
		return err
	}
	return r.buildController(mgr)
}

// registerIndexers registers field indexers for all child resource types.
func (r *DexInstallationReconciler) registerIndexers(mgr ctrl.Manager) error {
	indexedTypes := []client.Object{
		&dexv1.DexLDAPConnector{},
		&dexv1.DexGitHubConnector{},
		&dexv1.DexSAMLConnector{},
		&dexv1.DexGitLabConnector{},
		&dexv1.DexOIDCConnector{},
		&dexv1.DexOAuth2Connector{},
		&dexv1.DexGoogleConnector{},
		&dexv1.DexLinkedInConnector{},
		&dexv1.DexMicrosoftConnector{},
		&dexv1.DexAuthProxyConnector{},
		&dexv1.DexBitbucketConnector{},
		&dexv1.DexLocalConnector{},
		&dexv1.DexOpenShiftConnector{},
		&dexv1.DexAtlassianCrowdConnector{},
		&dexv1.DexGiteaConnector{},
		&dexv1.DexKeystoneConnector{},
		&dexv1.DexStaticClient{},
	}

	for _, obj := range indexedTypes {
		if err := mgr.GetFieldIndexer().IndexField(
			context.Background(), obj,
			InstallationRefIndexField,
			InstallationRefIndexFunc,
		); err != nil {
			return fmt.Errorf("registering indexer for %T: %w", obj, err)
		}
	}
	return nil
}

// buildController constructs the controller with all secondary watches.
func (r *DexInstallationReconciler) buildController(mgr ctrl.Manager) error {
	childWatches := childWatchSources()
	b := ctrl.NewControllerManagedBy(mgr).For(&dexv1.DexInstallation{})
	for _, obj := range childWatches {
		b = b.Watches(obj, handler.EnqueueRequestsFromMapFunc(mapChildToInstallation))
	}
	return b.Complete(r)
}

// childWatchSources returns all child object types the DexInstallation
// controller must watch to trigger re-reconciliation.
func childWatchSources() []client.Object {
	return []client.Object{
		&dexv1.DexLDAPConnector{},
		&dexv1.DexGitHubConnector{},
		&dexv1.DexSAMLConnector{},
		&dexv1.DexGitLabConnector{},
		&dexv1.DexOIDCConnector{},
		&dexv1.DexOAuth2Connector{},
		&dexv1.DexGoogleConnector{},
		&dexv1.DexLinkedInConnector{},
		&dexv1.DexMicrosoftConnector{},
		&dexv1.DexAuthProxyConnector{},
		&dexv1.DexBitbucketConnector{},
		&dexv1.DexLocalConnector{},
		&dexv1.DexOpenShiftConnector{},
		&dexv1.DexAtlassianCrowdConnector{},
		&dexv1.DexGiteaConnector{},
		&dexv1.DexKeystoneConnector{},
		&dexv1.DexStaticClient{},
	}
}
