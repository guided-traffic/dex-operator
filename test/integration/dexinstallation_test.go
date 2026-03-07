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

// TestIntegration_MinimalInstallation verifies that a minimal DexInstallation
// leads to both a config secret and an env secret being created in the same
// namespace, and that the status is set to Ready=True.
func TestIntegration_MinimalInstallation(t *testing.T) {
	ns := "it-minimal"
	createNamespace(t, ns)
	inst := createInstallation(t, ns, "dex", []string{"*"})

	// Config secret must be created.
	eventually(t, func() bool {
		return getSecret(ns, inst.Spec.ConfigSecretName) != nil
	}, "config secret not created")

	// Env secret must be created.
	eventually(t, func() bool {
		return getSecret(ns, inst.Spec.EnvSecretName) != nil
	}, "env secret not created")

	// config.yaml key must be present.
	eventually(t, func() bool {
		s := getSecret(ns, inst.Spec.ConfigSecretName)
		return s != nil && len(s.Data["config.yaml"]) > 0
	}, "config.yaml key missing")

	// Status condition Ready=True must be set.
	eventually(t, func() bool {
		var updated dexv1.DexInstallation
		if err := k8sClient.Get(context.Background(),
			client.ObjectKey{Namespace: ns, Name: inst.Name}, &updated); err != nil {
			return false
		}
		cond := findCondition(updated.Status.Conditions, dexv1.ConditionTypeReady)
		return cond != nil && cond.Status == metav1.ConditionTrue
	}, "Ready condition not True")
}

// TestIntegration_IssuerInConfig verifies that the issuer URL is present in
// the rendered config.yaml.
func TestIntegration_IssuerInConfig(t *testing.T) {
	ns := "it-issuer"
	createNamespace(t, ns)
	inst := createInstallation(t, ns, "dex", []string{"*"})

	eventually(t, func() bool {
		s := getSecret(ns, inst.Spec.ConfigSecretName)
		if s == nil {
			return false
		}
		return strings.Contains(string(s.Data["config.yaml"]), "https://dex.example.com")
	}, "issuer URL not in config.yaml")
}

// TestIntegration_ConfigSecretLabels verifies that the managed-by label is set
// on the generated secrets.
func TestIntegration_ConfigSecretLabels(t *testing.T) {
	ns := "it-labels"
	createNamespace(t, ns)
	inst := createInstallation(t, ns, "dex", []string{"*"})

	eventually(t, func() bool {
		s := getSecret(ns, inst.Spec.ConfigSecretName)
		if s == nil {
			return false
		}
		return s.Labels["app.kubernetes.io/managed-by"] == "dex-operator"
	}, "managed-by label not set on config secret")
}

// TestIntegration_Idempotency verifies that reconciling the same installation
// twice does not change the resource version of the config secret.
func TestIntegration_Idempotency(t *testing.T) {
	ns := "it-idempotent"
	createNamespace(t, ns)
	inst := createInstallation(t, ns, "dex", []string{"*"})

	// Wait for config secret to be created.
	eventually(t, func() bool {
		return getSecret(ns, inst.Spec.ConfigSecretName) != nil
	}, "config secret not created")

	s1 := getSecret(ns, inst.Spec.ConfigSecretName)
	rv1 := s1.ResourceVersion

	// Touch the installation to trigger another reconcile round by updating a label.
	// Use a retry loop because the reconciler may update the status concurrently,
	// causing an optimistic-concurrency conflict on the first attempt.
	eventually(t, func() bool {
		var latest dexv1.DexInstallation
		if err := k8sClient.Get(context.Background(),
			client.ObjectKey{Namespace: ns, Name: inst.Name}, &latest); err != nil {
			return false
		}
		if latest.Labels == nil {
			latest.Labels = map[string]string{}
		}
		latest.Labels["test.gtrfc.com/touch"] = "1"
		return k8sClient.Update(context.Background(), &latest) == nil
	}, "failed to update installation labels")

	// Give the manager a moment to process the update.
	eventually(t, func() bool {
		var latest dexv1.DexInstallation
		if err := k8sClient.Get(context.Background(),
			client.ObjectKey{Namespace: ns, Name: inst.Name}, &latest); err != nil {
			return false
		}
		return latest.Labels["test.gtrfc.com/touch"] == "1"
	}, "label update not observed")

	// The resource version of the secret must remain stable (no-op reconcile).
	s2 := getSecret(ns, inst.Spec.ConfigSecretName)
	if s2 == nil {
		t.Fatal("config secret disappeared")
	}
	if s2.ResourceVersion != rv1 {
		t.Errorf("config secret was re-patched (rv %s → %s)", rv1, s2.ResourceVersion)
	}
}

