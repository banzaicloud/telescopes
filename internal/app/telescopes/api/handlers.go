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
	"fmt"
	"github.com/banzaicloud/productinfo/pkg/logger"
	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	"net/http"
)

// swagger:route POST /recommender/:provider/:region/cluster recommend recommendClusterSetup
//
// Provides a recommended set of node pools on a given provider in a specific region.
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Security:
//
//     Responses:
//       200: RecommendationResponse
func (r *RouteHandler) recommendClusterSetup(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		pathParams := GetRecommendationParams{}

		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("%s", err)})
			return
		}

		ctxLog := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(pathParams.Provider).
			WithService(pathParams.Service).
			WithRegion(pathParams.Region).
			WithCorrelationId(logger.GetCorrelationId(c)).
			Build())

		log := logger.Extract(ctxLog)
		log.Info("recommend cluster setup")

		// request decorated with provider and region - used to validate the request
		req := ValidatingRequest{pathParams, recommender.ClusterRecommendationReq{}}

		if err := c.BindJSON(&req); err != nil {
			log.WithError(err).Error("failed to bind request body")
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    "bad_params",
				"message": "validation failed",
				"cause":   err.Error(),
			})
			return
		}

		if response, err := r.engine.RecommendCluster(ctxLog, pathParams.Provider, pathParams.Service, pathParams.Region, req.ClusterRecommendationReq); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
		} else {
			c.JSON(http.StatusOK, *response)
		}
	}
}

func (r *RouteHandler) versionHandler(c *gin.Context) {
	c.JSON(http.StatusOK, r.buildInfo)
}

// getPathParamMap transforms the path params into a map to be able to easily bind to param structs
func getPathParamMap(c *gin.Context) map[string]string {
	pm := make(map[string]string)
	for _, p := range c.Params {
		pm[p.Key] = p.Value
	}
	return pm
}
