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
	"errors"
	"testing"

	"fmt"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo-client/models"
	"github.com/stretchr/testify/assert"
)

const (
	OutOfLimits         = "2 values between 10 - 20"
	LargerValues        = "all values are larger than 20, return the closest value"
	SmallerValues       = "all values are less than 10, return the closest value"
	MinLargerThanMax    = "error, min > max"
	Error               = "error returned"
	DescribeRegionError = "could not describe region"
	ProductDetailsError = "could not get product details"
	AvgPriceNil         = "average price is nil"
)

type dummyCloudInfoSource struct {
	// test case id to drive the behaviour
	TcId string
}

func (piCli *dummyCloudInfoSource) GetAttributeValues(provider string, service string, region string, attr string) ([]float64, error) {
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
		return nil, fmt.Errorf(Error)
	}
	return []float64{15, 16, 17}, nil
}

func (piCli *dummyCloudInfoSource) GetRegion(provider string, service string, region string) ([]string, error) {
	switch piCli.TcId {
	case DescribeRegionError:
		return nil, errors.New(DescribeRegionError)
	default:
		return []string{"dummyZone1", "dummyZone2", "dummyZone3"}, nil
	}
}

func (piCli *dummyCloudInfoSource) GetProductDetails(provider string, service string, region string) ([]*models.ProductDetails, error) {
	switch piCli.TcId {
	case ProductDetailsError:
		return nil, errors.New(ProductDetailsError)
	case AvgPriceNil:
		return []*models.ProductDetails{
			{
				Type:          "type-1",
				CurrentGen:    true,
				OnDemandPrice: 0.68,
				Cpus:          16,
				Mem:           32,
				SpotPrice:     []*models.ZonePrice{{Price: 0.171, Zone: "invalidZone"}},
			},
		}, nil
	default:
		return []*models.ProductDetails{
			{
				Type:          "type-3",
				CurrentGen:    true,
				OnDemandPrice: 0.023,
				Cpus:          1,
				Mem:           2,
				SpotPrice:     []*models.ZonePrice{{Price: 0.0069, Zone: "dummyZone1"}},
			},
			{
				Type:          "type-4",
				CurrentGen:    true,
				OnDemandPrice: 0.096,
				Cpus:          2,
				Mem:           4,
				SpotPrice:     []*models.ZonePrice{{Price: 0.018, Zone: "dummyZone2"}},
			},
			{
				Type:          "type-5",
				CurrentGen:    true,
				OnDemandPrice: 0.046,
				Cpus:          2,
				Mem:           4,
				SpotPrice:     []*models.ZonePrice{{Price: 0.014, Zone: "dummyZone2"}},
			},
			{
				Type:          "type-6",
				CurrentGen:    true,
				OnDemandPrice: 0.096,
				Cpus:          2,
				Mem:           8,
				SpotPrice:     []*models.ZonePrice{{Price: 0.02, Zone: "dummyZone1"}},
			},
			{
				Type:          "type-7",
				CurrentGen:    true,
				OnDemandPrice: 0.17,
				Cpus:          4,
				Mem:           8,
				SpotPrice:     []*models.ZonePrice{{Price: 0.037, Zone: "dummyZone3"}},
			},
			{
				Type:          "type-8",
				CurrentGen:    true,
				OnDemandPrice: 0.186,
				Cpus:          4,
				Mem:           16,
				SpotPrice:     []*models.ZonePrice{{Price: 0.056, Zone: "dummyZone2"}},
			},
			{
				Type:          "type-9",
				CurrentGen:    true,
				OnDemandPrice: 0.34,
				Cpus:          8,
				Mem:           16,
				SpotPrice:     []*models.ZonePrice{{Price: 0.097, Zone: "dummyZone1"}},
			},
			{
				Type:          "type-10",
				CurrentGen:    true,
				OnDemandPrice: 0.68,
				Cpus:          16,
				Mem:           32,
				SpotPrice:     []*models.ZonePrice{{Price: 0.171, Zone: "dummyZone2"}},
			},
			{
				Type:          "type-11",
				CurrentGen:    true,
				OnDemandPrice: 0.91,
				Cpus:          16,
				Mem:           64,
				SpotPrice:     []*models.ZonePrice{{Price: 0.157, Zone: "dummyZone1"}},
			},
			{
				Type:          "type-12",
				CurrentGen:    true,
				OnDemandPrice: 1.872,
				Cpus:          32,
				Mem:           128,
				SpotPrice:     []*models.ZonePrice{{Price: 0.66, Zone: "dummyZone3"}},
			},
		}, nil
	}
}

