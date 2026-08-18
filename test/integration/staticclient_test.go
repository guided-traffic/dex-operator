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

	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
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

// ── Derived CORS origins ──────────────────────────────────────────────────────

// configAllowedOrigins parses the rendered config.yaml and returns
// web.allowedOrigins (nil if the Secret or the web block is absent).
//
// Parsing rather than substring matching is essential here: a derived origin is
// a prefix of the redirect URI it was derived from, so strings.Contains cannot
// tell "origin registered" from "redirect URI present" — and the removal test
// below relies on exactly that distinction.
func configAllowedOrigins(t *testing.T, ns, secretName string) []string {
	t.Helper()
	s := getSecret(ns, secretName)
	if s == nil {
		return nil
	}
	var cfg struct {
		Web *struct {
			AllowedOrigins []string `yaml:"allowedOrigins"`
		} `yaml:"web"`
	}
	if err := yaml.Unmarshal(s.Data["config.yaml"], &cfg); err != nil {
		t.Fatalf("invalid config.yaml: %v\n%s", err, string(s.Data["config.yaml"]))
	}
	if cfg.Web == nil {
		return nil
	}
	return cfg.Web.AllowedOrigins
}

func originsEqual(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func hasOrigin(origins []string, want string) bool {
	for _, o := range origins {
		if o == want {
			return true
		}
	}
	return false
}

// mutateResource re-fetches obj by its own key, applies mutate and writes it
// back, retrying on conflict.
//
// Mutations must not be driven through eventually(): that helper re-evaluates
// its condition once more after it first succeeded, which replays the write
// against a resourceVersion the operator has meanwhile bumped.
func mutateResource[T client.Object](t *testing.T, obj T, mutate func(T)) {
	t.Helper()
	key := client.ObjectKeyFromObject(obj)
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := k8sClient.Get(context.Background(), key, obj); err != nil {
			return err
		}
		mutate(obj)
		return k8sClient.Update(context.Background(), obj)
	})
	if err != nil {
		t.Fatalf("update %s: %v", key, err)
	}
}

// corsStaticClient returns a public client that opted into CORS origin
// derivation.
func corsStaticClient(name, ns string, inst *dexv1.DexInstallation, redirectURIs []string) *dexv1.DexStaticClient {
	return &dexv1.DexStaticClient{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: dexv1.DexStaticClientSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: inst.Namespace},
			DisplayName:     name,
			ClientID:        name + "-id",
			Public:          true,
			CORS:            true,
			RedirectURIs:    redirectURIs,
		},
	}
}

