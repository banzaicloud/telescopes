package recommender

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/banzaicloud/telescopes/productinfo"
	log "github.com/sirupsen/logrus"
)

const (
	// Gce is the identifier of the Google Cloud Engine provider
	Gce = "gce"
	// Ec2 is the identifier of the Google Cloud Engine provider
	Ec2 = "ec2"
	// Azure is the identifier of the MS Azure provider
	Azure = "azure"

	// vm types
	regular = "regular"
	spot    = "spot"
)

// ClusterRecommender defines operations for cluster recommendations
type ClusterRecommender interface {
	// RecommendAttrValues recommends attributes based on the input
	RecommendAttrValues(provider string, attr string, req ClusterRecommendationReq) ([]float64, error)

	// RecommendVms recommends a set of virtual machines based on the provided parameters
	RecommendVms(provider string, region string, attr string, values []float64, filters []vmFilter, req ClusterRecommendationReq) ([]VirtualMachine, error)

	// RecommendNodePools recommends a slice of node pools to be part of the caluster being recommended
	RecommendNodePools(attr string, vms []VirtualMachine, values []float64, req ClusterRecommendationReq) ([]NodePool, error)

	// RecommendCluster recommends a cluster layout on the given cloud provider, region and wanted resources
	RecommendCluster(provider string, region string, req ClusterRecommendationReq) (*ClusterRecommendationResp, error)
}

// Engine represents the recommendation engine, it operates on a map of provider -> VmRegistry
type Engine struct {
	productInfo productinfo.ProductInfo
}

// NewEngine creates a new Engine instance
func NewEngine(pi productinfo.ProductInfo) (*Engine, error) {
	if pi == nil {
		return nil, errors.New("could not create engine")
	}
	return &Engine{
		productInfo: pi,
	}, nil
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
	OnDemandPct int `json:"onDemandPct,omitempty" binding:"min=1,max=100"`
	// Availability zones that the cluster should expand to
	Zones []string `json:"zones,omitempty" binding:"dive,zone"`
	// Total number of GPUs requested for the cluster
	SumGpu int `json:"sumGpu,omitempty"`
	// Are burst instances allowed in recommendation
	AllowBurst *bool `json:"allowBurst,omitempty"`
	// NertworkPerf specifies the network performance category
	NetworkPerf *string `json:"networkPerf" binding:"omitempty,network"`
}

// ClusterRecommendationResp encapsulates recommendation result data
// swagger:response recommendationResp
type ClusterRecommendationResp struct {
	// The cloud provider
	Provider string `json:"provider"`
	// Availability zones in the recommendation - a multi-zone recommendation means that all node pools should expand to all zones
	Zones []string `json:"zones,omitempty"`
	// Recommended node pools
	NodePools []NodePool `json:"nodePools"`
}

// ProductsResp holds the list of available machine types in a region
// swagger:response recommendationResp
type ProductsResp struct {
	// The cloud provider
	Provider string           `json:"provider"`
	Products []VirtualMachine `json:"products,omitempty"`
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
	// NetworkPerf holds the network performance category
	NetworkPerf string `json:"networkPerf"`
}

func (v *VirtualMachine) getAttrValue(attr string) float64 {
	switch attr {
	case productinfo.Cpu:
		return v.Cpus
	case productinfo.Memory:
		return v.Mem
	default:
		return 0
	}
}

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
	if vm.NetworkPerf == *req.NetworkPerf { //the network performance category matches the vm
		return true
	}
	return false
}

