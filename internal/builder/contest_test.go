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

package builder_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
	"github.com/guided-traffic/dex-operator/internal/builder"
)

// The installation of every test in this file is "dex/test".
const home = "dex"

var (
	older = metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	newer = metav1.NewTime(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
)

var instRef = dexv1.InstallationRef{Name: "test", Namespace: home}

// confClient is a confidential client ns/name whose Secret ns/name holds the
// client-id and client-secret (see clientSecrets).
func confClient(ns, name string, created metav1.Time, peers ...string) dexv1.DexStaticClient {
	return dexv1.DexStaticClient{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name, CreationTimestamp: created},
		Spec: dexv1.DexStaticClientSpec{
			InstallationRef: instRef,
			DisplayName:     name,
			SecretRef:       &dexv1.StaticClientSecretRef{Name: name},
			RedirectURIs:    []string{"https://" + name + "." + ns + ".example.com/callback"},
			TrustedPeers:    peers,
			CORS:            true,
		},
	}
}

// pubClient is a secretless client ns/name with the inline ID id.
func pubClient(ns, name, id string) dexv1.DexStaticClient {
	return dexv1.DexStaticClient{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name, CreationTimestamp: older},
		Spec: dexv1.DexStaticClientSpec{
			InstallationRef: instRef, DisplayName: name, ClientID: id, Public: true,
		},
	}
}

// clientSecrets adds the Secret of a confClient ns/name with the given ID.
func clientSecrets(m map[string]string, ns, name, id string) map[string]string {
	if m == nil {
		m = map[string]string{}
	}
	m[ns+"/"+name+"[client-id]"] = id
	m[ns+"/"+name+"[client-secret]"] = ns + "-" + name + "-secret"
	return m
}

func oidcConn(ns, name, id string, created metav1.Time) dexv1.DexOIDCConnector {
	return dexv1.DexOIDCConnector{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name, CreationTimestamp: created},
		Spec: dexv1.DexOIDCConnectorSpec{
			InstallationRef: instRef,
			ID:              id,
			DisplayName:     name,
			Issuer:          "https://" + name + ".example.com",
			ClientIDRef:     dexv1.SecretKeyRef{Name: name, Key: "client-id"},
			ClientSecretRef: dexv1.SecretKeyRef{Name: name, Key: "client-secret"},
		},
	}
}

func oauth2Conn(ns, name, id string) dexv1.DexOAuth2Connector {
	return dexv1.DexOAuth2Connector{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name, CreationTimestamp: older},
		Spec: dexv1.DexOAuth2ConnectorSpec{
			InstallationRef:  instRef,
			ID:               id,
			DisplayName:      name,
			ClientIDRef:      dexv1.SecretKeyRef{Name: name, Key: "client-id"},
			ClientSecretRef:  dexv1.SecretKeyRef{Name: name, Key: "client-secret"},
			AuthorizationURL: "https://" + name + ".example.com/authorize",
			TokenURL:         "https://" + name + ".example.com/token",
		},
	}
}

// connSecrets adds the Secret of a connector ns/name.
func connSecrets(m map[string]string, ns, name string) map[string]string {
	if m == nil {
		m = map[string]string{}
	}
	m[ns+"/"+name+"[client-id]"] = name + "-upstream-id"
	m[ns+"/"+name+"[client-secret]"] = ns + "-" + name + "-upstream-secret"
	return m
}

// localConn is a DexLocalConnector in the installation's namespace.
func localConn() []dexv1.DexLocalConnector {
	return []dexv1.DexLocalConnector{{
		ObjectMeta: metav1.ObjectMeta{Namespace: home, Name: "local"},
		Spec:       dexv1.DexLocalConnectorSpec{InstallationRef: instRef, DisplayName: "Email"},
	}}
}

