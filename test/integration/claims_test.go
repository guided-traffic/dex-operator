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

// createClaimsClient creates a confidential DexStaticClient ns/name whose
// Secret ns/name holds clientID, for the installation inst.
func createClaimsClient(t *testing.T, inst *dexv1.DexInstallation, ns, name, clientID string, peers ...string) *dexv1.DexStaticClient {
	t.Helper()
	createSecret(t, ns, name, map[string][]byte{
		"client-id":     []byte(clientID),
		"client-secret": []byte(ns + "-" + name + "-secret"),
	})
	sc := &dexv1.DexStaticClient{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: dexv1.DexStaticClientSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: inst.Namespace},
			DisplayName:     name,
			RedirectURIs:    []string{"https://" + name + "." + ns + ".example.com/callback"},
			SecretRef:       &dexv1.StaticClientSecretRef{Name: name},
			TrustedPeers:    peers,
		},
	}
	if err := k8sClient.Create(context.Background(), sc); err != nil {
		t.Fatalf("create static client %s/%s: %v", ns, name, err)
	}
	t.Cleanup(func() { _ = k8sClient.Delete(context.Background(), sc) })
	return sc
}

// staticClient fetches a DexStaticClient or returns nil.
func staticClient(ns, name string) *dexv1.DexStaticClient {
	var sc dexv1.DexStaticClient
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: name}, &sc); err != nil {
		return nil
	}
	return &sc
}

// clientCondition returns the condition condType of a DexStaticClient.
func clientCondition(ns, name, condType string) *metav1.Condition {
	sc := staticClient(ns, name)
	if sc == nil {
		return nil
	}
	return findCondition(sc.Status.Conditions, condType)
}

// installation fetches a DexInstallation or returns nil.
func installation(ns, name string) *dexv1.DexInstallation {
	var inst dexv1.DexInstallation
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: name}, &inst); err != nil {
		return nil
	}
	return &inst
}

// TestIntegration_DuplicateClientID walks two tenant clients through a
// contest for one ID: both are left out and report DuplicateID, the
// installation lists both; deleting one makes the other render and turn
// Ready=True without any event on itself, through the installation watch.
func TestIntegration_DuplicateClientID(t *testing.T) {
	instNS, nsA, nsB := "it-dup-inst", "it-dup-a", "it-dup-b"
	for _, ns := range []string{instNS, nsA, nsB} {
		createNamespace(t, ns)
	}
	inst := createInstallation(t, instNS, "dex", []string{"*"})

	createClaimsClient(t, inst, nsA, "app", "it-dup-shared")
	scB := createClaimsClient(t, inst, nsB, "app", "it-dup-shared")

	for _, ns := range []string{nsA, nsB} {
		eventually(t, func() bool {
			cond := clientCondition(ns, "app", dexv1.ConditionTypeReady)
			return cond != nil && cond.Status == metav1.ConditionFalse && cond.Reason == dexv1.RejectionReasonDuplicateID
		}, ns+"/app not DuplicateID")
	}
	eventually(t, func() bool {
		i := installation(instNS, inst.Name)
		if i == nil || len(i.Status.RejectedChildren) != 2 {
			return false
		}
		cond := findCondition(i.Status.Conditions, dexv1.ConditionTypeChildrenRejected)
		return i.Status.RejectedChildren[0].ID == "it-dup-shared" &&
			i.Status.RejectedChildren[1].ID == "it-dup-shared" &&
			cond != nil && cond.Status == metav1.ConditionTrue
	}, "installation does not list both claimants")
	if s := getSecret(instNS, inst.Spec.ConfigSecretName); s == nil || strings.Contains(string(s.Data["config.yaml"]), "it-dup-shared") {
		t.Fatal("contested ID is in config.yaml")
	}

	if err := k8sClient.Delete(context.Background(), scB); err != nil {
		t.Fatalf("delete %s/app: %v", nsB, err)
	}
	eventually(t, func() bool {
		cond := clientCondition(nsA, "app", dexv1.ConditionTypeReady)
		return cond != nil && cond.Status == metav1.ConditionTrue
	}, nsA+"/app not Ready=True after the other claimant left")
	eventually(t, func() bool {
		i := installation(instNS, inst.Name)
		s := getSecret(instNS, inst.Spec.ConfigSecretName)
		return i != nil && len(i.Status.RejectedChildren) == 0 &&
			s != nil && strings.Contains(string(s.Data["config.yaml"]), "it-dup-shared")
	}, "the remaining claimant does not render")
}

