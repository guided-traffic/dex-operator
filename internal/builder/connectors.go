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
	"encoding/base64"
	"fmt"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// buildAllConnectors iterates over every connector type in cs, converts each
// to a ConnectorEntry and returns the combined slice together with any
// MountedSecret entries for TLS-certificate files.
//
// Local connectors are intentionally excluded: their presence is reflected by
// setting EnablePasswordDB in the DexConfig (see [assembleDexConfig]).
func buildAllConnectors(
	ctx context.Context,
	cs ConnectorSet,
	sr SecretResolver,
	envs map[string][]byte,
) ([]ConnectorEntry, []MountedSecret, error) {
	entries := make([]ConnectorEntry, 0, len(cs.LDAP)+len(cs.SAML)+len(cs.AuthProxy))
	var mounts []MountedSecret

	for i := range cs.LDAP {
		e, m, err := buildLDAPConnector(ctx, &cs.LDAP[i], sr, envs)
		if err != nil {
			return nil, nil, fmt.Errorf("LDAP connector %q: %w", cs.LDAP[i].Name, err)
		}
		entries = append(entries, e)
		mounts = append(mounts, m...)
	}

	for i := range cs.SAML {
		e, m, err := buildSAMLConnector(&cs.SAML[i])
		if err != nil {
			return nil, nil, fmt.Errorf("SAML connector %q: %w", cs.SAML[i].Name, err)
		}
		entries = append(entries, e)
		mounts = append(mounts, m...)
	}

	for i := range cs.AuthProxy {
		entries = append(entries, buildAuthProxyConnector(&cs.AuthProxy[i]))
	}

	oauthEntries, oauthMounts, err := buildAllOAuthConnectors(ctx, cs, sr, envs)
	if err != nil {
		return nil, nil, err
	}
	entries = append(entries, oauthEntries...)
	mounts = append(mounts, oauthMounts...)

	return entries, mounts, nil
}

// ── LDAP ─────────────────────────────────────────────────────────────────────

func buildLDAPConnector(
	ctx context.Context,
	c *dexv1.DexLDAPConnector,
	sr SecretResolver,
	envs map[string][]byte,
) (ConnectorEntry, []MountedSecret, error) {
	id := connectorID(c.Name, c.Spec.ID)
	cfg := map[string]any{"host": c.Spec.Host}
	var mounts []MountedSecret

	applyLDAPBoolFlags(cfg, c.Spec)

	if err := applyLDAPTLS(ctx, cfg, c.Spec, id, c.Namespace, sr, &mounts, envs); err != nil {
		return ConnectorEntry{}, nil, err
	}

	if c.Spec.BindDN != "" {
		cfg["bindDN"] = c.Spec.BindDN
	}

	if c.Spec.BindPWRef != nil {
		envKey := connectorEnvKey("ldap", id, "BIND_PW")
		ref, err := resolveEnvSecret(ctx, c.Namespace, *c.Spec.BindPWRef, envKey, sr, envs)
		if err != nil {
			return ConnectorEntry{}, nil, fmt.Errorf("bindPW: %w", err)
		}
		cfg["bindPW"] = ref
	}

	if c.Spec.UsernamePrompt != "" {
		cfg["usernamePrompt"] = c.Spec.UsernamePrompt
	}

	cfg["userSearch"] = buildLDAPUserSearch(c.Spec.UserSearch)

	if c.Spec.GroupSearch != nil {
		cfg["groupSearch"] = buildLDAPGroupSearch(*c.Spec.GroupSearch)
	}

	return ConnectorEntry{Type: "ldap", ID: id, Name: c.Spec.DisplayName, Config: cfg}, mounts, nil
}

func applyLDAPBoolFlags(cfg map[string]any, spec dexv1.DexLDAPConnectorSpec) {
	if spec.InsecureNoSSL {
		cfg["insecureNoSSL"] = true
	}
	if spec.InsecureSkipVerify {
		cfg["insecureSkipVerify"] = true
	}
	if spec.StartTLS {
		cfg["startTLS"] = true
	}
}

func applyLDAPTLS(
	ctx context.Context,
	cfg map[string]any,
	spec dexv1.DexLDAPConnectorSpec,
	id, namespace string,
	sr SecretResolver,
	mounts *[]MountedSecret,
	envs map[string][]byte,
) error {
	if spec.RootCARef != nil {
		val, err := resolveSecret(ctx, namespace, *spec.RootCARef, sr)
		if err != nil {
			return fmt.Errorf("rootCA: %w", err)
		}
		cfg["rootCAData"] = base64.StdEncoding.EncodeToString([]byte(val))
	}

	if spec.ClientCertRef != nil {
		cfg["clientCert"] = mountCertFile(*spec.ClientCertRef, namespace, id, "client-cert", mounts)
	}

	if spec.ClientKeyRef != nil {
		cfg["clientKey"] = mountCertFile(*spec.ClientKeyRef, namespace, id, "client-key", mounts)
	}

	_ = envs // unused for LDAP TLS, reserved for future use
	return nil
}

