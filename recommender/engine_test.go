package recommender

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type dummyVmRegistry struct {
	// implement the interface
	VmRegistry
}

func TestNewEngine(t *testing.T) {

	tests := []struct {
		name         string
		vmRegistries map[string]VmRegistry
		checker      func(engine *Engine, err error)
	}{
		{
			name:         "engine successfully created",
			vmRegistries: map[string]VmRegistry{"ec2": dummyVmRegistry{}},
			checker: func(engine *Engine, err error) {
				assert.Nil(t, err, "should not get error ")
				assert.NotNil(t, engine, "the engine should not be nil")
			},
		},
		{
			name:         "engine creation fails when registries is nil",
			vmRegistries: nil,
			checker: func(engine *Engine, err error) {
				assert.Nil(t, engine, "the engine should be nil")
				assert.NotNil(t, err, "the error shouldn't be nil")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.checker(NewEngine(test.vmRegistries))

		})
	}
}

func TestEngine_RecommendCluster(t *testing.T) {
	tests := []struct {
		name         string
		vmRegistries map[string]VmRegistry
		request      ClusterRecommendationReq
		provider     string
		region       string
		check        func(response *ClusterRecommendationResp, err error)
	}{
		{
			name:         "cluster recommendation success",
			vmRegistries: map[string]VmRegistry{"ec2": dummyVmRegistry{}},
			request:      ClusterRecommendationReq{},
			provider:     "ec2",
			region:       "us",

			check: func(response *ClusterRecommendationResp, err error) {
				assert.Nil(t, err, "should not get error when recommending")

			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine, err := NewEngine(test.vmRegistries)
			assert.Nil(t, err, "the engine couldn't be created")

			test.check(engine.RecommendCluster(test.provider, test.region, test.request))

		})
	}
}

// utility VmRegistry for mocking purposes
type DummyVmRegistry struct {
	// test case id to drive the behaviour
	TcId int
}

func (dvmr DummyVmRegistry) getAvailableAttributeValues(attr string) ([]float64, error) {
	switch dvmr.TcId {
	case 1:
		// 3 values between 10 - 20
		return []float64{12, 13, 14}, nil
	case 2:
		// 2 values between 10 - 20
		return []float64{8, 13, 14, 6}, nil
	case 3:
		// no values between 10-20, return the closest value
		return []float64{30, 40, 50, 60}, nil
	case 4:
		// no values between 10-20, return the closest value
		return []float64{1, 2, 3, 5, 9}, nil
	case 5:
		// error, min > max
		return []float64{1}, nil
	case 6:
		// error returned
		return nil, errors.New("")

	}

	return nil, nil
}

func (dvmr DummyVmRegistry) findVmsWithAttrValues(region string, zones []string, attr string, values []float64) ([]VirtualMachine, error) {
	return nil, nil
}

func TestEngine_RecommendAttrValues(t *testing.T) {

	tests := []struct {
		name         string
		vmRegistries map[string]VmRegistry
		request      ClusterRecommendationReq
		provider     string
		attribute    string
		check        func([]float64, error)
	}{
		{
			name:         "all attributes between limits",
			vmRegistries: map[string]VmRegistry{"ec2": DummyVmRegistry{TcId: 1}},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			provider:  "ec2",
			attribute: Cpu,

			check: func(values []float64, err error) {
				assert.Nil(t, err, "should not get error when recommending attributes")
				assert.Equal(t, 3, len(values), "recommended number of values is not as expected")

			},
		},
		{
			name:         "attributes out of limits not recommended",
			vmRegistries: map[string]VmRegistry{"ec2": DummyVmRegistry{TcId: 2}},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			provider:  "ec2",
			attribute: Cpu,

			check: func(values []float64, err error) {
				assert.Nil(t, err, "should not get error when recommending attributes")
				assert.Equal(t, 2, len(values), "recommended number of values is not as expected")

			},
		},
		{
			name:         "no values between limits found - smallest value returned",
			vmRegistries: map[string]VmRegistry{"ec2": DummyVmRegistry{TcId: 3}},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			provider:  "ec2",
			attribute: Cpu,

			check: func(values []float64, err error) {
				assert.Nil(t, err, "should not get error when recommending attributes")
				assert.Equal(t, 1, len(values), "recommended number of values is not as expected")
				assert.Equal(t, float64(30), values[0], "recommended number of values is not as expected")

			},
		},
		{
			name:         "no values between limits found - largest value returned",
			vmRegistries: map[string]VmRegistry{"ec2": DummyVmRegistry{TcId: 4}},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			provider:  "ec2",
			attribute: Cpu,

			check: func(values []float64, err error) {
				assert.Nil(t, err, "should not get error when recommending attributes")
				assert.Equal(t, 1, len(values), "recommended number of values is not as expected")
				assert.Equal(t, float64(9), values[0], "recommended number of values is not as expected")

			},
		},
		{
			name:         "error - min larger than max",
			vmRegistries: map[string]VmRegistry{"ec2": DummyVmRegistry{TcId: 5}},
			request: ClusterRecommendationReq{
				MinNodes: 10,
				MaxNodes: 5,
				SumMem:   100,
				SumCpu:   100,
			},
			provider:  "ec2",
			attribute: Cpu,

			check: func(values []float64, err error) {
				assert.Equal(t, err.Error(), "min value cannot be larger than the max value")

			},
		},
		{
			name:         "error - no values provided",
			vmRegistries: map[string]VmRegistry{"ec2": DummyVmRegistry{TcId: 100}},
			request: ClusterRecommendationReq{
				MinNodes: 10,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			provider:  "ec2",
			attribute: Cpu,

			check: func(values []float64, err error) {
				assert.Equal(t, err.Error(), "no attribute values provided")

			},
		},
		{
			name:         "error - attribute values could not be retrieved",
			vmRegistries: map[string]VmRegistry{"ec2": DummyVmRegistry{TcId: 6}},
			request: ClusterRecommendationReq{
				MinNodes: 10,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			provider:  "ec2",
			attribute: Cpu,

			check: func(values []float64, err error) {
				assert.Nil(t, values, "returned attr values should be nils")
				assert.NotNil(t, err.Error(), "no attribute values provided")

			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine, err := NewEngine(test.vmRegistries)
			assert.Nil(t, err, "the engine couldn't be created")

			test.check(engine.RecommendAttrValues(engine.VmRegistries["ec2"].(VmRegistry), test.attribute, test.request))

		})
	}
}

func TestEngine_RecommendVms(t *testing.T) {
	tests := []struct {
		name         string
		region       string
		vmRegistries map[string]VmRegistry
		values       []float64
		filters      []vmFilter
		request      ClusterRecommendationReq
		provider     string
		attribute    string
		check        func([]VirtualMachine, error)
	}{
		{
			name: "success - recommend vms",

			vmRegistries: map[string]VmRegistry{"ec2": DummyVmRegistry{TcId: 1}},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			provider:  "ec2",
			attribute: Cpu,

			check: func(vms []VirtualMachine, err error) {
				assert.Nil(t, err, "should not get error when recommending attributes")
				assert.Equal(t, 3, len(vms), "recommended number of values is not as expected")

			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine, err := NewEngine(test.vmRegistries)
			assert.Nil(t, err, "the engine couldn't be created")

			test.check(engine.RecommendVms(engine.VmRegistries["ec2"].(VmRegistry), test.region, test.attribute, test.values, test.filters, test.request))

		})
	}
}
