package recommender

import (
	"time"

	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

type Engine struct {
	ReevaluationInterval time.Duration
	Recommender          *Recommender
	RecommendationStore  *cache.Cache
}

func NewEngine(ri time.Duration, region string, cache *cache.Cache) (*Engine, error) {
	recommender, err := NewRecommender(region)
	if err != nil {
		return nil, err
	}
	return &Engine{
		ReevaluationInterval: ri,
		Recommender:          recommender,
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
			rec, err := e.Recommender.RecommendSpotInstanceTypes(region, nil, "m4.xlarge")
			if err != nil {
				log.WithError(err).Error("Failed to reevaluate recommendations, recommendation store won't be updated")
			}
			e.RecommendationStore.Set("m4.xlarge", rec, cache.NoExpiration)
		}
	}
}

func (e *Engine) RetrieveRecommendation(requestedAZs []string, baseInstanceType string) (AZRecommendation, error) {
	if rec, ok := e.RecommendationStore.Get("m4.xlarge"); ok {
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
		return recommendation, nil
	}
}
