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
	"time"

	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/banzaicloud/telescopes/api"
	"github.com/banzaicloud/telescopes/productinfo"
	"github.com/banzaicloud/telescopes/productinfo/azure"
	"github.com/banzaicloud/telescopes/productinfo/ec2"
	"github.com/banzaicloud/telescopes/productinfo/gce"
	"github.com/banzaicloud/telescopes/recommender"
	"github.com/gin-gonic/gin"
	flag "github.com/spf13/pflag"
)

const (
	// the list of flags supported by the application
	// these constants can be used to retrieve the passed in values or defaults via viper

	logLevelFlag               = "log-level"
	listenAddressFlag          = "listen-address"
	prodInfRenewalIntervalFlag = "product-info-renewal-interval"
	prometheusAddressFlag      = "prometheus-address"
	prometheusQueryFlag        = "prometheus-query"
	providerFlag               = "provider"
	devModeFlag                = "dev-mode"
	tokenSigningKeyFlag        = "tokensigningkey"
	vaultAddrFlag              = "vault_addr"

	//temporary flags
	gceProjectIdFlag    = "gce-project-id"
	gceApiKeyFlag       = "gce-api-key"
	azureSubscriptionId = "azure-subscription-id"

	cfgAppRole     = "telescopes_app_role"
	defaultAppRole = "telescopes"
)

var (
	// env vars required by the application
	cfgEnvVars = []string{tokenSigningKeyFlag, vaultAddrFlag}
)

// defineFlags defines supported flags and makes them available for viper
func defineFlags() {
	flag.String(logLevelFlag, "info", "log level")
	flag.String(listenAddressFlag, ":9090", "The address to listen on for HTTP requests.")
	flag.Duration(prodInfRenewalIntervalFlag, 24*time.Hour, "Duration (in go syntax) between renewing the ec2 product info. Example: 2h30m")
	flag.String(prometheusAddressFlag, "", "http address of a Prometheus instance that has AWS spot price metrics via banzaicloud/spot-price-exporter. If empty, the recommender will use current spot prices queried directly from the AWS API.")
	flag.String(prometheusQueryFlag, "avg_over_time(aws_spot_current_price{region=\"%s\", product_description=\"Linux/UNIX\"}[1w])", "advanced configuration: change the query used to query spot price info from Prometheus.")
	flag.Bool(devModeFlag, false, "advanced configuration - development mode, no auth")
	flag.String(gceProjectIdFlag, "", "GCE project ID to use")
	flag.String(gceApiKeyFlag, "", "GCE API key to use for getting SKUs")
	flag.String(azureSubscriptionId, "", "Azure subscription ID to use with the APIs")
	flag.StringSlice(providerFlag, []string{recommender.Ec2, recommender.Gce, recommender.Azure}, "Providers that will be used with the recommender.")
	flag.String(tokenSigningKeyFlag, "", "The token signing key for the authentication process")
	flag.String(vaultAddrFlag, "", "The vault address for authentication token management")
}

// bindFlags binds parsed flags into viper
func bindFlags() {
	flag.Parse()
	viper.BindPFlags(flag.CommandLine)
}

// setLogLevel sets the log level
func setLogLevel() {
	parsedLevel, err := log.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		log.WithError(err).Warnf("Couldn't parse log level, using default: %s", log.GetLevel())
	} else {
		log.SetLevel(parsedLevel)
		log.Debugf("Set log level to %s", parsedLevel)
	}
}
func init() {

	// describe the flags for the application
	defineFlags()

	// all the flegs should be referenced through viper after this call
	// flags are available through the entire application via viper
	bindFlags()

	// handle log level
	setLogLevel()

	// set configuration defaults
	viper.SetDefault(cfgAppRole, defaultAppRole)
	viper.BindEnv(tokenSigningKeyFlag)

}

// ensureCfg ensures that the application configuration is available
// currently this only refers to configuration as environment variable
// to be extended for app critical entries (flags, config files ...)
func ensureCfg() {

	for _, envVar := range cfgEnvVars {
		// bind the env var
		viper.BindEnv(envVar)

		// read the env var value
		if nil == viper.Get(envVar) {
			flag.Usage()
			log.Fatalf("application is missing configuration: %s \n %s", envVar)
		}
	}
}

func main() {

	ensureCfg()

	c := cache.New(24*time.Hour, 24.*time.Hour)

	productInfo, err := productinfo.NewCachingProductInfo(viper.GetDuration(prodInfRenewalIntervalFlag), c, infoers())
	if err != nil {
		log.Fatal(err)
	}
	//go productInfo.Start(context.Background())

	engine, err := recommender.NewEngine(productInfo)
	if err != nil {
		log.Fatal(err)
	}

	routeHandler := api.NewRouteHandler(engine, api.NewValidator(viper.GetStringSlice(providerFlag)), productInfo)

	// new default gin engine (recovery, logger middleware)
	router := gin.Default()

	// enable authentication if not dev-mode
	if !viper.GetBool(devModeFlag) {
		log.Debug("enable authentication")
		signingKey := viper.GetString(tokenSigningKeyFlag)
		appRole := viper.GetString(cfgAppRole)

		routeHandler.EnableAuth(router, appRole, signingKey)
	}

	log.Info("Initialized gin router")
	routeHandler.ConfigureRoutes(router)
	log.Info("Configured routes")

	router.Run(viper.GetString(listenAddressFlag))
}

func infoers() map[string]productinfo.ProductInfoer {
	providers := viper.GetStringSlice(providerFlag)
	infoers := make(map[string]productinfo.ProductInfoer, len(providers))
	for _, p := range providers {
		var infoer productinfo.ProductInfoer
		var err error
		if err != nil {
			log.Fatalf(err.Error())
		}
		switch p {
		case recommender.Ec2:
			infoer, err = ec2.NewEc2Infoer(ec2.NewPricing(ec2.NewConfig()), viper.GetString(prometheusAddressFlag),
				viper.GetString(prometheusQueryFlag))
		case recommender.Gce:
			infoer, err = gce.NewGceInfoer(viper.GetString(gceApiKeyFlag), viper.GetString(gceProjectIdFlag))
		case recommender.Azure:
			infoer, err = azure.NewAzureInfoer(viper.GetString(azureSubscriptionId))
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
