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

// Package e2e contains end-to-end tests for the dex-operator.
// These tests run against a real Kubernetes cluster (e.g. Kind) with the
// dex-operator already deployed. Set KUBECONFIG to point to the target cluster,
// or rely on the default ~/.kube/config.
//
// Tests skip automatically when the dex-operator CRDs are not installed.
package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// e2eClient is the Kubernetes client used by all E2E tests.
var e2eClient client.Client

const (
	e2ePollInterval = 500 * time.Millisecond
	e2ePollTimeout  = 120 * time.Second
)

// TestMain sets up the shared Kubernetes client.  If no cluster is reachable,
// all tests are skipped.
func TestMain(m *testing.M) {
	scheme := k8sruntime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		panic("scheme: " + err.Error())
	}
	if err := dexv1.AddToScheme(scheme); err != nil {
		panic("scheme: " + err.Error())
	}

	kubeconfig := os.Getenv("KUBECONFIG")
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		// No cluster available — skip gracefully.
		os.Exit(0)
	}

	e2eClient, err = client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		panic("client: " + err.Error())
	}

	// Verify that the CRDs are installed by listing DexInstallations in the
	// default namespace.  If this call fails, the operator isn't deployed.
	if !crdsInstalled() {
		_, _ = os.Stderr.WriteString("dex-operator CRDs not installed; skipping E2E tests\n")
		os.Exit(0)
	}

	os.Exit(m.Run())
}

// crdsInstalled returns true if DexInstallation CRDs are present in the cluster.
func crdsInstalled() bool {
	list := &dexv1.DexInstallationList{}
	err := e2eClient.List(context.Background(), list, client.InNamespace("default"))
	return err == nil
}

// ── Test helpers ─────────────────────────────────────────────────────────────

// e2eEventually polls cond until it returns true or e2ePollTimeout is
// exceeded.  It marks the calling test as failed on timeout.
func e2eEventually(t *testing.T, cond func() bool, msgAndArgs ...any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), e2ePollTimeout)
	defer cancel()

	_ = wait.PollUntilContextTimeout(ctx, e2ePollInterval, e2ePollTimeout, true, func(_ context.Context) (bool, error) {
		return cond(), nil
	})
	if !cond() {
		t.Fatalf("E2E condition not met within %s: %v", e2ePollTimeout, msgAndArgs)
	}
}

// e2eCreateNamespace creates a test namespace and registers deletion on cleanup.
func e2eCreateNamespace(t *testing.T, name string) {
	t.Helper()
	ctx := context.Background()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "dex-operator-e2e",
			},
		},
	}
	if err := e2eClient.Create(ctx, ns); err != nil {
		t.Fatalf("create namespace %s: %v", name, err)
	}
	t.Cleanup(func() {
		_ = e2eClient.Delete(ctx, ns)
	})
}

// e2eCreateSecret creates a Secret and registers deletion on cleanup.
func e2eCreateSecret(t *testing.T, ns, name string, data map[string][]byte) {
	t.Helper()
	ctx := context.Background()
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Data:       data,
	}
	if err := e2eClient.Create(ctx, s); err != nil {
		t.Fatalf("create secret %s/%s: %v", ns, name, err)
	}
	t.Cleanup(func() { _ = e2eClient.Delete(ctx, s) })
}

// e2eGetSecret returns the Secret or nil.
func e2eGetSecret(ns, name string) *corev1.Secret {
	var s corev1.Secret
	if err := e2eClient.Get(context.Background(),
		client.ObjectKey{Namespace: ns, Name: name}, &s); err != nil {
		return nil
	}
	return &s
}

// e2eFindCondition returns the named condition or nil.
func e2eFindCondition(conds []metav1.Condition, condType string) *metav1.Condition {
	for i := range conds {
		if conds[i].Type == condType {
			return &conds[i]
		}
	}
	return nil
}
