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
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// e2eRenderedClient is the part of a rendered staticClients entry the
// claim tests look at.
type e2eRenderedClient struct {
	ID           string   `yaml:"id"`
	SecretEnv    string   `yaml:"secretEnv"`
	RedirectURIs []string `yaml:"redirectURIs"`
	TrustedPeers []string `yaml:"trustedPeers"`
}

// e2eRenderedClients parses the staticClients of the installation's config
// Secret, or returns nil when it does not exist yet.
func e2eRenderedClients(ns string) []e2eRenderedClient {
	s := e2eGetSecret(ns, "dex-config")
	if s == nil {
		return nil
	}
	var cfg struct {
		StaticClients []e2eRenderedClient `yaml:"staticClients"`
	}
	if err := yaml.Unmarshal(s.Data["config.yaml"], &cfg); err != nil {
		return nil
	}
	return cfg.StaticClients
}

// e2eClaimsInstallation creates the installation "dex" in ns, admitting
// static clients from every namespace.
func e2eClaimsInstallation(t *testing.T, ns string) {
	t.Helper()
	inst := &dexv1.DexInstallation{
		ObjectMeta: metav1.ObjectMeta{Name: "dex", Namespace: ns},
		Spec: dexv1.DexInstallationSpec{
			Issuer:            "https://dex." + ns + ".example.com",
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
}

// e2eClaimsClient creates a confidential DexStaticClient ns/name with the
// client ID clientID for the installation instNS/dex.
func e2eClaimsClient(t *testing.T, instNS, ns, name, clientID string, peers ...string) {
	t.Helper()
	e2eCreateSecret(t, ns, name, map[string][]byte{
		"client-id":     []byte(clientID),
		"client-secret": []byte(ns + "-" + name + "-secret"),
	})
	sc := &dexv1.DexStaticClient{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: dexv1.DexStaticClientSpec{
			InstallationRef: dexv1.InstallationRef{Name: "dex", Namespace: instNS},
			DisplayName:     name,
			RedirectURIs:    []string{"https://" + name + "." + ns + ".example.com/callback"},
			SecretRef:       &dexv1.StaticClientSecretRef{Name: name},
			TrustedPeers:    peers,
		},
	}
	if err := e2eClient.Create(context.Background(), sc); err != nil {
		t.Fatalf("create static client %s/%s: %v", ns, name, err)
	}
	t.Cleanup(func() { _ = e2eClient.Delete(context.Background(), sc) })
}

// e2eRejected returns "namespace/name reason" of every child the
// installation instNS/dex reports as rejected.
func e2eRejected(instNS string) []string {
	var inst dexv1.DexInstallation
	if err := e2eClient.Get(context.Background(), client.ObjectKey{Namespace: instNS, Name: "dex"}, &inst); err != nil {
		return nil
	}
	keys := make([]string, 0, len(inst.Status.RejectedChildren))
	for _, rc := range inst.Status.RejectedChildren {
		keys = append(keys, rc.Namespace+"/"+rc.Name+" "+rc.Reason)
	}
	return keys
}

// TestE2E_TenantReusesPlatformClientID verifies that a tenant client reusing
// the ID of a client in the installation's namespace does not replace it:
// the rendered config holds only the original.
func TestE2E_TenantReusesPlatformClientID(t *testing.T) {
	instNS, tenantNS := "e2e-claim-platform", "e2e-claim-platform-tenant"
	for _, ns := range []string{instNS, tenantNS} {
		e2eCreateNamespace(t, ns)
	}
	e2eClaimsInstallation(t, instNS)
	e2eClaimsClient(t, instNS, instNS, "argocd", "argocd")
	e2eClaimsClient(t, instNS, tenantNS, "rogue", "argocd")

	e2eEventually(t, func() bool {
		return reflect.DeepEqual(e2eRejected(instNS), []string{tenantNS + "/rogue " + dexv1.RejectionReasonDuplicateID})
	}, "tenant client not rejected as DuplicateID")

	clients := e2eRenderedClients(instNS)
	if len(clients) != 1 || clients[0].ID != "argocd" ||
		!reflect.DeepEqual(clients[0].RedirectURIs, []string{"https://argocd." + instNS + ".example.com/callback"}) {
		t.Errorf("rendered staticClients = %+v; want the platform client only", clients)
	}

	// status.clientID is patched under the chart's ClusterRole, for the
	// rejected claimant too.
	for _, key := range []client.ObjectKey{{Namespace: instNS, Name: "argocd"}, {Namespace: tenantNS, Name: "rogue"}} {
		e2eEventually(t, func() bool {
			var sc dexv1.DexStaticClient
			return e2eClient.Get(context.Background(), key, &sc) == nil && sc.Status.ClientID == "argocd"
		}, key.String()+" status.clientID not recorded")
	}
}

// TestE2E_TwoTenantsShareClientID verifies that two tenant clients with one
// ID both stay out of the config.
func TestE2E_TwoTenantsShareClientID(t *testing.T) {
	instNS, nsA, nsB := "e2e-claim-tenants", "e2e-claim-tenants-a", "e2e-claim-tenants-b"
	for _, ns := range []string{instNS, nsA, nsB} {
		e2eCreateNamespace(t, ns)
	}
	e2eClaimsInstallation(t, instNS)
	e2eClaimsClient(t, instNS, nsA, "app", "shared-app")
	e2eClaimsClient(t, instNS, nsB, "app", "shared-app")

	e2eEventually(t, func() bool {
		return reflect.DeepEqual(e2eRejected(instNS), []string{
			nsA + "/app " + dexv1.RejectionReasonDuplicateID,
			nsB + "/app " + dexv1.RejectionReasonDuplicateID,
		})
	}, "both tenant clients not rejected as DuplicateID")

	for _, c := range e2eRenderedClients(instNS) {
		if c.ID == "shared-app" {
			t.Errorf("contested ID rendered: %+v", c)
		}
	}
}

// TestE2E_PlatformClientTrustsTenantHeldPeer verifies that a platform client
// trusting headlamp does not trust a tenant client that registers headlamp.
func TestE2E_PlatformClientTrustsTenantHeldPeer(t *testing.T) {
	instNS, tenantNS := "e2e-claim-peer", "e2e-claim-peer-tenant"
	for _, ns := range []string{instNS, tenantNS} {
		e2eCreateNamespace(t, ns)
	}
	e2eClaimsInstallation(t, instNS)
	e2eClaimsClient(t, instNS, instNS, "kubernetes", "kubernetes", "kubectl", "headlamp")
	e2eClaimsClient(t, instNS, instNS, "kubectl", "kubectl")
	e2eClaimsClient(t, instNS, tenantNS, "headlamp", "headlamp")

	e2eEventually(t, func() bool {
		var ids []string
		var peers []string
		for _, c := range e2eRenderedClients(instNS) {
			ids = append(ids, c.ID)
			if c.ID == "kubernetes" {
				peers = c.TrustedPeers
			}
		}
		return len(ids) == 3 && reflect.DeepEqual(peers, []string{"kubectl"})
	}, "platform client's trustedPeers still name the tenant-held peer")

	var inst dexv1.DexInstallation
	if err := e2eClient.Get(context.Background(), client.ObjectKey{Namespace: instNS, Name: "dex"}, &inst); err != nil {
		t.Fatalf("get installation: %v", err)
	}
	want := []dexv1.DroppedTrustedPeer{{Namespace: instNS, Name: "kubernetes", PeerID: "headlamp"}}
	if !reflect.DeepEqual(inst.Status.DroppedTrustedPeers, want) {
		t.Errorf("droppedTrustedPeers = %+v; want %+v", inst.Status.DroppedTrustedPeers, want)
	}
}
