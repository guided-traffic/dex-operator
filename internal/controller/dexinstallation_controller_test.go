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
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
	"github.com/guided-traffic/dex-operator/internal/controller"
)

// newTestScheme builds a runtime.Scheme with all required types registered.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		t.Fatalf("adding client-go scheme: %v", err)
	}
	if err := dexv1.AddToScheme(s); err != nil {
		t.Fatalf("adding dex scheme: %v", err)
	}
	return s
}

// minimalInstallation creates a DexInstallation suitable for controller tests.
func minimalInstallation() *dexv1.DexInstallation {
	return &dexv1.DexInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-installation",
			Namespace: "dex",
		},
		Spec: dexv1.DexInstallationSpec{
			Issuer:            "https://dex.example.com",
			ConfigSecretName:  "dex-config",
			EnvSecretName:     "dex-env",
			AllowedNamespaces: []string{"*"},
			Storage: dexv1.DexStorageSpec{
				Type: dexv1.StorageKubernetes,
			},
		},
	}
}

// newReconciler builds a DexInstallationReconciler backed by a fake client.
func newReconciler(t *testing.T, objs ...client.Object) (*controller.DexInstallationReconciler, client.Client) {
	t.Helper()
	scheme := newTestScheme(t)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(&dexv1.DexInstallation{}).
		WithIndex(&dexv1.DexOIDCConnector{}, controller.InstallationRefIndexField, controller.InstallationRefIndexFunc).
		WithIndex(&dexv1.DexLDAPConnector{}, controller.InstallationRefIndexField, controller.InstallationRefIndexFunc).
		WithIndex(&dexv1.DexGitHubConnector{}, controller.InstallationRefIndexField, controller.InstallationRefIndexFunc).
		WithIndex(&dexv1.DexSAMLConnector{}, controller.InstallationRefIndexField, controller.InstallationRefIndexFunc).
		WithIndex(&dexv1.DexGitLabConnector{}, controller.InstallationRefIndexField, controller.InstallationRefIndexFunc).
		WithIndex(&dexv1.DexOAuth2Connector{}, controller.InstallationRefIndexField, controller.InstallationRefIndexFunc).
		WithIndex(&dexv1.DexGoogleConnector{}, controller.InstallationRefIndexField, controller.InstallationRefIndexFunc).
		WithIndex(&dexv1.DexLinkedInConnector{}, controller.InstallationRefIndexField, controller.InstallationRefIndexFunc).
		WithIndex(&dexv1.DexMicrosoftConnector{}, controller.InstallationRefIndexField, controller.InstallationRefIndexFunc).
		WithIndex(&dexv1.DexAuthProxyConnector{}, controller.InstallationRefIndexField, controller.InstallationRefIndexFunc).
		WithIndex(&dexv1.DexBitbucketConnector{}, controller.InstallationRefIndexField, controller.InstallationRefIndexFunc).
		WithIndex(&dexv1.DexLocalConnector{}, controller.InstallationRefIndexField, controller.InstallationRefIndexFunc).
		WithIndex(&dexv1.DexOpenShiftConnector{}, controller.InstallationRefIndexField, controller.InstallationRefIndexFunc).
		WithIndex(&dexv1.DexAtlassianCrowdConnector{}, controller.InstallationRefIndexField, controller.InstallationRefIndexFunc).
		WithIndex(&dexv1.DexGiteaConnector{}, controller.InstallationRefIndexField, controller.InstallationRefIndexFunc).
		WithIndex(&dexv1.DexKeystoneConnector{}, controller.InstallationRefIndexField, controller.InstallationRefIndexFunc).
		WithIndex(&dexv1.DexStaticClient{}, controller.InstallationRefIndexField, controller.InstallationRefIndexFunc).
		WithIndex(&dexv1.DexOIDCConnector{}, controller.SecretRefIndexField, controller.SecretRefIndexFunc).
		WithIndex(&dexv1.DexLDAPConnector{}, controller.SecretRefIndexField, controller.SecretRefIndexFunc).
		WithIndex(&dexv1.DexGitHubConnector{}, controller.SecretRefIndexField, controller.SecretRefIndexFunc).
		WithIndex(&dexv1.DexSAMLConnector{}, controller.SecretRefIndexField, controller.SecretRefIndexFunc).
		WithIndex(&dexv1.DexGitLabConnector{}, controller.SecretRefIndexField, controller.SecretRefIndexFunc).
		WithIndex(&dexv1.DexOAuth2Connector{}, controller.SecretRefIndexField, controller.SecretRefIndexFunc).
		WithIndex(&dexv1.DexGoogleConnector{}, controller.SecretRefIndexField, controller.SecretRefIndexFunc).
		WithIndex(&dexv1.DexLinkedInConnector{}, controller.SecretRefIndexField, controller.SecretRefIndexFunc).
		WithIndex(&dexv1.DexMicrosoftConnector{}, controller.SecretRefIndexField, controller.SecretRefIndexFunc).
		WithIndex(&dexv1.DexAuthProxyConnector{}, controller.SecretRefIndexField, controller.SecretRefIndexFunc).
		WithIndex(&dexv1.DexBitbucketConnector{}, controller.SecretRefIndexField, controller.SecretRefIndexFunc).
		WithIndex(&dexv1.DexLocalConnector{}, controller.SecretRefIndexField, controller.SecretRefIndexFunc).
		WithIndex(&dexv1.DexOpenShiftConnector{}, controller.SecretRefIndexField, controller.SecretRefIndexFunc).
		WithIndex(&dexv1.DexAtlassianCrowdConnector{}, controller.SecretRefIndexField, controller.SecretRefIndexFunc).
		WithIndex(&dexv1.DexGiteaConnector{}, controller.SecretRefIndexField, controller.SecretRefIndexFunc).
		WithIndex(&dexv1.DexKeystoneConnector{}, controller.SecretRefIndexField, controller.SecretRefIndexFunc).
		WithIndex(&dexv1.DexStaticClient{}, controller.SecretRefIndexField, controller.SecretRefIndexFunc).
		Build()

	r := &controller.DexInstallationReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}
	return r, fakeClient
}