// build renders the input against "dex/test" and fails on a build error.
// Clients go in as given; the controller hands them over sorted.
func build(t *testing.T, cs builder.ConnectorSet, clients []dexv1.DexStaticClient, secrets map[string]string) builder.Output {
	t.Helper()
	out, err := builder.Build(context.Background(), builder.Input{
		Installation:  minimalInstallation(home),
		Connectors:    cs,
		StaticClients: clients,
		Secrets:       mockResolver(secrets),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return out
}

// renderedClientIDs returns the IDs of the rendered static clients in order.
func renderedClientIDs(t *testing.T, out builder.Output) []string {
	t.Helper()
	scs, _ := parseYAML(t, out.ConfigYAML)["staticClients"].([]any)
	ids := make([]string, 0, len(scs))
	for _, sc := range scs {
		ids = append(ids, sc.(map[string]any)["id"].(string))
	}
	return ids
}

// renderedClient returns the rendered static client with the given ID.
func renderedClient(t *testing.T, out builder.Output, id string) map[string]any {
	t.Helper()
	scs, _ := parseYAML(t, out.ConfigYAML)["staticClients"].([]any)
	for _, sc := range scs {
		if m := sc.(map[string]any); m["id"] == id {
			return m
		}
	}
	t.Fatalf("static client %q not rendered", id)
	return nil
}

// renderedConnectors returns "type/id" of the rendered connectors in order.
func renderedConnectors(t *testing.T, out builder.Output) []string {
	t.Helper()
	conns, _ := parseYAML(t, out.ConfigYAML)["connectors"].([]any)
	ids := make([]string, 0, len(conns))
	for _, c := range conns {
		m := c.(map[string]any)
		ids = append(ids, m["type"].(string)+"/"+m["id"].(string))
	}
	return ids
}

// rejectedKeys returns "kind ns/name reason id" of every rejected child.
func rejectedKeys(out builder.Output) []string {
	keys := make([]string, 0, len(out.Rejected))
	for _, rc := range out.Rejected {
		keys = append(keys, rc.Kind+" "+rc.Namespace+"/"+rc.Name+" "+rc.Reason+" "+rc.ID)
	}
	return keys
}

func assertStrings(t *testing.T, what string, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %q; want %q", what, got, want)
	}
}

// ── contest rule ──────────────────────────────────────────────────────────────

func TestContest(t *testing.T) {
	cases := []struct {
		name       string
		namespaces []string
		ids        []string
		want       []bool
	}{
		{"single claimant", []string{"team-a"}, []string{"x"}, []bool{true}},
		{"installation namespace wins", []string{"team-a", home, "zzz"}, []string{"x", "x", "x"}, []bool{false, true, false}},
		{"two other namespaces", []string{"team-a", "team-b"}, []string{"x", "x"}, []bool{false, false}},
		{"two in one namespace", []string{"team-a", "team-a"}, []string{"x", "x"}, []bool{false, false}},
		{"two in the installation namespace", []string{home, home, "team-a"}, []string{"x", "x", "x"}, []bool{false, false, false}},
		{"independent IDs", []string{"team-a", "team-b"}, []string{"x", "y"}, []bool{true, true}},
		{"empty ID is no claim", []string{"team-a", "team-b"}, []string{"", ""}, []bool{false, false}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := builder.ExportedContest(tc.namespaces, tc.ids, home)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("contest = %v; want %v", got, tc.want)
			}
			// Order of the claims never matters.
			n := len(tc.ids)
			revNS, revIDs, revWant := make([]string, n), make([]string, n), make([]bool, n)
			for i := range tc.ids {
				revNS[n-1-i], revIDs[n-1-i], revWant[n-1-i] = tc.namespaces[i], tc.ids[i], tc.want[i]
			}
			if got := builder.ExportedContest(revNS, revIDs, home); !reflect.DeepEqual(got, revWant) {
				t.Errorf("reversed contest = %v; want %v", got, revWant)
			}
		})
	}
}

// TestBuild_ClientID_InstallationNamespaceWins verifies that a client in the
// installation's namespace keeps its ID against a client of another
// namespace, whether that namespace sorts before or after the installation's
// and whichever of the two is older.
func TestBuild_ClientID_InstallationNamespaceWins(t *testing.T) {
	for _, otherNS := range []string{"aaa", "zzz"} {
		for _, homeIsOlder := range []bool{true, false} {
			homeAge, otherAge := older, newer
			if !homeIsOlder {
				homeAge, otherAge = newer, older
			}
			platform := confClient(home, "argocd", homeAge)
			tenant := confClient(otherNS, "rogue", otherAge)
			secrets := clientSecrets(nil, home, "argocd", "argocd")
			clientSecrets(secrets, otherNS, "rogue", "argocd")

			clients := []dexv1.DexStaticClient{platform, tenant}
			if otherNS < home {
				clients = []dexv1.DexStaticClient{tenant, platform}
			}
			out := build(t, builder.ConnectorSet{}, clients, secrets)

			assertStrings(t, "rendered", renderedClientIDs(t, out), []string{"argocd"})
			assertStrings(t, "rejected", rejectedKeys(out), []string{"DexStaticClient " + otherNS + "/rogue DuplicateID argocd"})
			if got := renderedClient(t, out, "argocd")["secretEnv"]; got != "ARGOCD_CLIENT_SECRET" {
				t.Errorf("secretEnv = %v; want the platform client's", got)
			}
			if got := string(out.EnvSecretData["ARGOCD_CLIENT_SECRET"]); got != "dex-argocd-secret" {
				t.Errorf("ARGOCD_CLIENT_SECRET = %q; want the platform client's secret", got)
			}
		}
	}
}

