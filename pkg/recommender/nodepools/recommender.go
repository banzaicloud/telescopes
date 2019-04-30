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

package nodepools

import (
	"fmt"
	"math"
	"sort"

	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/goph/logur"
)

type nodePoolSelector struct {
	log logur.Logger
}

func NewNodePoolSelector(log logur.Logger) *nodePoolSelector {
	return &nodePoolSelector{
		log: log,
	}
}

// RecommendNodePools finds the slice of NodePools that may participate in the recommendation process
func (s *nodePoolSelector) RecommendNodePools(attr string, req recommender.SingleClusterRecommendationReq, layout []recommender.NodePool, odVms []recommender.VirtualMachine, spotVms []recommender.VirtualMachine) []recommender.NodePool {
	s.log.Debug(fmt.Sprintf("requested sum for attribute [%s]: [%f]", attr, sum(req, attr)))
	var sumOnDemandValue = sum(req, attr) * float64(req.OnDemandPct) / 100
	s.log.Debug(fmt.Sprintf("on demand sum value for attr [%s]: [%f]", attr, sumOnDemandValue))

	// recommend on-demands
	odNps := make([]recommender.NodePool, 0)

	//TODO: validate if there's no on-demand in layout but we want to add ondemands
	for _, np := range layout {
		if np.VmClass == recommender.Regular {
			odNps = append(odNps, np)
		}
	}
	var actualOnDemandResources float64
	var odNodesToAdd int
	if len(odVms) > 0 && req.OnDemandPct != 0 {
		// find cheapest onDemand instance from the list - based on price per attribute
		selectedOnDemand := odVms[0]
		for _, vm := range odVms {
			if vm.OnDemandPrice/vm.GetAttrValue(attr) < selectedOnDemand.OnDemandPrice/selectedOnDemand.GetAttrValue(attr) {
				selectedOnDemand = vm
			}
		}
		odNodesToAdd = int(math.Ceil(sumOnDemandValue / selectedOnDemand.GetAttrValue(attr)))
		if layout == nil {
			odNps = append(odNps, recommender.NodePool{
				SumNodes: odNodesToAdd,
				VmClass:  recommender.Regular,
				VmType:   selectedOnDemand,
				Role:     recommender.Worker,
			})
		} else {
			for i, np := range odNps {
				if np.VmType.Type == selectedOnDemand.Type {
					odNps[i].SumNodes += odNodesToAdd
				}
			}
		}
		actualOnDemandResources = selectedOnDemand.GetAttrValue(attr) * float64(odNodesToAdd)
	}

	spotNps := make([]recommender.NodePool, 0)

	if req.OnDemandPct != 100 {
		// recalculate required spot resources by taking actual on-demand resources into account
		var sumSpotValue = sum(req, attr) - actualOnDemandResources
		s.log.Debug(fmt.Sprintf("spot sum value for attr [%s]: [%f]", attr, sumSpotValue))

		// recommend spot pools
		excludedSpotNps := make([]recommender.NodePool, 0)

		s.sortByAttrValue(attr, spotVms)

		var N int
		if layout == nil {
			// the "magic" number of machines for diversifying the types
			N = int(math.Min(float64(findN(avgSpotNodeCount(req.MinNodes, req.MaxNodes, odNodesToAdd))), float64(len(spotVms))))
			// the second "magic" number for diversifying the layout
			M := findM(N, spotVms)
			s.log.Debug(fmt.Sprintf("Magic 'Marton' numbers: N=%d, M=%d", N, M))

			// the first M vm-s
			recommendedVms := spotVms[:M]

			// create spot nodepools - one for the first M vm-s
			for _, vm := range recommendedVms {
				spotNps = append(spotNps, recommender.NodePool{
					SumNodes: 0,
					VmClass:  recommender.Spot,
					VmType:   vm,
					Role:     recommender.Worker,
				})
			}
		} else {
			sort.Sort(ByNonZeroNodePools(layout))
			var nonZeroNPs int
			for _, np := range layout {
				if np.VmClass == recommender.Spot {
					if np.SumNodes > 0 {
						nonZeroNPs += 1
					}
					included := false
					for _, vm := range spotVms {
						if np.VmType.Type == vm.Type {
							spotNps = append(spotNps, np)
							included = true
							break
						}
					}
					if !included {
						excludedSpotNps = append(excludedSpotNps, np)
					}
				}
			}
			N = findNWithLayout(nonZeroNPs, len(spotVms))
			s.log.Debug(fmt.Sprintf("Magic 'Marton' number: N=%d", N))
		}
		spotNps = s.fillSpotNodePools(sumSpotValue, N, spotNps, attr)
		if len(excludedSpotNps) > 0 {
			spotNps = append(spotNps, excludedSpotNps...)
		}
	}

	s.log.Debug(fmt.Sprintf("created [%d] regular and [%d] spot price node pools", len(odNps), len(spotNps)))

	return append(odNps, spotNps...)
}

