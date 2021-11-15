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

func boolref(b bool) *bool {
	return &b
}

func TestVmSelector_filtersApply(t *testing.T) {
	tests := []struct {
		name     string
		vm       recommender.VirtualMachine
		req      recommender.SingleClusterRecommendationReq
		attr     string
		provider string
		check    func(filtersApply bool)
	}{
		{
			name: "filter applies for cpu/mem and burst allowed",
			// minRatio = SumCpu/SumMem = 0.5
			req: recommender.SingleClusterRecommendationReq{
				ClusterRecommendationReq: recommender.ClusterRecommendationReq{
					SumCpu:     4,
					SumMem:     8,
					AllowBurst: boolref(true),
				},
			},
			// ratio = Cpus/Mem = 1
			vm:       recommender.VirtualMachine{Cpus: 4, Mem: 4, Burst: true, CurrentGen: true, AllocatableMem: 4, AllocatableCpus: 4},
			attr:     recommender.Memory,
			provider: "amazon",
			check: func(filtersApply bool) {
				assert.Equal(t, true, filtersApply, "vm should pass all filters")
			},
		},
		{
			name: "filter doesn't apply for cpu/mem and burst not allowed ",
			// minRatio = SumCpu/SumMem = 0.5
			req: recommender.SingleClusterRecommendationReq{
				ClusterRecommendationReq: recommender.ClusterRecommendationReq{
					SumCpu:     4,
					SumMem:     8,
					AllowBurst: boolref(false),
				},
			},
			// ratio = Cpus/Mem = 1
			vm:       recommender.VirtualMachine{Cpus: 4, Mem: 4, Burst: true, CurrentGen: true, AllocatableCpus: 4, AllocatableMem: 4},
			attr:     recommender.Cpu,
			provider: "amazon",
			check: func(filtersApply bool) {
				assert.Equal(t, false, filtersApply, "vm should not pass all filters")
			},
		},
		{
			name: "filter applies for mem/cpu and burst allowed",
			// minRatio = SumMem/SumCpu = 2
			req: recommender.SingleClusterRecommendationReq{
				ClusterRecommendationReq: recommender.ClusterRecommendationReq{
					SumMem:     8,
					SumCpu:     4,
					AllowBurst: boolref(true),
				},
			},
			// ratio = Mem/Cpus = 1
			vm:       recommender.VirtualMachine{Mem: 20, Cpus: 4, Burst: true, CurrentGen: true, AllocatableMem: 20, AllocatableCpus: 4},
			attr:     recommender.Cpu,
			provider: "amazon",
			check: func(filtersApply bool) {
				assert.Equal(t, true, filtersApply, "vm should pass all filters")
			},
		},
		{
			name: "filter doesn't apply for mem/cpu and burst not allowed ",
			// minRatio = SumMem/SumCpu = 2
			req: recommender.SingleClusterRecommendationReq{
				ClusterRecommendationReq: recommender.ClusterRecommendationReq{
					SumMem:     8,
					SumCpu:     4,
					AllowBurst: boolref(false),
				},
			},
			// ratio = Mem/Cpus = 1
			vm:       recommender.VirtualMachine{Mem: 20, Cpus: 4, Burst: true, CurrentGen: true, AllocatableMem: 20, AllocatableCpus: 4},
			attr:     recommender.Memory,
			provider: "amazon",
			check: func(filtersApply bool) {
				assert.Equal(t, false, filtersApply, "vm should not pass all filters")
			},
		},
	}
	for _, test := range tests {
		test := test // scopelint
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
		req   recommender.SingleClusterRecommendationReq
		check func(filterApplies bool)
	}{
		{
			name: "minCpuRatioFilter applies",
			// minRatio = SumCpu/SumMem = 0.5
			req: recommender.SingleClusterRecommendationReq{
				ClusterRecommendationReq: recommender.ClusterRecommendationReq{
					SumCpu: 4,
					SumMem: 8,
				},
			},
			// ratio = Cpus/Mem = 1
			vm:   recommender.VirtualMachine{Cpus: 4, Mem: 4, AllocatableCpus: 4, AllocatableMem: 4},
			attr: recommender.Cpu,
			check: func(filterApplies bool) {
				assert.Equal(t, true, filterApplies, "vm should pass the minCpuRatioFilter")
			},
		},
		{
			name: "minCpuRatioFilter doesn't apply",
			// minRatio = SumCpu/SumMem = 1
			req: recommender.SingleClusterRecommendationReq{
				ClusterRecommendationReq: recommender.ClusterRecommendationReq{
					SumCpu: 4,
					SumMem: 4,
				},
			},
			// ratio = Cpus/Mem = 0.5
			vm:   recommender.VirtualMachine{Cpus: 4, Mem: float64(8)},
			attr: recommender.Cpu,
			check: func(filterApplies bool) {
				assert.Equal(t, false, filterApplies, "vm should not pass the minCpuRatioFilter")
			},
		},
	}
	for _, test := range tests {
		test := test // scopelint
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger())
			test.check(selector.minCpuRatioFilter(test.vm, test.req))
		})
	}
}

