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
	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
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
func (r *RouteHandler) recommendClusterSetup(c *gin.Context) {
	logrus.Info("recommend cluster setup")

	pathParams := GetRecommendationParams{}
	err := mapstructure.Decode(getPathParamMap(c), &pathParams)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "bad_params",
			"message": "could not decode path params",
			"cause":   err.Error(),
		})
	}

	// request decorated with provider and region - used to validate the request
	req := recommender.ClusterRecommendationReq{}

	if err := c.BindJSON(&req); err != nil {
		logrus.Errorf("failed to bind request body: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "bad_params",
			"message": "validation failed",
			"cause":   err.Error(),
		})
		return
	}

	if response, err := r.engine.RecommendCluster(pathParams.Provider, pathParams.Region, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
	} else {
		c.JSON(http.StatusOK, *response)
	}
}

// getPathParamMap transforms the path params into a map to be able to easily bind to param structs
func getPathParamMap(c *gin.Context) map[string]string {
	pm := make(map[string]string)
	for _, p := range c.Params {
		pm[p.Key] = p.Value
	}
	return pm
}
