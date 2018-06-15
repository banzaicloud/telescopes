package api

import (
	"net/http"

	"fmt"
	"github.com/banzaicloud/bank-vaults/auth"
	"github.com/banzaicloud/telescopes/productinfo"
	"github.com/banzaicloud/telescopes/recommender"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	log "github.com/sirupsen/logrus"
	"gopkg.in/go-playground/validator.v8"
)

const (
	providerParam = "provider"
	regionParam   = "region"
)

// RouteHandler struct that wraps the recommender engine
type RouteHandler struct {
	engine *recommender.Engine
	prod   *productinfo.CachingProductInfo
}

// NewRouteHandler creates a new RouteHandler and returns a reference to it
func NewRouteHandler(e *recommender.Engine, p *productinfo.CachingProductInfo) *RouteHandler {
	return &RouteHandler{
		engine: e,
		prod:   p,
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

	v := binding.Validator.Engine().(*validator.Validate)

	// set validation middlewares for request path parameter validation
	router.Use(ValidatePathParam(providerParam, v, "provider_supported"))
	router.Use(ValidateRegionData(v))

	base := router.Group("/")
	base.Use(cors.New(getCorsConfig()))
	{
		base.GET("/status", r.signalStatus)
	}

	// the v1 api group
	v1 := router.Group("/api/v1")

	// recommender api group
	recGroup := v1.Group("/recommender/")
	{
		recGroup.POST("/:provider/:region/cluster/", r.recommendClusterSetup)
	}

	// product api group
	piGroup := v1.Group("/products/")
	{
		piGroup.GET("/:provider/:region/", r.getProductInfo)
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
	provider := c.Param(providerParam)
	region := c.Param(regionParam)

	// request decorated with provider and region
	reqWr := RequestWrapper{P: provider, R: region}

	if err := c.BindJSON(&reqWr); err != nil {
		log.Errorf("failed to bind request body: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "bad_params",
			"message": "invalid zone(s) or network performance",
			"cause":   err.Error(),
		})
		return
	}

	if response, err := r.engine.RecommendCluster(provider, region, reqWr.ClusterRecommendationReq); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
	} else {
		c.JSON(http.StatusOK, *response)
	}
}

func (r *RouteHandler) getProductInfo(c *gin.Context) {
	cProv := c.GetString(providerParam)
	regIon := c.GetString(regionParam)

	log.Infof("getting product info for provider: %s, region: %s", cProv, regIon)
	c.JSON(http.StatusOK, "")

}

// RequestWrapper internal struct for passing provider/zone info to the validator
type RequestWrapper struct {
	recommender.ClusterRecommendationReq
	P string
	R string
}
