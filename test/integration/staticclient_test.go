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
	"sigs.k8s.io/controller-runtime/pkg/client"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// TestIntegration_StaticClient verifies that a DexStaticClient is rendered
// into config.yaml and its secret is injected into the env secret.
func TestIntegration_StaticClient(t *testing.T) {
	ns := "it-static-client"
	createNamespace(t, ns)
	inst := createInstallation(t, ns, "dex", []string{"*"})

	createSecret(t, ns, "grafana-creds", map[string][]byte{
		"client-id":     []byte("grafana"),
		"client-secret": []byte("grafana-secret-value"),
	})

	sc := &dexv1.DexStaticClient{
		ObjectMeta: metav1.ObjectMeta{Name: "grafana", Namespace: ns},
		Spec: dexv1.DexStaticClientSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: ns},
			DisplayName:     "Grafana",
			RedirectURIs:    []string{"https://grafana.example.com/login/generic_oauth"},
			SecretRef: &dexv1.StaticClientSecretRef{
				Name:            "grafana-creds",
				ClientIDKey:     "client-id",
				ClientSecretKey: "client-secret",
			},
		},
	}
	if err := k8sClient.Create(context.Background(), sc); err != nil {
		t.Fatalf("create static client: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), sc) })

	// config.yaml must list the static client.
	eventually(t, func() bool {
		s := getSecret(ns, inst.Spec.ConfigSecretName)
		return s != nil && strings.Contains(string(s.Data["config.yaml"]), "grafana.example.com")
	}, "static client redirect URI not in config.yaml")

	// env secret must contain the client secret.
	eventually(t, func() bool {
		s := getSecret(ns, inst.Spec.EnvSecretName)
		if s == nil {
			return false
		}
		for _, v := range s.Data {
			if string(v) == "grafana-secret-value" {
				return true
			}
		}
		return false
	}, "static client secret not in env secret")

	// StaticClientCount status must be 1.
	eventually(t, func() bool {
		var latest dexv1.DexInstallation
		if err := k8sClient.Get(context.Background(),
			client.ObjectKey{Namespace: ns, Name: inst.Name}, &latest); err != nil {
			return false
		}
		return latest.Status.StaticClientCount == 1
	}, "StaticClientCount should be 1")

	// DexStaticClient status must be Ready=True.
	eventually(t, func() bool {
		var updated dexv1.DexStaticClient
		if err := k8sClient.Get(context.Background(),
			client.ObjectKey{Namespace: ns, Name: sc.Name}, &updated); err != nil {
			return false
		}
		cond := findCondition(updated.Status.Conditions, dexv1.ConditionTypeReady)
		return cond != nil && cond.Status == metav1.ConditionTrue
	}, "DexStaticClient Ready condition not True")
}

// TestIntegration_StaticClientDelete verifies that deleting a DexStaticClient
// triggers a re-reconcile that removes it from config.yaml.
func TestIntegration_StaticClientDelete(t *testing.T) {
	ns := "it-sc-delete"
	createNamespace(t, ns)
	inst := createInstallation(t, ns, "dex", []string{"*"})

	createSecret(t, ns, "app-creds", map[string][]byte{
		"client-id":     []byte("my-app"),
		"client-secret": []byte("my-app-secret"),
	})

	sc := &dexv1.DexStaticClient{
		ObjectMeta: metav1.ObjectMeta{Name: "my-app", Namespace: ns},
		Spec: dexv1.DexStaticClientSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: ns},
			DisplayName:     "My App",
			RedirectURIs:    []string{"https://my-app.example.com/callback"},
			SecretRef: &dexv1.StaticClientSecretRef{
				Name:            "app-creds",
				ClientIDKey:     "client-id",
				ClientSecretKey: "client-secret",
			},
		},
	}
	if err := k8sClient.Create(context.Background(), sc); err != nil {
		t.Fatalf("create static client: %v", err)
	}

	// Wait for it to appear in config.
	eventually(t, func() bool {
		s := getSecret(ns, inst.Spec.ConfigSecretName)
		return s != nil && strings.Contains(string(s.Data["config.yaml"]), "my-app.example.com")
	}, "static client not in config.yaml")

	// Delete the static client.
	if err := k8sClient.Delete(context.Background(), sc); err != nil {
		t.Fatalf("delete static client: %v", err)
	}

	// Config must no longer reference it.
	eventually(t, func() bool {
		s := getSecret(ns, inst.Spec.ConfigSecretName)
		return s != nil && !strings.Contains(string(s.Data["config.yaml"]), "my-app.example.com")
	}, "deleted static client still in config.yaml")

	// StaticClientCount must drop back to 0.
	eventually(t, func() bool {
		var latest dexv1.DexInstallation
		if err := k8sClient.Get(context.Background(),
			client.ObjectKey{Namespace: ns, Name: inst.Name}, &latest); err != nil {
			return false
		}
		return latest.Status.StaticClientCount == 0
	}, "StaticClientCount should be 0 after deletion")
}