func TestEngine_RecommendAttrValues(t *testing.T) {

	tests := []struct {
		name      string
		pi        CloudInfoSource
		request   ClusterRecommendationReq
		attribute string
		check     func([]float64, error)
	}{
		{
			name: "all attributes between limits",
			pi:   &dummyCloudInfoSource{},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			attribute: Cpu,
			check: func(values []float64, err error) {
				assert.Nil(t, err, "should not get error when recommending attributes")
				assert.Equal(t, 3, len(values), "recommended number of values is not as expected")
				assert.Equal(t, []float64{15, 16, 17}, values)
			},
		},
		{
			name: "attributes out of limits not recommended",
			pi:   &dummyCloudInfoSource{OutOfLimits},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			attribute: Cpu,
			check: func(values []float64, err error) {
				assert.Nil(t, err, "should not get error when recommending attributes")
				assert.Equal(t, 2, len(values), "recommended number of values is not as expected")

			},
		},
		{
			name: "no values between limits found - smallest value returned",
			pi:   &dummyCloudInfoSource{LargerValues},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			attribute: Cpu,
			check: func(values []float64, err error) {
				assert.Nil(t, err, "should not get error when recommending attributes")
				assert.Equal(t, 1, len(values), "recommended number of values is not as expected")
				assert.Equal(t, float64(30), values[0], "recommended number of values is not as expected")

			},
		},
		{
			name: "no values between limits found - largest value returned",
			pi:   &dummyCloudInfoSource{SmallerValues},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			attribute: Cpu,
			check: func(values []float64, err error) {
				assert.Nil(t, err, "should not get error when recommending attributes")
				assert.Equal(t, 1, len(values), "recommended number of values is not as expected")
				assert.Equal(t, float64(9), values[0], "recommended number of values is not as expected")

			},
		},
		{
			name: "error - attribute values could not be retrieved",
			pi:   &dummyCloudInfoSource{Error},
			request: ClusterRecommendationReq{
				MinNodes: 10,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			attribute: Cpu,
			check: func(values []float64, err error) {
				assert.Nil(t, values, "returned attr values should be nils")
				assert.EqualError(t, err, Error)

			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine := NewEngine(test.pi)
			test.check(engine.RecommendAttrValues(context.TODO(), "dummy", "region", "service", test.attribute, test.request))
		})
	}
}

func TestEngine_RecommendVms(t *testing.T) {
	tests := []struct {
		name      string
		pi        CloudInfoSource
		values    []float64
		filters   []vmFilter
		request   ClusterRecommendationReq
		attribute string
		check     func([]VirtualMachine, error)
	}{
		{
			name: "could not describe region",
			filters: []vmFilter{func(ctx context.Context, vm VirtualMachine, req ClusterRecommendationReq) bool {
				return true
			}},
			pi:     &dummyCloudInfoSource{DescribeRegionError},
			values: []float64{1, 2},
			check: func(vms []VirtualMachine, err error) {
				assert.EqualError(t, err, DescribeRegionError)
				assert.Nil(t, vms, "the vms should be nil")
			},
		},
		{
			name: "could not get product details",
			filters: []vmFilter{func(ctx context.Context, vm VirtualMachine, req ClusterRecommendationReq) bool {
				return true
			}},
			pi:     &dummyCloudInfoSource{ProductDetailsError},
			values: []float64{1, 2},
			check: func(vms []VirtualMachine, err error) {
				assert.EqualError(t, err, ProductDetailsError)
				assert.Nil(t, vms, "the vms should be nil")
			},
		},
		{
			name: "recommend three vm-s",
			filters: []vmFilter{func(ctx context.Context, vm VirtualMachine, req ClusterRecommendationReq) bool {
				return true
			}},
			pi:        &dummyCloudInfoSource{},
			values:    []float64{2},
			attribute: Cpu,
			check: func(vms []VirtualMachine, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, 3, len(vms))
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine := NewEngine(test.pi)

			test.check(engine.RecommendVms(context.TODO(), "dummy", "dummyService", "dummyRegion", test.attribute, test.values, test.filters, test.request))

		})
	}
}

