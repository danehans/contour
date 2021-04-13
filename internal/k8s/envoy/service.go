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

package envoy

import (
	"context"
	"fmt"

	contour_api_v1alpha1 "github.com/projectcontour/contour/apis/projectcontour/v1alpha1"
	"github.com/projectcontour/contour/internal/equality"
	"github.com/projectcontour/contour/pkg/labels"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	envoySvcName = "envoy"
	// awsLbBackendProtoAnnotation is a Service annotation that places the AWS ELB into
	// "TCP" mode so that it does not do HTTP negotiation for HTTPS connections at the
	// ELB edge. The downside of this is the remote IP address of all connections will
	// appear to be the internal address of the ELB.
	// TODO [danehans]: Make proxy protocol configurable or automatically enabled. See
	// https://github.com/projectcontour/contour-operator/issues/49 for details.
	awsLbBackendProtoAnnotation = "service.beta.kubernetes.io/aws-load-balancer-backend-protocol"
	// awsProviderType is the name of the Amazon Web Services provider.
	awsProviderType = "AWS"
	// azureProviderType is the name of the Microsoft Azure provider.
	azureProviderType = "Azure"
	// gcpProviderType is the name of the Google Cloud Platform provider.
	gcpProviderType = "GCP"
	// awsInternalLBAnnotation is the annotation used on a service to specify an AWS
	// load balancer as being internal.
	awsInternalLBAnnotation = "service.beta.kubernetes.io/aws-load-balancer-internal"
	// azureInternalLBAnnotation is the annotation used on a service to specify an Azure
	// load balancer as being internal.
	azureInternalLBAnnotation = "service.beta.kubernetes.io/azure-load-balancer-internal"
	// gcpLBTypeAnnotation is the annotation used on a service to specify a GCP load balancer
	// type.
	gcpLBTypeAnnotation = "cloud.google.com/load-balancer-type"
	// EnvoyServiceHTTPPort is the HTTP port number of the Envoy service.
	EnvoyServiceHTTPPort = int32(80)
	// EnvoyServiceHTTPSPort is the HTTPS port number of the Envoy service.
	EnvoyServiceHTTPSPort = int32(443)
	// EnvoyNodePortHTTPPort is the NodePort port number for Envoy's HTTP service. For NodePort
	// details see: https://kubernetes.io/docs/concepts/services-networking/service/#nodeport
	EnvoyNodePortHTTPPort = int32(30080)
	// EnvoyNodePortHTTPSPort is the NodePort port number for Envoy's HTTPS service. For NodePort
	// details see: https://kubernetes.io/docs/concepts/services-networking/service/#nodeport
	EnvoyNodePortHTTPSPort = int32(30443)
)

var (
	// LbAnnotations maps cloud providers to the provider's annotation
	// key/value pair used for managing a load balancer. For additional
	// details see:
	//  https://kubernetes.io/docs/concepts/services-networking/service/#internal-load-balancer
	//
	LbAnnotations = map[contour_api_v1alpha1.LoadBalancerProviderType]map[string]string{
		awsProviderType: {
			awsLbBackendProtoAnnotation: "tcp",
		},
	}

	// InternalLBAnnotations maps cloud providers to the provider's annotation
	// key/value pair used for managing an internal load balancer. For additional
	// details see:
	//  https://kubernetes.io/docs/concepts/services-networking/service/#internal-load-balancer
	//
	InternalLBAnnotations = map[contour_api_v1alpha1.LoadBalancerProviderType]map[string]string{
		awsProviderType: {
			awsInternalLBAnnotation: "0.0.0.0/0",
		},
		azureProviderType: {
			// Azure load balancers are not customizable and are set to (2 fail @ 5s interval, 2 healthy)
			azureInternalLBAnnotation: "true",
		},
		gcpProviderType: {
			gcpLBTypeAnnotation: "Internal",
		},
	}
)

// EnsureService ensures that a Service resource exists for the given envoy.
func EnsureService(ctx context.Context, cli client.Client, envoy *contour_api_v1alpha1.Envoy) error {
	desired := DesiredService(envoy)
	current, err := currentService(ctx, cli, envoy)
	if err != nil {
		if errors.IsNotFound(err) {
			return createService(ctx, cli, desired)
		}
		return fmt.Errorf("failed to get service %s/%s: %w", desired.Namespace, desired.Name, err)
	}
	if err := updateServiceIfNeeded(ctx, cli, envoy, current, desired); err != nil {
		return fmt.Errorf("failed to update service %s/%s: %w", desired.Namespace, desired.Name, err)
	}
	return nil
}

// EnsureServiceDeleted ensures that the Service resource for the provided envoy is deleted.
func EnsureServiceDeleted(ctx context.Context, cli client.Client, envoy *contour_api_v1alpha1.Envoy) error {
	svc, err := currentService(ctx, cli, envoy)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if labels.Exist(svc, ownerLabels(envoy)) {
		if err := cli.Delete(ctx, svc); err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
	}
	return nil
}

