//go:build e2e

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

package e2e

import (
	"context"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// TestE2E_MinimalInstallation creates a DexInstallation in a fresh namespace
// and asserts that the operator creates both secrets with valid content.
func TestE2E_MinimalInstallation(t *testing.T) {
	ns := "e2e-minimal"
	e2eCreateNamespace(t, ns)

	inst := &dexv1.DexInstallation{
		ObjectMeta: metav1.ObjectMeta{Name: "dex", Namespace: ns},
		Spec: dexv1.DexInstallationSpec{
			Issuer:            "https://dex.e2e.example.com",
			ConfigSecretName:  "dex-config",
			EnvSecretName:     "dex-env",
			AllowedNamespaces: []string{"*"},
			Storage: dexv1.DexStorageSpec{
				Type: dexv1.StorageKubernetes,
			},
		},
	}
	if err := e2eClient.Create(context.Background(), inst); err != nil {
		t.Fatalf("create DexInstallation: %v", err)
	}
	t.Cleanup(func() { _ = e2eClient.Delete(context.Background(), inst) })

	// Both secrets must be created within the timeout.
	e2eEventually(t, func() bool {
		return e2eGetSecret(ns, "dex-config") != nil
	}, "config secret not created")

	e2eEventually(t, func() bool {
		return e2eGetSecret(ns, "dex-env") != nil
	}, "env secret not created")

	// config.yaml must contain the issuer.
	e2eEventually(t, func() bool {
		s := e2eGetSecret(ns, "dex-config")
		return s != nil && strings.Contains(string(s.Data["config.yaml"]), "dex.e2e.example.com")
	}, "issuer not in config.yaml")

	// DexInstallation status must be Ready=True.
	e2eEventually(t, func() bool {
		var updated dexv1.DexInstallation
		if err := e2eClient.Get(context.Background(),
			client.ObjectKey{Namespace: ns, Name: "dex"}, &updated); err != nil {
			return false
		}
		cond := e2eFindCondition(updated.Status.Conditions, dexv1.ConditionTypeReady)
		return cond != nil && cond.Status == metav1.ConditionTrue
	}, "Ready condition not True")
}

// TestE2E_OIDCConnector creates a DexInstallation and a DexOIDCConnector, then
// asserts that the connector appears in the rendered config.yaml.
func TestE2E_OIDCConnector(t *testing.T) {
	ns := "e2e-oidc"
	e2eCreateNamespace(t, ns)

	inst := &dexv1.DexInstallation{
		ObjectMeta: metav1.ObjectMeta{Name: "dex", Namespace: ns},
		Spec: dexv1.DexInstallationSpec{
			Issuer:            "https://dex.e2e-oidc.example.com",
			ConfigSecretName:  "dex-config",
			EnvSecretName:     "dex-env",
			AllowedNamespaces: []string{"*"},
			Storage:           dexv1.DexStorageSpec{Type: dexv1.StorageKubernetes},
		},
	}
	if err := e2eClient.Create(context.Background(), inst); err != nil {
		t.Fatalf("create installation: %v", err)
	}
	t.Cleanup(func() { _ = e2eClient.Delete(context.Background(), inst) })

	e2eCreateSecret(t, ns, "oidc-creds", map[string][]byte{
		"client-id":     []byte("e2e-client-id"),
		"client-secret": []byte("e2e-client-secret"),
	})

	conn := &dexv1.DexOIDCConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "okta", Namespace: ns},
		Spec: dexv1.DexOIDCConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: "dex", Namespace: ns},
			DisplayName:     "Okta",
			Issuer:          "https://e2e-okta.example.com",
			ClientIDRef:     dexv1.SecretKeyRef{Name: "oidc-creds", Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "oidc-creds", Key: "client-secret"},
		},
	}
	if err := e2eClient.Create(context.Background(), conn); err != nil {
		t.Fatalf("create OIDC connector: %v", err)
	}
	t.Cleanup(func() { _ = e2eClient.Delete(context.Background(), conn) })

	// Connector must appear in config.yaml.
	e2eEventually(t, func() bool {
		s := e2eGetSecret(ns, "dex-config")
		return s != nil && strings.Contains(string(s.Data["config.yaml"]), "oidc")
	}, "OIDC connector not in config.yaml")

	// ConnectorCount must be updated.
	e2eEventually(t, func() bool {
		var updated dexv1.DexInstallation
		if err := e2eClient.Get(context.Background(),
			client.ObjectKey{Namespace: ns, Name: "dex"}, &updated); err != nil {
			return false
		}
		return updated.Status.ConnectorCount >= 1
	}, "ConnectorCount should be >= 1")
}

