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

	contour_api_v1alpha1 "github.com/projectcontour/contour/apis/projectcontour/v1alpha1"
	k8s_envoy "github.com/projectcontour/contour/internal/k8s/envoy"
	"github.com/projectcontour/contour/pkg/slice"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Envoy returns true if envoy is valid.
func Envoy(ctx context.Context, cli client.Client, envoy *contour_api_v1alpha1.Envoy) error {
	// Only 1 envoy per namespace is supported.
	exist, err := k8s_envoy.OthersExistInNs(ctx, cli, envoy)
	if err != nil {
		return fmt.Errorf("failed to verify if other envoys exist in namespace %q: %w",
			envoy.Namespace, err)
	}
	if exist {
		return fmt.Errorf("other envoys exist in namespace %q", envoy.Namespace)
	}
	return containerPorts(envoy)
}

// containerPorts validates container ports of envoy, returning an
// error if the container ports do not meet the API specification.
func containerPorts(envoy *contour_api_v1alpha1.Envoy) error {
	var numsFound []int32
	var namesFound []string
	httpFound := false
	httpsFound := false
	for _, port := range envoy.Spec.NetworkPublishing.ContainerPorts {
		if len(numsFound) > 0 && slice.ContainsInt32(numsFound, port.PortNumber) {
			return fmt.Errorf("duplicate container port number %q", port.PortNumber)
		}
		numsFound = append(numsFound, port.PortNumber)
		if len(namesFound) > 0 && slice.ContainsString(namesFound, port.Name) {
			return fmt.Errorf("duplicate container port name %q", port.Name)
		}
		namesFound = append(namesFound, port.Name)
		switch {
		case port.Name == "http":
			httpFound = true
		case port.Name == "https":
			httpsFound = true
		}
	}
	if httpFound && httpsFound {
		return nil
	}
	return fmt.Errorf("http and https container ports are unspecified")
}