// DesiredService generates the desired Service for the given envoy.
func DesiredService(envoy *contour_api_v1alpha1.Envoy) *corev1.Service {
	var ports []corev1.ServicePort
	for _, port := range envoy.Spec.NetworkPublishing.ContainerPorts {
		var p corev1.ServicePort
		httpFound := false
		httpsFound := false
		switch {
		case httpsFound && httpFound:
			break
		case port.Name == "http":
			httpFound = true
			p.Name = port.Name
			p.Port = EnvoyServiceHTTPPort
			p.Protocol = corev1.ProtocolTCP
			p.TargetPort = intstr.IntOrString{IntVal: port.PortNumber}
			ports = append(ports, p)
		case port.Name == "https":
			httpsFound = true
			p.Name = port.Name
			p.Port = EnvoyServiceHTTPSPort
			p.Protocol = corev1.ProtocolTCP
			p.TargetPort = intstr.IntOrString{IntVal: port.PortNumber}
			ports = append(ports, p)
		}
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   envoy.Namespace,
			Name:        envoySvcName,
			Annotations: map[string]string{},
			Labels: map[string]string{
				contour_api_v1alpha1.OwningEnvoyNameLabel: envoy.Name,
				contour_api_v1alpha1.OwningEnvoyNsLabel:   envoy.Namespace,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports:           ports,
			Selector:        labelSelector().MatchLabels,
			SessionAffinity: corev1.ServiceAffinityNone,
		},
	}
	epType := envoy.Spec.NetworkPublishing.Type
	if epType == contour_api_v1alpha1.LoadBalancerServicePublishingType ||
		epType == contour_api_v1alpha1.NodePortServicePublishingType {
		svc.Spec.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicyTypeLocal
	}
	switch epType {
	case contour_api_v1alpha1.LoadBalancerServicePublishingType:
		svc.Spec.Type = corev1.ServiceTypeLoadBalancer
		provider := envoy.Spec.NetworkPublishing.LoadBalancer.ProviderParameters.Type
		lbAnnotations := LbAnnotations[provider]
		for name, value := range lbAnnotations {
			svc.Annotations[name] = value
		}
		isInternal := envoy.Spec.NetworkPublishing.LoadBalancer.Scope == contour_api_v1alpha1.InternalLoadBalancer
		if isInternal {
			internalAnnotations := InternalLBAnnotations[provider]
			for name, value := range internalAnnotations {
				svc.Annotations[name] = value
			}
		}
	case contour_api_v1alpha1.NodePortServicePublishingType:
		svc.Spec.Type = corev1.ServiceTypeNodePort
		svc.Spec.Ports[0].NodePort = EnvoyNodePortHTTPPort
		svc.Spec.Ports[1].NodePort = EnvoyNodePortHTTPSPort
	case contour_api_v1alpha1.ClusterIPServicePublishingType:
		svc.Spec.Type = corev1.ServiceTypeClusterIP
	}
	return svc
}

// currentService returns the current Envoy Service for the provided envoy.
func currentService(ctx context.Context, cli client.Client, envoy *contour_api_v1alpha1.Envoy) (*corev1.Service, error) {
	current := &corev1.Service{}
	key := types.NamespacedName{
		Namespace: envoy.Namespace,
		Name:      envoySvcName,
	}
	err := cli.Get(ctx, key, current)
	if err != nil {
		return nil, err
	}
	return current, nil
}

// createService creates a Service resource for the provided svc.
func createService(ctx context.Context, cli client.Client, svc *corev1.Service) error {
	if err := cli.Create(ctx, svc); err != nil {
		return fmt.Errorf("failed to create service %s/%s: %w", svc.Namespace, svc.Name, err)
	}
	return nil
}

// updateServiceIfNeeded updates a Service resource if current doesn't match desired,
// using nvoy to verify the existence of owner labels.
func updateServiceIfNeeded(ctx context.Context, cli client.Client, envoy *contour_api_v1alpha1.Envoy, current, desired *corev1.Service) error {
	if labels.Exist(current, ownerLabels(envoy)) {
		updated := false
		switch envoy.Spec.NetworkPublishing.Type {
		case contour_api_v1alpha1.NodePortServicePublishingType:
			_, updated = equality.NodePortServiceChanged(current, desired)
		case contour_api_v1alpha1.ClusterIPServicePublishingType:
			_, updated = equality.ClusterIPServiceChanged(current, desired)
		// Add additional network publishing types as they are introduced.
		default:
			// LoadBalancerService is the default network publishing type.
			_, updated = equality.LoadBalancerServiceChanged(current, desired)
		}
		if updated {
			if err := cli.Update(ctx, desired); err != nil {
				return fmt.Errorf("failed to update service %q in namespace %q: %w", desired.Name, desired.Namespace, err)
			}
			return nil
		}
	}
	return nil
}

// labelSelector returns a label selector using "app: envoy" as the key/value pair.
// TODO [danehans]: Update when https://github.com/projectcontour/contour/issues/1821 is fixed.
func labelSelector() *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app": "envoy",
		},
	}
}
