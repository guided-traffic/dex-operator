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
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
	"github.com/guided-traffic/dex-operator/internal/controller"
)

// claimsRef is the installation of every test in this file:
// minimalInstallation, "dex/test-installation", allowedNamespaces ["*"].
var claimsRef = dexv1.InstallationRef{Name: "test-installation", Namespace: "dex"}

func claimsSecret(ns, name, clientID string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Data: map[string][]byte{
			"client-id":     []byte(clientID),
			"client-secret": []byte(ns + "-" + name + "-secret"),
		},
	}
}

func claimsClient(ns, name string, peers ...string) *dexv1.DexStaticClient {
	return &dexv1.DexStaticClient{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec: dexv1.DexStaticClientSpec{
			InstallationRef: claimsRef,
			DisplayName:     name,
			SecretRef:       &dexv1.StaticClientSecretRef{Name: name, ClientIDKey: "client-id", ClientSecretKey: "client-secret"},
			RedirectURIs:    []string{"https://" + name + "." + ns + ".example.com/callback"},
			TrustedPeers:    peers,
		},
	}
}

func reconcileInstallation(t *testing.T, r *controller.DexInstallationReconciler) error {
	t.Helper()
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: claimsRef.Namespace, Name: claimsRef.Name},
	})
	return err
}

func getInstallation(t *testing.T, c client.Client) *dexv1.DexInstallation {
	t.Helper()
	var inst dexv1.DexInstallation
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: claimsRef.Namespace, Name: claimsRef.Name}, &inst); err != nil {
		t.Fatalf("fetching installation: %v", err)
	}
	return &inst
}

func getStaticClient(t *testing.T, c client.Client, ns, name string) *dexv1.DexStaticClient {
	t.Helper()
	var sc dexv1.DexStaticClient
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: name}, &sc); err != nil {
		t.Fatalf("fetching static client %s/%s: %v", ns, name, err)
	}
	return &sc
}

// TestReconcile_ReportsRejectedChildren verifies the installation status of a
// render with a contested ID and a dropped peer: rejectedChildren,
// droppedTrustedPeers, both conditions with their counts, and Ready=True,
// because the config was rendered and written.
func TestReconcile_ReportsRejectedChildren(t *testing.T) {
	r, c := newReconciler(t,
		minimalInstallation(),
		claimsSecret("team-a", "app", "shared"), claimsClient("team-a", "app"),
		claimsSecret("team-b", "app", "shared"), claimsClient("team-b", "app"),
		claimsSecret("team-c", "web", "web"), claimsClient("team-c", "web", "foreign"),
		claimsSecret("team-d", "foreign", "foreign"), claimsClient("team-d", "foreign"),
	)
	if err := reconcileInstallation(t, r); err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	inst := getInstallation(t, c)
	want := []dexv1.RejectedChild{
		{Kind: "DexStaticClient", Namespace: "team-a", Name: "app", ID: "shared", Reason: dexv1.RejectionReasonDuplicateID,
			Message: `client ID "shared" is claimed by more than one DexStaticClient of DexInstallation dex/test-installation`},
		{Kind: "DexStaticClient", Namespace: "team-b", Name: "app", ID: "shared", Reason: dexv1.RejectionReasonDuplicateID,
			Message: `client ID "shared" is claimed by more than one DexStaticClient of DexInstallation dex/test-installation`},
	}
	if !reflect.DeepEqual(inst.Status.RejectedChildren, want) {
		t.Errorf("rejectedChildren = %+v; want %+v", inst.Status.RejectedChildren, want)
	}
	wantDropped := []dexv1.DroppedTrustedPeer{{Namespace: "team-c", Name: "web", PeerID: "foreign"}}
	if !reflect.DeepEqual(inst.Status.DroppedTrustedPeers, wantDropped) {
		t.Errorf("droppedTrustedPeers = %+v; want %+v", inst.Status.DroppedTrustedPeers, wantDropped)
	}
	if inst.Status.StaticClientCount != 2 {
		t.Errorf("StaticClientCount = %d; want 2", inst.Status.StaticClientCount)
	}

	for condType, msg := range map[string]string{
		dexv1.ConditionTypeChildrenRejected:    "rejected children: 2, see status.rejectedChildren",
		dexv1.ConditionTypeTrustedPeersDropped: "dropped trustedPeers entries: 1, see status.droppedTrustedPeers",
	} {
		cond := meta.FindStatusCondition(inst.Status.Conditions, condType)
		if cond == nil || cond.Status != metav1.ConditionTrue || cond.Message != msg {
			t.Errorf("%s = %+v; want True with %q", condType, cond, msg)
		}
	}
	if cond := meta.FindStatusCondition(inst.Status.Conditions, dexv1.ConditionTypeReady); cond == nil || cond.Status != metav1.ConditionTrue {
		t.Errorf("Ready = %+v; want True", cond)
	}
}

