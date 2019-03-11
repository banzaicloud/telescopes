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

type vmFilter func(vm recommender.VirtualMachine, req recommender.ClusterRecommendationReq) bool

// filtersForAttr returns the slice for
func (s *vmSelector) filtersForAttr(attr string, provider string) ([]vmFilter, error) {
	// generic filters - not depending on providers and attributes
	var filters = []vmFilter{s.includesFilter, s.excludesFilter}

	// provider specific filters
	switch provider {
	case "amazon":
		filters = append(filters, s.currentGenFilter, s.burstFilter, s.ntwPerformanceFilter)
	case "google", "alibaba":
		filters = append(filters, s.ntwPerformanceFilter)
	}

	// attribute specific filters
	switch attr {
	case recommender.Cpu:
		filters = append(filters, s.minMemRatioFilter)
	case recommender.Memory:
		filters = append(filters, s.minCpuRatioFilter)
	default:
		return nil, emperror.With(errors.New("unsupported attribute"), "attribute", attr)
	}

	return filters, nil
}

// filtersApply returns true if all the filters apply for the given vm
func (s *vmSelector) filtersApply(vm recommender.VirtualMachine, filters []vmFilter, req recommender.ClusterRecommendationReq) bool {
	for _, filter := range filters {
		if !filter(vm, req) {
			// one of the filters doesn't apply - quit the iteration
			return false
		}
	}
	// no filters or applies
	return true
}

func (s *vmSelector) minMemRatioFilter(vm recommender.VirtualMachine, req recommender.ClusterRecommendationReq) bool {
	minMemToCpuRatio := req.SumMem / req.SumCpu
	return minMemToCpuRatio <= vm.Mem/vm.Cpus
}

func (s *vmSelector) burstFilter(vm recommender.VirtualMachine, req recommender.ClusterRecommendationReq) bool {
	// if not specified in req or it's allowed the filter passes
	if (req.AllowBurst == nil) || *(req.AllowBurst) {
		return true
	}
	// burst is not allowed
	return !vm.Burst
}

func (s *vmSelector) minCpuRatioFilter(vm recommender.VirtualMachine, req recommender.ClusterRecommendationReq) bool {
	minCpuToMemRatio := req.SumCpu / req.SumMem
	return minCpuToMemRatio <= vm.Cpus/vm.Mem
}

func (s *vmSelector) ntwPerformanceFilter(vm recommender.VirtualMachine, req recommender.ClusterRecommendationReq) bool {
	if req.NetworkPerf == nil { //there is no filter set
		return true
	}
	if vm.NetworkPerfCat == *req.NetworkPerf { //the network performance category matches the vm
		return true
	}
	return false
}

// excludeFilter checks for the vm type in the request' exclude list, the filter  passes if the type is not excluded
func (s *vmSelector) excludesFilter(vm recommender.VirtualMachine, req recommender.ClusterRecommendationReq) bool {
	if req.Excludes == nil || len(req.Excludes) == 0 {
		s.log.Debug("no blacklist provided - all vm types are welcome")
		return true
	}
	if s.contains(req.Excludes, vm.Type) {
		s.log.Debug("the vm type is blacklisted", map[string]interface{}{"type": vm.Type})
		return false
	}
	return true
}

// includesFilter checks whether the vm type is in the includes list; the filter passes if the type is in the list
func (s *vmSelector) includesFilter(vm recommender.VirtualMachine, req recommender.ClusterRecommendationReq) bool {
	if req.Includes == nil || len(req.Includes) == 0 {
		s.log.Debug("no whitelist specified - all vm types are welcome")
		return true
	}
	if s.contains(req.Includes, vm.Type) {
		s.log.Debug("the vm type is whitelisted", map[string]interface{}{"type": vm.Type})
		return true
	}
	return false
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
func (s *vmSelector) currentGenFilter(vm recommender.VirtualMachine, req recommender.ClusterRecommendationReq) bool {
	if req.AllowOlderGen == nil || !*req.AllowOlderGen {
		// filter by current generation by default (if it's not specified in the request) or it's explicitly set to false
		return vm.CurrentGen
	}
	// the flag it's set into the req AND it's true
	return true
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
