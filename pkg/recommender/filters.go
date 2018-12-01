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

	"github.com/banzaicloud/cloudinfo/pkg/logger"
)

type vmFilter func(ctx context.Context, vm VirtualMachine, req ClusterRecommendationReq) bool

func (e *Engine) minMemRatioFilter(ctx context.Context, vm VirtualMachine, req ClusterRecommendationReq) bool {
	minMemToCpuRatio := req.SumMem / req.SumCpu
	return minMemToCpuRatio <= vm.Mem/vm.Cpus
}

func (e *Engine) burstFilter(ctx context.Context, vm VirtualMachine, req ClusterRecommendationReq) bool {
	// if not specified in req or it's allowed the filter passes
	if (req.AllowBurst == nil) || *(req.AllowBurst) {
		return true
	}
	// burst is not allowed
	return !vm.Burst
}

func (e *Engine) minCpuRatioFilter(ctx context.Context, vm VirtualMachine, req ClusterRecommendationReq) bool {
	minCpuToMemRatio := req.SumCpu / req.SumMem
	return minCpuToMemRatio <= vm.Cpus/vm.Mem
}

func (e *Engine) ntwPerformanceFilter(ctx context.Context, vm VirtualMachine, req ClusterRecommendationReq) bool {
	if req.NetworkPerf == nil { //there is no filter set
		return true
	}
	if vm.NetworkPerfCat == *req.NetworkPerf { //the network performance category matches the vm
		return true
	}
	return false
}

// excludeFilter checks for the vm type in the request' exclude list, the filter  passes if the type is not excluded
func (e *Engine) excludesFilter(ctx context.Context, vm VirtualMachine, req ClusterRecommendationReq) bool {
	ctxLog := logger.Extract(ctx)
	if req.Excludes == nil || len(req.Excludes) == 0 {
		ctxLog.Debug("no blacklist provided - all vm types are welcome")
		return true
	}
	if contains(req.Excludes, vm.Type) {
		ctxLog.Debugf("the vm type [%s] is blacklisted", vm.Type)
		return false
	}
	return true
}

// includesFilter checks whether the vm type is in the includes list; the filter passes if the type is in the list
func (e *Engine) includesFilter(ctx context.Context, vm VirtualMachine, req ClusterRecommendationReq) bool {
	ctxLog := logger.Extract(ctx)
	if req.Includes == nil || len(req.Includes) == 0 {
		ctxLog.Debug("no whitelist specified - all vm types are welcome")
		return true
	}
	if contains(req.Includes, vm.Type) {
		ctxLog.Debugf("the vm type [%s] is whitelisted", vm.Type)
		return true
	}
	return false
}

// filterSpots selects vm-s that potentially can be part of "spot" node pools
func (e *Engine) filterSpots(ctx context.Context, vms []VirtualMachine) []VirtualMachine {
	logger.Extract(ctx).Debug("selecting spot instances for recommending spot pools")
	fvms := make([]VirtualMachine, 0)
	for _, vm := range vms {
		if vm.AvgPrice != 0 {
			fvms = append(fvms, vm)
		}
	}
	return fvms
}

// currentGenFilter removes instance types that are not the current generation (amazon only)
func (e *Engine) currentGenFilter(ctx context.Context, vm VirtualMachine, req ClusterRecommendationReq) bool {
	if req.AllowOlderGen == nil || !*req.AllowOlderGen {
		// filter by current generation by default (if it's not specified in the request) or it's explicitly set to false
		return vm.CurrentGen
	}
	// the flag it's set into the req AND it's true
	return true
}
