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
	"fmt"
	"math"

	"github.com/goph/logur"
)

// Engine represents the recommendation engine, it operates on a map of provider -> VmRegistry
type Engine struct {
	np  NodePoolRecommender
	log logur.Logger
}

// NewEngine creates a new Engine instance
func NewEngine(nodePools NodePoolRecommender, log logur.Logger) *Engine {
	return &Engine{
		np:  nodePools,
		log: log,
	}
}

// RecommendCluster performs recommendation based on the provided arguments
func (e *Engine) RecommendCluster(provider string, service string, region string, req ClusterRecommendationReq, layoutDesc []NodePoolDesc, log logur.Logger) (*ClusterRecommendationResp, error) {
	e.log = log

	e.log.Info(fmt.Sprintf("recommending cluster configuration. request: [%#v]", req))

	nodePools, err := e.np.RecommendNodePools(provider, service, region, req, log, layoutDesc)
	if err != nil {
		return nil, err
	}

	cheapestNodePoolSet := e.findCheapestNodePoolSet(nodePools)

	accuracy := findResponseSum(req.Zones, cheapestNodePoolSet)

	return &ClusterRecommendationResp{
		Provider:  provider,
		Service:   service,
		Region:    region,
		Zones:     req.Zones,
		NodePools: cheapestNodePoolSet,
		Accuracy:  accuracy,
	}, nil
}

// RecommendClusterScaleOut performs recommendation for an existing layout's scale out
func (e *Engine) RecommendClusterScaleOut(provider string, service string, region string, req ClusterScaleoutRecommendationReq, log logur.Logger) (*ClusterRecommendationResp, error) {
	log.Info(fmt.Sprintf("recommending cluster configuration. request: [%#v]", req))

	includes := make([]string, len(req.ActualLayout))
	for i, npd := range req.ActualLayout {
		includes[i] = npd.InstanceType
	}

	clReq := ClusterRecommendationReq{
		Zones:         req.Zones,
		AllowBurst:    boolPointer(true),
		Includes:      includes,
		Excludes:      req.Excludes,
		AllowOlderGen: boolPointer(true),
		MaxNodes:      math.MaxInt8,
		MinNodes:      1,
		NetworkPerf:   nil,
		OnDemandPct:   req.OnDemandPct,
		SameSize:      false,
		SumCpu:        req.DesiredCpu,
		SumMem:        req.DesiredMem,
		SumGpu:        req.DesiredGpu,
	}

	return e.RecommendCluster(provider, service, region, clReq, req.ActualLayout, log)
}

func boolPointer(b bool) *bool {
	return &b
}

func findResponseSum(zones []string, nodePoolSet []NodePool) ClusterRecommendationAccuracy {
	var sumCpus float64
	var sumMem float64
	var sumNodes int
	var sumRegularPrice float64
	var sumRegularNodes int
	var sumSpotPrice float64
	var sumSpotNodes int
	var sumTotalPrice float64
	for _, nodePool := range nodePoolSet {
		sumCpus += GetSum(nodePool, Cpu)
		sumMem += GetSum(nodePool, Memory)
		sumNodes += nodePool.SumNodes
		if nodePool.VmClass == Regular {
			sumRegularPrice += nodePool.poolPrice()
			sumRegularNodes += nodePool.SumNodes
		} else {
			sumSpotPrice += nodePool.poolPrice()
			sumSpotNodes += nodePool.SumNodes
		}
		sumTotalPrice += nodePool.poolPrice()
	}

	return ClusterRecommendationAccuracy{
		RecCpu:          sumCpus,
		RecMem:          sumMem,
		RecNodes:        sumNodes,
		RecZone:         zones,
		RecRegularPrice: sumRegularPrice,
		RecRegularNodes: sumRegularNodes,
		RecSpotPrice:    sumSpotPrice,
		RecSpotNodes:    sumSpotNodes,
		RecTotalPrice:   sumTotalPrice,
	}
}

// findCheapestNodePoolSet looks up the "cheapest" node pool set from the provided map
func (e *Engine) findCheapestNodePoolSet(nodePoolSets map[string][]NodePool) []NodePool {
	e.log.Info("finding cheapest pool set...")
	var cheapestNpSet []NodePool
	var bestPrice float64

	for attr, nodePools := range nodePoolSets {
		var sumPrice float64
		var sumCpus float64
		var sumMem float64

		for _, np := range nodePools {
			sumPrice += np.poolPrice()
			sumCpus += GetSum(np, Cpu)
			sumMem += GetSum(np, Memory)
		}
		e.log.Debug("checking node pool",
			map[string]interface{}{"attribute": attr, "cpu": sumCpus, "memory": sumMem, "price": sumPrice})

		if bestPrice == 0 || bestPrice > sumPrice {
			e.log.Debug("cheaper node pool set is found", map[string]interface{}{"price": sumPrice})
			bestPrice = sumPrice
			cheapestNpSet = nodePools
		}
	}
	return cheapestNpSet
}

// GetSum gets the total value for the given attribute per pool
func GetSum(n NodePool, attr string) float64 {
	return float64(n.SumNodes) * GetAttrValue(n.VmType, attr)
}

func GetAttrValue(v VirtualMachine, attr string) float64 {
	switch attr {
	case Cpu:
		return v.Cpus
	case Memory:
		return v.Mem
	default:
		return 0
	}
}

// poolPrice calculates the price of the pool
func (n *NodePool) poolPrice() float64 {
	var sum = float64(0)
	switch n.VmClass {
	case Regular:
		sum = float64(n.SumNodes) * n.VmType.OnDemandPrice
	case Spot:
		sum = float64(n.SumNodes) * n.VmType.AvgPrice
	}
	return sum
}
