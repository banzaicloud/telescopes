package main

import (
	"flag"
	"strings"
	"time"

	"github.com/banzaicloud/cluster-recommender/api"
	"github.com/banzaicloud/cluster-recommender/recommender"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

var (
	addr                 = flag.String("listen-address", ":9090", "The address to listen on for HTTP requests.")
	reevaluationInterval = flag.Duration("reevaluation-interval", 60*time.Second, "Time (in seconds) between reevaluating the recommendations")
	rawLevel             = flag.String("log-level", "info", "log level")
	region               = flag.String("region", "eu-west-1", "AWS region where the recommender should work")
	cacheInstanceTypes   = flag.String("cache-instance-types", "m4.xlarge,m5.xlarge,c5.xlarge", "Recommendations are cached for these instance types (comma separated list)")
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
	c := cache.New(5*time.Hour, 10.*time.Hour)
	cachedInstanceTypes := strings.Split(strings.Replace(*cacheInstanceTypes, " ", "", -1), ",")

	vmRegistries := make(map[string]recommender.VmRegistry, 1)
	vmRegistries["ec2"] = &recommender.Ec2VmRegistry{}

	engine, err := recommender.NewEngine(*reevaluationInterval, *region, cachedInstanceTypes, c, vmRegistries)
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
