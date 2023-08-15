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

package resource

import (
	"errors"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getTypeFromGVK(gvk schema.GroupVersionKind, scheme *runtime.Scheme) reflect.Type {
	for typename, reflectType := range scheme.KnownTypes(gvk.GroupVersion()) {
		if typename == gvk.Kind {
			return reflectType
		}
	}
	return nil
}

// NewObjectFromGVK instanciates a new client.Object for the provided GVK.
// GVK should be known in the provided scheme.
func NewObjectFromGVK(gvk schema.GroupVersionKind, scheme *runtime.Scheme) (client.Object, error) {
	objectType := getTypeFromGVK(gvk, scheme)
	if objectType == nil {
		return nil, fmt.Errorf("can't get type for %s", gvk)
	}

	object, ok := reflect.New(objectType).Interface().(client.Object)
	if !ok {
		return nil, errors.New("can't create a new client.Object instance from known type")
	}

	return object, nil
}
