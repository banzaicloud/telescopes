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
	"fmt"
	"net/url"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo-client/client"
	"github.com/banzaicloud/telescopes/internal/app/telescopes/api"
	"github.com/banzaicloud/telescopes/internal/platform/buildinfo"
	"github.com/banzaicloud/telescopes/internal/platform/log"
	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/gin-gonic/gin"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {

	// read configuration (commandline, env etc)
	Configure(viper.GetViper(), pflag.CommandLine)

	// parse the command line
	pflag.Parse()

	if viper.GetBool(helpFlag) {
		pflag.Usage()
		return
	}

	err := viper.ReadInConfig()
	_, configFileNotFound := err.(viper.ConfigFileNotFoundError)
	if !configFileNotFound {
		emperror.Panic(errors.Wrap(err, "failed to read configuration"))
	}

	var config Config
	// configuration gets populated here - external configuration sources (flags, env vars) are processed into the instance
	err = viper.Unmarshal(&config)
	emperror.Panic(errors.Wrap(err, "failed to unmarshal configuration"))

	// Create logger (first thing after configuration loading)
	logger := log.NewLogger(config.Log)

	// Provide some basic context to all log lines
	logger = log.WithFields(logger, map[string]interface{}{"environment": config.Environment, "application": serviceName})

	logger.Info("initializing the application",
		map[string]interface{}{"version": Version, "commit_hash": CommitHash, "build_date": BuildDate})

	piUrl := parseCloudInfoAddress()
	transport := httptransport.New(piUrl.Host, piUrl.Path, []string{piUrl.Scheme})
	pc := client.New(transport, strfmt.Default)

	ciCli := recommender.NewCloudInfoClient(pc)

	// configure the gin validator
	err = api.ConfigureValidator(ciCli)
	emperror.Panic(err)

	buildInfo := buildinfo.New(Version, CommitHash, BuildDate)
	routeHandler := api.NewRouteHandler(buildInfo, ciCli, logger)

	// new default gin engine (recovery, logger middleware)
	router := gin.Default()

	// enable authentication if not dev-mode
	if !viper.GetBool(devModeFlag) {
		logger.Debug("enable authentication")
		signingKey := viper.GetString(tokenSigningKeyFlag)
		appRole := viper.GetString(cfgAppRole)

		routeHandler.EnableAuth(router, appRole, signingKey)
	}

	// add prometheus metric endpoint
	if config.Metrics.Enabled {
		routeHandler.EnableMetrics(router, config.Metrics.Address)
	}

	routeHandler.ConfigureRoutes(router)
	logger.Info("configured routes")

	err = router.Run(viper.GetString(listenAddressFlag))
	emperror.Panic(errors.Wrap(err, "failed to run router"))
}

func parseCloudInfoAddress() *url.URL {
	u, err := url.ParseRequestURI(viper.GetString(cloudInfoFlag))
	emperror.Panic(errors.Wrap(err, fmt.Sprintf("invalid URI: %s", viper.GetString(cloudInfoFlag))))

	return u
}
