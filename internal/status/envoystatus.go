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

package status

import (
	"fmt"
	"time"

	projectcontour "github.com/projectcontour/contour/apis/projectcontour/v1"
	contour_api_v1alpha1 "github.com/projectcontour/contour/apis/projectcontour/v1alpha1"
	"github.com/projectcontour/contour/internal/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const available = contour_api_v1alpha1.EnvoyAvailableConditionType

// EnvoyUpdate holds status updates for a particular Envoy object
type EnvoyUpdate struct {
	Fullname        types.NamespacedName
	Generation      int64
	TransitionTime  v1.Time
	AvailableEnvoys int32

	// Conditions holds all the DetailedConditions to add to the object
	// keyed by the Type (since that's what the apiserver will end up
	// doing.)
	Conditions map[string]*projectcontour.DetailedCondition
}

// EnvoyAccessor returns an EnvoyUpdate that allows a client to build up a list of
// errors and warnings to go onto the envoy as conditions, and a function to commit the change
// back to the cache when everything is done.
// The commit function pattern is used so that the EnvoyUpdate does not need to know any of
// the cache internals.
func (c *Cache) EnvoyAccessor(envoy *contour_api_v1alpha1.Envoy) (*EnvoyUpdate, func()) {
	eu := &EnvoyUpdate{
		Fullname:       k8s.NamespacedNameOf(envoy),
		Generation:     envoy.Generation,
		TransitionTime: metav1.NewTime(time.Now()),
		// TODO [danehans]: Think through how to keep the # of available Envoy's updated, i.e. Envoy controller needs
		// to update available # of Envoys based on the DaemonSet status.
		AvailableEnvoys: envoy.Status.AvailableEnvoys,
		Conditions:      make(map[string]*projectcontour.DetailedCondition),
	}

	return eu, func() {
		c.commitEnvoy(eu)
	}
}

func (c *Cache) commitEnvoy(eu *EnvoyUpdate) {
	if len(eu.Conditions) == 0 {
		return
	}
	c.envoyUpdates[eu.Fullname] = eu
}

// ConditionFor returns a metav1 Condition for the given condition type.
// Currently only the "Available" condition type is supported.
func (eu *EnvoyUpdate) ConditionFor(condType string) *projectcontour.DetailedCondition {
	c, ok := eu.Conditions[condType]
	if !ok {
		new := &projectcontour.DetailedCondition{}
		new.Type = condType
		new.ObservedGeneration = eu.Generation
		if condType == available {
			new.Status = metav1.ConditionTrue
			new.Reason = "EnvoyAvailable"
			new.Message = "At least 1 envoy pod is reporting ready"
		} else {
			new.Status = metav1.ConditionFalse
		}
		eu.Conditions[condType] = new
		return new
	}
	return c

}

func (eu *EnvoyUpdate) Mutate(obj interface{}) interface{} {
	o, ok := obj.(*contour_api_v1alpha1.Envoy)
	if !ok {
		panic(fmt.Sprintf("Unsupported %T object %s/%s in status mutator",
			obj, eu.Fullname.Namespace, eu.Fullname.Name,
		))
	}

	envoy := o.DeepCopy()

	for condType, cond := range eu.Conditions {
		cond.ObservedGeneration = eu.Generation
		cond.LastTransitionTime = eu.TransitionTime

		currCond := envoy.Status.GetConditionFor(condType)
		if currCond == nil {
			envoy.Status.Conditions = append(envoy.Status.Conditions, *cond)
			continue
		}

		// Don't update the condition if our observation is stale.
		if currCond.ObservedGeneration > cond.ObservedGeneration {
			continue
		}

		cond.DeepCopyInto(currCond)

	}

	return envoy
}