func TestVmSelector_minMemRatioFilter(t *testing.T) {
	tests := []struct {
		name  string
		req   recommender.SingleClusterRecommendationReq
		vm    recommender.VirtualMachine
		attr  string
		check func(filterApplies bool)
	}{
		{
			name: "minMemRatioFilter applies",
			// minRatio = SumMem/SumCpu = 2
			req: recommender.SingleClusterRecommendationReq{
				ClusterRecommendationReq: recommender.ClusterRecommendationReq{
					SumMem: 8,
					SumCpu: 4,
				},
			},
			// ratio = Mem/Cpus = 4
			vm:   recommender.VirtualMachine{Mem: 16, Cpus: 4, AllocatableMem: 16, AllocatableCpus: 4},
			attr: recommender.Cpu,
			check: func(filterApplies bool) {
				assert.Equal(t, true, filterApplies, "vm should pass the minMemRatioFilter")
			},
		},
		{
			name: "minMemRatioFilter doesn't apply",
			// minRatio = SumMem/SumCpu = 2
			req: recommender.SingleClusterRecommendationReq{
				ClusterRecommendationReq: recommender.ClusterRecommendationReq{
					SumMem: 8,
					SumCpu: 4,
				},
			},
			// ratio = Mem/Cpus = 0.5
			vm:   recommender.VirtualMachine{Cpus: 4, Mem: 4},
			attr: recommender.Cpu,
			check: func(filterApplies bool) {
				assert.Equal(t, false, filterApplies, "vm should not pass the minMemRatioFilter")
			},
		},
	}
	for _, test := range tests {
		test := test
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
		test := test // scopelint
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger())
			test.check(selector.burstFilter(test.vm, recommender.SingleClusterRecommendationReq{}))
		})
	}
}

func TestVmSelector_excludeTypesFilter(t *testing.T) {
	tests := []struct {
		name  string
		vm    recommender.VirtualMachine
		req   recommender.SingleClusterRecommendationReq
		check func(res bool)
	}{
		{
			name: "nil blacklist",
			vm: recommender.VirtualMachine{
				Type: "vm-type",
			},
			req: recommender.SingleClusterRecommendationReq{},
			check: func(res bool) {
				assert.True(t, res, "all vms should pass")
			},
		},
		{
			name: "empty blacklist",
			vm: recommender.VirtualMachine{
				Type: "vm-type",
			},
			req: recommender.SingleClusterRecommendationReq{
				ExcludeTypes: []string{},
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
			req: recommender.SingleClusterRecommendationReq{
				ExcludeTypes: []string{"blacklisted-type", "other type"},
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
			req: recommender.SingleClusterRecommendationReq{
				ExcludeTypes: []string{"blacklisted-type", "other type"},
			},
			check: func(res bool) {
				assert.True(t, res, "the filter should fail")
			},
		},
	}
	for _, test := range tests {
		test := test // scopelint
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger())
			test.check(selector.excludeTypeFilter(test.vm, test.req))
		})
	}
}

