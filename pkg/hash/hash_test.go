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

package hash_test

import (
	"testing"

	"github.com/alexandrevilain/controller-tools/pkg/hash"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSha256Object(t *testing.T) {
	tests := map[string]struct {
		object       client.Object
		expectedHash string
		expectedErr  string
	}{
		"configmap": {
			object: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: corev1.NamespaceDefault,
				},
				Data: map[string]string{
					"a": "b",
				},
			},
			expectedHash: "39b034ab36fc323b196ebe4f2992a6fa782a77bdc3bbf6beb7e93da963138e8a",
		},
	}

	for name, test := range tests {
		t.Run(name, func(tt *testing.T) {
			sum, err := hash.Sha256Object(test.object)
			if test.expectedErr != "" {
				assert.Error(tt, err)
				assert.ErrorContains(tt, err, test.expectedErr)

				return
			}

			assert.Equal(tt, test.expectedHash, sum)
		})
	}
}
