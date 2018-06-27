package recommender

import (
	"errors"
	"testing"

	"fmt"
	"github.com/banzaicloud/telescopes/pkg/productinfo"
	"github.com/banzaicloud/telescopes/pkg/productinfo-client/models"
	"github.com/stretchr/testify/assert"
)

const (
	OutOfLimits         = "2 values between 10 - 20"
	LargerValues        = "all values are larger than 20, return the closest value"
	SmallerValues       = "all values are less than 10, return the closest value"
	MinLargerThanMax    = "error, min > max"
	Error               = "error returned"
	ZoneError           = "zone error"
	DescribeRegionError = "could not describe region"
	ProductDetailsError = "could not get product details"
)

// dummy network mapper
type dummyNetworkMapper struct {
}

func (dm dummyNetworkMapper) MapNetworkPerf(vm productinfo.VmInfo) (string, error) {
	return productinfo.NTW_HIGH, nil
}

type dummyProductInfoSource struct {
	// test case id to drive the behaviour
	TcId string
}

func (piCli *dummyProductInfoSource) GetAttributeValues(provider string, region string, attr string) ([]float64, error) {
	switch piCli.TcId {
	case OutOfLimits:
		return []float64{8, 13, 14, 6}, nil
	case LargerValues:
		return []float64{30, 40, 50, 60}, nil
	case SmallerValues:
		return []float64{1, 2, 3, 5, 9}, nil
	case MinLargerThanMax:
		return []float64{1}, nil
	case Error:
		return nil, fmt.Errorf("error")
	}
	return []float64{15, 16, 17}, nil
}

func (piCli *dummyProductInfoSource) GetRegion(provider string, region string) ([]string, error) {
	switch piCli.TcId {
	case ZoneError:
		return nil, errors.New("no zone available")
	case DescribeRegionError:
		return nil, errors.New(DescribeRegionError)
	default:
		return []string{"dummyZone1", "dummyZone2", "dummyZone3"}, nil
	}
}

func (piCli *dummyProductInfoSource) GetProductDetails(provider string, region string) ([]*models.ProductDetails, error) {
	switch piCli.TcId {
	case ProductDetailsError:
		return nil, errors.New(ProductDetailsError)
	default:
		return []*models.ProductDetails{
			&models.ProductDetails{
				Type:          "type-1",
				OnDemandPrice: 12.5,
				Cpus:          2,
			},
			&models.ProductDetails{
				Type:          "type-2",
				OnDemandPrice: 13.2,
				Cpus:          2,
			},
			&models.ProductDetails{
				Type:          "type-3",
				OnDemandPrice: 13.2,
				Cpus:          2,
			},
		}, nil
	}
}

func (piCli *dummyProductInfoSource) GetNetworkPerfMapper(provider string) (productinfo.NetworkPerfMapper, error) {
	nm := dummyNetworkMapper{}
	return nm, nil
}

