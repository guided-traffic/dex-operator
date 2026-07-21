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
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
	"github.com/guided-traffic/dex-operator/internal/controller"
)

// staticClient builds a DexStaticClient with the given namespace and name.
func staticClient(namespace, name string) dexv1.DexStaticClient {
	return dexv1.DexStaticClient{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
	}
}

// clientKeys returns the "namespace/name" key of each client in order.
func clientKeys(clients []dexv1.DexStaticClient) []string {
	keys := make([]string, len(clients))
	for i, c := range clients {
		keys[i] = c.Namespace + "/" + c.Name
	}
	return keys
}

// wantOrder is the deterministic order the set below must always produce:
// sorted by namespace first, then by name.
var wantOrder = []string{"a/alpha", "a/beta", "a/zeta", "b/grafana", "b/harbor"}

// permutations of the same five clients. Each must collapse to wantOrder once
// filtered. The reversed and rotated variants guarantee we exercise input
// orders that differ from the desired output.
func clientPermutations() [][]dexv1.DexStaticClient {
	return [][]dexv1.DexStaticClient{
		{
			staticClient("a", "zeta"), staticClient("b", "harbor"),
			staticClient("a", "alpha"), staticClient("b", "grafana"),
			staticClient("a", "beta"),
		},
		{
			staticClient("b", "harbor"), staticClient("b", "grafana"),
			staticClient("a", "zeta"), staticClient("a", "beta"),
			staticClient("a", "alpha"),
		},
		{
			staticClient("a", "beta"), staticClient("a", "alpha"),
			staticClient("b", "grafana"), staticClient("a", "zeta"),
			staticClient("b", "harbor"),
		},
	}
}

// TestFilterStaticClients_SortsDeterministically reproduces the production
// rollout-restart loop from namespace iam: the cache-backed List returned the
// seven static clients in non-deterministic order, so the generated config
// reordered on every reconcile. filterItems must return a stable,
// namespace/name-sorted order regardless of input order.
//
// AllowedNamespaces "*" is the exact production configuration and the code
// path that previously returned the input slice untouched.
func TestFilterStaticClients_SortsDeterministically(t *testing.T) {
	tests := []struct {
		name    string
		allowed []string
		want    []string
	}{
		{name: "wildcard allows all", allowed: []string{"*"}, want: wantOrder},
		{name: "explicit namespaces", allowed: []string{"a", "b"}, want: wantOrder},
		{name: "single namespace filtered", allowed: []string{"a"}, want: []string{"a/alpha", "a/beta", "a/zeta"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for i, perm := range clientPermutations() {
				got := clientKeys(controller.FilterStaticClients(perm, tc.allowed))
				if !reflect.DeepEqual(got, tc.want) {
					t.Errorf("permutation %d: got %v, want %v", i, got, tc.want)
				}
			}
		})
	}
}

// TestFilterStaticClients_StableAcrossShuffles asserts that every input
// ordering of the same set produces the identical output ordering. This is the
// invariant that keeps the config YAML byte-stable across reconciles and thus
// prevents the spurious "config changed" rollout restarts.
func TestFilterStaticClients_StableAcrossShuffles(t *testing.T) {
	var first []string
	for i, perm := range clientPermutations() {
		got := clientKeys(controller.FilterStaticClients(perm, []string{"*"}))
		if i == 0 {
			first = got
			continue
		}
		if !reflect.DeepEqual(got, first) {
			t.Errorf("permutation %d produced %v, differs from first %v", i, got, first)
		}
	}
}

// TestFilterStaticClients_EmptyAllowedDeniesAll keeps the existing deny-all
// behaviour intact after the sorting change.
func TestFilterStaticClients_EmptyAllowedDeniesAll(t *testing.T) {
	got := controller.FilterStaticClients(clientPermutations()[0], nil)
	if got != nil {
		t.Errorf("expected nil for empty allowed list, got %v", clientKeys(got))
	}
}
