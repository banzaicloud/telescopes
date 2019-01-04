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
	"math"
	"sort"

	"github.com/goph/emperror"
	"github.com/pkg/errors"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo-client/models"
	"github.com/banzaicloud/cloudinfo/pkg/logger"
)

const (
	// vm types - regular and ondemand means the same, they are both accepted on the API
	regular  = "regular"
	ondemand = "ondemand"
	spot     = "spot"
	// Memory represents the memory attribute for the recommender
	Memory = "memory"
	// Cpu represents the cpu attribute for the recommender
	Cpu = "cpu"

	recommenderErrorTag = "recommender"
)

// Engine represents the recommendation engine, it operates on a map of provider -> VmRegistry
type Engine struct {
	ciSource CloudInfoSource
}

// NewEngine creates a new Engine instance
func NewEngine(cis CloudInfoSource) *Engine {
	return &Engine{
		ciSource: cis,
	}
}

// ClusterRecommendationReq encapsulates the recommendation input data
// swagger:parameters recommendClusterSetup
type ClusterRecommendationReq struct {
	// Total number of CPUs requested for the cluster
	SumCpu float64 `json:"sumCpu" binding:"min=1"`
	// Total memory requested for the cluster (GB)
	SumMem float64 `json:"sumMem" binding:"min=1"`
	// Minimum number of nodes in the recommended cluster
	MinNodes int `json:"minNodes,omitempty" binding:"min=1,ltefield=MaxNodes"`
	// Maximum number of nodes in the recommended cluster
	MaxNodes int `json:"maxNodes,omitempty"`
	// If true, recommended instance types will have a similar size
	SameSize bool `json:"sameSize,omitempty"`
	// Percentage of regular (on-demand) nodes in the recommended cluster
	OnDemandPct int `json:"onDemandPct,omitempty" binding:"min=0,max=100"`
	// Availability zones that the cluster should expand to
	Zones []string `json:"zones,omitempty" binding:"dive,zone"`
	// Total number of GPUs requested for the cluster
	SumGpu int `json:"sumGpu,omitempty"`
	// Are burst instances allowed in recommendation
	AllowBurst *bool `json:"allowBurst,omitempty"`
	// NetworkPerf specifies the network performance category
	NetworkPerf *string `json:"networkPerf" binding:"omitempty,network"`
	// Excludes is a blacklist - a slice with vm types to be excluded from the recommendation
	Excludes []string `json:"excludes,omitempty"`
	// Includes is a whitelist - a slice with vm types to be contained in the recommendation
	Includes []string `json:"includes,omitempty"`
	// AllowOlderGen allow older generations of virtual machines (applies for EC2 only)
	AllowOlderGen *bool `json:"allowOlderGen,omitempty"`
}

// ClusterRecommendationReq encapsulates the recommendation input data
// swagger:parameters recommendClusterScaleOut
type ClusterScaleoutRecommendationReq struct {
	// Total desired number of CPUs in the cluster after the scale out
	DesiredCpu float64 `json:"desiredCpu" binding:"min=1"`
	// Total desired memory (GB) in the cluster after the scale out
	DesiredMem float64 `json:"desiredCpu" binding:"min=1"`
	// Total desired number of GPUs in the cluster after the scale out
	DesiredGpu int `json:"desiredCpu" binding:"min=1"`
	// Percentage of regular (on-demand) nodes among the scale out nodes
	OnDemandPct int `json:"onDemandPct,omitempty" binding:"min=0,max=100"`
	// Availability zones to be included in the recommendation
	Zones []string `json:"zones,omitempty" binding:"dive,zone"`
	// Excludes is a blacklist - a slice with vm types to be excluded from the recommendation
	Excludes []string `json:"excludes,omitempty"`
	// Description of the current cluster layout
	ActualLayout []NodePoolDesc `json:"actualLayout" binding:"required"`
}

type NodePoolDesc struct {
	// Instance type of VMs in the node pool
	InstanceType string `json:"instanceType" binding:"required"`
	// Signals that the node pool consists of regular or spot/preemptible instance types
	VmClass string `json:"vmClass" binding:"required"`
	// Number of VMs in the node pool
	SumNodes int `json:"sumNodes" binding:"required"`
	// TODO: AZ?
	// Zones []string `json:"zones,omitempty" binding:"dive,zone"`
}

func (n NodePoolDesc) getVmClass() string {
	switch n.VmClass {
	case regular, spot:
		return n.VmClass
	case ondemand:
		return regular
	default:
		return spot
	}
}

