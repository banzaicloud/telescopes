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
