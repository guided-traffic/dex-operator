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

package controller_test

import (
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
	"github.com/guided-traffic/dex-operator/internal/builder"
	"github.com/guided-traffic/dex-operator/internal/controller"
)

// ── secretDataEqual ───────────────────────────────────────────────────────────

func TestSecretDataEqual(t *testing.T) {
	tests := []struct {
		name string
		a, b map[string][]byte
		want bool
	}{
		{"both nil", nil, nil, true},
		{"both empty", map[string][]byte{}, map[string][]byte{}, true},
		{"equal single key", map[string][]byte{"k": []byte("v")}, map[string][]byte{"k": []byte("v")}, true},
		{"different values", map[string][]byte{"k": []byte("a")}, map[string][]byte{"k": []byte("b")}, false},
		{"different key counts", map[string][]byte{"k1": []byte("v")}, map[string][]byte{"k1": []byte("v"), "k2": []byte("v")}, false},
		{"nil vs empty", nil, map[string][]byte{}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := controller.SecretDataEqual(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("SecretDataEqual() = %v; want %v", got, tc.want)
			}
		})
	}
}

// ── yamlSecretDataEqual ───────────────────────────────────────────────────────

func TestYamlSecretDataEqual(t *testing.T) {
	tests := []struct {
		name string
		a, b map[string][]byte
		want bool
	}{
		{"both nil", nil, nil, true},
		{"both empty", map[string][]byte{}, map[string][]byte{}, true},
		{"identical YAML", map[string][]byte{"c": []byte("a: 1\nb: 2\n")}, map[string][]byte{"c": []byte("a: 1\nb: 2\n")}, true},
		{
			"different map key order same content",
			map[string][]byte{"config.yaml": []byte("storage:\n  type: kubernetes\n  config:\n    inCluster: true\nissuer: https://dex.example.com\n")},
			map[string][]byte{"config.yaml": []byte("issuer: https://dex.example.com\nstorage:\n  config:\n    inCluster: true\n  type: kubernetes\n")},
			true,
		},
		{
			"actually different YAML values",
			map[string][]byte{"config.yaml": []byte("issuer: https://old.example.com\n")},
			map[string][]byte{"config.yaml": []byte("issuer: https://new.example.com\n")},
			false,
		},
		{"different key counts", map[string][]byte{"a": []byte("v")}, map[string][]byte{"a": []byte("v"), "b": []byte("v")}, false},
		{"missing key in b", map[string][]byte{"a": []byte("v"), "b": []byte("v")}, map[string][]byte{"a": []byte("v")}, false},
		{
			"non-YAML binary data equal",
			map[string][]byte{"k": {0x00, 0x01, 0x02}},
			map[string][]byte{"k": {0x00, 0x01, 0x02}},
			true,
		},
		{
			"non-YAML binary data different",
			map[string][]byte{"k": {0x00, 0x01}},
			map[string][]byte{"k": {0x00, 0x02}},
			false,
		},
		{
			"nested maps different order",
			map[string][]byte{"c": []byte("connectors:\n- type: oidc\n  config:\n    clientID: foo\n    issuer: bar\n")},
			map[string][]byte{"c": []byte("connectors:\n- type: oidc\n  config:\n    issuer: bar\n    clientID: foo\n")},
			true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := controller.YamlSecretDataEqual(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("YamlSecretDataEqual() = %v; want %v", got, tc.want)
			}
		})
	}
}

// TestYamlSecretDataEqual_PhantomRestart verifies the core issue: two
// yaml.Marshal outputs of the same config with map[string]any fields
// producing different byte orderings are still considered equal.
func TestYamlSecretDataEqual_PhantomRestart(t *testing.T) {
	// Simulate two marshals of the same config with different map ordering
	marshalA := []byte(`issuer: https://dex.example.com
storage:
  type: postgres
  config:
    host: db.example.com
    database: dex
    user: dex
    password: $STORAGE_POSTGRES_PASSWORD
connectors:
- type: oidc
  id: okta
  name: Okta
  config:
    clientID: my-id
    clientSecret: $OIDC_OKTA_CLIENT_SECRET
    issuer: https://okta.example.com
`)

	marshalB := []byte(`issuer: https://dex.example.com
storage:
  config:
    database: dex
    host: db.example.com
    password: $STORAGE_POSTGRES_PASSWORD
    user: dex
  type: postgres
connectors:
- type: oidc
  id: okta
  name: Okta
  config:
    issuer: https://okta.example.com
    clientID: my-id
    clientSecret: $OIDC_OKTA_CLIENT_SECRET
`)

	a := map[string][]byte{"config.yaml": marshalA}
	b := map[string][]byte{"config.yaml": marshalB}

	if !controller.YamlSecretDataEqual(a, b) {
		t.Error("YamlSecretDataEqual should treat reordered YAML as equal (phantom restart prevention)")
	}

	// Unchanged byte comparison would detect a false diff
	if controller.SecretDataEqual(a, b) {
		t.Error("SecretDataEqual should detect the byte-level difference (this IS the bug)")
	}
}

// ── rolloutEnabled ────────────────────────────────────────────────────────────

