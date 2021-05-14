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

	gatewayapi_v1alpha1 "sigs.k8s.io/gateway-api/apis/v1alpha1"
)

// GatewayClass returns nil if gc is a valid GatewayClass, otherwise an error.
func GatewayClass(gc *gatewayapi_v1alpha1.GatewayClass) error {
	return parameterRef(gc)
}

// parameterRef returns nil if parametersRef of gc is valid, otherwise an error.
func parameterRef(gc *gatewayapi_v1alpha1.GatewayClass) error {
	// ParametersRef is optional. Default config should be used when nil.
	if gc.Spec.ParametersRef != nil {
		return fmt.Errorf("invalid parametersRef; field must be unspecified")
	}

	return nil
}
