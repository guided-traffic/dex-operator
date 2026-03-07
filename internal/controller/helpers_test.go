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
