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
	"github.com/projectcontour/contour/pkg/slice"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var finalizer = contour_api_v1alpha1.EnvoyFinalizer

// EnsureFinalizer ensures the finalizer is added to the provided envoy.
func EnsureFinalizer(ctx context.Context, cli client.Client, envoy *contour_api_v1alpha1.Envoy) error {
	if !slice.ContainsString(envoy.Finalizers, finalizer) {
		updated := envoy.DeepCopy()
		updated.Finalizers = append(updated.Finalizers, finalizer)
		if err := cli.Update(ctx, updated); err != nil {
			return fmt.Errorf("failed to add finalizer %s to envoy %s/%s: %w",
				finalizer, envoy.Namespace, envoy.Name, err)
		}
	}
	return nil
}

// EnsureFinalizerRemoved ensures the finalizer is removed from the provided envoy.
func EnsureFinalizerRemoved(ctx context.Context, cli client.Client, envoy *contour_api_v1alpha1.Envoy) error {
	if slice.ContainsString(envoy.Finalizers, finalizer) {
		updated := envoy.DeepCopy()
		updated.Finalizers = slice.RemoveString(updated.Finalizers, finalizer)
		if err := cli.Update(ctx, updated); err != nil {
			return fmt.Errorf("failed to remove finalizer %s from contour %s/%s: %w",
				finalizer, envoy.Namespace, envoy.Name, err)
		}
	}
	return nil
}
