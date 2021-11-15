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
	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

type vmFilter func(vm recommender.VirtualMachine, req recommender.SingleClusterRecommendationReq) bool

// filtersForAttr returns the slice for
func (s *vmSelector) filtersForAttr(attr string, provider string, req recommender.SingleClusterRecommendationReq) ([]vmFilter, error) {
	var filters []vmFilter
	// generic filters - not depending on providers and attributes
	if len(req.IncludeSeries) != 0 {
		filters = append(filters, s.includeSeriesFilter)
	}

	if len(req.ExcludeSeries) != 0 {
		filters = append(filters, s.excludeSeriesFilter)
	}

	if len(req.IncludeTypes) != 0 {
		filters = append(filters, s.includeTypeFilter)
	}

	if len(req.ExcludeTypes) != 0 {
		filters = append(filters, s.excludeTypeFilter)
	}

	if len(req.Category) != 0 {
		filters = append(filters, s.categoryFilter)
	}

	if req.Zone != "" {
		filters = append(filters, s.zonesFilter)
	}

	if len(req.NetworkPerf) != 0 {
		filters = append(filters, s.ntwPerformanceFilter)
	}

	// provider specific filters
	switch provider {
	case "amazon":
		// burst is not allowed
		if req.AllowBurst != nil && !*req.AllowBurst {
			filters = append(filters, s.burstFilter)
		}
		if req.AllowOlderGen == nil || !*req.AllowOlderGen {
			filters = append(filters, s.currentGenFilter)
		}
	}

	// attribute specific filters
	switch attr {
	case recommender.Cpu:
		// nNodes * vm.Mem >= req.SumMem
		filters = append(filters, s.minMemRatioFilter)
	case recommender.Memory:
		filters = append(filters, s.minCpuRatioFilter)
	default:
		return nil, emperror.With(errors.New("unsupported attribute"), "attribute", attr)
	}

	s.log.Debug("filters are successfully registered", map[string]interface{}{"numberOfFilters": len(filters)})
	return filters, nil
}

// filtersApply returns true if all the filters apply for the given vm
func (s *vmSelector) filtersApply(vm recommender.VirtualMachine, filters []vmFilter, req recommender.SingleClusterRecommendationReq) bool {
	for _, filter := range filters {
		if !filter(vm, req) {
			// one of the filters doesn't apply - quit the iteration
			return false
		}
	}
	// no filters or applies
	return true
}

func (s *vmSelector) zonesFilter(vm recommender.VirtualMachine, req recommender.SingleClusterRecommendationReq) bool {
	if len(vm.Zones) != 0 {
		return s.contains(vm.Zones, req.Zone)
	}
	return true
}

func (s *vmSelector) minMemRatioFilter(vm recommender.VirtualMachine, req recommender.SingleClusterRecommendationReq) bool {
	minMemToCpuRatio := req.SumMem / req.SumCpu
	return minMemToCpuRatio <= vm.GetAllocatableAttrValue(recommender.Memory)/vm.GetAllocatableAttrValue(recommender.Cpu)
}

func (s *vmSelector) burstFilter(vm recommender.VirtualMachine, req recommender.SingleClusterRecommendationReq) bool {
	return !vm.Burst
}

func (s *vmSelector) minCpuRatioFilter(vm recommender.VirtualMachine, req recommender.SingleClusterRecommendationReq) bool {
	minCpuToMemRatio := req.SumCpu / req.SumMem
	return minCpuToMemRatio <= vm.GetAllocatableAttrValue(recommender.Cpu)/vm.GetAllocatableAttrValue(recommender.Memory)
}

func (s *vmSelector) ntwPerformanceFilter(vm recommender.VirtualMachine, req recommender.SingleClusterRecommendationReq) bool {
	return s.contains(req.NetworkPerf, vm.NetworkPerfCat)
}

func (s *vmSelector) categoryFilter(vm recommender.VirtualMachine, req recommender.SingleClusterRecommendationReq) bool {
	return s.contains(req.Category, vm.Category)
}

// excludeTypeFilter checks for the vm type in the request' exclude list, the filter passes if the type is not excluded
func (s *vmSelector) excludeTypeFilter(vm recommender.VirtualMachine, req recommender.SingleClusterRecommendationReq) bool {
	if s.contains(req.ExcludeTypes, vm.Type) {
		s.log.Debug("the vm type is blacklisted", map[string]interface{}{"type": vm.Type})
		return false
	}
	return true
}

// includeTypeFilter checks whether the vm type is in the includes list; the filter passes if the type is in the list
func (s *vmSelector) includeTypeFilter(vm recommender.VirtualMachine, req recommender.SingleClusterRecommendationReq) bool {
	if s.contains(req.IncludeTypes, vm.Type) {
		s.log.Debug("the vm type is whitelisted", map[string]interface{}{"type": vm.Type})
		return true
	}
	return false
}

// includeSeriesFilter checks whether the vm type is in the includes list; the filter passes if the series is in the list
func (s *vmSelector) includeSeriesFilter(vm recommender.VirtualMachine, req recommender.SingleClusterRecommendationReq) bool {
	if s.contains(req.IncludeSeries, vm.Series) {
		s.log.Debug("the vm series is whitelisted", map[string]interface{}{"series": vm.Series})
		return true
	}
	return false
}

// excludeSeriesFilter checks whether the vm type is in the exclude list; the filter passes if the series is not excluded
func (s *vmSelector) excludeSeriesFilter(vm recommender.VirtualMachine, req recommender.SingleClusterRecommendationReq) bool {
	if s.contains(req.ExcludeSeries, vm.Series) {
		s.log.Debug("the vm series is blacklisted", map[string]interface{}{"series": vm.Series})
		return false
	}
	return true
}

// filterSpots selects vm-s that potentially can be part of "spot" node pools
func (s *vmSelector) filterSpots(vms []recommender.VirtualMachine) []recommender.VirtualMachine {
	s.log.Debug("selecting spot instances for recommending spot pools")
	fvms := make([]recommender.VirtualMachine, 0)
	for _, vm := range vms {
		if vm.AvgPrice != 0 {
			fvms = append(fvms, vm)
		}
	}
	return fvms
}

// currentGenFilter removes instance types that are not the current generation (amazon only)
func (s *vmSelector) currentGenFilter(vm recommender.VirtualMachine, req recommender.SingleClusterRecommendationReq) bool {
	// filter by current generation
	return vm.CurrentGen
}

// contains is a helper function to check if a slice contains a string
func (s *vmSelector) contains(slice []string, str string) bool {
	for _, e := range slice {
		if e == str {
			return true
		}
	}
	return false
}