func TestVmSelector_includeTypesFilter(t *testing.T) {
	tests := []struct {
		name  string
		vm    recommender.VirtualMachine
		req   recommender.SingleClusterRecommendationReq
		check func(res bool)
	}{
		{
			name: "vm whitelisted",
			vm: recommender.VirtualMachine{
				Type: "whitelisted-type",
			},
			req: recommender.SingleClusterRecommendationReq{
				IncludeTypes: []string{"whitelisted-type", "other type"},
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
			req: recommender.SingleClusterRecommendationReq{
				IncludeTypes: []string{"blacklisted-type", "other type"},
			},
			check: func(res bool) {
				assert.False(t, res, "the filter should fail")
			},
		},
	}
	for _, test := range tests {
		test := test // scopelint
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger())
			test.check(selector.includeTypeFilter(test.vm, test.req))
		})
	}
}

func TestVmSelector_includeSeriesFilter(t *testing.T) {
	tests := []struct {
		name  string
		vm    recommender.VirtualMachine
		req   recommender.SingleClusterRecommendationReq
		check func(res bool)
	}{
		{
			name: "vm whitelisted",
			vm: recommender.VirtualMachine{
				Series: "whitelisted-series",
			},
			req: recommender.SingleClusterRecommendationReq{
				IncludeSeries: []string{"whitelisted-series", "other type"},
			},
			check: func(res bool) {
				assert.True(t, res, "the filter should pass")
			},
		},
		{
			name: "vm not whitelisted",
			vm: recommender.VirtualMachine{
				Series: "other-type",
			},
			req: recommender.SingleClusterRecommendationReq{
				IncludeSeries: []string{"whitelisted-series"},
			},
			check: func(res bool) {
				assert.False(t, res, "the filter should fail")
			},
		},
	}
	for _, test := range tests {
		test := test // scopelint
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger())
			test.check(selector.includeSeriesFilter(test.vm, test.req))
		})
	}
}

func TestVmSelector_excludeSeriesFilter(t *testing.T) {
	tests := []struct {
		name  string
		vm    recommender.VirtualMachine
		req   recommender.SingleClusterRecommendationReq
		check func(res bool)
	}{
		{
			name: "nil blacklist",
			vm: recommender.VirtualMachine{
				Series: "vm-series",
			},
			req: recommender.SingleClusterRecommendationReq{},
			check: func(res bool) {
				assert.True(t, res, "all vms should pass")
			},
		},
		{
			name: "empty blacklist",
			vm: recommender.VirtualMachine{
				Series: "vm-series",
			},
			req: recommender.SingleClusterRecommendationReq{
				ExcludeSeries: []string{},
			},
			check: func(res bool) {
				assert.True(t, res, "all vms should pass")
			},
		},
		{
			name: "vm blacklisted",
			vm: recommender.VirtualMachine{
				Series: "blacklisted-series",
			},
			req: recommender.SingleClusterRecommendationReq{
				ExcludeSeries: []string{"blacklisted-series", "other-series"},
			},
			check: func(res bool) {
				assert.False(t, res, "the filter should fail")
			},
		},
		{
			name: "vm not blacklisted",
			vm: recommender.VirtualMachine{
				Series: "not-blacklisted-series",
			},
			req: recommender.SingleClusterRecommendationReq{
				ExcludeSeries: []string{"blacklisted-series", "other-series"},
			},
			check: func(res bool) {
				assert.True(t, res, "the filter should fail")
			},
		},
	}
	for _, test := range tests {
		test := test // scopelint
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger())
			test.check(selector.excludeSeriesFilter(test.vm, test.req))
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
		test := test // scopelint
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
		req   recommender.SingleClusterRecommendationReq
		vm    recommender.VirtualMachine
		check func(passed bool)
	}{
		{
			name: "vm passes the network performance filter",
			req: recommender.SingleClusterRecommendationReq{
				ClusterRecommendationReq: recommender.ClusterRecommendationReq{
					NetworkPerf: []string{ntwLow},
				},
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
			req: recommender.SingleClusterRecommendationReq{
				ClusterRecommendationReq: recommender.ClusterRecommendationReq{
					NetworkPerf: []string{ntwLow},
				},
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
		test := test // scopelint
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
		test := test // scopelint
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger())
			test.check(selector.currentGenFilter(test.vm, recommender.SingleClusterRecommendationReq{}))
		})
	}
}
