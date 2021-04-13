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

package equality

import (
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
)

// ClusterIPServiceChanged checks if the spec of current and expected match and if not,
// returns true and the expected Service resource. The cluster IP is not compared
// as it's assumed to be dynamically assigned.
func ClusterIPServiceChanged(current, expected *corev1.Service) (*corev1.Service, bool) {
	changed := false
	updated := current.DeepCopy()

	// Spec can't simply be matched since clusterIP is being dynamically assigned.
	if len(current.Spec.Ports) != len(expected.Spec.Ports) {
		updated.Spec.Ports = expected.Spec.Ports
		changed = true
	} else {
		if !apiequality.Semantic.DeepEqual(current.Spec.Ports, expected.Spec.Ports) {
			updated.Spec.Ports = expected.Spec.Ports
			changed = true
		}
	}

	if !apiequality.Semantic.DeepEqual(current.Spec.Selector, expected.Spec.Selector) {
		updated.Spec.Selector = expected.Spec.Selector
		changed = true
	}

	if !apiequality.Semantic.DeepEqual(current.Spec.SessionAffinity, expected.Spec.SessionAffinity) {
		updated.Spec.SessionAffinity = expected.Spec.SessionAffinity
		changed = true
	}

	if !apiequality.Semantic.DeepEqual(current.Spec.Type, expected.Spec.Type) {
		updated.Spec.Type = expected.Spec.Type
		changed = true
	}

	if !changed {
		return nil, false
	}

	return updated, true
}

// LoadBalancerServiceChanged checks if current and expected match and if not, returns
// true and the expected Service resource. The healthCheckNodePort and a port's nodePort
// are not compared since they are dynamically assigned.
func LoadBalancerServiceChanged(current, expected *corev1.Service) (*corev1.Service, bool) {
	changed := false
	updated := current.DeepCopy()

	// Ports can't simply be matched since some fields are being dynamically assigned.
	if len(current.Spec.Ports) != len(expected.Spec.Ports) {
		updated.Spec.Ports = expected.Spec.Ports
		changed = true
	} else {
		for i, p := range current.Spec.Ports {
			if !apiequality.Semantic.DeepEqual(p.Name, expected.Spec.Ports[i].Name) {
				updated.Spec.Ports[i].Name = expected.Spec.Ports[i].Name
				changed = true
			}
			if !apiequality.Semantic.DeepEqual(p.Protocol, expected.Spec.Ports[i].Protocol) {
				updated.Spec.Ports[i].Protocol = expected.Spec.Ports[i].Protocol
				changed = true
			}
			if !apiequality.Semantic.DeepEqual(p.Port, expected.Spec.Ports[i].Port) {
				updated.Spec.Ports[i].Port = expected.Spec.Ports[i].Port
				changed = true
			}
			if !apiequality.Semantic.DeepEqual(p.TargetPort, expected.Spec.Ports[i].TargetPort) {
				updated.Spec.Ports[i].TargetPort = expected.Spec.Ports[i].TargetPort
				changed = true
			}
		}
	}

	if !apiequality.Semantic.DeepEqual(current.Spec.Selector, expected.Spec.Selector) {
		updated.Spec.Selector = expected.Spec.Selector
		changed = true
	}

	if !apiequality.Semantic.DeepEqual(current.Spec.ExternalTrafficPolicy, expected.Spec.ExternalTrafficPolicy) {
		updated.Spec.ExternalTrafficPolicy = expected.Spec.ExternalTrafficPolicy
		changed = true
	}

	if !apiequality.Semantic.DeepEqual(current.Spec.SessionAffinity, expected.Spec.SessionAffinity) {
		updated.Spec.SessionAffinity = expected.Spec.SessionAffinity
		changed = true
	}

	if !apiequality.Semantic.DeepEqual(current.Spec.Type, expected.Spec.Type) {
		updated.Spec.Type = expected.Spec.Type
		changed = true
	}

	if !apiequality.Semantic.DeepEqual(current.Annotations, expected.Annotations) {
		updated.Annotations = expected.Annotations
		changed = true
	}

	if !changed {
		return nil, false
	}

	return updated, true
}

// NodePortServiceChanged checks if current and expected match and if not, returns
// true and the expected Service resource. The healthCheckNodePort is not compared
// since it's dynamically assigned.
func NodePortServiceChanged(current, expected *corev1.Service) (*corev1.Service, bool) {
	changed := false
	updated := current.DeepCopy()

	if len(current.Spec.Ports) != len(expected.Spec.Ports) {
		updated.Spec.Ports = expected.Spec.Ports
		changed = true
	}

	for i, p := range current.Spec.Ports {
		if !apiequality.Semantic.DeepEqual(p, expected.Spec.Ports[i]) {
			updated.Spec.Ports = expected.Spec.Ports
			changed = true
		}
	}

	if !apiequality.Semantic.DeepEqual(current.Spec.Selector, expected.Spec.Selector) {
		updated.Spec.Selector = expected.Spec.Selector
		changed = true
	}

	if !apiequality.Semantic.DeepEqual(current.Spec.ExternalTrafficPolicy, expected.Spec.ExternalTrafficPolicy) {
		updated.Spec.ExternalTrafficPolicy = expected.Spec.ExternalTrafficPolicy
		changed = true
	}

	if !apiequality.Semantic.DeepEqual(current.Spec.SessionAffinity, expected.Spec.SessionAffinity) {
		updated.Spec.SessionAffinity = expected.Spec.SessionAffinity
		changed = true
	}

	if !apiequality.Semantic.DeepEqual(current.Spec.Type, expected.Spec.Type) {
		updated.Spec.Type = expected.Spec.Type
		changed = true
	}

	if !apiequality.Semantic.DeepEqual(current.Annotations, expected.Annotations) {
		updated.Annotations = expected.Annotations
		changed = true
	}

	if !changed {
		return nil, false
	}

	return updated, true
}