// TestReconcile_ConditionsFalseWithoutRejections verifies that both report
// conditions are False on a render that leaves nothing out.
func TestReconcile_ConditionsFalseWithoutRejections(t *testing.T) {
	r, c := newReconciler(t, minimalInstallation(), claimsSecret("team-a", "app", "app"), claimsClient("team-a", "app"))
	if err := reconcileInstallation(t, r); err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	inst := getInstallation(t, c)
	for _, condType := range []string{dexv1.ConditionTypeChildrenRejected, dexv1.ConditionTypeTrustedPeersDropped} {
		if cond := meta.FindStatusCondition(inst.Status.Conditions, condType); cond == nil || cond.Status != metav1.ConditionFalse {
			t.Errorf("%s = %+v; want False", condType, cond)
		}
	}
}

// TestReconcile_RecordsClientID verifies that every collected client gets
// status.clientID, rendered or not, and that a client whose Secret is gone
// keeps claiming it on the next render.
func TestReconcile_RecordsClientID(t *testing.T) {
	victimSecret := claimsSecret("team-a", "app", "app")
	r, c := newReconciler(t,
		minimalInstallation(),
		victimSecret, claimsClient("team-a", "app"),
		claimsSecret("team-c", "x", "dup"), claimsClient("team-c", "x"),
		claimsSecret("team-d", "x", "dup"), claimsClient("team-d", "x"),
	)
	if err := reconcileInstallation(t, r); err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	for _, k := range []struct{ ns, name, id string }{{"team-a", "app", "app"}, {"team-c", "x", "dup"}, {"team-d", "x", "dup"}} {
		if got := getStaticClient(t, c, k.ns, k.name).Status.ClientID; got != k.id {
			t.Errorf("%s/%s status.clientID = %q; want %q", k.ns, k.name, got, k.id)
		}
	}

	// The victim's Secret disappears and another namespace claims its ID.
	if err := c.Delete(context.Background(), victimSecret); err != nil {
		t.Fatalf("deleting secret: %v", err)
	}
	for _, obj := range []client.Object{claimsSecret("team-b", "rogue", "app"), claimsClient("team-b", "rogue")} {
		if err := c.Create(context.Background(), obj); err != nil {
			t.Fatalf("creating %T: %v", obj, err)
		}
	}
	if err := reconcileInstallation(t, r); err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	var cfg corev1.Secret
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "dex", Name: "dex-config"}, &cfg); err != nil {
		t.Fatalf("config secret not found: %v", err)
	}
	if bytes.Contains(cfg.Data["config.yaml"], []byte("id: app")) {
		t.Errorf("the victim's ID renders for the rogue client:\n%s", cfg.Data["config.yaml"])
	}
	got := map[string]string{}
	for _, rc := range getInstallation(t, c).Status.RejectedChildren {
		got[rc.Namespace+"/"+rc.Name] = rc.Reason + " " + rc.ID
	}
	if got["team-a/app"] != "BuildFailed app" || got["team-b/rogue"] != "DuplicateID app" {
		t.Errorf("rejections = %v; want team-a/app BuildFailed and team-b/rogue DuplicateID", got)
	}
	if id := getStaticClient(t, c, "team-a", "app").Status.ClientID; id != "app" {
		t.Errorf("victim's status.clientID = %q; want it kept", id)
	}
}