// TestBuild_ClientID_ContestedRendersForNobody covers rule 3: two claimants
// outside the installation's namespace, two inside one namespace, two inside
// the installation's namespace - every one of them is rejected.
func TestBuild_ClientID_ContestedRendersForNobody(t *testing.T) {
	cases := []struct {
		name   string
		ns1    string
		ns2    string
		name2  string
		wantNS []string
	}{
		{"two other namespaces", "team-a", "team-b", "app", []string{"team-a/app", "team-b/app"}},
		{"one other namespace", "team-a", "team-a", "app-2", []string{"team-a/app", "team-a/app-2"}},
		{"installation namespace", home, home, "app-2", []string{home + "/app", home + "/app-2"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			secrets := clientSecrets(nil, tc.ns1, "app", "shared")
			clientSecrets(secrets, tc.ns2, tc.name2, "shared")
			clientSecrets(secrets, "team-c", "bystander", "bystander")
			out := build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{
				confClient(tc.ns1, "app", older),
				confClient(tc.ns2, tc.name2, newer),
				confClient("team-c", "bystander", older),
			}, secrets)

			assertStrings(t, "rendered", renderedClientIDs(t, out), []string{"bystander"})
			var got []string
			for _, rc := range out.Rejected {
				if rc.Reason != dexv1.RejectionReasonDuplicateID || rc.ID != "shared" {
					t.Errorf("rejection %+v; want DuplicateID for shared", rc)
				}
				got = append(got, rc.Namespace+"/"+rc.Name)
			}
			assertStrings(t, "rejected", got, tc.wantNS)
			if strings.Contains(string(out.ConfigYAML), "shared") {
				t.Errorf("contested ID is in the config:\n%s", out.ConfigYAML)
			}
		})
	}
}

// TestBuild_ClientID_MessageNamesNoNamespace verifies that a DuplicateID
// message is the same under rule 2 and rule 3 and names no namespace but the
// installation's.
func TestBuild_ClientID_MessageNamesNoNamespace(t *testing.T) {
	secrets := clientSecrets(nil, home, "platform", "x")
	clientSecrets(secrets, "team-secret-a", "app", "x")
	rule2 := build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{
		confClient(home, "platform", older), confClient("team-secret-a", "app", older),
	}, secrets)

	clientSecrets(secrets, "team-secret-b", "app", "x")
	rule3 := build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{
		confClient("team-secret-a", "app", older), confClient("team-secret-b", "app", older),
	}, secrets)

	want := `client ID "x" is claimed by more than one DexStaticClient of DexInstallation dex/test`
	for _, rc := range append(rule2.Rejected, rule3.Rejected...) {
		if rc.Message != want {
			t.Errorf("message of %s/%s = %q; want %q", rc.Namespace, rc.Name, rc.Message, want)
		}
		if strings.Contains(rc.Message, "team-secret") {
			t.Errorf("message names a namespace: %q", rc.Message)
		}
	}
	if len(rule2.Rejected) != 1 || len(rule3.Rejected) != 2 {
		t.Errorf("rejected = %d and %d; want 1 and 2", len(rule2.Rejected), len(rule3.Rejected))
	}
}

// TestBuild_ClientID_StatusClaimWhileSecretMissing verifies that a
// confidential client whose Secret is missing keeps claiming
// status.clientID: it is BuildFailed, its ID renders for nobody and a
// claimant of another namespace is DuplicateID.  Without status.clientID
// the other claimant renders.
func TestBuild_ClientID_StatusClaimWhileSecretMissing(t *testing.T) {
	victim := confClient("team-a", "app", older)
	victim.Status.ClientID = "app"
	secrets := clientSecrets(nil, "team-b", "rogue", "app")
	rogue := confClient("team-b", "rogue", newer)

	out := build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{victim, rogue}, secrets)
	if ids := renderedClientIDs(t, out); len(ids) != 0 {
		t.Errorf("rendered = %v; want none", ids)
	}
	assertStrings(t, "rejected", rejectedKeys(out), []string{
		"DexStaticClient team-a/app BuildFailed app",
		"DexStaticClient team-b/rogue DuplicateID app",
	})
	if msg := out.Rejected[0].Message; !strings.Contains(msg, "client-id") {
		t.Errorf("BuildFailed message = %q; want the client's own error", msg)
	}

	victim.Status.ClientID = ""
	out = build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{victim, rogue}, secrets)
	assertStrings(t, "rendered", renderedClientIDs(t, out), []string{"app"})
	assertStrings(t, "rejected", rejectedKeys(out), []string{"DexStaticClient team-a/app BuildFailed "})
}

// TestBuild_ClientID_StatusIgnoredWhileSecretResolves verifies that
// status.clientID only counts while the Secret does not resolve.
func TestBuild_ClientID_StatusIgnoredWhileSecretResolves(t *testing.T) {
	c := confClient("team-a", "app", older)
	c.Status.ClientID = "x"
	secrets := clientSecrets(nil, "team-a", "app", "y")
	clientSecrets(secrets, "team-b", "other", "x")

	out := build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{c, confClient("team-b", "other", older)}, secrets)
	assertStrings(t, "rendered", renderedClientIDs(t, out), []string{"y", "x"})
	if len(out.Rejected) != 0 {
		t.Errorf("rejected = %v; want none", rejectedKeys(out))
	}
}

// TestBuild_ClientIDs_EveryResolvedClient verifies that Output carries the
// resolved ID of every client, rendered or not, and none for a client whose
// ID could not be resolved.
func TestBuild_ClientIDs_EveryResolvedClient(t *testing.T) {
	missing := confClient("team-c", "missing", older)
	missing.Status.ClientID = "old"
	secrets := clientSecrets(nil, "team-a", "a", "shared")
	clientSecrets(secrets, "team-b", "b", "shared")

	out := build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{
		confClient("team-a", "a", older),
		confClient("team-b", "b", older),
		missing,
		pubClient("team-d", "cli", "cli"),
	}, secrets)

	want := map[types.NamespacedName]string{
		{Namespace: "team-a", Name: "a"}:   "shared",
		{Namespace: "team-b", Name: "b"}:   "shared",
		{Namespace: "team-d", Name: "cli"}: "cli",
	}
	if !reflect.DeepEqual(out.ClientIDs, want) {
		t.Errorf("ClientIDs = %v; want %v", out.ClientIDs, want)
	}
}

