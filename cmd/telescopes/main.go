// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	"net/url"
	"os"
	"strings"

	"github.com/banzaicloud/go-gin-prometheus"
	"github.com/banzaicloud/productinfo/pkg/logger"
	"github.com/banzaicloud/productinfo/pkg/productinfo-client/client"
	"github.com/banzaicloud/telescopes/internal/app/telescopes/api"
	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/gin-gonic/gin"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	// the list of flags supported by the application
	// these constants can be used to retrieve the passed in values or defaults via viper
	logLevelFlag         = "log-level"
	logFormatFlag        = "log-format"
	listenAddressFlag    = "listen-address"
	productInfoFlag      = "productinfo-address"
	devModeFlag          = "dev-mode"
	tokenSigningKeyFlag  = "token-signing-key"
	tokenSigningKeyAlias = "tokensigningkey"
	vaultAddrAlias       = "vault_addr"
	vaultAddrFlag        = "vault-address"
	helpFlag             = "help"
	metricsEnabledFlag   = "metrics-enabled"
	metricsAddressFlag   = "metrics-address"

	cfgAppRole     = "telescopes-app-role"
	defaultAppRole = "telescopes"
)

var (
	// env vars required by the application
	cfgEnvVars = []string{tokenSigningKeyFlag, vaultAddrFlag}
	//addressRegex, _ = regexp.Compile("^((http[s]?):\\/)?\\/?([^:\\/\\s]+)((\\/\\w+)*\\/)([\\w\\-\\.]+[^#?\\s]+)(.*)$")
)

// defineFlags defines supported flags and makes them available for viper
func defineFlags() {
	flag.String(logLevelFlag, "info", "log level")
	flag.String(logFormatFlag, "", "log format")
	flag.String(listenAddressFlag, ":9090", "the address where the server listens to HTTP requests.")
	flag.String(productInfoFlag, "http://localhost:9090/api/v1", "the address of the Product Info service to retrieve attribute and pricing info [format=scheme://host:port/basepath]")
	flag.Bool(devModeFlag, false, "development mode, if true token based authentication is disabled, false by default")
	flag.String(tokenSigningKeyFlag, "", "The token signing key for the authentication process")
	flag.String(vaultAddrFlag, "", "The vault address for authentication token management")
	flag.Bool(helpFlag, false, "print usage")
	flag.Bool(metricsEnabledFlag, false, "internal metrics are exposed if enabled")
	flag.String(metricsAddressFlag, ":9900", "the address where internal metrics are exposed")
}

// bindFlags binds parsed flags into viper
func bindFlags() {
	flag.Parse()
	viper.BindPFlags(flag.CommandLine)
}

// setLogLevel sets the log level
func setLogLevel() {
	logger.InitLogger(viper.GetString(logLevelFlag), viper.GetString(logFormatFlag))
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

}

// ensureCfg ensures that the application configuration is available
// currently this only refers to configuration as environment variable
// to be extended for app critical entries (flags, config files ...)
func ensureCfg(ctx context.Context) {

	for _, envVar := range cfgEnvVars {
		// bind the env var
		viper.BindEnv(envVar)

		// read the env var value
		if nil == viper.Get(envVar) {
			flag.Usage()
			logger.Extract(ctx).Fatal("application is missing configuration:", envVar)
		}
	}

	// translating flags to aliases for supporting legacy env vars
	switchFlagsToAliases(ctx)

}

// switchFlagsToAliases sets the environment variables required by legacy components from application flags
// todo investigate if there's a better way for this
func switchFlagsToAliases(ctx context.Context) {
	log := logger.Extract(ctx)
	// vault signing token hack / need to support legacy components (vault, auth)
	os.Setenv(strings.ToUpper(vaultAddrAlias), viper.GetString(vaultAddrFlag))
	os.Setenv(strings.ToUpper(tokenSigningKeyAlias), viper.GetString(tokenSigningKeyFlag))
	viper.BindEnv(vaultAddrAlias)
	viper.BindEnv(tokenSigningKeyAlias)
	log.Debugf("%s : %s", vaultAddrFlag, viper.Get(vaultAddrFlag))
	log.Debugf("%s : %s", tokenSigningKeyFlag, viper.Get(tokenSigningKeyFlag))
	log.Debugf("%s : %s", vaultAddrAlias, viper.Get(vaultAddrAlias))
	log.Debugf("%s : %s", tokenSigningKeyAlias, viper.Get(tokenSigningKeyAlias))
}

func main() {

	// application context, intended to hold extra information
	appCtx := logger.ToContext(context.Background(), logger.NewLogCtxBuilder().WithField("application", "telescope").Build())
	ctxLog := logger.Extract(appCtx)

	if viper.GetBool(helpFlag) {
		flag.Usage()
		return
	}

	ensureCfg(appCtx)

	piUrl := parseProductInfoAddress(appCtx)
	transport := httptransport.New(piUrl.Host, piUrl.Path, []string{piUrl.Scheme})
	pc := client.New(transport, strfmt.Default)

	engine, err := recommender.NewEngine(recommender.NewProductInfoClient(pc))
	quitOnError(appCtx, "failed to start telescopes", err)

	// configure the gin validator
	err = api.ConfigureValidator(appCtx, pc)
	quitOnError(appCtx, "failed to start telescopes", err)

	routeHandler := api.NewRouteHandler(engine)

	// new default gin engine (recovery, logger middleware)
	router := gin.Default()

	// enable authentication if not dev-mode
	if !viper.GetBool(devModeFlag) {
		ctxLog.Debug("enable authentication")
		signingKey := viper.GetString(tokenSigningKeyAlias)
		appRole := viper.GetString(cfgAppRole)

		routeHandler.EnableAuth(router, appRole, signingKey)
	}

	// add prometheus metric endpoint
	if viper.GetBool(metricsEnabledFlag) {
		p := ginprometheus.NewPrometheus("gin", []string{"provider", "service", "region"})
		p.SetListenAddress(viper.GetString(metricsAddressFlag))
		p.Use(router)
	}

	routeHandler.ConfigureRoutes(appCtx, router)
	ctxLog.Info("configured routes")

	router.Run(viper.GetString(listenAddressFlag))
}

func parseProductInfoAddress(ctx context.Context) *url.URL {
	productInfoAddress := viper.GetString(productInfoFlag)
	u, err := url.ParseRequestURI(productInfoAddress)
	if err != nil {
		logger.Extract(ctx).Fatal("invalid URI: ", productInfoFlag)
	}
	return u
}

func quitOnError(ctx context.Context, msg string, err error) {
	if err != nil {
		logger.Extract(ctx).WithField("cause", msg).WithError(err)
		flag.Usage()
		os.Exit(-1)
	}
}