// TestReconcile_Minimal verifies that a minimal DexInstallation creates both secrets.
func TestReconcile_Minimal(t *testing.T) {
	inst := minimalInstallation()
	r, c := newReconciler(t, inst)

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: inst.Name, Namespace: inst.Namespace},
	})
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	// Config secret must exist.
	var configSecret corev1.Secret
	if err := c.Get(context.Background(), types.NamespacedName{
		Namespace: inst.Namespace,
		Name:      inst.Spec.ConfigSecretName,
	}, &configSecret); err != nil {
		t.Errorf("config secret not created: %v", err)
	}

	if _, ok := configSecret.Data["config.yaml"]; !ok {
		t.Error("config.yaml key missing from config secret")
	}

	// Env secret must exist.
	var envSecret corev1.Secret
	if err := c.Get(context.Background(), types.NamespacedName{
		Namespace: inst.Namespace,
		Name:      inst.Spec.EnvSecretName,
	}, &envSecret); err != nil {
		t.Errorf("env secret not created: %v", err)
	}

	// Status must be updated.
	var updated dexv1.DexInstallation
	if err := c.Get(context.Background(), types.NamespacedName{
		Name:      inst.Name,
		Namespace: inst.Namespace,
	}, &updated); err != nil {
		t.Fatalf("fetching updated installation: %v", err)
	}

	readyCondition := findCondition(updated.Status.Conditions, dexv1.ConditionTypeReady)
	if readyCondition == nil {
		t.Fatal("Ready condition not set on DexInstallation status")
	}
	if readyCondition.Status != metav1.ConditionTrue {
		t.Errorf("Ready condition = %v; want True (message: %s)", readyCondition.Status, readyCondition.Message)
	}
}

// TestReconcile_NotFound verifies that a missing DexInstallation is handled gracefully.
func TestReconcile_NotFound(t *testing.T) {
	r, _ := newReconciler(t) // no objects

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "nonexistent", Namespace: "default"},
	})
	if err != nil {
		t.Errorf("expected no error for missing installation, got %v", err)
	}
}

