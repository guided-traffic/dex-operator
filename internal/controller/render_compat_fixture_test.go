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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// compatFixture is a multi-tenant installation "dex/main" without duplicate
// IDs, without failing children and with trustedPeers held in the trusting
// client's own namespace only.  testdata/render-compat-v2.3.0/ holds what
// v2.3.0 rendered for it; the fixture uses no field newer than v2.3.0, so
// the golden files were produced by running that release's reconciler on
// exactly these objects.
func compatFixture() []client.Object {
	ref := dexv1.InstallationRef{Name: "main", Namespace: "dex"}
	created := metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	meta := func(ns, name string) metav1.ObjectMeta {
		return metav1.ObjectMeta{Namespace: ns, Name: name, CreationTimestamp: created}
	}
	creds := func(ns, name string) *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: meta(ns, name),
			Data: map[string][]byte{
				"client-id":     []byte(name + "-id"),
				"client-secret": []byte(name + "-secret"),
				"password":      []byte(name + "-password"),
			},
		}
	}
	confidential := func(ns, name string, peers ...string) *dexv1.DexStaticClient {
		return &dexv1.DexStaticClient{
			ObjectMeta: meta(ns, name),
			Spec: dexv1.DexStaticClientSpec{
				InstallationRef: ref,
				DisplayName:     name,
				SecretRef:       &dexv1.StaticClientSecretRef{Name: name, ClientIDKey: "client-id", ClientSecretKey: "client-secret"},
				RedirectURIs:    []string{"https://" + name + "." + ns + ".example.com/callback"},
				TrustedPeers:    peers,
				CORS:            true,
			},
		}
	}
	public := func(ns, name string) *dexv1.DexStaticClient {
		return &dexv1.DexStaticClient{
			ObjectMeta: meta(ns, name),
			Spec: dexv1.DexStaticClientSpec{
				InstallationRef: ref, DisplayName: name, ClientID: name, Public: true,
			},
		}
	}
	oidc := func(ns, name, id string) *dexv1.DexOIDCConnector {
		return &dexv1.DexOIDCConnector{
			ObjectMeta: meta(ns, name),
			Spec: dexv1.DexOIDCConnectorSpec{
				InstallationRef: ref,
				ID:              id,
				DisplayName:     name,
				Issuer:          "https://" + name + ".example.com",
				ClientIDRef:     dexv1.SecretKeyRef{Name: name, Key: "client-id"},
				ClientSecretRef: dexv1.SecretKeyRef{Name: name, Key: "client-secret"},
			},
		}
	}

	return []client.Object{
		&dexv1.DexInstallation{
			ObjectMeta: meta("dex", "main"),
			Spec: dexv1.DexInstallationSpec{
				Issuer:                     "https://dex.example.com",
				ConfigSecretName:           "dex-config",
				EnvSecretName:              "dex-env",
				AllowedNamespaces:          []string{"*"},
				AllowedConnectorNamespaces: []string{"dex", "idp"},
				Storage: dexv1.DexStorageSpec{
					Type: dexv1.StoragePostgres,
					Postgres: &dexv1.DexPostgresStorageSpec{
						Host: "db.example.com", Database: "dex", User: "dex",
						PasswordRef: &dexv1.SecretKeyRef{Name: "postgres", Key: "password"},
					},
				},
				Web: &dexv1.DexWebSpec{AllowedOrigins: []string{"https://portal.example.com"}},
			},
		},
		creds("dex", "postgres"),
		creds("dex", "okta"), oidc("dex", "okta", ""),
		creds("idp", "keycloak"), oidc("idp", "keycloak", "corp-sso"),
		creds("dex", "github"),
		&dexv1.DexGitHubConnector{
			ObjectMeta: meta("dex", "github"),
			Spec: dexv1.DexGitHubConnectorSpec{
				InstallationRef: ref,
				DisplayName:     "GitHub",
				ClientIDRef:     dexv1.SecretKeyRef{Name: "github", Key: "client-id"},
				ClientSecretRef: dexv1.SecretKeyRef{Name: "github", Key: "client-secret"},
			},
		},
		creds("dex", "ldap"),
		&dexv1.DexLDAPConnector{
			ObjectMeta: meta("dex", "ldap"),
			Spec: dexv1.DexLDAPConnectorSpec{
				InstallationRef: ref,
				DisplayName:     "LDAP",
				Host:            "ldap.example.com:636",
				BindDN:          "cn=dex,dc=example,dc=com",
				BindPWRef:       &dexv1.SecretKeyRef{Name: "ldap", Key: "password"},
				UserSearch:      dexv1.LDAPUserSearch{BaseDN: "ou=people,dc=example,dc=com", Username: "uid"},
			},
		},
		&dexv1.DexLocalConnector{
			ObjectMeta: meta("dex", "local"),
			Spec:       dexv1.DexLocalConnectorSpec{InstallationRef: ref, DisplayName: "Email"},
		},
		creds("dex", "kubernetes"), confidential("dex", "kubernetes", "kubectl"),
		public("dex", "kubectl"),
		creds("team-a", "grafana"), confidential("team-a", "grafana", "grafana-cli"),
		public("team-a", "grafana-cli"),
		creds("team-b", "argocd"), confidential("team-b", "argocd"),
	}
}
