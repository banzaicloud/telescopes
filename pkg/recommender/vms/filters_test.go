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

package vms

import (
	"testing"

	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/goph/logur"
	"github.com/stretchr/testify/assert"
)

var (
	trueVal  = true
	falseVal = false
)

func TestVmSelector_filtersApply(t *testing.T) {
	tests := []struct {
		name     string
		vm       recommender.VirtualMachine
		req      recommender.ClusterRecommendationReq
		attr     string
		provider string
		check    func(filtersApply bool)
	}{
		{
			name: "filter applies for cpu/mem and burst allowed",
			// minRatio = SumCpu/SumMem = 0.5
			req: recommender.ClusterRecommendationReq{SumCpu: 4, SumMem: 8, AllowBurst: &trueVal},
			// ratio = Cpus/Mem = 1
			vm:       recommender.VirtualMachine{Cpus: 4, Mem: 4, Burst: true, CurrentGen: true},
			attr:     recommender.Memory,
			provider: "amazon",
			check: func(filtersApply bool) {
				assert.Equal(t, true, filtersApply, "vm should pass all filters")
			},
		},
		{
			name: "filter doesn't apply for cpu/mem and burst not allowed ",
			// minRatio = SumCpu/SumMem = 0.5
			req: recommender.ClusterRecommendationReq{SumCpu: 4, SumMem: 8, AllowBurst: &falseVal},
			// ratio = Cpus/Mem = 1
			vm:       recommender.VirtualMachine{Cpus: 4, Mem: 4, Burst: true, CurrentGen: true},
			attr:     recommender.Cpu,
			provider: "amazon",
			check: func(filtersApply bool) {
				assert.Equal(t, false, filtersApply, "vm should not pass all filters")
			},
		},
		{
			name: "filter applies for mem/cpu and burst allowed",
			// minRatio = SumMem/SumCpu = 2
			req: recommender.ClusterRecommendationReq{SumMem: 8, SumCpu: 4, AllowBurst: &trueVal},
			// ratio = Mem/Cpus = 1
			vm:       recommender.VirtualMachine{Mem: 20, Cpus: 4, Burst: true, CurrentGen: true},
			attr:     recommender.Cpu,
			provider: "amazon",
			check: func(filtersApply bool) {
				assert.Equal(t, true, filtersApply, "vm should pass all filters")
			},
		},
		{
			name: "filter doesn't apply for mem/cpu and burst not allowed ",
			// minRatio = SumMem/SumCpu = 2
			req: recommender.ClusterRecommendationReq{SumMem: 8, SumCpu: 4, AllowBurst: &falseVal},
			// ratio = Mem/Cpus = 1
			vm:       recommender.VirtualMachine{Mem: 20, Cpus: 4, Burst: true, CurrentGen: true},
			attr:     recommender.Memory,
			provider: "amazon",
			check: func(filtersApply bool) {
				assert.Equal(t, false, filtersApply, "vm should not pass all filters")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger())
			vmFilter, _ := selector.filtersForAttr(test.attr, test.provider, test.req)
			test.check(selector.filtersApply(test.vm, vmFilter, test.req))
		})
	}
}

func TestVmSelector_minCpuRatioFilter(t *testing.T) {
	tests := []struct {
		name  string
		vm    recommender.VirtualMachine
		attr  string
		req   recommender.ClusterRecommendationReq
		check func(filterApplies bool)
	}{
		{
			name: "minCpuRatioFilter applies",
			// minRatio = SumCpu/SumMem = 0.5
			req: recommender.ClusterRecommendationReq{SumCpu: 4, SumMem: 8},
			// ratio = Cpus/Mem = 1
			vm:   recommender.VirtualMachine{Cpus: 4, Mem: 4},
			attr: recommender.Cpu,
			check: func(filterApplies bool) {
				assert.Equal(t, true, filterApplies, "vm should pass the minCpuRatioFilter")
			},
		},
		{
			name: "minCpuRatioFilter doesn't apply",
			// minRatio = SumCpu/SumMem = 1
			req: recommender.ClusterRecommendationReq{SumCpu: 4, SumMem: float64(4)},
			// ratio = Cpus/Mem = 0.5
			vm:   recommender.VirtualMachine{Cpus: 4, Mem: float64(8)},
			attr: recommender.Cpu,
			check: func(filterApplies bool) {
				assert.Equal(t, false, filterApplies, "vm should not pass the minCpuRatioFilter")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger())
			test.check(selector.minCpuRatioFilter(test.vm, test.req))
		})
	}
}

func TestVmSelector_minMemRatioFilter(t *testing.T) {
	tests := []struct {
		name  string
		req   recommender.ClusterRecommendationReq
		vm    recommender.VirtualMachine
		attr  string
		check func(filterApplies bool)
	}{
		{
			name: "minMemRatioFilter applies",
			// minRatio = SumMem/SumCpu = 2
			req: recommender.ClusterRecommendationReq{SumMem: 8, SumCpu: 4},
			// ratio = Mem/Cpus = 4
			vm:   recommender.VirtualMachine{Mem: 16, Cpus: 4},
			attr: recommender.Cpu,
			check: func(filterApplies bool) {
				assert.Equal(t, true, filterApplies, "vm should pass the minMemRatioFilter")
			},
		},
		{
			name: "minMemRatioFilter doesn't apply",
			// minRatio = SumMem/SumCpu = 2
			req: recommender.ClusterRecommendationReq{SumMem: 8, SumCpu: 4},
			// ratio = Mem/Cpus = 0.5
			vm:   recommender.VirtualMachine{Cpus: 4, Mem: 4},
			attr: recommender.Cpu,
			check: func(filterApplies bool) {
				assert.Equal(t, false, filterApplies, "vm should not pass the minMemRatioFilter")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger())
			test.check(selector.minMemRatioFilter(test.vm, test.req))
		})
	}
}