// ── failing children ──────────────────────────────────────────────────────────

// TestBuild_FailingClientIsSkipped verifies that a client whose Secret is
// missing is left out and reported in any namespace, the installation's
// included, while every other client renders.
func TestBuild_FailingClientIsSkipped(t *testing.T) {
	for _, ns := range []string{"team-b", home} {
		t.Run(ns, func(t *testing.T) {
			secrets := clientSecrets(nil, "team-a", "good", "good")
			out := build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{
				confClient("team-a", "good", older),
				confClient(ns, "bad", older),
			}, secrets)

			assertStrings(t, "rendered", renderedClientIDs(t, out), []string{"good"})
			assertStrings(t, "rejected", rejectedKeys(out), []string{"DexStaticClient " + ns + "/bad BuildFailed "})
			if out.StaticClientCount != 1 {
				t.Errorf("StaticClientCount = %d; want 1", out.StaticClientCount)
			}
		})
	}
}

// TestBuild_FailingClientSecretIsSkipped covers a client whose client-id
// resolves but whose client-secret does not.
func TestBuild_FailingClientSecretIsSkipped(t *testing.T) {
	secrets := map[string]string{"team-a/app[client-id]": "app"}
	out := build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{confClient("team-a", "app", older)}, secrets)

	assertStrings(t, "rejected", rejectedKeys(out), []string{"DexStaticClient team-a/app BuildFailed app"})
	if !strings.Contains(out.Rejected[0].Message, "client-secret") {
		t.Errorf("message = %q; want the client-secret error", out.Rejected[0].Message)
	}
	if got := out.ClientIDs[types.NamespacedName{Namespace: "team-a", Name: "app"}]; got != "app" {
		t.Errorf("ClientIDs[team-a/app] = %q; want app", got)
	}
}

// TestBuild_MissingStorageSecretFails verifies that storage, which belongs
// to the installation and not to a child, still aborts the render.
func TestBuild_MissingStorageSecretFails(t *testing.T) {
	inst := minimalInstallation(home)
	inst.Spec.Storage = dexv1.DexStorageSpec{
		Type: dexv1.StoragePostgres,
		Postgres: &dexv1.DexPostgresStorageSpec{
			Host: "db", Database: "dex", User: "dex",
			PasswordRef: &dexv1.SecretKeyRef{Name: "pg", Key: "password"},
		},
	}
	_, err := builder.Build(context.Background(), builder.Input{
		Installation: inst,
		Secrets:      mockResolver(nil),
	})
	if err == nil || !strings.Contains(err.Error(), "storage") {
		t.Errorf("err = %v; want a storage error", err)
	}
}

// TestBuild_RejectedClientsContributeNoCORSOrigin verifies that CORS origins
// and the client count come from rendered clients only.
func TestBuild_RejectedClientsContributeNoCORSOrigin(t *testing.T) {
	secrets := clientSecrets(nil, "team-a", "a", "shared")
	clientSecrets(secrets, "team-b", "b", "shared")
	clientSecrets(secrets, "team-c", "good", "good")

	out := build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{
		confClient("team-a", "a", older),
		confClient("team-b", "b", older),
		confClient("team-c", "failed", older),
		confClient("team-c", "good", older),
	}, secrets)

	assertOrigins(t, allowedOrigins(t, parseYAML(t, out.ConfigYAML)), []string{"https://good.team-c.example.com"})
}

// ── connectors ────────────────────────────────────────────────────────────────

// TestBuild_ConnectorID_InstallationNamespaceWinsAcrossKinds verifies that
// the contest runs across connector kinds: an OAuth2 connector of another
// namespace no longer replaces an OIDC connector of the installation's
// namespace because OAuth2 renders later.
func TestBuild_ConnectorID_InstallationNamespaceWinsAcrossKinds(t *testing.T) {
	secrets := connSecrets(nil, home, "corp")
	connSecrets(secrets, "idp", "rogue")
	out := build(t, builder.ConnectorSet{
		OIDC:   []dexv1.DexOIDCConnector{oidcConn(home, "corp", "x", newer)},
		OAuth2: []dexv1.DexOAuth2Connector{oauth2Conn("idp", "rogue", "x")},
	}, nil, secrets)

	assertStrings(t, "rendered", renderedConnectors(t, out), []string{"oidc/x"})
	assertStrings(t, "rejected", rejectedKeys(out), []string{"DexOAuth2Connector idp/rogue DuplicateID x"})
	want := `connector ID "x" is claimed by more than one connector of DexInstallation dex/test`
	if out.Rejected[0].Message != want {
		t.Errorf("message = %q; want %q", out.Rejected[0].Message, want)
	}
	if got := string(out.EnvSecretData["OIDC_X_CLIENT_SECRET"]); got != "dex-corp-upstream-secret" {
		t.Errorf("OIDC_X_CLIENT_SECRET = %q; want the installation connector's", got)
	}
	if _, ok := out.EnvSecretData["OAUTH2_X_CLIENT_SECRET"]; ok {
		t.Error("rejected connector's credential is in the env Secret")
	}
}

