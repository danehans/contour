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

package slice

import (
	"reflect"
	"testing"
)

func TestRemoveString(t *testing.T) {
	testCases := []struct {
		name   string
		in     []string
		remove string
		out    []string
	}{
		{
			name:   "one string, remove one",
			in:     []string{"one"},
			remove: "one",
			out:    nil,
		},
		{
			name:   "two strings, remove first string",
			in:     []string{"one", "two"},
			remove: "one",
			out:    []string{"two"},
		},
		{
			name:   "two strings, remove second string",
			in:     []string{"one", "two"},
			remove: "two",
			out:    []string{"one"},
		},
		{
			name:   "two strings, remove one that doesn't exist",
			in:     []string{"one", "two"},
			remove: "three",
			out:    []string{"one", "two"},
		},
		{
			name:   "three strings, remove the second string",
			in:     []string{"one", "two", "three"},
			remove: "two",
			out:    []string{"one", "three"},
		},
		{
			name:   "three strings, remove empty string",
			in:     []string{"one", "two", "three"},
			remove: "",
			out:    []string{"one", "two", "three"},
		},
	}

	for _, tc := range testCases {
		out := RemoveString(tc.in, tc.remove)
		if !reflect.DeepEqual(out, tc.out) {
			t.Errorf("%s, expected slice to be %v, got %v", tc.name, tc.out, out)
		}
	}
}