func TestVmSelector_burstFilter(t *testing.T) {
	tests := []struct {
		name  string
		vm    recommender.VirtualMachine
		check func(filterApplies bool)
	}{
		{
			name: "burst filter applies - not burst vm, burst not allowed",
			// not a burst vm!
			vm: recommender.VirtualMachine{Burst: false},
			check: func(filterApplies bool) {
				assert.Equal(t, true, filterApplies, "vm should pass the burst filter")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger())
			test.check(selector.burstFilter(test.vm, recommender.ClusterRecommendationReq{}))
		})
	}
}

func TestVmSelector_excludesFilter(t *testing.T) {
	tests := []struct {
		name  string
		vm    recommender.VirtualMachine
		req   recommender.ClusterRecommendationReq
		check func(res bool)
	}{
		{
			name: "nil blacklist",
			vm: recommender.VirtualMachine{
				Type: "vm-type",
			},
			req: recommender.ClusterRecommendationReq{},
			check: func(res bool) {
				assert.True(t, res, "all vms should pass")
			},
		},
		{
			name: "empty blacklist",
			vm: recommender.VirtualMachine{
				Type: "vm-type",
			},
			req: recommender.ClusterRecommendationReq{
				Excludes: []string{},
			},
			check: func(res bool) {
				assert.True(t, res, "all vms should pass")
			},
		},
		{
			name: "vm blacklisted",
			vm: recommender.VirtualMachine{
				Type: "blacklisted-type",
			},
			req: recommender.ClusterRecommendationReq{
				Excludes: []string{"blacklisted-type", "other type"},
			},
			check: func(res bool) {
				assert.False(t, res, "the filter should fail")
			},
		},
		{
			name: "vm not blacklisted",
			vm: recommender.VirtualMachine{
				Type: "not-blacklisted-type",
			},
			req: recommender.ClusterRecommendationReq{
				Excludes: []string{"blacklisted-type", "other type"},
			},
			check: func(res bool) {
				assert.True(t, res, "the filter should fail")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger())
			test.check(selector.excludesFilter(test.vm, test.req))
		})
	}
}

func TestVmSelector_includesFilter(t *testing.T) {
	tests := []struct {
		name  string
		vm    recommender.VirtualMachine
		req   recommender.ClusterRecommendationReq
		check func(res bool)
	}{
		{
			name: "vm whitelisted",
			vm: recommender.VirtualMachine{
				Type: "whitelisted-type",
			},
			req: recommender.ClusterRecommendationReq{
				Includes: []string{"whitelisted-type", "other type"},
			},
			check: func(res bool) {
				assert.True(t, res, "the filter should pass")
			},
		},
		{
			name: "vm not whitelisted",
			vm: recommender.VirtualMachine{
				Type: "not-blacklisted-type",
			},
			req: recommender.ClusterRecommendationReq{
				Includes: []string{"blacklisted-type", "other type"},
			},
			check: func(res bool) {
				assert.False(t, res, "the filter should fail")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger())
			test.check(selector.includesFilter(test.vm, test.req))
		})
	}
}

func TestVmSelector_filterSpots(t *testing.T) {
	tests := []struct {
		name  string
		vms   []recommender.VirtualMachine
		check func(filtered []recommender.VirtualMachine)
	}{
		{
			name: "vm-s filtered out",
			vms: []recommender.VirtualMachine{
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
			check: func(filtered []recommender.VirtualMachine) {
				assert.Equal(t, 1, len(filtered), "vm is not filtered out")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger())
			test.check(selector.filterSpots(test.vms))
		})
	}
}

func TestVmSelector_ntwPerformanceFilter(t *testing.T) {

	var (
		ntwLow  = "low"
		ntwHigh = "high"
	)
	tests := []struct {
		name  string
		req   recommender.ClusterRecommendationReq
		vm    recommender.VirtualMachine
		check func(passed bool)
	}{
		{
			name: "vm passes the network performance filter",
			req: recommender.ClusterRecommendationReq{
				NetworkPerf: &ntwLow,
			},
			vm: recommender.VirtualMachine{
				NetworkPerfCat: ntwLow,
				Type:           "instance type",
			},
			check: func(passed bool) {
				assert.True(t, passed, "vm should pass the check")
			},
		},
		{
			name: "vm doesn't pass the network performance filter",
			req: recommender.ClusterRecommendationReq{
				NetworkPerf: &ntwLow,
			},
			vm: recommender.VirtualMachine{
				NetworkPerfCat: ntwHigh,
				Type:           "instance type",
			},
			check: func(passed bool) {
				assert.False(t, passed, "vm should not pass the check")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger())
			test.check(selector.ntwPerformanceFilter(test.vm, test.req))
		})
	}
}

func TestVmSelector_currentGenFilter(t *testing.T) {
	tests := []struct {
		name  string
		vm    recommender.VirtualMachine
		check func(passed bool)
	}{
		{
			name: "filter should apply when vm IS of current generation",
			vm: recommender.VirtualMachine{
				Type:       "instance type",
				CurrentGen: true,
			},
			check: func(passed bool) {
				assert.True(t, passed, "vm should pass the filter")
			},
		},
		{
			name: "filter should not apply when vm IS NOT of current generation",
			vm: recommender.VirtualMachine{
				Type:       "instance type",
				CurrentGen: false,
			},
			check: func(passed bool) {
				assert.False(t, passed, "vm should not pass the filter")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger())
			test.check(selector.currentGenFilter(test.vm, recommender.ClusterRecommendationReq{}))
		})
	}
}
