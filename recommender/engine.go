package recommender

import (
	"errors"
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
	Provider    string   `json:"provider"`
	SumCpu      int      `json:"sumCpu"`
	SumMem      int      `json:"sumMem"`
	MinNodes    int      `json:"minNodes,omitempty"`
	MaxNodes    int      `json:"maxNodes,omitempty"`
	SameSize    bool     `json:"sameSize,omitempty"`
	OnDemandPct int      `json:"onDemandPct,omitempty"`
	Zones       []string `json:"zones,omitempty"`
	SumGpu      int      `json:"sumGpu,omitempty"`
	//??? cost optimized vs stability optimized?
	//??? i/o, network
}

type ClusterRecommendationResp struct {
	Provider  string     `json:provider`
	NodePools []NodePool `json:nodePools`
}

type NodePool struct {
	VmType   VirtualMachine `json:vm`
	SumNodes int            `json:sumNodes`
	VmClass  string         `json:vmClass`
	Zones    []string       `json:"zones,omitempty"`
}

type VirtualMachine struct {
	Type          string  `json:type`
	AvgPrice      float32 `json:avgPrice`
	OnDemandPrice float32 `json:onDemandPrice`
	Cpus          float32 `json:cpusPerVm`
	Mem           float32 `json:memPerVm`
	Gpus          float32 `json:gpusPerVm`
	// i/o, network
}

type vmFilter func(vm VirtualMachine, req ClusterRecommendationReq) bool

func (e *Engine) minMemRatioFilter(vm VirtualMachine, req ClusterRecommendationReq) bool {
	minMemToCpuRatio := float32(req.SumMem) / float32(req.SumCpu)
	if float32(vm.Mem)/float32(vm.Cpus) < minMemToCpuRatio {
		return false
	}
	return true
}

// TODO: i/o filter, nw filter, gpu filter, etc...

type VmRegistry interface {
	findNearestCpuUnit(base int, larger bool) (int, error)
	findVmsWithCpuLimits(minCpuPerVm int, maxCpuPerVm int) ([]VirtualMachine, error)
}

func (e *Engine) RecommendCluster(req ClusterRecommendationReq) (*ClusterRecommendationResp, error) {
	log.Infof("recommending cluster configuration")

	// 1. CPU based recommendation
	maxCpuPerVm := req.SumCpu / req.MinNodes
	minCpuPerVm := req.SumCpu / req.MaxNodes

	// TODO: provider specific logic

	vmRegistry := e.VmRegistries[req.Provider]

	maxCpu, err := vmRegistry.findNearestCpuUnit(maxCpuPerVm, false)
	if err != nil {
		return nil, err
	}
	minCpu, err := vmRegistry.findNearestCpuUnit(minCpuPerVm, true)
	if err != nil {
		return nil, err
	}

	vmsInRange, err := vmRegistry.findVmsWithCpuLimits(minCpu, maxCpu)
	if err != nil {
		return nil, err
	}

	var recommendedVms []VirtualMachine

	for _, vm := range vmsInRange {
		for _, filter := range []vmFilter{e.minMemRatioFilter} {
			if filter(vm, req) {
				recommendedVms = append(recommendedVms, vm)
			}
		}
	}

	if len(recommendedVms) == 0 {
		return nil, errors.New("couldn't find any VMs to recommend")
	}

	var nps []NodePool

	// TODO: find on-demand instance type

	// TODO: create NodePool for on-demand instances

	// TODO: sort vm types per price_per_cpu

	// TODO: pick the first N? types

	// TODO: create nodepools for the first N types

	nps = []NodePool{
		{
			SumNodes: 0,
			VmClass:  "regular",
			VmType: VirtualMachine{
				Type:          "m5.xlarge",
				OnDemandPrice: 0.192,
				AvgPrice:      0.192,
				Cpus:          4,
				Mem:           16,
				Gpus:          0,
			},
			Zones: []string{"eu-west-1a", "eu-west-1b", "eu-west-1c"},
		},
		{
			SumNodes: 0,
			VmClass:  "spot",
			VmType: VirtualMachine{
				Type:          "m5.xlarge",
				OnDemandPrice: 0.192,
				AvgPrice:      0.08,
				Cpus:          4,
				Mem:           16,
				Gpus:          0,
			},
			Zones: []string{"eu-west-1a", "eu-west-1b", "eu-west-1c"},
		},
		{
			SumNodes: 0,
			VmClass:  "spot",
			VmType: VirtualMachine{
				Type:          "r4.xlarge",
				OnDemandPrice: 0.266,
				AvgPrice:      0.07,
				Cpus:          4,
				Mem:           30.5,
				Gpus:          0,
			},
			Zones: []string{"eu-west-1a", "eu-west-1b", "eu-west-1c"},
		},
	}

	N := 3 // TODO: N=??? (min(len(nps), X)

	i := 0
	var sumCpuInPools float32 = 0
	for sumCpuInPools < float32(req.SumCpu) {
		nodePoolIdx := i % N
		if nodePoolIdx == 0 {
			// always add a new instance to the cheapest option and move on
			nps[nodePoolIdx].SumNodes += 1
			sumCpuInPools += nps[nodePoolIdx].VmType.Cpus
			i++
		} else if float32(nps[nodePoolIdx].SumNodes+1)*nps[nodePoolIdx].VmType.Cpus > float32(nps[0].SumNodes)*nps[0].VmType.Cpus {
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
		NodePools: nps,
	}, nil
}
