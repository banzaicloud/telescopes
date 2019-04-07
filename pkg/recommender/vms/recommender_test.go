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
	"github.com/stretchr/testify/assert"
)

var productDetails = []*models.ProductDetails{
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
		Cpus:          17,
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
			selector := NewVmSelector(logur.NewTestLogger())
			test.check(selector.RecommendVms("google", vms, test.attribute, test.request, nil))
		})
	}
}

func TestVmSelector_recommendAttrValues(t *testing.T) {
	tests := []struct {
		name      string
		request   recommender.ClusterRecommendationReq
		attribute string
		check     func([]float64, error)
	}{

		{
			name: "successfully get recommended attribute values",
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
				assert.Equal(t, float64(16), values[0], "recommended number of values is not as expected")

			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selector := NewVmSelector(logur.NewTestLogger())
			test.check(selector.recommendAttrValues(productDetails, test.attribute, test.request))
		})
	}
}