// TestReconcile_WithOIDCConnector verifies that an OIDC connector is included
// in the config when it references the installation and its credentials secret exists.
func TestReconcile_WithOIDCConnector(t *testing.T) {
	inst := minimalInstallation()

	credSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "oidc-creds", Namespace: "dex"},
		Data: map[string][]byte{
			"client-id":     []byte("my-client-id"),
			"client-secret": []byte("my-client-secret"),
		},
	}

	connector := &dexv1.DexOIDCConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "okta", Namespace: "dex"},
		Spec: dexv1.DexOIDCConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: inst.Namespace},
			DisplayName:     "Okta",
			Issuer:          "https://okta.example.com",
			ClientIDRef:     dexv1.SecretKeyRef{Name: "oidc-creds", Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "oidc-creds", Key: "client-secret"},
		},
	}

	r, c := newReconciler(t, inst, credSecret, connector)

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: inst.Name, Namespace: inst.Namespace},
	})
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	var configSecret corev1.Secret
	if err := c.Get(context.Background(), types.NamespacedName{
		Namespace: inst.Namespace, Name: inst.Spec.ConfigSecretName,
	}, &configSecret); err != nil {
		t.Fatalf("config secret not found: %v", err)
	}

	yaml := string(configSecret.Data["config.yaml"])
	if yaml == "" {
		t.Fatal("config.yaml is empty")
	}
	if !contains(yaml, "oidc") {
		t.Errorf("config.yaml does not reference OIDC connector:\n%s", yaml)
	}

	// Status should report 1 connector.
	var updated dexv1.DexInstallation
	if err := c.Get(context.Background(), types.NamespacedName{
		Name: inst.Name, Namespace: inst.Namespace,
	}, &updated); err != nil {
		t.Fatalf("fetching updated installation: %v", err)
	}
	if updated.Status.ConnectorCount != 1 {
		t.Errorf("ConnectorCount = %d; want 1", updated.Status.ConnectorCount)
	}
}

// TestReconcile_SecretReuse verifies that a second reconcile does not recreate
// the secrets if nothing changed.
func TestReconcile_SecretReuse(t *testing.T) {
	inst := minimalInstallation()
	r, c := newReconciler(t, inst)

	reconcileOnce := func() {
		t.Helper()
		_, err := r.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{Name: inst.Name, Namespace: inst.Namespace},
		})
		if err != nil {
			t.Fatalf("Reconcile returned error: %v", err)
		}
	}

	reconcileOnce()

	// Capture resource version after first reconcile.
	var secretAfterFirst corev1.Secret
	if err := c.Get(context.Background(), types.NamespacedName{
		Namespace: inst.Namespace, Name: inst.Spec.ConfigSecretName,
	}, &secretAfterFirst); err != nil {
		t.Fatalf("secret not found after first reconcile: %v", err)
	}
	rvFirst := secretAfterFirst.ResourceVersion

	reconcileOnce()

	var secretAfterSecond corev1.Secret
	if err := c.Get(context.Background(), types.NamespacedName{
		Namespace: inst.Namespace, Name: inst.Spec.ConfigSecretName,
	}, &secretAfterSecond); err != nil {
		t.Fatalf("secret not found after second reconcile: %v", err)
	}

	// Resource version should be unchanged (no patch if data is identical).
	if secretAfterSecond.ResourceVersion != rvFirst {
		t.Errorf("config secret was re-patched on second reconcile (rv %s → %s)",
			rvFirst, secretAfterSecond.ResourceVersion)
	}
}

// ── child reconciler ─────────────────────────────────────────────────────────