// ClusterRecommendationResp encapsulates recommendation result data
// swagger:model RecommendationResponse
type ClusterRecommendationResp struct {
	// The cloud provider
	Provider string `json:"provider"`
	// Availability zones in the recommendation - a multi-zone recommendation means that all node pools should expand to all zones
	Zones []string `json:"zones,omitempty"`
	// Recommended node pools
	NodePools []NodePool `json:"nodePools"`
	// Accuracy of the recommendation
	Accuracy ClusterRecommendationAccuracy `json:"accuracy"`
}

// NodePool represents a set of instances with a specific vm type
type NodePool struct {
	// Recommended virtual machine type
	VmType VirtualMachine `json:"vm"`
	// Recommended number of nodes in the node pool
	SumNodes int `json:"sumNodes"`
	// Specifies if the recommended node pool consists of regular or spot/preemptible instance types
	VmClass string `json:"vmClass"`
}

// ClusterRecommendationAccuracy encapsulates recommendation accuracy
type ClusterRecommendationAccuracy struct {
	// The summarised amount of memory in the recommended cluster
	RecMem float64 `json:"memory"`
	// Number of recommended cpus
	RecCpu float64 `json:"cpu"`
	// Number of recommended nodes
	RecNodes int `json:"nodes"`
	// Availability zones in the recommendation
	RecZone []string `json:"zone,omitempty"`
	// Amount of regular instance type prices in the recommended cluster
	RecRegularPrice float64 `json:"regularPrice"`
	// Number of regular instance type in the recommended cluster
	RecRegularNodes int `json:"regularNodes"`
	// Amount of spot instance type prices in the recommended cluster
	RecSpotPrice float64 `json:"spotPrice"`
	// Number of spot instance type in the recommended cluster
	RecSpotNodes int `json:"spotNodes"`
	// Total price in the recommended cluster
	RecTotalPrice float64 `json:"totalPrice"`
}

// VirtualMachine describes an instance type
type VirtualMachine struct {
	// Instance type
	Type string `json:"type"`
	// Average price of the instance (differs from on demand price in case of spot or preemptible instances)
	AvgPrice float64 `json:"avgPrice"`
	// Regular price of the instance type
	OnDemandPrice float64 `json:"onDemandPrice"`
	// Number of CPUs in the instance type
	Cpus float64 `json:"cpusPerVm"`
	// Available memory in the instance type (GB)
	Mem float64 `json:"memPerVm"`
	// Number of GPUs in the instance type
	Gpus float64 `json:"gpusPerVm"`
	// Burst signals a burst type instance
	Burst bool `json:"burst"`
	// NetworkPerf holds the network performance
	NetworkPerf string `json:"networkPerf"`
	// NetworkPerfCat holds the network performance category
	NetworkPerfCat string `json:"networkPerfCategory"`
	// CurrentGen the vm is of current generation
	CurrentGen bool `json:"currentGen"`
}

func (v *VirtualMachine) getAttrValue(attr string) float64 {
	switch attr {
	case Cpu:
		return v.Cpus
	case Memory:
		return v.Mem
	default:
		return 0
	}
}

// ByAvgPricePerCpu type for custom sorting of a slice of vms
type ByAvgPricePerCpu []VirtualMachine

func (a ByAvgPricePerCpu) Len() int      { return len(a) }
func (a ByAvgPricePerCpu) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByAvgPricePerCpu) Less(i, j int) bool {
	pricePerCpu1 := a[i].AvgPrice / a[i].Cpus
	pricePerCpu2 := a[j].AvgPrice / a[j].Cpus
	return pricePerCpu1 < pricePerCpu2
}

// ByAvgPricePerMemory type for custom sorting of a slice of vms
type ByAvgPricePerMemory []VirtualMachine

func (a ByAvgPricePerMemory) Len() int      { return len(a) }
func (a ByAvgPricePerMemory) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByAvgPricePerMemory) Less(i, j int) bool {
	pricePerMem1 := a[i].AvgPrice / a[i].Mem
	pricePerMem2 := a[j].AvgPrice / a[j].Mem
	return pricePerMem1 < pricePerMem2
}

type ByNonZeroNodePools []NodePool

func (a ByNonZeroNodePools) Len() int      { return len(a) }
func (a ByNonZeroNodePools) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByNonZeroNodePools) Less(i, j int) bool {
	return a[i].SumNodes > a[j].SumNodes
}

