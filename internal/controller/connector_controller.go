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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// newChildReconciler is a convenience constructor for GenericChildReconciler.
func newChildReconciler[T interface {
	*U
	client.Object
	ChildObject
}, U any](c client.Client, scheme *runtime.Scheme) *GenericChildReconciler[T, U] {
	return &GenericChildReconciler[T, U]{Client: c, Scheme: scheme}
}

// SetupConnectorControllers registers a reconciler for every connector CRD and
// for DexStaticClient.  It is called once from main during manager setup.
func SetupConnectorControllers(mgr ctrl.Manager) error {
	c := mgr.GetClient()
	s := mgr.GetScheme()

	reconcilers := []interface {
		SetupWithManager(ctrl.Manager) error
	}{
		newChildReconciler[*dexv1.DexLDAPConnector, dexv1.DexLDAPConnector](c, s),
		newChildReconciler[*dexv1.DexGitHubConnector, dexv1.DexGitHubConnector](c, s),
		newChildReconciler[*dexv1.DexSAMLConnector, dexv1.DexSAMLConnector](c, s),
		newChildReconciler[*dexv1.DexGitLabConnector, dexv1.DexGitLabConnector](c, s),
		newChildReconciler[*dexv1.DexOIDCConnector, dexv1.DexOIDCConnector](c, s),
		newChildReconciler[*dexv1.DexOAuth2Connector, dexv1.DexOAuth2Connector](c, s),
		newChildReconciler[*dexv1.DexGoogleConnector, dexv1.DexGoogleConnector](c, s),
		newChildReconciler[*dexv1.DexLinkedInConnector, dexv1.DexLinkedInConnector](c, s),
		newChildReconciler[*dexv1.DexMicrosoftConnector, dexv1.DexMicrosoftConnector](c, s),
		newChildReconciler[*dexv1.DexAuthProxyConnector, dexv1.DexAuthProxyConnector](c, s),
		newChildReconciler[*dexv1.DexBitbucketConnector, dexv1.DexBitbucketConnector](c, s),
		newChildReconciler[*dexv1.DexLocalConnector, dexv1.DexLocalConnector](c, s),
		newChildReconciler[*dexv1.DexOpenShiftConnector, dexv1.DexOpenShiftConnector](c, s),
		newChildReconciler[*dexv1.DexAtlassianCrowdConnector, dexv1.DexAtlassianCrowdConnector](c, s),
		newChildReconciler[*dexv1.DexGiteaConnector, dexv1.DexGiteaConnector](c, s),
		newChildReconciler[*dexv1.DexKeystoneConnector, dexv1.DexKeystoneConnector](c, s),
		newChildReconciler[*dexv1.DexStaticClient, dexv1.DexStaticClient](c, s),
	}

	for i, r := range reconcilers {
		if err := r.SetupWithManager(mgr); err != nil {
			return fmt.Errorf("setting up child controller %d: %w", i, err)
		}
	}
	return nil
}