// TestChildReconciler_NamespaceNotAllowed verifies that a connector in a
// forbidden namespace gets an Error condition on its status and is requeued
// after a long interval without returning an error (no stack trace).
func TestChildReconciler_NamespaceNotAllowed(t *testing.T) {
	inst := &dexv1.DexInstallation{
		ObjectMeta: metav1.ObjectMeta{Name: "test-inst", Namespace: "dex"},
		Spec: dexv1.DexInstallationSpec{
			Issuer:            "https://dex.example.com",
			ConfigSecretName:  "dex-config",
			EnvSecretName:     "dex-env",
			AllowedNamespaces: []string{"dex"}, // only "dex" namespace allowed
			Storage:           dexv1.DexStorageSpec{Type: dexv1.StorageKubernetes},
		},
	}

	connector := &dexv1.DexLocalConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "local-conn", Namespace: "forbidden-ns"},
		Spec: dexv1.DexLocalConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: "test-inst", Namespace: "dex"},
			DisplayName:     "Local",
		},
	}

	scheme := newTestScheme(t)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(inst, connector).
		WithStatusSubresource(&dexv1.DexLocalConnector{}).
		Build()

	r := &controller.GenericChildReconciler[*dexv1.DexLocalConnector, dexv1.DexLocalConnector]{
		Client: fakeClient,
		Scheme: scheme,
	}

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "local-conn", Namespace: "forbidden-ns"},
	})
	// Config errors must NOT be returned as errors (no stack trace / backoff).
	if err != nil {
		t.Fatalf("expected no error for config problem, got %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Error("expected non-zero RequeueAfter for config error")
	}

	var updated dexv1.DexLocalConnector
	if err := fakeClient.Get(context.Background(), types.NamespacedName{
		Name:      "local-conn",
		Namespace: "forbidden-ns",
	}, &updated); err != nil {
		t.Fatalf("fetching updated connector: %v", err)
	}

	readyCond := findCondition(updated.Status.Conditions, dexv1.ConditionTypeReady)
	if readyCond == nil {
		t.Fatal("Ready condition not set")
	}
	if readyCond.Status != metav1.ConditionFalse {
		t.Errorf("expected Ready=False for forbidden namespace, got %v", readyCond.Status)
	}
}

// TestChildReconciler_InstallationNotFound verifies that a connector referencing
// a non-existent DexInstallation gets a clean warning rather than a noisy error.
func TestChildReconciler_InstallationNotFound(t *testing.T) {
	connector := &dexv1.DexLocalConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "local-conn", Namespace: "iam"},
		Spec: dexv1.DexLocalConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: "nonexistent", Namespace: "dex"},
			DisplayName:     "Local",
		},
	}

	scheme := newTestScheme(t)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(connector).
		WithStatusSubresource(&dexv1.DexLocalConnector{}).
		Build()

	r := &controller.GenericChildReconciler[*dexv1.DexLocalConnector, dexv1.DexLocalConnector]{
		Client: fakeClient,
		Scheme: scheme,
	}

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "local-conn", Namespace: "iam"},
	})
	if err != nil {
		t.Fatalf("expected no error for missing installation, got %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Error("expected non-zero RequeueAfter for config error")
	}

	var updated dexv1.DexLocalConnector
	if err := fakeClient.Get(context.Background(), types.NamespacedName{
		Name: "local-conn", Namespace: "iam",
	}, &updated); err != nil {
		t.Fatalf("fetching updated connector: %v", err)
	}

	readyCond := findCondition(updated.Status.Conditions, dexv1.ConditionTypeReady)
	if readyCond == nil {
		t.Fatal("Ready condition not set")
	}
	if readyCond.Status != metav1.ConditionFalse {
		t.Errorf("expected Ready=False, got %v", readyCond.Status)
	}
	if !contains(readyCond.Message, "not found") {
		t.Errorf("expected message to mention 'not found', got %q", readyCond.Message)
	}
}

// ── secret watch ─────────────────────────────────────────────────────────────

