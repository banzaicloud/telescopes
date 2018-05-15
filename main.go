package main

import (
	"context"
	"flag"
	"time"

	"github.com/banzaicloud/cluster-recommender/api"
	"github.com/banzaicloud/cluster-recommender/ec2_productinfo"
	"github.com/banzaicloud/cluster-recommender/recommender"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

var (
	rawLevel                   = flag.String("log-level", "info", "log level")
	addr                       = flag.String("listen-address", ":9090", "The address to listen on for HTTP requests.")
	productInfoRenewalInterval = flag.Duration("product-info-renewal-interval", 24*time.Hour, "Duration (in go syntax) between renewing the ec2 product info. Example: 2h30m")
	prometheusAddress          = flag.String("prometheus-address", "", "http address of a Prometheus instance that has AWS spot price metrics via banzaicloud/spot-price-exporter. If empty, the recommender will use current spot prices queried directly from the AWS API.")
	promQuery                  = flag.String("prometheus-query", "avg(avg_over_time(aws_spot_current_price{region=\"%s\", instance_type=\"%s\", availability_zone=~\"%s\", product_description=\"Linux/UNIX\"}[1w]))", "advanced configuration: change the query used to query spot price info from Prometheus.")
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

	productInfoProvider, err := ec2_productinfo.NewAwsInfoer()
	if err != nil {
		log.Fatalf("could not initialize product info provider: %s", err.Error())
		return
	}

	c := cache.New(24*time.Hour, 24.*time.Hour)

	ec2ProductInfo, err := ec2_productinfo.NewProductInfo(*productInfoRenewalInterval, c, productInfoProvider)
	if err != nil {
		log.Fatal(err)
	}
	go ec2ProductInfo.Start(context.Background())

	vmRegistries := make(map[string]recommender.VmRegistry, 1)
	ec2VmRegistry, err := recommender.NewEc2VmRegistry(ec2ProductInfo, *prometheusAddress, *promQuery)
	if err != nil {
		log.Fatal(err)
	}
	vmRegistries["ec2"] = ec2VmRegistry

	engine, err := recommender.NewEngine(vmRegistries)
	if err != nil {
		log.Fatal(err)
	}

	routeHandler := api.NewRouteHandler(engine)

	router := gin.Default()
	log.Info("Initialized gin router")
	routeHandler.ConfigureRoutes(router)
	log.Info("Configured routes")

	router.Run(*addr)
}