// TestIntegration_MultipleStaticClients verifies that multiple static clients
// are all rendered into config.yaml.
func TestIntegration_MultipleStaticClients(t *testing.T) {
	ns := "it-multi-sc"
	createNamespace(t, ns)
	inst := createInstallation(t, ns, "dex", []string{"*"})

	clients := []struct {
		name        string
		redirectURI string
	}{
		{"prometheus", "https://prometheus.example.com/oauth"},
		{"argocd", "https://argocd.example.com/auth/callback"},
	}

	for _, c := range clients {
		createSecret(t, ns, c.name+"-creds", map[string][]byte{
			"client-id":     []byte(c.name),
			"client-secret": []byte(c.name + "-secret"),
		})
		sc := &dexv1.DexStaticClient{
			ObjectMeta: metav1.ObjectMeta{Name: c.name, Namespace: ns},
			Spec: dexv1.DexStaticClientSpec{
				InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: ns},
				DisplayName:     c.name,
				RedirectURIs:    []string{c.redirectURI},
				SecretRef: &dexv1.StaticClientSecretRef{
					Name:            c.name + "-creds",
					ClientIDKey:     "client-id",
					ClientSecretKey: "client-secret",
				},
			},
		}
		if err := k8sClient.Create(context.Background(), sc); err != nil {
			t.Fatalf("create static client %s: %v", c.name, err)
		}
		scCopy := sc
		t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), scCopy) })
	}

	// Both redirect URIs must appear in config.yaml.
	eventually(t, func() bool {
		s := getSecret(ns, inst.Spec.ConfigSecretName)
		if s == nil {
			return false
		}
		cfg := string(s.Data["config.yaml"])
		return strings.Contains(cfg, "prometheus.example.com") &&
			strings.Contains(cfg, "argocd.example.com")
	}, "not all static clients in config.yaml")

	eventually(t, func() bool {
		var latest dexv1.DexInstallation
		if err := k8sClient.Get(context.Background(),
			client.ObjectKey{Namespace: ns, Name: inst.Name}, &latest); err != nil {
			return false
		}
		return latest.Status.StaticClientCount == 2
	}, "StaticClientCount should be 2")
}