// RecommendCluster performs recommendation based on the provided arguments
func (e *Engine) RecommendCluster(ctx context.Context, provider string, service string, region string, req ClusterRecommendationReq, layoutDesc []NodePoolDesc) (*ClusterRecommendationResp, error) {
	log := logger.Extract(ctx)
	log.Infof("recommending cluster configuration. request: [%#v]", req)

	attributes := []string{Cpu, Memory}
	nodePools := make(map[string][]NodePool, 2)

	for _, attr := range attributes {
		var (
			values []float64
			err    error
		)
		if layoutDesc == nil {
			values, err = e.RecommendAttrValues(ctx, provider, service, region, attr, req)
			if err != nil {
				return nil, emperror.Wrap(err, "failed to recommend attribute values")
			}
			log.Debugf("recommended values for [%s]: count:[%d] , values: [%#v./te]", attr, len(values), values)
		}

		vmsInRange, err := e.findVmsWithAttrValues(ctx, provider, service, region, req.Zones, attr, values)
		if err != nil {
			return nil, emperror.With(err, recommenderErrorTag, "vms")
		}

		layout := e.transformLayout(layoutDesc, vmsInRange)

		var sumCurrentCpu, sumCurrentMem float64
		for _, np := range layout {
			sumCurrentCpu += float64(np.SumNodes) * np.VmType.Cpus
			sumCurrentMem += float64(np.SumNodes) * np.VmType.Mem
		}
		req.SumCpu = req.SumCpu - sumCurrentCpu
		req.SumMem = req.SumMem - sumCurrentMem

		vmFilters, _ := e.filtersForAttr(ctx, attr, provider)
		if err != nil {
			return nil, emperror.Wrap(err, "failed to identify filters")
		}
		odVms, spotVms, err := e.RecommendVms(ctx, vmsInRange, attr, vmFilters, req, layout)
		if err != nil {
			return nil, emperror.Wrap(err, "failed to recommend virtual machines")
		}

		if (len(odVms) == 0 && req.OnDemandPct > 0) || (len(spotVms) == 0 && req.OnDemandPct < 100) {
			log.Debugf("no vms with the requested resources found. attribute: %s", attr)
			// skip the nodepool creation, go to the next attr
			continue
		}
		log.Debugf("recommended on-demand vms for [%s]: count:[%d] , values: [%#v]", attr, len(odVms), odVms)
		log.Debugf("recommended spot vms for [%s]: count:[%d] , values: [%#v]", attr, len(spotVms), spotVms)

		//todo add request validation for interdependent request fields, eg: onDemandPct is always 100 when spot
		// instances are not available for provider
		if provider == "oracle" {
			log.Warn("onDemand percentage in the request ignored for provider ", provider)
			req.OnDemandPct = 100
		}
		nps, err := e.RecommendNodePools(ctx, attr, odVms, spotVms, req, layout)
		if err != nil {
			return nil, emperror.Wrap(err, "failed to recommend nodepools")
		}
		log.Debugf("recommended node pools for [%s]: count:[%d] , values: [%#v]", attr, len(nps), nps)

		nodePools[attr] = nps
	}

	if len(nodePools) == 0 {
		log.Debugf("could not recommend node pools for request: %v", req)
		return nil, emperror.With(errors.New("could not recommend cluster with the requested resources"), recommenderErrorTag)
	}

	cheapestNodePoolSet := e.findCheapestNodePoolSet(ctx, nodePools)

	accuracy := findResponseSum(req.Zones, cheapestNodePoolSet)

	return &ClusterRecommendationResp{
		Provider:  provider,
		Zones:     req.Zones,
		NodePools: cheapestNodePoolSet,
		Accuracy:  accuracy,
	}, nil
}

func boolPointer(b bool) *bool {
	return &b
}

