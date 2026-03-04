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

// ChildObject is implemented by all child resources of a DexInstallation
// (connectors and static clients). It allows controllers to use shared
// reconciliation logic regardless of the concrete type.
type ChildObject interface {
	// GetInstallationRef returns the reference to the owning DexInstallation.
	GetInstallationRef() InstallationRef
	// GetCommonStatus returns a pointer to the shared status block so
	// controllers can read / write conditions and observedGeneration.
	GetCommonStatus() *CommonStatus
}

// ─── DexLDAPConnector ─────────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexLDAPConnector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexLDAPConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// ─── DexGitHubConnector ───────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexGitHubConnector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexGitHubConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// ─── DexSAMLConnector ─────────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexSAMLConnector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexSAMLConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// ─── DexGitLabConnector ───────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexGitLabConnector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexGitLabConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// ─── DexOIDCConnector ─────────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexOIDCConnector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexOIDCConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// ─── DexOAuth2Connector ───────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexOAuth2Connector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexOAuth2Connector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// ─── DexGoogleConnector ───────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexGoogleConnector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexGoogleConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// ─── DexLinkedInConnector ─────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexLinkedInConnector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexLinkedInConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// ─── DexMicrosoftConnector ────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexMicrosoftConnector) GetInstallationRef() InstallationRef {
	return c.Spec.InstallationRef
}

// GetCommonStatus implements ChildObject.
func (c *DexMicrosoftConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// ─── DexAuthProxyConnector ────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexAuthProxyConnector) GetInstallationRef() InstallationRef {
	return c.Spec.InstallationRef
}

// GetCommonStatus implements ChildObject.
func (c *DexAuthProxyConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// ─── DexBitbucketConnector ────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexBitbucketConnector) GetInstallationRef() InstallationRef {
	return c.Spec.InstallationRef
}

// GetCommonStatus implements ChildObject.
func (c *DexBitbucketConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// ─── DexLocalConnector ────────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexLocalConnector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexLocalConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// ─── DexOpenShiftConnector ────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexOpenShiftConnector) GetInstallationRef() InstallationRef {
	return c.Spec.InstallationRef
}

// GetCommonStatus implements ChildObject.
func (c *DexOpenShiftConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// ─── DexAtlassianCrowdConnector ───────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexAtlassianCrowdConnector) GetInstallationRef() InstallationRef {
	return c.Spec.InstallationRef
}

// GetCommonStatus implements ChildObject.
func (c *DexAtlassianCrowdConnector) GetCommonStatus() *CommonStatus {
	return &c.Status.CommonStatus
}

// ─── DexGiteaConnector ────────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexGiteaConnector) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexGiteaConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// ─── DexKeystoneConnector ─────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexKeystoneConnector) GetInstallationRef() InstallationRef {
	return c.Spec.InstallationRef
}

// GetCommonStatus implements ChildObject.
func (c *DexKeystoneConnector) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }

// ─── DexStaticClient ──────────────────────────────────────────────────────────

// GetInstallationRef implements ChildObject.
func (c *DexStaticClient) GetInstallationRef() InstallationRef { return c.Spec.InstallationRef }

// GetCommonStatus implements ChildObject.
func (c *DexStaticClient) GetCommonStatus() *CommonStatus { return &c.Status.CommonStatus }
