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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	trueVal  = true
	falseVal = false
)

func TestEngine_filtersApply(t *testing.T) {
	tests := []struct {
		name     string
		engine   Engine
		vm       VirtualMachine
		req      ClusterRecommendationReq
		attr     string
		provider string
		check    func(filtersApply bool)
	}{
		{
			name:   "filter applies for cpu/mem and burst allowed",
			engine: Engine{},
			// minRatio = SumCpu/SumMem = 0.5
			req: ClusterRecommendationReq{SumCpu: 4, SumMem: float64(8), AllowBurst: &trueVal},
			// ratio = Cpus/Mem = 1
			vm:       VirtualMachine{Cpus: 4, Mem: float64(4), Burst: true, CurrentGen: true},
			attr:     Memory,
			provider: "ec2",
			check: func(filtersApply bool) {
				assert.Equal(t, true, filtersApply, "vm should pass all filters")
			},
		},
		{
			name:   "filter doesn't apply for cpu/mem and burst not allowed ",
			engine: Engine{},
			// minRatio = SumCpu/SumMem = 0.5
			req: ClusterRecommendationReq{SumCpu: 4, SumMem: float64(8), AllowBurst: &falseVal},
			// ratio = Cpus/Mem = 1
			vm:       VirtualMachine{Cpus: 4, Mem: float64(4), Burst: true, CurrentGen: true},
			attr:     Cpu,
			provider: "ec2",
			check: func(filtersApply bool) {
				assert.Equal(t, false, filtersApply, "vm should not pass all filters")
			},
		},
		{
			name:   "filter applies for mem/cpu and burst allowed",
			engine: Engine{},
			// minRatio = SumMem/SumCpu = 2
			req: ClusterRecommendationReq{SumMem: float64(8), SumCpu: 4, AllowBurst: &trueVal},
			// ratio = Mem/Cpus = 1
			vm:       VirtualMachine{Mem: float64(20), Cpus: 4, Burst: true, CurrentGen: true},
			attr:     Cpu,
			provider: "ec2",
			check: func(filtersApply bool) {
				assert.Equal(t, true, filtersApply, "vm should pass all filters")
			},
		},
		{
			name:   "filter doesn't apply for mem/cpu and burst not allowed ",
			engine: Engine{},
			// minRatio = SumMem/SumCpu = 2
			req: ClusterRecommendationReq{SumMem: float64(8), SumCpu: 4, AllowBurst: &falseVal},
			// ratio = Mem/Cpus = 1
			vm:       VirtualMachine{Mem: float64(20), Cpus: 4, Burst: true, CurrentGen: true},
			attr:     Memory,
			provider: "ec2",
			check: func(filtersApply bool) {
				assert.Equal(t, false, filtersApply, "vm should not pass all filters")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filters, err := test.engine.filtersForAttr(context.TODO(), test.attr, test.provider)
			assert.Nil(t, err, "should get filters for attribute")
			test.check(test.engine.filtersApply(context.TODO(), test.vm, filters, test.req))
		})
	}
}

func TestEngine_minCpuRatioFilter(t *testing.T) {
	tests := []struct {
		name   string
		engine Engine
		vm     VirtualMachine
		attr   string
		req    ClusterRecommendationReq
		check  func(filterApplies bool)
	}{
		{
			name:   "minCpuRatioFilter applies",
			engine: Engine{},
			// minRatio = SumCpu/SumMem = 0.5
			req: ClusterRecommendationReq{SumCpu: 4, SumMem: float64(8)},
			// ratio = Cpus/Mem = 1
			vm:   VirtualMachine{Cpus: 4, Mem: float64(4)},
			attr: Cpu,
			check: func(filterApplies bool) {
				assert.Equal(t, true, filterApplies, "vm should pass the minCpuRatioFilter")
			},
		},
		{
			name:   "minCpuRatioFilter doesn't apply",
			engine: Engine{},
			// minRatio = SumCpu/SumMem = 1
			req: ClusterRecommendationReq{SumCpu: 4, SumMem: float64(4)},
			// ratio = Cpus/Mem = 0.5
			vm:   VirtualMachine{Cpus: 4, Mem: float64(8)},
			attr: Cpu,
			check: func(filterApplies bool) {
				assert.Equal(t, false, filterApplies, "vm should not pass the minCpuRatioFilter")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			test.check(test.engine.minCpuRatioFilter(context.TODO(), test.vm, test.req))

		})
	}
}

