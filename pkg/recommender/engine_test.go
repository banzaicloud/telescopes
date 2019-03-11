// Copyright © 2018 Banzai Cloud
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

package recommender

import (
	"testing"

	"github.com/goph/logur"
	"github.com/stretchr/testify/assert"
)

type dummyNodePools struct {
	// test case id to drive the behaviour
	TcId string
}

func (nps *dummyNodePools) RecommendNodePools(provider, service, region string, req ClusterRecommendationReq, log logur.Logger, layoutDesc []NodePoolDesc) (map[string][]NodePool, error) {
	return map[string][]NodePool{
		Memory: {
			NodePool{ // price = 2*3 +2*2 = 10
				VmType: VirtualMachine{
					Cpus:          16,
					Mem:           42,
					AvgPrice:      2,
					OnDemandPrice: 3,
				},
				SumNodes: 0,
				VmClass:  Regular,
			}, NodePool{
				VmType: VirtualMachine{
					Cpus:          16,
					Mem:           42,
					AvgPrice:      2,
					OnDemandPrice: 3,
				},
				SumNodes: 1,
				VmClass:  Spot,
			},
		},
		Cpu: { // price = 2*2 +2*2 = 8
			NodePool{
				VmType: VirtualMachine{
					Cpus:          8,
					Mem:           21,
					AvgPrice:      1,
					OnDemandPrice: 2,
				},
				SumNodes: 0,
				VmClass:  Regular,
			}, NodePool{
				VmType: VirtualMachine{
					Cpus:          16,
					Mem:           42,
					AvgPrice:      2,
					OnDemandPrice: 4,
				},
				SumNodes: 1,
				VmClass:  Spot,
			}, NodePool{
				VmType: VirtualMachine{
					Cpus:          16,
					Mem:           42,
					AvgPrice:      2,
					OnDemandPrice: 4,
				},
				SumNodes: 0,
				VmClass:  Spot,
			},
		},
	}, nil
}

func TestEngine_RecommendCluster(t *testing.T) {
	tests := []struct {
		name    string
		nps     NodePoolRecommender
		request ClusterRecommendationReq
		check   func(resp *ClusterRecommendationResp, err error)
	}{
		{
			name: "cluster recommendation success",
			request: ClusterRecommendationReq{
				MinNodes: 1,
				MaxNodes: 1,
				SumMem:   32,
				SumCpu:   16,
			},
			nps: &dummyNodePools{},
			check: func(resp *ClusterRecommendationResp, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, float64(42), resp.Accuracy.RecMem)
				assert.Equal(t, float64(16), resp.Accuracy.RecCpu)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine := NewEngine(test.nps, logur.NewTestLogger())

			test.check(engine.RecommendCluster("dummyProvider", "dummyService", "dummyRegion", test.request, nil, logur.NewTestLogger()))
		})
	}
}

func TestEngine_findCheapestNodePoolSet(t *testing.T) {
	tests := []struct {
		name      string
		nodePools map[string][]NodePool
		check     func(npo []NodePool)
	}{
		{
			name: "find cheapest node pool set",
			nodePools: map[string][]NodePool{
				Memory: {
					NodePool{ // price = 2*3 +2*2 = 10
						VmType: VirtualMachine{
							AvgPrice:      2,
							OnDemandPrice: 3,
						},
						SumNodes: 2,
						VmClass:  Regular,
					}, NodePool{
						VmType: VirtualMachine{
							AvgPrice:      2,
							OnDemandPrice: 3,
						},
						SumNodes: 2,
						VmClass:  Spot,
					},
				},
				Cpu: { // price = 2*2 +2*2 = 8
					NodePool{
						VmType: VirtualMachine{
							AvgPrice:      1,
							OnDemandPrice: 2,
						},
						SumNodes: 2,
						VmClass:  Regular,
					}, NodePool{
						VmType: VirtualMachine{
							AvgPrice:      2,
							OnDemandPrice: 4,
						},
						SumNodes: 2,
						VmClass:  Spot,
					}, NodePool{
						VmType: VirtualMachine{
							AvgPrice:      2,
							OnDemandPrice: 4,
						},
						SumNodes: 0,
						VmClass:  Spot,
					},
				},
			},
			check: func(npo []NodePool) {
				assert.Equal(t, 3, len(npo), "wrong selection")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine := NewEngine(nil, logur.NewTestLogger())
			test.check(engine.findCheapestNodePoolSet(test.nodePools))
		})
	}
}
