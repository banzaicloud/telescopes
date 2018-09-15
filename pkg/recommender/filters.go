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
	log "github.com/sirupsen/logrus"
)

type vmFilter func(vm VirtualMachine, req ClusterRecommendationReq) bool

func (e *Engine) minMemRatioFilter(vm VirtualMachine, req ClusterRecommendationReq) bool {
	minMemToCpuRatio := req.SumMem / req.SumCpu
	if vm.Mem/vm.Cpus < minMemToCpuRatio {
		return false
	}
	return true
}

func (e *Engine) burstFilter(vm VirtualMachine, req ClusterRecommendationReq) bool {
	// if not specified in req or it's allowed the filter passes
	if (req.AllowBurst == nil) || *(req.AllowBurst) {
		return true
	}
	// burst is not allowed
	return !vm.Burst
}

func (e *Engine) minCpuRatioFilter(vm VirtualMachine, req ClusterRecommendationReq) bool {
	minCpuToMemRatio := req.SumCpu / req.SumMem
	if vm.Cpus/vm.Mem < minCpuToMemRatio {
		return false
	}
	return true
}

func (e *Engine) ntwPerformanceFilter(vm VirtualMachine, req ClusterRecommendationReq) bool {
	if req.NetworkPerf == nil { //there is no filter set
		return true
	}
	if vm.NetworkPerfCat == *req.NetworkPerf { //the network performance category matches the vm
		return true
	}
	return false
}

// excludeFilter checks for the vm type in the request' exclude list, the filter  passes if the type is not excluded
func (e *Engine) excludesFilter(vm VirtualMachine, req ClusterRecommendationReq) bool {
	if req.Excludes == nil || len(req.Excludes) == 0 {
		log.Debugf("no blacklist provided - all vm types are welcome")
		return true
	}
	if contains(req.Excludes, vm.Type) {
		log.Debugf("the vm type [%s] is blacklisted", vm.Type)
		return false
	}
	return true
}

// includesFilter checks whether the vm type is in the includes list; the filter passes if the type is in the list
func (e *Engine) includesFilter(vm VirtualMachine, req ClusterRecommendationReq) bool {
	if req.Includes == nil || len(req.Includes) == 0 {
		log.Debugf("no whitelist specified - all vm types are welcome")
		return true
	}
	if contains(req.Includes, vm.Type) {
		log.Debugf("the vm type [%s] is whitelisted", vm.Type)
		return true
	}
	return false
}

// filterSpots selects vm-s that potentially can be part of "spot" node pools
func (e *Engine) filterSpots(vms []VirtualMachine) []VirtualMachine {
	log.Debugf("selecting spot instances for recommending spot pools")
	fvms := make([]VirtualMachine, 0)
	for _, vm := range vms {
		if vm.AvgPrice != 0 {
			fvms = append(fvms, vm)
		}
	}
	return fvms
}

// currentGenFilter removes instance types that are not the current generation (amazon only)
func (e *Engine) currentGenFilter(vm VirtualMachine, req ClusterRecommendationReq) bool {
	if req.AllowOlderGen == nil || !*req.AllowOlderGen {
		// filter by current generation by default (if it's not specified in the request) or it's explicitly set to false
		return vm.CurrentGen
	}
	// the flag it's set into the req AND ist's true
	return true
}