func TestEngine_minMemRatioFilter(t *testing.T) {
	tests := []struct {
		name   string
		engine Engine
		req    ClusterRecommendationReq
		vm     VirtualMachine
		attr   string
		check  func(filterApplies bool)
	}{
		{
			name:   "minMemRatioFilter applies",
			engine: Engine{},
			// minRatio = SumMem/SumCpu = 2
			req: ClusterRecommendationReq{SumMem: float64(8), SumCpu: 4},
			// ratio = Mem/Cpus = 4
			vm:   VirtualMachine{Mem: float64(16), Cpus: 4},
			attr: Cpu,
			check: func(filterApplies bool) {
				assert.Equal(t, true, filterApplies, "vm should pass the minMemRatioFilter")
			},
		},
		{
			name:   "minMemRatioFilter doesn't apply",
			engine: Engine{},
			// minRatio = SumMem/SumCpu = 2
			req: ClusterRecommendationReq{SumMem: float64(8), SumCpu: 4},
			// ratio = Mem/Cpus = 0.5
			vm:   VirtualMachine{Cpus: 4, Mem: float64(4)},
			attr: Cpu,
			check: func(filterApplies bool) {
				assert.Equal(t, false, filterApplies, "vm should not pass the minMemRatioFilter")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			test.check(test.engine.minMemRatioFilter(context.TODO(), test.vm, test.req))

		})
	}
}

func TestEngine_burstFilter(t *testing.T) {
	tests := []struct {
		name   string
		engine Engine
		req    ClusterRecommendationReq
		vm     VirtualMachine
		check  func(filterApplies bool)
	}{
		{
			name:   "burst filter applies - burst vm, burst allowed in req",
			engine: Engine{},
			req:    ClusterRecommendationReq{AllowBurst: &trueVal},
			vm:     VirtualMachine{Burst: true},
			check: func(filterApplies bool) {
				assert.Equal(t, true, filterApplies, "vm should pass the burst filter")
			},
		},
		{
			name:   "burst filter applies - burst vm, burst not set in req",
			engine: Engine{},
			// BurstAllowed not specified
			req: ClusterRecommendationReq{},
			vm:  VirtualMachine{Burst: true},
			check: func(filterApplies bool) {
				assert.Equal(t, true, filterApplies, "vm should pass the burst filter")
			},
		},
		{
			name:   "burst filter doesn't apply - burst vm, burst not allowed",
			engine: Engine{},
			req:    ClusterRecommendationReq{AllowBurst: &falseVal},
			vm:     VirtualMachine{Burst: true},
			check: func(filterApplies bool) {
				assert.Equal(t, false, filterApplies, "vm should not pass the burst filter")
			},
		},
		{
			name:   "burst filter applies - not burst vm, burst not allowed",
			engine: Engine{},
			req:    ClusterRecommendationReq{AllowBurst: &falseVal},
			// not a burst vm!
			vm: VirtualMachine{Burst: false},
			check: func(filterApplies bool) {
				assert.Equal(t, true, filterApplies, "vm should pass the burst filter")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(test.engine.burstFilter(context.TODO(), test.vm, test.req))
		})
	}
}

func TestEngine_ExcludesFilter(t *testing.T) {
	tests := []struct {
		name   string
		engine Engine
		vm     VirtualMachine
		req    ClusterRecommendationReq
		check  func(res bool)
	}{
		{
			name:   "nil blacklist",
			engine: Engine{},
			vm: VirtualMachine{
				Type: "vm-type",
			},
			req: ClusterRecommendationReq{},
			check: func(res bool) {
				assert.True(t, res, "all vms should pass")
			},
		},
		{
			name:   "empty blacklist",
			engine: Engine{},
			vm: VirtualMachine{
				Type: "vm-type",
			},
			req: ClusterRecommendationReq{
				Excludes: []string{},
			},
			check: func(res bool) {
				assert.True(t, res, "all vms should pass")
			},
		},
		{
			name:   "vm blacklisted",
			engine: Engine{},
			vm: VirtualMachine{
				Type: "blacklisted-type",
			},
			req: ClusterRecommendationReq{
				Excludes: []string{"blacklisted-type", "other type"},
			},
			check: func(res bool) {
				assert.False(t, res, "the filter should fail")
			},
		},
		{
			name:   "vm not blacklisted",
			engine: Engine{},
			vm: VirtualMachine{
				Type: "not-blacklisted-type",
			},
			req: ClusterRecommendationReq{
				Excludes: []string{"blacklisted-type", "other type"},
			},
			check: func(res bool) {
				assert.True(t, res, "the filter should fail")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(test.engine.excludesFilter(context.TODO(), test.vm, test.req))
		})
	}
}

