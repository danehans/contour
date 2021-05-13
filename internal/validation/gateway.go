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

package validation

import (
	"context"
	"fmt"
	"net"

	retryable "github.com/projectcontour/contour/internal/retryableerror"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapi_v1alpha1 "sigs.k8s.io/gateway-api/apis/v1alpha1"
	"sigs.k8s.io/gateway-api/apis/v1alpha1/validation"
)

const (
	KindHTTPRoute = "HTTPRoute"
)

// Gateway returns an error if gw is an invalid Gateway.
func Gateway(ctx context.Context, cli client.Client, gw *gatewayapi_v1alpha1.Gateway) field.ErrorList {
	var errs field.ErrorList
	gc := &gatewayapi_v1alpha1.GatewayClass{}
	if err := cli.Get(ctx, types.NamespacedName{Name: gw.Spec.GatewayClassName}, gc); err != nil {
		if errors.IsNotFound(err) {
			errs = append(errs, field.InternalError(field.NewPath("spec", "GatewayClassName"), err))
		}
		errs = append(errs, fmt.Errorf("failed to get gatewayclass %q: %w", gw.Spec.GatewayClassName, err))
	}
	// See if the referenced gatewayclass is admitted.
	gcAdmitted := false
	for _, c := range gc.Status.Conditions {
		if c.Type == string(gatewayapi_v1alpha1.ConditionRouteAdmitted) && c.Status == metav1.ConditionTrue {
			gcAdmitted = true
		}
	}
	if !gcAdmitted {
		errs = append(errs, fmt.Errorf("referenced gatewayclass %q is not admitted", gw.Spec.GatewayClassName))
	}

	// Perform upstream validation
	if err := validation.ValidateGateway(gw); err != nil {
		errs = append(errs, fmt.Errorf("failed to validate gateway %s/%s: %w", gw.Namespace,	gw.Name, err))
	}

	// Perform downstream validation
	if err := gatewayListeners(gw.Spec.Listeners); err != nil {
		errs = append(errs, fmt.Errorf("failed to validate listeners for gateway %s/%s: %w", gw.Namespace,
			gw.Name, err))
	}
	if err := gatewayAddresses(gw); err != nil {
		errs = append(errs, fmt.Errorf("failed to validate addresses for gateway %s/%s: %w", gw.Namespace,
			gw.Name, err))
	}
	if len(errs) != 0 {
		return retryable.NewMaybeRetryableAggregate(errs)
	}
	return nil
}

// gatewayListeners returns an error if the listeners of the provided gw are invalid.
// TODO [danehans]: Refactor when more than 2 listeners are supported.
func gatewayListeners(listeners []gatewayapi_v1alpha1.Listener) error {
	numListeners := len(listeners)
	if numListeners != 1 && numListeners != 2 {
		return fmt.Errorf("%d is an invalid number of listeners", len(listeners))
	}
	if numListeners == 2 {
		if listeners[0].Port == listeners[1].Port {
			return fmt.Errorf("invalid listeners, port %v is non-unique", listeners[0].Port)
		}
	}
	for _, listener := range listeners {
		// Validate the listener protocol.
		if listener.Protocol != gatewayapi_v1alpha1.HTTPProtocolType ||
			listener.Protocol != gatewayapi_v1alpha1.HTTPSProtocolType ||
			listener.Protocol != gatewayapi_v1alpha1.TLSProtocolType {
			return fmt.Errorf("invalid listener protocol %q", listener.Protocol)
		}
		if listener.Protocol == gatewayapi_v1alpha1.HTTPSProtocolType ||
			listener.Protocol == gatewayapi_v1alpha1.TLSProtocolType {
			if listener.TLS == nil {
				return fmt.Errorf("listener TLS is required when protocol is %q", listener.Protocol)
			}
			// Validate the listener TLS config.
			if err := listenerTLS(listener); err != nil {
				return fmt.Errorf("invalid listener TLS: %w", err)
			}
		}

		// Validate the Group on the selector is a supported type.
		if listener.Routes.Group != nil {
			if *listener.Routes.Group != gatewayapi_v1alpha1.GroupName {
				return fmt.Errorf("listener routes group %q is not supported", listener.Routes.Group)
			}
		}

		// Validate the Kind on the selector is a supported type.
		if listener.Routes.Kind != KindHTTPRoute {
			return fmt.Errorf("listener routes kind %q is not supported", listener.Routes.Kind)
		}
	}

	return nil
}

func listenerTLS(listener gatewayapi_v1alpha1.Listener) error {
	// Validate the CertificateRef is configured.
	if listener.TLS.CertificateRef == nil {
		return fmt.Errorf("spec virtualHost.TLS.CertificateRef is not configured.")
		return nil
	}

	// Validate the correct protocol is specified.
	if listener.Protocol != gatewayapi_v1alpha1.HTTPSProtocolType {
		p.Errorf("Spec.VirtualHost.Protocol %q is not valid.", listener.Protocol)
		return nil
	}

	// Validate a v1.Secret is referenced which can be kind: secret & group: core.
	// ref: https://github.com/kubernetes-sigs/gateway-api/pull/562
	if !isSecretRef(listener.TLS.CertificateRef) {
		p.Error("Spec.VirtualHost.TLS Secret must be type core.Secret")
		return nil
	}

	listenerSecret, err := p.source.LookupSecret(types.NamespacedName{Name: listener.TLS.CertificateRef.Name, Namespace: p.source.gateway.Namespace}, validSecret)
	if err != nil {
		p.Errorf("Spec.VirtualHost.TLS Secret %q is invalid: %s", listener.TLS.CertificateRef.Name, err)
		return nil
	}
	return listenerSecret
}

// gatewayAddresses returns an error if any gw addresses are invalid.
func gatewayAddresses(gw *gatewayapi_v1alpha1.Gateway) error {
	if len(gw.Spec.Addresses) > 0 {
		for _, a := range gw.Spec.Addresses {
			if a.Type == nil || *a.Type != gatewayapi_v1alpha1.IPAddressType {
				return fmt.Errorf("invalid address type; only %v is suported", gatewayapi_v1alpha1.IPAddressType)
			}
			if ip := net.ParseIP(a.Value); ip == nil {
				return fmt.Errorf("invalid address value %s", a.Value)
			}
		}
	}
	return nil
}
