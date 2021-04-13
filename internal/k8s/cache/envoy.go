// Copyright Project Contour Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import (
	"context"
	"fmt"

	contour_api_v1alpha1 "github.com/projectcontour/contour/apis/projectcontour/v1alpha1"
	k8s_envoy "github.com/projectcontour/contour/internal/k8s/envoy"
	retryable "github.com/projectcontour/contour/internal/retryableerror"
	validation "github.com/projectcontour/contour/internal/validation"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type envoyReconciler struct {
	client       client.Client
	eventHandler cache.ResourceEventHandler
	log          logrus.FieldLogger
}

// NewEnvoyController creates the Envoy controller from mgr. The controller will be pre-configured
// to watch for Envoy objects across all namespaces.
// TODO [danehans]: Should the controller only watch envoy custom resources in the same namespace as the controller?
func NewEnvoyController(mgr manager.Manager, eventHandler cache.ResourceEventHandler, log logrus.FieldLogger) (controller.Controller, error) {
	r := &envoyReconciler{
		client:       mgr.GetClient(),
		eventHandler: eventHandler,
		log:          log,
	}
	c, err := controller.New("envoy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return nil, err
	}
	if err := c.Watch(&source.Kind{Type: &contour_api_v1alpha1.Envoy{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return nil, err
	}

	return c, nil
}

func (r *envoyReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {

	// Fetch the Envoy from the cache.
	envoy := &contour_api_v1alpha1.Envoy{}
	err := r.client.Get(ctx, request.NamespacedName, envoy)
	if errors.IsNotFound(err) {
		r.log.WithField("name", envoy.Name).WithField("namespace", envoy.Namespace).Info("failed to find envoy")
		return reconcile.Result{}, nil
	}

	// Check if object is deleted.
	if envoy.ObjectMeta.DeletionTimestamp.IsZero() {
		// Pass the object off to the eventHandler.
		r.eventHandler.OnAdd(envoy)

		if err := validation.Envoy(ctx, r.client, envoy); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to validate envoy %q in namespace %q: %w",
				envoy.Name, envoy.Namespace, err)
		}
		if !envoy.IsFinalized() {
			// Before doing anything with the envoy, ensure it has a finalizer
			// so it can cleaned-up later.
			if err := k8s_envoy.EnsureFinalizer(ctx, r.client, envoy); err != nil {
				return reconcile.Result{}, err
			}
			r.log.WithField("name", envoy.Name).WithField("namespace", envoy.Namespace).Info("finalized envoy")
		} else {
			r.log.WithField("name", envoy.Name).WithField("namespace", envoy.Namespace).Info("envoy finalized")
			if err := r.ensureEnvoy(ctx, envoy); err != nil {
				switch e := err.(type) {
				case retryable.Error:
					r.log.WithField("after", e.After()).WithField("error", e).Error("got retryable error; requeueing")
					return reconcile.Result{RequeueAfter: e.After()}, nil
				default:
					return reconcile.Result{}, err
				}
			}
			r.log.WithField("name", envoy.Name).WithField("namespace", envoy.Namespace).Info("ensured envoy")
		}
		return reconcile.Result{}, nil
	}

	r.eventHandler.OnDelete(envoy)
	if err := r.ensureEnvoyDeleted(ctx, envoy); err != nil {
		switch e := err.(type) {
		case retryable.Error:
			r.log.WithField("after", e.After()).WithField("error", e).Error("got retryable error; requeueing")
			return reconcile.Result{RequeueAfter: e.After()}, nil
		default:
			return reconcile.Result{}, err
		}
	}
	r.log.WithField("name", envoy.Name).WithField("namespace", envoy.Namespace).Info("deleted envoy")

	return reconcile.Result{}, nil
}

// ensureEnvoy ensures all necessary resources exist for the given envoy.
func (r *envoyReconciler) ensureEnvoy(ctx context.Context, envoy *contour_api_v1alpha1.Envoy) error {
	if envoy.Spec.NetworkPublishing.Type == contour_api_v1alpha1.LoadBalancerServicePublishingType ||
		envoy.Spec.NetworkPublishing.Type == contour_api_v1alpha1.NodePortServicePublishingType ||
		envoy.Spec.NetworkPublishing.Type == contour_api_v1alpha1.ClusterIPServicePublishingType {
		if err := k8s_envoy.EnsureService(ctx, r.client, envoy); err != nil {
			return fmt.Errorf("failed to ensure service for envoy %q in namespace %q: %w",
				envoy.Name, envoy.Namespace, err)
		}
		r.log.WithField("name", envoy.Name).WithField("namespace", envoy.Namespace).Info("ensured service for envoy")
	}

	return nil
}

// ensureContourDeleted ensures envoy and all child resources have been deleted.
func (r *envoyReconciler) ensureEnvoyDeleted(ctx context.Context, envoy *contour_api_v1alpha1.Envoy) error {
	if envoy.Spec.NetworkPublishing.Type == contour_api_v1alpha1.LoadBalancerServicePublishingType ||
		envoy.Spec.NetworkPublishing.Type == contour_api_v1alpha1.NodePortServicePublishingType {
		if err := k8s_envoy.EnsureServiceDeleted(ctx, r.client, envoy); err != nil {
			return fmt.Errorf("failed to delete service for envoy %q in namespace %q: %w",
				envoy.Name, envoy.Namespace, err)
		}
		r.log.WithField("name", envoy.Name).WithField("namespace", envoy.Namespace).Info("deleted service for envoy")
	}

	if err := k8s_envoy.EnsureFinalizerRemoved(ctx, r.client, envoy); err != nil {
		return fmt.Errorf("failed to remove finalizer from envoy %q in namespace %q: %w",
			envoy.Name, envoy.Namespace, err)
	}
	r.log.WithField("name", envoy.Name).WithField("namespace", envoy.Namespace).Info("removed finalizer from envoy")

	return nil
}
