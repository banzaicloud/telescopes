package recommender

import (
	"time"

	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

type Engine struct {
	ReevaluationInterval time.Duration
	Recommender          *Recommender
	CachedInstanceTypes  []string
	//TODO: if we want to host the recommender as a service and create an HA deployment then we'll need to find a proper KV store instead of this cache
	RecommendationStore *cache.Cache
}

func NewEngine(ri time.Duration, region string, it []string, cache *cache.Cache) (*Engine, error) {
	recommender, err := NewRecommender(region)
	if err != nil {
		return nil, err
	}
	return &Engine{
		ReevaluationInterval: ri,
		Recommender:          recommender,
		CachedInstanceTypes:  it,
		RecommendationStore:  cache,
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

func (e *Engine) RecommendCluster(req ClusterRecommendationReq) (ClusterRecommendationResp, error) {
	log.Infof("recommending cluster configuration")
	return ClusterRecommendationResp{
		Provider: "aws",
		NodePools: []NodePool{
			{
				SumNodes: 8,
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
				SumNodes: 8,
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
				SumNodes: 9,
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
		},
	}, nil
}
