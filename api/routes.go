package api

import (
	"fmt"
	"net/http"

	"github.com/banzaicloud/bank-vaults/auth"
	"github.com/banzaicloud/telescopes/recommender"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gopkg.in/go-playground/validator.v8"
)

// RouteHandler struct that wraps the recommender engine
type RouteHandler struct {
	engine    *recommender.Engine
	validator *validator.Validate
}

// NewRouteHandler creates a new RouteHandler and returns a reference to it
func NewRouteHandler(e *recommender.Engine, v *validator.Validate) *RouteHandler {
	return &RouteHandler{
		engine:    e,
		validator: v,
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
func (r *RouteHandler) ConfigureRoutes(router *gin.Engine) {
	log.Info("configuring routes")

	router.Use(ValidatePathParam("provider", r.validator, "provider_supported"))
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

// EnableAuth enables authentication middleware
func (r *RouteHandler) EnableAuth(router *gin.Engine, role string, sgnKey string) {
	router.Use(auth.JWTAuth(auth.NewVaultTokenStore(role), sgnKey, nil))
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
	if response, err := r.engine.RecommendCluster(provider, region, request); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
	} else {
		c.JSON(http.StatusOK, *response)
	}
}