func TestEngine_RecommendNodePools(t *testing.T) {
	vms := []VirtualMachine{
		{OnDemandPrice: float64(10), AvgPrice: 99, Cpus: float64(10), Mem: float64(10), Gpus: float64(0)},
		{OnDemandPrice: float64(12), AvgPrice: 89, Cpus: float64(10), Mem: float64(10), Gpus: float64(0)},
	}
	req := ClusterRecommendationReq{
		MinNodes: 5,
		MaxNodes: 10,
		SumMem:   100,
		SumCpu:   100,
	}

	tests := []struct {
		name  string
		pi    CloudInfoSource
		attr  string
		vms   []VirtualMachine
		check func([]NodePool, error)
	}{
		{
			name: "successful",
			pi:   &dummyCloudInfoSource{},
			vms:  vms,
			attr: Cpu,
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
			name: "no spot instances available",
			pi:   &dummyCloudInfoSource{},
			vms:  []VirtualMachine{{OnDemandPrice: float64(2), AvgPrice: 0, Cpus: float64(10), Mem: float64(10), Gpus: float64(0)}},
			attr: Cpu,
			check: func(nps []NodePool, err error) {
				assert.EqualError(t, err, "no vms suitable for spot pools")
				assert.Nil(t, nps, "the nps should be nil")

			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine := NewEngine(test.pi)

			test.check(engine.RecommendNodePools(context.TODO(), test.attr, test.vms, []float64{4}, req))

		})
	}
}

