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
	"strings"

	"github.com/banzaicloud/telescopes/internal/app/telescopes/api"
	"github.com/banzaicloud/telescopes/internal/platform/buildinfo"
	"github.com/banzaicloud/telescopes/internal/platform/log"
	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/banzaicloud/telescopes/pkg/recommender/nodepools"
	"github.com/banzaicloud/telescopes/pkg/recommender/vms"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Provisioned by ldflags
// nolint: gochecknoglobals
var (
	version    string
	commitHash string
	buildDate  string
	branch     string
)

func main() {

	// read configuration (commandline, env etc)
	Configure(viper.GetViper(), pflag.CommandLine)

	// parse the command line
	pflag.Parse()

	if viper.GetBool("help") {
		pflag.Usage()
		return
	}

	err := viper.ReadInConfig()
	_, configFileNotFound := err.(viper.ConfigFileNotFoundError)
	if !configFileNotFound {
		emperror.Panic(errors.Wrap(err, "failed to read configuration"))
	}

	var config configuration
	// configuration gets populated here - external configuration sources (flags, env vars) are processed into the instance
	err = viper.Unmarshal(&config)
	emperror.Panic(errors.Wrap(err, "failed to unmarshal configuration"))

	// Create logger (first thing after configuration loading)
	logger := log.NewLogger(config.Log)

	logger.Info("initializing the application",
		map[string]interface{}{"version": version, "commit_hash": commitHash, "build_date": buildDate, "cloudInfoAddress": config.Cloudinfo.Address, "environment": config.Environment})

	piUrl := parseCloudInfoAddress(config.Cloudinfo.Address)
	ciCli := recommender.NewCloudInfoClient(piUrl.String(), logger)

	// configure the gin validator
	err = api.ConfigureValidator()
	emperror.Panic(err)

	vmSelector := vms.NewVmSelector(logger)
	nodePoolSelector := nodepools.NewNodePoolSelector(logger)
	engine := recommender.NewEngine(logger, ciCli, vmSelector, nodePoolSelector)

	buildInfo := buildinfo.New(version, commitHash, buildDate, branch)
	routeHandler := api.NewRouteHandler(engine, buildInfo, ciCli, logger)

	// new gin engine (recovery, logger middleware) with adapted logger
	router := gin.New()
	router.Use(log.GinLogger(logger), gin.Recovery())

	// enable authentication if not dev-mode

	if !config.App.DevMode {
		logger.Debug("enable authentication")
		//appRole := viper.GetString(cfgAppRole)
		//routeHandler.EnableAuth(router, appRole, config.App.Vault.TokenSigningKey)
	}

	// add prometheus metric endpoint
	if config.Metrics.Enabled {
		routeHandler.EnableMetrics(router, config.Metrics.Address)
	}

	routeHandler.ConfigureRoutes(router)
	logger.Info("configured routes")

	err = router.Run(config.App.Address)
	emperror.Panic(errors.Wrap(err, "failed to run router"))
}

func parseCloudInfoAddress(ciUrl string) *url.URL {
	ciUrl = strings.TrimSuffix(ciUrl, "/")
	u, err := url.ParseRequestURI(ciUrl)
	emperror.Panic(errors.Wrap(err, fmt.Sprintf("invalid URI: %s", ciUrl)))
	return u
}
