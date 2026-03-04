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
	"testing"

	"github.com/guided-traffic/dex-operator/internal/controller"
)

func TestIsNamespaceAllowed(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		allowed   []string
		want      bool
	}{
		{
			name:      "empty allowedNamespaces denies everything",
			namespace: "default",
			allowed:   nil,
			want:      false,
		},
		{
			name:      "wildcard allows all namespaces",
			namespace: "some-ns",
			allowed:   []string{"*"},
			want:      true,
		},
		{
			name:      "exact match allows specific namespace",
			namespace: "dex",
			allowed:   []string{"dex", "monitoring"},
			want:      true,
		},
		{
			name:      "non-matching namespace is denied",
			namespace: "other-ns",
			allowed:   []string{"dex", "monitoring"},
			want:      false,
		},
		{
			name:      "single allowed namespace matches",
			namespace: "prod",
			allowed:   []string{"prod"},
			want:      true,
		},
		{
			name:      "wildcard in list allows any",
			namespace: "random",
			allowed:   []string{"specific", "*"},
			want:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := controller.IsNamespaceAllowed(tc.namespace, tc.allowed)
			if got != tc.want {
				t.Errorf("IsNamespaceAllowed(%q, %v) = %v; want %v",
					tc.namespace, tc.allowed, got, tc.want)
			}
		})
	}
}
