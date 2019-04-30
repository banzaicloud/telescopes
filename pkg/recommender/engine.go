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
	"sort"
	"strings"

	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/pkg/errors"
)

// Engine represents the recommendation engine, it operates on a map of provider -> VmRegistry
type Engine struct {
	log              logur.Logger
	ciSource         CloudInfoSource
	vmSelector       VmRecommender
	nodePoolSelector NodePoolRecommender
}

// NewEngine creates a new Engine instance
func NewEngine(log logur.Logger, ciSource CloudInfoSource, vmSelector VmRecommender, nodePoolSelector NodePoolRecommender) *Engine {
	return &Engine{
		log:              log,
		ciSource:         ciSource,
		vmSelector:       vmSelector,
		nodePoolSelector: nodePoolSelector,
	}
}

// RecommendCluster performs recommendation based on the provided arguments
func (e *Engine) RecommendCluster(provider string, service string, region string, req SingleClusterRecommendationReq, layoutDesc []NodePoolDesc) (*ClusterRecommendationResp, error) {
	e.log.Info(fmt.Sprintf("recommending cluster configuration. request: [%#v]", req))

	allProducts, err := e.ciSource.GetProductDetails(provider, service, region)
	if err != nil {
		return nil, err
	}

	if req.OnDemandPct != 100 {
		availableSpotPrice := false
		for _, vm := range allProducts {
			if vm.AvgPrice != 0.0 {
				availableSpotPrice = true
				break
			}
		}
		if !availableSpotPrice {
			e.log.Warn("onDemand percentage in the request ignored")
			req.OnDemandPct = 100
		}
	}

	cheapestMaster, err := e.recommendMaster(provider, service, req, allProducts, layoutDesc)
	if err != nil {
		return nil, err
	}

	cheapestNodePoolSet, err := e.getCheapestNodePoolSet(provider, req, layoutDesc, allProducts)
	if err != nil {
		return nil, err
	}
	if cheapestMaster != nil {
		cheapestNodePoolSet = append(cheapestNodePoolSet, *cheapestMaster)
	}

	accuracy := findResponseSum(req.Zone, cheapestNodePoolSet)

	return &ClusterRecommendationResp{
		Provider:  provider,
		Service:   service,
		Region:    region,
		Zone:      req.Zone,
		NodePools: cheapestNodePoolSet,
		Accuracy:  accuracy,
	}, nil
}

func (e *Engine) recommendMaster(provider, service string, req SingleClusterRecommendationReq, allProducts []VirtualMachine, layoutDesc []NodePoolDesc) (*NodePool, error) {
	if layoutDesc != nil {
		e.log.Debug("there is an existing layout, does not require a master recommendation")
		return nil, nil
	}

	switch service {
	case "pke":
		if provider == "amazon" {
			req.Includes = []string{
				"c5.large",
				"c5.xlarge",
				"c5.2xlarge",
				"c5.4xlarge",
				"c5.9xlarge",
				"c4.large",
				"c4.xlarge",
				"c4.2xlarge",
				"c4.4xlarge",
				"c4.8xlarge",
			}
		}

		masterNodePool, err := e.masterNodeRecommendation(provider, req, allProducts)
		if err != nil {
			return nil, err
		}

		return masterNodePool, nil

	case "ack":
		masterNodePool, err := e.masterNodeRecommendation(provider, req, allProducts)
		if err != nil {
			return nil, err
		}
		masterNodePool.SumNodes = 3

		return masterNodePool, nil

	case "eks":
		var masterNodePool *NodePool
		for _, instance := range allProducts {
			if instance.Type == "EKS Control Plane" {
				masterNodePool = &NodePool{
					VmType:   instance,
					SumNodes: 1,
					VmClass:  Regular,
					Role:     Master,
				}
				break
			}
		}
		return masterNodePool, nil
	default:
		e.log.Debug("service does not require master recommendation", map[string]interface{}{"provider": provider, "service": service})
		return nil, nil
	}
}

