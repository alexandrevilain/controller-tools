// Licensed to Alexandre VILAIN under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Alexandre VILAIN licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package discovery

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newFakeManager(t *testing.T) *manager {
	// Create the fake client.
	client := fakeclientset.NewSimpleClientset()
	fakeDiscovery, ok := client.Discovery().(*fakediscovery.FakeDiscovery)
	if !ok {
		t.Fatalf("couldn't convert Discovery() to *FakeDiscovery")
	}

	fakeDiscovery.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "pods",
					Namespaced: true,
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, batchv1.AddToScheme(scheme))

	cache := newCache()
	cache.Set(schema.GroupVersionKind{
		Group:   "apps",
		Kind:    "Deployment",
		Version: "v1",
	}, true)

	return &manager{
		scheme: scheme,
		client: fakeDiscovery,
		cache:  cache,
	}
}

func TestIsGVKSupported(t *testing.T) {
	manager := newFakeManager(t)

	tests := map[string]struct {
		gvk               schema.GroupVersionKind
		expectedSupported bool
	}{
		"existing gvk": {
			gvk: schema.GroupVersionKind{
				Group:   "",
				Kind:    "Pod",
				Version: "v1",
			},
			expectedSupported: true,
		},
		"gvk in cache": {
			gvk: schema.GroupVersionKind{
				Group:   "apps",
				Kind:    "Deployment",
				Version: "v1",
			},
			expectedSupported: true,
		},
		"non existing kind in a known group": {
			gvk: schema.GroupVersionKind{
				Group:   "",
				Kind:    "Foo",
				Version: "v1",
			},
			expectedSupported: false,
		},
		"non existing kind in an unknown group": {
			gvk: schema.GroupVersionKind{
				Group:   "fake.io",
				Kind:    "Foo",
				Version: "v1",
			},
			expectedSupported: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(tt *testing.T) {
			supported, err := manager.IsGVKSupported(test.gvk)
			assert.NoError(tt, err)

			assert.Equal(tt, test.expectedSupported, supported)

			// Check that the value is in the cache.
			value, found := manager.cache.Get(test.gvk)
			assert.Equal(tt, test.expectedSupported, value)
			assert.Equal(tt, true, found)
		})
	}
}

func TestAreObjectsSupported(t *testing.T) {
	manager := newFakeManager(t)

	tests := map[string]struct {
		object            client.Object
		expectedSupported bool
	}{
		"existing object": {
			object:            &corev1.Pod{},
			expectedSupported: true,
		},
		"object in cache": {
			object:            &appsv1.Deployment{},
			expectedSupported: true,
		},
		"non existing object": {
			object:            &batchv1.Job{},
			expectedSupported: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(tt *testing.T) {
			supported, err := manager.AreObjectsSupported(test.object)
			assert.NoError(tt, err)

			assert.Equal(tt, test.expectedSupported, supported)
		})
	}
}