func TestEngine_RecommendAttrValues(t *testing.T) {

	tests := []struct {
		name      string
		pi        ProductInfoSource
		request   ClusterRecommendationReq
		provider  string
		attribute string
		check     func([]float64, error)
	}{
		{
			name: "all attributes between limits",
			pi:   &dummyProductInfoSource{},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			provider:  "dummy",
			attribute: Cpu,

			check: func(values []float64, err error) {
				assert.Nil(t, err, "should not get error when recommending attributes")
				assert.Equal(t, 3, len(values), "recommended number of values is not as expected")
				assert.Equal(t, []float64{15, 16, 17}, values)

			},
		},
		{
			name: "attributes out of limits not recommended",
			pi:   &dummyProductInfoSource{OutOfLimits},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			provider:  "dummy",
			attribute: Cpu,

			check: func(values []float64, err error) {
				assert.Nil(t, err, "should not get error when recommending attributes")
				assert.Equal(t, 2, len(values), "recommended number of values is not as expected")

			},
		},
		{
			name: "no values between limits found - smallest value returned",
			pi:   &dummyProductInfoSource{LargerValues},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			provider:  "dummy",
			attribute: Cpu,

			check: func(values []float64, err error) {
				assert.Nil(t, err, "should not get error when recommending attributes")
				assert.Equal(t, 1, len(values), "recommended number of values is not as expected")
				assert.Equal(t, float64(30), values[0], "recommended number of values is not as expected")

			},
		},
		{
			name: "no values between limits found - largest value returned",
			pi:   &dummyProductInfoSource{SmallerValues},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			provider:  "dummy",
			attribute: Cpu,

			check: func(values []float64, err error) {
				assert.Nil(t, err, "should not get error when recommending attributes")
				assert.Equal(t, 1, len(values), "recommended number of values is not as expected")
				assert.Equal(t, float64(9), values[0], "recommended number of values is not as expected")

			},
		},
		{
			name: "error - min larger than max",
			pi:   &dummyProductInfoSource{MinLargerThanMax},
			request: ClusterRecommendationReq{
				MinNodes: 10,
				MaxNodes: 5,
				SumMem:   100,
				SumCpu:   100,
			},
			provider:  "dummy",
			attribute: Cpu,

			check: func(values []float64, err error) {
				assert.Equal(t, err.Error(), "min value cannot be larger than the max value")

			},
		},
		{
			name: "error - attribute values could not be retrieved",
			pi:   &dummyProductInfoSource{Error},
			request: ClusterRecommendationReq{
				MinNodes: 10,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			provider:  "dummy",
			attribute: Cpu,

			check: func(values []float64, err error) {
				assert.Nil(t, values, "returned attr values should be nils")
				assert.EqualError(t, err, "error")

			},
		},
		{
			name: "error - unsupported attribute",
			pi:   &dummyProductInfoSource{},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			provider:  "dummy",
			attribute: "error",

			check: func(values []float64, err error) {
				assert.Nil(t, values, "the values should be nil")
				assert.EqualError(t, err, "unsupported attribute: [error]")

			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine, err := NewEngine(test.pi)
			assert.Nil(t, err, "the engine couldn't be created")

			test.check(engine.RecommendAttrValues("dummy", "region", test.attribute, test.request))

		})
	}
}

func TestEngine_RecommendVms(t *testing.T) {
	tests := []struct {
		name      string
		region    string
		pi        ProductInfoSource
		values    []float64
		filters   []vmFilter
		request   ClusterRecommendationReq
		provider  string
		attribute string
		check     func([]VirtualMachine, error)
	}{
		{
			name:   "could not describe region",
			region: "us-west-2",
			pi:     &dummyProductInfoSource{DescribeRegionError},
			values: []float64{1, 2},
			check: func(vms []VirtualMachine, err error) {
				assert.EqualError(t, err, DescribeRegionError)
				assert.Nil(t, vms, "the vms should be nil")
			},
		},
		{
			name:   "could not get product details",
			region: "us-west-2",
			pi:     &dummyProductInfoSource{ProductDetailsError},
			values: []float64{1, 2},
			check: func(vms []VirtualMachine, err error) {
				assert.EqualError(t, err, ProductDetailsError)
				assert.Nil(t, vms, "the vms should be nil")
			},
		},
		{
			name:   "recommend three vm-s",
			region: "us-west-2",
			filters: []vmFilter{func(vm VirtualMachine, req ClusterRecommendationReq) bool {
				return true
			}},
			pi:        &dummyProductInfoSource{},
			values:    []float64{2},
			provider:  "dummy",
			attribute: Cpu,

			check: func(vms []VirtualMachine, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, 3, len(vms))
			},
		},
		{
			name:   "could not find any VMs to recommender",
			region: "us-west-2",
			filters: []vmFilter{func(vm VirtualMachine, req ClusterRecommendationReq) bool {
				return false
			}},
			pi:        &dummyProductInfoSource{},
			values:    []float64{1, 2},
			provider:  "dummy",
			attribute: Cpu,

			check: func(vms []VirtualMachine, err error) {
				assert.EqualError(t, err, "couldn't find any VMs to recommend")
				assert.Nil(t, vms, "the vms should be nil")

			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine, err := NewEngine(test.pi)
			assert.Nil(t, err, "the engine couldn't be created")

			test.check(engine.RecommendVms("dummy", test.region, test.attribute, test.values, test.filters, test.request))

		})
	}
}

