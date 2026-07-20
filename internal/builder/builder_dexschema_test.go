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

package builder_test

// These tests pin the emitted config keys to the real upstream Dex schema
// (dexidp/dex). They guard against re-introducing "ghost" keys — keys that
// Dex silently ignores or rejects — for connectors and storage backends whose
// mapping was corrected.

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
	"github.com/guided-traffic/dex-operator/internal/builder"
)

// firstConnectorConfig builds cs against a minimal installation and returns the
// config map of the first emitted connector.
func firstConnectorConfig(t *testing.T, cs builder.ConnectorSet, secrets map[string]string) map[string]any {
	t.Helper()
	out, err := builder.Build(context.Background(), builder.Input{
		Installation: minimalInstallation("ns"),
		Connectors:   cs,
		Secrets:      mockResolver(secrets),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := parseYAML(t, out.ConfigYAML)
	conns, ok := m["connectors"].([]any)
	if !ok || len(conns) == 0 {
		t.Fatalf("no connectors emitted: %v", m["connectors"])
	}
	return conns[0].(map[string]any)["config"].(map[string]any)
}

func assertNoKeys(t *testing.T, cfg map[string]any, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if _, ok := cfg[k]; ok {
			t.Errorf("config must not contain ghost key %q: %v", k, cfg[k])
		}
	}
}

func TestBuild_AuthProxyConnector_DexSchema(t *testing.T) {
	cfg := firstConnectorConfig(t, builder.ConnectorSet{
		AuthProxy: []dexv1.DexAuthProxyConnector{{
			ObjectMeta: metav1.ObjectMeta{Name: "ap", Namespace: "ns"},
			Spec: dexv1.DexAuthProxyConnectorSpec{
				InstallationRef: dexv1.InstallationRef{Name: "test", Namespace: "ns"},
				DisplayName:     "AuthProxy",
				UserHeader:      "X-Remote-User",
				GroupHeader:     "X-Remote-Group",
				StaticGroups:    []string{"admins"},
			},
		}},
	}, nil)

	if cfg["userHeader"] != "X-Remote-User" {
		t.Errorf("userHeader = %v", cfg["userHeader"])
	}
	if cfg["groupHeader"] != "X-Remote-Group" {
		t.Errorf("groupHeader = %v", cfg["groupHeader"])
	}
	groups := cfg["staticGroups"].([]any)
	if len(groups) != 1 || groups[0] != "admins" {
		t.Errorf("staticGroups = %v", groups)
	}
	assertNoKeys(t, cfg, "header", "getUserInfo", "headers", "groups")
}

func TestBuild_KeystoneConnector_DexSchema(t *testing.T) {
	cfg := firstConnectorConfig(t, builder.ConnectorSet{
		Keystone: []dexv1.DexKeystoneConnector{{
			ObjectMeta: metav1.ObjectMeta{Name: "kst", Namespace: "ns"},
			Spec: dexv1.DexKeystoneConnectorSpec{
				InstallationRef:     dexv1.InstallationRef{Name: "test", Namespace: "ns"},
				DisplayName:         "Keystone",
				KeystoneHost:        "https://keystone.example.com:5000",
				Domain:              "default",
				KeystoneUsername:    "admin",
				KeystonePasswordRef: &dexv1.SecretKeyRef{Name: "kst-creds", Key: "pw"},
			},
		}},
	}, map[string]string{"ns/kst-creds[pw]": "s3cr3t"})

	if cfg["keystoneHost"] != "https://keystone.example.com:5000" {
		t.Errorf("keystoneHost = %v", cfg["keystoneHost"])
	}
	if cfg["keystoneUsername"] != "admin" {
		t.Errorf("keystoneUsername = %v", cfg["keystoneUsername"])
	}
	if cfg["keystonePassword"] != "$KEYSTONE_KST_PASSWORD" {
		t.Errorf("keystonePassword = %v, want $KEYSTONE_KST_PASSWORD", cfg["keystonePassword"])
	}
	assertNoKeys(t, cfg, "adminUsername", "adminPassword", "keystoneAdminURL", "groups")
}

func TestBuild_AtlassianCrowdConnector_DexSchema(t *testing.T) {
	cfg := firstConnectorConfig(t, builder.ConnectorSet{
		AtlassianCrowd: []dexv1.DexAtlassianCrowdConnector{{
			ObjectMeta: metav1.ObjectMeta{Name: "crowd", Namespace: "ns"},
			Spec: dexv1.DexAtlassianCrowdConnectorSpec{
				InstallationRef:        dexv1.InstallationRef{Name: "test", Namespace: "ns"},
				DisplayName:            "Crowd",
				BaseURL:                "https://crowd.example.com",
				ClientIDRef:            dexv1.SecretKeyRef{Name: "crowd-creds", Key: "id"},
				ClientSecretRef:        dexv1.SecretKeyRef{Name: "crowd-creds", Key: "secret"},
				Groups:                 []string{"dev"},
				PreferredUsernameField: "email",
				UsernamePrompt:         "Crowd Login",
			},
		}},
	}, map[string]string{"ns/crowd-creds[id]": "app", "ns/crowd-creds[secret]": "pw"})

	if cfg["preferredUsernameField"] != "email" {
		t.Errorf("preferredUsernameField = %v", cfg["preferredUsernameField"])
	}
	if cfg["usernamePrompt"] != "Crowd Login" {
		t.Errorf("usernamePrompt = %v", cfg["usernamePrompt"])
	}
	assertNoKeys(t, cfg, "redirectURI", "adminUser", "adminPassword")
}

func TestBuild_OAuth2Connector_DexSchema(t *testing.T) {
	cfg := firstConnectorConfig(t, builder.ConnectorSet{
		OAuth2: []dexv1.DexOAuth2Connector{{
			ObjectMeta: metav1.ObjectMeta{Name: "oa", Namespace: "ns"},
			Spec: dexv1.DexOAuth2ConnectorSpec{
				InstallationRef:    dexv1.InstallationRef{Name: "test", Namespace: "ns"},
				DisplayName:        "OAuth2",
				ClientIDRef:        dexv1.SecretKeyRef{Name: "oa-creds", Key: "id"},
				ClientSecretRef:    dexv1.SecretKeyRef{Name: "oa-creds", Key: "secret"},
				AuthorizationURL:   "https://idp.example.com/authorize",
				TokenURL:           "https://idp.example.com/token",
				InsecureSkipVerify: true,
				RootCARef:          &dexv1.SecretKeyRef{Name: "oa-ca", Key: "ca.pem"},
				ClaimMapping: &dexv1.OAuth2ClaimMapping{
					PreferredUsername: "preferred_username",
					Email:             "mail",
					Groups:            "roles",
				},
			},
		}},
	}, map[string]string{"ns/oa-creds[id]": "id", "ns/oa-creds[secret]": "secret"})

	rootCAs := cfg["rootCAs"].([]any)
	if len(rootCAs) != 1 || rootCAs[0] != "/etc/dex/certs/oa-root-ca.pem" {
		t.Errorf("rootCAs = %v", rootCAs)
	}
	if cfg["insecureSkipVerify"] != true {
		t.Errorf("insecureSkipVerify = %v", cfg["insecureSkipVerify"])
	}
	cm := cfg["claimMapping"].(map[string]any)
	if cm["preferredUsernameKey"] != "preferred_username" || cm["emailKey"] != "mail" || cm["groupsKey"] != "roles" {
		t.Errorf("claimMapping = %v", cm)
	}
	assertNoKeys(t, cfg, "rootCA", "headerPrefix", "insecureTrustEmail")
	assertNoKeys(t, cm, "preferred_username", "email", "groups")
}

func TestBuild_OIDCConnector_RootCAs_DexSchema(t *testing.T) {
	cfg := firstConnectorConfig(t, builder.ConnectorSet{
		OIDC: []dexv1.DexOIDCConnector{{
			ObjectMeta: metav1.ObjectMeta{Name: "okta", Namespace: "ns"},
			Spec: dexv1.DexOIDCConnectorSpec{
				InstallationRef: dexv1.InstallationRef{Name: "test", Namespace: "ns"},
				DisplayName:     "Okta",
				Issuer:          "https://example.okta.com",
				ClientIDRef:     dexv1.SecretKeyRef{Name: "okta-creds", Key: "id"},
				ClientSecretRef: dexv1.SecretKeyRef{Name: "okta-creds", Key: "secret"},
				RootCARef:       &dexv1.SecretKeyRef{Name: "okta-ca", Key: "ca.pem"},
			},
		}},
	}, map[string]string{"ns/okta-creds[id]": "id", "ns/okta-creds[secret]": "secret"})

	rootCAs := cfg["rootCAs"].([]any)
	if len(rootCAs) != 1 || rootCAs[0] != "/etc/dex/certs/okta-root-ca.pem" {
		t.Errorf("rootCAs = %v", rootCAs)
	}
	assertNoKeys(t, cfg, "rootCA", "discoveryPollInterval")
}

func TestBuild_GiteaConnector_DexSchema(t *testing.T) {
	cfg := firstConnectorConfig(t, builder.ConnectorSet{
		Gitea: []dexv1.DexGiteaConnector{{
			ObjectMeta: metav1.ObjectMeta{Name: "gitea", Namespace: "ns"},
			Spec: dexv1.DexGiteaConnectorSpec{
				InstallationRef: dexv1.InstallationRef{Name: "test", Namespace: "ns"},
				DisplayName:     "Gitea",
				BaseURL:         "https://gitea.example.com",
				ClientIDRef:     dexv1.SecretKeyRef{Name: "gitea-creds", Key: "id"},
				ClientSecretRef: dexv1.SecretKeyRef{Name: "gitea-creds", Key: "secret"},
				Orgs:            []dexv1.GiteaOrg{{Name: "myorg", Teams: []string{"platform"}}},
				LoadAllGroups:   true,
			},
		}},
	}, map[string]string{"ns/gitea-creds[id]": "id", "ns/gitea-creds[secret]": "secret"})

	orgs := cfg["orgs"].([]any)
	if len(orgs) != 1 {
		t.Fatalf("expected 1 org, got %d", len(orgs))
	}
	org := orgs[0].(map[string]any)
	if org["name"] != "myorg" {
		t.Errorf("org name = %v", org["name"])
	}
	teams := org["teams"].([]any)
	if len(teams) != 1 || teams[0] != "platform" {
		t.Errorf("org teams = %v", teams)
	}
	if cfg["loadAllGroups"] != true {
		t.Errorf("loadAllGroups = %v", cfg["loadAllGroups"])
	}
	assertNoKeys(t, cfg, "rootCA", "insecureSkipVerify")
}

func TestBuild_GoogleConnector_DexSchema(t *testing.T) {
	cfg := firstConnectorConfig(t, builder.ConnectorSet{
		Google: []dexv1.DexGoogleConnector{{
			ObjectMeta: metav1.ObjectMeta{Name: "google", Namespace: "ns"},
			Spec: dexv1.DexGoogleConnectorSpec{
				InstallationRef:                dexv1.InstallationRef{Name: "test", Namespace: "ns"},
				DisplayName:                    "Google",
				ClientIDRef:                    dexv1.SecretKeyRef{Name: "g-creds", Key: "id"},
				ClientSecretRef:                dexv1.SecretKeyRef{Name: "g-creds", Key: "secret"},
				FetchTransitiveGroupMembership: true,
			},
		}},
	}, map[string]string{"ns/g-creds[id]": "id", "ns/g-creds[secret]": "secret"})

	if cfg["fetchTransitiveGroupMembership"] != true {
		t.Errorf("fetchTransitiveGroupMembership = %v", cfg["fetchTransitiveGroupMembership"])
	}
	assertNoKeys(t, cfg, "fetchTransitiveMembership")
}

func TestBuild_SAMLConnector_NoCABundle_DexSchema(t *testing.T) {
	cfg := firstConnectorConfig(t, builder.ConnectorSet{
		SAML: []dexv1.DexSAMLConnector{{
			ObjectMeta: metav1.ObjectMeta{Name: "saml", Namespace: "ns"},
			Spec: dexv1.DexSAMLConnectorSpec{
				InstallationRef: dexv1.InstallationRef{Name: "test", Namespace: "ns"},
				DisplayName:     "SAML",
				SSOURL:          "https://idp.example.com/sso",
				CARef:           &dexv1.SecretKeyRef{Name: "saml-ca", Key: "ca.crt"},
			},
		}},
	}, nil)

	if cfg["ca"] != "/etc/dex/certs/saml-ca.pem" {
		t.Errorf("ca = %v", cfg["ca"])
	}
	assertNoKeys(t, cfg, "caBundle")
}

func TestBuild_MySQLStorage_DexSchema(t *testing.T) {
	inst := minimalInstallation("ns")
	inst.Spec.Storage = dexv1.DexStorageSpec{
		Type: dexv1.StorageMySQL,
		MySQL: &dexv1.DexMySQLStorageSpec{
			Host:        "mysql:3306",
			Database:    "dex",
			User:        "dex",
			PasswordRef: &dexv1.SecretKeyRef{Name: "mysql-creds", Key: "pw"},
		},
	}

	out, err := builder.Build(context.Background(), builder.Input{
		Installation: inst,
		Secrets:      mockResolver(map[string]string{"ns/mysql-creds[pw]": "s3cr3t"}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := parseYAML(t, out.ConfigYAML)
	storage := m["storage"].(map[string]any)
	if storage["type"] != "mysql" {
		t.Errorf("storage type = %v", storage["type"])
	}
	cfg := storage["config"].(map[string]any)
	if cfg["host"] != "mysql:3306" || cfg["database"] != "dex" || cfg["user"] != "dex" {
		t.Errorf("mysql config = %v", cfg)
	}
	if cfg["password"] != "$STORAGE_MYSQL_PASSWORD" {
		t.Errorf("mysql password = %v, want $STORAGE_MYSQL_PASSWORD", cfg["password"])
	}
	if string(out.EnvSecretData["STORAGE_MYSQL_PASSWORD"]) != "s3cr3t" {
		t.Errorf("env STORAGE_MYSQL_PASSWORD = %q", string(out.EnvSecretData["STORAGE_MYSQL_PASSWORD"]))
	}
	assertNoKeys(t, cfg, "dsn")
}

func TestBuild_StaticClient_NoAllowedScopes_DexSchema(t *testing.T) {
	out, err := builder.Build(context.Background(), builder.Input{
		Installation: minimalInstallation("ns"),
		StaticClients: []dexv1.DexStaticClient{{
			ObjectMeta: metav1.ObjectMeta{Name: "grafana", Namespace: "ns"},
			Spec: dexv1.DexStaticClientSpec{
				InstallationRef: dexv1.InstallationRef{Name: "test", Namespace: "ns"},
				SecretRef: dexv1.StaticClientSecretRef{
					Name:            "grafana-oidc",
					ClientIDKey:     "client-id",
					ClientSecretKey: "client-secret",
				},
				DisplayName:  "Grafana",
				RedirectURIs: []string{"https://grafana.example.com/login"},
			},
		}},
		Secrets: mockResolver(map[string]string{
			"ns/grafana-oidc[client-id]":     "grafana",
			"ns/grafana-oidc[client-secret]": "verysecret",
		}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := parseYAML(t, out.ConfigYAML)
	sc := m["staticClients"].([]any)[0].(map[string]any)
	assertNoKeys(t, sc, "allowedScopes")
}
