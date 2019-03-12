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
	"fmt"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo-client/models"
	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/pkg/errors"
)

type vmSelector struct {
	ciSource recommender.CloudInfoSource
	log      logur.Logger
}

func NewVmSelector(log logur.Logger, ciSource recommender.CloudInfoSource) *vmSelector {
	return &vmSelector{
		ciSource: ciSource,
		log:      log,
	}
}

// RecommendVms selects a slice of VirtualMachines for the given attribute and requirements in the request
func (s *vmSelector) RecommendVms(provider string, vms []recommender.VirtualMachine, attr string, req recommender.ClusterRecommendationReq, layout []recommender.NodePool, log logur.Logger) ([]recommender.VirtualMachine, []recommender.VirtualMachine, error) {
	s.log = log
	s.log.Info("recommending virtual machines", map[string]interface{}{"attribute": attr})

	vmFilters, err := s.filtersForAttr(attr, provider)
	if err != nil {
		return nil, nil, emperror.Wrap(err, "failed to identify filters")
	}

	var filteredVms []recommender.VirtualMachine
	for _, vm := range vms {
		if s.filtersApply(vm, vmFilters, req) {
			filteredVms = append(filteredVms, vm)
		}
	}

	if len(filteredVms) == 0 {
		s.log.Debug("no virtual machines found", map[string]interface{}{"attribute": attr})
		return []recommender.VirtualMachine{}, []recommender.VirtualMachine{}, nil
	}

	var odVms, spotVms []recommender.VirtualMachine
	if layout == nil {
		odVms, spotVms = filteredVms, filteredVms
	} else {
		for _, np := range layout {
			for _, vm := range filteredVms {
				if np.VmType.Type == vm.Type {
					if np.VmClass == recommender.Regular {
						odVms = append(odVms, vm)
					} else {
						spotVms = append(spotVms, vm)
					}
					continue
				}
			}
		}
	}

	if req.OnDemandPct < 100 {
		// retain only the nodes that are available as spot instances
		spotVms = s.filterSpots(spotVms)
		if len(spotVms) == 0 {
			s.log.Debug("no vms suitable for spot pools", map[string]interface{}{"attribute": attr})
			return []recommender.VirtualMachine{}, []recommender.VirtualMachine{}, nil
		}
	}

	return odVms, spotVms, nil
}

func (s *vmSelector) FindVmsWithAttrValues(provider string, service string, region string, attr string, req recommender.ClusterRecommendationReq, layoutDesc []recommender.NodePoolDesc) ([]recommender.VirtualMachine, error) {
	var (
		values []float64
		err    error
	)
	if layoutDesc == nil {
		values, err = s.recommendAttrValues(provider, service, region, attr, req)
		if err != nil {
			return nil, emperror.Wrap(err, "failed to recommend attribute values")
		}
		s.log.Debug(fmt.Sprintf("recommended values for [%s]: count:[%d] , values: [%#v./te]", attr, len(values), values))
	}
	s.log.Info("looking for instance types", map[string]interface{}{"attribute": attr, "values": values})

	var (
		vms []recommender.VirtualMachine
	)
	zones := req.Zones

	if len(zones) == 0 {
		if zones, err = s.ciSource.GetZones(provider, service, region); err != nil {
			return nil, err
		}
	}

	allProducts, err := s.ciSource.GetProductDetails(provider, service, region)
	if err != nil {
		return nil, err
	}

	for _, p := range allProducts {
		included := true
		if len(values) > 0 {
			included = false
			for _, v := range values {
				switch attr {
				case recommender.Cpu:
					if p.Cpus == v {
						included = true
						continue
					}
				case recommender.Memory:
					if p.Mem == v {
						included = true
						continue
					}
				default:
					return nil, errors.New("unsupported attribute")
				}
			}
		}
		if included {
			vms = append(vms, recommender.VirtualMachine{
				Type:           p.Type,
				OnDemandPrice:  p.OnDemandPrice,
				AvgPrice:       avg(p.SpotPrice, zones),
				Cpus:           p.Cpus,
				Mem:            p.Mem,
				Gpus:           p.Gpus,
				Burst:          p.Burst,
				NetworkPerf:    p.NtwPerf,
				NetworkPerfCat: p.NtwPerfCat,
				CurrentGen:     p.CurrentGen,
			})
		}
	}

	s.log.Debug("found vms", map[string]interface{}{attr: values, "vms": vms})
	return vms, nil
}

// recommendAttrValues selects the attribute values allowed to participate in the recommendation process
func (s *vmSelector) recommendAttrValues(provider string, service string, region string, attr string, req recommender.ClusterRecommendationReq) ([]float64, error) {

	allValues, err := s.ciSource.GetAttributeValues(provider, service, region, attr)
	if err != nil {
		return nil, err
	}

	s.log.Debug("selecting attributes", map[string]interface{}{"attribute": attr, "values": allValues})
	values, err := AttributeValues(allValues).SelectAttributeValues(minValuePerVm(req, attr), maxValuePerVm(req, attr))
	if err != nil {
		return nil, emperror.With(err, recommender.RecommenderErrorTag, "attributes")
	}

	return values, nil
}

// maxValuePerVm calculates the maximum value per node for the given attribute
func maxValuePerVm(req recommender.ClusterRecommendationReq, attr string) float64 {
	switch attr {
	case recommender.Cpu:
		return req.SumCpu / float64(req.MinNodes)
	case recommender.Memory:
		return req.SumMem / float64(req.MinNodes)
	default:
		return 0
	}
}

// minValuePerVm calculates the minimum value per node for the given attribute
func minValuePerVm(req recommender.ClusterRecommendationReq, attr string) float64 {
	switch attr {
	case recommender.Cpu:
		return req.SumCpu / float64(req.MaxNodes)
	case recommender.Memory:
		return req.SumMem / float64(req.MaxNodes)
	default:
		return 0
	}
}

func avg(prices []*models.ZonePrice, recZones []string) float64 {
	if len(prices) == 0 {
		return 0.0
	}
	avgPrice := 0.0
	for _, price := range prices {
		for _, z := range recZones {
			if z == price.Zone {
				avgPrice += price.Price
			}
		}
	}
	return avgPrice / float64(len(prices))
}
