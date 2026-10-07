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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// connectorBuildFunc builds the config entry of one connector of type T.
type connectorBuildFunc[T any] func(
	ctx context.Context,
	c *T,
	sr SecretResolver,
	env *childEnv,
) (ConnectorEntry, []MountedSecret, error)

// connectorUnits returns one render unit per connector in render order:
// LDAP, SAML, AuthProxy, then the OAuth-style kinds (see
// [oauthConnectorUnits]), each kind in input order.  Every unit claims the
// connector's effective ID; claiming needs no Secret.
//
// Local connectors are intentionally excluded: their presence is reflected by
// setting EnablePasswordDB in the DexConfig (see [assembleDexConfig]).
func connectorUnits(cs ConnectorSet, sr SecretResolver) []*renderUnit {
	var units []*renderUnit
	units = addConnectorUnits(units, "DexLDAPConnector", cs.LDAP,
		func(c *dexv1.DexLDAPConnector) string { return c.Spec.ID }, buildLDAPConnector, sr)
	units = addConnectorUnits(units, "DexSAMLConnector", cs.SAML,
		func(c *dexv1.DexSAMLConnector) string { return c.Spec.ID },
		func(_ context.Context, c *dexv1.DexSAMLConnector, _ SecretResolver, _ *childEnv) (ConnectorEntry, []MountedSecret, error) {
			return buildSAMLConnector(c)
		}, sr)
	units = addConnectorUnits(units, "DexAuthProxyConnector", cs.AuthProxy,
		func(c *dexv1.DexAuthProxyConnector) string { return c.Spec.ID },
		func(_ context.Context, c *dexv1.DexAuthProxyConnector, _ SecretResolver, _ *childEnv) (ConnectorEntry, []MountedSecret, error) {
			return buildAuthProxyConnector(c), nil, nil
		}, sr)
	return oauthConnectorUnits(units, cs, sr)
}

// addConnectorUnits appends one render unit per item to units.  specID
// returns the item's spec.id, build builds its entry.
func addConnectorUnits[T any, PT interface {
	*T
	metav1.Object
}](
	units []*renderUnit,
	kind string,
	items []T,
	specID func(*T) string,
	build connectorBuildFunc[T],
	sr SecretResolver,
) []*renderUnit {
	for i := range items {
		c := &items[i]
		obj := PT(c)
		units = append(units, &renderUnit{
			kind:      kind,
			namespace: obj.GetNamespace(),
			name:      obj.GetName(),
			created:   obj.GetCreationTimestamp(),
			id:        connectorID(obj.GetName(), specID(c)),
			resolved:  true,
			build: func(ctx context.Context, env *childEnv) (unitResult, error) {
				e, m, err := build(ctx, c, sr, env)
				return unitResult{connector: e, mounts: m}, err
			},
		})
	}
	return units
}

// reserveLocal marks every connector that claims Dex's own password
// database ID.  Called only while a DexLocalConnector renders.
func reserveLocal(units []*renderUnit) {
	for _, u := range units {
		if u.id == localConnectorID {
			u.reserved = true
		}
	}
}

// renderedConnectors returns the entries and mounted files of the rendered
// connectors in render order.
func renderedConnectors(units []*renderUnit) ([]ConnectorEntry, []MountedSecret) {
	entries := make([]ConnectorEntry, 0, len(units))
	var mounts []MountedSecret
	for _, u := range units {
		if !u.rendered {
			continue
		}
		entries = append(entries, u.result.connector)
		mounts = append(mounts, u.result.mounts...)
	}
	return entries, mounts
}

// ── LDAP ─────────────────────────────────────────────────────────────────────

func buildLDAPConnector(
	ctx context.Context,
	c *dexv1.DexLDAPConnector,
	sr SecretResolver,
	env *childEnv,
) (ConnectorEntry, []MountedSecret, error) {
	id := connectorID(c.Name, c.Spec.ID)
	cfg := map[string]any{cfgKeyHost: c.Spec.Host}
	var mounts []MountedSecret

	applyLDAPBoolFlags(cfg, c.Spec)

	if err := applyLDAPTLS(ctx, cfg, c.Spec, id, c.Namespace, sr, &mounts); err != nil {
		return ConnectorEntry{}, nil, err
	}

	if c.Spec.BindDN != "" {
		cfg["bindDN"] = c.Spec.BindDN
	}

	if c.Spec.BindPWRef != nil {
		ref, err := resolveEnvSecret(ctx, c.Namespace, *c.Spec.BindPWRef, connectorEnvBase("ldap", id), "BIND_PW", sr, env)
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
	matchers := make([]map[string]any, 0, len(s.UserMatchers))
	for _, m := range s.UserMatchers {
		entry := map[string]any{
			"userAttr":  m.UserAttr,
			"groupAttr": m.GroupAttr,
		}
		if m.RecursionGroupAttr != "" {
			entry["recursionGroupAttr"] = m.RecursionGroupAttr
		}
		matchers = append(matchers, entry)
	}
	cfg["userMatchers"] = matchers
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

	if c.Spec.UserIDHeader != "" {
		cfg["userIDHeader"] = c.Spec.UserIDHeader
	}
	if c.Spec.UserHeader != "" {
		cfg["userHeader"] = c.Spec.UserHeader
	}
	if c.Spec.UserNameHeader != "" {
		cfg["userNameHeader"] = c.Spec.UserNameHeader
	}
	if c.Spec.EmailHeader != "" {
		cfg["emailHeader"] = c.Spec.EmailHeader
	}
	if c.Spec.GroupHeader != "" {
		cfg["groupHeader"] = c.Spec.GroupHeader
	}
	if c.Spec.GroupHeaderSeparator != "" {
		cfg["groupHeaderSeparator"] = c.Spec.GroupHeaderSeparator
	}
	if len(c.Spec.StaticGroups) > 0 {
		cfg["staticGroups"] = c.Spec.StaticGroups
	}

	return ConnectorEntry{Type: "authproxy", ID: id, Name: c.Spec.DisplayName, Config: cfg}
}
