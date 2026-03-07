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

import (
	"context"
	"fmt"
	"testing"

	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
	"github.com/guided-traffic/dex-operator/internal/builder"
)

// mockResolver returns a SecretResolver backed by a flat key→value map.
// Key format: "namespace/secret-name[key]".
func mockResolver(secrets map[string]string) builder.SecretResolver {
	return func(ctx context.Context, namespace string, ref dexv1.SecretKeyRef) (string, error) {
		k := fmt.Sprintf("%s/%s[%s]", namespace, ref.Name, ref.Key)
		if v, ok := secrets[k]; ok {
			return v, nil
		}
		return "", fmt.Errorf("secret not found: %s", k)
	}
}

// parseYAML is a test helper that parses YAML bytes into a generic map.
func parseYAML(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if err := yaml.Unmarshal(data, &out); err != nil {
		t.Fatalf("invalid YAML output: %v\n%s", err, string(data))
	}
	return out
}

func minimalInstallation(namespace string) *dexv1.DexInstallation {
	return &dexv1.DexInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: namespace,
		},
		Spec: dexv1.DexInstallationSpec{
			Issuer:           "https://dex.example.com",
			ConfigSecretName: "dex-config",
			EnvSecretName:    "dex-env",
			Storage: dexv1.DexStorageSpec{
				Type: dexv1.StorageKubernetes,
			},
		},
	}
}

// ── Build: minimal ────────────────────────────────────────────────────────────

