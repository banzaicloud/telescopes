package recommender

import (
	"time"

	log "github.com/sirupsen/logrus"
)

type Engine struct {
	ReevaluationInterval time.Duration
	Recommender          *Recommender
}

func NewEngine(ri time.Duration, region string) (*Engine, error) {
	recommender, err := NewRecommender(region)
	if err != nil {
		return nil, err
	}
	return &Engine{
		ReevaluationInterval: ri,
		Recommender:          recommender,
	}, nil
}

func (e *Engine) Start() {
	ticker := time.NewTicker(e.ReevaluationInterval)

	for {
		select {
		//TODO: case close
		case <-ticker.C:
			log.Info("tick...", time.Now())
		}
	}
}

func (e *Engine) RetrieveRecommendation(region string, requestedAZs []string, baseInstanceType string) (AZRecommendation, error) {
	rec, err := e.Recommender.RecommendSpotInstanceTypes(region, requestedAZs, baseInstanceType)
	if err != nil {
		return nil, err
	}
	return rec, nil
}