// TestBuild_ConnectorID_ContestedRendersForNobody covers rule 3 for
// connectors, here two of one type and ID in two other namespaces, which
// used to overwrite each other's env key.
func TestBuild_ConnectorID_ContestedRendersForNobody(t *testing.T) {
	secrets := connSecrets(nil, "idp-a", "okta")
	connSecrets(secrets, "idp-b", "okta")
	connSecrets(secrets, home, "github")
	out := build(t, builder.ConnectorSet{
		OIDC: []dexv1.DexOIDCConnector{oidcConn("idp-a", "okta", "", older), oidcConn("idp-b", "okta", "", older)},
		GitHub: []dexv1.DexGitHubConnector{{
			ObjectMeta: metav1.ObjectMeta{Namespace: home, Name: "github"},
			Spec: dexv1.DexGitHubConnectorSpec{
				InstallationRef: instRef, DisplayName: "GitHub",
				ClientIDRef:     dexv1.SecretKeyRef{Name: "github", Key: "client-id"},
				ClientSecretRef: dexv1.SecretKeyRef{Name: "github", Key: "client-secret"},
			},
		}},
	}, nil, secrets)

	assertStrings(t, "rendered", renderedConnectors(t, out), []string{"github/github"})
	assertStrings(t, "rejected", rejectedKeys(out), []string{
		"DexOIDCConnector idp-a/okta DuplicateID okta",
		"DexOIDCConnector idp-b/okta DuplicateID okta",
	})
	if _, ok := out.EnvSecretData["OIDC_OKTA_CLIENT_SECRET"]; ok {
		t.Error("contested connector's credential is in the env Secret")
	}
	if out.ConnectorCount != 1 {
		t.Errorf("ConnectorCount = %d; want 1", out.ConnectorCount)
	}
}

// TestBuild_ConnectorEnvKeys_SanitizeAlike verifies that two connectors whose
// IDs differ but sanitize to one env key both render with their own key.
func TestBuild_ConnectorEnvKeys_SanitizeAlike(t *testing.T) {
	secrets := connSecrets(nil, home, "okta")
	connSecrets(secrets, home, "okta-upper")
	out := build(t, builder.ConnectorSet{
		OIDC: []dexv1.DexOIDCConnector{oidcConn(home, "okta", "okta", older), oidcConn(home, "okta-upper", "Okta", newer)},
	}, nil, secrets)

	assertStrings(t, "rendered", renderedConnectors(t, out), []string{"oidc/okta", "oidc/Okta"})
	fallback := "OIDC_OKTA_" + strings.ToUpper(builder.ExportedChildHash("DexOIDCConnector", home, "okta-upper")) + "_CLIENT_SECRET"
	if got := string(out.EnvSecretData["OIDC_OKTA_CLIENT_SECRET"]); got != "dex-okta-upstream-secret" {
		t.Errorf("OIDC_OKTA_CLIENT_SECRET = %q; want the older connector's", got)
	}
	if got := string(out.EnvSecretData[fallback]); got != "dex-okta-upper-upstream-secret" {
		t.Errorf("%s = %q; want the newer connector's", fallback, got)
	}
	if !strings.Contains(string(out.ConfigYAML), "$"+fallback) {
		t.Errorf("config does not reference %s:\n%s", fallback, out.ConfigYAML)
	}
}

// TestBuild_ClientNamedLikeConnectorKey verifies that a tenant client named
// after a connector's env key no longer replaces the connector's credential.
func TestBuild_ClientNamedLikeConnectorKey(t *testing.T) {
	secrets := connSecrets(nil, home, "okta")
	clientSecrets(secrets, "team-a", "oidc-okta", "tenant-app")
	out := build(t, builder.ConnectorSet{
		OIDC: []dexv1.DexOIDCConnector{oidcConn(home, "okta", "", newer)},
	}, []dexv1.DexStaticClient{confClient("team-a", "oidc-okta", older)}, secrets)

	if got := string(out.EnvSecretData["OIDC_OKTA_CLIENT_SECRET"]); got != "dex-okta-upstream-secret" {
		t.Errorf("OIDC_OKTA_CLIENT_SECRET = %q; want the connector's upstream secret", got)
	}
	fallback := "OIDC_OKTA_" + strings.ToUpper(builder.ExportedChildHash("DexStaticClient", "team-a", "oidc-okta")) + "_CLIENT_SECRET"
	if got := renderedClient(t, out, "tenant-app")["secretEnv"]; got != fallback {
		t.Errorf("client secretEnv = %v; want %s", got, fallback)
	}
}