// TestIntegration_ClientIDSurvivesMissingSecret verifies that a rendered
// client records status.clientID, and that after its Secret is deleted a
// client of another namespace claiming that ID is DuplicateID and the ID is
// not in the config.
func TestIntegration_ClientIDSurvivesMissingSecret(t *testing.T) {
	instNS, nsA, nsB := "it-cid-inst", "it-cid-a", "it-cid-b"
	for _, ns := range []string{instNS, nsA, nsB} {
		createNamespace(t, ns)
	}
	inst := createInstallation(t, instNS, "dex", []string{"*"})
	createClaimsClient(t, inst, nsA, "app", "it-cid-app")

	eventually(t, func() bool {
		sc := staticClient(nsA, "app")
		return sc != nil && sc.Status.ClientID == "it-cid-app"
	}, "status.clientID not recorded")

	var secret = getSecret(nsA, "app")
	if err := k8sClient.Delete(context.Background(), secret); err != nil {
		t.Fatalf("delete secret: %v", err)
	}
	createClaimsClient(t, inst, nsB, "rogue", "it-cid-app")

	eventually(t, func() bool {
		cond := clientCondition(nsB, "rogue", dexv1.ConditionTypeReady)
		return cond != nil && cond.Status == metav1.ConditionFalse && cond.Reason == dexv1.RejectionReasonDuplicateID
	}, "the second claimant is not DuplicateID")
	eventually(t, func() bool {
		cond := clientCondition(nsA, "app", dexv1.ConditionTypeReady)
		return cond != nil && cond.Status == metav1.ConditionFalse && cond.Reason == dexv1.RejectionReasonBuildFailed
	}, "the client with the missing Secret is not BuildFailed")
	if s := getSecret(instNS, inst.Spec.ConfigSecretName); s == nil || strings.Contains(string(s.Data["config.yaml"]), "it-cid-app") {
		t.Fatal("the ID renders while its holder's Secret is missing")
	}
}

// TestIntegration_TrustedPeerInOtherNamespace verifies that a trustedPeers
// entry held in another namespace is dropped and reported on the trusting
// client, and comes back once a client of its own namespace holds the ID.
func TestIntegration_TrustedPeerInOtherNamespace(t *testing.T) {
	instNS, nsA, nsB := "it-peer-inst", "it-peer-a", "it-peer-b"
	for _, ns := range []string{instNS, nsA, nsB} {
		createNamespace(t, ns)
	}
	inst := createInstallation(t, instNS, "dex", []string{"*"})
	createClaimsClient(t, inst, nsA, "web", "it-peer-web", "it-peer-cli")
	peerB := createClaimsClient(t, inst, nsB, "cli", "it-peer-cli")

	eventually(t, func() bool {
		cond := clientCondition(nsA, "web", dexv1.ConditionTypeTrustedPeersDropped)
		return cond != nil && cond.Status == metav1.ConditionTrue && strings.Contains(cond.Message, `"it-peer-cli"`)
	}, "TrustedPeersDropped not True on the trusting client")
	if cond := clientCondition(nsA, "web", dexv1.ConditionTypeReady); cond == nil || cond.Status != metav1.ConditionTrue {
		t.Errorf("trusting client Ready = %+v; want True", cond)
	}

	// The peer moves into the trusting client's namespace.
	if err := k8sClient.Delete(context.Background(), peerB); err != nil {
		t.Fatalf("delete %s/cli: %v", nsB, err)
	}
	createClaimsClient(t, inst, nsA, "cli", "it-peer-cli")

	eventually(t, func() bool {
		cond := clientCondition(nsA, "web", dexv1.ConditionTypeTrustedPeersDropped)
		return cond != nil && cond.Status == metav1.ConditionFalse
	}, "TrustedPeersDropped not cleared")
	eventually(t, func() bool {
		s := getSecret(instNS, inst.Spec.ConfigSecretName)
		return s != nil && strings.Contains(string(s.Data["config.yaml"]), "trustedPeers:\n        - it-peer-cli")
	}, "trustedPeers entry not rendered")
}
