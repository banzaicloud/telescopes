package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/banzaicloud/spot-recommender/recommender"
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
		v1.GET("/recommender/:region", r.recommendSpotInstanceTypes)
	}
}

func (r *RouteHandler) signalStatus(c *gin.Context) {
	c.JSON(http.StatusOK, "ok")
}

func (r *RouteHandler) recommendSpotInstanceTypes(c *gin.Context) {
	log.Info("recommend spot instance types")
	baseInstanceType := c.DefaultQuery("baseInstanceType", "m4.xlarge")
	azsQuery := c.DefaultQuery("availabilityZones", "")
	var azs []string
	if azsQuery == "" {
		azs = nil
	} else {
		azs = strings.Split(azsQuery, ",")
	}
	if response, err := r.Engine.RetrieveRecommendation(azs, baseInstanceType); err != nil {
		// TODO: handle different error types
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
	} else {
		c.JSON(http.StatusOK, response)
	}
}