// TestBuild_LocalConnectorReservesLocalID verifies that while a
// DexLocalConnector renders, a connector with ID "local" is DuplicateID in
// any namespace; without one it renders.
func TestBuild_LocalConnectorReservesLocalID(t *testing.T) {
	secrets := connSecrets(nil, home, "corp")
	cs := builder.ConnectorSet{OIDC: []dexv1.DexOIDCConnector{oidcConn(home, "corp", "local", older)}}

	out := build(t, cs, nil, secrets)
	assertStrings(t, "rendered without local", renderedConnectors(t, out), []string{"oidc/local"})

	cs.Local = localConn()
	out = build(t, cs, nil, secrets)
	if conns := renderedConnectors(t, out); len(conns) != 0 {
		t.Errorf("rendered = %v; want none", conns)
	}
	assertStrings(t, "rejected", rejectedKeys(out), []string{"DexOIDCConnector dex/corp DuplicateID local"})
	want := `connector ID "local" is reserved for the password database of DexInstallation dex/test`
	if out.Rejected[0].Message != want {
		t.Errorf("message = %q; want %q", out.Rejected[0].Message, want)
	}
}

// TestBuild_ConnectorGuard verifies that the render aborts when connectors
// were collected and none of them renders, unless a DexLocalConnector keeps
// Dex startable.
func TestBuild_ConnectorGuard(t *testing.T) {
	secrets := connSecrets(nil, "idp-a", "shared")
	connSecrets(secrets, "idp-b", "shared")
	cs := builder.ConnectorSet{
		OIDC: []dexv1.DexOIDCConnector{
			oidcConn("idp-a", "shared", "", older), // contested
			oidcConn("idp-b", "shared", "", older), // contested
			oidcConn(home, "broken", "", older),    // Secret missing
		},
	}

	out, err := builder.Build(context.Background(), builder.Input{
		Installation: minimalInstallation(home),
		Connectors:   cs,
		Secrets:      mockResolver(secrets),
	})
	if !errors.Is(err, builder.ErrNoConnector) {
		t.Fatalf("err = %v; want ErrNoConnector", err)
	}
	if out.ConfigYAML != nil {
		t.Error("guard returned a config")
	}
	assertStrings(t, "rejected", rejectedKeys(out), []string{
		"DexOIDCConnector dex/broken BuildFailed broken",
		"DexOIDCConnector idp-a/shared DuplicateID shared",
		"DexOIDCConnector idp-b/shared DuplicateID shared",
	})
	for _, want := range []string{"dex/broken", "idp-a/shared", "idp-b/shared"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %s", err, want)
		}
	}

	cs.Local = localConn()
	out = build(t, cs, nil, secrets)
	if m := parseYAML(t, out.ConfigYAML); m["enablePasswordDB"] != true || m["connectors"] != nil {
		t.Errorf("want the password database only, got connectors=%v enablePasswordDB=%v", m["connectors"], m["enablePasswordDB"])
	}
	if out.ConnectorCount != 1 {
		t.Errorf("ConnectorCount = %d; want 1 (the local connector)", out.ConnectorCount)
	}
}

// TestBuild_ConnectorGuard_NoConnectorsCollected keeps an installation
// without any connector renderable, as before.
func TestBuild_ConnectorGuard_NoConnectorsCollected(t *testing.T) {
	out := build(t, builder.ConnectorSet{}, nil, nil)
	if out.ConfigYAML == nil {
		t.Error("no config rendered")
	}
}

// ── env key priority ──────────────────────────────────────────────────────────

// TestBuild_EnvKeyPriority verifies that the older of two clients named
// grafana keeps GRAFANA_CLIENT_SECRET whatever the render order, the newer
// renders with its fallback key, and moves back to the plain key once the
// older is gone.
func TestBuild_EnvKeyPriority(t *testing.T) {
	secrets := clientSecrets(nil, "team-a", "grafana", "grafana-a")
	clientSecrets(secrets, "team-b", "grafana", "grafana-b")
	olderB := confClient("team-b", "grafana", older)
	newerA := confClient("team-a", "grafana", newer)

	out := build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{newerA, olderB}, secrets)
	fallback := "GRAFANA_" + strings.ToUpper(builder.ExportedChildHash("DexStaticClient", "team-a", "grafana")) + "_CLIENT_SECRET"
	if got := renderedClient(t, out, "grafana-b")["secretEnv"]; got != "GRAFANA_CLIENT_SECRET" {
		t.Errorf("older client's secretEnv = %v; want GRAFANA_CLIENT_SECRET", got)
	}
	if got := renderedClient(t, out, "grafana-a")["secretEnv"]; got != fallback {
		t.Errorf("newer client's secretEnv = %v; want %s", got, fallback)
	}
	if got := string(out.EnvSecretData[fallback]); got != "team-a-grafana-secret" {
		t.Errorf("%s = %q; want the newer client's secret", fallback, got)
	}
	assertStrings(t, "render order", renderedClientIDs(t, out), []string{"grafana-a", "grafana-b"})

	out = build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{newerA}, secrets)
	if got := renderedClient(t, out, "grafana-a")["secretEnv"]; got != "GRAFANA_CLIENT_SECRET" {
		t.Errorf("after the older is gone, secretEnv = %v; want GRAFANA_CLIENT_SECRET", got)
	}
}