// TestE2E_StaticClient creates a DexInstallation and DexStaticClient, then
// asserts that the redirect URI appears in config.yaml.
func TestE2E_StaticClient(t *testing.T) {
	ns := "e2e-sc"
	e2eCreateNamespace(t, ns)

	inst := &dexv1.DexInstallation{
		ObjectMeta: metav1.ObjectMeta{Name: "dex", Namespace: ns},
		Spec: dexv1.DexInstallationSpec{
			Issuer:            "https://dex.e2e-sc.example.com",
			ConfigSecretName:  "dex-config",
			EnvSecretName:     "dex-env",
			AllowedNamespaces: []string{"*"},
			Storage:           dexv1.DexStorageSpec{Type: dexv1.StorageKubernetes},
		},
	}
	if err := e2eClient.Create(context.Background(), inst); err != nil {
		t.Fatalf("create installation: %v", err)
	}
	t.Cleanup(func() { _ = e2eClient.Delete(context.Background(), inst) })

	e2eCreateSecret(t, ns, "grafana-creds", map[string][]byte{
		"client-id":     []byte("grafana"),
		"client-secret": []byte("grafana-e2e-secret"),
	})

	sc := &dexv1.DexStaticClient{
		ObjectMeta: metav1.ObjectMeta{Name: "grafana", Namespace: ns},
		Spec: dexv1.DexStaticClientSpec{
			InstallationRef: dexv1.InstallationRef{Name: "dex", Namespace: ns},
			DisplayName:     "Grafana",
			RedirectURIs:    []string{"https://grafana.e2e.example.com/login/generic_oauth"},
			SecretRef: dexv1.StaticClientSecretRef{
				Name:            "grafana-creds",
				ClientIDKey:     "client-id",
				ClientSecretKey: "client-secret",
			},
		},
	}
	if err := e2eClient.Create(context.Background(), sc); err != nil {
		t.Fatalf("create static client: %v", err)
	}
	t.Cleanup(func() { _ = e2eClient.Delete(context.Background(), sc) })

	e2eEventually(t, func() bool {
		s := e2eGetSecret(ns, "dex-config")
		return s != nil && strings.Contains(string(s.Data["config.yaml"]), "grafana.e2e.example.com")
	}, "static client not in config.yaml")

	e2eEventually(t, func() bool {
		var updated dexv1.DexInstallation
		if err := e2eClient.Get(context.Background(),
			client.ObjectKey{Namespace: ns, Name: "dex"}, &updated); err != nil {
			return false
		}
		return updated.Status.StaticClientCount >= 1
	}, "StaticClientCount should be >= 1")
}