// TestReconcile_ConnectorGuardKeepsLastConfig verifies that when no
// collected connector renders, the last written config stays, the
// installation is Ready=False and the rejections are reported.
func TestReconcile_ConnectorGuardKeepsLastConfig(t *testing.T) {
	okta := &dexv1.DexOIDCConnector{
		ObjectMeta: metav1.ObjectMeta{Namespace: "dex", Name: "okta"},
		Spec: dexv1.DexOIDCConnectorSpec{
			InstallationRef: claimsRef,
			DisplayName:     "Okta",
			Issuer:          "https://okta.example.com",
			ClientIDRef:     dexv1.SecretKeyRef{Name: "okta", Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: "okta", Key: "client-secret"},
		},
	}
	oktaSecret := claimsSecret("dex", "okta", "okta-upstream")
	r, c := newReconciler(t, minimalInstallation(), okta, oktaSecret)
	if err := reconcileInstallation(t, r); err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	var before corev1.Secret
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "dex", Name: "dex-config"}, &before); err != nil {
		t.Fatalf("config secret not found: %v", err)
	}

	if err := c.Delete(context.Background(), oktaSecret); err != nil {
		t.Fatalf("deleting secret: %v", err)
	}
	err := reconcileInstallation(t, r)
	if err == nil || !strings.Contains(err.Error(), "no connector renders") {
		t.Fatalf("err = %v; want the connector guard", err)
	}

	var after corev1.Secret
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "dex", Name: "dex-config"}, &after); err != nil {
		t.Fatalf("config secret not found: %v", err)
	}
	if !bytes.Equal(before.Data["config.yaml"], after.Data["config.yaml"]) {
		t.Error("the connector guard overwrote the last written config")
	}

	inst := getInstallation(t, c)
	if cond := meta.FindStatusCondition(inst.Status.Conditions, dexv1.ConditionTypeReady); cond == nil ||
		cond.Status != metav1.ConditionFalse || !strings.Contains(cond.Message, "DexOIDCConnector dex/okta: BuildFailed") {
		t.Errorf("Ready = %+v; want False naming the failed connector", cond)
	}
	if len(inst.Status.RejectedChildren) != 1 || inst.Status.RejectedChildren[0].Reason != dexv1.RejectionReasonBuildFailed {
		t.Errorf("rejectedChildren = %+v; want okta BuildFailed", inst.Status.RejectedChildren)
	}
	if cond := meta.FindStatusCondition(inst.Status.Conditions, dexv1.ConditionTypeChildrenRejected); cond == nil || cond.Status != metav1.ConditionTrue {
		t.Errorf("ChildrenRejected = %+v; want True", cond)
	}
}

// TestChildReconciler_RejectedChild verifies that a child the installation
// left out is Ready=False with the installation's reason and message, and
// that a static client gets TrustedPeersDropped from the installation.
func TestChildReconciler_RejectedChild(t *testing.T) {
	r, c := newReconciler(t,
		minimalInstallation(),
		claimsSecret("team-a", "app", "shared"), claimsClient("team-a", "app"),
		claimsSecret("team-b", "app", "shared"), claimsClient("team-b", "app"),
		claimsSecret("team-c", "web", "web"), claimsClient("team-c", "web", "foreign", "other"),
	)
	if err := reconcileInstallation(t, r); err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	scR := &controller.GenericChildReconciler[*dexv1.DexStaticClient, dexv1.DexStaticClient]{Client: c, Scheme: newTestScheme(t)}
	cond := readyCondition(t, c, scR, claimsClient("team-a", "app"))
	if cond.Status != metav1.ConditionFalse || cond.Reason != dexv1.RejectionReasonDuplicateID ||
		cond.Message != `client ID "shared" is claimed by more than one DexStaticClient of DexInstallation dex/test-installation` {
		t.Errorf("Ready = %+v; want False/DuplicateID", cond)
	}

	if cond := readyCondition(t, c, scR, claimsClient("team-c", "web")); cond.Status != metav1.ConditionTrue {
		t.Errorf("trusting client Ready = %+v; want True", cond)
	}
	web := getStaticClient(t, c, "team-c", "web")
	dropped := meta.FindStatusCondition(web.Status.Conditions, dexv1.ConditionTypeTrustedPeersDropped)
	want := `trustedPeers "foreign", "other" are left out of the rendered config: no DexStaticClient in namespace "team-c" holds these IDs`
	if dropped == nil || dropped.Status != metav1.ConditionTrue || dropped.Message != want {
		t.Errorf("TrustedPeersDropped = %+v; want True with %q", dropped, want)
	}

	app := getStaticClient(t, c, "team-a", "app")
	if cond := meta.FindStatusCondition(app.Status.Conditions, dexv1.ConditionTypeTrustedPeersDropped); cond == nil || cond.Status != metav1.ConditionFalse {
		t.Errorf("TrustedPeersDropped of a client without peers = %+v; want False", cond)
	}
	if app.Status.ClientID != "shared" {
		t.Errorf("child reconciler lost status.clientID: %q", app.Status.ClientID)
	}
}

