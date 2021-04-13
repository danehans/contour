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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Config is the configuration of an Envoy.
type Config struct {
	Name        string
	Namespace   string
	NetworkType contour_api_v1alpha1.NetworkPublishingType
}

// New makes an Envoy object using the provided ns/name for the object's
// namespace/name, pubType for the Envoy network publishing type, and
// Envoy container ports 8080/8443.
func New(cfg Config) *contour_api_v1alpha1.Envoy {
	e := &contour_api_v1alpha1.Envoy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cfg.Namespace,
			Name:      cfg.Name,
		},
		Spec: contour_api_v1alpha1.EnvoySpec{
			NetworkPublishing: contour_api_v1alpha1.NetworkPublishing{
				Type: cfg.NetworkType,
				ContainerPorts: []contour_api_v1alpha1.ContainerPort{
					{
						Name:       "http",
						PortNumber: int32(8080),
					},
					{
						Name:       "https",
						PortNumber: int32(8443),
					},
				},
			},
		},
	}

	return e
}

// CurrentEnvoy returns the current Envoy for the provided ns/name.
func CurrentEnvoy(ctx context.Context, cli client.Client, ns, name string) (*contour_api_v1alpha1.Envoy, error) {
	e := &contour_api_v1alpha1.Envoy{}
	key := types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}
	if err := cli.Get(ctx, key, e); err != nil {
		return nil, err
	}
	return e, nil
}

// OthersExistInNs lists Envoy objects in the same namespace as the provided envoy,
// returning true if any exist.
func OthersExistInNs(ctx context.Context, cli client.Client, envoy *contour_api_v1alpha1.Envoy) (bool, error) {
	exist, envoys, err := othersExist(ctx, cli, envoy)
	if err != nil {
		return false, err
	}
	if exist {
		for _, e := range envoys.Items {
			if e.Name == envoy.Name && e.Namespace == envoy.Namespace {
				// Skip the envoy from the list that matches the provided envoy.
				continue
			}
			if e.Namespace == envoy.Namespace {
				return true, nil
			}
		}
	}
	return false, nil
}

// othersExist lists Envoy objects in all namespaces, returning the list
// and true if any exist other than contour.
func othersExist(ctx context.Context, cli client.Client, envoy *contour_api_v1alpha1.Envoy) (bool, *contour_api_v1alpha1.EnvoyList, error) {
	envoys := &contour_api_v1alpha1.EnvoyList{}
	if err := cli.List(ctx, envoys); err != nil {
		return false, nil, fmt.Errorf("failed to list envoys: %w", err)
	}
	if len(envoys.Items) == 0 || len(envoys.Items) == 1 && envoys.Items[0].Name == envoy.Name {
		return false, nil, nil
	}
	return true, envoys, nil
}

// ownerLabels returns owner labels for the provided envoy.
func ownerLabels(envoy *contour_api_v1alpha1.Envoy) map[string]string {
	return map[string]string{
		contour_api_v1alpha1.OwningEnvoyNameLabel: envoy.Name,
		contour_api_v1alpha1.OwningEnvoyNsLabel:   envoy.Namespace,
	}
}