func TestEngine_RecommendNodePools(t *testing.T) {
	vms := []VirtualMachine{
		{OnDemandPrice: float64(10), AvgPrice: 99, Cpus: float64(10), Mem: float64(10), Gpus: float64(0)},
		{OnDemandPrice: float64(12), AvgPrice: 89, Cpus: float64(10), Mem: float64(10), Gpus: float64(0)},
	}

	tests := []struct {
		name    string
		pi      ProductInfoSource
		attr    string
		vms     []VirtualMachine
		values  []float64
		request ClusterRecommendationReq
		check   func([]NodePool, error)
	}{
		{
			name:   "successful",
			pi:     &dummyProductInfoSource{},
			vms:    vms,
			attr:   Cpu,
			values: []float64{4},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			check: func(nps []NodePool, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, []NodePool{
					{VmType: VirtualMachine{Type: "", AvgPrice: 99, OnDemandPrice: 10, Cpus: 10, Mem: 10, Gpus: 0, Burst: false}, SumNodes: 0, VmClass: "regular"},
					{VmType: VirtualMachine{Type: "", AvgPrice: 89, OnDemandPrice: 12, Cpus: 10, Mem: 10, Gpus: 0, Burst: false}, SumNodes: 5, VmClass: "spot"},
					{VmType: VirtualMachine{Type: "", AvgPrice: 99, OnDemandPrice: 10, Cpus: 10, Mem: 10, Gpus: 0, Burst: false}, SumNodes: 5, VmClass: "spot"}},
					nps)
				assert.Equal(t, 3, len(nps))

			},
		},
		{
			name:   "attribute error",
			pi:     &dummyProductInfoSource{},
			vms:    vms,
			attr:   "error",
			values: []float64{4},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			check: func(nps []NodePool, err error) {
				assert.EqualError(t, err, "could not get sum for attr: [error], cause: [unsupported attribute: [error]]")
				assert.Nil(t, nps, "the nps should be nil")

			},
		},
		{
			name:   "no spot instances available",
			pi:     &dummyProductInfoSource{},
			vms:    []VirtualMachine{{OnDemandPrice: float64(2), AvgPrice: 0, Cpus: float64(10), Mem: float64(10), Gpus: float64(0)}},
			attr:   Cpu,
			values: []float64{4},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			check: func(nps []NodePool, err error) {
				assert.EqualError(t, err, "no vms suitable for spot pools")
				assert.Nil(t, nps, "the nps should be nil")

			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine, err := NewEngine(test.pi)
			assert.Nil(t, err, "the engine couldn't be created")

			test.check(engine.RecommendNodePools(test.attr, test.vms, test.values, test.request))

		})
	}
}

func TestEngine_sortByAttrValue(t *testing.T) {
	vms := []VirtualMachine{
		{OnDemandPrice: float64(10), AvgPrice: 99, Cpus: float64(10), Mem: float64(10), Gpus: float64(0)},
		{OnDemandPrice: float64(12), AvgPrice: 89, Cpus: float64(10), Mem: float64(10), Gpus: float64(0)},
	}

	tests := []struct {
		name  string
		pi    ProductInfoSource
		attr  string
		vms   []VirtualMachine
		check func(err error)
	}{
		{
			name: "error - unsupported attribute",
			pi:   &dummyProductInfoSource{},
			attr: "error",
			vms:  vms,
			check: func(err error) {
				assert.EqualError(t, err, "unsupported attribute: [error]")
			},
		},
		{
			name: "successful - memory",
			pi:   &dummyProductInfoSource{},
			attr: Memory,
			vms:  vms,
			check: func(err error) {
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name: "successful - cpu",
			pi:   &dummyProductInfoSource{},
			attr: Cpu,
			vms:  vms,
			check: func(err error) {
				assert.Nil(t, err, "the error should be nil")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine, err := NewEngine(test.pi)
			assert.Nil(t, err, "the engine couldn't be created")

			test.check(engine.sortByAttrValue(test.attr, test.vms))

		})
	}
}

func TestEngine_filtersForAttr(t *testing.T) {
	tests := []struct {
		name  string
		pi    ProductInfoSource
		attr  string
		check func(vms []vmFilter, err error)
	}{
		{
			name: "error - unsupported attribute",
			pi:   &dummyProductInfoSource{},
			attr: "error",
			check: func(vms []vmFilter, err error) {
				assert.EqualError(t, err, "unsupported attribute: [error]")
				assert.Nil(t, vms, "the vms should be nil")
			},
		},
		{
			name: "all filters added - cpu",
			pi:   &dummyProductInfoSource{},
			attr: Cpu,
			check: func(vms []vmFilter, err error) {
				assert.Equal(t, 5, len(vms), "invalid filter count")
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name: "all filters added - memory",
			pi:   &dummyProductInfoSource{},
			attr: Memory,
			check: func(vms []vmFilter, err error) {
				assert.Equal(t, 5, len(vms), "invalid filter count")
				assert.Nil(t, err, "the error should be nil")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine, err := NewEngine(test.pi)
			assert.Nil(t, err, "the engine couldn't be created")

			test.check(engine.filtersForAttr(test.attr))

		})
	}
}
