package recommender

import (
	"errors"
	"math"
	"sort"
	"time"

	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

type Engine struct {
	ReevaluationInterval time.Duration
	Recommender          *Recommender
	CachedInstanceTypes  []string
	RecommendationStore  *cache.Cache
	VmRegistries         map[string]VmRegistry

	//TODO: if we want to host the recommender as a service and create an HA deployment then we'll need to find a proper KV store instead of this cache
}

func NewEngine(ri time.Duration, region string, it []string, cache *cache.Cache, vmRegistries map[string]VmRegistry) (*Engine, error) {
	recommender, err := NewRecommender(region)
	if err != nil {
		return nil, err
	}
	return &Engine{
		ReevaluationInterval: ri,
		Recommender:          recommender,
		CachedInstanceTypes:  it,
		RecommendationStore:  cache,
		VmRegistries:         vmRegistries,
	}, nil
}

func (e *Engine) Start() {
	ticker := time.NewTicker(e.ReevaluationInterval)
	region := *e.Recommender.Session.Config.Region
	for {
		select {
		//TODO: case close
		case <-ticker.C:
			log.Info("reevaluating recommendations...", time.Now())
			//TODO: this is a very naive implementation: if we want to cache all the instance types then we should make it parallel, cache some AWS info, etc..
			// depending on the complexity of the recommendation engine, we may need to make it even more complex
			for _, it := range e.CachedInstanceTypes {
				rec, err := e.Recommender.RecommendSpotInstanceTypes(region, nil, it)
				if err != nil {
					log.WithError(err).Error("Failed to reevaluate recommendations, recommendation store won't be updated")
				}
				e.RecommendationStore.Set(it, rec, cache.NoExpiration)
			}
		}
	}
}

func (e *Engine) RetrieveRecommendation(requestedAZs []string, baseInstanceType string) (AZRecommendation, error) {
	if rec, ok := e.RecommendationStore.Get(baseInstanceType); ok {
		log.Info("recommendation found in cache, filtering by az")
		var recommendations AZRecommendation
		if requestedAZs != nil {
			recommendations = make(AZRecommendation)
			for _, az := range requestedAZs {
				recs := rec.(AZRecommendation)
				recommendations[az] = recs[az]
			}
		} else {
			recommendations = rec.(AZRecommendation)
		}
		return recommendations, nil
	} else {
		log.Info("recommendation not found in cache")
		recommendation, err := e.Recommender.RecommendSpotInstanceTypes(*e.Recommender.Session.Config.Region, requestedAZs, baseInstanceType)
		if err != nil {
			return nil, err
		}
		e.RecommendationStore.Set(baseInstanceType, recommendation, 1*time.Minute)
		return recommendation, nil
	}
}

type ClusterRecommendationReq struct {
	SumCpu      float64  `json:"sumCpu"`
	SumMem      float64  `json:"sumMem"`
	MinNodes    int      `json:"minNodes,omitempty"`
	MaxNodes    int      `json:"maxNodes,omitempty"`
	SameSize    bool     `json:"sameSize,omitempty"`
	OnDemandPct int      `json:"onDemandPct,omitempty"`
	Zones       []string `json:"zones,omitempty"`
	SumGpu      int      `json:"sumGpu,omitempty"`
	// TODO: i/o, network
}

type ClusterRecommendationResp struct {
	Provider  string     `json:provider`
	Zones     []string   `json:"zones,omitempty"`
	NodePools []NodePool `json:nodePools`
}

type NodePool struct {
	VmType   VirtualMachine `json:vm`
	SumNodes int            `json:sumNodes`
	VmClass  string         `json:vmClass`
}

type VirtualMachine struct {
	Type          string  `json:type`
	AvgPrice      float64 `json:avgPrice`
	OnDemandPrice float64 `json:onDemandPrice`
	Cpus          float64 `json:cpusPerVm`
	Mem           float64 `json:memPerVm`
	Gpus          float64 `json:gpusPerVm`
	// i/o, network
}

type vmFilter func(vm VirtualMachine, req ClusterRecommendationReq) bool

func (e *Engine) minMemRatioFilter(vm VirtualMachine, req ClusterRecommendationReq) bool {
	minMemToCpuRatio := req.SumMem / req.SumCpu
	if vm.Mem/vm.Cpus < minMemToCpuRatio {
		return false
	}
	return true
}

// TODO: i/o filter, nw filter, gpu filter, etc...

type VmRegistry interface {
	findCpuUnits(min float64, max float64) ([]float64, error)
	findVmsWithCpuUnits(region string, zones []string, cpuUnits []float64) ([]VirtualMachine, error)
}

type ByAvgPricePerCpu []VirtualMachine

func (a ByAvgPricePerCpu) Len() int      { return len(a) }
func (a ByAvgPricePerCpu) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByAvgPricePerCpu) Less(i, j int) bool {
	pricePerCpu1 := a[i].AvgPrice / a[i].Cpus
	pricePerCpu2 := a[j].AvgPrice / a[j].Cpus
	return pricePerCpu1 < pricePerCpu2
}

