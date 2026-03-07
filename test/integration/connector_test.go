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
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// TestIntegration_OIDCConnector verifies that a DexOIDCConnector is rendered
// into config.yaml and that its credentials are injected as env vars.
func TestIntegration_OIDCConnector(t *testing.T) {
	ns := "it-oidc"
	createNamespace(t, ns)
	inst := createInstallation(t, ns, "dex", []string{"*"})

	createSecret(t, ns, "oidc-creds", map[string][]byte{
		"client-id":     []byte("oidc-client-id"),
		"client-secret": []byte("oidc-super-secret"),
	})

	conn := &dexv1.DexOIDCConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "google-workspace", Namespace: ns},
		Spec: dexv1.DexOIDCConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: ns},
			Name:            "Google Workspace",
			Issuer:          "https://accounts.google.com",
			ClientIDRef:     dexv1.SecretKeyRef{Name: "oidc-creds", Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "oidc-creds", Key: "client-secret"},
		},
	}
	if err := k8sClient.Create(context.Background(), conn); err != nil {
		t.Fatalf("create OIDC connector: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), conn) })

	// config.yaml must reference the OIDC connector type.
	eventually(t, func() bool {
		s := getSecret(ns, inst.Spec.ConfigSecretName)
		return s != nil && strings.Contains(string(s.Data["config.yaml"]), "oidc")
	}, "OIDC connector not in config.yaml")

	// env secret must contain the client-secret env var.
	eventually(t, func() bool {
		s := getSecret(ns, inst.Spec.EnvSecretName)
		if s == nil {
			return false
		}
		for k, v := range s.Data {
			if strings.HasSuffix(k, "_CLIENT_SECRET") && string(v) == "oidc-super-secret" {
				return true
			}
		}
		return false
	}, "OIDC client secret not in env secret")

	// Connector status should be Ready=True.
	eventually(t, func() bool {
		var updated dexv1.DexOIDCConnector
		if err := k8sClient.Get(context.Background(),
			client.ObjectKey{Namespace: ns, Name: conn.Name}, &updated); err != nil {
			return false
		}
		cond := findCondition(updated.Status.Conditions, dexv1.ConditionTypeReady)
		return cond != nil && cond.Status == metav1.ConditionTrue
	}, "OIDC connector Ready condition not True")
}

// TestIntegration_LDAPConnector verifies that a DexLDAPConnector is rendered
// into config.yaml.
func TestIntegration_LDAPConnector(t *testing.T) {
	ns := "it-ldap"
	createNamespace(t, ns)
	inst := createInstallation(t, ns, "dex", []string{"*"})

	createSecret(t, ns, "ldap-creds", map[string][]byte{
		"bind-password": []byte("ldap-password"),
	})

	bindPWRef := dexv1.SecretKeyRef{Name: "ldap-creds", Key: "bind-password"}
	conn := &dexv1.DexLDAPConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "company-ldap", Namespace: ns},
		Spec: dexv1.DexLDAPConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: ns},
			Name:            "Company LDAP",
			Host:            "ldap.example.com:636",
			BindDN:          "cn=admin,dc=example,dc=com",
			BindPWRef:       &bindPWRef,
			UserSearch: dexv1.LDAPUserSearch{
				BaseDN:   "ou=users,dc=example,dc=com",
				Username: "cn",
			},
		},
	}
	if err := k8sClient.Create(context.Background(), conn); err != nil {
		t.Fatalf("create LDAP connector: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), conn) })

	eventually(t, func() bool {
		s := getSecret(ns, inst.Spec.ConfigSecretName)
		return s != nil && strings.Contains(string(s.Data["config.yaml"]), "ldap")
	}, "LDAP connector not in config.yaml")
}

// TestIntegration_GitHubConnector verifies that a DexGitHubConnector
// is included in the rendered config.
func TestIntegration_GitHubConnector(t *testing.T) {
	ns := "it-github"
	createNamespace(t, ns)
	inst := createInstallation(t, ns, "dex", []string{"*"})

	createSecret(t, ns, "gh-creds", map[string][]byte{
		"client-id":     []byte("gh-id"),
		"client-secret": []byte("gh-secret"),
	})

	conn := &dexv1.DexGitHubConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "github", Namespace: ns},
		Spec: dexv1.DexGitHubConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: ns},
			Name:            "GitHub",
			ClientIDRef:     dexv1.SecretKeyRef{Name: "gh-creds", Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "gh-creds", Key: "client-secret"},
		},
	}
	if err := k8sClient.Create(context.Background(), conn); err != nil {
		t.Fatalf("create GitHub connector: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), conn) })

	eventually(t, func() bool {
		s := getSecret(ns, inst.Spec.ConfigSecretName)
		return s != nil && strings.Contains(string(s.Data["config.yaml"]), "github")
	}, "GitHub connector not in config.yaml")
}

// TestIntegration_ConnectorForbiddenNamespace verifies that a connector placed
// in a namespace that is not in AllowedNamespaces gets Ready=False status.
func TestIntegration_ConnectorForbiddenNamespace(t *testing.T) {
	nsInst := "it-conn-inst"
	nsForbidden := "it-conn-forbidden"
	for _, ns := range []string{nsInst, nsForbidden} {
		createNamespace(t, ns)
	}
	inst := createInstallation(t, nsInst, "dex", []string{nsInst}) // only nsInst allowed

	conn := &dexv1.DexLocalConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "local", Namespace: nsForbidden},
		Spec: dexv1.DexLocalConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: nsInst},
			Name:            "Local",
		},
	}
	if err := k8sClient.Create(context.Background(), conn); err != nil {
		t.Fatalf("create local connector: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), conn) })

	// The connector must show Ready=False.
	eventually(t, func() bool {
		var updated dexv1.DexLocalConnector
		if err := k8sClient.Get(context.Background(),
			client.ObjectKey{Namespace: nsForbidden, Name: conn.Name}, &updated); err != nil {
			return false
		}
		cond := findCondition(updated.Status.Conditions, dexv1.ConditionTypeReady)
		return cond != nil && cond.Status == metav1.ConditionFalse
	}, "connector in forbidden namespace should have Ready=False")
}

// TestIntegration_ConnectorDeleteTriggersReconcile verifies that deleting a
// connector triggers a re-reconcile that removes it from the config.
func TestIntegration_ConnectorDeleteTriggersReconcile(t *testing.T) {
	ns := "it-conn-delete"
	createNamespace(t, ns)
	inst := createInstallation(t, ns, "dex", []string{"*"})

	createSecret(t, ns, "oidc-del-creds", map[string][]byte{
		"client-id":     []byte("del-id"),
		"client-secret": []byte("del-secret"),
	})

	conn := &dexv1.DexOIDCConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "to-delete", Namespace: ns},
		Spec: dexv1.DexOIDCConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: ns},
			Name:            "ToDelete",
			Issuer:          "https://delete-me.example.com",
			ClientIDRef:     dexv1.SecretKeyRef{Name: "oidc-del-creds", Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "oidc-del-creds", Key: "client-secret"},
		},
	}
	if err := k8sClient.Create(context.Background(), conn); err != nil {
		t.Fatalf("create connector: %v", err)
	}

	// Wait until the connector is included.
	eventually(t, func() bool {
		s := getSecret(ns, inst.Spec.ConfigSecretName)
		return s != nil && strings.Contains(string(s.Data["config.yaml"]), "delete-me.example.com")
	}, "connector not in config.yaml")

	// Delete the connector.
	if err := k8sClient.Delete(context.Background(), conn); err != nil {
		t.Fatalf("delete connector: %v", err)
	}

	// Config must no longer reference the deleted connector.
	eventually(t, func() bool {
		s := getSecret(ns, inst.Spec.ConfigSecretName)
		return s != nil && !strings.Contains(string(s.Data["config.yaml"]), "delete-me.example.com")
	}, "deleted connector still in config.yaml")
}