func TestEngine_IncludesFilter(t *testing.T) {
	tests := []struct {
		name   string
		engine Engine
		vm     VirtualMachine
		req    ClusterRecommendationReq
		check  func(res bool)
	}{
		{
			name:   "nil whitelist",
			engine: Engine{},
			vm: VirtualMachine{
				Type: "vm-type",
			},
			req: ClusterRecommendationReq{},
			check: func(res bool) {
				assert.True(t, res, "all vms should pass")
			},
		},
		{
			name:   "empty whitelist",
			engine: Engine{},
			vm: VirtualMachine{
				Type: "vm-type",
			},
			req: ClusterRecommendationReq{
				Includes: []string{},
			},
			check: func(res bool) {
				assert.True(t, res, "all vms should pass")
			},
		},
		{
			name:   "vm whitelisted",
			engine: Engine{},
			vm: VirtualMachine{
				Type: "whitelisted-type",
			},
			req: ClusterRecommendationReq{
				Includes: []string{"whitelisted-type", "other type"},
			},
			check: func(res bool) {
				assert.True(t, res, "the filter should pass")
			},
		},
		{
			name:   "vm not whitelisted",
			engine: Engine{},
			vm: VirtualMachine{
				Type: "not-blacklisted-type",
			},
			req: ClusterRecommendationReq{
				Includes: []string{"blacklisted-type", "other type"},
			},
			check: func(res bool) {
				assert.False(t, res, "the filter should fail")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(test.engine.includesFilter(context.TODO(), test.vm, test.req))
		})
	}
}

