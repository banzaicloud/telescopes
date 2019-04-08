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

package engine

import (
	"testing"

	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/goph/logur"
	"github.com/stretchr/testify/assert"
)

type dummyProducts struct {
	// test case id to drive the behaviour
	TcId string
}

func (p *dummyProducts) GetProductDetails(provider string, service string, region string) ([]recommender.VirtualMachine, error) {
	return []recommender.VirtualMachine{
		{
			Cpus:          16,
			Mem:           42,
			OnDemandPrice: 3,
			AvgPrice:      0.8,
		},
	}, nil
}

func TestEngine_RecommendCluster(t *testing.T) {
	tests := []struct {
		name     string
		ciSource recommender.CloudInfoSource
		request  recommender.ClusterRecommendationReq
		check    func(resp *recommender.ClusterRecommendationResp, err error)
	}{
		{
			name: "cluster recommendation success",
			request: recommender.ClusterRecommendationReq{
				MinNodes: 1,
				MaxNodes: 1,
				SumMem:   32,
				SumCpu:   16,
			},
			ciSource: &dummyProducts{},
			check: func(resp *recommender.ClusterRecommendationResp, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, float64(42), resp.Accuracy.RecMem)
				assert.Equal(t, float64(16), resp.Accuracy.RecCpu)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine := NewEngine(logur.NewTestLogger(), test.ciSource)

			test.check(engine.RecommendCluster("dummyProvider", "dummyService", "dummyRegion", test.request, nil))
		})
	}
}

func TestEngine_findCheapestNodePoolSet(t *testing.T) {
	tests := []struct {
		name      string
		nodePools map[string][]recommender.NodePool
		check     func(nps []recommender.NodePool)
	}{
		{
			name: "find cheapest node pool set",
			nodePools: map[string][]recommender.NodePool{
				recommender.Memory: {
					recommender.NodePool{ // price = 2*3 +2*2 = 10
						VmType: recommender.VirtualMachine{
							AvgPrice:      2,
							OnDemandPrice: 3,
						},
						SumNodes: 2,
						VmClass:  recommender.Regular,
					}, recommender.NodePool{
						VmType: recommender.VirtualMachine{
							AvgPrice:      2,
							OnDemandPrice: 3,
						},
						SumNodes: 2,
						VmClass:  recommender.Spot,
					},
				},
				recommender.Cpu: { // price = 2*2 +2*2 = 8
					recommender.NodePool{
						VmType: recommender.VirtualMachine{
							AvgPrice:      1,
							OnDemandPrice: 2,
						},
						SumNodes: 2,
						VmClass:  recommender.Regular,
					}, recommender.NodePool{
						VmType: recommender.VirtualMachine{
							AvgPrice:      2,
							OnDemandPrice: 4,
						},
						SumNodes: 2,
						VmClass:  recommender.Spot,
					}, recommender.NodePool{
						VmType: recommender.VirtualMachine{
							AvgPrice:      2,
							OnDemandPrice: 4,
						},
						SumNodes: 0,
						VmClass:  recommender.Spot,
					},
				},
			},
			check: func(nps []recommender.NodePool) {
				assert.Equal(t, 3, len(nps), "wrong selection")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine := NewEngine(logur.NewTestLogger(), nil)
			test.check(engine.findCheapestNodePoolSet(test.nodePools))
		})
	}
}
