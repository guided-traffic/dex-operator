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
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
	intbuilder "github.com/guided-traffic/dex-operator/internal/builder"
)

// ownNamespaceFixture is an installation "dex/main" whose connectors and
// static client all live in its own namespace, as in every README example.
type ownNamespaceFixture struct {
	inst    *dexv1.DexInstallation
	secrets []*corev1.Secret
	oidc    []dexv1.DexOIDCConnector
	github  []dexv1.DexGitHubConnector
	local   []dexv1.DexLocalConnector
	clients []dexv1.DexStaticClient
}

func newOwnNamespaceFixture(allowedNamespaces []string) ownNamespaceFixture {
	ref := dexv1.InstallationRef{Name: "main", Namespace: "dex"}
	creds := func(name string) *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "dex"},
			Data: map[string][]byte{
				"client-id":     []byte(name + "-id"),
				"client-secret": []byte(name + "-secret"),
			},
		}
	}
	oidc := func(name string) dexv1.DexOIDCConnector {
		return dexv1.DexOIDCConnector{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "dex"},
			Spec: dexv1.DexOIDCConnectorSpec{
				InstallationRef: ref,
				DisplayName:     name,
				Issuer:          "https://" + name + ".example.com",
				ClientIDRef:     dexv1.SecretKeyRef{Name: name, Key: "client-id"},
				ClientSecretRef: dexv1.SecretKeyRef{Name: name, Key: "client-secret"},
			},
		}
	}

	return ownNamespaceFixture{
		inst: &dexv1.DexInstallation{
			ObjectMeta: metav1.ObjectMeta{Name: "main", Namespace: "dex"},
			Spec: dexv1.DexInstallationSpec{
				Issuer:            "https://dex.example.com",
				ConfigSecretName:  "dex-config",
				EnvSecretName:     "dex-env",
				AllowedNamespaces: allowedNamespaces,
				Storage:           dexv1.DexStorageSpec{Type: dexv1.StorageKubernetes},
			},
		},
		secrets: []*corev1.Secret{creds("keycloak"), creds("okta"), creds("github")},
		oidc:    []dexv1.DexOIDCConnector{oidc("keycloak"), oidc("okta")},
		github: []dexv1.DexGitHubConnector{{
			ObjectMeta: metav1.ObjectMeta{Name: "github", Namespace: "dex"},
			Spec: dexv1.DexGitHubConnectorSpec{
				InstallationRef: ref,
				DisplayName:     "GitHub",
				ClientIDRef:     dexv1.SecretKeyRef{Name: "github", Key: "client-id"},
				ClientSecretRef: dexv1.SecretKeyRef{Name: "github", Key: "client-secret"},
			},
		}},
		local: []dexv1.DexLocalConnector{{
			ObjectMeta: metav1.ObjectMeta{Name: "local", Namespace: "dex"},
			Spec:       dexv1.DexLocalConnectorSpec{InstallationRef: ref, DisplayName: "Email"},
		}},
		clients: []dexv1.DexStaticClient{{
			ObjectMeta: metav1.ObjectMeta{Name: "cli", Namespace: "dex"},
			Spec: dexv1.DexStaticClientSpec{
				InstallationRef: ref,
				DisplayName:     "CLI",
				ClientID:        "cli",
				Public:          true,
			},
		}},
	}
}

func (f ownNamespaceFixture) objects() []client.Object {
	objs := make([]client.Object, 0, 1+len(f.secrets)+len(f.oidc)+len(f.github)+len(f.local)+len(f.clients))
	objs = append(objs, f.inst)
	for _, s := range f.secrets {
		objs = append(objs, s)
	}
	for i := range f.oidc {
		objs = append(objs, &f.oidc[i])
	}
	for i := range f.github {
		objs = append(objs, &f.github[i])
	}
	for i := range f.local {
		objs = append(objs, &f.local[i])
	}
	for i := range f.clients {
		objs = append(objs, &f.clients[i])
	}
	return objs
}

// unfilteredConfig renders the fixture with every child admitted, which is
// what the operator rendered before allowedConnectorNamespaces existed
// whenever allowedNamespaces admitted the installation's own namespace.
func (f ownNamespaceFixture) unfilteredConfig(t *testing.T) []byte {
	t.Helper()
	secrets := make(map[string]*corev1.Secret, len(f.secrets))
	for _, s := range f.secrets {
		secrets[s.Name] = s
	}
	out, err := intbuilder.Build(context.Background(), intbuilder.Input{
		Installation: f.inst,
		Connectors: intbuilder.ConnectorSet{
			OIDC:   f.oidc,
			GitHub: f.github,
			Local:  f.local,
		},
		StaticClients: f.clients,
		Secrets: func(_ context.Context, ns string, ref dexv1.SecretKeyRef) (string, error) {
			s, ok := secrets[ref.Name]
			if !ok || s.Namespace != ns {
				return "", fmt.Errorf("secret %s/%s not found", ns, ref.Name)
			}
			return string(s.Data[ref.Key]), nil
		},
	})
	if err != nil {
		t.Fatalf("building reference config: %v", err)
	}
	return out.ConfigYAML
}

// TestReconcile_OwnNamespaceConnectorsRenderUnchanged guards the upgrade
// promise for allowedConnectorNamespaces: an installation whose connectors
// live in its own namespace and that sets only allowedNamespaces renders
// byte-identical config, so the upgrade causes no config diff and no dex
// rollout.
func TestReconcile_OwnNamespaceConnectorsRenderUnchanged(t *testing.T) {
	for _, allowed := range [][]string{{"*"}, {"dex"}, {"apps", "dex"}} {
		t.Run(strings.Join(allowed, ","), func(t *testing.T) {
			f := newOwnNamespaceFixture(allowed)
			r, c := newReconciler(t, f.objects()...)

			if _, err := r.Reconcile(context.Background(), ctrl.Request{
				NamespacedName: types.NamespacedName{Name: f.inst.Name, Namespace: f.inst.Namespace},
			}); err != nil {
				t.Fatalf("Reconcile returned error: %v", err)
			}

			var cfg corev1.Secret
			if err := c.Get(context.Background(), types.NamespacedName{
				Namespace: f.inst.Namespace, Name: f.inst.Spec.ConfigSecretName,
			}, &cfg); err != nil {
				t.Fatalf("config secret not found: %v", err)
			}

			want := f.unfilteredConfig(t)
			if got := cfg.Data["config.yaml"]; !bytes.Equal(got, want) {
				t.Errorf("rendered config differs from the unfiltered render\n--- got ---\n%s\n--- want ---\n%s", got, want)
			}

			var updated dexv1.DexInstallation
			if err := c.Get(context.Background(), client.ObjectKeyFromObject(f.inst), &updated); err != nil {
				t.Fatalf("fetching installation: %v", err)
			}
			if updated.Status.ConnectorCount != 4 {
				t.Errorf("ConnectorCount = %d; want 4", updated.Status.ConnectorCount)
			}
		})
	}
}
