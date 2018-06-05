// Package main Cluster Recommender.
//
// This project can be used to recommend instance type groups on different cloud providers consisting of regular and spot/preemptible instances.
// The main goal is to provide and continuously manage a cost-effective but still stable cluster layout that's built up from a diverse set of regular and spot instances.
//
//     Schemes: http, https
//     BasePath: /api/v1
//     Version: 0.0.1
//     License: Apache 2.0 http://www.apache.org/licenses/LICENSE-2.0.html
//     Contact: Banzai Cloud<info@banzaicloud.com>
//
// swagger:meta
package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/banzaicloud/telescopes/api"
	"github.com/banzaicloud/telescopes/productinfo"
	"github.com/banzaicloud/telescopes/productinfo/ec2"
	"github.com/banzaicloud/telescopes/productinfo/gce"
	"github.com/banzaicloud/telescopes/recommender"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return fmt.Sprintf("%s", *i)
}

func (i *arrayFlags) Set(value string) error {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' '
	})
	*i = append(*i, parts...)
	return nil
}

var (
	rawLevel                   = flag.String("log-level", "info", "log level")
	addr                       = flag.String("listen-address", ":9090", "The address to listen on for HTTP requests.")
	productInfoRenewalInterval = flag.Duration("product-info-renewal-interval", 24*time.Hour, "Duration (in go syntax) between renewing the ec2 product info. Example: 2h30m")
	prometheusAddress          = flag.String("prometheus-address", "", "http address of a Prometheus instance that has AWS spot price metrics via banzaicloud/spot-price-exporter. If empty, the recommender will use current spot prices queried directly from the AWS API.")
	promQuery                  = flag.String("prometheus-query", "avg_over_time(aws_spot_current_price{region=\"%s\", product_description=\"Linux/UNIX\"}[1w])", "advanced configuration: change the query used to query spot price info from Prometheus.")
	devMode                    = flag.Bool("dev-mode", false, "advanced configuration - development mode, no auth")
	gceProjectId               = flag.String("gce-project-id", "", "GCE project ID to use")
	gceApiKey                  = flag.String("gce-api-key", "", "GCE API key to use for getting SKUs")
	providers                  arrayFlags
)

func init() {

	flag.Var(&providers, "provider", "Providers that will be used with the recommender.")
	flag.Parse()
	parsedLevel, err := log.ParseLevel(*rawLevel)
	if err != nil {
		log.WithError(err).Warnf("Couldn't parse log level, using default: %s", log.GetLevel())
	} else {
		log.SetLevel(parsedLevel)
		log.Debugf("Set log level to %s", parsedLevel)
	}
	if len(providers) == 0 {
		providers = arrayFlags{recommender.Ec2, recommender.Gce}
	}

}

const (
	cfgTokenSigningKey = "tokensigningkey"
	cfgVaultAddr       = "vault_addr"
)

var (
	// env vars required by the application
	cfgEnvVar = [3]string{cfgTokenSigningKey, cfgVaultAddr}
)

// ensureCfg ensures that the application configuration is available
// currently this only refers to configuration as environment variable
// to be extended for app critical entries (flags, config files ...)
func ensureCfg() {
	for _, envVar := range cfgEnvVar {
		// bind the env var
		viper.BindEnv(envVar)

		// read the env var value
		if nil == viper.Get(envVar) {
			panic(fmt.Sprintf("application is missing configuration: %s", envVar))
		}
	}
}

func main() {

	ensureCfg()

	c := cache.New(24*time.Hour, 24.*time.Hour)

	productInfo, err := productinfo.NewCachingProductInfo(*productInfoRenewalInterval, c, infoers())
	if err != nil {
		log.Fatal(err)
	}
	go productInfo.Start(context.Background())

	engine, err := recommender.NewEngine(productInfo)
	if err != nil {
		log.Fatal(err)
	}

	routeHandler := api.NewRouteHandler(engine, api.NewValidator(providers))

	// new default gin engine (recovery, logger middleware)
	router := gin.Default()

	// enable authentication if not dev-mode
	if !*devMode {
		log.Debug("enable authentication")
		routeHandler.EnableAuth(router)
	}

	log.Info("Initialized gin router")
	routeHandler.ConfigureRoutes(router)
	log.Info("Configured routes")

	router.Run(*addr)
}

func infoers() map[string]productinfo.ProductInfoer {
	infoers := make(map[string]productinfo.ProductInfoer, len(providers))
	for _, p := range providers {
		var infoer productinfo.ProductInfoer
		var err error
		if err != nil {
			log.Fatalf(err.Error())
		}
		switch p {
		case recommender.Ec2:
			infoer, err = ec2.NewEc2Infoer(ec2.NewPricing(ec2.NewConfig()), *prometheusAddress, *promQuery)
		case recommender.Gce:
			infoer, err = gce.NewGceInfoer(*gceApiKey, *gceProjectId)
		default:
			log.Fatalf("provider %s is not supported", p)
		}
		if err != nil {
			log.Fatalf("could not initialize product info provider: %s", err.Error())
		}
		infoers[p] = infoer
		log.Infof("Configured '%s' product info provider", p)
	}
	return infoers
}
