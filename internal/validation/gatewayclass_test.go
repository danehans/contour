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
	"testing"

	gatewayapi_v1alpha1 "sigs.k8s.io/gateway-api/apis/v1alpha1"
)

func TestGatewayClass(t *testing.T) {
	testCases := []struct {
		name   string
		gc     *gatewayapi_v1alpha1.GatewayClass
		expect bool
	}{
		{
			name: "invalid gatewayclass params",
			gc: &gatewayapi_v1alpha1.GatewayClass{
				Spec: gatewayapi_v1alpha1.GatewayClassSpec{
					Controller: "foo",
					ParametersRef: &gatewayapi_v1alpha1.ParametersReference{
						Group: "foo-group",
						Kind:  "foo-kind",
						Name:  "foo",
					},
				},
			},
			expect: false,
		},
	}

	for _, tc := range testCases {
		actual := GatewayClass(tc.gc)
		if actual != nil && tc.expect {
			t.Fatalf("%q: expected %#v, got %#v", tc.name, tc.expect, actual)
		}
		if actual == nil && !tc.expect {
			t.Fatalf("%q: expected %#v, got %#v", tc.name, tc.expect, actual)
		}
	}
}
