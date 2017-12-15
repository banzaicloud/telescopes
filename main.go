package main

import (
	"github.com/banzaicloud/spot-recommender/api"
	"github.com/sirupsen/logrus"
	"github.com/gin-gonic/gin"
)

var log = logrus.New().WithField("package", "main")

func main() {
	router := gin.Default()
	log.Info("Initialized gin router")
	api.ConfigureRoutes(router)
	log.Info("Configured routes")
	router.Run(":9090")
}
