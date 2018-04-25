package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/banzaicloud/cluster-recommender/recommender"
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

func (r *RouteHandler) ConfigureRoutes(router *gin.Engine) {
	log.Info("configuring routes")
	base := router.Group("/")
	{
		base.GET("/status", r.signalStatus)
	}
	v1 := router.Group("/api/v1/")
	{
		v1.GET("/recommender/:provider/:region/vm", r.recommendInstanceTypes)
		v1.POST("/recommender/:provider/:region/cluster", r.recommendClusterSetup)
	}
}

func (r *RouteHandler) signalStatus(c *gin.Context) {
	c.JSON(http.StatusOK, "ok")
}

func (r *RouteHandler) recommendInstanceTypes(c *gin.Context) {
	log.Info("recommend spot instance types")
	region := c.Param("region")
	baseInstanceType := c.DefaultQuery("baseInstanceType", "m4.xlarge")
	azsQuery := c.DefaultQuery("availabilityZones", "")
	var azs []string
	if azsQuery == "" {
		azs = nil
	} else {
		azs = strings.Split(azsQuery, ",")
	}
	if response, err := r.Engine.RetrieveRecommendation(region, azs, baseInstanceType); err != nil {
		// TODO: handle different error types
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
	} else {
		c.JSON(http.StatusOK, response)
	}
}

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