// TestMapSecretToInstallation_OIDCConnector verifies that changing a Secret
// referenced by an OIDC connector results in a reconcile request for the
// owning DexInstallation.
func TestMapSecretToInstallation_OIDCConnector(t *testing.T) {
	inst := minimalInstallation()

	credSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "oidc-creds", Namespace: "dex"},
		Data: map[string][]byte{
			"client-id":     []byte("my-id"),
			"client-secret": []byte("my-secret"),
		},
	}

	connector := &dexv1.DexOIDCConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "okta", Namespace: "dex"},
		Spec: dexv1.DexOIDCConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: inst.Namespace},
			DisplayName:     "Okta",
			Issuer:          "https://okta.example.com",
			ClientIDRef:     dexv1.SecretKeyRef{Name: "oidc-creds", Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "oidc-creds", Key: "client-secret"},
		},
	}

	r, _ := newReconciler(t, inst, credSecret, connector)
	requests := r.MapSecretToInstallation(context.Background(), credSecret)

	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	if requests[0].Name != inst.Name || requests[0].Namespace != inst.Namespace {
		t.Errorf("expected request for %s/%s, got %s/%s",
			inst.Namespace, inst.Name, requests[0].Namespace, requests[0].Name)
	}
}

// TestMapSecretToInstallation_StaticClient verifies that changing a Secret
// referenced by a DexStaticClient triggers reconciliation.
func TestMapSecretToInstallation_StaticClient(t *testing.T) {
	inst := minimalInstallation()

	clientSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "grafana-creds", Namespace: "dex"},
		Data: map[string][]byte{
			"client-id":     []byte("grafana"),
			"client-secret": []byte("grafana-secret"),
		},
	}

	sc := &dexv1.DexStaticClient{
		ObjectMeta: metav1.ObjectMeta{Name: "grafana", Namespace: "dex"},
		Spec: dexv1.DexStaticClientSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: inst.Namespace},
			SecretRef:       dexv1.StaticClientSecretRef{Name: "grafana-creds"},
			DisplayName:     "Grafana",
			RedirectURIs:    []string{"https://grafana.example.com/callback"},
		},
	}

	r, _ := newReconciler(t, inst, clientSecret, sc)
	requests := r.MapSecretToInstallation(context.Background(), clientSecret)

	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	if requests[0].Name != inst.Name || requests[0].Namespace != inst.Namespace {
		t.Errorf("expected request for %s/%s, got %s/%s",
			inst.Namespace, inst.Name, requests[0].Namespace, requests[0].Name)
	}
}

// TestMapSecretToInstallation_UnreferencedSecret verifies that a Secret
// not referenced by any child object produces no reconcile requests.
func TestMapSecretToInstallation_UnreferencedSecret(t *testing.T) {
	inst := minimalInstallation()

	unrelatedSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "unrelated", Namespace: "dex"},
		Data:       map[string][]byte{"key": []byte("val")},
	}

	r, _ := newReconciler(t, inst, unrelatedSecret)
	requests := r.MapSecretToInstallation(context.Background(), unrelatedSecret)

	if len(requests) != 0 {
		t.Errorf("expected 0 requests for unreferenced secret, got %d", len(requests))
	}
}

// TestMapSecretToInstallation_MultipleConnectorsSameInstallation verifies
// deduplication: two connectors referencing the same secret and installation
// produce only one request.
func TestMapSecretToInstallation_MultipleConnectorsSameInstallation(t *testing.T) {
	inst := minimalInstallation()

	sharedSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "shared-creds", Namespace: "dex"},
		Data: map[string][]byte{
			"client-id":     []byte("id"),
			"client-secret": []byte("secret"),
		},
	}

	conn1 := &dexv1.DexOIDCConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "conn-1", Namespace: "dex"},
		Spec: dexv1.DexOIDCConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: inst.Namespace},
			DisplayName:     "C1",
			Issuer:          "https://idp1.example.com",
			ClientIDRef:     dexv1.SecretKeyRef{Name: "shared-creds", Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "shared-creds", Key: "client-secret"},
		},
	}
	conn2 := &dexv1.DexGitLabConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "conn-2", Namespace: "dex"},
		Spec: dexv1.DexGitLabConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: inst.Namespace},
			DisplayName:     "C2",
			ClientIDRef:     dexv1.SecretKeyRef{Name: "shared-creds", Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "shared-creds", Key: "client-secret"},
		},
	}

	r, _ := newReconciler(t, inst, sharedSecret, conn1, conn2)
	requests := r.MapSecretToInstallation(context.Background(), sharedSecret)

	if len(requests) != 1 {
		t.Fatalf("expected 1 deduplicated request, got %d", len(requests))
	}
}