// RecommendClusterScaleOut performs recommendation for an existing layout's scale out
func (e *Engine) RecommendClusterScaleOut(ctx context.Context, provider string, service string, region string, req ClusterScaleoutRecommendationReq) (*ClusterRecommendationResp, error) {
	log := logger.Extract(ctx)
	log.Infof("recommending cluster configuration. request: [%#v]", req)

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

	return e.RecommendCluster(ctx, provider, service, region, clReq, req.ActualLayout)
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
		sumCpus += nodePool.getSum(Cpu)
		sumMem += nodePool.getSum(Memory)
		sumNodes += nodePool.SumNodes
		if nodePool.VmClass == regular {
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
func (e *Engine) findCheapestNodePoolSet(ctx context.Context, nodePoolSets map[string][]NodePool) []NodePool {
	log := logger.Extract(ctx)
	log.Info("finding  cheapest pool set...")
	var cheapestNpSet []NodePool
	var bestPrice float64

	for attr, nodePools := range nodePoolSets {
		log.Debugf("checking node pool for attr: [%s]", attr)
		var sumPrice float64
		var sumCpus float64
		var sumMem float64

		for _, np := range nodePools {
			sumPrice += np.poolPrice()
			sumCpus += np.getSum(Cpu)
			sumMem += np.getSum(Memory)
		}
		log.Debugf("sum cpus [%s]: %v", attr, sumCpus)
		log.Debugf("sum mem [%s]: %v", attr, sumMem)
		log.Debugf("sum price [%s]: %v", attr, sumPrice)

		if bestPrice == 0 || bestPrice > sumPrice {
			log.Debugf("cheaper nodepoolset is found. price: [%f]", sumPrice)
			bestPrice = sumPrice
			cheapestNpSet = nodePools
		}
	}
	return cheapestNpSet
}

func avgNodeCount(minNodes, maxNodes int) int {
	return (minNodes + maxNodes) / 2
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

// RecommendVms selects a slice of VirtualMachines for the given attribute and requirements in the request
func (e *Engine) RecommendVms(ctx context.Context, vms []VirtualMachine, attr string, filters []vmFilter, req ClusterRecommendationReq, layout []NodePool) ([]VirtualMachine, []VirtualMachine, error) {
	log := logger.Extract(ctx)
	log.Infof("recommending virtual machines for attribute: [%s]", attr)

	var filteredVms []VirtualMachine
	for _, vm := range vms {
		if e.filtersApply(ctx, vm, filters, req) {
			filteredVms = append(filteredVms, vm)
		}
	}

	if len(filteredVms) == 0 {
		log.Debugf("no vms found for attribute: %s", attr)
		return []VirtualMachine{}, []VirtualMachine{}, nil
	}

	var odVms, spotVms []VirtualMachine
	if layout == nil {
		odVms, spotVms = filteredVms, filteredVms
	} else {
		for _, np := range layout {
			for _, vm := range filteredVms {
				if np.VmType.Type == vm.Type {
					if np.VmClass == regular {
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
		spotVms = e.filterSpots(ctx, spotVms)
		if len(spotVms) == 0 {
			log.Debugf("no vms suitable for spot pools: %s", attr)
			return []VirtualMachine{}, []VirtualMachine{}, nil
		}
	}
	return odVms, spotVms, nil

}

func (e *Engine) findVmsWithAttrValues(ctx context.Context, provider string, service string, region string, zones []string, attr string, values []float64) ([]VirtualMachine, error) {
	var err error
	log := logger.Extract(ctx)
	log.Infof("looking for instance types and on demand prices with value %v, attribute %s", values, attr)
	var (
		vms []VirtualMachine
	)

	if len(zones) == 0 {
		if zones, err = e.ciSource.GetZones(provider, service, region); err != nil {
			log.WithError(err).Debugf("couldn't describe region: %s, provider: %s", region, provider)
			return nil, emperror.With(err, "retrieval", "region")
		}
	}

	allProducts, err := e.ciSource.GetProductDetails(provider, service, region)
	if err != nil {
		log.WithError(err).Debugf("couldn't get product details. region: %s, provider: %s", region, provider)
		return nil, emperror.With(err, "retrieval", "productdetails")
	}

	for _, p := range allProducts {
		included := true
		if len(values) > 0 {
			included = false
			for _, v := range values {
				switch attr {
				case Cpu:
					if p.Cpus == v {
						included = true
						continue
					}
				case Memory:
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
			vms = append(vms, VirtualMachine{
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

	log.Debugf("found vms with %s values %v: %v", attr, values, vms)
	return vms, nil
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
					VmClass:  npd.getVmClass(),
					SumNodes: npd.SumNodes,
				}
				break
			}
		}
	}
	return nps
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

// filtersApply returns true if all the filters apply for the given vm
func (e *Engine) filtersApply(ctx context.Context, vm VirtualMachine, filters []vmFilter, req ClusterRecommendationReq) bool {

	for _, filter := range filters {
		if !filter(ctx, vm, req) {
			// one of the filters doesn't apply - quit the iteration
			return false
		}
	}
	// no filters or applies
	return true
}

// RecommendAttrValues selects the attribute values allowed to participate in the recommendation process
func (e *Engine) RecommendAttrValues(ctx context.Context, provider string, service string, region string, attr string, req ClusterRecommendationReq) ([]float64, error) {

	allValues, err := e.ciSource.GetAttributeValues(provider, service, region, attr)
	if err != nil {
		return nil, emperror.With(err, recommenderErrorTag, "attributes")
	}

	values, err := AttributeValues(allValues).SelectAttributeValues(ctx, req.minValuePerVm(ctx, attr), req.maxValuePerVm(ctx, attr))
	if err != nil {
		return nil, emperror.With(err, recommenderErrorTag, "attributes")
	}

	return values, nil
}

// filtersForAttr returns the slice for
func (e *Engine) filtersForAttr(ctx context.Context, attr string, provider string) ([]vmFilter, error) {
	// generic filters - not depending on providers and attributes
	var filters = []vmFilter{e.includesFilter, e.excludesFilter}

	// provider specific filters
	switch provider {
	case "amazon":
		filters = append(filters, e.currentGenFilter, e.burstFilter, e.ntwPerformanceFilter)
	case "google":
		filters = append(filters, e.ntwPerformanceFilter)
	}

	// attribute specific filters
	switch attr {
	case Cpu:
		filters = append(filters, e.minMemRatioFilter)
	case Memory:
		filters = append(filters, e.minCpuRatioFilter)
	default:
		return nil, emperror.With(errors.New("unsupported attribute"), recommenderErrorTag, "attrVal", attr)
	}

	return filters, nil
}

// sortByAttrValue returns the slice for
func (e *Engine) sortByAttrValue(ctx context.Context, attr string, vms []VirtualMachine) {
	// sort and cut
	switch attr {
	case Memory:
		sort.Sort(ByAvgPricePerMemory(vms))
	case Cpu:
		sort.Sort(ByAvgPricePerCpu(vms))
	default:
		logger.Extract(ctx).Error("unsupported attribute: ", attr)
	}
}

// RecommendNodePools finds the slice of NodePools that may participate in the recommendation process
func (e *Engine) RecommendNodePools(ctx context.Context, attr string, odVms []VirtualMachine, spotVms []VirtualMachine, req ClusterRecommendationReq, layout []NodePool) ([]NodePool, error) {
	log := logger.Extract(ctx)

	log.Debugf("requested sum for attribute [%s]: [%f]", attr, req.sum(ctx, attr))
	var sumOnDemandValue = req.sum(ctx, attr) * float64(req.OnDemandPct) / 100
	var sumSpotValue = req.sum(ctx, attr) - sumOnDemandValue

	log.Debugf("on demand sum value for attr [%s]: [%f]", attr, sumOnDemandValue)
	log.Debugf("spot sum value for attr [%s]: [%f]", attr, sumSpotValue)

	// recommend on-demands
	odNps := make([]NodePool, 0)

	//TODO: validate if there's no on-demand in layout but we want to add ondemands
	for _, np := range layout {
		if np.VmClass == regular {
			odNps = append(odNps, np)
		}
	}
	if len(odVms) > 0 {
		// find cheapest onDemand instance from the list - based on price per attribute
		selectedOnDemand := odVms[0]
		for _, vm := range odVms {
			if vm.OnDemandPrice/vm.getAttrValue(attr) < selectedOnDemand.OnDemandPrice/selectedOnDemand.getAttrValue(attr) {
				selectedOnDemand = vm
			}
		}
		nodesToAdd := int(math.Ceil(sumOnDemandValue / selectedOnDemand.getAttrValue(attr)))
		if layout == nil {
			odNps = append(odNps, NodePool{
				SumNodes: nodesToAdd,
				VmClass:  regular,
				VmType:   selectedOnDemand,
			})
		} else {
			for i, np := range odNps {
				if np.VmType.Type == selectedOnDemand.Type {
					odNps[i].SumNodes += nodesToAdd
				}
			}
		}
	}

	// recommend spot pools
	spotNps := make([]NodePool, 0)
	excludedSpotNps := make([]NodePool, 0)

	e.sortByAttrValue(ctx, attr, spotVms)

	var N int
	if layout == nil {
		// the "magic" number of machines for diversifying the types
		N = int(math.Min(float64(findN(avgNodeCount(req.MinNodes, req.MaxNodes))), float64(len(spotVms))))
		// the second "magic" number for diversifying the layout
		M := int(math.Min(math.Ceil(float64(N)*1.5), float64(len(spotVms))))
		log.Debugf("Magic 'Marton' numbers: N=%d, M=%d", N, M)

		// the first M vm-s
		recommendedVms := spotVms[:M]

		// create spot nodepools - one for the first M vm-s
		for _, vm := range recommendedVms {
			spotNps = append(spotNps, NodePool{
				SumNodes: 0,
				VmClass:  spot,
				VmType:   vm,
			})
		}
	} else {
		sort.Sort(ByNonZeroNodePools(layout))
		var nonZeroNPs int
		for _, np := range layout {
			if np.VmClass == spot {
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
		log.Debugf("Magic 'Marton' number: N=%d", N)
	}
	log.Debugf("created [%d] regular and [%d] spot price node pools", len(odNps), len(spotNps))
	spotNps = fillSpotNodePools(ctx, sumSpotValue, N, spotNps, attr)
	if len(excludedSpotNps) > 0 {
		spotNps = append(spotNps, excludedSpotNps...)
	}
	return append(odNps, spotNps...), nil
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

func fillSpotNodePools(ctx context.Context, sumSpotValue float64, N int, nps []NodePool, attr string) []NodePool {
	log := logger.Extract(ctx)
	var (
		sumValueInPools, minValue float64
		idx, minIndex             int
	)
	for i := 0; i < N; i++ {
		v := float64(nps[i].SumNodes) * nps[i].VmType.getAttrValue(attr)
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
			sumValueInPools += nps[nodePoolIdx].VmType.getAttrValue(attr)
			log.Debugf("adding vm to the [%d]th (min sized) node pool, sum value in pools: [%f]", nodePoolIdx, sumValueInPools)
			idx++
		} else if nps[nodePoolIdx].getNextSum(attr) > nps[minIndex].getSum(attr) {
			// for other pools, if adding another vm would exceed the current sum of the cheapest option, move on to the next one
			log.Debugf("skip adding vm to the [%d]th node pool", nodePoolIdx)
			idx++
		} else {
			// otherwise add a new one, but do not move on to the next one
			nps[nodePoolIdx].SumNodes += 1
			sumValueInPools += nps[nodePoolIdx].VmType.getAttrValue(attr)
			log.Debugf("adding vm to the [%d]th node pool, sum value in pools: [%f]", nodePoolIdx, sumValueInPools)
		}
	}
	return nps
}

// maxValuePerVm calculates the maximum value per node for the given attribute
func (req *ClusterRecommendationReq) maxValuePerVm(ctx context.Context, attr string) float64 {
	switch attr {
	case Cpu:
		return req.SumCpu / float64(req.MinNodes)
	case Memory:
		return req.SumMem / float64(req.MinNodes)
	default:
		logger.Extract(ctx).Error("unsupported attribute: ", attr)
		return 0
	}
}

// minValuePerVm calculates the minimum value per node for the given attribute
func (req *ClusterRecommendationReq) minValuePerVm(ctx context.Context, attr string) float64 {
	switch attr {
	case Cpu:
		return req.SumCpu / float64(req.MaxNodes)
	case Memory:
		return req.SumMem / float64(req.MaxNodes)
	default:
		logger.Extract(ctx).Error("unsupported attribute: ", attr)
		return 0
	}
}

// gets the requested sum for the attribute value
func (req *ClusterRecommendationReq) sum(ctx context.Context, attr string) float64 {
	switch attr {
	case Cpu:
		return req.SumCpu
	case Memory:
		return req.SumMem
	default:
		logger.Extract(ctx).Error("unsupported attribute: ", attr)
		return 0
	}
}

// getSum gets the total value for the given attribute per pool
func (n *NodePool) getSum(attr string) float64 {
	return float64(n.SumNodes) * n.VmType.getAttrValue(attr)
}

// getNextSum gets the total value if the pool was increased by one
func (n *NodePool) getNextSum(attr string) float64 {
	return n.getSum(attr) + n.VmType.getAttrValue(attr)
}

// poolPrice calculates the price of the pool
func (n *NodePool) poolPrice() float64 {
	var sum = float64(0)
	switch n.VmClass {
	case regular:
		sum = float64(n.SumNodes) * n.VmType.OnDemandPrice
	case spot:
		sum = float64(n.SumNodes) * n.VmType.AvgPrice
	}
	return sum
}

func contains(slice []string, s string) bool {
	for _, e := range slice {
		if e == s {
			return true
		}
	}
	return false
}