// filterSpots selects vm-s that potentially can be part of "spot" node pools
func (e *Engine) filterSpots(vms []VirtualMachine) []VirtualMachine {
	fvms := make([]VirtualMachine, 0)
	for _, vm := range vms {
		if vm.AvgPrice != 0 {
			fvms = append(fvms, vm)
		}
	}
	return fvms
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

// RecommendCluster performs recommendation based on the provided arguments
func (e *Engine) RecommendCluster(provider string, region string, req ClusterRecommendationReq) (*ClusterRecommendationResp, error) {

	log.Infof("recommending cluster configuration. Provider: [%s], region: [%s], recommendation request: [%#v]",
		provider, region, req)

	attributes := []string{productinfo.Cpu, productinfo.Memory}
	nodePools := make(map[string][]NodePool, 2)

	for _, attr := range attributes {

		values, err := e.RecommendAttrValues(provider, attr, req)
		if err != nil {
			return nil, fmt.Errorf("could not get values for attr: [%s], cause: [%s]", attr, err.Error())
		}
		log.Debugf("recommended values for [%s]: count:[%d] , values: [%#v./te]", attr, len(values), values)

		vmFilters, _ := e.filtersForAttr(attr)

		filteredVms, err := e.RecommendVms(provider, region, attr, values, vmFilters, req)
		if err != nil {
			return nil, fmt.Errorf("could not get virtual machines for attr: [%s], cause: [%s]", attr, err.Error())
		}
		log.Debugf("recommended vms for [%s]: count:[%d] , values: [%#V]", attr, len(filteredVms), filteredVms)

		nps, err := e.RecommendNodePools(attr, filteredVms, values, req)
		if err != nil {
			return nil, fmt.Errorf("error while recommending node pools for attr: [%s], cause: [%s]", attr, err.Error())
		}
		log.Debugf("recommended node pools for [%s]: count:[%d] , values: [%#V]", attr, len(nps), nps)

		nodePools[attr] = nps
	}

	cheapestNodePoolSet := e.findCheapestNodePoolSet(nodePools)

	return &ClusterRecommendationResp{
		Provider:  provider,
		Zones:     req.Zones,
		NodePools: cheapestNodePoolSet,
	}, nil
}

func (e *Engine) GetProducts(provider string, region string) (*ProductsResp, error) {
	response := &ProductsResp{
		Provider: provider,
		Products: []VirtualMachine{
			{
				Type:          "m4.xlarge",
				Mem:           8,
				Cpus:          4,
				Gpus:          0,
				OnDemandPrice: 0.2,
				Burst:         false,
				AvgPrice:      0.02,
				NetworkPerf:   "high",
			},
			{
				Type:          "m4.2xlarge",
				Mem:           16,
				Cpus:          8,
				Gpus:          0,
				OnDemandPrice: 0.38,
				Burst:         false,
				AvgPrice:      0.04,
				NetworkPerf:   "high",
			},
		},
	}
	return response, nil
}

// findCheapestNodePoolSet looks up the "cheapest" node pool set from the provided map
func (e *Engine) findCheapestNodePoolSet(nodePoolSets map[string][]NodePool) []NodePool {
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
			sumCpus += np.getSum(productinfo.Cpu)
			sumMem += np.getSum(productinfo.Memory)
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

func (e *Engine) findValuesBetween(attrValues []float64, min float64, max float64) ([]float64, error) {
	if len(attrValues) == 0 {
		return nil, errors.New("no attribute values provided")
	}

	if min > max {
		return nil, errors.New("min value cannot be larger than the max value")
	}

	log.Debugf("finding values between: [%v, %v]", min, max)
	// sort attribute values in ascending order
	sort.Float64s(attrValues)

	if max < attrValues[0] {
		log.Debug("returning smallest value: %v", attrValues[0])
		return []float64{attrValues[0]}, nil
	}

	if min > attrValues[len(attrValues)-1] {
		log.Debugf("returning largest value: %v", attrValues[len(attrValues)-1])
		return []float64{attrValues[len(attrValues)-1]}, nil
	}

	var values []float64
	for i, attrVal := range attrValues {
		if attrVal >= min && attrVal <= max {
			values = append(values, attrValues[i])
		}
	}

	return values, nil
}

// avgNodeCount calculates the minimum number of nodes having the "average attribute value" required to fill up the
// requested value of the given attribute
func avgNodeCount(attrValues []float64, reqSum float64) int {
	var total float64

	// calculate the total value of the attributes
	for _, v := range attrValues {
		total += v
	}
	// the average attribute value
	avgValue := total / float64(len(attrValues))

	// the (rounded up) number of nodes with average attribute value that are needed to reach the "sum"
	return int(math.Ceil(reqSum / avgValue))
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
func (e *Engine) RecommendVms(provider string, region string, attr string, values []float64, filters []vmFilter, req ClusterRecommendationReq) ([]VirtualMachine, error) {
	log.Infof("recommending virtual machines for attribute: [%s]", attr)

	vmsInRange, err := e.findVmsWithAttrValues(provider, region, req.Zones, attr, values)
	if err != nil {
		return nil, err
	}

	var filteredVms []VirtualMachine
	for _, vm := range vmsInRange {
		if e.filtersApply(vm, filters, req) {
			filteredVms = append(filteredVms, vm)
		}
	}

	if len(filteredVms) == 0 {
		return nil, errors.New("couldn't find any VMs to recommend")
	}

	return filteredVms, nil
}

func (e *Engine) findVmsWithAttrValues(provider string, region string, zones []string, attr string, values []float64) ([]VirtualMachine, error) {
	log.Infof("Getting instance types and on demand prices with %v %s", values, attr)
	var vms []VirtualMachine

	if zones == nil || len(zones) == 0 {
		zones = []string{}
	}

	if len(zones) == 0 {
		z, err := e.productInfo.GetZones(provider, region)
		if err != nil {
			return nil, err
		}
		zones = z
	}

	ntwMapper, err := e.productInfo.GetNetworkPerfMapper(provider)
	if err != nil {
		return nil, err
	}

	for _, v := range values {
		vmInfos, err := e.productInfo.GetVmsWithAttrValue(provider, region, attr, v)
		if err != nil {
			return nil, err
		}
		for _, vmInfo := range vmInfos {
			vm := VirtualMachine{
				Type:          vmInfo.Type,
				OnDemandPrice: vmInfo.OnDemandPrice,
				Cpus:          vmInfo.Cpus,
				Mem:           vmInfo.Mem,
				Gpus:          vmInfo.Gpus,
				Burst:         vmInfo.IsBurst(),
				NetworkPerf:   vmInfo.NetworkPerformance(ntwMapper),
			}
			odPrice, spotPrice, err := e.productInfo.GetPrice(provider, region, vmInfo.Type, zones)
			if err != nil {
				log.WithError(err).Warnf("couldn't get price for instance type %s, provider=%s, region=%s, zones=%s", vmInfo.Type, provider, region, zones)
				continue
			}
			if odPrice > 0 {
				vm.OnDemandPrice = odPrice
			}
			vm.AvgPrice = spotPrice
			vms = append(vms, vm)
		}
	}

	log.Debugf("found vms with %s values %v: %v", attr, values, vms)
	return vms, nil
}

// filtersApply returns true if all the filters apply for the given vm
func (e *Engine) filtersApply(vm VirtualMachine, filters []vmFilter, req ClusterRecommendationReq) bool {

	for _, filter := range filters {
		if !filter(vm, req) {
			// one of the filters doesn't apply - quit the iteration
			return false
		}
	}
	// no filters or applies
	return true
}

// RecommendAttrValues selects the attribute values allowed to participate in the recommendation process
func (e *Engine) RecommendAttrValues(provider string, attr string, req ClusterRecommendationReq) ([]float64, error) {

	allValues, err := e.productInfo.GetAttrValues(provider, attr)
	if err != nil {
		return nil, err
	}

	minValuePerVm, err := req.minValuePerVm(attr)
	if err != nil {
		return nil, err
	}

	maxValuePerVm, _ := req.maxValuePerVm(attr)

	values, err := e.findValuesBetween(allValues, minValuePerVm, maxValuePerVm)
	if err != nil {
		return nil, err
	}

	return values, nil
}

// filtersForAttr returns the slice for
func (e *Engine) filtersForAttr(attr string) ([]vmFilter, error) {
	switch attr {
	case productinfo.Cpu:
		return []vmFilter{e.ntwPerformanceFilter, e.minMemRatioFilter, e.burstFilter}, nil
	case productinfo.Memory:
		return []vmFilter{e.ntwPerformanceFilter, e.minCpuRatioFilter, e.burstFilter}, nil
	default:
		return nil, fmt.Errorf("unsupported attribute: [%s]", attr)
	}
}

// sortByAttrValue returns the slice for
func (e *Engine) sortByAttrValue(attr string, vms []VirtualMachine) error {
	// sort and cut
	switch attr {
	case productinfo.Memory:
		sort.Sort(ByAvgPricePerMemory(vms))
	case productinfo.Cpu:
		sort.Sort(ByAvgPricePerCpu(vms))
	default:
		return fmt.Errorf("unsupported attribute: [%s]", attr)
	}
	return nil
}

// RecommendNodePools finds the slice of NodePools that may participate in the recommendation process
func (e *Engine) RecommendNodePools(attr string, vms []VirtualMachine, values []float64, req ClusterRecommendationReq) ([]NodePool, error) {

	var nps []NodePool

	// find cheapest onDemand instance from the list - based on price per attribute
	selectedOnDemand := vms[0]
	for _, vm := range vms {
		if vm.OnDemandPrice/vm.getAttrValue(attr) < selectedOnDemand.OnDemandPrice/selectedOnDemand.getAttrValue(attr) {
			selectedOnDemand = vm
		}
	}

	requestedSum, err := req.sum(attr)
	if err != nil {
		return nil, fmt.Errorf("could not get sum for attr: [%s], cause: [%s]", attr, err.Error())
	}
	log.Debugf("requested sum for attribute [%s]: [%f]", attr, requestedSum)

	var sumOnDemandValue = requestedSum * float64(req.OnDemandPct) / 100
	var sumSpotValue = requestedSum - sumOnDemandValue

	log.Debugf("on demand sum value for attr [%s]: [%f]", attr, sumOnDemandValue)
	log.Debugf("spot sum value for attr [%s]: [%f]", attr, sumSpotValue)

	// create and append on-demand pool
	onDemandPool := NodePool{
		SumNodes: int(math.Ceil(sumOnDemandValue / selectedOnDemand.getAttrValue(attr))),
		VmClass:  regular,
		VmType:   selectedOnDemand,
	}

	nps = append(nps, onDemandPool)

	// retain only the nodes that are available as spot instances
	vms = e.filterSpots(vms)
	if len(vms) == 0 {
		return nil, errors.New("no vms suitable for spot pools")
	}

	// vms are sorted by attribute value
	err = e.sortByAttrValue(attr, vms)

	// the "magic" number of machines for diversifying the types
	N := int(math.Min(float64(findN(avgNodeCount(values, requestedSum))), float64(len(vms))))

	// the second "magic" number for diversifying the layout
	M := int(math.Min(math.Ceil(float64(N)*1.5), float64(len(vms))))

	log.Debugf("Magic 'Marton' numbers: N=%d, M=%d", N, M)

	// the first M vm-s
	recommendedVms := vms[:M]

	// create spot nodepools - one for the first M vm-s
	for _, vm := range recommendedVms {
		nps = append(nps, NodePool{
			SumNodes: 0,
			VmClass:  spot,
			VmType:   vm,
		})
	}
	log.Debugf("totally created [%d] regular and spot price node pools", len(nps))

	// fill up instances in spot pools
	i := 0
	var sumValueInPools float64
	for sumValueInPools < sumSpotValue {
		nodePoolIdx := i%N + 1
		if nodePoolIdx == 1 {
			// always add a new instance to the cheapest option and move on
			nps[nodePoolIdx].SumNodes += 1
			sumValueInPools += nps[nodePoolIdx].VmType.getAttrValue(attr)
			log.Debugf("adding vm to the [%d]th node pool sum value in pools: [%f]", nodePoolIdx, sumValueInPools)
			i++
		} else if nps[nodePoolIdx].getNextSum(attr) > nps[1].getSum(attr) {
			// for other pools, if adding another vm would exceed the current sum of the cheapest option, move on to the next one
			log.Debugf("skip adding vm to the [%d]th node pool - (price would exceed the sum)", nodePoolIdx)
			i++
		} else {
			// otherwise add a new one, but do not move on to the next one
			nps[nodePoolIdx].SumNodes += 1
			sumValueInPools += nps[nodePoolIdx].VmType.getAttrValue(attr)
			log.Debugf("adding vm to the [%d]th node pool sum value in pools: [%f]", nodePoolIdx, sumValueInPools)
		}
	}

	return nps, nil
}

// maxValuePerVm calculates the maximum value per node for the given attribute
func (req *ClusterRecommendationReq) maxValuePerVm(attr string) (float64, error) {
	switch attr {
	case productinfo.Cpu:
		return req.SumCpu / float64(req.MinNodes), nil
	case productinfo.Memory:
		return req.SumMem / float64(req.MinNodes), nil
	default:
		return 0, fmt.Errorf("unsupported attribute: [%s]", attr)
	}
}

// minValuePerVm calculates the minimum value per node for the given attribute
func (req *ClusterRecommendationReq) minValuePerVm(attr string) (float64, error) {
	switch attr {
	case productinfo.Cpu:
		return req.SumCpu / float64(req.MaxNodes), nil
	case productinfo.Memory:
		return req.SumMem / float64(req.MaxNodes), nil
	default:
		return 0, fmt.Errorf("unsupported attribute: [%s]", attr)
	}
}

// gets the requested sum for the attribute value
func (req *ClusterRecommendationReq) sum(attr string) (float64, error) {
	switch attr {
	case productinfo.Cpu:
		return req.SumCpu, nil
	case productinfo.Memory:
		return req.SumMem, nil
	default:
		return 0, fmt.Errorf("unsupported attribute: [%s]", attr)
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

// getSum gets the total value if the pool was increased by one
func (n *NodePool) addNode(attr string) float64 {
	n.SumNodes += 1
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
