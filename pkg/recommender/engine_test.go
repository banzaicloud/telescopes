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

package recommender

import (
	"testing"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo-client/models"
	"github.com/goph/logur"
	"github.com/stretchr/testify/assert"
)

type dummyProducts struct {
	// test case id to drive the behaviour
	TcId string
}

func (p *dummyProducts) GetProductDetails(provider string, service string, region string) ([]VirtualMachine, error) {
	return []VirtualMachine{
		{
			Cpus:          16,
			Mem:           42,
			OnDemandPrice: 3,
			AvgPrice:      0.8,
		},
	}, nil
}

func (p *dummyProducts) GetRegions(provider, service string) ([]*models.Continent, error) {
	return nil, nil
}

type dummyVms struct {
	// test case id to drive the behaviour
	TcId string
}

func (v *dummyVms) RecommendVms(provider string, vms []VirtualMachine, attr string, req SingleClusterRecommendationReq, layout []NodePool) ([]VirtualMachine, []VirtualMachine, error) {
	return nil, []VirtualMachine{
		{
			Cpus:          16,
			Mem:           42,
			AvgPrice:      2,
			OnDemandPrice: 3,
		},
		{
			Cpus:          16,
			Mem:           42,
			AvgPrice:      2,
			OnDemandPrice: 3,
		},
		{
			Cpus:          16,
			Mem:           42,
			AvgPrice:      2,
			OnDemandPrice: 4,
		},
		{
			Cpus:          16,
			Mem:           42,
			AvgPrice:      2,
			OnDemandPrice: 4,
		},
	}, nil
}

func (v *dummyVms) FindVmsWithAttrValues(attr string, req SingleClusterRecommendationReq, layoutDesc []NodePoolDesc, allProducts []VirtualMachine) ([]VirtualMachine, error) {
	return nil, nil
}

type dummyNodePools struct {
	// test case id to drive the behaviour
	TcId string
}

func (nps *dummyNodePools) RecommendNodePools(attr string, req SingleClusterRecommendationReq, layout []NodePool, odVms []VirtualMachine, spotVms []VirtualMachine) []NodePool {
	return []NodePool{
		{ // price = 2*3 +2*2 = 10
			VmType: VirtualMachine{
				Cpus:          16,
				Mem:           42,
				AvgPrice:      2,
				OnDemandPrice: 3,
			},
			SumNodes: 0,
			VmClass:  Regular,
		},
		{
			VmType: VirtualMachine{
				Cpus:          16,
				Mem:           42,
				AvgPrice:      2,
				OnDemandPrice: 3,
			},
			SumNodes: 0,
			VmClass:  Spot,
		},
		{
			VmType: VirtualMachine{
				Cpus:          8,
				Mem:           21,
				AvgPrice:      1,
				OnDemandPrice: 2,
			},
			SumNodes: 0,
			VmClass:  Regular,
		},
		{
			VmType: VirtualMachine{
				Cpus:          16,
				Mem:           42,
				AvgPrice:      2,
				OnDemandPrice: 4,
			},
			SumNodes: 1,
			VmClass:  Spot,
		},
		{
			VmType: VirtualMachine{
				Cpus:          16,
				Mem:           42,
				AvgPrice:      2,
				OnDemandPrice: 4,
			},
			SumNodes: 0,
			VmClass:  Spot,
		},
	}
}

func TestEngine_RecommendCluster(t *testing.T) {
	tests := []struct {
		name     string
		vms      VmRecommender
		np       NodePoolRecommender
		ciSource CloudInfoSource
		request  SingleClusterRecommendationReq
		check    func(resp *ClusterRecommendationResp, err error)
	}{
		{
			name: "cluster recommendation success",
			vms:  &dummyVms{},
			np:   &dummyNodePools{},
			request: SingleClusterRecommendationReq{
				ClusterRecommendationReq: ClusterRecommendationReq{
					MinNodes: 1,
					MaxNodes: 1,
					SumMem:   32,
					SumCpu:   16,
				},
			},
			ciSource: &dummyProducts{},
			check: func(resp *ClusterRecommendationResp, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, float64(42), resp.Accuracy.RecMem)
				assert.Equal(t, float64(16), resp.Accuracy.RecCpu)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine := NewEngine(logur.NewTestLogger(), test.ciSource, test.vms, test.np)

			test.check(engine.RecommendCluster("dummyProvider", "dummyService", "dummyRegion", test.request, nil))
		})
	}
}

func TestEngine_findCheapestNodePoolSet(t *testing.T) {
	tests := []struct {
		name      string
		vms       VmRecommender
		np        NodePoolRecommender
		nodePools map[string][]NodePool
		check     func(nps []NodePool)
	}{
		{
			name: "find cheapest node pool set",
			vms:  &dummyVms{},
			np:   &dummyNodePools{},
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
			check: func(nps []NodePool) {
				assert.Equal(t, 3, len(nps), "wrong selection")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine := NewEngine(logur.NewTestLogger(), nil, test.vms, test.np)
			test.check(engine.findCheapestNodePoolSet(test.nodePools))
		})
	}
}
