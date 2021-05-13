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

package controller

import (
	"context"
	"fmt"

	"github.com/projectcontour/contour/internal/slice"
	"github.com/projectcontour/contour/internal/status"
	"github.com/projectcontour/contour/internal/validation"
	"github.com/projectcontour/contour/pkg/config"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	gatewayapi_v1alpha1 "sigs.k8s.io/gateway-api/apis/v1alpha1"
)

const finalizer = "gateway.networking.x-k8s.io/finalizer"

type gatewayReconciler struct {
	client       client.Client
	eventHandler cache.ResourceEventHandler
	log          logrus.FieldLogger
}

// NewGatewayController creates the gateway controller from mgr. The controller will be pre-configured
// to watch for Gateway objects across all namespaces.
func NewGatewayController(mgr manager.Manager, eventHandler cache.ResourceEventHandler, log logrus.FieldLogger) (controller.Controller, error) {
	r := &gatewayReconciler{
		client:       mgr.GetClient(),
		eventHandler: eventHandler,
		log:          log,
	}
	c, err := controller.New("gateway-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return nil, err
	}
	if err := c.Watch(&source.Kind{Type: &gatewayapi_v1alpha1.Gateway{}}, r.enqueueRequestForOwnedGateway()); err != nil {
		return nil, err
	}
	// TODO: Add a watch for gatewayclasses owned by contour to keep gateway status updated.
	return c, nil
}

// enqueueRequestForOwnedGateway returns an event handler that maps events to
// Gateway objects that reference a GatewayClass owned by Contour.
func (r *gatewayReconciler) enqueueRequestForOwnedGateway() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(a client.Object) []reconcile.Request {
		gw, ok := a.(*gatewayapi_v1alpha1.Gateway)
		if !ok {
			r.log.WithField("name", a.GetName()).WithField("namespace", a.GetNamespace()).Info("invalid object, bypassing reconciliation.")
			return []reconcile.Request{}
		}
		if err := classForGateway(context.Background(), r.client, gw); err != nil {
			r.log.WithField("namespace", gw.Namespace).WithField("name", gw.Name).Info(err, ", bypassing reconciliation")
			return []reconcile.Request{}
		}
		// The gateway references a gatewayclass that exists and is managed
		// by Contour, so enqueue it for reconciliation.
		r.log.WithField("namespace", gw.Namespace).WithField("name", gw.Name).Info("queueing gateway")
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: gw.Namespace,
					Name:      gw.Name,
				},
			},
		}
	})
}

// classForGateway returns an error if gw does not exist or is not owned by Contour.
func classForGateway(ctx context.Context, cli client.Client, gw *gatewayapi_v1alpha1.Gateway) error {
	gc := &gatewayapi_v1alpha1.GatewayClass{}
	if err := cli.Get(ctx, types.NamespacedName{Name: gw.Spec.GatewayClassName}, gc); err != nil {
		return fmt.Errorf("failed to get gatewayclass %s: %w", gw.Spec.GatewayClassName, err)
	}
	if !isController(gc) {
		return fmt.Errorf("gatewayclass %s not owned by contour", gw.Spec.GatewayClassName)
	}
	return nil
}

// isController returns true if Contour is the controller for gc.
func isController(gc *gatewayapi_v1alpha1.GatewayClass) bool {
	return gc.Spec.Controller == config.ContourGatewayClass
}

func (r *gatewayReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.log.WithField("namespace", request.Namespace).WithField("name", request.Name).Info("reconciling gateway")

	// Fetch the Gateway from the cache.
	gw := &gatewayapi_v1alpha1.Gateway{}
	if err := r.client.Get(ctx, request.NamespacedName, gw); err != nil {
		if errors.IsNotFound(err) {
			r.log.WithField("name", request.Name).WithField("namespace", request.Namespace).Info("failed to find gateway")
			return reconcile.Result{}, nil
		}
		// Error reading the object, so requeue the request.
		return reconcile.Result{}, fmt.Errorf("failed to get gateway %s/%s: %w", request.Namespace, request.Name, err)
	}

	// Check if object is deleted.
	if !gw.ObjectMeta.DeletionTimestamp.IsZero() {
		r.eventHandler.OnDelete(gw)
		// TODO: Add method to remove gateway sub-resources and finalizer.
		return reconcile.Result{}, nil
	}

	// Pass the new changed object off to the eventHandler.
	r.eventHandler.OnAdd(gw)

	// Check if the gateway is valid.
	valid := true
	if err := validation.Gateway(ctx, r.client, gw); err != nil {
		r.log.WithField("namespace", gw.Namespace).WithField("name", gw.Name).Info("invalid gateway: ", err)
		valid = false
	}

	if valid {
		if !isFinalized(gw) {
			// Before doing anything with the gateway, ensure it has a finalizer
			// so it can cleaned-up later.
			if err := ensureFinalizer(ctx, r.client, gw); err != nil {
				return reconcile.Result{}, fmt.Errorf("failed to finalize gateway %s/%s: %w", gw.Namespace, gw.Name, err)
			}
			r.log.WithField("name", request.Name).WithField("namespace", request.Namespace).Info("finalized gateway")
			// The gateway has been mutated, so get the latest.
			if err := r.client.Get(ctx, request.NamespacedName, gw); err != nil {
				if errors.IsNotFound(err) {
					r.log.WithField("name", request.Name).WithField("namespace", request.Namespace).Info("failed to find gateway")
					return reconcile.Result{}, nil
				}
				// Error reading the object, so requeue the request.
				return reconcile.Result{}, fmt.Errorf("failed to get gateway %s/%s: %w", request.Namespace, request.Name, err)
			}
		}
		// TODO: Ensure the gateway by creating manage infrastructure, i.e. Envoy service.
	}

	if err := status.SyncGateway(ctx, r.client, gw, valid); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to sync gateway %s/%s status: %w", gw.Namespace, gw.Name, err)
	}
	r.log.WithField("namespace", gw.Namespace).WithField("name", gw.Name).Info("synced gateway status")

	return reconcile.Result{}, nil
}

// isFinalized returns true if gw is finalized.
func isFinalized(gw *gatewayapi_v1alpha1.Gateway) bool {
	for _, f := range gw.Finalizers {
		if f == finalizer {
			return true
		}
	}
	return false
}

// ensureFinalizer ensures the finalizer is added to the given gw.
func ensureFinalizer(ctx context.Context, cli client.Client, gw *gatewayapi_v1alpha1.Gateway) error {
	if !slice.ContainsString(gw.Finalizers, finalizer) {
		updated := gw.DeepCopy()
		updated.Finalizers = append(updated.Finalizers, finalizer)
		if err := cli.Update(ctx, updated); err != nil {
			return fmt.Errorf("failed to add finalizer %s: %w", finalizer, err)
		}
	}
	return nil
}
