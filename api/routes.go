package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/banzaicloud/spot-recommender/recommender"
)

var log = logrus.New().WithField("package", "api")

func ConfigureRoutes(router *gin.Engine) {
	log.Info("configuring routes")
	v1 := router.Group("/api/v1/")
	{
		v1.GET("/recommender/:region", recommendSpotInstanceTypes)
	}
}

func recommendSpotInstanceTypes(c *gin.Context) {
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
	if response, err := recommender.RecommendSpotInstanceTypes(region, azs, baseInstanceType); err != nil {
		// TODO: handle different error types
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
	} else {
		c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": response})
	}
}