// TestBuild_EnvKeyPriority_InstallationNamespaceFirst verifies that a
// client in the installation's namespace keeps its plain key against an
// older client of another namespace.
func TestBuild_EnvKeyPriority_InstallationNamespaceFirst(t *testing.T) {
	secrets := clientSecrets(nil, home, "grafana", "grafana-platform")
	clientSecrets(secrets, "team-a", "grafana", "grafana-tenant")
	out := build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{
		confClient(home, "grafana", newer), confClient("team-a", "grafana", older),
	}, secrets)
	if got := renderedClient(t, out, "grafana-platform")["secretEnv"]; got != "GRAFANA_CLIENT_SECRET" {
		t.Errorf("platform client's secretEnv = %v; want GRAFANA_CLIENT_SECRET", got)
	}
}

// TestBuild_EnvKeyPlainAndFallbackTaken verifies that a child whose plain
// and fallback key are both taken by older children is BuildFailed.
func TestBuild_EnvKeyPlainAndFallbackTaken(t *testing.T) {
	hash := builder.ExportedChildHash("DexStaticClient", "team-b", "grafana")
	squatter := "grafana-" + hash
	secrets := clientSecrets(nil, "team-a", "grafana", "grafana-a")
	clientSecrets(secrets, "team-a", squatter, "squatter")
	clientSecrets(secrets, "team-b", "grafana", "grafana-b")

	out := build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{
		confClient("team-a", "grafana", older),
		confClient("team-a", squatter, older),
		confClient("team-b", "grafana", newer),
	}, secrets)

	assertStrings(t, "rejected", rejectedKeys(out), []string{"DexStaticClient team-b/grafana BuildFailed grafana-b"})
	if msg := out.Rejected[0].Message; !strings.Contains(msg, "GRAFANA_CLIENT_SECRET") || strings.Contains(msg, "team-a") {
		t.Errorf("message = %q; want both keys and no other namespace", msg)
	}
}

// TestBuild_EnvKeysUnchangedWithoutCollision verifies that every key that
// collides with nothing keeps its documented plain name.
func TestBuild_EnvKeysUnchangedWithoutCollision(t *testing.T) {
	secrets := connSecrets(nil, home, "okta")
	clientSecrets(secrets, "team-a", "grafana", "grafana")
	out := build(t, builder.ConnectorSet{
		OIDC: []dexv1.DexOIDCConnector{oidcConn(home, "okta", "", newer)},
	}, []dexv1.DexStaticClient{confClient("team-a", "grafana", older)}, secrets)

	keys := make([]string, 0, len(out.EnvSecretData))
	for k := range out.EnvSecretData {
		keys = append(keys, k)
	}
	if len(keys) != 2 || out.EnvSecretData["OIDC_OKTA_CLIENT_SECRET"] == nil || out.EnvSecretData["GRAFANA_CLIENT_SECRET"] == nil {
		t.Errorf("env keys = %v; want OIDC_OKTA_CLIENT_SECRET and GRAFANA_CLIENT_SECRET", keys)
	}
}

// ── trusted peers ─────────────────────────────────────────────────────────────

func droppedKeys(out builder.Output) []string {
	keys := make([]string, 0, len(out.DroppedTrustedPeers))
	for _, d := range out.DroppedTrustedPeers {
		keys = append(keys, d.Namespace+"/"+d.Name+" "+d.PeerID)
	}
	return keys
}

// TestBuild_TrustedPeers covers which trustedPeers entries render: only IDs
// held by a static client in the trusting client's own namespace.
func TestBuild_TrustedPeers(t *testing.T) {
	secrets := clientSecrets(nil, "team-a", "web", "web")
	clientSecrets(secrets, "team-x", "contested-1", "contested")
	clientSecrets(secrets, "team-y", "contested-2", "contested")

	out := build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{
		confClient("team-a", "web", older, "cli", "foreign", "contested", "nobody", "cli"),
		pubClient("team-a", "cli", "cli"),
		pubClient("team-b", "foreign", "foreign"),
		confClient("team-x", "contested-1", older),
		confClient("team-y", "contested-2", older),
	}, secrets)

	got, _ := renderedClient(t, out, "web")["trustedPeers"].([]any)
	if !reflect.DeepEqual(got, []any{"cli", "cli"}) {
		t.Errorf("trustedPeers = %v; want [cli cli]", got)
	}
	assertStrings(t, "dropped", droppedKeys(out), []string{
		"team-a/web contested", "team-a/web foreign", "team-a/web nobody",
	})
}

// TestBuild_TrustedPeers_PlatformClientTrustsTenantID verifies that a
// client in the installation's namespace drops a peer ID a tenant holds.
func TestBuild_TrustedPeers_PlatformClientTrustsTenantID(t *testing.T) {
	secrets := clientSecrets(nil, home, "kubernetes", "kubernetes")
	out := build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{
		confClient(home, "kubernetes", older, "kubectl", "headlamp"),
		pubClient(home, "kubectl", "kubectl"),
		pubClient("team-a", "headlamp", "headlamp"),
	}, secrets)

	got, _ := renderedClient(t, out, "kubernetes")["trustedPeers"].([]any)
	if !reflect.DeepEqual(got, []any{"kubectl"}) {
		t.Errorf("trustedPeers = %v; want [kubectl]", got)
	}
	assertStrings(t, "dropped", droppedKeys(out), []string{"dex/kubernetes headlamp"})
	assertStrings(t, "rendered", renderedClientIDs(t, out), []string{"kubernetes", "kubectl", "headlamp"})
}