// TestE2E_NamespaceIsolation verifies that only connectors from allowed
// namespaces are included in the rendered config.
func TestE2E_NamespaceIsolation(t *testing.T) {
	nsInst := "e2e-ns-inst"
	nsAllowed := "e2e-ns-allowed"
	nsForbidden := "e2e-ns-forbidden"
	for _, ns := range []string{nsInst, nsAllowed, nsForbidden} {
		e2eCreateNamespace(t, ns)
	}

	inst := &dexv1.DexInstallation{
		ObjectMeta: metav1.ObjectMeta{Name: "dex", Namespace: nsInst},
		Spec: dexv1.DexInstallationSpec{
			Issuer:            "https://dex.e2e-ns.example.com",
			ConfigSecretName:  "dex-config",
			EnvSecretName:     "dex-env",
			AllowedNamespaces: []string{nsAllowed}, // only nsAllowed
			Storage:           dexv1.DexStorageSpec{Type: dexv1.StorageKubernetes},
		},
	}
	if err := e2eClient.Create(context.Background(), inst); err != nil {
		t.Fatalf("create installation: %v", err)
	}
	t.Cleanup(func() { _ = e2eClient.Delete(context.Background(), inst) })

	// Create connector in allowed namespace.
	e2eCreateSecret(t, nsAllowed, "oidc-a", map[string][]byte{
		"client-id":     []byte("a-id"),
		"client-secret": []byte("a-secret"),
	})
	allowedConn := &dexv1.DexOIDCConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "allowed", Namespace: nsAllowed},
		Spec: dexv1.DexOIDCConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: "dex", Namespace: nsInst},
			DisplayName:     "Allowed",
			Issuer:          "https://allowed.e2e.example.com",
			ClientIDRef:     dexv1.SecretKeyRef{Name: "oidc-a", Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "oidc-a", Key: "client-secret"},
		},
	}
	if err := e2eClient.Create(context.Background(), allowedConn); err != nil {
		t.Fatalf("create allowed connector: %v", err)
	}
	t.Cleanup(func() { _ = e2eClient.Delete(context.Background(), allowedConn) })

	// Create connector in forbidden namespace.
	e2eCreateSecret(t, nsForbidden, "oidc-f", map[string][]byte{
		"client-id":     []byte("f-id"),
		"client-secret": []byte("f-secret"),
	})
	forbiddenConn := &dexv1.DexOIDCConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "forbidden", Namespace: nsForbidden},
		Spec: dexv1.DexOIDCConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: "dex", Namespace: nsInst},
			DisplayName:     "Forbidden",
			Issuer:          "https://forbidden.e2e.example.com",
			ClientIDRef:     dexv1.SecretKeyRef{Name: "oidc-f", Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "oidc-f", Key: "client-secret"},
		},
	}
	if err := e2eClient.Create(context.Background(), forbiddenConn); err != nil {
		t.Fatalf("create forbidden connector: %v", err)
	}
	t.Cleanup(func() { _ = e2eClient.Delete(context.Background(), forbiddenConn) })

	// Wait for allowed connector to appear.
	e2eEventually(t, func() bool {
		s := e2eGetSecret(nsInst, "dex-config")
		return s != nil && strings.Contains(string(s.Data["config.yaml"]), "allowed.e2e.example.com")
	}, "allowed connector should be in config.yaml")

	// Forbidden connector must not appear.
	s := e2eGetSecret(nsInst, "dex-config")
	if s != nil && strings.Contains(string(s.Data["config.yaml"]), "forbidden.e2e.example.com") {
		t.Error("forbidden connector appeared in config.yaml")
	}
}

// TestE2E_ConnectorLifecycle tests the full lifecycle: add a connector, verify
// it appears, delete it, verify it disappears.
func TestE2E_ConnectorLifecycle(t *testing.T) {
	ns := "e2e-lifecycle"
	e2eCreateNamespace(t, ns)

	inst := &dexv1.DexInstallation{
		ObjectMeta: metav1.ObjectMeta{Name: "dex", Namespace: ns},
		Spec: dexv1.DexInstallationSpec{
			Issuer:            "https://dex.e2e-lifecycle.example.com",
			ConfigSecretName:  "dex-config",
			EnvSecretName:     "dex-env",
			AllowedNamespaces: []string{"*"},
			Storage:           dexv1.DexStorageSpec{Type: dexv1.StorageKubernetes},
		},
	}
	if err := e2eClient.Create(context.Background(), inst); err != nil {
		t.Fatalf("create installation: %v", err)
	}
	t.Cleanup(func() { _ = e2eClient.Delete(context.Background(), inst) })

	// Wait for base reconcile.
	e2eEventually(t, func() bool {
		return e2eGetSecret(ns, "dex-config") != nil
	}, "config secret not created")

	e2eCreateSecret(t, ns, "lc-creds", map[string][]byte{
		"client-id":     []byte("lc-id"),
		"client-secret": []byte("lc-secret"),
	})

	conn := &dexv1.DexOIDCConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "lifecycle-conn", Namespace: ns},
		Spec: dexv1.DexOIDCConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: "dex", Namespace: ns},
			DisplayName:     "Lifecycle",
			Issuer:          "https://lifecycle.e2e.example.com",
			ClientIDRef:     dexv1.SecretKeyRef{Name: "lc-creds", Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "lc-creds", Key: "client-secret"},
		},
	}
	if err := e2eClient.Create(context.Background(), conn); err != nil {
		t.Fatalf("create connector: %v", err)
	}

	// Connector must appear.
	e2eEventually(t, func() bool {
		s := e2eGetSecret(ns, "dex-config")
		return s != nil && strings.Contains(string(s.Data["config.yaml"]), "lifecycle.e2e.example.com")
	}, "connector not in config.yaml after create")

	// Delete connector.
	if err := e2eClient.Delete(context.Background(), conn); err != nil {
		t.Fatalf("delete connector: %v", err)
	}

	// Connector must disappear from config.
	e2eEventually(t, func() bool {
		s := e2eGetSecret(ns, "dex-config")
		return s != nil && !strings.Contains(string(s.Data["config.yaml"]), "lifecycle.e2e.example.com")
	}, "connector still in config.yaml after delete")
}
