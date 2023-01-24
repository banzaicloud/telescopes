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

	"github.com/banzaicloud/telescopes/internal/platform/classifier"
	"github.com/banzaicloud/telescopes/internal/platform/errorresponse"
	"github.com/banzaicloud/telescopes/internal/platform/log"
	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/mitchellh/mapstructure"
)

// swagger:operation POST /recommender/provider/{provider}/service/{service}/region/{region}/cluster recommend recommendCluster
// ---
// summary: Provides a recommended set of node pools on a given provider in a specific region.
// description: Provides a recommended set of node pools on a given provider in a specific region.
// parameters:
//   - name: provider
//     in: path
//     description: provider
//     required: true
//   - name: service
//     in: path
//     description: service
//     required: true
//   - name: region
//     in: path
//     description: region
//     required: true
//   - name: recommendRequestBody
//     in: body
//     description: request params
//     schema:
//     "$ref": "#/definitions/recommendClusterRequest"
//     required: true
//
// responses:
//
//	"200":
//	  description: recommendation response
//	  schema:
//	    "$ref": "#/definitions/recommendationResponse"
func (r *RouteHandler) recommendCluster() gin.HandlerFunc {
	return func(c *gin.Context) {
		pathParams := GetRecommendationParams{}

		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			errorresponse.NewErrorResponder(c).Respond(emperror.Wrap(err, "failed to decode path parameters"))
			return
		}

		logger := log.WithFieldsForHandlers(c, r.log,
			map[string]interface{}{"provider": pathParams.Provider, "service": pathParams.Service, "region": pathParams.Region})

		logger.Info("recommend cluster setup")

		if err := NewCloudInfoValidator(r.ciCli).ValidatePathParams(pathParams); err != nil {
			errorresponse.NewErrorResponder(c).Respond(err)
			return
		}

		// request decorated with provider and region - used to validate the request
		req := recommender.SingleClusterRecommendationReq{}

		if err := c.BindJSON(&req); err != nil {
			errorresponse.NewErrorResponder(c).Respond(
				emperror.WrapWith(err, "failed to bind request body", classifier.ValidationErrTag))
			return
		}

		response, err := r.engine.RecommendCluster(pathParams.Provider, pathParams.Service, pathParams.Region, req, nil)
		if err != nil {
			errorresponse.NewErrorResponder(c).Respond(err)
			return
		}
		c.JSON(http.StatusOK, RecommendationResponse{*response})
	}
}

// swagger:operation PUT /recommender/provider/{provider}/service/{service}/region/{region}/cluster recommend recommendClusterScaleOut
// ---
// summary: Provides a recommendation for a scale-out, based on a current cluster layout on a given provider in a specific region.
// description: Provides a recommendation for a scale-out, based on a current cluster layout on a given provider in a specific region.
// parameters:
//   - name: provider
//     in: path
//     description: provider
//     required: true
//   - name: service
//     in: path
//     description: service
//     required: true
//   - name: region
//     in: path
//     description: region
//     required: true
//   - name: recommendRequestBody
//     in: body
//     description: request params
//     schema:
//     "$ref": "#/definitions/recommendClusterScaleOutRequest"
//     required: true
//
// responses:
//
//	"200":
//	  description: recommendation response
//	  schema:
//	    "$ref": "#/definitions/recommendationResponse"
func (r *RouteHandler) recommendClusterScaleOut() gin.HandlerFunc {
	return func(c *gin.Context) {
		pathParams := GetRecommendationParams{}

		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			errorresponse.NewErrorResponder(c).Respond(emperror.Wrap(err, "failed to decode path parameters"))
			return
		}

		logger := log.WithFieldsForHandlers(c, r.log,
			map[string]interface{}{"provider": pathParams.Provider, "service": pathParams.Service, "region": pathParams.Region})

		logger.Info("recommend cluster scale out")

		if e := NewCloudInfoValidator(r.ciCli).ValidatePathParams(pathParams); e != nil {
			errorresponse.NewErrorResponder(c).Respond(e)
			return
		}

		req := recommender.ClusterScaleoutRecommendationReq{}

		if err := c.BindJSON(&req); err != nil {
			errorresponse.NewErrorResponder(c).Respond(
				emperror.WrapWith(err, "failed to bind request body", classifier.ValidationErrTag))
			return
		}

		response, err := r.engine.RecommendClusterScaleOut(pathParams.Provider, pathParams.Service, pathParams.Region, req)
		if err != nil {
			errorresponse.NewErrorResponder(c).Respond(err)
			return
		}
		c.JSON(http.StatusOK, RecommendationResponse{*response})
	}
}

// swagger:operation POST /recommender/multicloud recommend recommendMultiCluster
// ---
// summary: Provides a recommended set of node pools on a given provider in a specific region.
// description: Provides a recommended set of node pools on a given provider in a specific region.
// parameters:
//   - name: recommendRequestBody
//     in: body
//     description: request params
//     schema:
//     "$ref": "#/definitions/recommendMultiClusterRequest"
//     required: true
//
// responses:
//
//	"200":
//	  description: recommendation response
//	  schema:
//	    "$ref": "#/definitions/recommendationResponse"
func (r *RouteHandler) recommendMultiCluster() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := log.WithFieldsForHandlers(c, r.log, map[string]interface{}{})

		logger.Info("recommend cluster setup")

		req := recommender.MultiClusterRecommendationReq{}
		if err := c.BindJSON(&req); err != nil {
			errorresponse.NewErrorResponder(c).Respond(
				emperror.WrapWith(err, "failed to bind request body", classifier.ValidationErrTag))
			return
		}

		if err := NewCloudInfoValidator(r.ciCli).ValidateContinents(req.Continents); err != nil {
			errorresponse.NewErrorResponder(c).Respond(emperror.With(err, classifier.ValidationErrTag))
			return
		}

		response, err := r.engine.RecommendMultiCluster(req)
		if err != nil {
			errorresponse.NewErrorResponder(c).Respond(err)
			return
		}

		c.JSON(http.StatusOK, response)
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