func TestEngine_findCheapestNodePoolSet(t *testing.T) {
	tests := []struct {
		name      string
		engine    Engine
		nodePools map[string][]NodePool
		check     func(npo []NodePool)
	}{
		{
			name:   "find cheapest node pool set",
			engine: Engine{},
			nodePools: map[string][]NodePool{
				"memory": {
					NodePool{ // price = 2*3 +2*2 = 10
						VmType: VirtualMachine{
							AvgPrice:      2,
							OnDemandPrice: 3,
						},
						SumNodes: 2,
						VmClass:  regular,
					}, NodePool{
						VmType: VirtualMachine{
							AvgPrice:      2,
							OnDemandPrice: 3,
						},
						SumNodes: 2,
						VmClass:  spot,
					},
				},
				"cpu": { // price = 2*2 +2*2 = 8
					NodePool{
						VmType: VirtualMachine{
							AvgPrice:      1,
							OnDemandPrice: 2,
						},
						SumNodes: 2,
						VmClass:  regular,
					}, NodePool{
						VmType: VirtualMachine{
							AvgPrice:      2,
							OnDemandPrice: 4,
						},
						SumNodes: 2,
						VmClass:  spot,
					}, NodePool{
						VmType: VirtualMachine{
							AvgPrice:      2,
							OnDemandPrice: 4,
						},
						SumNodes: 0,
						VmClass:  spot,
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
			test.check(test.engine.findCheapestNodePoolSet(context.TODO(), test.nodePools))
		})
	}
}

func TestEngine_filterSpots(t *testing.T) {
	tests := []struct {
		name   string
		engine Engine
		vms    []VirtualMachine
		check  func(filtered []VirtualMachine)
	}{
		{
			name:   "vm-s filtered out",
			engine: Engine{},
			vms: []VirtualMachine{
				{
					AvgPrice:      0,
					OnDemandPrice: 1,
					Type:          "t100",
				},
				{
					AvgPrice:      2,
					OnDemandPrice: 5,
					Type:          "t200",
				},
			},
			check: func(filtered []VirtualMachine) {
				assert.Equal(t, 1, len(filtered), "vm is not filtered out")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(test.engine.filterSpots(context.TODO(), test.vms))
		})
	}
}

func TestEngine_ntwPerformanceFilter(t *testing.T) {

	var (
		NTW_LOW  = "low"
		NTW_HIGH = "high"
	)
	tests := []struct {
		name   string
		engine Engine
		req    ClusterRecommendationReq
		vm     VirtualMachine
		check  func(passed bool)
	}{
		{
			name:   "vm passes the network performance filter",
			engine: Engine{},
			req: ClusterRecommendationReq{
				NetworkPerf: &NTW_LOW,
			},
			vm: VirtualMachine{
				NetworkPerfCat: NTW_LOW,
				Type:           "instance type",
			},
			check: func(passed bool) {
				assert.True(t, passed, "vm should pass the check")
			},
		},
		{
			name:   "vm doesn't pass the network performance filter",
			engine: Engine{},
			req: ClusterRecommendationReq{
				NetworkPerf: &NTW_LOW,
			},
			vm: VirtualMachine{
				NetworkPerfCat: NTW_HIGH,
				Type:           "instance type",
			},
			check: func(passed bool) {
				assert.False(t, passed, "vm should not pass the check")
			},
		},
		{
			name:   "vm passes the network performance filter - no filter in req",
			engine: Engine{},
			req:    ClusterRecommendationReq{ // filter is missing
			},
			vm: VirtualMachine{
				NetworkPerf: NTW_LOW,
				Type:        "instance type",
			},
			check: func(passed bool) {
				assert.True(t, passed, "vm should pass the check")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(test.engine.ntwPerformanceFilter(context.TODO(), test.vm, test.req))
		})
	}
}

func TestEngine_CurrGenFilter(t *testing.T) {
	tests := []struct {
		name   string
		engine Engine
		req    ClusterRecommendationReq
		vm     VirtualMachine
		check  func(passed bool)
	}{
		{
			name:   "filter should apply when AllowOlderGen IS NIL in the request and vm IS of current generation",
			engine: Engine{},
			req:    ClusterRecommendationReq{ // AllowOlderGen is nil;
			},
			vm: VirtualMachine{
				Type:       "instance type",
				CurrentGen: true,
			},
			check: func(passed bool) {
				assert.True(t, passed, "vm should pass the check")
			},
		},
		{
			name:   "filter should not apply when AllowOlderGen IS NIL in the request and vm IS NOT of current generation",
			engine: Engine{},
			req:    ClusterRecommendationReq{ // AllowOlderGen is nil;
			},
			vm: VirtualMachine{
				Type:       "instance type",
				CurrentGen: false,
			},
			check: func(passed bool) {
				assert.False(t, passed, "vm should fail the check")
			},
		},
		{
			name:   "filter should apply when AllowOlderGen is FALSE in the request and vm IS of current generation",
			engine: Engine{},
			req: ClusterRecommendationReq{
				AllowOlderGen: boolPointer(false),
			},
			vm: VirtualMachine{
				Type:       "instance type",
				CurrentGen: true,
			},
			check: func(passed bool) {
				assert.True(t, passed, "vm should pass the filter")
			},
		},
		{
			name:   "filter should not apply when AllowOlderGen is FALSE  in the request and vm IS NOT of current generation",
			engine: Engine{},
			req: ClusterRecommendationReq{
				AllowOlderGen: boolPointer(false),
			},
			vm: VirtualMachine{
				Type:       "instance type",
				CurrentGen: false,
			},
			check: func(passed bool) {
				assert.False(t, passed, "vm should not pass the filter")
			},
		},
		{
			name:   "filter should apply when AllowOlderGen is TRUE  in the request and vm IS of current generation",
			engine: Engine{},
			req: ClusterRecommendationReq{
				AllowOlderGen: boolPointer(true),
			},
			vm: VirtualMachine{
				Type:       "instance type",
				CurrentGen: true,
			},
			check: func(passed bool) {
				assert.True(t, passed, "vm should pass the filter")
			},
		},
		{
			name:   "filter should apply when AllowOlderGen is TRUE  in the request and vm IS NOT of current generation",
			engine: Engine{},
			req: ClusterRecommendationReq{
				AllowOlderGen: boolPointer(true),
			},
			vm: VirtualMachine{
				Type:       "instance type",
				CurrentGen: false,
			},
			check: func(passed bool) {
				assert.True(t, passed, "vm should pass the filter")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(test.engine.currentGenFilter(context.TODO(), test.vm, test.req))
		})
	}
}
