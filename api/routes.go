package api

import (
	"fmt"
	"net/http"

	"github.com/banzaicloud/bank-vaults/auth"
	"github.com/banzaicloud/telescopes/productinfo"
	"github.com/banzaicloud/telescopes/recommender"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	log "github.com/sirupsen/logrus"
	"gopkg.in/go-playground/validator.v8"
	"os"
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

	basePath := "/"
	if basePathFromEnv := os.Getenv("BASEPATH"); basePathFromEnv != "" {
		basePath = basePathFromEnv
	}

	router.Use(static.Serve(basePath, static.LocalFile("./ui/dist/ui", true)))

	base := router.Group(basePath)
	{
		base.GET("/status", r.signalStatus)
		base.Use(cors.New(getCorsConfig()))
	}

	// the v1 api group
	v1 := base.Group("/api/v1")
	// set validation middlewares for request path parameter validation
	v1.Use(ValidatePathParam(providerParam, v, "provider_supported"))

	// recommender api group
	recGroup := v1.Group("/recommender")
	{
		recGroup.Use(ValidateRegionData(v))
		recGroup.POST("/:provider/:region/cluster/", r.recommendClusterSetup)
	}

	// product api group
	piGroup := v1.Group("/products")
	{
		piGroup.Use(ValidateRegionData(v))
		piGroup.GET("/:provider/:region/", r.getProductDetails)
	}

	metaGroup := v1.Group("/regions")
	{
		metaGroup.GET("/:provider", r.getRegions)
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

// swagger:route GET /products/:provider/:region products getProductDetails
//
// Provides a list of available machine types on a given provider in a specific region.
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Security:
//
//     Responses:
//       200: productDetailsResponse
func (r *RouteHandler) getProductDetails(c *gin.Context) {
	cProv := c.Param(providerParam)
	regIon := c.Param(regionParam)

	log.Infof("getting product details for provider: %s, region: %s", cProv, regIon)

	var err error
	if details, err := r.prod.GetProductDetails(cProv, regIon); err == nil {
		log.Debugf("successfully retrieved product details:  %s, region: %s", cProv, regIon)
		c.JSON(http.StatusOK, *details)
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
}

// swagger:route GET /regions/:provider regions getRegions
//
// Provides the list of available regions of a cloud provider
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Security:
//
//     Responses:
//       200: regionsResponse
func (r *RouteHandler) getRegions(c *gin.Context) {
	provider := c.Param("provider")

	if response, err := r.engine.GetRegions(provider); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
	} else {
		c.JSON(http.StatusOK, response)
	}
}

// RequestWrapper internal struct for passing provider/zone info to the validator
type RequestWrapper struct {
	recommender.ClusterRecommendationReq
	P string
	R string
}
