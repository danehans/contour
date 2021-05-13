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

// TODO [danehans]: Refactor to use upstream validation pkg.

package validation

import (
	"fmt"

	retryable "github.com/projectcontour/contour/internal/retryableerror"
	"github.com/projectcontour/contour/pkg/config"

	gatewayv1a1 "sigs.k8s.io/gateway-api/apis/v1alpha1"
)

// GatewayClass returns nil if gw is a valid GatewayClass,
// otherwise an error.
func GatewayClass(gc *gatewayv1a1.GatewayClass) error {
	return parameterRef(gc)
}

// parameterRef returns nil if parametersRef of gw is valid,
// otherwise an error.
func parameterRef(gc *gatewayv1a1.GatewayClass) error {
	var errs []error

	if gc.Spec.Controller != config.ContourGatewayClass {
		errs = append(errs, fmt.Errorf("invalid controller %q; value must be %q", gc.Name, config.ContourGatewayClass))
	}

	// ParametersRef is optional. Default config should be used when nil.
	if gc.Spec.ParametersRef != nil {
		errs = append(errs, fmt.Errorf("invalid parametersRef; field must be unspecified"))
	}

	return retryable.NewMaybeRetryableAggregate(errs)
}