func TestBuild_Minimal(t *testing.T) {
	out, err := builder.Build(context.Background(), builder.Input{
		Installation: minimalInstallation("default"),
		Secrets:      mockResolver(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := parseYAML(t, out.ConfigYAML)

	if got := m["issuer"]; got != "https://dex.example.com" {
		t.Errorf("issuer = %v, want https://dex.example.com", got)
	}

	storage, _ := m["storage"].(map[string]any)
	if storage["type"] != "kubernetes" {
		t.Errorf("storage.type = %v, want kubernetes", storage["type"])
	}

	if len(out.EnvSecretData) != 0 {
		t.Errorf("expected empty env secret for minimal installation, got %v", out.EnvSecretData)
	}
}

// ── Build: Kubernetes storage ─────────────────────────────────────────────────

func TestBuild_KubernetesStorage(t *testing.T) {
	out, err := builder.Build(context.Background(), builder.Input{
		Installation: minimalInstallation("default"),
		Secrets:      mockResolver(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := parseYAML(t, out.ConfigYAML)
	storage := m["storage"].(map[string]any)
	if storage["type"] != "kubernetes" {
		t.Errorf("storage type = %v, want kubernetes", storage["type"])
	}
}

// ── Build: Postgres storage ───────────────────────────────────────────────────

func TestBuild_PostgresStorage(t *testing.T) {
	inst := minimalInstallation("ns")
	inst.Spec.Storage = dexv1.DexStorageSpec{
		Type: dexv1.StoragePostgres,
		Postgres: &dexv1.DexPostgresStorageSpec{
			Host:     "postgres:5432",
			Database: "dex",
			User:     "dex",
			PasswordRef: &dexv1.SecretKeyRef{
				Name: "pg-secret",
				Key:  "password",
			},
		},
	}

	secrets := map[string]string{
		"ns/pg-secret[password]": "supersecret",
	}

	out, err := builder.Build(context.Background(), builder.Input{
		Installation: inst,
		Secrets:      mockResolver(secrets),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := parseYAML(t, out.ConfigYAML)
	storage := m["storage"].(map[string]any)
	if storage["type"] != "postgres" {
		t.Fatalf("storage type = %v, want postgres", storage["type"])
	}
	cfg := storage["config"].(map[string]any)
	if cfg["host"] != "postgres:5432" {
		t.Errorf("storage.config.host = %v, want postgres:5432", cfg["host"])
	}
	// Password should be an env-var reference.
	if cfg["password"] != "$STORAGE_POSTGRES_PASSWORD" {
		t.Errorf("storage.config.password = %v, want $STORAGE_POSTGRES_PASSWORD", cfg["password"])
	}

	if string(out.EnvSecretData["STORAGE_POSTGRES_PASSWORD"]) != "supersecret" {
		t.Errorf("env[STORAGE_POSTGRES_PASSWORD] = %q, want supersecret",
			string(out.EnvSecretData["STORAGE_POSTGRES_PASSWORD"]))
	}
}

// ── Build: local connector → enablePasswordDB ─────────────────────────────────

func TestBuild_LocalConnector_EnablesPasswordDB(t *testing.T) {
	inst := minimalInstallation("default")

	connectors := builder.ConnectorSet{
		Local: []dexv1.DexLocalConnector{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "local", Namespace: "default"},
				Spec: dexv1.DexLocalConnectorSpec{
					InstallationRef: dexv1.InstallationRef{Name: "test", Namespace: "default"},
					Name:            "Local Users",
				},
			},
		},
	}

	out, err := builder.Build(context.Background(), builder.Input{
		Installation: inst,
		Connectors:   connectors,
		Secrets:      mockResolver(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := parseYAML(t, out.ConfigYAML)
	if m["enablePasswordDB"] != true {
		t.Errorf("enablePasswordDB = %v, want true", m["enablePasswordDB"])
	}
	// Local connectors do NOT produce a connector list entry.
	if conns := m["connectors"]; conns != nil {
		t.Errorf("expected no connectors list for local-only setup, got %v", conns)
	}
}

// ── Build: LDAP connector ─────────────────────────────────────────────────────

func TestBuild_LDAPConnector(t *testing.T) {
	inst := minimalInstallation("ns")

	connectors := builder.ConnectorSet{
		LDAP: []dexv1.DexLDAPConnector{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "my-ldap", Namespace: "ns"},
				Spec: dexv1.DexLDAPConnectorSpec{
					InstallationRef: dexv1.InstallationRef{Name: "test", Namespace: "ns"},
					Name:            "Corp LDAP",
					Host:            "ldap.corp.example.com:636",
					BindDN:          "cn=admin,dc=example,dc=com",
					BindPWRef: &dexv1.SecretKeyRef{
						Name: "ldap-creds",
						Key:  "bind-password",
					},
					UserSearch: dexv1.LDAPUserSearch{
						BaseDN:    "ou=users,dc=example,dc=com",
						Username:  "uid",
						IDAttr:    "DN",
						EmailAttr: "mail",
					},
				},
			},
		},
	}

	secrets := map[string]string{
		"ns/ldap-creds[bind-password]": "s3cret",
	}

	out, err := builder.Build(context.Background(), builder.Input{
		Installation: inst,
		Connectors:   connectors,
		Secrets:      mockResolver(secrets),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := parseYAML(t, out.ConfigYAML)
	conns := m["connectors"].([]any)
	if len(conns) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(conns))
	}
	conn := conns[0].(map[string]any)
	if conn["type"] != "ldap" {
		t.Errorf("connector type = %v, want ldap", conn["type"])
	}
	if conn["id"] != "my-ldap" {
		t.Errorf("connector id = %v, want my-ldap (fallback to metadata.name)", conn["id"])
	}

	cfg := conn["config"].(map[string]any)
	if cfg["host"] != "ldap.corp.example.com:636" {
		t.Errorf("connector host = %v", cfg["host"])
	}
	if cfg["bindDN"] != "cn=admin,dc=example,dc=com" {
		t.Errorf("connector bindDN = %v", cfg["bindDN"])
	}
	// bindPW should be an env var reference
	if cfg["bindPW"] != "$LDAP_MY_LDAP_BIND_PW" {
		t.Errorf("connector bindPW = %v, want $LDAP_MY_LDAP_BIND_PW", cfg["bindPW"])
	}

	if string(out.EnvSecretData["LDAP_MY_LDAP_BIND_PW"]) != "s3cret" {
		t.Errorf("env LDAP_MY_LDAP_BIND_PW = %q, want s3cret", string(out.EnvSecretData["LDAP_MY_LDAP_BIND_PW"]))
	}
}

// ── Build: OIDC connector ─────────────────────────────────────────────────────

func TestBuild_OIDCConnector(t *testing.T) {
	inst := minimalInstallation("ns")

	connectors := builder.ConnectorSet{
		OIDC: []dexv1.DexOIDCConnector{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "okta", Namespace: "ns"},
				Spec: dexv1.DexOIDCConnectorSpec{
					InstallationRef: dexv1.InstallationRef{Name: "test", Namespace: "ns"},
					Name:            "Okta",
					Issuer:          "https://example.okta.com",
					ClientIDRef: dexv1.SecretKeyRef{
						Name: "okta-creds",
						Key:  "client-id",
					},
					ClientSecretRef: dexv1.SecretKeyRef{
						Name: "okta-creds",
						Key:  "client-secret",
					},
					Scopes: []string{"openid", "email", "groups"},
				},
			},
		},
	}

	secrets := map[string]string{
		"ns/okta-creds[client-id]":     "okta-client-id-value",
		"ns/okta-creds[client-secret]": "okta-secret-value",
	}

	out, err := builder.Build(context.Background(), builder.Input{
		Installation: inst,
		Connectors:   connectors,
		Secrets:      mockResolver(secrets),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := parseYAML(t, out.ConfigYAML)
	conns := m["connectors"].([]any)
	conn := conns[0].(map[string]any)
	cfg := conn["config"].(map[string]any)

	if cfg["issuer"] != "https://example.okta.com" {
		t.Errorf("oidc issuer = %v", cfg["issuer"])
	}
	if cfg["clientID"] != "okta-client-id-value" {
		t.Errorf("oidc clientID = %v, want okta-client-id-value", cfg["clientID"])
	}
	if cfg["clientSecret"] != "$OIDC_OKTA_CLIENT_SECRET" {
		t.Errorf("oidc clientSecret = %v, want $OIDC_OKTA_CLIENT_SECRET", cfg["clientSecret"])
	}

	if string(out.EnvSecretData["OIDC_OKTA_CLIENT_SECRET"]) != "okta-secret-value" {
		t.Errorf("env OIDC_OKTA_CLIENT_SECRET = %q", string(out.EnvSecretData["OIDC_OKTA_CLIENT_SECRET"]))
	}
}

// ── Build: GitHub connector ───────────────────────────────────────────────────

func TestBuild_GitHubConnector(t *testing.T) {
	inst := minimalInstallation("ns")

	connectors := builder.ConnectorSet{
		GitHub: []dexv1.DexGitHubConnector{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "github", Namespace: "ns"},
				Spec: dexv1.DexGitHubConnectorSpec{
					InstallationRef: dexv1.InstallationRef{Name: "test", Namespace: "ns"},
					Name:            "GitHub",
					ClientIDRef:     dexv1.SecretKeyRef{Name: "gh-creds", Key: "client-id"},
					ClientSecretRef: dexv1.SecretKeyRef{Name: "gh-creds", Key: "client-secret"},
					Orgs: []dexv1.GitHubOrg{
						{Name: "myorg", Teams: []string{"platform"}},
					},
				},
			},
		},
	}

	secrets := map[string]string{
		"ns/gh-creds[client-id]":     "gh-id",
		"ns/gh-creds[client-secret]": "gh-secret",
	}

	out, err := builder.Build(context.Background(), builder.Input{
		Installation: inst,
		Connectors:   connectors,
		Secrets:      mockResolver(secrets),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := parseYAML(t, out.ConfigYAML)
	conn := m["connectors"].([]any)[0].(map[string]any)
	cfg := conn["config"].(map[string]any)

	if cfg["clientID"] != "gh-id" {
		t.Errorf("github clientID = %v, want gh-id", cfg["clientID"])
	}
	orgs := cfg["orgs"].([]any)
	if len(orgs) != 1 {
		t.Fatalf("expected 1 org, got %d", len(orgs))
	}
	org := orgs[0].(map[string]any)
	if org["name"] != "myorg" {
		t.Errorf("org name = %v, want myorg", org["name"])
	}
}

// ── Build: static client ──────────────────────────────────────────────────────

func TestBuild_StaticClient(t *testing.T) {
	inst := minimalInstallation("ns")

	clients := []dexv1.DexStaticClient{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "grafana", Namespace: "ns"},
			Spec: dexv1.DexStaticClientSpec{
				InstallationRef: dexv1.InstallationRef{Name: "test", Namespace: "ns"},
				SecretRef: dexv1.StaticClientSecretRef{
					Name:            "grafana-oidc",
					ClientIDKey:     "client-id",
					ClientSecretKey: "client-secret",
				},
				Name:         "Grafana",
				RedirectURIs: []string{"https://grafana.example.com/login/generic_oauth"},
			},
		},
	}

	secrets := map[string]string{
		"ns/grafana-oidc[client-id]":     "grafana",
		"ns/grafana-oidc[client-secret]": "verysecret",
	}

	out, err := builder.Build(context.Background(), builder.Input{
		Installation:  inst,
		StaticClients: clients,
		Secrets:       mockResolver(secrets),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := parseYAML(t, out.ConfigYAML)
	sc := m["staticClients"].([]any)[0].(map[string]any)

	if sc["id"] != "grafana" {
		t.Errorf("staticClient id = %v, want grafana", sc["id"])
	}
	if sc["secret"] != "$GRAFANA_CLIENT_SECRET" {
		t.Errorf("staticClient secret = %v, want $GRAFANA_CLIENT_SECRET", sc["secret"])
	}
	if sc["name"] != "Grafana" {
		t.Errorf("staticClient name = %v, want Grafana", sc["name"])
	}

	if string(out.EnvSecretData["GRAFANA_CLIENT_SECRET"]) != "verysecret" {
		t.Errorf("env GRAFANA_CLIENT_SECRET = %q, want verysecret",
			string(out.EnvSecretData["GRAFANA_CLIENT_SECRET"]))
	}
}

// ── Build: SAML connector with CA mount ───────────────────────────────────────

func TestBuild_SAMLConnector_CACertMounted(t *testing.T) {
	inst := minimalInstallation("ns")

	connectors := builder.ConnectorSet{
		SAML: []dexv1.DexSAMLConnector{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "saml-idp", Namespace: "ns"},
				Spec: dexv1.DexSAMLConnectorSpec{
					InstallationRef: dexv1.InstallationRef{Name: "test", Namespace: "ns"},
					Name:            "SAML IdP",
					SSOURL:          "https://idp.example.com/sso",
					CARef: &dexv1.SecretKeyRef{
						Name: "saml-ca",
						Key:  "ca.crt",
					},
				},
			},
		},
	}

	out, err := builder.Build(context.Background(), builder.Input{
		Installation: inst,
		Connectors:   connectors,
		Secrets:      mockResolver(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(out.MountedSecrets) != 1 {
		t.Fatalf("expected 1 mounted secret, got %d", len(out.MountedSecrets))
	}
	ms := out.MountedSecrets[0]
	if ms.SecretName != "saml-ca" || ms.SecretKey != "ca.crt" {
		t.Errorf("mounted secret = %+v, want saml-ca/ca.crt", ms)
	}
	if ms.MountPath != "/etc/dex/certs/saml-idp-ca.pem" {
		t.Errorf("mount path = %v, want /etc/dex/certs/saml-idp-ca.pem", ms.MountPath)
	}
}

// ── Build: full installation options ─────────────────────────────────────────

func TestBuild_FullInstallationOptions(t *testing.T) {
	inst := minimalInstallation("ns")
	inst.Spec.Web = &dexv1.DexWebSpec{HTTP: "0.0.0.0:5556"}
	inst.Spec.Logger = &dexv1.DexLoggerSpec{Level: "debug", Format: "json"}
	inst.Spec.Expiry = &dexv1.DexExpirySpec{IDTokens: "24h", SigningKeys: "6h"}
	inst.Spec.OAuth2 = &dexv1.DexOAuth2ConfigSpec{SkipApprovalScreen: true}
	inst.Spec.GRPC = &dexv1.DexGRPCSpec{Addr: "0.0.0.0:5557"}
	inst.Spec.Frontend = &dexv1.DexFrontendSpec{Theme: "dark", Issuer: "My Corp"}

	out, err := builder.Build(context.Background(), builder.Input{
		Installation: inst,
		Secrets:      mockResolver(nil),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := parseYAML(t, out.ConfigYAML)

	web := m["web"].(map[string]any)
	if web["http"] != "0.0.0.0:5556" {
		t.Errorf("web.http = %v", web["http"])
	}

	logger := m["logger"].(map[string]any)
	if logger["level"] != "debug" || logger["format"] != "json" {
		t.Errorf("logger = %v", logger)
	}

	expiry := m["expiry"].(map[string]any)
	if expiry["idTokens"] != "24h" {
		t.Errorf("expiry.idTokens = %v", expiry["idTokens"])
	}

	oauth2 := m["oauth2"].(map[string]any)
	if oauth2["skipApprovalScreen"] != true {
		t.Errorf("oauth2.skipApprovalScreen = %v", oauth2["skipApprovalScreen"])
	}

	grpc := m["grpc"].(map[string]any)
	if grpc["addr"] != "0.0.0.0:5557" {
		t.Errorf("grpc.addr = %v", grpc["addr"])
	}

	frontend := m["frontend"].(map[string]any)
	if frontend["theme"] != "dark" {
		t.Errorf("frontend.theme = %v", frontend["theme"])
	}
	if frontend["issuer"] != "My Corp" {
		t.Errorf("frontend.issuer = %v", frontend["issuer"])
	}
}

// ── Build: secret resolution error ───────────────────────────────────────────

func TestBuild_SecretResolutionError(t *testing.T) {
	inst := minimalInstallation("ns")

	connectors := builder.ConnectorSet{
		OIDC: []dexv1.DexOIDCConnector{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "okta", Namespace: "ns"},
				Spec: dexv1.DexOIDCConnectorSpec{
					InstallationRef: dexv1.InstallationRef{Name: "test", Namespace: "ns"},
					Name:            "Okta",
					Issuer:          "https://example.okta.com",
					ClientIDRef:     dexv1.SecretKeyRef{Name: "missing", Key: "client-id"},
					ClientSecretRef: dexv1.SecretKeyRef{Name: "missing", Key: "client-secret"},
				},
			},
		},
	}

	_, err := builder.Build(context.Background(), builder.Input{
		Installation: inst,
		Connectors:   connectors,
		Secrets:      mockResolver(nil), // empty resolver → all secrets missing
	})

	if err == nil {
		t.Fatal("expected error for missing secret, got nil")
	}
}

// ── EnvVar helpers ────────────────────────────────────────────────────────────

func TestSanitizeEnvKey_Patterns(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"grafana", "GRAFANA"},
		{"my-connector", "MY_CONNECTOR"},
		{"github.com", "GITHUB_COM"},
		{"abc_123", "ABC_123"},
		{"ALREADY_UPPER", "ALREADY_UPPER"},
		{"with spaces", "WITH_SPACES"},
	}

	for _, tc := range cases {
		// Use the exported behaviour indirectly via clientEnvKey.
		// clientEnvKey("grafana", "CLIENT_SECRET") should give GRAFANA_CLIENT_SECRET
		got := builder.ExportedSanitizeEnvKey(tc.input)
		if got != tc.want {
			t.Errorf("sanitize(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestConnectorID_Fallback(t *testing.T) {
	if got := builder.ExportedConnectorID("meta-name", ""); got != "meta-name" {
		t.Errorf("connectorID fallback = %q, want meta-name", got)
	}
	if got := builder.ExportedConnectorID("meta-name", "spec-id"); got != "spec-id" {
		t.Errorf("connectorID spec = %q, want spec-id", got)
	}
}
