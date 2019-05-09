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

package api

import (
	"net/http"
	"os"

	"github.com/banzaicloud/bank-vaults/pkg/auth"
	ginprometheus "github.com/banzaicloud/go-gin-prometheus"
	"github.com/banzaicloud/telescopes/internal/platform/buildinfo"
	"github.com/banzaicloud/telescopes/internal/platform/log"
	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/goph/logur"
)

const (
	// environment variable name to override base path if necessary
	appBasePath = "TELESCOPES_BASEPATH"
)

// RouteHandler struct that wraps the recommender engine
type RouteHandler struct {
	engine    recommender.ClusterRecommender
	buildInfo buildinfo.BuildInfo
	ciCli     *recommender.CloudInfoClient
	log       logur.Logger
}

// NewRouteHandler creates a new RouteHandler and returns a reference to it
func NewRouteHandler(engine recommender.ClusterRecommender, info buildinfo.BuildInfo, ciCli *recommender.CloudInfoClient, log logur.Logger) *RouteHandler {
	return &RouteHandler{
		engine:    engine,
		buildInfo: info,
		ciCli:     ciCli,
		log:       log,
	}
}

func getCorsConfig() cors.Config {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	if !config.AllowAllOrigins {
		config.AllowOrigins = []string{"http://", "https://"}
	}
	config.AllowMethods = []string{http.MethodPut, http.MethodDelete, http.MethodGet, http.MethodPost, http.MethodOptions}
	config.AllowHeaders = []string{"Origin", "Authorization", "Content-Type"}
	config.ExposeHeaders = []string{"Content-Length"}
	config.AllowCredentials = true
	config.MaxAge = 12
	return config
}

// ConfigureRoutes configures the gin engine, defines the rest API for this application
func (r *RouteHandler) ConfigureRoutes(router *gin.Engine) {
	r.log.Info("configuring routes")

	basePath := "/"

	if basePathFromEnv := os.Getenv(appBasePath); basePathFromEnv != "" {
		basePath = basePathFromEnv
	}

	router.Use(log.MiddlewareCorrelationId())
	router.Use(log.Middleware())
	router.Use(cors.New(getCorsConfig()))

	base := router.Group(basePath)
	{
		base.GET("/status", r.signalStatus)
		base.GET("/version", r.versionHandler)
	}

	v1 := base.Group("/api/v1")

	recGroup := v1.Group("/recommender")
	{
		recGroup.POST("/multicloud", r.recommendMultiCluster())
		recGroup.POST("/provider/:provider/service/:service/region/:region/cluster", r.recommendCluster())
		recGroup.PUT("/provider/:provider/service/:service/region/:region/cluster", r.recommendClusterScaleOut())

		// TODO: remove legacy endpoint
		recGroup.POST("/:provider/:service/:region/cluster", r.recommendCluster())
		recGroup.PUT("/:provider/:service/:region/cluster", r.recommendClusterScaleOut())
	}
}

// EnableAuth enables authentication middleware
func (r *RouteHandler) EnableAuth(router *gin.Engine, role string, sgnKey string) {
	router.Use(auth.JWTAuth(auth.NewVaultTokenStore(role), sgnKey, nil))
}

func (r *RouteHandler) signalStatus(c *gin.Context) {
	c.JSON(http.StatusOK, "ok")
}

func (r *RouteHandler) EnableMetrics(router *gin.Engine, metricsAddr string) {
	p := ginprometheus.NewPrometheus("http", []string{"provider", "service", "region"})
	p.SetListenAddress(metricsAddr)
	p.Use(router, "/metrics")
}