func TestEngine_RecommendCluster(t *testing.T) {
	tests := []struct {
		name    string
		pi      CloudInfoSource
		request ClusterRecommendationReq
		check   func(resp *ClusterRecommendationResp, err error)
	}{
		{
			name: "cluster recommendation success - only 1 vm is requested (min = max = 1)",
			pi:   &dummyCloudInfoSource{},
			request: ClusterRecommendationReq{
				MinNodes: 1,
				MaxNodes: 1,
				SumMem:   32,
				SumCpu:   16,
			},
			check: func(resp *ClusterRecommendationResp, err error) {
				assert.Equal(t, 3, len(resp.NodePools))
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, float64(64), resp.Accuracy.RecMem)
				assert.Equal(t, float64(16), resp.Accuracy.RecCpu)
				assert.Equal(t, 1, resp.Accuracy.RecNodes)
			},
		},
		{
			name: "cluster recommendation success - on-demand pct is 0%,",
			pi:   &dummyCloudInfoSource{},
			request: ClusterRecommendationReq{
				MinNodes:    5,
				MaxNodes:    10,
				SumMem:      100,
				SumCpu:      100,
				OnDemandPct: 0,
			},
			check: func(resp *ClusterRecommendationResp, err error) {
				assert.Equal(t, 7, resp.Accuracy.RecNodes)
				assert.Equal(t, float64(112), resp.Accuracy.RecCpu)
				assert.Equal(t, float64(352), resp.Accuracy.RecMem)
				assert.Equal(t, 3, len(resp.NodePools))
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, 7, resp.Accuracy.RecSpotNodes)
				assert.Equal(t, 0, resp.Accuracy.RecRegularNodes)
				assert.Equal(t, resp.Accuracy.RecSpotPrice, resp.Accuracy.RecTotalPrice)

			},
		},
		{
			name: "cluster recommendation success - the zone is specified",
			pi:   &dummyCloudInfoSource{},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
				Zones:    []string{"dummyZone1"},
			},
			check: func(resp *ClusterRecommendationResp, err error) {
				assert.Equal(t, 7, resp.Accuracy.RecNodes)
				assert.Equal(t, float64(112), resp.Accuracy.RecCpu)
				assert.Equal(t, float64(448), resp.Accuracy.RecMem)
				assert.Equal(t, 2, len(resp.NodePools))
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name: "cluster recommendation success - on-demand pct is 100%, only regular nodes in the recommended cluster",
			pi:   &dummyCloudInfoSource{},
			request: ClusterRecommendationReq{
				MinNodes:    5,
				MaxNodes:    10,
				SumMem:      100,
				SumCpu:      100,
				OnDemandPct: 100,
			},
			check: func(resp *ClusterRecommendationResp, err error) {
				assert.Equal(t, 7, resp.Accuracy.RecNodes)
				assert.Equal(t, float64(112), resp.Accuracy.RecCpu)
				assert.Equal(t, float64(224), resp.Accuracy.RecMem)
				assert.Equal(t, 3, len(resp.NodePools))
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, 0, resp.Accuracy.RecSpotNodes)
				assert.Equal(t, 7, resp.Accuracy.RecRegularNodes)
				assert.Equal(t, resp.Accuracy.RecRegularPrice, resp.Accuracy.RecTotalPrice)
			},
		},
		{
			name: "cluster recommendation success - mem / cpu ratio is very large",
			pi:   &dummyCloudInfoSource{},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   1,
				SumCpu:   100,
			},
			check: func(resp *ClusterRecommendationResp, err error) {
				assert.Equal(t, 7, resp.Accuracy.RecNodes)
				assert.Equal(t, float64(112), resp.Accuracy.RecCpu)
				assert.Equal(t, float64(352), resp.Accuracy.RecMem)
				assert.Equal(t, 3, len(resp.NodePools))
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name: "cluster recommendation success - cpu / mem ratio is very large",
			pi:   &dummyCloudInfoSource{},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   1,
			},
			check: func(resp *ClusterRecommendationResp, err error) {
				assert.Equal(t, 7, resp.Accuracy.RecNodes)
				assert.Equal(t, float64(40), resp.Accuracy.RecCpu)
				assert.Equal(t, float64(112), resp.Accuracy.RecMem)
				assert.Equal(t, 3, len(resp.NodePools))
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name: "cluster recommendation success - min nodes is 1 and max is 100",
			pi:   &dummyCloudInfoSource{},
			request: ClusterRecommendationReq{
				MinNodes: 1,
				MaxNodes: 100,
				SumMem:   100,
				SumCpu:   100,
			},
			check: func(resp *ClusterRecommendationResp, err error) {
				assert.Equal(t, 7, resp.Accuracy.RecNodes)
				assert.Equal(t, float64(112), resp.Accuracy.RecCpu)
				assert.Equal(t, float64(352), resp.Accuracy.RecMem)
				assert.Equal(t, 3, len(resp.NodePools))
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name: "cluster recommendation success - a very large number of nodes, cpus and mems are requested",
			pi:   &dummyCloudInfoSource{},
			request: ClusterRecommendationReq{
				MinNodes: 10000,
				MaxNodes: 100000,
				SumMem:   1000000,
				SumCpu:   1000000,
			},
			check: func(resp *ClusterRecommendationResp, err error) {
				assert.Equal(t, 62500, resp.Accuracy.RecNodes)
				assert.Equal(t, float64(1e+06), resp.Accuracy.RecCpu)
				assert.Equal(t, float64(3e+06), resp.Accuracy.RecMem)
				assert.Equal(t, 3, len(resp.NodePools))
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name: "when neither of the selected VMs have a spot price available (avgPrice = 0 for all VMs), we should report an error",
			pi:   &dummyCloudInfoSource{TcId: AvgPriceNil},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			check: func(resp *ClusterRecommendationResp, err error) {
				assert.Nil(t, resp, "the response should be nil")
				assert.EqualError(t, err, "failed to recommend virtual machines: no vms suitable for spot pools")
			},
		},
		{
			name: "when we could not select a slice of VMs for the requirements in the request (could not get product details), we should report an error",
			pi:   &dummyCloudInfoSource{ProductDetailsError},
			request: ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			check: func(resp *ClusterRecommendationResp, err error) {
				assert.Nil(t, resp, "the response should be nil")
				assert.EqualError(t, err, "failed to recommend virtual machines: could not get product details")
			},
		},
		{
			name: "could not find the slice of NodePools that may participate in the recommendation process (len(nodePools) = 0)",
			pi:   &dummyCloudInfoSource{},
			request: ClusterRecommendationReq{
				MinNodes: 8,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			check: func(resp *ClusterRecommendationResp, err error) {
				assert.Nil(t, resp, "the response should be nil")
				assert.EqualError(t, err, "could not recommend cluster with the requested resources")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine := NewEngine(test.pi)

			test.check(engine.RecommendCluster(context.TODO(), "dummy", "dummyService", "dummyRegion", test.request))
		})
	}
}
