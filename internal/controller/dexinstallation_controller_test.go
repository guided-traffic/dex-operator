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
func minimalInstallation(ns string) *dexv1.DexInstallation {
	return &dexv1.DexInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-installation",
			Namespace: ns,
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
		Build()

	r := &controller.DexInstallationReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}
	return r, fakeClient
}

// TestReconcile_Minimal verifies that a minimal DexInstallation creates both secrets.
func TestReconcile_Minimal(t *testing.T) {
	inst := minimalInstallation("dex")
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
	inst := minimalInstallation("dex")

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
			Name:            "Okta",
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
	inst := minimalInstallation("dex")
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
			Name:            "Local",
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
			Name:            "Local",
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
