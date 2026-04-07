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

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// secretWatchPredicate returns a predicate that allows Create and Update
// events but suppresses Delete events.  When a referenced Secret is deleted
// while the owning child resource (connector / static client) still exists,
// we assume the Secret will be recreated shortly (e.g. during a credential
// rotation).  Reconciling on the delete would error out (secret not found)
// and cause a noisy retry loop.  The subsequent Create event for the
// replacement Secret will trigger a proper reconcile and rollout restart.
func secretWatchPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc:  func(event.CreateEvent) bool { return true },
		UpdateFunc:  func(event.UpdateEvent) bool { return true },
		DeleteFunc:  func(event.DeleteEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return true },
	}
}

// mapSecretToInstallation maps a changed Secret to the DexInstallation(s)
// that should be re-reconciled. It queries all child resource types
// (connectors and static clients) for objects that reference the changed
// Secret by name and returns requests for their owning DexInstallations.
func (r *DexInstallationReconciler) mapSecretToInstallation(
	ctx context.Context,
	obj client.Object,
) []ctrl.Request {
	secretName := obj.GetName()
	secretNS := obj.GetNamespace()

	seen := make(map[types.NamespacedName]struct{})
	var requests []ctrl.Request

	add := func(ref dexv1.InstallationRef) {
		key := types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			requests = append(requests, ctrl.Request{NamespacedName: key})
		}
	}

	opts := []client.ListOption{
		client.InNamespace(secretNS),
		client.MatchingFields{SecretRefIndexField: secretName},
	}

	lookupChildSecretRefs(ctx, r.Client, opts, add)

	return requests
}

// lookupChildSecretRefs queries every child resource type for objects that
// reference a given Secret (via the SecretRefIndexField) and calls add for
// each discovered installationRef.
func lookupChildSecretRefs(
	ctx context.Context,
	c client.Reader,
	opts []client.ListOption,
	add func(dexv1.InstallationRef),
) {
	// Static clients
	var scList dexv1.DexStaticClientList
	lookupChildBySecret[dexv1.DexStaticClient, *dexv1.DexStaticClient](
		ctx, c, &scList, func() []dexv1.DexStaticClient { return scList.Items }, opts, add)

	// Connector types that reference secrets (AuthProxy and Local are excluded)
	var ldapList dexv1.DexLDAPConnectorList
	lookupChildBySecret[dexv1.DexLDAPConnector, *dexv1.DexLDAPConnector](
		ctx, c, &ldapList, func() []dexv1.DexLDAPConnector { return ldapList.Items }, opts, add)

	var githubList dexv1.DexGitHubConnectorList
	lookupChildBySecret[dexv1.DexGitHubConnector, *dexv1.DexGitHubConnector](
		ctx, c, &githubList, func() []dexv1.DexGitHubConnector { return githubList.Items }, opts, add)

	var samlList dexv1.DexSAMLConnectorList
	lookupChildBySecret[dexv1.DexSAMLConnector, *dexv1.DexSAMLConnector](
		ctx, c, &samlList, func() []dexv1.DexSAMLConnector { return samlList.Items }, opts, add)

	var gitlabList dexv1.DexGitLabConnectorList
	lookupChildBySecret[dexv1.DexGitLabConnector, *dexv1.DexGitLabConnector](
		ctx, c, &gitlabList, func() []dexv1.DexGitLabConnector { return gitlabList.Items }, opts, add)

	var oidcList dexv1.DexOIDCConnectorList
	lookupChildBySecret[dexv1.DexOIDCConnector, *dexv1.DexOIDCConnector](
		ctx, c, &oidcList, func() []dexv1.DexOIDCConnector { return oidcList.Items }, opts, add)

	var oauth2List dexv1.DexOAuth2ConnectorList
	lookupChildBySecret[dexv1.DexOAuth2Connector, *dexv1.DexOAuth2Connector](
		ctx, c, &oauth2List, func() []dexv1.DexOAuth2Connector { return oauth2List.Items }, opts, add)

	var googleList dexv1.DexGoogleConnectorList
	lookupChildBySecret[dexv1.DexGoogleConnector, *dexv1.DexGoogleConnector](
		ctx, c, &googleList, func() []dexv1.DexGoogleConnector { return googleList.Items }, opts, add)

	var linkedInList dexv1.DexLinkedInConnectorList
	lookupChildBySecret[dexv1.DexLinkedInConnector, *dexv1.DexLinkedInConnector](
		ctx, c, &linkedInList, func() []dexv1.DexLinkedInConnector { return linkedInList.Items }, opts, add)

	var msftList dexv1.DexMicrosoftConnectorList
	lookupChildBySecret[dexv1.DexMicrosoftConnector, *dexv1.DexMicrosoftConnector](
		ctx, c, &msftList, func() []dexv1.DexMicrosoftConnector { return msftList.Items }, opts, add)

	var bitbucketList dexv1.DexBitbucketConnectorList
	lookupChildBySecret[dexv1.DexBitbucketConnector, *dexv1.DexBitbucketConnector](
		ctx, c, &bitbucketList, func() []dexv1.DexBitbucketConnector { return bitbucketList.Items }, opts, add)

	var openshiftList dexv1.DexOpenShiftConnectorList
	lookupChildBySecret[dexv1.DexOpenShiftConnector, *dexv1.DexOpenShiftConnector](
		ctx, c, &openshiftList, func() []dexv1.DexOpenShiftConnector { return openshiftList.Items }, opts, add)

	var crowdList dexv1.DexAtlassianCrowdConnectorList
	lookupChildBySecret[dexv1.DexAtlassianCrowdConnector, *dexv1.DexAtlassianCrowdConnector](
		ctx, c, &crowdList, func() []dexv1.DexAtlassianCrowdConnector { return crowdList.Items }, opts, add)

	var giteaList dexv1.DexGiteaConnectorList
	lookupChildBySecret[dexv1.DexGiteaConnector, *dexv1.DexGiteaConnector](
		ctx, c, &giteaList, func() []dexv1.DexGiteaConnector { return giteaList.Items }, opts, add)

	var keystoneList dexv1.DexKeystoneConnectorList
	lookupChildBySecret[dexv1.DexKeystoneConnector, *dexv1.DexKeystoneConnector](
		ctx, c, &keystoneList, func() []dexv1.DexKeystoneConnector { return keystoneList.Items }, opts, add)
}

// lookupChildBySecret is a generic helper that lists child objects matching
// the given options and calls add for each discovered installationRef.
func lookupChildBySecret[T any, PT interface {
	*T
	client.Object
	ChildObject
}](
	ctx context.Context,
	c client.Reader,
	list client.ObjectList,
	getItems func() []T,
	opts []client.ListOption,
	add func(dexv1.InstallationRef),
) {
	if err := c.List(ctx, list, opts...); err != nil {
		return // best effort: if the list fails, skip this type
	}
	items := getItems()
	for i := range items {
		add(PT(&items[i]).GetInstallationRef())
	}
}
