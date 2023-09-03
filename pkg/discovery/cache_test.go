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
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var podGVK = schema.GroupVersionKind{
	Group:   "",
	Version: "v1",
	Kind:    "Pod",
}

func TestCache(t *testing.T) {
	cache := newCache()

	value, found := cache.Get(podGVK)
	assert.Equal(t, false, value)
	assert.Equal(t, false, found)

	cache.Set(podGVK, true)

	value, found = cache.Get(podGVK)
	assert.Equal(t, true, value)
	assert.Equal(t, true, found)

	cache.Set(podGVK, false)

	value, found = cache.Get(podGVK)
	assert.Equal(t, false, value)
	assert.Equal(t, true, found)
}