func TestRolloutEnabled(t *testing.T) {
	tests := []struct {
		name    string
		rollout *dexv1.RolloutRestartSpec
		want    bool
	}{
		{"nil spec", nil, false},
		{"disabled", &dexv1.RolloutRestartSpec{Enabled: false, DeploymentName: "dex"}, false},
		{"enabled no name", &dexv1.RolloutRestartSpec{Enabled: true, DeploymentName: ""}, false},
		{"enabled with name", &dexv1.RolloutRestartSpec{Enabled: true, DeploymentName: "dex"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inst := &dexv1.DexInstallation{
				Spec: dexv1.DexInstallationSpec{
					RolloutRestart: tc.rollout,
				},
			}
			got := controller.RolloutEnabled(inst)
			if got != tc.want {
				t.Errorf("RolloutEnabled() = %v; want %v", got, tc.want)
			}
		})
	}
}

// ── isConfigError ─────────────────────────────────────────────────────────────

func TestIsConfigError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain error", fmt.Errorf("something broke"), false},
		{"config error", controller.NewConfigError("bad ref"), true},
		{"wrapped config error", fmt.Errorf("outer: %w", controller.NewConfigError("bad ref")), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := controller.IsConfigError(tc.err)
			if got != tc.want {
				t.Errorf("IsConfigError() = %v; want %v", got, tc.want)
			}
		})
	}
}

// ── countConnectors ───────────────────────────────────────────────────────────

func TestCountConnectors(t *testing.T) {
	tests := []struct {
		name string
		cs   builder.ConnectorSet
		want int
	}{
		{"empty", builder.ConnectorSet{}, 0},
		{
			"single LDAP",
			builder.ConnectorSet{
				LDAP: []dexv1.DexLDAPConnector{
					{ObjectMeta: metav1.ObjectMeta{Name: "ldap-1"}},
				},
			},
			1,
		},
		{
			"mixed connectors",
			builder.ConnectorSet{
				LDAP:   []dexv1.DexLDAPConnector{{}, {}},
				GitHub: []dexv1.DexGitHubConnector{{}},
				OIDC:   []dexv1.DexOIDCConnector{{}, {}, {}},
			},
			6,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := controller.CountConnectors(tc.cs)
			if got != tc.want {
				t.Errorf("CountConnectors() = %d; want %d", got, tc.want)
			}
		})
	}
}

// ── GetReferencedSecretNames ──────────────────────────────────────────────────

func TestGetReferencedSecretNames_OIDCConnector(t *testing.T) {
	c := &dexv1.DexOIDCConnector{
		Spec: dexv1.DexOIDCConnectorSpec{
			ClientIDRef:     dexv1.SecretKeyRef{Name: "creds", Key: "id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "creds", Key: "secret"},
			RootCARef:       &dexv1.SecretKeyRef{Name: "tls-ca", Key: "ca.pem"},
		},
	}
	names := c.GetReferencedSecretNames()

	// "creds" appears in both clientIDRef and clientSecretRef but should be deduplicated
	if len(names) != 2 {
		t.Fatalf("expected 2 unique secret names, got %d: %v", len(names), names)
	}
	wantSet := map[string]bool{"creds": true, "tls-ca": true}
	for _, n := range names {
		if !wantSet[n] {
			t.Errorf("unexpected secret name: %q", n)
		}
	}
}

func TestGetReferencedSecretNames_StaticClient(t *testing.T) {
	sc := &dexv1.DexStaticClient{
		Spec: dexv1.DexStaticClientSpec{
			SecretRef: dexv1.StaticClientSecretRef{Name: "my-client-creds"},
		},
	}
	names := sc.GetReferencedSecretNames()
	if len(names) != 1 || names[0] != "my-client-creds" {
		t.Errorf("expected [my-client-creds], got %v", names)
	}
}

func TestGetReferencedSecretNames_LDAPConnector(t *testing.T) {
	c := &dexv1.DexLDAPConnector{
		Spec: dexv1.DexLDAPConnectorSpec{
			BindPWRef:     &dexv1.SecretKeyRef{Name: "ldap-bind", Key: "pw"},
			RootCARef:     &dexv1.SecretKeyRef{Name: "ldap-ca", Key: "ca"},
			ClientCertRef: &dexv1.SecretKeyRef{Name: "ldap-tls", Key: "cert"},
			ClientKeyRef:  &dexv1.SecretKeyRef{Name: "ldap-tls", Key: "key"}, // same secret as cert
		},
	}
	names := c.GetReferencedSecretNames()

	// "ldap-tls" appears twice but should be deduplicated: expect 3 unique names
	if len(names) != 3 {
		t.Fatalf("expected 3 unique secret names, got %d: %v", len(names), names)
	}
}

func TestGetReferencedSecretNames_NoSecrets(t *testing.T) {
	authProxy := &dexv1.DexAuthProxyConnector{}
	if names := authProxy.GetReferencedSecretNames(); len(names) != 0 {
		t.Errorf("AuthProxy should return no secrets, got %v", names)
	}

	local := &dexv1.DexLocalConnector{}
	if names := local.GetReferencedSecretNames(); len(names) != 0 {
		t.Errorf("Local should return no secrets, got %v", names)
	}
}
