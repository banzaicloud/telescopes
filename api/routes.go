package api

import (
	"fmt"
	"net/http"

	"github.com/banzaicloud/cluster-recommender/recommender"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type RouteHandler struct {
	Engine *recommender.Engine
}

func NewRouteHandler(engine *recommender.Engine) *RouteHandler {
	return &RouteHandler{
		Engine: engine,
	}
}

func getCorsConfig() cors.Config {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	if !config.AllowAllOrigins {
		config.AllowOrigins = []string{"http://", "https://"}
	}
	config.AllowMethods = []string{"PUT", "DELETE", "GET", "POST", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Authorization", "Content-Type"}
	config.ExposeHeaders = []string{"Content-Length"}
	config.AllowCredentials = true
	config.MaxAge = 12
	return config
}

func (r *RouteHandler) ConfigureRoutes(router *gin.Engine) {
	log.Info("configuring routes")
	base := router.Group("/")
	base.Use(cors.New(getCorsConfig()))
	{
		base.GET("/status", r.signalStatus)
	}
	v1 := router.Group("/api/v1/")
	{
		v1.POST("/recommender/:provider/:region/cluster", r.recommendClusterSetup)
	}
}

func (r *RouteHandler) signalStatus(c *gin.Context) {
	c.JSON(http.StatusOK, "ok")
}

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
//       200: recommendationResp
func (r *RouteHandler) recommendClusterSetup(c *gin.Context) {
	log.Info("recommend cluster setup")
	provider := c.Param("provider")
	region := c.Param("region")
	var request recommender.ClusterRecommendationReq
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// TODO: validation
	if response, err := r.Engine.RecommendCluster(provider, region, request); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
	} else {
		c.JSON(http.StatusOK, *response)
	}
}
