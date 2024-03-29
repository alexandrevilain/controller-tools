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

package hash

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Sha256 returns the sha256 hash of the json representation of the provided paramater.
func Sha256(o any) (string, error) {
	data, err := json.Marshal(o)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	_, err = h.Write(data)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// Deprecated: use Sha256.
// Sha256Object returns provided object's sha256 hash.
func Sha256Object(o client.Object) (string, error) {
	return Sha256(o)
}
