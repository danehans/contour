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

package equality_test

import (
	"testing"

	contour_api_v1alpha1 "github.com/projectcontour/contour/apis/projectcontour/v1alpha1"
	"github.com/projectcontour/contour/internal/equality"
	k8s_envoy "github.com/projectcontour/contour/internal/k8s/envoy"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	testName = "test"
	cfg      = k8s_envoy.Config{
		Name:        testName,
		Namespace:   "projectcontour",
		NetworkType: contour_api_v1alpha1.LoadBalancerServicePublishingType,
	}
	envoy = k8s_envoy.New(cfg)
)

func TestClusterIpServiceChanged(t *testing.T) {
	testCases := []struct {
		description string
		mutate      func(service *corev1.Service)
		expect      bool
	}{
		{
			description: "if nothing changed",
			mutate:      func(_ *corev1.Service) {},
			expect:      false,
		},
		{
			description: "if the port number changed",
			mutate: func(svc *corev1.Service) {
				svc.Spec.Ports[0].Port = int32(1234)
			},
			expect: true,
		},
		{
			description: "if the target port number changed",
			mutate: func(svc *corev1.Service) {
				intStrPort := intstr.IntOrString{IntVal: int32(1234)}
				svc.Spec.Ports[0].TargetPort = intStrPort
			},
			expect: true,
		},
		{
			description: "if the port name changed",
			mutate: func(svc *corev1.Service) {
				svc.Spec.Ports[0].Name = "foo"
			},
			expect: true,
		},
		{
			description: "if the port protocol changed",
			mutate: func(svc *corev1.Service) {
				svc.Spec.Ports[0].Protocol = corev1.ProtocolUDP
			},
			expect: true,
		},
		{
			description: "if ports are added",
			mutate: func(svc *corev1.Service) {
				port := corev1.ServicePort{
					Name:       "foo",
					Protocol:   corev1.ProtocolUDP,
					Port:       int32(1234),
					TargetPort: intstr.IntOrString{IntVal: int32(1234)},
				}
				svc.Spec.Ports = append(svc.Spec.Ports, port)
			},
			expect: true,
		},
		{
			description: "if ports are removed",
			mutate: func(svc *corev1.Service) {
				svc.Spec.Ports = []corev1.ServicePort{}
			},
			expect: true,
		},
		{
			description: "if the cluster IP changed",
			mutate: func(svc *corev1.Service) {
				svc.Spec.ClusterIP = "1.2.3.4"
			},
			expect: false,
		},
		{
			description: "if selector changed",
			mutate: func(svc *corev1.Service) {
				svc.Spec.Selector = map[string]string{"foo": "bar"}
			},
			expect: true,
		},
		{
			description: "if service type changed",
			mutate: func(svc *corev1.Service) {
				svc.Spec.Type = corev1.ServiceTypeNodePort
			},
			expect: true,
		},
		{
			description: "if session affinity changed",
			mutate: func(svc *corev1.Service) {
				svc.Spec.SessionAffinity = corev1.ServiceAffinityClientIP
			},
			expect: true,
		},
	}

	for _, tc := range testCases {
		expected := k8s_envoy.DesiredService(envoy)

		mutated := expected.DeepCopy()
		tc.mutate(mutated)
		if updated, changed := equality.ClusterIPServiceChanged(mutated, expected); changed != tc.expect {
			t.Errorf("%s, expect ClusterIpServiceChanged to be %t, got %t", tc.description, tc.expect, changed)
		} else if changed {
			if _, changedAgain := equality.ClusterIPServiceChanged(updated, expected); changedAgain {
				t.Errorf("%s, ClusterIpServiceChanged does not behave as a fixed point function", tc.description)
			}
		}
	}
}

