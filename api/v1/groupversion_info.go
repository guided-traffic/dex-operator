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

// Package v1 contains API Schema definitions for the dex.gtrfc.com v1 API group.
// +kubebuilder:object:generate=true
// +groupName=dex.gtrfc.com
package v1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "dex.gtrfc.com", Version: "v1"}

	// SchemeBuilder is used to add functions to this group's scheme.
	SchemeBuilder = &builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// builder mirrors sigs.k8s.io/controller-runtime/pkg/scheme.Builder without
// pulling controller-runtime into the api package.
type builder struct {
	GroupVersion schema.GroupVersion
	runtime.SchemeBuilder
}

func (b *builder) Register(objects ...runtime.Object) *builder {
	b.SchemeBuilder.Register(func(s *runtime.Scheme) error {
		for _, obj := range objects {
			gvk := b.GroupVersion.WithKind(reflect.TypeOf(obj).Elem().Name())
			s.AddKnownTypeWithName(gvk, obj)
		}
		metav1.AddToGroupVersion(s, b.GroupVersion)
		return nil
	})
	return b
}

func (b *builder) AddToScheme(s *runtime.Scheme) error {
	return b.SchemeBuilder.AddToScheme(s)
}
