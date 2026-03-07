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

// Package integration contains integration tests for the dex-operator controllers.
// These tests use controller-runtime's envtest to spin up a real Kubernetes
// API server and etcd, then run all controllers against it.
package integration

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
	"github.com/guided-traffic/dex-operator/internal/controller"
)

// Global envtest state, shared across all tests in the package.
var (
	testEnv    *envtest.Environment
	k8sClient  client.Client
	testScheme *k8sruntime.Scheme
	cancelMgr  context.CancelFunc
)

// crdBasesPath returns the absolute path to config/crd/bases from within the
// test package, which is two directories above test/integration.
func crdBasesPath() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "config", "crd", "bases")
}

// TestMain sets up the envtest environment, starts the controller manager,
// runs all tests, then tears everything down.
func TestMain(m *testing.M) {
	testScheme = k8sruntime.NewScheme()
	mustAddScheme(clientgoscheme.AddToScheme)
	mustAddScheme(dexv1.AddToScheme)

	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{crdBasesPath()},
		ErrorIfCRDPathMissing: true,
		Scheme:                testScheme,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		panic("failed to start envtest: " + err.Error())
	}

	k8sClient, err = client.New(cfg, client.Options{Scheme: testScheme})
	if err != nil {
		panic("failed to create k8s client: " + err.Error())
	}

	// Start the controller manager with all reconcilers in the background.
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: testScheme,
		Metrics: server.Options{
			BindAddress: "0",
		},
		HealthProbeBindAddress: "0",
	})
	if err != nil {
		panic("failed to create manager: " + err.Error())
	}

	if err = (&controller.DexInstallationReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		panic("failed to setup DexInstallation controller: " + err.Error())
	}

	if err = controller.SetupConnectorControllers(mgr); err != nil {
		panic("failed to setup connector controllers: " + err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancelMgr = cancel

	go func() {
		if err := mgr.Start(ctx); err != nil {
			panic("manager failed: " + err.Error())
		}
	}()

	code := m.Run()

	cancelMgr()
	if err := testEnv.Stop(); err != nil {
		panic("failed to stop envtest: " + err.Error())
	}

	os.Exit(code)
}

// mustAddScheme panics if the scheme registration function fails.
func mustAddScheme(fn func(*k8sruntime.Scheme) error) {
	if err := fn(testScheme); err != nil {
		panic("scheme registration failed: " + err.Error())
	}
}

// ── Test helpers ─────────────────────────────────────────────────────────────

const (
	pollInterval = 200 * time.Millisecond
	pollTimeout  = 30 * time.Second
)

// eventually polls cond until it returns true or the timeout expires.
// It fails the test on timeout.
func eventually(t *testing.T, cond func() bool, msgAndArgs ...any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), pollTimeout)
	defer cancel()

	_ = wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(_ context.Context) (bool, error) {
		return cond(), nil
	})
	if !cond() {
		t.Fatalf("condition not met within %s: %v", pollTimeout, msgAndArgs)
	}
}

// createNamespace creates a namespace for the test and registers a cleanup
// function to delete all resources in it, then the namespace itself.
func createNamespace(t *testing.T, name string) {
	t.Helper()
	ctx := context.Background()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	if err := k8sClient.Create(ctx, ns); err != nil {
		t.Fatalf("create namespace %s: %v", name, err)
	}
	t.Cleanup(func() {
		_ = k8sClient.Delete(ctx, ns)
	})
}

// createSecret creates a Kubernetes Secret and registers a cleanup function.
func createSecret(t *testing.T, ns, name string, data map[string][]byte) {
	t.Helper()
	ctx := context.Background()
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Data:       data,
	}
	if err := k8sClient.Create(ctx, secret); err != nil {
		t.Fatalf("create secret %s/%s: %v", ns, name, err)
	}
	t.Cleanup(func() {
		_ = k8sClient.Delete(ctx, secret)
	})
}

// createInstallation creates a minimal DexInstallation and registers cleanup.
func createInstallation(t *testing.T, ns, name string, allowedNamespaces []string) *dexv1.DexInstallation {
	t.Helper()
	ctx := context.Background()
	inst := &dexv1.DexInstallation{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: dexv1.DexInstallationSpec{
			Issuer:            "https://dex.example.com",
			ConfigSecretName:  name + "-config",
			EnvSecretName:     name + "-env",
			AllowedNamespaces: allowedNamespaces,
			Storage: dexv1.DexStorageSpec{
				Type: dexv1.StorageKubernetes,
			},
		},
	}
	if err := k8sClient.Create(ctx, inst); err != nil {
		t.Fatalf("create installation %s/%s: %v", ns, name, err)
	}
	t.Cleanup(func() {
		_ = k8sClient.Delete(ctx, inst)
	})
	return inst
}

// getSecret fetches a Secret or returns nil if not found.
func getSecret(ns, name string) *corev1.Secret {
	var s corev1.Secret
	if err := k8sClient.Get(context.Background(),
		client.ObjectKey{Namespace: ns, Name: name}, &s); err != nil {
		return nil
	}
	return &s
}

// findCondition returns the condition with condType or nil.
func findCondition(conditions []metav1.Condition, condType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == condType {
			return &conditions[i]
		}
	}
	return nil
}
