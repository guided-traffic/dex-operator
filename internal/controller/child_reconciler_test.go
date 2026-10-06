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
	"reflect"
	"sort"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
	"github.com/guided-traffic/dex-operator/internal/controller"
)

// readyCondition reconciles obj with r and returns its Ready condition.
func readyCondition[T interface {
	*U
	client.Object
	controller.ChildObject
}, U any](t *testing.T, c client.Client, r *controller.GenericChildReconciler[T, U], obj T) *metav1.Condition {
	t.Helper()
	key := client.ObjectKeyFromObject(obj)
	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: key}); err != nil {
		t.Fatalf("Reconcile(%s) returned error: %v", key, err)
	}
	var updated U
	if err := c.Get(context.Background(), key, T(&updated)); err != nil {
		t.Fatalf("fetching %s: %v", key, err)
	}
	cond := meta.FindStatusCondition(T(&updated).GetCommonStatus().Conditions, dexv1.ConditionTypeReady)
	if cond == nil {
		t.Fatalf("Ready condition not set on %s", key)
	}
	return cond
}

// TestChildReconciler_ClientAndConnectorSameTenantNamespace verifies the
// split: with allowedNamespaces [tenant] and allowedConnectorNamespaces
// omitted, a static client in the tenant namespace is Ready while a
// connector next to it is rejected with a message naming the new field.
func TestChildReconciler_ClientAndConnectorSameTenantNamespace(t *testing.T) {
	ref := dexv1.InstallationRef{Name: "main", Namespace: "dex"}
	inst := &dexv1.DexInstallation{
		ObjectMeta: metav1.ObjectMeta{Name: "main", Namespace: "dex"},
		Spec: dexv1.DexInstallationSpec{
			Issuer:            "https://dex.example.com",
			ConfigSecretName:  "dex-config",
			EnvSecretName:     "dex-env",
			AllowedNamespaces: []string{"tenant"},
			Storage:           dexv1.DexStorageSpec{Type: dexv1.StorageKubernetes},
		},
	}
	sc := &dexv1.DexStaticClient{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "tenant"},
		Spec: dexv1.DexStaticClientSpec{
			InstallationRef: ref, DisplayName: "App", ClientID: "app", Public: true,
		},
	}
	conn := &dexv1.DexLocalConnector{
		ObjectMeta: metav1.ObjectMeta{Name: "local", Namespace: "tenant"},
		Spec:       dexv1.DexLocalConnectorSpec{InstallationRef: ref, DisplayName: "Local"},
	}

	scheme := newTestScheme(t)
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(inst, sc, conn).
		WithStatusSubresource(&dexv1.DexStaticClient{}, &dexv1.DexLocalConnector{}).
		Build()

	scR := &controller.GenericChildReconciler[*dexv1.DexStaticClient, dexv1.DexStaticClient]{Client: c, Scheme: scheme}
	if cond := readyCondition(t, c, scR, sc); cond.Status != metav1.ConditionTrue {
		t.Errorf("static client Ready = %v (%s); want True", cond.Status, cond.Message)
	}

	connR := &controller.GenericChildReconciler[*dexv1.DexLocalConnector, dexv1.DexLocalConnector]{Client: c, Scheme: scheme}
	cond := readyCondition(t, c, connR, conn)
	if cond.Status != metav1.ConditionFalse {
		t.Fatalf("connector Ready = %v; want False", cond.Status)
	}
	want := `namespace "tenant" is not in DexInstallation dex/main allowedConnectorNamespaces (omitted: only "dex" is allowed)`
	if cond.Message != want {
		t.Errorf("connector message = %q; want %q", cond.Message, want)
	}
}

// requestKeys returns the sorted "namespace/name" keys of requests.
func requestKeys(requests []ctrl.Request) []string {
	keys := make([]string, len(requests))
	for i, r := range requests {
		keys[i] = r.Namespace + "/" + r.Name
	}
	sort.Strings(keys)
	return keys
}

// TestMapInstallationToChildren verifies that a DexInstallation event
// enqueues exactly the children of the reconciler's kind that reference that
// installation, across namespaces, and nothing that references another one.
func TestMapInstallationToChildren(t *testing.T) {
	main := minimalInstallation() // dex/test-installation
	mainRef := dexv1.InstallationRef{Name: main.Name, Namespace: main.Namespace}
	otherRef := dexv1.InstallationRef{Name: "other", Namespace: "dex"}

	oidc := func(ns, name string, ref dexv1.InstallationRef) *dexv1.DexOIDCConnector {
		return &dexv1.DexOIDCConnector{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
			Spec:       dexv1.DexOIDCConnectorSpec{InstallationRef: ref},
		}
	}
	sc := &dexv1.DexStaticClient{
		ObjectMeta: metav1.ObjectMeta{Name: "grafana", Namespace: "apps"},
		Spec:       dexv1.DexStaticClientSpec{InstallationRef: mainRef},
	}

	_, c := newReconciler(t,
		main,
		oidc("dex", "okta", mainRef),
		oidc("tenant", "keycloak", mainRef),
		oidc("dex", "foreign", otherRef),
		sc,
	)
	scheme := newTestScheme(t)

	connR := &controller.GenericChildReconciler[*dexv1.DexOIDCConnector, dexv1.DexOIDCConnector]{Client: c, Scheme: scheme}
	got, err := connR.MapInstallationToChildren(context.Background(), main)
	if err != nil {
		t.Fatalf("MapInstallationToChildren: %v", err)
	}
	if want := []string{"dex/okta", "tenant/keycloak"}; !reflect.DeepEqual(requestKeys(got), want) {
		t.Errorf("connector requests = %v; want %v", requestKeys(got), want)
	}

	scR := &controller.GenericChildReconciler[*dexv1.DexStaticClient, dexv1.DexStaticClient]{Client: c, Scheme: scheme}
	got, err = scR.MapInstallationToChildren(context.Background(), main)
	if err != nil {
		t.Fatalf("MapInstallationToChildren: %v", err)
	}
	if want := []string{"apps/grafana"}; !reflect.DeepEqual(requestKeys(got), want) {
		t.Errorf("static client requests = %v; want %v", requestKeys(got), want)
	}

	other := &dexv1.DexInstallation{ObjectMeta: metav1.ObjectMeta{Name: "unreferenced", Namespace: "dex"}}
	got, err = connR.MapInstallationToChildren(context.Background(), other)
	if err != nil {
		t.Fatalf("MapInstallationToChildren: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("unreferenced installation produced requests %v; want none", requestKeys(got))
	}
}

// TestMapInstallationToChildren_UnknownListKind verifies that a scheme
// without the child's kind is reported as an error instead of silently
// mapping to nothing. SetupWithManager runs the same check at startup.
func TestMapInstallationToChildren_UnknownListKind(t *testing.T) {
	r := &controller.GenericChildReconciler[*dexv1.DexOIDCConnector, dexv1.DexOIDCConnector]{
		Client: fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build(),
		Scheme: runtime.NewScheme(), // no dex kinds registered
	}
	if _, err := r.MapInstallationToChildren(context.Background(), minimalInstallation()); err == nil {
		t.Error("expected an error for a scheme that does not know the connector kind")
	}
}