// TestChildReconciler_RejectedConnector verifies the lookup by kind: a
// connector rejection reaches the connector, not a static client of the same
// namespace and name.
func TestChildReconciler_RejectedConnector(t *testing.T) {
	inst := minimalInstallation()
	inst.Status.RejectedChildren = []dexv1.RejectedChild{{
		Kind: "DexOIDCConnector", Namespace: "dex", Name: "okta", ID: "okta",
		Reason: dexv1.RejectionReasonBuildFailed, Message: "clientID: secret dex/okta[client-id]: not found",
	}}
	conn := &dexv1.DexOIDCConnector{
		ObjectMeta: metav1.ObjectMeta{Namespace: "dex", Name: "okta"},
		Spec:       dexv1.DexOIDCConnectorSpec{InstallationRef: claimsRef},
	}
	sc := &dexv1.DexStaticClient{
		ObjectMeta: metav1.ObjectMeta{Namespace: "dex", Name: "okta"},
		Spec:       dexv1.DexStaticClientSpec{InstallationRef: claimsRef, ClientID: "okta", Public: true},
	}
	_, c := newReconciler(t, inst, conn, sc)
	scheme := newTestScheme(t)

	connR := &controller.GenericChildReconciler[*dexv1.DexOIDCConnector, dexv1.DexOIDCConnector]{Client: c, Scheme: scheme}
	if cond := readyCondition(t, c, connR, conn); cond.Status != metav1.ConditionFalse || cond.Reason != dexv1.RejectionReasonBuildFailed {
		t.Errorf("connector Ready = %+v; want False/BuildFailed", cond)
	}
	scR := &controller.GenericChildReconciler[*dexv1.DexStaticClient, dexv1.DexStaticClient]{Client: c, Scheme: scheme}
	if cond := readyCondition(t, c, scR, sc); cond.Status != metav1.ConditionTrue {
		t.Errorf("static client Ready = %+v; want True", cond)
	}
}

// TestChildReportChangedPredicate verifies that installation status updates
// pass the child watch only when rejectedChildren or droppedTrustedPeers
// change.
func TestChildReportChangedPredicate(t *testing.T) {
	base := minimalInstallation()
	withRejected := base.DeepCopy()
	withRejected.Status.RejectedChildren = []dexv1.RejectedChild{{Kind: "DexStaticClient", Namespace: "a", Name: "b"}}
	withDropped := base.DeepCopy()
	withDropped.Status.DroppedTrustedPeers = []dexv1.DroppedTrustedPeer{{Namespace: "a", Name: "b", PeerID: "c"}}
	countOnly := base.DeepCopy()
	countOnly.Status.StaticClientCount = 7

	p := controller.ChildReportChangedPredicate()
	for name, tc := range map[string]struct {
		newObj *dexv1.DexInstallation
		want   bool
	}{
		"rejectedChildren changed":    {withRejected, true},
		"droppedTrustedPeers changed": {withDropped, true},
		"other status changed":        {countOnly, false},
	} {
		if got := p.Update(event.UpdateEvent{ObjectOld: base, ObjectNew: tc.newObj}); got != tc.want {
			t.Errorf("%s: Update = %v; want %v", name, got, tc.want)
		}
	}
}

// TestReconcile_DeterministicWithDuplicates verifies that the same set of
// children with duplicates, failures and dropped peers renders identically
// whatever order the objects were created in.
func TestReconcile_DeterministicWithDuplicates(t *testing.T) {
	objs := []client.Object{
		claimsSecret("team-a", "app", "shared"), claimsClient("team-a", "app", "grafana"),
		claimsSecret("team-b", "app", "shared"), claimsClient("team-b", "app"),
		claimsSecret("team-c", "grafana", "g1"), claimsClient("team-c", "grafana", "g2"),
		claimsSecret("team-d", "grafana", "g2"), claimsClient("team-d", "grafana", "g2"),
		claimsClient("team-e", "missing"),
	}

	render := func(order []client.Object) (string, []dexv1.RejectedChild, []dexv1.DroppedTrustedPeer) {
		r, c := newReconciler(t, append([]client.Object{minimalInstallation()}, order...)...)
		if err := reconcileInstallation(t, r); err != nil {
			t.Fatalf("Reconcile returned error: %v", err)
		}
		var cfg corev1.Secret
		if err := c.Get(context.Background(), types.NamespacedName{Namespace: "dex", Name: "dex-config"}, &cfg); err != nil {
			t.Fatalf("config secret not found: %v", err)
		}
		inst := getInstallation(t, c)
		return string(cfg.Data["config.yaml"]), inst.Status.RejectedChildren, inst.Status.DroppedTrustedPeers
	}

	reversed := make([]client.Object, len(objs))
	for i, o := range objs {
		reversed[len(objs)-1-i] = o.DeepCopyObject().(client.Object)
	}
	cfg1, rej1, drop1 := render(objs)
	cfg2, rej2, drop2 := render(reversed)
	if cfg1 != cfg2 || !reflect.DeepEqual(rej1, rej2) || !reflect.DeepEqual(drop1, drop2) {
		t.Errorf("renders differ by creation order\n--- first ---\n%s\n--- second ---\n%s", cfg1, cfg2)
	}
	if len(rej1) != 3 || len(drop1) != 1 {
		t.Errorf("rejected = %+v, dropped = %+v; want 3 and 1", rej1, drop1)
	}
}