func (e *Engine) RecommendCluster(provider string, region string, req ClusterRecommendationReq) (*ClusterRecommendationResp, error) {
	log.Infof("recommending cluster configuration")

	// TODO: MEM based recommendation

	// 1. CPU based recommendation
	maxCpuPerVm := req.SumCpu / float64(req.MinNodes)
	minCpuPerVm := req.SumCpu / float64(req.MaxNodes)

	vmRegistry := e.VmRegistries[provider]

	cpuUnits, err := vmRegistry.findCpuUnits(minCpuPerVm, maxCpuPerVm)
	if err != nil {
		return nil, err
	}

	vmsInRange, err := vmRegistry.findVmsWithCpuUnits(region, req.Zones, cpuUnits)
	if err != nil {
		return nil, err
	}

	var filteredVms []VirtualMachine

	for _, vm := range vmsInRange {
		for _, filter := range []vmFilter{e.minMemRatioFilter} {
			if filter(vm, req) {
				filteredVms = append(filteredVms, vm)
			}
		}
	}

	if len(filteredVms) == 0 {
		return nil, errors.New("couldn't find any VMs to recommend")
	}

	var nps []NodePool

	// find cheapest onDemand instance from the list - based on pricePerCpu
	selectedOnDemand := filteredVms[0]
	for _, vm := range filteredVms {
		if vm.OnDemandPrice/vm.Cpus < selectedOnDemand.OnDemandPrice/selectedOnDemand.Cpus {
			selectedOnDemand = vm
		}
	}

	var sumOnDemandCpu = req.SumCpu * float64(req.OnDemandPct) / 100
	var sumSpotCpu = req.SumCpu - sumOnDemandCpu

	// create and append on-demand pool
	onDemandPool := NodePool{
		SumNodes: int(math.Ceil(sumOnDemandCpu / selectedOnDemand.Cpus)),
		VmClass:  "regular",
		VmType:   selectedOnDemand,
	}

	nps = append(nps, onDemandPool)

	// sort and cut
	sort.Sort(ByAvgPricePerCpu(filteredVms))

	N := int(math.Min(float64(findN(cpuUnits, req.SumCpu)), float64(len(filteredVms))))
	M := int(math.Min(math.Ceil(float64(N)*1.5), float64(len(filteredVms))))

	recommendedVms := filteredVms[:M]

	// create spot nodepools
	for _, vm := range recommendedVms {
		nps = append(nps, NodePool{
			SumNodes: 0,
			VmClass:  "spot",
			VmType:   vm,
		})
	}

	// fill up instances in spot pools
	i := 0
	var sumCpuInPools float64 = 0
	for sumCpuInPools < sumSpotCpu {
		nodePoolIdx := i%N + 1
		if nodePoolIdx == 1 {
			// always add a new instance to the cheapest option and move on
			nps[nodePoolIdx].SumNodes += 1
			sumCpuInPools += nps[nodePoolIdx].VmType.Cpus
			i++
		} else if float64(nps[nodePoolIdx].SumNodes+1)*nps[nodePoolIdx].VmType.Cpus > float64(nps[1].SumNodes)*nps[1].VmType.Cpus {
			// for other pools, if adding another vm would exceed the current sum cpu of the cheapest option, move on to the next one
			i++
		} else {
			// otherwise add a new one, but do not move on to the next one
			nps[nodePoolIdx].SumNodes += 1
			sumCpuInPools += nps[nodePoolIdx].VmType.Cpus
		}
	}

	return &ClusterRecommendationResp{
		Provider:  "aws",
		Zones:     req.Zones,
		NodePools: nps,
	}, nil
}
func avgNodeCount(cpuUnits []float64, sumCpu float64) int {
	var totalUnit float64
	for _, unit := range cpuUnits {
		totalUnit += unit
	}
	avgUnit := totalUnit / float64(len(cpuUnits))
	return int(math.Ceil(sumCpu / avgUnit))
}

func findN(cpuUnits []float64, sumCpu float64) int {
	avg := avgNodeCount(cpuUnits, sumCpu)
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
