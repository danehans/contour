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

	contour_api_v1alpha1 "github.com/projectcontour/contour/apis/projectcontour/v1alpha1"
	"github.com/projectcontour/contour/internal/k8s"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ExtensionCacheEntry holds status updates for a particular ExtensionService
type EnvoyCacheEntry struct {
	ConditionCache

	Name           types.NamespacedName
	Generation     int64
	TransitionTime v1.Time
}

var _ CacheEntry = &EnvoyCacheEntry{}

func (e *EnvoyCacheEntry) AsStatusUpdate() k8s.StatusUpdate {
	m := k8s.StatusMutatorFunc(func(obj interface{}) interface{} {
		o, ok := obj.(*contour_api_v1alpha1.Envoy)
		if !ok {
			panic(fmt.Sprintf("unsupported %T object %q in status mutator", obj, e.Name))
		}

		envoy := o.DeepCopy()

		for condType, cond := range e.Conditions {
			cond.ObservedGeneration = e.Generation
			cond.LastTransitionTime = e.TransitionTime

			currCond := envoy.Status.GetConditionFor(string(condType))
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
	})

	return k8s.StatusUpdate{
		NamespacedName: e.Name,
		Resource:       contour_api_v1alpha1.EnvoyGVR,
		Mutator:        m,
	}
}

// EnvoyAccessor returns a pointer to a shared status cache entry
// for the given ExtensionStatus object. If no such entry exists, a
// new entry is added. When the caller finishes with the cache entry,
// it must call the returned function to release the entry back to the
// cache.
func EnvoyAccessor(c *Cache, envoy *contour_api_v1alpha1.Envoy) (*EnvoyCacheEntry, func()) {
	entry := c.Get(envoy)
	if entry == nil {
		entry = &EnvoyCacheEntry{
			Name:           k8s.NamespacedNameOf(envoy),
			Generation:     envoy.GetGeneration(),
			TransitionTime: v1.NewTime(time.Now()),
		}

		// Populate the cache with the new entry
		c.Put(envoy, entry)
	}

	entry = c.Get(envoy)
	return entry.(*EnvoyCacheEntry), func() {
		c.Put(envoy, entry)
	}
}