// TestIntegration_StatusConnectorCount verifies that the ConnectorCount status
// field reflects the number of connectors referencing the installation.
func TestIntegration_StatusConnectorCount(t *testing.T) {
	ns := "it-count"
	createNamespace(t, ns)
	inst := createInstallation(t, ns, "dex", []string{"*"})

	// Wait for initial reconcile.
	eventually(t, func() bool {
		return getSecret(ns, inst.Spec.ConfigSecretName) != nil
	}, "config secret not created")

	// Confirm zero connectors initially.
	eventually(t, func() bool {
		var latest dexv1.DexInstallation
		if err := k8sClient.Get(context.Background(),
			client.ObjectKey{Namespace: ns, Name: inst.Name}, &latest); err != nil {
			return false
		}
		return latest.Status.ConnectorCount == 0
	}, "initial connector count should be 0")

	// Create an OIDC connector secret and connector.
	createSecret(t, ns, "oidc-creds", map[string][]byte{
		"client-id":     []byte("cid"),
		"client-secret": []byte("csecret"),
	})

	conn := &dexv1.DexOIDCConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "okta", Namespace: ns},
		Spec: dexv1.DexOIDCConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: ns},
			Name:            "Okta",
			Issuer:          "https://okta.example.com",
			ClientIDRef:     dexv1.SecretKeyRef{Name: "oidc-creds", Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "oidc-creds", Key: "client-secret"},
		},
	}
	if err := k8sClient.Create(context.Background(), conn); err != nil {
		t.Fatalf("create OIDC connector: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), conn) })

	// The installation should reflect 1 connector.
	eventually(t, func() bool {
		var latest dexv1.DexInstallation
		if err := k8sClient.Get(context.Background(),
			client.ObjectKey{Namespace: ns, Name: inst.Name}, &latest); err != nil {
			return false
		}
		return latest.Status.ConnectorCount == 1
	}, "connector count should be 1 after adding OIDC connector")
}

// TestIntegration_AllowedNamespacesFilter verifies that connectors in
// disallowed namespaces are excluded from the rendered config.
func TestIntegration_AllowedNamespacesFilter(t *testing.T) {
	nsInst := "it-ns-inst"
	nsAllowed := "it-ns-allowed"
	nsForbidden := "it-ns-forbidden"
	for _, ns := range []string{nsInst, nsAllowed, nsForbidden} {
		createNamespace(t, ns)
	}

	// Installation only allows nsAllowed.
	inst := createInstallation(t, nsInst, "dex", []string{nsAllowed})

	// Create a connector in the allowed namespace.
	createSecret(t, nsAllowed, "oidc-creds-allowed", map[string][]byte{
		"client-id":     []byte("allowed-id"),
		"client-secret": []byte("allowed-secret"),
	})
	allowedConn := &dexv1.DexOIDCConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "oidc-allowed", Namespace: nsAllowed},
		Spec: dexv1.DexOIDCConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: nsInst},
			Name:            "Allowed",
			Issuer:          "https://allowed.example.com",
			ClientIDRef:     dexv1.SecretKeyRef{Name: "oidc-creds-allowed", Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "oidc-creds-allowed", Key: "client-secret"},
		},
	}
	if err := k8sClient.Create(context.Background(), allowedConn); err != nil {
		t.Fatalf("create allowed connector: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), allowedConn) })

	// Create a connector in the forbidden namespace.
	createSecret(t, nsForbidden, "oidc-creds-forbidden", map[string][]byte{
		"client-id":     []byte("forbidden-id"),
		"client-secret": []byte("forbidden-secret"),
	})
	forbiddenConn := &dexv1.DexOIDCConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "oidc-forbidden", Namespace: nsForbidden},
		Spec: dexv1.DexOIDCConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: nsInst},
			Name:            "Forbidden",
			Issuer:          "https://forbidden.example.com",
			ClientIDRef:     dexv1.SecretKeyRef{Name: "oidc-creds-forbidden", Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "oidc-creds-forbidden", Key: "client-secret"},
		},
	}
	if err := k8sClient.Create(context.Background(), forbiddenConn); err != nil {
		t.Fatalf("create forbidden connector: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), forbiddenConn) })

	// Wait for installation to reconcile with 1 connector (allowed only).
	eventually(t, func() bool {
		var latest dexv1.DexInstallation
		if err := k8sClient.Get(context.Background(),
			client.ObjectKey{Namespace: nsInst, Name: inst.Name}, &latest); err != nil {
			return false
		}
		return latest.Status.ConnectorCount == 1
	}, "connector count should be 1 (only allowed namespace)")

	// Forbidden connector's issuer must not appear in config.
	s := getSecret(nsInst, inst.Spec.ConfigSecretName)
	if s == nil {
		t.Fatal("config secret not found")
	}
	cfg := string(s.Data["config.yaml"])
	if strings.Contains(cfg, "forbidden.example.com") {
		t.Errorf("forbidden connector appears in config.yaml:\n%s", cfg)
	}
	if !strings.Contains(cfg, "allowed.example.com") {
		t.Errorf("allowed connector missing from config.yaml:\n%s", cfg)
	}
}