// TestBuild_TrustedPeers_BuildFailedHolderStays verifies that a peer whose
// sole claimant fails to build still holds its ID, so the trusting client's
// entry is not rewritten by a gap in the peer's Secret.
func TestBuild_TrustedPeers_BuildFailedHolderStays(t *testing.T) {
	peer := confClient("team-a", "worker", older)
	peer.Status.ClientID = "worker"
	secrets := clientSecrets(nil, "team-a", "web", "web")

	out := build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{
		confClient("team-a", "web", older, "worker"), peer,
	}, secrets)

	got, _ := renderedClient(t, out, "web")["trustedPeers"].([]any)
	if !reflect.DeepEqual(got, []any{"worker"}) {
		t.Errorf("trustedPeers = %v; want [worker]", got)
	}
	if len(out.DroppedTrustedPeers) != 0 {
		t.Errorf("dropped = %v; want none", droppedKeys(out))
	}
}

// TestBuild_TrustedPeers_HolderMoves verifies that when the holder of a peer
// ID moves to another namespace, the trusting client drops the entry in the
// same render that makes the new holder live.
func TestBuild_TrustedPeers_HolderMoves(t *testing.T) {
	secrets := clientSecrets(nil, "team-a", "web", "web")
	web := confClient("team-a", "web", older, "cli")

	before := build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{web, pubClient("team-a", "cli", "cli")}, secrets)
	if got, _ := renderedClient(t, before, "web")["trustedPeers"].([]any); !reflect.DeepEqual(got, []any{"cli"}) {
		t.Fatalf("before the move trustedPeers = %v; want [cli]", got)
	}

	after := build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{web, pubClient("team-b", "cli", "cli")}, secrets)
	if _, has := renderedClient(t, after, "web")["trustedPeers"]; has {
		t.Errorf("after the move trustedPeers is still rendered")
	}
	assertStrings(t, "rendered", renderedClientIDs(t, after), []string{"web", "cli"})
	assertStrings(t, "dropped", droppedKeys(after), []string{"team-a/web cli"})
}

// TestBuild_Deterministic verifies that repeated builds of an input with
// duplicates, failures and dropped peers produce identical output.
func TestBuild_Deterministic(t *testing.T) {
	secrets := clientSecrets(nil, "team-a", "a", "shared")
	clientSecrets(secrets, "team-b", "a", "shared")
	clientSecrets(secrets, "team-c", "grafana", "g1")
	clientSecrets(secrets, "team-d", "grafana", "g2")
	connSecrets(secrets, home, "okta")
	connSecrets(secrets, "idp", "okta")
	cs := builder.ConnectorSet{OIDC: []dexv1.DexOIDCConnector{oidcConn(home, "okta", "", older), oidcConn("idp", "okta", "", older)}}
	clients := []dexv1.DexStaticClient{
		confClient("team-a", "a", older, "g1"),
		confClient("team-b", "a", older),
		confClient("team-c", "grafana", older, "g2"),
		confClient("team-d", "grafana", older),
		confClient("team-e", "missing", older),
	}

	first := build(t, cs, clients, secrets)
	for i := 0; i < 20; i++ {
		next := build(t, cs, clients, secrets)
		if string(next.ConfigYAML) != string(first.ConfigYAML) ||
			!reflect.DeepEqual(next.EnvSecretData, first.EnvSecretData) ||
			!reflect.DeepEqual(next.Rejected, first.Rejected) ||
			!reflect.DeepEqual(next.DroppedTrustedPeers, first.DroppedTrustedPeers) {
			t.Fatalf("build %d differs from the first", i)
		}
	}
}

// TestBuild_EmptyClientIDIsBuildFailed verifies that a client whose ID is
// empty — inline or in its Secret — is left out instead of rendering an
// entry Dex refuses to start on ("ID or IDEnv field is required").
func TestBuild_EmptyClientIDIsBuildFailed(t *testing.T) {
	secrets := map[string]string{"team-a/conf[client-id]": "", "team-a/conf[client-secret]": "s"}
	out := build(t, builder.ConnectorSet{}, []dexv1.DexStaticClient{
		confClient("team-a", "conf", older),
		pubClient("team-a", "pub", ""),
	}, secrets)

	if ids := renderedClientIDs(t, out); len(ids) != 0 {
		t.Errorf("rendered = %q; want none", ids)
	}
	assertStrings(t, "rejected", rejectedKeys(out), []string{
		"DexStaticClient team-a/conf BuildFailed ",
		"DexStaticClient team-a/pub BuildFailed ",
	})
	if len(out.ClientIDs) != 0 {
		t.Errorf("ClientIDs = %v; want none", out.ClientIDs)
	}
}
