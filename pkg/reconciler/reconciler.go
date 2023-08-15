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

package reconciler

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/alexandrevilain/controller-tools/pkg/discovery"
	"github.com/alexandrevilain/controller-tools/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Reconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
	Discovery discovery.Manager
}

type reconcileResource struct {
	builder resource.Builder
	current client.Object
	found   bool
}

func (r *Reconciler) ReconcileBuilder(ctx context.Context, owner client.Object, builder resource.Builder) (client.Object, error) {
	res := builder.Build()
	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, res, func() error {
		return builder.Update(res)
	})
	r.logAndRecordOperationResult(ctx, owner, res, result, err)
	return res, err
}

func (r *Reconciler) ReconcileBuilders(ctx context.Context, owner client.Object, builders []resource.Builder) ([]*resource.Status, time.Duration, error) {
	logger := log.FromContext(ctx)

	resources, err := r.getReconcileResourceFromBuilders(ctx, builders)
	if err != nil {
		return nil, 0, err
	}

	logger.Info("Reconciling resources", "count", len(resources))

	statuses := []*resource.Status{}

	for _, res := range resources {
		// If the builder isn't enabled, check if it needs to be deleted, then skip iteration.
		if !res.builder.Enabled() {
			if res.found {
				err := r.Client.Delete(ctx, res.current)
				r.logAndRecordOperationResult(ctx, owner, res.current, controllerutil.OperationResult("deleted"), err)
				if err != nil {
					return nil, 0, fmt.Errorf("can't delete resource: %w", err)
				}
			}

			continue
		}

		// The build can provide a custom compare function, ensure equality.Semantic knowns it.
		if comparer, ok := res.builder.(resource.Comparer); ok {
			err := equality.Semantic.AddFunc(comparer.Equal)
			if err != nil {
				return nil, 0, err
			}
		}

		// Create case
		if !res.found && res.builder.Enabled() {
			res.current = res.builder.Build()
			err := res.builder.Update(res.current)
			if err != nil {
				return nil, 0, err
			}

			err = r.Client.Create(ctx, res.current)
			r.logAndRecordOperationResult(ctx, owner, res.current, controllerutil.OperationResultCreated, err)
			if err != nil {
				return nil, 0, err
			}
		}

		// Update case
		if res.found && res.builder.Enabled() {
			before := res.current.DeepCopyObject()
			err := res.builder.Update(res.current)
			if err != nil {
				return nil, 0, err
			}

			if !equality.Semantic.DeepEqual(before, res.current) {
				err = r.Client.Update(ctx, res.current)
				r.logAndRecordOperationResult(ctx, owner, res.current, controllerutil.OperationResultUpdated, err)
				if err != nil {
					return nil, 0, err
				}
			}
		}

		// Report status
		status, err := r.getResourceStatus(res.current)
		if err != nil {
			return nil, 0, err
		}

		statuses = append(statuses, status)
	}

	return statuses, 0, nil
}

func (r *Reconciler) getResourceStatus(res client.Object) (*resource.Status, error) {
	uobj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(res)
	if err != nil {
		return nil, err
	}

	u := &unstructured.Unstructured{}
	u.SetUnstructuredContent(uobj)

	status, err := kstatus.Compute(u)
	if err != nil {
		return nil, err
	}

	return &resource.Status{
		GVK:       res.GetObjectKind().GroupVersionKind(),
		Name:      res.GetName(),
		Namespace: res.GetNamespace(),
		Labels:    res.GetLabels(),
		Ready:     status.Status == kstatus.CurrentStatus,
	}, nil
}

func (r *Reconciler) getTypeFromGVK(gvk schema.GroupVersionKind) reflect.Type {
	for typename, reflectType := range r.Scheme.KnownTypes(gvk.GroupVersion()) {
		if typename == gvk.Kind {
			return reflectType
		}
	}
	return nil
}

func (r *Reconciler) getReconcileResourceFromBuilders(ctx context.Context, builders []resource.Builder) ([]*reconcileResource, error) {
	logger := log.FromContext(ctx)

	result := []*reconcileResource{}

	for _, builder := range builders {
		res := builder.Build()
		gvk, err := apiutil.GVKForObject(res, r.Scheme)
		if err != nil {
			return nil, err
		}

		objectType := r.getTypeFromGVK(gvk)
		if objectType == nil {
			return nil, fmt.Errorf("can't get type for %s", gvk)
		}

		object, ok := reflect.New(objectType).Interface().(client.Object)
		if !ok {
			return nil, errors.New("can't create a new client.Object instance from known type")
		}

		supported, err := r.Discovery.IsGVKSupported(gvk)
		if err != nil {
			return nil, fmt.Errorf("can't determine if GVK \"%s\" is supported: %w", gvk.String(), err)
		}

		if !supported {
			logger.V(2).Info("Skipping resource due to unsupported by apiserver", "kind", gvk.Kind)
			continue
		}

		found := true
		err = r.Client.Get(ctx, client.ObjectKeyFromObject(res), object, &client.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				found = false
			} else {
				return nil, err
			}
		}

		result = append(result, &reconcileResource{
			builder: builder,
			current: object,
			found:   found,
		})
	}

	return result, nil
}

// logAndRecordOperationResult logs and records an event for the provided object operation result.
func (r *Reconciler) logAndRecordOperationResult(ctx context.Context, owner, resource runtime.Object, operationResult controllerutil.OperationResult, err error) {
	logger := log.FromContext(ctx)

	var (
		action string
		reason string
	)
	switch operationResult {
	case controllerutil.OperationResultCreated:
		action = "create"
		reason = "RessourceCreate"
	case controllerutil.OperationResultUpdated, controllerutil.OperationResultUpdatedStatus, controllerutil.OperationResultUpdatedStatusOnly:
		action = "update"
		reason = "ResourceUpdate"
	case controllerutil.OperationResult("deleted"):
		action = "delete"
		reason = "ResourceDelete"
	case controllerutil.OperationResultNone:
		fallthrough
	default:
		return
	}

	if err == nil {
		msg := fmt.Sprintf("%sd resource %s of type %T", action, resource.(metav1.Object).GetName(), resource.(metav1.Object))
		reason := fmt.Sprintf("%sSuccess", reason)
		logger.Info(msg)
		r.Recorder.Event(owner, corev1.EventTypeNormal, reason, msg)
	}

	if err != nil {
		msg := fmt.Sprintf("failed to %s resource %s of Type %T", action, resource.(metav1.Object).GetName(), resource.(metav1.Object))
		reason := fmt.Sprintf("%sError", reason)
		logger.Error(err, msg)
		r.Recorder.Event(owner, corev1.EventTypeWarning, reason, msg)
	}
}
