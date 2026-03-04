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

package builder

import (
	"context"
	"fmt"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// buildAllOAuthConnectors builds connector entries for all OAuth-style
// connectors (GitHub, GitLab, Google, LinkedIn, Microsoft, OIDC, OAuth2,
// OpenShift, AtlassianCrowd, Gitea, Bitbucket, Keystone).
func buildAllOAuthConnectors(
	ctx context.Context,
	cs ConnectorSet,
	sr SecretResolver,
	envs map[string][]byte,
) ([]ConnectorEntry, []MountedSecret, error) {
	var entries []ConnectorEntry
	var mounts []MountedSecret

	e1, m1, err := buildSocialConnectors(ctx, cs, sr, envs)
	if err != nil {
		return nil, nil, err
	}
	entries = append(entries, e1...)
	mounts = append(mounts, m1...)

	e2, m2, err := buildIdentityConnectors(ctx, cs, sr, envs)
	if err != nil {
		return nil, nil, err
	}
	entries = append(entries, e2...)
	mounts = append(mounts, m2...)

	e3, m3, err := buildHostingConnectors(ctx, cs, sr, envs)
	if err != nil {
		return nil, nil, err
	}
	entries = append(entries, e3...)
	mounts = append(mounts, m3...)

	return entries, mounts, nil
}

// buildSocialConnectors handles GitHub, GitLab, Google, LinkedIn, Microsoft.
func buildSocialConnectors(
	ctx context.Context,
	cs ConnectorSet,
	sr SecretResolver,
	envs map[string][]byte,
) ([]ConnectorEntry, []MountedSecret, error) {
	var entries []ConnectorEntry
	var mounts []MountedSecret

	for i := range cs.GitHub {
		e, m, err := buildGitHubConnector(ctx, &cs.GitHub[i], sr, envs)
		if err != nil {
			return nil, nil, fmt.Errorf("GitHub connector %q: %w", cs.GitHub[i].Name, err)
		}
		entries = append(entries, e)
		mounts = append(mounts, m...)
	}

	for i := range cs.GitLab {
		e, m, err := buildGitLabConnector(ctx, &cs.GitLab[i], sr, envs)
		if err != nil {
			return nil, nil, fmt.Errorf("GitLab connector %q: %w", cs.GitLab[i].Name, err)
		}
		entries = append(entries, e)
		mounts = append(mounts, m...)
	}

	for i := range cs.Google {
		e, m, err := buildGoogleConnector(ctx, &cs.Google[i], sr, envs)
		if err != nil {
			return nil, nil, fmt.Errorf("Google connector %q: %w", cs.Google[i].Name, err)
		}
		entries = append(entries, e)
		mounts = append(mounts, m...)
	}

	for i := range cs.LinkedIn {
		e, m, err := buildLinkedInConnector(ctx, &cs.LinkedIn[i], sr, envs)
		if err != nil {
			return nil, nil, fmt.Errorf("LinkedIn connector %q: %w", cs.LinkedIn[i].Name, err)
		}
		entries = append(entries, e)
		mounts = append(mounts, m...)
	}

	for i := range cs.Microsoft {
		e, m, err := buildMicrosoftConnector(ctx, &cs.Microsoft[i], sr, envs)
		if err != nil {
			return nil, nil, fmt.Errorf("Microsoft connector %q: %w", cs.Microsoft[i].Name, err)
		}
		entries = append(entries, e)
		mounts = append(mounts, m...)
	}

	return entries, mounts, nil
}

// buildIdentityConnectors handles OIDC, OAuth2 (generic), OpenShift,
// AtlassianCrowd, Keystone.
func buildIdentityConnectors(
	ctx context.Context,
	cs ConnectorSet,
	sr SecretResolver,
	envs map[string][]byte,
) ([]ConnectorEntry, []MountedSecret, error) {
	var entries []ConnectorEntry
	var mounts []MountedSecret

	for i := range cs.OIDC {
		e, m, err := buildOIDCConnector(ctx, &cs.OIDC[i], sr, envs)
		if err != nil {
			return nil, nil, fmt.Errorf("OIDC connector %q: %w", cs.OIDC[i].Name, err)
		}
		entries = append(entries, e)
		mounts = append(mounts, m...)
	}

	for i := range cs.OAuth2 {
		e, m, err := buildOAuth2Connector(ctx, &cs.OAuth2[i], sr, envs)
		if err != nil {
			return nil, nil, fmt.Errorf("OAuth2 connector %q: %w", cs.OAuth2[i].Name, err)
		}
		entries = append(entries, e)
		mounts = append(mounts, m...)
	}

	for i := range cs.OpenShift {
		e, m, err := buildOpenShiftConnector(ctx, &cs.OpenShift[i], sr, envs)
		if err != nil {
			return nil, nil, fmt.Errorf("OpenShift connector %q: %w", cs.OpenShift[i].Name, err)
		}
		entries = append(entries, e)
		mounts = append(mounts, m...)
	}

	for i := range cs.AtlassianCrowd {
		e, m, err := buildAtlassianCrowdConnector(ctx, &cs.AtlassianCrowd[i], sr, envs)
		if err != nil {
			return nil, nil, fmt.Errorf("AtlassianCrowd connector %q: %w", cs.AtlassianCrowd[i].Name, err)
		}
		entries = append(entries, e)
		mounts = append(mounts, m...)
	}

	for i := range cs.Keystone {
		e, m, err := buildKeystoneConnector(ctx, &cs.Keystone[i], sr, envs)
		if err != nil {
			return nil, nil, fmt.Errorf("Keystone connector %q: %w", cs.Keystone[i].Name, err)
		}
		entries = append(entries, e)
		mounts = append(mounts, m...)
	}

	return entries, mounts, nil
}

// buildHostingConnectors handles Gitea and Bitbucket.
func buildHostingConnectors(
	ctx context.Context,
	cs ConnectorSet,
	sr SecretResolver,
	envs map[string][]byte,
) ([]ConnectorEntry, []MountedSecret, error) {
	var entries []ConnectorEntry
	var mounts []MountedSecret

	for i := range cs.Gitea {
		e, m, err := buildGiteaConnector(ctx, &cs.Gitea[i], sr, envs)
		if err != nil {
			return nil, nil, fmt.Errorf("Gitea connector %q: %w", cs.Gitea[i].Name, err)
		}
		entries = append(entries, e)
		mounts = append(mounts, m...)
	}

	for i := range cs.Bitbucket {
		e, m, err := buildBitbucketConnector(ctx, &cs.Bitbucket[i], sr, envs)
		if err != nil {
			return nil, nil, fmt.Errorf("Bitbucket connector %q: %w", cs.Bitbucket[i].Name, err)
		}
		entries = append(entries, e)
		mounts = append(mounts, m...)
	}

	return entries, mounts, nil
}

// ── shared OAuth helper ───────────────────────────────────────────────────────

// resolveOAuthCreds resolves the clientID (inline) and clientSecret (env var)
// for connectors that follow the standard OAuth2 client-credentials pattern.
func resolveOAuthCreds(
	ctx context.Context,
	namespace, connType, id string,
	clientIDRef, clientSecretRef dexv1.SecretKeyRef,
	sr SecretResolver,
	envs map[string][]byte,
) (clientID, clientSecretEnvRef string, err error) {
	clientID, err = resolveSecret(ctx, namespace, clientIDRef, sr)
	if err != nil {
		return "", "", fmt.Errorf("clientID: %w", err)
	}

	csKey := connectorEnvKey(connType, id, "CLIENT_SECRET")
	clientSecretEnvRef, err = resolveEnvSecret(ctx, namespace, clientSecretRef, csKey, sr, envs)
	if err != nil {
		return "", "", fmt.Errorf("clientSecret: %w", err)
	}

	return clientID, clientSecretEnvRef, nil
}

// ── GitHub ────────────────────────────────────────────────────────────────────

func buildGitHubConnector(
	ctx context.Context,
	c *dexv1.DexGitHubConnector,
	sr SecretResolver,
	envs map[string][]byte,
) (ConnectorEntry, []MountedSecret, error) {
	id := connectorID(c.Name, c.Spec.ID)
	clientID, csRef, err := resolveOAuthCreds(ctx, c.Namespace, "github", id, c.Spec.ClientIDRef, c.Spec.ClientSecretRef, sr, envs)
	if err != nil {
		return ConnectorEntry{}, nil, err
	}

	cfg := map[string]any{
		"clientID":     clientID,
		"clientSecret": csRef,
	}

	if c.Spec.RedirectURI != "" {
		cfg["redirectURI"] = c.Spec.RedirectURI
	}
	if c.Spec.HostName != "" {
		cfg["hostName"] = c.Spec.HostName
	}
	if len(c.Spec.Orgs) > 0 {
		cfg["orgs"] = buildGitHubOrgs(c.Spec.Orgs)
	}
	if c.Spec.LoadAllGroups {
		cfg["loadAllGroups"] = true
	}
	if c.Spec.TeamNameField != "" {
		cfg["teamNameField"] = c.Spec.TeamNameField
	}
	if c.Spec.UseLoginAsID {
		cfg["useLoginAsID"] = true
	}

	var mounts []MountedSecret
	if c.Spec.RootCARef != nil {
		cfg["rootCA"] = mountCertFile(*c.Spec.RootCARef, c.Namespace, id, "root-ca", &mounts)
	}

	return ConnectorEntry{Type: "github", ID: id, Name: c.Spec.Name, Config: cfg}, mounts, nil
}

func buildGitHubOrgs(orgs []dexv1.GitHubOrg) []map[string]any {
	result := make([]map[string]any, 0, len(orgs))
	for _, o := range orgs {
		entry := map[string]any{"name": o.Name}
		if len(o.Teams) > 0 {
			entry["teams"] = o.Teams
		}
		result = append(result, entry)
	}
	return result
}

// ── GitLab ────────────────────────────────────────────────────────────────────

func buildGitLabConnector(
	ctx context.Context,
	c *dexv1.DexGitLabConnector,
	sr SecretResolver,
	envs map[string][]byte,
) (ConnectorEntry, []MountedSecret, error) {
	id := connectorID(c.Name, c.Spec.ID)
	clientID, csRef, err := resolveOAuthCreds(ctx, c.Namespace, "gitlab", id, c.Spec.ClientIDRef, c.Spec.ClientSecretRef, sr, envs)
	if err != nil {
		return ConnectorEntry{}, nil, err
	}

	cfg := map[string]any{
		"clientID":     clientID,
		"clientSecret": csRef,
	}

	if c.Spec.BaseURL != "" {
		cfg["baseURL"] = c.Spec.BaseURL
	}
	if c.Spec.RedirectURI != "" {
		cfg["redirectURI"] = c.Spec.RedirectURI
	}
	if len(c.Spec.Groups) > 0 {
		cfg["groups"] = c.Spec.Groups
	}
	if c.Spec.UseLoginAsID {
		cfg["useLoginAsID"] = true
	}

	return ConnectorEntry{Type: "gitlab", ID: id, Name: c.Spec.Name, Config: cfg}, nil, nil
}

// ── Google ────────────────────────────────────────────────────────────────────

func buildGoogleConnector(
	ctx context.Context,
	c *dexv1.DexGoogleConnector,
	sr SecretResolver,
	envs map[string][]byte,
) (ConnectorEntry, []MountedSecret, error) {
	id := connectorID(c.Name, c.Spec.ID)
	clientID, csRef, err := resolveOAuthCreds(ctx, c.Namespace, "google", id, c.Spec.ClientIDRef, c.Spec.ClientSecretRef, sr, envs)
	if err != nil {
		return ConnectorEntry{}, nil, err
	}

	cfg := map[string]any{
		"clientID":     clientID,
		"clientSecret": csRef,
	}

	if c.Spec.RedirectURI != "" {
		cfg["redirectURI"] = c.Spec.RedirectURI
	}
	if len(c.Spec.HostedDomains) > 0 {
		cfg["hostedDomains"] = c.Spec.HostedDomains
	}
	if len(c.Spec.Groups) > 0 {
		cfg["groups"] = c.Spec.Groups
	}
	if c.Spec.AdminEmail != "" {
		cfg["adminEmail"] = c.Spec.AdminEmail
	}
	if c.Spec.FetchTransitiveMembership {
		cfg["fetchTransitiveMembership"] = true
	}

	var mounts []MountedSecret
	if c.Spec.ServiceAccountFileRef != nil {
		path := fmt.Sprintf("/etc/dex/secrets/%s-service-account.json", id)
		cfg["serviceAccountFilePath"] = mountSecretAsFile(*c.Spec.ServiceAccountFileRef, c.Namespace, path, &mounts)
	}

	return ConnectorEntry{Type: "google", ID: id, Name: c.Spec.Name, Config: cfg}, mounts, nil
}

// ── LinkedIn ──────────────────────────────────────────────────────────────────

func buildLinkedInConnector(
	ctx context.Context,
	c *dexv1.DexLinkedInConnector,
	sr SecretResolver,
	envs map[string][]byte,
) (ConnectorEntry, []MountedSecret, error) {
	id := connectorID(c.Name, c.Spec.ID)
	clientID, csRef, err := resolveOAuthCreds(ctx, c.Namespace, "linkedin", id, c.Spec.ClientIDRef, c.Spec.ClientSecretRef, sr, envs)
	if err != nil {
		return ConnectorEntry{}, nil, err
	}

	cfg := map[string]any{
		"clientID":     clientID,
		"clientSecret": csRef,
	}
	if c.Spec.RedirectURI != "" {
		cfg["redirectURI"] = c.Spec.RedirectURI
	}

	return ConnectorEntry{Type: "linkedin", ID: id, Name: c.Spec.Name, Config: cfg}, nil, nil
}

// ── Microsoft ─────────────────────────────────────────────────────────────────

func buildMicrosoftConnector(
	ctx context.Context,
	c *dexv1.DexMicrosoftConnector,
	sr SecretResolver,
	envs map[string][]byte,
) (ConnectorEntry, []MountedSecret, error) {
	id := connectorID(c.Name, c.Spec.ID)
	clientID, csRef, err := resolveOAuthCreds(ctx, c.Namespace, "microsoft", id, c.Spec.ClientIDRef, c.Spec.ClientSecretRef, sr, envs)
	if err != nil {
		return ConnectorEntry{}, nil, err
	}

	cfg := map[string]any{
		"clientID":     clientID,
		"clientSecret": csRef,
	}

	if c.Spec.RedirectURI != "" {
		cfg["redirectURI"] = c.Spec.RedirectURI
	}
	if c.Spec.Tenant != "" {
		cfg["tenant"] = c.Spec.Tenant
	}
	if c.Spec.OnlySecurityGroups {
		cfg["onlySecurityGroups"] = true
	}
	if len(c.Spec.Groups) > 0 {
		cfg["groups"] = c.Spec.Groups
	}
	if c.Spec.GroupNameFormat != "" {
		cfg["groupNameFormat"] = c.Spec.GroupNameFormat
	}
	if c.Spec.DomainHint != "" {
		cfg["domainHint"] = c.Spec.DomainHint
	}

	return ConnectorEntry{Type: "microsoft", ID: id, Name: c.Spec.Name, Config: cfg}, nil, nil
}

// ── OIDC ──────────────────────────────────────────────────────────────────────

func buildOIDCConnector(
	ctx context.Context,
	c *dexv1.DexOIDCConnector,
	sr SecretResolver,
	envs map[string][]byte,
) (ConnectorEntry, []MountedSecret, error) {
	id := connectorID(c.Name, c.Spec.ID)
	clientID, csRef, err := resolveOAuthCreds(ctx, c.Namespace, "oidc", id, c.Spec.ClientIDRef, c.Spec.ClientSecretRef, sr, envs)
	if err != nil {
		return ConnectorEntry{}, nil, err
	}

	cfg := map[string]any{
		"issuer":       c.Spec.Issuer,
		"clientID":     clientID,
		"clientSecret": csRef,
	}

	applyOIDCOptionalFields(cfg, c.Spec)

	var mounts []MountedSecret
	if c.Spec.RootCARef != nil {
		cfg["rootCA"] = mountCertFile(*c.Spec.RootCARef, c.Namespace, id, "root-ca", &mounts)
	}

	if c.Spec.ClaimMapping != nil {
		cfg["claimMapping"] = buildOIDCClaimMapping(*c.Spec.ClaimMapping)
	}

	return ConnectorEntry{Type: "oidc", ID: id, Name: c.Spec.Name, Config: cfg}, mounts, nil
}

func applyOIDCOptionalFields(cfg map[string]any, spec dexv1.DexOIDCConnectorSpec) {
	if spec.RedirectURI != "" {
		cfg["redirectURI"] = spec.RedirectURI
	}
	if len(spec.Scopes) > 0 {
		cfg["scopes"] = spec.Scopes
	}
	if spec.GetUserInfo {
		cfg["getUserInfo"] = true
	}
	if spec.UserNameKey != "" {
		cfg["userNameKey"] = spec.UserNameKey
	}
	if spec.UserIDKey != "" {
		cfg["userIDKey"] = spec.UserIDKey
	}
	if spec.PromptType != "" {
		cfg["promptType"] = spec.PromptType
	}
	if spec.OverrideClaimMapping {
		cfg["overrideClaimMapping"] = true
	}
	if spec.InsecureSkipEmailVerified {
		cfg["insecureSkipEmailVerified"] = true
	}
	if spec.InsecureEnableGroups {
		cfg["insecureEnableGroups"] = true
	}
	if spec.BasicAuthUnsupported {
		cfg["basicAuthUnsupported"] = true
	}
	if len(spec.HostedDomains) > 0 {
		cfg["hostedDomains"] = spec.HostedDomains
	}
	if len(spec.ACRValues) > 0 {
		cfg["acrValues"] = spec.ACRValues
	}
	if spec.DiscoveryPollInterval != "" {
		cfg["discoveryPollInterval"] = spec.DiscoveryPollInterval
	}
}

func buildOIDCClaimMapping(m dexv1.OIDCClaimMapping) map[string]any {
	cfg := map[string]any{}
	if m.PreferredUsername != "" {
		cfg["preferred_username"] = m.PreferredUsername
	}
	if m.Email != "" {
		cfg["email"] = m.Email
	}
	if m.Groups != "" {
		cfg["groups"] = m.Groups
	}
	return cfg
}

// ── OAuth2 (generic) ─────────────────────────────────────────────────────────

func buildOAuth2Connector(
	ctx context.Context,
	c *dexv1.DexOAuth2Connector,
	sr SecretResolver,
	envs map[string][]byte,
) (ConnectorEntry, []MountedSecret, error) {
	id := connectorID(c.Name, c.Spec.ID)
	clientID, csRef, err := resolveOAuthCreds(ctx, c.Namespace, "oauth2", id, c.Spec.ClientIDRef, c.Spec.ClientSecretRef, sr, envs)
	if err != nil {
		return ConnectorEntry{}, nil, err
	}

	cfg := map[string]any{
		"clientID":         clientID,
		"clientSecret":     csRef,
		"authorizationURL": c.Spec.AuthorizationURL,
		"tokenURL":         c.Spec.TokenURL,
	}

	if c.Spec.RedirectURI != "" {
		cfg["redirectURI"] = c.Spec.RedirectURI
	}
	if c.Spec.UserInfoURL != "" {
		cfg["userInfoURL"] = c.Spec.UserInfoURL
	}
	if len(c.Spec.Scopes) > 0 {
		cfg["scopes"] = c.Spec.Scopes
	}
	if c.Spec.HeaderPrefix != "" {
		cfg["headerPrefix"] = c.Spec.HeaderPrefix
	}
	if c.Spec.InsecureTrustEmail {
		cfg["insecureTrustEmail"] = true
	}
	if c.Spec.ClaimMapping != nil {
		cfg["claimMapping"] = buildOAuth2ClaimMapping(*c.Spec.ClaimMapping)
	}

	var mounts []MountedSecret
	if c.Spec.RootCARef != nil {
		cfg["rootCA"] = mountCertFile(*c.Spec.RootCARef, c.Namespace, id, "root-ca", &mounts)
	}

	return ConnectorEntry{Type: "oauth", ID: id, Name: c.Spec.Name, Config: cfg}, mounts, nil
}

func buildOAuth2ClaimMapping(m dexv1.OAuth2ClaimMapping) map[string]any {
	cfg := map[string]any{}
	if m.PreferredUsername != "" {
		cfg["preferred_username"] = m.PreferredUsername
	}
	if m.Email != "" {
		cfg["email"] = m.Email
	}
	if m.Groups != "" {
		cfg["groups"] = m.Groups
	}
	return cfg
}

// ── OpenShift ─────────────────────────────────────────────────────────────────

func buildOpenShiftConnector(
	ctx context.Context,
	c *dexv1.DexOpenShiftConnector,
	sr SecretResolver,
	envs map[string][]byte,
) (ConnectorEntry, []MountedSecret, error) {
	id := connectorID(c.Name, c.Spec.ID)
	clientID, csRef, err := resolveOAuthCreds(ctx, c.Namespace, "openshift", id, c.Spec.ClientIDRef, c.Spec.ClientSecretRef, sr, envs)
	if err != nil {
		return ConnectorEntry{}, nil, err
	}

	cfg := map[string]any{
		"issuer":       c.Spec.Issuer,
		"clientID":     clientID,
		"clientSecret": csRef,
	}

	if c.Spec.RedirectURI != "" {
		cfg["redirectURI"] = c.Spec.RedirectURI
	}
	if len(c.Spec.Groups) > 0 {
		cfg["groups"] = c.Spec.Groups
	}
	if c.Spec.InsecureCA {
		cfg["insecureCA"] = true
	}

	var mounts []MountedSecret
	if c.Spec.RootCARef != nil {
		cfg["rootCA"] = mountCertFile(*c.Spec.RootCARef, c.Namespace, id, "root-ca", &mounts)
	}

	return ConnectorEntry{Type: "openshift", ID: id, Name: c.Spec.Name, Config: cfg}, mounts, nil
}

// ── AtlassianCrowd ────────────────────────────────────────────────────────────

func buildAtlassianCrowdConnector(
	ctx context.Context,
	c *dexv1.DexAtlassianCrowdConnector,
	sr SecretResolver,
	envs map[string][]byte,
) (ConnectorEntry, []MountedSecret, error) {
	id := connectorID(c.Name, c.Spec.ID)
	clientID, csRef, err := resolveOAuthCreds(ctx, c.Namespace, "crowd", id, c.Spec.ClientIDRef, c.Spec.ClientSecretRef, sr, envs)
	if err != nil {
		return ConnectorEntry{}, nil, err
	}

	cfg := map[string]any{
		"baseURL":      c.Spec.BaseURL,
		"clientID":     clientID,
		"clientSecret": csRef,
	}

	if c.Spec.RedirectURI != "" {
		cfg["redirectURI"] = c.Spec.RedirectURI
	}
	if len(c.Spec.Groups) > 0 {
		cfg["groups"] = c.Spec.Groups
	}
	if c.Spec.AdminUser != "" {
		cfg["adminUser"] = c.Spec.AdminUser
	}
	if c.Spec.AdminPasswordRef != nil {
		pwKey := connectorEnvKey("crowd", id, "ADMIN_PASSWORD")
		pwRef, err := resolveEnvSecret(ctx, c.Namespace, *c.Spec.AdminPasswordRef, pwKey, sr, envs)
		if err != nil {
			return ConnectorEntry{}, nil, fmt.Errorf("adminPassword: %w", err)
		}
		cfg["adminPassword"] = pwRef
	}

	return ConnectorEntry{Type: "atlassian-crowd", ID: id, Name: c.Spec.Name, Config: cfg}, nil, nil
}

// ── Gitea ─────────────────────────────────────────────────────────────────────

func buildGiteaConnector(
	ctx context.Context,
	c *dexv1.DexGiteaConnector,
	sr SecretResolver,
	envs map[string][]byte,
) (ConnectorEntry, []MountedSecret, error) {
	id := connectorID(c.Name, c.Spec.ID)
	clientID, csRef, err := resolveOAuthCreds(ctx, c.Namespace, "gitea", id, c.Spec.ClientIDRef, c.Spec.ClientSecretRef, sr, envs)
	if err != nil {
		return ConnectorEntry{}, nil, err
	}

	cfg := map[string]any{
		"baseURL":      c.Spec.BaseURL,
		"clientID":     clientID,
		"clientSecret": csRef,
	}

	if c.Spec.RedirectURI != "" {
		cfg["redirectURI"] = c.Spec.RedirectURI
	}
	if len(c.Spec.Orgs) > 0 {
		cfg["orgs"] = c.Spec.Orgs
	}
	if c.Spec.UseLoginAsID {
		cfg["useLoginAsID"] = true
	}
	if c.Spec.InsecureSkipVerify {
		cfg["insecureSkipVerify"] = true
	}

	var mounts []MountedSecret
	if c.Spec.RootCARef != nil {
		cfg["rootCA"] = mountCertFile(*c.Spec.RootCARef, c.Namespace, id, "root-ca", &mounts)
	}

	return ConnectorEntry{Type: "gitea", ID: id, Name: c.Spec.Name, Config: cfg}, mounts, nil
}

// ── Bitbucket ─────────────────────────────────────────────────────────────────

func buildBitbucketConnector(
	ctx context.Context,
	c *dexv1.DexBitbucketConnector,
	sr SecretResolver,
	envs map[string][]byte,
) (ConnectorEntry, []MountedSecret, error) {
	id := connectorID(c.Name, c.Spec.ID)
	clientID, csRef, err := resolveOAuthCreds(ctx, c.Namespace, "bitbucket", id, c.Spec.ClientIDRef, c.Spec.ClientSecretRef, sr, envs)
	if err != nil {
		return ConnectorEntry{}, nil, err
	}

	cfg := map[string]any{
		"clientID":     clientID,
		"clientSecret": csRef,
	}

	if c.Spec.RedirectURI != "" {
		cfg["redirectURI"] = c.Spec.RedirectURI
	}
	if len(c.Spec.Teams) > 0 {
		cfg["teams"] = c.Spec.Teams
	}

	return ConnectorEntry{Type: "bitbucket-cloud", ID: id, Name: c.Spec.Name, Config: cfg}, nil, nil
}

// ── Keystone ──────────────────────────────────────────────────────────────────

func buildKeystoneConnector(
	ctx context.Context,
	c *dexv1.DexKeystoneConnector,
	sr SecretResolver,
	envs map[string][]byte,
) (ConnectorEntry, []MountedSecret, error) {
	id := connectorID(c.Name, c.Spec.ID)
	cfg := map[string]any{
		"keystoneHost": c.Spec.KeystoneHost,
	}

	if c.Spec.KeystoneAdminURL != "" {
		cfg["keystoneAdminURL"] = c.Spec.KeystoneAdminURL
	}
	if c.Spec.Domain != "" {
		cfg["domain"] = c.Spec.Domain
	}
	if c.Spec.AdminUsername != "" {
		cfg["adminUsername"] = c.Spec.AdminUsername
	}
	if c.Spec.AdminPasswordRef != nil {
		pwKey := connectorEnvKey("keystone", id, "ADMIN_PASSWORD")
		pwRef, err := resolveEnvSecret(ctx, c.Namespace, *c.Spec.AdminPasswordRef, pwKey, sr, envs)
		if err != nil {
			return ConnectorEntry{}, nil, fmt.Errorf("adminPassword: %w", err)
		}
		cfg["adminPassword"] = pwRef
	}
	if len(c.Spec.Groups) > 0 {
		cfg["groups"] = c.Spec.Groups
	}

	return ConnectorEntry{Type: "keystone", ID: id, Name: c.Spec.Name, Config: cfg}, nil, nil
}