func buildLDAPUserSearch(s dexv1.LDAPUserSearch) map[string]any {
	cfg := map[string]any{
		"baseDN":   s.BaseDN,
		"username": s.Username,
	}
	if s.Filter != "" {
		cfg["filter"] = s.Filter
	}
	if s.Scope != "" {
		cfg["scope"] = s.Scope
	}
	if s.IDAttr != "" {
		cfg["idAttr"] = s.IDAttr
	}
	if s.EmailAttr != "" {
		cfg["emailAttr"] = s.EmailAttr
	}
	if s.NameAttr != "" {
		cfg["nameAttr"] = s.NameAttr
	}
	if s.PreferredUsernameAttr != "" {
		cfg["preferredUsernameAttr"] = s.PreferredUsernameAttr
	}
	if s.EmailSuffix != "" {
		cfg["emailSuffix"] = s.EmailSuffix
	}
	return cfg
}

func buildLDAPGroupSearch(s dexv1.LDAPGroupSearch) map[string]any {
	cfg := map[string]any{"baseDN": s.BaseDN}
	if s.Filter != "" {
		cfg["filter"] = s.Filter
	}
	if s.Scope != "" {
		cfg["scope"] = s.Scope
	}
	if s.UserAttr != "" {
		cfg["userAttr"] = s.UserAttr
	}
	if s.GroupAttr != "" {
		cfg["groupAttr"] = s.GroupAttr
	}
	if s.NameAttr != "" {
		cfg["nameAttr"] = s.NameAttr
	}
	return cfg
}

// ── SAML ─────────────────────────────────────────────────────────────────────

//nolint:unparam // error return kept for API consistency; this connector never errors
func buildSAMLConnector(c *dexv1.DexSAMLConnector) (ConnectorEntry, []MountedSecret, error) {
	id := connectorID(c.Name, c.Spec.ID)
	cfg := map[string]any{"ssoURL": c.Spec.SSOURL}
	var mounts []MountedSecret

	if c.Spec.CARef != nil {
		cfg["ca"] = mountCertFile(*c.Spec.CARef, c.Namespace, id, "ca", &mounts)
	}
	if c.Spec.CABundleRef != nil {
		cfg["caBundle"] = mountCertFile(*c.Spec.CABundleRef, c.Namespace, id, "ca-bundle", &mounts)
	}
	if c.Spec.SSOIssuer != "" {
		cfg["ssoIssuer"] = c.Spec.SSOIssuer
	}
	if c.Spec.EntityIssuer != "" {
		cfg["entityIssuer"] = c.Spec.EntityIssuer
	}
	if c.Spec.RedirectURI != "" {
		cfg["redirectURI"] = c.Spec.RedirectURI
	}
	if c.Spec.NameIDPolicyFormat != "" {
		cfg["nameIDPolicyFormat"] = c.Spec.NameIDPolicyFormat
	}
	if c.Spec.UsernameAttr != "" {
		cfg["usernameAttr"] = c.Spec.UsernameAttr
	}
	if c.Spec.EmailAttr != "" {
		cfg["emailAttr"] = c.Spec.EmailAttr
	}
	if c.Spec.GroupsAttr != "" {
		cfg["groupsAttr"] = c.Spec.GroupsAttr
	}
	if len(c.Spec.AllowedGroups) > 0 {
		cfg["allowedGroups"] = c.Spec.AllowedGroups
	}
	if c.Spec.InsecureSkipSignatureValidation {
		cfg["insecureSkipSignatureValidation"] = true
	}

	return ConnectorEntry{Type: "saml", ID: id, Name: c.Spec.DisplayName, Config: cfg}, mounts, nil
}

// ── AuthProxy ─────────────────────────────────────────────────────────────────

func buildAuthProxyConnector(c *dexv1.DexAuthProxyConnector) ConnectorEntry {
	id := connectorID(c.Name, c.Spec.ID)
	cfg := map[string]any{}

	if c.Spec.Header != "" {
		cfg["header"] = c.Spec.Header
	}
	if c.Spec.GetUserInfo {
		cfg["getUserInfo"] = true
	}
	if len(c.Spec.Headers) > 0 {
		cfg["headers"] = c.Spec.Headers
	}
	if c.Spec.Groups != "" {
		cfg["groups"] = c.Spec.Groups
	}

	return ConnectorEntry{Type: "authproxy", ID: id, Name: c.Spec.DisplayName, Config: cfg}
}
