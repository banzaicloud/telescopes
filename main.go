package main

import (
	"flag"
	"time"

	"github.com/banzaicloud/spot-recommender/api"
	"github.com/banzaicloud/spot-recommender/recommender"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var (
	addr                 = flag.String("listen-address", ":9090", "The address to listen on for HTTP requests.")
	reevaluationInterval = flag.Duration("reevaluation-interval", 60*time.Second, "Time (in seconds) between reevaluating the recommendations")
	rawLevel             = flag.String("log-level", "info", "log level")
	region               = flag.String("region", "eu-west-1", "AWS region where the recommender should work")
)

func init() {
	flag.Parse()
	parsedLevel, err := log.ParseLevel(*rawLevel)
	if err != nil {
		log.WithError(err).Warnf("Couldn't parse log level, using default: %s", log.GetLevel())
	} else {
		log.SetLevel(parsedLevel)
		log.Debugf("Set log level to %s", parsedLevel)
	}
}

func main() {
	engine, err := recommender.NewEngine(*reevaluationInterval, *region)
	if err != nil {
		log.Fatal(err)
	}
	go engine.Start()

	routeHandler := api.NewRouteHandler(engine)

	router := gin.Default()
	log.Info("Initialized gin router")
	routeHandler.ConfigureRoutes(router)
	log.Info("Configured routes")

	router.Run(*addr)
}