func (e *Engine) masterNodeRecommendation(provider string, req SingleClusterRecommendationReq, allProducts []VirtualMachine) (*NodePool, error) {
	request := SingleClusterRecommendationReq{
		ClusterRecommendationReq: ClusterRecommendationReq{
			SumCpu:      2,
			SumMem:      4,
			MinNodes:    1,
			MaxNodes:    1,
			OnDemandPct: 100,
		},
		Zone:     req.Zone,
		Includes: req.Includes,
	}

	cheapestMaster, err := e.getCheapestNodePoolSet(provider, request, nil, allProducts)
	if err != nil {
		return nil, err
	}

	master := &NodePool{
		VmType:   cheapestMaster[0].VmType,
		SumNodes: cheapestMaster[0].SumNodes,
		VmClass:  cheapestMaster[0].VmClass,
		Role:     Master,
	}

	return master, nil
}

func (e *Engine) getCheapestNodePoolSet(provider string, req SingleClusterRecommendationReq, layoutDesc []NodePoolDesc, allProducts []VirtualMachine) ([]NodePool, error) {
	desiredCpu := req.SumCpu
	desiredMem := req.SumMem
	desiredOdPct := req.OnDemandPct

	attributes := []string{Cpu, Memory}
	nodePools := make(map[string][]NodePool, 2)

	for _, attr := range attributes {
		vmsInRange, err := e.vmSelector.FindVmsWithAttrValues(attr, req, layoutDesc, allProducts)
		if err != nil {
			return nil, emperror.With(err, RecommenderErrorTag, "vms")
		}

		layout := e.transformLayout(layoutDesc, vmsInRange)
		if layout != nil {
			req.SumCpu, req.SumMem, req.OnDemandPct, err = e.computeScaleoutResources(layout, attr, desiredCpu, desiredMem, desiredOdPct)
			if err != nil {
				e.log.Error(emperror.Wrap(err, "failed to compute scaleout resources").Error())
				continue
			}
			if req.SumCpu < 0 && req.SumMem < 0 {
				return nil, emperror.With(fmt.Errorf("there's already enough resources in the cluster. Total resources available: CPU: %v, Mem: %v", desiredCpu-req.SumCpu, desiredMem-req.SumMem))
			}
		}

		odVms, spotVms, err := e.vmSelector.RecommendVms(provider, vmsInRange, attr, req, layout)
		if err != nil {
			return nil, emperror.Wrap(err, "failed to recommend virtual machines")
		}

		if (len(odVms) == 0 && req.OnDemandPct > 0) || (len(spotVms) == 0 && req.OnDemandPct < 100) {
			e.log.Debug("no vms with the requested resources found", map[string]interface{}{"attribute": attr})
			// skip the nodepool creation, go to the next attr
			continue
		}
		e.log.Debug("recommended vms", map[string]interface{}{"attribute": attr,
			"odVmsCount": len(odVms), "odVmsValues": odVms, "spotVmsCount": len(spotVms), "spotVmsValues": spotVms})

		nps := e.nodePoolSelector.RecommendNodePools(attr, req, layout, odVms, spotVms)

		e.log.Debug(fmt.Sprintf("recommended node pools for [%s]: count:[%d] , values: [%#v]", attr, len(nps), nps))

		nodePools[attr] = nps
	}

	if len(nodePools) == 0 {
		e.log.Debug(fmt.Sprintf("could not recommend node pools for request: %v", req))
		return nil, emperror.With(errors.New("could not recommend cluster with the requested resources"), RecommenderErrorTag)
	}

	return e.findCheapestNodePoolSet(nodePools), nil
}

// RecommendClusterScaleOut performs recommendation for an existing layout's scale out
func (e *Engine) RecommendClusterScaleOut(provider string, service string, region string, req ClusterScaleoutRecommendationReq) (*ClusterRecommendationResp, error) {
	e.log.Info(fmt.Sprintf("recommending cluster configuration. request: [%#v]", req))

	includes := make([]string, len(req.ActualLayout))
	for i, npd := range req.ActualLayout {
		includes[i] = npd.InstanceType
	}

	clReq := SingleClusterRecommendationReq{
		ClusterRecommendationReq: ClusterRecommendationReq{
			AllowBurst:    boolPointer(true),
			AllowOlderGen: boolPointer(true),
			MaxNodes:      math.MaxInt8,
			MinNodes:      1,
			NetworkPerf:   nil,
			OnDemandPct:   req.OnDemandPct,
			SameSize:      false,
			SumCpu:        req.DesiredCpu,
			SumMem:        req.DesiredMem,
			SumGpu:        req.DesiredGpu,
		},
		Includes: includes,
		Excludes: req.Excludes,
		Zone:     req.Zone,
	}

	return e.RecommendCluster(provider, service, region, clReq, req.ActualLayout)
}

