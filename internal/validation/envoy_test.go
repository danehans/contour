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
	"fmt"
	"testing"

	contour_api_v1alpha1 "github.com/projectcontour/contour/apis/projectcontour/v1alpha1"

	k8s_envoy "github.com/projectcontour/contour/internal/k8s/envoy"
)

const (
	insecureContainerPort = int32(8080)
	secureContainerPort   = int32(8443)
)

func TestContainerPorts(t *testing.T) {
	testCases := []struct {
		description string
		ports       []contour_api_v1alpha1.ContainerPort
		expected    bool
	}{
		{
			description: "default http and https port",
			expected:    true,
		},
		{
			description: "non-default http and https ports",
			ports: []contour_api_v1alpha1.ContainerPort{
				{
					Name:       "http",
					PortNumber: int32(8081),
				},
				{
					Name:       "https",
					PortNumber: int32(8444),
				},
			},
			expected: true,
		},
		{
			description: "duplicate port names",
			ports: []contour_api_v1alpha1.ContainerPort{
				{
					Name:       "http",
					PortNumber: insecureContainerPort,
				},
				{
					Name:       "http",
					PortNumber: secureContainerPort,
				},
			},
			expected: false,
		},
		{
			description: "duplicate port numbers",
			ports: []contour_api_v1alpha1.ContainerPort{
				{
					Name:       "http",
					PortNumber: insecureContainerPort,
				},
				{
					Name:       "https",
					PortNumber: insecureContainerPort,
				},
			},
			expected: false,
		},
		{
			description: "only http port specified",
			ports: []contour_api_v1alpha1.ContainerPort{
				{
					Name:       "http",
					PortNumber: insecureContainerPort,
				},
			},
			expected: false,
		},
		{
			description: "only https port specified",
			ports: []contour_api_v1alpha1.ContainerPort{
				{
					Name:       "https",
					PortNumber: secureContainerPort,
				},
			},
			expected: false,
		},
		{
			description: "empty ports",
			ports:       []contour_api_v1alpha1.ContainerPort{},
			expected:    false,
		},
	}

	name := "test-validation"
	cfg := k8s_envoy.Config{
		Name:        name,
		Namespace:   fmt.Sprintf("%s-ns", name),
		NetworkType: contour_api_v1alpha1.LoadBalancerServicePublishingType,
	}
	envoy := k8s_envoy.New(cfg)
	for _, tc := range testCases {
		if tc.ports != nil {
			envoy.Spec.NetworkPublishing.ContainerPorts = tc.ports
		}
		err := containerPorts(envoy)
		if err != nil && tc.expected {
			t.Fatalf("%q: failed with error: %#v", tc.description, err)
		}
		if err == nil && !tc.expected {
			t.Fatalf("%q: expected to fail but received no error", tc.description)
		}
	}
}
