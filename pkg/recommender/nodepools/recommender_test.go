// Copyright © 2019 Banzai Cloud
//
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

package nodepools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_avgSpotNodeCount(t *testing.T) {
	tests := []struct {
		name     string
		minNodes int
		maxNodes int
		odNodes  int
		check    func(resp int)
	}{
		{
			name:     "0 < on demand percentage < 100",
			minNodes: 1,
			maxNodes: 2,
			odNodes:  1,
			check: func(resp int) {
				assert.Equal(t, 1, resp)
			},
		},
		{
			name:     "on demand percentage = 100",
			minNodes: 1,
			maxNodes: 2,
			odNodes:  2,
			check: func(resp int) {
				assert.Equal(t, 0, resp)
			},
		},
		{
			name:     "on demand percentage = 0",
			minNodes: 1,
			maxNodes: 2,
			odNodes:  0,
			check: func(resp int) {
				assert.Equal(t, 2, resp)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(avgSpotNodeCount(test.minNodes, test.maxNodes, test.odNodes))
		})
	}
}