// TestIntegration_StaticClientCORSOrigin verifies the full path from a
// DexStaticClient with cors: true through collection and rendering into the
// config Secret — including that the API server actually persists the field
// (a stale or unsynced CRD would silently drop it).
func TestIntegration_StaticClientCORSOrigin(t *testing.T) {
	ns := "it-sc-cors"
	createNamespace(t, ns)
	inst := createInstallation(t, ns, "dex", []string{"*"}) // no spec.web at all

	sc := corsStaticClient("dtrack", ns, inst, []string{
		"https://dtrack.example.com/static/oidc-callback.html",
		"http://localhost:8000/cb", // loopback: not a browser origin
	})
	if err := k8sClient.Create(context.Background(), sc); err != nil {
		t.Fatalf("create cors static client: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), sc) })

	// Schema round-trip: the API server must persist spec.cors.
	var stored dexv1.DexStaticClient
	if err := k8sClient.Get(context.Background(),
		client.ObjectKey{Namespace: ns, Name: sc.Name}, &stored); err != nil {
		t.Fatalf("get static client: %v", err)
	}
	if !stored.Spec.CORS {
		t.Fatal("spec.cors was not persisted by the API server — CRD out of sync?")
	}

	// The derived origin creates the web block; the loopback URI is skipped.
	eventually(t, func() bool {
		return originsEqual(configAllowedOrigins(t, ns, inst.Spec.ConfigSecretName),
			[]string{"https://dtrack.example.com"})
	}, "derived CORS origin not rendered into config.yaml")
}

// TestIntegration_StaticClientCORSUnionAndRemoval verifies that derived origins
// are appended after the installation's authored list (which keeps its order)
// and that clearing the flag removes the origin again while the client — and
// its redirect URI — stay in the config.
func TestIntegration_StaticClientCORSUnionAndRemoval(t *testing.T) {
	ns := "it-sc-cors-union"
	createNamespace(t, ns)
	inst := createInstallation(t, ns, "dex", []string{"*"})

	// Add an authored origin list to the installation.
	mutateResource(t, inst, func(i *dexv1.DexInstallation) {
		i.Spec.Web = &dexv1.DexWebSpec{
			AllowedOrigins: []string{"https://zzz.example.com"},
		}
	})

	sc := corsStaticClient("spa", ns, inst, []string{"https://spa.example.com/cb"})
	if err := k8sClient.Create(context.Background(), sc); err != nil {
		t.Fatalf("create cors static client: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), sc) })

	// Authored entry keeps its position, the derived origin is appended.
	eventually(t, func() bool {
		return originsEqual(configAllowedOrigins(t, ns, inst.Spec.ConfigSecretName),
			[]string{"https://zzz.example.com", "https://spa.example.com"})
	}, "derived origin not appended after the authored allowedOrigins list")

	// Clearing the flag must drop the origin on the next render.
	mutateResource(t, sc, func(c *dexv1.DexStaticClient) { c.Spec.CORS = false })

	eventually(t, func() bool {
		return originsEqual(configAllowedOrigins(t, ns, inst.Spec.ConfigSecretName),
			[]string{"https://zzz.example.com"})
	}, "derived origin was not removed after clearing spec.cors")

	// The client itself must still be rendered — only the origin went away.
	s := getSecret(ns, inst.Spec.ConfigSecretName)
	if s == nil || !strings.Contains(string(s.Data["config.yaml"]), "https://spa.example.com/cb") {
		t.Error("clearing spec.cors must not remove the client's redirect URI from the config")
	}
}

// TestIntegration_StaticClientCORSForbiddenNamespace verifies the security
// claim in SECURITY_ARCHITECTURE.md: allowedNamespaces bounds origin
// registration exactly like it bounds client registration.  The allowed client
// acts as the control — without it the negative assertion could pass simply
// because nothing was ever rendered.
func TestIntegration_StaticClientCORSForbiddenNamespace(t *testing.T) {
	nsInst := "it-sc-cors-inst"
	nsForbidden := "it-sc-cors-forbidden"
	for _, ns := range []string{nsInst, nsForbidden} {
		createNamespace(t, ns)
	}
	inst := createInstallation(t, nsInst, "dex", []string{nsInst}) // nsForbidden excluded

	allowed := corsStaticClient("allowed-spa", nsInst, inst, []string{"https://allowed.example.com/cb"})
	if err := k8sClient.Create(context.Background(), allowed); err != nil {
		t.Fatalf("create allowed cors client: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), allowed) })

	rogue := corsStaticClient("rogue-spa", nsForbidden, inst, []string{"https://rogue.example.com/cb"})
	if err := k8sClient.Create(context.Background(), rogue); err != nil {
		t.Fatalf("create rogue cors client: %v", err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), rogue) })

	// The rogue client must be rejected with Ready=False, proving the
	// reconciler saw and evaluated it.
	eventually(t, func() bool {
		var updated dexv1.DexStaticClient
		if err := k8sClient.Get(context.Background(),
			client.ObjectKey{Namespace: nsForbidden, Name: rogue.Name}, &updated); err != nil {
			return false
		}
		cond := findCondition(updated.Status.Conditions, dexv1.ConditionTypeReady)
		return cond != nil && cond.Status == metav1.ConditionFalse
	}, "client in forbidden namespace should have Ready=False")

	// Control: the allowed client's origin must be rendered.
	eventually(t, func() bool {
		return hasOrigin(configAllowedOrigins(t, nsInst, inst.Spec.ConfigSecretName),
			"https://allowed.example.com")
	}, "allowed client's CORS origin missing — negative assertion would be vacuous")

	// The rogue origin must never appear.
	if origins := configAllowedOrigins(t, nsInst, inst.Spec.ConfigSecretName); hasOrigin(origins, "https://rogue.example.com") {
		t.Errorf("client from a non-allowlisted namespace registered a CORS origin: %v", origins)
	}
}
