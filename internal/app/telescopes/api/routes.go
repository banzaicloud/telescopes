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
	"context"
	"github.com/banzaicloud/telescopes/internal/platform"
	"net/http"
	"os"

	"github.com/banzaicloud/bank-vaults/auth"
	"github.com/banzaicloud/cloudinfo/pkg/logger"
	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const (
	// environment variable name to override base path if necessary
	appBasePath = "TELESCOPES_BASEPATH"
)

// RouteHandler struct that wraps the recommender engine
type RouteHandler struct {
	engine    *recommender.Engine
	buildInfo buildinfo.BuildInfo
	ciCli     *recommender.CloudInfoClient
}

// NewRouteHandler creates a new RouteHandler and returns a reference to it
func NewRouteHandler(e *recommender.Engine, info buildinfo.BuildInfo, ciCli *recommender.CloudInfoClient) *RouteHandler {
	return &RouteHandler{
		engine:    e,
		buildInfo: info,
		ciCli:     ciCli,
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
func (r *RouteHandler) ConfigureRoutes(ctx context.Context, router *gin.Engine) {
	ctxLog := logger.Extract(ctx)
	ctxLog.Info("configuring routes")

	basePath := "/"

	if basePathFromEnv := os.Getenv(appBasePath); basePathFromEnv != "" {
		basePath = basePathFromEnv
	}

	router.Use(logger.MiddlewareCorrelationId())
	router.Use(logger.Middleware())
	router.Use(cors.New(getCorsConfig()))

	base := router.Group(basePath)
	{
		base.GET("/status", r.signalStatus)
		base.GET("/version", r.versionHandler)
	}

	v1 := base.Group("/api/v1")

	recGroup := v1.Group("/recommender")
	{
		recGroup.POST("/:provider/:service/:region/cluster", r.recommendClusterSetup(ctx))
		recGroup.PUT("/:provider/:service/:region/cluster", r.recommendClusterScaleOut(ctx))
	}
}

// EnableAuth enables authentication middleware
func (r *RouteHandler) EnableAuth(router *gin.Engine, role string, sgnKey string) {
	router.Use(auth.JWTAuth(auth.NewVaultTokenStore(role), sgnKey, nil))
}

func (r *RouteHandler) signalStatus(c *gin.Context) {
	c.JSON(http.StatusOK, "ok")
}
