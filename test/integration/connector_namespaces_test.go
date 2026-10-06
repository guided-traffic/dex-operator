//go:build integration

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

package integration

import (
	"context"
	"strconv"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// TestIntegration_AllowedConnectorNamespacesValidation verifies the schema
// rules on spec.allowedConnectorNamespaces against the real API server.
//
// The objects are written as unstructured: the typed Go struct carries
// omitempty, so an empty list would never reach the API server and the
// MinItems rule could not be exercised.
func TestIntegration_AllowedConnectorNamespacesValidation(t *testing.T) {
	ns := "it-cns-validation"
	createNamespace(t, ns)

	tests := []struct {
		name    string
		list    []any
		wantMsg string // empty: must be accepted
	}{
		{name: "empty list rejected", list: []any{}, wantMsg: "should have at least 1 items"},
		{name: "wildcard with other entries rejected", list: []any{"*", "a"}, wantMsg: `"*" must be the only entry`},
		{name: "duplicate entries rejected", list: []any{"a", "a"}, wantMsg: "Duplicate value"},
		{name: "wildcard alone accepted", list: []any{"*"}},
		{name: "explicit list accepted", list: []any{"a", "b"}},
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			obj := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": dexv1.GroupVersion.String(),
				"kind":       "DexInstallation",
				"metadata": map[string]any{
					"name":      "cns-" + strconv.Itoa(i),
					"namespace": ns,
				},
				"spec": map[string]any{
					"issuer":                     "https://dex.example.com",
					"configSecretName":           "cfg-" + strconv.Itoa(i),
					"envSecretName":              "env-" + strconv.Itoa(i),
					"storage":                    map[string]any{"type": "memory"},
					"allowedConnectorNamespaces": tc.list,
				},
			}}

			err := k8sClient.Create(context.Background(), obj)
			if err == nil {
				t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), obj) })
			}

			if tc.wantMsg == "" {
				if err != nil {
					t.Fatalf("expected %v to be accepted, got %v", tc.list, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected %v to be rejected, but it was accepted", tc.list)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantMsg)
			}
		})
	}
}

// connectorReady returns the Ready condition of the OIDC connector, or nil.
func connectorReady(ns, name string) *metav1.Condition {
	var conn dexv1.DexOIDCConnector
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: name}, &conn); err != nil {
		return nil
	}
	return findCondition(conn.Status.Conditions, dexv1.ConditionTypeReady)
}

// installationCounts returns the connector and static client counts reported
// in the installation's status.
func installationCounts(ns, name string) (connectors, clients int, ok bool) {
	var inst dexv1.DexInstallation
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: name}, &inst); err != nil {
		return 0, 0, false
	}
	return inst.Status.ConnectorCount, inst.Status.StaticClientCount, true
}

// TestIntegration_ConnectorNamespaceSplit walks one tenant namespace through
// the connector allowlist:
//
//  1. allowedNamespaces [tenant], allowedConnectorNamespaces omitted: the
//     static client is rendered, the connector next to it is not and reports
//     Ready=False with the "omitted" hint.
//  2. allowedConnectorNamespaces [tenant]: the connector is rendered and turns
//     Ready=True without any event on the connector itself.
//  3. allowedConnectorNamespaces [inst]: the connector is dropped and turns
//     Ready=False again. Before the child reconcilers watched
//     DexInstallation, it kept a stale Ready=True here.
func TestIntegration_ConnectorNamespaceSplit(t *testing.T) {
	nsInst := "it-cns-inst"
	nsTenant := "it-cns-tenant"
	for _, ns := range []string{nsInst, nsTenant} {
		createNamespace(t, ns)
	}
	inst := createInstallation(t, nsInst, "dex", []string{nsTenant})
	ref := dexv1.InstallationRef{Name: inst.Name, Namespace: nsInst}

	sc := &dexv1.DexStaticClient{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-app", Namespace: nsTenant},
		Spec: dexv1.DexStaticClientSpec{
			InstallationRef: ref,
			DisplayName:     "Tenant App",
			ClientID:        "tenant-app-id",
			Public:          true,
		},
	}
	if err := k8sClient.Create(context.Background(), sc); err != nil {
		t.Fatalf("create static client: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), sc) })

	createSecret(t, nsTenant, "tenant-idp", map[string][]byte{
		"client-id":     []byte("tenant-idp-id"),
		"client-secret": []byte("tenant-idp-secret"),
	})
	conn := &dexv1.DexOIDCConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "tenant-idp", Namespace: nsTenant},
		Spec: dexv1.DexOIDCConnectorSpec{
			InstallationRef: ref,
			DisplayName:     "Tenant IdP",
			Issuer:          "https://tenant-idp.example.com",
			ClientIDRef:     dexv1.SecretKeyRef{Name: "tenant-idp", Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "tenant-idp", Key: "client-secret"},
		},
	}
	if err := k8sClient.Create(context.Background(), conn); err != nil {
		t.Fatalf("create connector: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), conn) })

	configContains := func(sub string) bool {
		s := getSecret(nsInst, inst.Spec.ConfigSecretName)
		return s != nil && strings.Contains(string(s.Data["config.yaml"]), sub)
	}

	// Step 1: client admitted, connector not.
	eventually(t, func() bool {
		connectors, clients, ok := installationCounts(nsInst, inst.Name)
		return ok && clients == 1 && connectors == 0 && configContains("tenant-app-id")
	}, "static client should be rendered, connector not")
	if configContains("tenant-idp.example.com") {
		t.Error("connector from a namespace admitted only via allowedNamespaces was rendered")
	}
	wantHint := `allowedConnectorNamespaces (omitted: only "` + nsInst + `" is allowed)`
	eventually(t, func() bool {
		cond := connectorReady(nsTenant, conn.Name)
		return cond != nil && cond.Status == metav1.ConditionFalse && strings.Contains(cond.Message, wantHint)
	}, "connector should be Ready=False with the omitted hint")

	// Step 2: admit the tenant for connectors.
	mutateResource(t, inst, func(i *dexv1.DexInstallation) {
		i.Spec.AllowedConnectorNamespaces = []string{nsTenant}
	})
	eventually(t, func() bool {
		connectors, _, ok := installationCounts(nsInst, inst.Name)
		return ok && connectors == 1 && configContains("tenant-idp.example.com")
	}, "connector should be rendered once its namespace is admitted")
	eventually(t, func() bool {
		cond := connectorReady(nsTenant, conn.Name)
		return cond != nil && cond.Status == metav1.ConditionTrue
	}, "connector Ready should follow the installation to True")

	// Step 3: revoke the tenant again.
	mutateResource(t, inst, func(i *dexv1.DexInstallation) {
		i.Spec.AllowedConnectorNamespaces = []string{nsInst}
	})
	eventually(t, func() bool {
		connectors, clients, ok := installationCounts(nsInst, inst.Name)
		return ok && connectors == 0 && clients == 1 && !configContains("tenant-idp.example.com")
	}, "connector should be dropped from the config after revocation")
	eventually(t, func() bool {
		cond := connectorReady(nsTenant, conn.Name)
		return cond != nil && cond.Status == metav1.ConditionFalse &&
			strings.Contains(cond.Message, "allowedConnectorNamespaces") &&
			!strings.Contains(cond.Message, "omitted")
	}, "connector Ready should follow the installation back to False (stale-status regression)")
}
