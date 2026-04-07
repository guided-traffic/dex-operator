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

package controller

import (
	"bytes"
	"context"
	"reflect"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// dataEqualFunc compares two secret data maps and returns true when their
// contents are considered equal.
type dataEqualFunc func(a, b map[string][]byte) bool

// applySecret creates or updates a Secret with the given data.
// It returns true when the secret data has actually changed, false when
// the existing content already matched.  The equal function determines how
// existing and desired data are compared.
func applySecret(
	ctx context.Context,
	c client.Client,
	namespace, name string,
	labels map[string]string,
	data map[string][]byte,
	equal dataEqualFunc,
) (changed bool, err error) {
	var existing corev1.Secret
	nsName := types.NamespacedName{Namespace: namespace, Name: name}

	getErr := c.Get(ctx, nsName, &existing)
	if getErr != nil && !errors.IsNotFound(getErr) {
		return false, getErr
	}

	if errors.IsNotFound(getErr) {
		secret := buildSecret(namespace, name, labels, data)
		return true, c.Create(ctx, secret)
	}

	if equal(existing.Data, data) {
		return false, nil
	}

	patch := client.MergeFrom(existing.DeepCopy())
	existing.Data = data
	if existing.Labels == nil {
		existing.Labels = make(map[string]string)
	}
	for k, v := range labels {
		existing.Labels[k] = v
	}
	return true, c.Patch(ctx, &existing, patch)
}

func buildSecret(namespace, name string, labels map[string]string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: data,
	}
}

func secretDataEqual(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok || !bytes.Equal(va, vb) {
			return false
		}
	}
	return true
}

// yamlSecretDataEqual compares two secret data maps using semantic YAML
// comparison for each value.  This avoids false diffs caused by non-
// deterministic map key ordering in yaml.Marshal output.
// If a value cannot be parsed as YAML, it falls back to byte comparison.
func yamlSecretDataEqual(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if !yamlBytesEqual(va, vb) {
			return false
		}
	}
	return true
}

// yamlBytesEqual compares two byte slices semantically as YAML.
// If either slice cannot be parsed, it falls back to bytes.Equal.
func yamlBytesEqual(a, b []byte) bool {
	var va, vb any
	if err := yaml.Unmarshal(a, &va); err != nil {
		return bytes.Equal(a, b)
	}
	if err := yaml.Unmarshal(b, &vb); err != nil {
		return bytes.Equal(a, b)
	}
	return reflect.DeepEqual(va, vb)
}
