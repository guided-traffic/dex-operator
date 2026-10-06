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
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
	"github.com/guided-traffic/dex-operator/internal/controller"
)

func TestIsNamespaceAllowed(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		allowed   []string
		want      bool
	}{
		{
			name:      "empty allowedNamespaces denies everything",
			namespace: "default",
			allowed:   nil,
			want:      false,
		},
		{
			name:      "wildcard allows all namespaces",
			namespace: "some-ns",
			allowed:   []string{"*"},
			want:      true,
		},
		{
			name:      "exact match allows specific namespace",
			namespace: "dex",
			allowed:   []string{"dex", "monitoring"},
			want:      true,
		},
		{
			name:      "non-matching namespace is denied",
			namespace: "other-ns",
			allowed:   []string{"dex", "monitoring"},
			want:      false,
		},
		{
			name:      "single allowed namespace matches",
			namespace: "prod",
			allowed:   []string{"prod"},
			want:      true,
		},
		{
			name:      "wildcard in list allows any",
			namespace: "random",
			allowed:   []string{"specific", "*"},
			want:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := controller.IsNamespaceAllowed(tc.namespace, tc.allowed)
			if got != tc.want {
				t.Errorf("IsNamespaceAllowed(%q, %v) = %v; want %v",
					tc.namespace, tc.allowed, got, tc.want)
			}
		})
	}
}

// installationWithAllowlists returns a DexInstallation "dex/main" with the
// given static-client and connector allowlists.
func installationWithAllowlists(clientNS, connectorNS []string) *dexv1.DexInstallation {
	return &dexv1.DexInstallation{
		ObjectMeta: metav1.ObjectMeta{Name: "main", Namespace: "dex"},
		Spec: dexv1.DexInstallationSpec{
			AllowedNamespaces:          clientNS,
			AllowedConnectorNamespaces: connectorNS,
		},
	}
}

// TestConnectorNamespaces verifies the effective connector allowlist,
// evaluated through isNamespaceAllowed exactly like the build-time filter
// and the child reconciler do.
func TestConnectorNamespaces(t *testing.T) {
	tests := []struct {
		name        string
		connectorNS []string
		admitted    []string
		denied      []string
	}{
		{
			name:     "omitted admits only the installation namespace",
			admitted: []string{"dex"},
			denied:   []string{"tenant-a", "platform-idp", "default"},
		},
		{
			name:        "empty list is treated like omitted",
			connectorNS: []string{},
			admitted:    []string{"dex"},
			denied:      []string{"tenant-a"},
		},
		{
			name:        "wildcard admits all",
			connectorNS: []string{"*"},
			admitted:    []string{"dex", "tenant-a", "platform-idp"},
		},
		{
			name:        "explicit list admits exactly its entries",
			connectorNS: []string{"dex", "platform-idp"},
			admitted:    []string{"dex", "platform-idp"},
			denied:      []string{"tenant-a"},
		},
		{
			name:        "list without the own namespace denies the own namespace",
			connectorNS: []string{"platform-idp"},
			admitted:    []string{"platform-idp"},
			denied:      []string{"dex", "tenant-a"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// allowedNamespaces admits everything, so any denial below can
			// only come from the connector allowlist.
			inst := installationWithAllowlists([]string{"*"}, tc.connectorNS)
			allowed := controller.ConnectorNamespaces(inst)
			for _, ns := range tc.admitted {
				if !controller.IsNamespaceAllowed(ns, allowed) {
					t.Errorf("namespace %q denied, want admitted (effective list %v)", ns, allowed)
				}
			}
			for _, ns := range tc.denied {
				if controller.IsNamespaceAllowed(ns, allowed) {
					t.Errorf("namespace %q admitted, want denied (effective list %v)", ns, allowed)
				}
			}
		})
	}
}

// TestCheckChildNamespace verifies that each child kind is checked against
// its own allowlist and that the Ready=False message names the right field.
func TestCheckChildNamespace(t *testing.T) {
	inTenant := metav1.ObjectMeta{Name: "x", Namespace: "tenant-a"}
	inDex := metav1.ObjectMeta{Name: "x", Namespace: "dex"}

	tests := []struct {
		name        string
		child       client.Object
		clientNS    []string
		connectorNS []string
		wantMsg     string // empty: admitted
		notWantMsg  string
	}{
		{
			name:     "static client admitted by allowedNamespaces",
			child:    &dexv1.DexStaticClient{ObjectMeta: inTenant},
			clientNS: []string{"tenant-a"},
		},
		{
			name:        "static client ignores allowedConnectorNamespaces",
			child:       &dexv1.DexStaticClient{ObjectMeta: inTenant},
			clientNS:    []string{"dex"},
			connectorNS: []string{"tenant-a"},
			wantMsg:     `namespace "tenant-a" is not in DexInstallation dex/main allowedNamespaces`,
		},
		{
			name:    "static client in own namespace still needs allowedNamespaces",
			child:   &dexv1.DexStaticClient{ObjectMeta: inDex},
			wantMsg: `namespace "dex" is not in DexInstallation dex/main allowedNamespaces`,
		},
		{
			name:  "connector in own namespace admitted by default",
			child: &dexv1.DexOIDCConnector{ObjectMeta: inDex},
		},
		{
			// Upgrade corner case: allowedNamespaces used to exclude the
			// own namespace for connectors too; the new default admits it.
			name:     "connector in own namespace admitted although allowedNamespaces excludes it",
			child:    &dexv1.DexOIDCConnector{ObjectMeta: inDex},
			clientNS: []string{"tenant-a"},
		},
		{
			name:     "connector no longer admitted by allowedNamespaces",
			child:    &dexv1.DexOIDCConnector{ObjectMeta: inTenant},
			clientNS: []string{"tenant-a"},
			wantMsg: `namespace "tenant-a" is not in DexInstallation dex/main allowedConnectorNamespaces` +
				` (omitted: only "dex" is allowed)`,
		},
		{
			name:        "connector admitted by allowedConnectorNamespaces",
			child:       &dexv1.DexLocalConnector{ObjectMeta: inTenant},
			connectorNS: []string{"tenant-a"},
		},
		{
			name:        "connector denied by explicit list carries no omitted hint",
			child:       &dexv1.DexAuthProxyConnector{ObjectMeta: inTenant},
			clientNS:    []string{"*"},
			connectorNS: []string{"platform-idp"},
			wantMsg:     `namespace "tenant-a" is not in DexInstallation dex/main allowedConnectorNamespaces`,
			notWantMsg:  "omitted",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inst := installationWithAllowlists(tc.clientNS, tc.connectorNS)
			err := controller.CheckChildNamespace(tc.child, inst)

			if tc.wantMsg == "" {
				if err != nil {
					t.Fatalf("want admitted, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("want denied with %q, got nil", tc.wantMsg)
			}
			if !controller.IsConfigError(err) {
				t.Errorf("want a configError (long requeue, no back-off), got %T", err)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("message %q does not contain %q", err.Error(), tc.wantMsg)
			}
			if tc.notWantMsg != "" && strings.Contains(err.Error(), tc.notWantMsg) {
				t.Errorf("message %q must not contain %q", err.Error(), tc.notWantMsg)
			}
		})
	}
}