// TestMapSecretToInstallation_DifferentNamespace verifies that a Secret in
// a different namespace than the connector does not trigger a reconcile.
func TestMapSecretToInstallation_DifferentNamespace(t *testing.T) {
	inst := minimalInstallation()

	// Connector is in namespace "apps" referencing a secret "creds" in "apps"
	connector := &dexv1.DexOIDCConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "oidc", Namespace: "apps"},
		Spec: dexv1.DexOIDCConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: inst.Namespace},
			DisplayName:     "OIDC",
			Issuer:          "https://idp.example.com",
			ClientIDRef:     dexv1.SecretKeyRef{Name: "creds", Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "creds", Key: "client-secret"},
		},
	}

	// Secret with SAME name but in "dex" namespace (different from connector's ns)
	secretInDex := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "creds", Namespace: "dex"},
		Data:       map[string][]byte{"client-id": []byte("id")},
	}

	r, _ := newReconciler(t, inst, connector, secretInDex)
	requests := r.MapSecretToInstallation(context.Background(), secretInDex)

	if len(requests) != 0 {
		t.Errorf("expected 0 requests (wrong namespace), got %d", len(requests))
	}
}

// TestReconcile_SecretRotationUpdatesEnv verifies the full flow: when a
// credential secret is rotated, the reconcile picks up the new value in the
// env secret.
func TestReconcile_SecretRotationUpdatesEnv(t *testing.T) {
	inst := minimalInstallation()

	credSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "oidc-creds", Namespace: "dex"},
		Data: map[string][]byte{
			"client-id":     []byte("my-client-id"),
			"client-secret": []byte("old-secret"),
		},
	}

	connector := &dexv1.DexOIDCConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "okta", Namespace: "dex"},
		Spec: dexv1.DexOIDCConnectorSpec{
			InstallationRef: dexv1.InstallationRef{Name: inst.Name, Namespace: inst.Namespace},
			DisplayName:     "Okta",
			Issuer:          "https://okta.example.com",
			ClientIDRef:     dexv1.SecretKeyRef{Name: "oidc-creds", Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "oidc-creds", Key: "client-secret"},
		},
	}

	r, c := newReconciler(t, inst, credSecret, connector)

	// First reconcile
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: inst.Name, Namespace: inst.Namespace},
	})
	if err != nil {
		t.Fatalf("first reconcile: %v", err)
	}

	// Verify env secret contains the old secret
	var envSecret corev1.Secret
	if err := c.Get(context.Background(), types.NamespacedName{
		Namespace: inst.Namespace, Name: inst.Spec.EnvSecretName,
	}, &envSecret); err != nil {
		t.Fatalf("env secret not found: %v", err)
	}

	// Rotate the credential
	credSecret.Data["client-secret"] = []byte("new-secret")
	if err := c.Update(context.Background(), credSecret); err != nil {
		t.Fatalf("updating secret: %v", err)
	}

	// Second reconcile (triggered by secret watch in production)
	_, err = r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: inst.Name, Namespace: inst.Namespace},
	})
	if err != nil {
		t.Fatalf("second reconcile: %v", err)
	}

	// Verify env secret now contains the new secret value
	if err := c.Get(context.Background(), types.NamespacedName{
		Namespace: inst.Namespace, Name: inst.Spec.EnvSecretName,
	}, &envSecret); err != nil {
		t.Fatalf("env secret not found after rotation: %v", err)
	}

	found := false
	for _, v := range envSecret.Data {
		if string(v) == "new-secret" {
			found = true
			break
		}
	}
	if !found {
		t.Error("env secret does not contain the rotated credential value")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func findCondition(conditions []metav1.Condition, condType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == condType {
			return &conditions[i]
		}
	}
	return nil
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
