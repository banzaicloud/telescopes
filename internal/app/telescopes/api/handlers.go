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
	"fmt"
	"github.com/banzaicloud/telescopes/internal/platform/log"
	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/mitchellh/mapstructure"
	"net/http"
)

// swagger:route POST /recommender/{provider}/{service}/{region}/cluster recommend recommendClusterSetup
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
func (r *RouteHandler) recommendClusterSetup() gin.HandlerFunc {
	return func(c *gin.Context) {
		pathParams := GetRecommendationParams{}

		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("%s", err)})
			return
		}

		logger := log.WithFieldsForHandlers(c, r.log,
			map[string]interface{}{"provider": pathParams.Provider, "service": pathParams.Service, "region": pathParams.Region})

		logger.Info("recommend cluster setup")

		if e := NewCloudInfoValidator(r.ciCli).Validate(pathParams); e != nil {
			recommender.NewErrorResponder(c).Respond(e)
			return
		}

		// request decorated with provider and region - used to validate the request
		req := recommender.ClusterRecommendationReq{}

		if err := c.BindJSON(&req); err != nil {
			logger.Error(emperror.Wrap(err, "failed to bind request body").Error())
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    "bad_params",
				"message": "validation failed",
				"cause":   err.Error(),
			})
			return
		}

		if response, err := r.engine.RecommendCluster(pathParams.Provider, pathParams.Service, pathParams.Region, req, nil, logger); err != nil {
			recommender.NewErrorResponder(c).Respond(err)
			return
		} else {
			c.JSON(http.StatusOK, RecommendationResponse{*response})
		}
	}
}

// swagger:route PUT /recommender/{provider}/{service}/{region}/cluster recommend recommendClusterScaleOut
//
// Provides a recommendation for a scale-out, based on a current cluster layout on a given provider in a specific region.
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
func (r *RouteHandler) recommendClusterScaleOut() gin.HandlerFunc {
	return func(c *gin.Context) {
		pathParams := GetRecommendationParams{}

		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("%s", err)})
			return
		}

		logger := log.WithFieldsForHandlers(c, r.log,
			map[string]interface{}{"provider": pathParams.Provider, "service": pathParams.Service, "region": pathParams.Region})

		logger.Info("recommend cluster scale out")

		if e := NewCloudInfoValidator(r.ciCli).Validate(pathParams); e != nil {
			recommender.NewErrorResponder(c).Respond(e)
			return
		}

		req := recommender.ClusterScaleoutRecommendationReq{}

		if err := c.BindJSON(&req); err != nil {
			logger.Error(emperror.Wrap(err, "failed to bind request body").Error())
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    "bad_params",
				"message": "validation failed",
				"cause":   err.Error(),
			})
			return
		}

		if response, err := r.engine.RecommendClusterScaleOut(pathParams.Provider, pathParams.Service, pathParams.Region, req, logger); err != nil {
			recommender.NewErrorResponder(c).Respond(err)
			return
		} else {
			c.JSON(http.StatusOK, RecommendationResponse{*response})
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
