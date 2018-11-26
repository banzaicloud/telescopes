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
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/banzaicloud/telescopes/internal/platform"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo-client/client"
	"github.com/banzaicloud/cloudinfo/pkg/logger"
	"github.com/banzaicloud/go-gin-prometheus"
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
	logLevelFlag        = "log-level"
	logFormatFlag       = "log-format"
	listenAddressFlag   = "listen-address"
	cloudInfoFlag       = "cloudinfo-address"
	devModeFlag         = "dev-mode"
	tokenSigningKeyFlag = "tokensigningkey"
	vaultAddrFlag       = "vault-address"
	helpFlag            = "help"
	metricsEnabledFlag  = "metrics-enabled"
	metricsAddressFlag  = "metrics-address"

	cfgAppRole     = "telescopes-app-role"
	defaultAppRole = "telescopes"
)

// defineFlags defines supported flags and makes them available for viper
func defineFlags() {
	flag.String(logLevelFlag, "info", "log level")
	flag.String(logFormatFlag, "", "log format")
	flag.String(listenAddressFlag, ":9090", "the address where the server listens to HTTP requests.")
	flag.String(cloudInfoFlag, "http://localhost:9090/api/v1", "the address of the Cloud Info service to retrieve attribute and pricing info [format=scheme://host:port/basepath]")
	flag.Bool(devModeFlag, false, "development mode, if true token based authentication is disabled, false by default")
	flag.String(tokenSigningKeyFlag, "", "The token signing key for the authentication process")
	flag.String(vaultAddrFlag, ":8200", "The vault address for authentication token management")
	flag.Bool(helpFlag, false, "print usage")
	flag.Bool(metricsEnabledFlag, false, "internal metrics are exposed if enabled")
	flag.String(metricsAddressFlag, ":9900", "the address where internal metrics are exposed")
}

// bindFlags binds parsed flags into viper
func bindFlags() {
	flag.Parse()
	if err := viper.BindPFlags(flag.CommandLine); err != nil {
		panic(fmt.Errorf("could not parse flags. error: %s", err))
	}
}

// setLogLevel sets the log level
func setLogLevel() {
	logger.InitLogger(viper.GetString(logLevelFlag), viper.GetString(logFormatFlag))
}
func init() {

	// describe the flags for the application
	defineFlags()

	// all the flags should be referenced through viper after this call
	// flags are available through the entire application via viper
	bindFlags()

	// Viper check for an environment variable
	viper.AutomaticEnv()
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)

	// handle log level
	setLogLevel()

	// set configuration defaults
	viper.SetDefault(cfgAppRole, defaultAppRole)

}

func main() {

	// application context, intended to hold extra information
	appCtx := logger.ToContext(context.Background(), logger.NewLogCtxBuilder().WithField("application", "telescope").Build())
	ctxLog := logger.Extract(appCtx)

	ctxLog.WithField("version", Version).WithField("commit_hash", CommitHash).WithField("build_date", BuildDate).Info("initializing telescopes")

	if viper.GetBool(helpFlag) {
		flag.Usage()
		return
	}

	piUrl := parseCloudInfoAddress(appCtx)
	transport := httptransport.New(piUrl.Host, piUrl.Path, []string{piUrl.Scheme})
	pc := client.New(transport, strfmt.Default)

	piCli := recommender.NewCloudInfoClient(pc)

	engine := recommender.NewEngine(piCli)

	// configure the gin validator
	if err := api.ConfigureValidator(appCtx, piCli); err != nil {
		quitOnError(appCtx, "failed to start telescopes", err)
	}

	buildInfo := buildinfo.New(Version, CommitHash, BuildDate)
	routeHandler := api.NewRouteHandler(engine, buildInfo)

	// new default gin engine (recovery, logger middleware)
	router := gin.Default()

	// enable authentication if not dev-mode
	if !viper.GetBool(devModeFlag) {
		ctxLog.Debug("enable authentication")
		signingKey := viper.GetString(tokenSigningKeyFlag)
		appRole := viper.GetString(cfgAppRole)

		routeHandler.EnableAuth(router, appRole, signingKey)
	}

	// add prometheus metric endpoint
	if viper.GetBool(metricsEnabledFlag) {
		p := ginprometheus.NewPrometheus("gin", []string{"provider", "service", "region"})
		p.SetListenAddress(viper.GetString(metricsAddressFlag))
		p.Use(router, "/metrics")
	}

	routeHandler.ConfigureRoutes(appCtx, router)
	ctxLog.Info("configured routes")

	if err := router.Run(viper.GetString(listenAddressFlag)); err != nil {
		panic(fmt.Errorf("could not run router. error: %s", err))
	}

}

func parseCloudInfoAddress(ctx context.Context) *url.URL {
	cloudInfoAddress := viper.GetString(cloudInfoFlag)
	u, err := url.ParseRequestURI(cloudInfoAddress)
	if err != nil {
		logger.Extract(ctx).Fatal("invalid URI: ", viper.GetString(cloudInfoFlag))
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
