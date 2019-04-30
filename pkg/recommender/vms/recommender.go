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

	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/pkg/errors"
)

type vmSelector struct {
	log logur.Logger
}

func NewVmSelector(log logur.Logger) *vmSelector {
	return &vmSelector{
		log: log,
	}
}

// RecommendVms selects a slice of VirtualMachines for the given attribute and requirements in the request
func (s *vmSelector) RecommendVms(provider string, vms []recommender.VirtualMachine, attr string, req recommender.SingleClusterRecommendationReq, layout []recommender.NodePool) ([]recommender.VirtualMachine, []recommender.VirtualMachine, error) {
	s.log.Info("recommending virtual machines", map[string]interface{}{"attribute": attr})

	vmFilters, err := s.filtersForAttr(attr, provider, req)
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

func (s *vmSelector) FindVmsWithAttrValues(attr string, req recommender.SingleClusterRecommendationReq, layoutDesc []recommender.NodePoolDesc, allProducts []recommender.VirtualMachine) ([]recommender.VirtualMachine, error) {
	var (
		vms    []recommender.VirtualMachine
		values []float64
		err    error
	)

	if layoutDesc == nil {
		values, err = s.recommendAttrValues(allProducts, attr, req)
		if err != nil {
			return nil, emperror.Wrap(err, "failed to recommend attribute values")
		}
		s.log.Debug(fmt.Sprintf("recommended values for [%s]: count:[%d] , values: [%#v./te]", attr, len(values), values))
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
					}
				case recommender.Memory:
					if p.Mem == v {
						included = true
					}
				default:
					return nil, errors.New("unsupported attribute")
				}
			}
		}
		if included {
			vms = append(vms, p)
		}
	}

	s.log.Debug("found vms", map[string]interface{}{attr: values, "vms": vms})
	return vms, nil
}

// recommendAttrValues selects the attribute values allowed to participate in the recommendation process
func (s *vmSelector) recommendAttrValues(allProducts []recommender.VirtualMachine, attr string, req recommender.SingleClusterRecommendationReq) ([]float64, error) {

	allValues := make([]float64, 0)
	valueSet := make(map[float64]interface{})

	for _, vm := range allProducts {
		switch attr {
		case recommender.Cpu:
			valueSet[vm.Cpus] = ""
		case recommender.Memory:
			valueSet[vm.Mem] = ""
		}
	}
	for attr := range valueSet {
		allValues = append(allValues, attr)
	}

	s.log.Debug("selecting attributes", map[string]interface{}{"attribute": attr, "values": allValues})
	values, err := AttributeValues(allValues).SelectAttributeValues(minValuePerVm(req, attr), maxValuePerVm(req, attr))
	if err != nil {
		return nil, emperror.With(err, recommender.RecommenderErrorTag, "attributes")
	}

	return values, nil
}

// maxValuePerVm calculates the maximum value per node for the given attribute
func maxValuePerVm(req recommender.SingleClusterRecommendationReq, attr string) float64 {
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
func minValuePerVm(req recommender.SingleClusterRecommendationReq, attr string) float64 {
	switch attr {
	case recommender.Cpu:
		return req.SumCpu / float64(req.MaxNodes)
	case recommender.Memory:
		return req.SumMem / float64(req.MaxNodes)
	default:
		return 0
	}
}