// TestIntegration_SecretlessPublicStaticClient verifies that a public client
// without a secretRef is rendered with its inline clientID and contributes
// nothing to the env secret.
func TestIntegration_SecretlessPublicStaticClient(t *testing.T) {
	ns := "it-secretless-sc"
	createNamespace(t, ns)
	inst := createInstallation(t, ns, "dex", []string{"*"})

	sc := &dexv1.DexStaticClient{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cli", Namespace: ns},
		Spec: dexv1.DexStaticClientSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: ns},
			DisplayName:     "My CLI",
			ClientID:        "my-cli-id",
			Public:          true,
			RedirectURIs:    []string{"http://127.0.0.1:8085/callback"},
		},
	}
	if err := k8sClient.Create(context.Background(), sc); err != nil {
		t.Fatalf("create secretless static client: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), sc) })

	eventually(t, func() bool {
		s := getSecret(ns, inst.Spec.ConfigSecretName)
		if s == nil {
			return false
		}
		cfg := string(s.Data["config.yaml"])
		return strings.Contains(cfg, "id: my-cli-id") &&
			strings.Contains(cfg, "public: true") &&
			!strings.Contains(cfg, "secretEnv")
	}, "secretless public client not rendered as expected in config.yaml")

	// The env secret must not gain an entry for this client.
	eventually(t, func() bool {
		s := getSecret(ns, inst.Spec.EnvSecretName)
		if s == nil {
			return false
		}
		_, ok := s.Data["MY_CLI_CLIENT_SECRET"]
		return !ok
	}, "secretless public client must not add an env secret entry")

	eventually(t, func() bool {
		var updated dexv1.DexStaticClient
		if err := k8sClient.Get(context.Background(),
			client.ObjectKey{Namespace: ns, Name: sc.Name}, &updated); err != nil {
			return false
		}
		cond := findCondition(updated.Status.Conditions, dexv1.ConditionTypeReady)
		return cond != nil && cond.Status == metav1.ConditionTrue
	}, "secretless DexStaticClient Ready condition not True")
}

// TestIntegration_SecretlessPublicStaticClient_NoRedirectURIs verifies that a
// public client may omit redirectURIs entirely, so that Dex falls back to its
// loopback / OOB / device-flow defaults.
func TestIntegration_SecretlessPublicStaticClient_NoRedirectURIs(t *testing.T) {
	ns := "it-secretless-noredirect"
	createNamespace(t, ns)
	inst := createInstallation(t, ns, "dex", []string{"*"})

	sc := &dexv1.DexStaticClient{
		ObjectMeta: metav1.ObjectMeta{Name: "native-app", Namespace: ns},
		Spec: dexv1.DexStaticClientSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: ns},
			DisplayName:     "Native App",
			ClientID:        "native-app-id",
			Public:          true,
		},
	}
	if err := k8sClient.Create(context.Background(), sc); err != nil {
		t.Fatalf("create secretless static client without redirectURIs: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), sc) })

	eventually(t, func() bool {
		s := getSecret(ns, inst.Spec.ConfigSecretName)
		if s == nil {
			return false
		}
		cfg := string(s.Data["config.yaml"])
		return strings.Contains(cfg, "id: native-app-id") &&
			!strings.Contains(cfg, "redirectURIs")
	}, "public client without redirectURIs not rendered as expected in config.yaml")
}

// TestIntegration_StaticClientCELValidation verifies that the CEL rules on
// DexStaticClientSpec reject invalid clientID / secretRef / redirectURIs
// combinations at admission time.
func TestIntegration_StaticClientCELValidation(t *testing.T) {
	ns := "it-sc-cel"
	createNamespace(t, ns)
	inst := createInstallation(t, ns, "dex", []string{"*"})

	tests := []struct {
		name    string
		spec    dexv1.DexStaticClientSpec
		wantMsg string
	}{
		{
			name: "confidential without secretRef",
			spec: dexv1.DexStaticClientSpec{
				DisplayName:  "No Secret Ref",
				ClientID:     "no-secret-ref",
				RedirectURIs: []string{"https://example.com/callback"},
			},
			wantMsg: "secretRef is required for confidential (non-public) clients",
		},
		{
			name: "clientID and secretRef both set",
			spec: dexv1.DexStaticClientSpec{
				DisplayName:  "Both",
				ClientID:     "both",
				SecretRef:    &dexv1.StaticClientSecretRef{Name: "some-creds"},
				RedirectURIs: []string{"https://example.com/callback"},
			},
			wantMsg: "exactly one of clientID or secretRef must be set",
		},
		{
			name: "confidential without redirectURIs",
			spec: dexv1.DexStaticClientSpec{
				DisplayName: "No Redirects",
				SecretRef:   &dexv1.StaticClientSecretRef{Name: "some-creds"},
			},
			wantMsg: "redirectURIs is required for confidential (non-public) clients",
		},
		{
			name: "neither clientID nor secretRef",
			spec: dexv1.DexStaticClientSpec{
				DisplayName:  "Neither",
				Public:       true,
				RedirectURIs: []string{"https://example.com/callback"},
			},
			wantMsg: "exactly one of clientID or secretRef must be set",
		},
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec := tc.spec
			spec.InstallationRef = dexv1.InstallationRef{Name: inst.Name, Namespace: ns}
			sc := &dexv1.DexStaticClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-" + strconv.Itoa(i),
					Namespace: ns,
				},
				Spec: spec,
			}
			err := k8sClient.Create(context.Background(), sc)
			if err == nil {
				_ = k8sClient.Delete(context.Background(), sc)
				t.Fatalf("expected creation to be rejected, but it succeeded")
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantMsg)
			}
		})
	}
}