// RecommendMultiCluster performs recommendation
func (e *Engine) RecommendMultiCluster(req MultiClusterRecommendationReq) (map[string][]*ClusterRecommendationResp, error) {
	respPerService := make(map[string][]*ClusterRecommendationResp)

	for _, provider := range req.Providers {
		for _, service := range provider.Services {
			regions, err := e.getRegions(provider.Provider, service, req)
			if err != nil {
				return nil, err
			}

			var responses []*ClusterRecommendationResp
			for _, region := range regions {
				singleClusterRecommendationReq := SingleClusterRecommendationReq{
					ClusterRecommendationReq: req.ClusterRecommendationReq,
					Excludes:                 req.Excludes[provider.Provider][service],
					Includes:                 req.Includes[provider.Provider][service],
				}
				if response, err := e.RecommendCluster(provider.Provider, service, region, singleClusterRecommendationReq, nil); err != nil {
					e.log.Warn("could not recommend cluster")
				} else {
					responses = append(responses, response)
				}
			}

			limitedResponses := e.getLimitedResponses(responses, req.RespPerService)
			key := strings.Join([]string{strings.ToLower(provider.Provider), strings.ToUpper(service)}, "")
			respPerService[key] = limitedResponses
		}
	}

	if len(respPerService) == 0 {
		return nil, errors.New("failed to recommend clusters")
	}
	return respPerService, nil
}

func (e *Engine) getRegions(provider, service string, req MultiClusterRecommendationReq) ([]string, error) {
	var regions []string
	continents, err := e.ciSource.GetRegions(provider, service)
	if err != nil {
		return nil, err
	}

	for _, validContinent := range req.Continents {
		for _, continent := range continents {
			if validContinent == continent.Name {
				for _, region := range continent.Regions {
					regions = append(regions, region.ID)
				}
			}
		}
	}
	return regions, nil
}

func (e *Engine) getLimitedResponses(responses []*ClusterRecommendationResp, respPerService int) []*ClusterRecommendationResp {
	sort.Sort(ByPricePerService(responses))
	if len(responses) > respPerService {
		var limit = 0
		for i := range responses {
			if responses[respPerService-1].Accuracy.RecTotalPrice < responses[i].Accuracy.RecTotalPrice {
				limit = i
				break
			}
		}
		if limit != 0 {
			responses = responses[:limit]
		}
	}

	return responses
}

// ByPricePerService type for custom sorting of a slice of response
type ByPricePerService []*ClusterRecommendationResp

func (a ByPricePerService) Len() int      { return len(a) }
func (a ByPricePerService) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByPricePerService) Less(i, j int) bool {
	totalPrice1 := a[i].Accuracy.RecTotalPrice
	totalPrice2 := a[j].Accuracy.RecTotalPrice
	return totalPrice1 < totalPrice2
}

func boolPointer(b bool) *bool {
	return &b
}

