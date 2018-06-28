package recommender

import (
	"github.com/banzaicloud/telescopes/pkg/productinfo"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	trueVal  = true
	falseVal = false
)

func TestEngine_filtersApply(t *testing.T) {
	tests := []struct {
		name   string
		engine Engine
		vm     VirtualMachine
		req    ClusterRecommendationReq
		attr   string
		check  func(filtersApply bool)
	}{
		{
			name:   "filter applies for cpu/mem and burst allowed",
			engine: Engine{},
			// minRatio = SumCpu/SumMem = 0.5
			req: ClusterRecommendationReq{SumCpu: 4, SumMem: float64(8), AllowBurst: &trueVal},
			// ratio = Cpus/Mem = 1
			vm:   VirtualMachine{Cpus: 4, Mem: float64(4), Burst: true},
			attr: Memory,
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
			vm:   VirtualMachine{Cpus: 4, Mem: float64(4), Burst: true},
			attr: Cpu,
			check: func(filtersApply bool) {
				assert.Equal(t, false, filtersApply, "vm should not pass all filters")
			},
		},
		{
			name:   "filter applies for mem/cpu and burst allowed",
			engine: Engine{},
			// minRatio = AumMem/SumCpu = 2
			req: ClusterRecommendationReq{SumMem: float64(8), SumCpu: 4, AllowBurst: &trueVal},
			// ratio = Mem/Cpus = 1
			vm:   VirtualMachine{Mem: float64(20), Cpus: 4, Burst: true},
			attr: Cpu,
			check: func(filtersApply bool) {
				assert.Equal(t, true, filtersApply, "vm should pass all filters")
			},
		},
		{
			name:   "filter doesn't apply for mem/cpu and burst not allowed ",
			engine: Engine{},
			// minRatio = AumMem/SumCpu = 2
			req: ClusterRecommendationReq{SumMem: float64(8), SumCpu: 4, AllowBurst: &falseVal},
			// ratio = Mem/Cpus = 1
			vm:   VirtualMachine{Mem: float64(20), Cpus: 4, Burst: true},
			attr: Memory,
			check: func(filtersApply bool) {
				assert.Equal(t, false, filtersApply, "vm should not pass all filters")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filters, err := test.engine.filtersForAttr(test.attr)
			assert.Nil(t, err, "should get filters for attribute")
			test.check(test.engine.filtersApply(test.vm, filters, test.req))
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

			test.check(test.engine.minCpuRatioFilter(test.vm, test.req))

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

			test.check(test.engine.minMemRatioFilter(test.vm, test.req))

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
			test.check(test.engine.burstFilter(test.vm, test.req))
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
			test.check(test.engine.excludesFilter(test.vm, test.req))
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
			test.check(test.engine.includesFilter(test.vm, test.req))
		})
	}
}

func TestEngine_findValuesBetween(t *testing.T) {
	tests := []struct {
		name   string
		engine Engine
		check  func([]float64, error)
		min    float64
		max    float64
		values []float64
	}{
		{
			name:   "failure - no attribute values to filer",
			engine: Engine{},
			values: nil,
			min:    0,
			max:    100,
			check: func(res []float64, err error) {
				assert.Equal(t, "no attribute values provided", err.Error())
				assert.Nil(t, res, "the result should be nil")
			},
		},
		{
			name:   "failure - min > max",
			engine: Engine{},
			values: []float64{0},
			min:    100,
			max:    0,
			check: func(res []float64, err error) {
				assert.Equal(t, "min value cannot be larger than the max value", err.Error())
				assert.Nil(t, res, "the result should be nil")
			},
		},
		{
			name:   "success - max is lower than the smallest value, return smallest value",
			engine: Engine{},
			values: []float64{200, 100, 500, 66},
			min:    0,
			max:    50,
			check: func(res []float64, err error) {
				assert.Nil(t, err, "should not get error")
				assert.Equal(t, 1, len(res), "returned invalid number of results")
				assert.Equal(t, float64(66), res[0], "invalid value returned")
			},
		},
		{
			name:   "success - min is greater than the largest value, return the largest value",
			engine: Engine{},
			values: []float64{200, 100, 500, 66},
			min:    1000,
			max:    2000,
			check: func(res []float64, err error) {
				assert.Nil(t, err, "should not get error")
				assert.Equal(t, 1, len(res), "returned invalid number of results")
				assert.Equal(t, float64(500), res[0], "invalid value returned")
			},
		},
		{
			name:   "success - should return values between min-max",
			engine: Engine{},
			values: []float64{1, 2, 10, 5, 30, 99, 55},
			min:    5,
			max:    55,
			check: func(res []float64, err error) {
				assert.Nil(t, err, "should not get error")
				assert.Equal(t, 4, len(res), "returned invalid number of results")
				assert.Equal(t, []float64{5, 10, 30, 55}, res, "invalid value returned")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(test.engine.findValuesBetween(test.values, test.min, test.max))
		})
	}
}

func TestEngine_avgNodeCount(t *testing.T) {
	tests := []struct {
		name       string
		attrValues []float64
		reqSum     float64
		check      func(avg int)
	}{
		{
			name: "calculate average value per node",
			// sum of vals = 36
			// avg = 36/8 = 4.5
			// avg/node = ceil(10/4.5)=3
			attrValues: []float64{1, 2, 3, 4, 5, 6, 7, 8},
			reqSum:     10,
			check: func(avg int) {
				assert.Equal(t, 3, avg, "the calculated avg is invalid")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(avgNodeCount(test.attrValues, test.reqSum))
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
			test.check(test.engine.findCheapestNodePoolSet(test.nodePools))
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
			test.check(test.engine.filterSpots(test.vms))
		})
	}
}

func TestEngine_ntwPerformanceFilter(t *testing.T) {
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
				NetworkPerf: &productinfo.NTW_LOW,
			},
			vm: VirtualMachine{
				NetworkPerf: productinfo.NTW_LOW,
				Type:        "instance type",
			},
			check: func(passed bool) {
				assert.True(t, passed, "vm should pass the check")
			},
		},
		{
			name:   "vm doesn't pass the network performance filter",
			engine: Engine{},
			req: ClusterRecommendationReq{
				NetworkPerf: &productinfo.NTW_LOW,
			},
			vm: VirtualMachine{
				NetworkPerf: productinfo.NTW_HIGH,
				Type:        "instance type",
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
				NetworkPerf: productinfo.NTW_LOW,
				Type:        "instance type",
			},
			check: func(passed bool) {
				assert.True(t, passed, "vm should pass the check")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(test.engine.ntwPerformanceFilter(test.vm, test.req))
		})
	}
}
