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

	"sigs.k8s.io/controller-runtime/pkg/client"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
	"github.com/guided-traffic/dex-operator/internal/builder"
)

// InstallationRefIndexField is the field-selector key used by all child
// resources to index their spec.installationRef value.
const InstallationRefIndexField = ".spec.installationRef"

// installationRefIndexValue returns the index value for a given installation
// reference in the format "namespace/name".
func installationRefIndexValue(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

// InstallationRefIndexFunc is the IndexerFunc for all child resource types.
// Register it with mgr.GetFieldIndexer().IndexField for every child GVK.
func InstallationRefIndexFunc(obj client.Object) []string {
	child, ok := obj.(dexv1.ChildObject)
	if !ok {
		return nil
	}
	ref := child.GetInstallationRef()
	return []string{installationRefIndexValue(ref.Namespace, ref.Name)}
}

// filterItems retains only items whose namespace appears in allowed.
// T is the value type; PT is the pointer type that implements client.Object.
// An empty allowed list denies all; "*" allows all.
func filterItems[T any, PT interface {
	*T
	client.Object
}](items []T, allowed []string) []T {
	if len(allowed) == 0 {
		return nil
	}
	if len(allowed) == 1 && allowed[0] == "*" {
		return items
	}
	result := make([]T, 0, len(items))
	for i := range items {
		if isNamespaceAllowed(PT(&items[i]).GetNamespace(), allowed) {
			result = append(result, items[i])
		}
	}
	return result
}

// connectorCollector collects all connector types for a DexInstallation.
// Errors are accumulated: once one listing fails, subsequent calls are no-ops
// and the first error is returned via [connectorCollector.err].
type connectorCollector struct {
	ctx     context.Context //nolint:containedctx
	c       client.Reader
	match   client.MatchingFields
	allowed []string
	err     error
}

// collectConnectors fetches all connectors for the given installation from
// allowed namespaces and assembles a [builder.ConnectorSet].
func collectConnectors(
	ctx context.Context,
	c client.Reader,
	installation *dexv1.DexInstallation,
) (builder.ConnectorSet, error) {
	key := installationRefIndexValue(installation.Namespace, installation.Name)
	cc := &connectorCollector{
		ctx:     ctx,
		c:       c,
		match:   client.MatchingFields{InstallationRefIndexField: key},
		allowed: installation.Spec.AllowedNamespaces,
	}

	cs := builder.ConnectorSet{
		LDAP:           cc.ldap(),
		GitHub:         cc.github(),
		SAML:           cc.saml(),
		GitLab:         cc.gitlab(),
		OIDC:           cc.oidc(),
		OAuth2:         cc.oauth2(),
		Google:         cc.google(),
		LinkedIn:       cc.linkedin(),
		Microsoft:      cc.microsoft(),
		AuthProxy:      cc.authProxy(),
		Bitbucket:      cc.bitbucket(),
		Local:          cc.local(),
		OpenShift:      cc.openShift(),
		AtlassianCrowd: cc.atlassianCrowd(),
		Gitea:          cc.gitea(),
		Keystone:       cc.keystone(),
	}
	return cs, cc.err
}

func (cc *connectorCollector) ldap() []dexv1.DexLDAPConnector {
	var list dexv1.DexLDAPConnectorList
	return doList[dexv1.DexLDAPConnector, *dexv1.DexLDAPConnector](cc, &list, func() []dexv1.DexLDAPConnector { return list.Items }, "LDAP")
}

func (cc *connectorCollector) github() []dexv1.DexGitHubConnector {
	var list dexv1.DexGitHubConnectorList
	return doList[dexv1.DexGitHubConnector, *dexv1.DexGitHubConnector](cc, &list, func() []dexv1.DexGitHubConnector { return list.Items }, "GitHub")
}

func (cc *connectorCollector) saml() []dexv1.DexSAMLConnector {
	var list dexv1.DexSAMLConnectorList
	return doList[dexv1.DexSAMLConnector, *dexv1.DexSAMLConnector](cc, &list, func() []dexv1.DexSAMLConnector { return list.Items }, "SAML")
}

func (cc *connectorCollector) gitlab() []dexv1.DexGitLabConnector {
	var list dexv1.DexGitLabConnectorList
	return doList[dexv1.DexGitLabConnector, *dexv1.DexGitLabConnector](cc, &list, func() []dexv1.DexGitLabConnector { return list.Items }, "GitLab")
}

func (cc *connectorCollector) oidc() []dexv1.DexOIDCConnector {
	var list dexv1.DexOIDCConnectorList
	return doList[dexv1.DexOIDCConnector, *dexv1.DexOIDCConnector](cc, &list, func() []dexv1.DexOIDCConnector { return list.Items }, "OIDC")
}

func (cc *connectorCollector) oauth2() []dexv1.DexOAuth2Connector {
	var list dexv1.DexOAuth2ConnectorList
	return doList[dexv1.DexOAuth2Connector, *dexv1.DexOAuth2Connector](cc, &list, func() []dexv1.DexOAuth2Connector { return list.Items }, "OAuth2")
}

func (cc *connectorCollector) google() []dexv1.DexGoogleConnector {
	var list dexv1.DexGoogleConnectorList
	return doList[dexv1.DexGoogleConnector, *dexv1.DexGoogleConnector](cc, &list, func() []dexv1.DexGoogleConnector { return list.Items }, "Google")
}

func (cc *connectorCollector) linkedin() []dexv1.DexLinkedInConnector {
	var list dexv1.DexLinkedInConnectorList
	return doList[dexv1.DexLinkedInConnector, *dexv1.DexLinkedInConnector](cc, &list, func() []dexv1.DexLinkedInConnector { return list.Items }, "LinkedIn")
}

func (cc *connectorCollector) microsoft() []dexv1.DexMicrosoftConnector {
	var list dexv1.DexMicrosoftConnectorList
	return doList[dexv1.DexMicrosoftConnector, *dexv1.DexMicrosoftConnector](cc, &list, func() []dexv1.DexMicrosoftConnector { return list.Items }, "Microsoft")
}

func (cc *connectorCollector) authProxy() []dexv1.DexAuthProxyConnector {
	var list dexv1.DexAuthProxyConnectorList
	return doList[dexv1.DexAuthProxyConnector, *dexv1.DexAuthProxyConnector](cc, &list, func() []dexv1.DexAuthProxyConnector { return list.Items }, "AuthProxy")
}

func (cc *connectorCollector) bitbucket() []dexv1.DexBitbucketConnector {
	var list dexv1.DexBitbucketConnectorList
	return doList[dexv1.DexBitbucketConnector, *dexv1.DexBitbucketConnector](cc, &list, func() []dexv1.DexBitbucketConnector { return list.Items }, "Bitbucket")
}

func (cc *connectorCollector) local() []dexv1.DexLocalConnector {
	var list dexv1.DexLocalConnectorList
	return doList[dexv1.DexLocalConnector, *dexv1.DexLocalConnector](cc, &list, func() []dexv1.DexLocalConnector { return list.Items }, "Local")
}

func (cc *connectorCollector) openShift() []dexv1.DexOpenShiftConnector {
	var list dexv1.DexOpenShiftConnectorList
	return doList[dexv1.DexOpenShiftConnector, *dexv1.DexOpenShiftConnector](cc, &list, func() []dexv1.DexOpenShiftConnector { return list.Items }, "OpenShift")
}

func (cc *connectorCollector) atlassianCrowd() []dexv1.DexAtlassianCrowdConnector {
	var list dexv1.DexAtlassianCrowdConnectorList
	return doList[dexv1.DexAtlassianCrowdConnector, *dexv1.DexAtlassianCrowdConnector](cc, &list, func() []dexv1.DexAtlassianCrowdConnector { return list.Items }, "AtlassianCrowd")
}

func (cc *connectorCollector) gitea() []dexv1.DexGiteaConnector {
	var list dexv1.DexGiteaConnectorList
	return doList[dexv1.DexGiteaConnector, *dexv1.DexGiteaConnector](cc, &list, func() []dexv1.DexGiteaConnector { return list.Items }, "Gitea")
}

func (cc *connectorCollector) keystone() []dexv1.DexKeystoneConnector {
	var list dexv1.DexKeystoneConnectorList
	return doList[dexv1.DexKeystoneConnector, *dexv1.DexKeystoneConnector](cc, &list, func() []dexv1.DexKeystoneConnector { return list.Items }, "Keystone")
}

// doList is the shared list+filter implementation used by each connector method.
// It is a package-level generic function because Go does not allow type parameters
// on methods.  T is the value type; PT is the pointer type that implements
// client.Object.
func doList[T any, PT interface {
	*T
	client.Object
}](
	cc *connectorCollector,
	objList client.ObjectList,
	getItems func() []T,
	kind string,
) []T {
	if cc.err != nil {
		return nil
	}
	if err := cc.c.List(cc.ctx, objList, cc.match); err != nil {
		cc.err = fmt.Errorf("listing %s connectors: %w", kind, err)
		return nil
	}
	return filterItems[T, PT](getItems(), cc.allowed)
}

// collectStaticClients fetches all DexStaticClient resources for the
// installation from allowed namespaces.
func collectStaticClients(
	ctx context.Context,
	c client.Reader,
	installation *dexv1.DexInstallation,
) ([]dexv1.DexStaticClient, error) {
	key := installationRefIndexValue(installation.Namespace, installation.Name)
	match := client.MatchingFields{InstallationRefIndexField: key}

	var list dexv1.DexStaticClientList
	if err := c.List(ctx, &list, match); err != nil {
		return nil, fmt.Errorf("listing static clients: %w", err)
	}
	return filterItems[dexv1.DexStaticClient, *dexv1.DexStaticClient](list.Items, installation.Spec.AllowedNamespaces), nil
}
