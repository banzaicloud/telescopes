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

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo-client/models"
	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/goph/logur"
	"github.com/pkg/errors"
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
		return nil, errors.New(Error)
	}
	return []float64{15, 16, 17}, nil
}

func (piCli *dummyCloudInfoSource) GetZones(provider string, service string, region string) ([]string, error) {
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

func TestVmSelector_RecommendVms(t *testing.T) {
	vms := []recommender.VirtualMachine{
		{
			Type:          "n1-standard-2",
			Cpus:          2,
			Mem:           7.5,
			OnDemandPrice: 0.0949995,
		},
		{
			Type:          "n1-highcpu-4",
			Cpus:          2,
			Mem:           7.5,
			OnDemandPrice: 0.0949995,
		},
		{
			Type:          "n1-highmem-4",
			Cpus:          2,
			Mem:           7.5,
			OnDemandPrice: 0.0949995,
		},
	}
	tests := []struct {
		name      string
		values    []float64
		request   recommender.ClusterRecommendationReq
		attribute string
		check     func([]recommender.VirtualMachine, []recommender.VirtualMachine, error)
	}{
		{
			name:   "recommend three vm-s",
			values: []float64{2},
			request: recommender.ClusterRecommendationReq{
				MinNodes:    3,
				MaxNodes:    3,
				OnDemandPct: 100,
				SumCpu:      6,
				SumMem:      13,
			},
			attribute: recommender.Cpu,
			check: func(odVms []recommender.VirtualMachine, spotVms []recommender.VirtualMachine, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, 3, len(odVms))
				assert.Equal(t, 3, len(spotVms))
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger(), nil)
			test.check(selector.RecommendVms("google", vms, test.attribute, test.request, nil, logur.NewTestLogger()))
		})
	}
}

func TestVmSelector_recommendAttrValues(t *testing.T) {

	tests := []struct {
		name      string
		pi        recommender.CloudInfoSource
		request   recommender.ClusterRecommendationReq
		attribute string
		check     func([]float64, error)
	}{
		{
			name: "all attributes between limits",
			pi:   &dummyCloudInfoSource{},
			request: recommender.ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			attribute: recommender.Cpu,
			check: func(values []float64, err error) {
				assert.Nil(t, err, "should not get error when recommending attributes")
				assert.Equal(t, 3, len(values), "recommended number of values is not as expected")
				assert.Equal(t, []float64{15, 16, 17}, values)
			},
		},
		{
			name: "attributes out of limits not recommended",
			pi:   &dummyCloudInfoSource{OutOfLimits},
			request: recommender.ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			attribute: recommender.Cpu,
			check: func(values []float64, err error) {
				assert.Nil(t, err, "should not get error when recommending attributes")
				assert.Equal(t, 2, len(values), "recommended number of values is not as expected")

			},
		},
		{
			name: "no values between limits found - smallest value returned",
			pi:   &dummyCloudInfoSource{LargerValues},
			request: recommender.ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			attribute: recommender.Cpu,
			check: func(values []float64, err error) {
				assert.Nil(t, err, "should not get error when recommending attributes")
				assert.Equal(t, 1, len(values), "recommended number of values is not as expected")
				assert.Equal(t, float64(30), values[0], "recommended number of values is not as expected")

			},
		},
		{
			name: "no values between limits found - largest value returned",
			pi:   &dummyCloudInfoSource{SmallerValues},
			request: recommender.ClusterRecommendationReq{
				MinNodes: 5,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			attribute: recommender.Cpu,
			check: func(values []float64, err error) {
				assert.Nil(t, err, "should not get error when recommending attributes")
				assert.Equal(t, 1, len(values), "recommended number of values is not as expected")
				assert.Equal(t, float64(9), values[0], "recommended number of values is not as expected")

			},
		},
		{
			name: "error - attribute values could not be retrieved",
			pi:   &dummyCloudInfoSource{Error},
			request: recommender.ClusterRecommendationReq{
				MinNodes: 10,
				MaxNodes: 10,
				SumMem:   100,
				SumCpu:   100,
			},
			attribute: recommender.Cpu,
			check: func(values []float64, err error) {
				assert.Nil(t, values, "returned attr values should be nils")
				assert.EqualError(t, err, Error)

			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger(), test.pi)
			test.check(selector.recommendAttrValues("dummyProvider", "dummyService", "dummyRegion", test.attribute, test.request))
		})
	}
}