func TestLoadBalancerServiceChanged(t *testing.T) {
	testCases := []struct {
		description string
		mutate      func(service *corev1.Service)
		expect      bool
	}{
		{
			description: "if nothing changed",
			mutate:      func(_ *corev1.Service) {},
			expect:      false,
		},
		{
			description: "if the port number changed",
			mutate: func(svc *corev1.Service) {
				svc.Spec.Ports[0].Port = int32(1234)
			},
			expect: true,
		},
		{
			description: "if the target port number changed",
			mutate: func(svc *corev1.Service) {
				intStrPort := intstr.IntOrString{IntVal: int32(1234)}
				svc.Spec.Ports[0].TargetPort = intStrPort
			},
			expect: true,
		},
		{
			description: "if the port name changed",
			mutate: func(svc *corev1.Service) {
				svc.Spec.Ports[0].Name = "foo"
			},
			expect: true,
		},
		{
			description: "if the port protocol changed",
			mutate: func(svc *corev1.Service) {
				svc.Spec.Ports[0].Protocol = corev1.ProtocolUDP
			},
			expect: true,
		},
		{
			description: "if ports are added",
			mutate: func(svc *corev1.Service) {
				port := corev1.ServicePort{
					Name:       "foo",
					Protocol:   corev1.ProtocolUDP,
					Port:       int32(1234),
					TargetPort: intstr.IntOrString{IntVal: int32(1234)},
				}
				svc.Spec.Ports = append(svc.Spec.Ports, port)
			},
			expect: true,
		},
		{
			description: "if ports are removed",
			mutate: func(svc *corev1.Service) {
				svc.Spec.Ports = []corev1.ServicePort{}
			},
			expect: true,
		},
		{
			description: "if the cluster IP changed",
			mutate: func(svc *corev1.Service) {
				svc.Spec.ClusterIP = "1.2.3.4"
			},
			expect: false,
		},
		{
			description: "if selector changed",
			mutate: func(svc *corev1.Service) {
				svc.Spec.Selector = map[string]string{"foo": "bar"}
			},
			expect: true,
		},
		{
			description: "if service type changed",
			mutate: func(svc *corev1.Service) {
				svc.Spec.Type = corev1.ServiceTypeClusterIP
			},
			expect: true,
		},
		{
			description: "if session affinity changed",
			mutate: func(svc *corev1.Service) {
				svc.Spec.SessionAffinity = corev1.ServiceAffinityClientIP
			},
			expect: true,
		},
		{
			description: "if external traffic policy changed",
			mutate: func(svc *corev1.Service) {
				svc.Spec.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicyTypeCluster
			},
			expect: true,
		},
		{
			description: "if annotations have changed",
			mutate: func(svc *corev1.Service) {
				svc.Annotations = map[string]string{}
			},
			expect: true,
		},
	}

	for _, tc := range testCases {
		envoy.Spec.NetworkPublishing.Type = contour_api_v1alpha1.LoadBalancerServicePublishingType
		envoy.Spec.NetworkPublishing.LoadBalancer.Scope = contour_api_v1alpha1.ExternalLoadBalancer
		envoy.Spec.NetworkPublishing.LoadBalancer.ProviderParameters.Type = contour_api_v1alpha1.AWSLoadBalancerProvider
		envoy.Spec.NetworkPublishing.ContainerPorts = []contour_api_v1alpha1.ContainerPort{
			{
				Name:       "http",
				PortNumber: int32(80),
			},
			{
				Name:       "https",
				PortNumber: int32(80),
			},
			{
				Name:       "https",
				PortNumber: int32(443),
			},
		}
		expected := k8s_envoy.DesiredService(envoy)

		mutated := expected.DeepCopy()
		tc.mutate(mutated)
		if updated, changed := equality.LoadBalancerServiceChanged(mutated, expected); changed != tc.expect {
			t.Errorf("%s, expect LoadBalancerServiceChanged to be %t, got %t", tc.description, tc.expect, changed)
		} else if changed {
			if _, changedAgain := equality.LoadBalancerServiceChanged(updated, expected); changedAgain {
				t.Errorf("%s, LoadBalancerServiceChanged does not behave as a fixed point function", tc.description)
			}
		}
	}
}

func TestNodePortServiceChanged(t *testing.T) {
	testCases := []struct {
		description string
		mutate      func(service *corev1.Service)
		expect      bool
	}{
		{
			description: "if nothing changed",
			mutate:      func(_ *corev1.Service) {},
			expect:      false,
		},
		{
			description: "if the nodeport port number changed",
			mutate: func(svc *corev1.Service) {
				svc.Spec.Ports[0].NodePort = int32(1234)
			},
			expect: true,
		},
	}

	for _, tc := range testCases {
		envoy.Spec.NetworkPublishing.Type = contour_api_v1alpha1.NodePortServicePublishingType
		envoy.Spec.NetworkPublishing.ContainerPorts = []contour_api_v1alpha1.ContainerPort{
			{
				Name:       "http",
				PortNumber: int32(80),
			},
			{
				Name:       "https",
				PortNumber: int32(443),
			},
			{
				Name:       "https",
				PortNumber: int32(443),
			},
		}
		expected := k8s_envoy.DesiredService(envoy)

		mutated := expected.DeepCopy()
		tc.mutate(mutated)
		if updated, changed := equality.NodePortServiceChanged(mutated, expected); changed != tc.expect {
			t.Errorf("%s, expect NodePortServiceChanged to be %t, got %t", tc.description, tc.expect, changed)
		} else if changed {
			if _, changedAgain := equality.NodePortServiceChanged(updated, expected); changedAgain {
				t.Errorf("%s, NodePortServiceChanged does not behave as a fixed point function", tc.description)
			}
		}
	}
}
