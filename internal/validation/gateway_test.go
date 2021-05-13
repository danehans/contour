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

	gatewayv1a1 "sigs.k8s.io/gateway-api/apis/v1alpha1"
)

var (
	fooHost = gatewayv1a1.Hostname("foo")
	ipHost = gatewayv1a1.Hostname("1.2.3.4")
	subDomainHost = gatewayv1a1.Hostname("my.subdomain.local")
	wildSubDomainHost = gatewayv1a1.Hostname("*.subdomain.local")
)

func TestGatewayListeners(t *testing.T) {
	testCases := []struct {
		name      string
		listeners []gatewayv1a1.Listener
		expect    bool
	}{
		{
			name: "one http listener",
			listeners: []gatewayv1a1.Listener{
				{
					Hostname: &fooHost,
					Port:     gatewayv1a1.PortNumber(1),
					Protocol: gatewayv1a1.HTTPProtocolType,
				},
			},
			expect: true,
		},
		{
			name: "one http listener, one https listener",
			listeners: []gatewayv1a1.Listener{
				{
					Hostname: &fooHost,
					Port:     gatewayv1a1.PortNumber(1),
					Protocol: gatewayv1a1.HTTPProtocolType,
				},
				{
					Hostname: &fooHost,
					Port:     gatewayv1a1.PortNumber(2),
					Protocol: gatewayv1a1.HTTPSProtocolType,
				},
			},
			expect: true,
		},
		{
			name: "conflicting ports listener",
			listeners: []gatewayv1a1.Listener{
				{
					Hostname: &fooHost,
					Port:     gatewayv1a1.PortNumber(1),
					Protocol: gatewayv1a1.HTTPProtocolType,
				},
				{
					Hostname: &fooHost,
					Port:     gatewayv1a1.PortNumber(1),
					Protocol: gatewayv1a1.HTTPSProtocolType,
				},
			},
			expect: false,
		},
		{
			name: "IP address hostname listener",
			listeners: []gatewayv1a1.Listener{
				{
					Hostname: &ipHost,
					Port:     gatewayv1a1.PortNumber(1),
					Protocol: gatewayv1a1.HTTPProtocolType,
				},
			},
			expect: false,
		},
		{
			name: "sub domain hostname listener",
			listeners: []gatewayv1a1.Listener{
				{
					Hostname: &subDomainHost,
					Port:     gatewayv1a1.PortNumber(1),
					Protocol: gatewayv1a1.HTTPProtocolType,
				},
			},
			expect: true,
		},
		{
			name: "wildcard sub domain hostname listener",
			listeners: []gatewayv1a1.Listener{
				{
					Hostname: &wildSubDomainHost,
					Port:     gatewayv1a1.PortNumber(1),
					Protocol: gatewayv1a1.HTTPProtocolType,
				},
			},
			expect: true,
		},
	}

	for _, tc := range testCases {
		gw := &gatewayv1a1.Gateway{Spec: gatewayv1a1.GatewaySpec{Listeners: tc.listeners}}
		actual := gatewayListeners(gw)
		if actual != nil && tc.expect {
			t.Fatalf("%q: expected %#v, got %#v", tc.name, tc.expect, actual)
		}
		if actual == nil && !tc.expect {
			t.Fatalf("%q: expected %#v, got %#v", tc.name, tc.expect, actual)
		}
	}
}