func findResponseSum(zone string, nodePoolSet []NodePool) ClusterRecommendationAccuracy {
	var sumCpus float64
	var sumMem float64
	var sumWorkerNodes int
	var sumRegularPrice float64
	var sumRegularNodes int
	var sumSpotPrice float64
	var sumSpotNodes int
	var sumTotalPrice float64
	for _, nodePool := range nodePoolSet {
		sumCpus += nodePool.GetSum(Cpu)
		sumMem += nodePool.GetSum(Memory)
		if nodePool.Role == "worker" {
			sumWorkerNodes += nodePool.SumNodes
		}
		if nodePool.VmClass == Regular {
			sumRegularPrice += nodePool.PoolPrice()
			sumRegularNodes += nodePool.SumNodes
		} else {
			sumSpotPrice += nodePool.PoolPrice()
			sumSpotNodes += nodePool.SumNodes
		}
		sumTotalPrice += nodePool.PoolPrice()
	}

	return ClusterRecommendationAccuracy{
		RecCpu:          sumCpus,
		RecMem:          sumMem,
		RecNodes:        sumWorkerNodes,
		RecZone:         zone,
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
			sumPrice += np.PoolPrice()
			sumCpus += np.GetSum(Cpu)
			sumMem += np.GetSum(Memory)
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

func (e *Engine) transformLayout(layoutDesc []NodePoolDesc, vms []VirtualMachine) []NodePool {
	if layoutDesc == nil {
		return nil
	}
	nps := make([]NodePool, len(layoutDesc))
	for i, npd := range layoutDesc {
		for _, vm := range vms {
			if vm.Type == npd.InstanceType {
				nps[i] = NodePool{
					VmType:   vm,
					VmClass:  npd.GetVmClass(),
					SumNodes: npd.SumNodes,
					Role:     Worker,
				}
				break
			}
		}
	}
	return nps
}

func (e *Engine) computeScaleoutResources(layout []NodePool, attr string, desiredCpu, desiredMem float64, desiredOdPct int) (float64, float64, int, error) {
	var currentCpuTotal, currentMemTotal, sumCurrentOdCpu, sumCurrentOdMem float64
	var scaleoutOdPct int
	for _, np := range layout {
		if np.VmClass == Regular {
			sumCurrentOdCpu += float64(np.SumNodes) * np.VmType.Cpus
			sumCurrentOdMem += float64(np.SumNodes) * np.VmType.Mem
		}
		currentCpuTotal += float64(np.SumNodes) * np.VmType.Cpus
		currentMemTotal += float64(np.SumNodes) * np.VmType.Mem
	}

	scaleoutCpu := desiredCpu - currentCpuTotal
	scaleoutMem := desiredMem - currentMemTotal

	if scaleoutCpu < 0 && scaleoutMem < 0 {
		return scaleoutCpu, scaleoutMem, 0, nil
	}

	e.log.Debug(fmt.Sprintf("desiredCpu: %v, desiredMem: %v, currentCpuTotal/currentCpuOnDemand: %v/%v, currentMemTotal/currentMemOnDemand: %v/%v", desiredCpu, desiredMem, currentCpuTotal, sumCurrentOdCpu, currentMemTotal, sumCurrentOdMem))
	e.log.Debug(fmt.Sprintf("total scaleout cpu/mem needed: %v/%v", scaleoutCpu, scaleoutMem))
	e.log.Debug(fmt.Sprintf("desired on-demand percentage: %v", desiredOdPct))

	switch attr {
	case Cpu:
		if scaleoutCpu < 0 {
			return 0, 0, 0, errors.New("there's already enough CPU resources in the cluster")
		}
		desiredOdCpu := desiredCpu * float64(desiredOdPct) / 100
		scaleoutOdCpu := desiredOdCpu - sumCurrentOdCpu
		scaleoutOdPct = int(scaleoutOdCpu / scaleoutCpu * 100)
		e.log.Debug(fmt.Sprintf("desired on-demand cpu: %v, cpu to add with the scaleout: %v", desiredOdCpu, scaleoutOdCpu))
	case Memory:
		if scaleoutMem < 0 {
			return 0, 0, 0, emperror.With(errors.New("there's already enough memory resources in the cluster"))
		}
		desiredOdMem := desiredMem * float64(desiredOdPct) / 100
		scaleoutOdMem := desiredOdMem - sumCurrentOdMem
		e.log.Debug(fmt.Sprintf("desired on-demand memory: %v, memory to add with the scaleout: %v", desiredOdMem, scaleoutOdMem))
		scaleoutOdPct = int(scaleoutOdMem / scaleoutMem * 100)
	}
	if scaleoutOdPct > 100 {
		// even if we add only on-demand instances, we still we can't reach the minimum ratio
		return 0, 0, 0, emperror.With(errors.New("couldn't scale out cluster with the provided parameters"), "onDemandPct", desiredOdPct)
	} else if scaleoutOdPct < 0 {
		// means that we already have enough resources in the cluster to keep the minimum ratio
		scaleoutOdPct = 0
	}
	e.log.Debug(fmt.Sprintf("percentage of on-demand resources in the scaleout: %v", scaleoutOdPct))
	return scaleoutCpu, scaleoutMem, scaleoutOdPct, nil
}