// sortByAttrValue returns the slice for
func (s *nodePoolSelector) sortByAttrValue(attr string, vms []recommender.VirtualMachine) {
	// sort and cut
	switch attr {
	case recommender.Memory:
		sort.Sort(ByAvgPricePerMemory(vms))
	case recommender.Cpu:
		sort.Sort(ByAvgPricePerCpu(vms))
	default:
		s.log.Error("unsupported attribute", map[string]interface{}{"attribute": attr})
	}
}

// ByAvgPricePerCpu type for custom sorting of a slice of vms
type ByAvgPricePerCpu []recommender.VirtualMachine

func (a ByAvgPricePerCpu) Len() int      { return len(a) }
func (a ByAvgPricePerCpu) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByAvgPricePerCpu) Less(i, j int) bool {
	pricePerCpu1 := a[i].AvgPrice / a[i].Cpus
	pricePerCpu2 := a[j].AvgPrice / a[j].Cpus
	return pricePerCpu1 < pricePerCpu2
}

// ByAvgPricePerMemory type for custom sorting of a slice of vms
type ByAvgPricePerMemory []recommender.VirtualMachine

func (a ByAvgPricePerMemory) Len() int      { return len(a) }
func (a ByAvgPricePerMemory) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByAvgPricePerMemory) Less(i, j int) bool {
	pricePerMem1 := a[i].AvgPrice / a[i].Mem
	pricePerMem2 := a[j].AvgPrice / a[j].Mem
	return pricePerMem1 < pricePerMem2
}

type ByNonZeroNodePools []recommender.NodePool

func (a ByNonZeroNodePools) Len() int      { return len(a) }
func (a ByNonZeroNodePools) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByNonZeroNodePools) Less(i, j int) bool {
	return a[i].SumNodes > a[j].SumNodes
}

// gets the requested sum for the attribute value
func sum(req recommender.SingleClusterRecommendationReq, attr string) float64 {
	switch attr {
	case recommender.Cpu:
		return req.SumCpu
	case recommender.Memory:
		return req.SumMem
	default:
		return 0
	}
}

func findNWithLayout(nonZeroNps, vmOptions int) int {
	// vmOptions cannot be 0 because validation would fail sooner
	if nonZeroNps == 0 {
		return 1
	}
	if nonZeroNps < vmOptions {
		return nonZeroNps
	} else {
		return vmOptions
	}
}

func (s *nodePoolSelector) fillSpotNodePools(sumSpotValue float64, N int, nps []recommender.NodePool, attr string) []recommender.NodePool {
	var (
		sumValueInPools, minValue float64
		idx, minIndex             int
	)
	for i := 0; i < N; i++ {
		v := float64(nps[i].SumNodes) * nps[i].VmType.GetAttrValue(attr)
		sumValueInPools += v
		if i == 0 {
			minValue = v
			minIndex = i
		} else if v < minValue {
			minValue = v
			minIndex = i
		}
	}
	desiredSpotValue := sumValueInPools + sumSpotValue
	idx = minIndex
	for sumValueInPools < desiredSpotValue {
		nodePoolIdx := idx % N
		if nodePoolIdx == minIndex {
			// always add a new instance to the option with the lowest attribute value to balance attributes and move on
			nps[nodePoolIdx].SumNodes += 1
			sumValueInPools += nps[nodePoolIdx].VmType.GetAttrValue(attr)
			s.log.Debug(fmt.Sprintf("adding vm to the [%d]th (min sized) node pool, sum value in pools: [%f]", nodePoolIdx, sumValueInPools))
			idx++
		} else if getNextSum(nps[nodePoolIdx], attr) > nps[minIndex].GetSum(attr) {
			// for other pools, if adding another vm would exceed the current sum of the cheapest option, move on to the next one
			s.log.Debug(fmt.Sprintf("skip adding vm to the [%d]th node pool", nodePoolIdx))
			idx++
		} else {
			// otherwise add a new one, but do not move on to the next one
			nps[nodePoolIdx].SumNodes += 1
			sumValueInPools += nps[nodePoolIdx].VmType.GetAttrValue(attr)
			s.log.Debug(fmt.Sprintf("adding vm to the [%d]th node pool, sum value in pools: [%f]", nodePoolIdx, sumValueInPools))
		}
	}
	return nps
}

// findN returns the number of nodes required
func findN(avg int) int {
	var N int
	switch {
	case avg <= 4:
		N = avg
	case avg <= 8:
		N = 4
	case avg <= 15:
		N = 5
	case avg <= 24:
		N = 6
	case avg <= 35:
		N = 7
	case avg > 35:
		N = 8
	}
	return N
}

func findM(N int, spotVms []recommender.VirtualMachine) int {
	if N > 0 {
		return int(math.Min(math.Ceil(float64(N)*1.5), float64(len(spotVms))))
	} else {
		return int(math.Min(3, float64(len(spotVms))))
	}
}

func avgSpotNodeCount(minNodes, maxNodes, odNodes int) int {
	count := float64(minNodes-odNodes+maxNodes-odNodes) / 2
	spotCount := int(math.Ceil(count))
	if spotCount < 0 {
		return 0
	}
	return spotCount
}

// getNextSum gets the total value if the pool was increased by one
func getNextSum(n recommender.NodePool, attr string) float64 {
	return n.GetSum(attr) + n.VmType.GetAttrValue(attr)
}
