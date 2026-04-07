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

package v1

// ─── DexLDAPConnector ─────────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexLDAPConnector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexLDAPConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// GetReferencedSecretNames implements ChildObject.
func (c *DexLDAPConnector) GetReferencedSecretNames() []string {
	return collectSecretNames(c.Spec.RootCARef, c.Spec.ClientCertRef, c.Spec.ClientKeyRef, c.Spec.BindPWRef)
}

// ─── DexGitHubConnector ───────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexGitHubConnector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexGitHubConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// GetReferencedSecretNames implements ChildObject.
func (c *DexGitHubConnector) GetReferencedSecretNames() []string {
	return collectSecretNames(&c.Spec.ClientIDRef, &c.Spec.ClientSecretRef, c.Spec.RootCARef)
}

// ─── DexSAMLConnector ─────────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexSAMLConnector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexSAMLConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// GetReferencedSecretNames implements ChildObject.
func (c *DexSAMLConnector) GetReferencedSecretNames() []string {
	return collectSecretNames(c.Spec.CARef, c.Spec.CABundleRef)
}

// ─── DexGitLabConnector ───────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexGitLabConnector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexGitLabConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// GetReferencedSecretNames implements ChildObject.
func (c *DexGitLabConnector) GetReferencedSecretNames() []string {
	return collectSecretNames(&c.Spec.ClientIDRef, &c.Spec.ClientSecretRef)
}

// ─── DexOIDCConnector ─────────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexOIDCConnector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexOIDCConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// GetReferencedSecretNames implements ChildObject.
func (c *DexOIDCConnector) GetReferencedSecretNames() []string {
	return collectSecretNames(&c.Spec.ClientIDRef, &c.Spec.ClientSecretRef, c.Spec.RootCARef)
}

// ─── DexOAuth2Connector ───────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexOAuth2Connector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexOAuth2Connector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// GetReferencedSecretNames implements ChildObject.
func (c *DexOAuth2Connector) GetReferencedSecretNames() []string {
	return collectSecretNames(&c.Spec.ClientIDRef, &c.Spec.ClientSecretRef, c.Spec.RootCARef)
}

// ─── DexGoogleConnector ───────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexGoogleConnector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexGoogleConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// GetReferencedSecretNames implements ChildObject.
func (c *DexGoogleConnector) GetReferencedSecretNames() []string {
	return collectSecretNames(&c.Spec.ClientIDRef, &c.Spec.ClientSecretRef, c.Spec.ServiceAccountFileRef)
}

// ─── DexLinkedInConnector ─────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexLinkedInConnector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexLinkedInConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// GetReferencedSecretNames implements ChildObject.
func (c *DexLinkedInConnector) GetReferencedSecretNames() []string {
	return collectSecretNames(&c.Spec.ClientIDRef, &c.Spec.ClientSecretRef)
}

// ─── DexMicrosoftConnector ────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexMicrosoftConnector) GetInstallationRef() InstallationRef {
	return c.Spec.InstallationRef
}

// GetCommonStatus implements ChildObject.
func (c *DexMicrosoftConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// GetReferencedSecretNames implements ChildObject.
func (c *DexMicrosoftConnector) GetReferencedSecretNames() []string {
	return collectSecretNames(&c.Spec.ClientIDRef, &c.Spec.ClientSecretRef)
}

// ─── DexAuthProxyConnector ────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexAuthProxyConnector) GetInstallationRef() InstallationRef {
	return c.Spec.InstallationRef
}

// GetCommonStatus implements ChildObject.
func (c *DexAuthProxyConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// GetReferencedSecretNames implements ChildObject.
func (c *DexAuthProxyConnector) GetReferencedSecretNames() []string { return nil }

// ─── DexBitbucketConnector ────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexBitbucketConnector) GetInstallationRef() InstallationRef {
	return c.Spec.InstallationRef
}

// GetCommonStatus implements ChildObject.
func (c *DexBitbucketConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// GetReferencedSecretNames implements ChildObject.
func (c *DexBitbucketConnector) GetReferencedSecretNames() []string {
	return collectSecretNames(&c.Spec.ClientIDRef, &c.Spec.ClientSecretRef)
}

// ─── DexLocalConnector ────────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexLocalConnector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexLocalConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// GetReferencedSecretNames implements ChildObject.
func (c *DexLocalConnector) GetReferencedSecretNames() []string { return nil }

// ─── DexOpenShiftConnector ────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexOpenShiftConnector) GetInstallationRef() InstallationRef {
	return c.Spec.InstallationRef
}

// GetCommonStatus implements ChildObject.
func (c *DexOpenShiftConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// GetReferencedSecretNames implements ChildObject.
func (c *DexOpenShiftConnector) GetReferencedSecretNames() []string {
	return collectSecretNames(&c.Spec.ClientIDRef, &c.Spec.ClientSecretRef, c.Spec.RootCARef)
}

// ─── DexAtlassianCrowdConnector ───────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexAtlassianCrowdConnector) GetInstallationRef() InstallationRef {
	return c.Spec.InstallationRef
}

// GetCommonStatus implements ChildObject.
func (c *DexAtlassianCrowdConnector) GetCommonStatus() *CommonStatus {
	return &c.Status.CommonStatus
}

// GetReferencedSecretNames implements ChildObject.
func (c *DexAtlassianCrowdConnector) GetReferencedSecretNames() []string {
	return collectSecretNames(&c.Spec.ClientIDRef, &c.Spec.ClientSecretRef, c.Spec.AdminPasswordRef)
}

// ─── DexGiteaConnector ────────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexGiteaConnector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexGiteaConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// GetReferencedSecretNames implements ChildObject.
func (c *DexGiteaConnector) GetReferencedSecretNames() []string {
	return collectSecretNames(&c.Spec.ClientIDRef, &c.Spec.ClientSecretRef, c.Spec.RootCARef)
}

// ─── DexKeystoneConnector ─────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexKeystoneConnector) GetInstallationRef() InstallationRef {
	return c.Spec.InstallationRef
}

// GetCommonStatus implements ChildObject.
func (c *DexKeystoneConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// GetReferencedSecretNames implements ChildObject.
func (c *DexKeystoneConnector) GetReferencedSecretNames() []string {
	return collectSecretNames(c.Spec.AdminPasswordRef)
}

// ─── DexStaticClient ──────────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexStaticClient) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexStaticClient) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// GetReferencedSecretNames implements ChildObject.
func (c *DexStaticClient) GetReferencedSecretNames() []string {
	if c.Spec.SecretRef.Name != "" {
		return []string{c.Spec.SecretRef.Name}
	}
	return nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// collectSecretNames gathers unique secret names from a list of optional
// SecretKeyRef pointers.  nil entries and empty names are silently skipped.
func collectSecretNames(refs ...*SecretKeyRef) []string {
	seen := make(map[string]struct{}, len(refs))
	var names []string
	for _, ref := range refs {
		if ref == nil || ref.Name == "" {
			continue
		}
		if _, ok := seen[ref.Name]; ok {
			continue
		}
		seen[ref.Name] = struct{}{}
		names = append(names, ref.Name)
	}
	return names
}
