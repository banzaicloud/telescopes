package api

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/banzaicloud/productinfo/pkg/productinfo-client/client"
	"github.com/banzaicloud/productinfo/pkg/productinfo-client/client/providers"
	"github.com/banzaicloud/productinfo/pkg/productinfo-client/client/regions"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"
	"gopkg.in/go-playground/validator.v8"
)

const (
	ntwLow    = "low"
	ntwMedium = "medium"
	ntwHigh   = "high"
	ntwExtra  = "extra"
)

// ConfigureValidator configures the Gin validator with custom validator functions
func ConfigureValidator(pc *client.Productinfo) error {
	v := binding.Validator.Engine().(*validator.Validate)
	v.RegisterValidation("provider", providerValidator(pc))
	v.RegisterValidation("region", regionValidator(pc))
	v.RegisterValidation("zone", zoneValidator(pc))
	v.RegisterValidation("network", networkPerfValidator())
	return nil
}

// ValidatePathParam is a gin middleware handler function that validates a named path parameter with specific Validate tags
func ValidatePathParam(name string, validate *validator.Validate, tags ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := c.Param(name)
		for _, tag := range tags {
			err := validate.Field(p, tag)
			if err != nil {
				logrus.Errorf("validation failed. err: %s", err.Error())
				c.Abort()
				c.JSON(http.StatusBadRequest, gin.H{
					"code":    "bad_params",
					"message": fmt.Sprintf("invalid %s parameter", name),
					"params":  map[string]string{name: p},
				})
				return
			}
		}
	}
}

// ValidateRegionData middleware function to validate region information in the request path
func ValidateRegionData(validate *validator.Validate) gin.HandlerFunc {
	const (
		providerParam = "provider"
		regionParam   = "region"
	)
	return func(c *gin.Context) {
		regionData := newRegionData(c.Param(providerParam), c.Param(regionParam))
		logrus.Debugf("region data being validated: %s", regionData)
		err := validate.Struct(regionData)
		if err != nil {
			logrus.Errorf("validation failed. err: %s", err.Error())
			c.Abort()
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    "bad_params",
				"message": fmt.Sprintf("invalid region in path: %s", regionData),
				"params":  regionData,
			})
			return
		}
	}
}

func providerValidator(pc *client.Productinfo) validator.Func {
	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		cProviders, err := pc.Providers.GetProviders(providers.NewGetProvidersParams())
		if err != nil {
			logrus.WithError(err).Errorf("failed to get providers")
			return false
		}
		for _, p := range cProviders.Payload.Providers {
			if p.Provider == field.String() {
				return true
			}
		}
		return false
	}
}

// regionData struct encapsulating data for region validation in the request path
type regionData struct {
	// Provider the cloud provider from the request path
	Provider string `binding:"required"`
	// Region the region in the request path
	Region string `binding:"region"`
}

// String representation of the path data
func (rd *regionData) String() string {
	return fmt.Sprintf("Provider: %s, Region: %s", rd.Provider, rd.Region)
}

// newRegionData constructs a new
func newRegionData(provider string, region string) regionData {
	return regionData{Provider: provider, Region: region}
}

// validationFn validation logic for the region data to be registered with the validator
func regionValidator(pc *client.Productinfo) validator.Func {

	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		currentProvider := currentStruct.FieldByName("Provider").String()
		currentRegion := currentStruct.FieldByName("Region").String()

		response, err := pc.Regions.GetRegions(regions.NewGetRegionsParams().WithProvider(currentProvider))
		if err != nil {
			logrus.WithError(err).Errorf("could not get regions for provider: %s", currentProvider)
			return false
		}

		logrus.Debugf("current region: %s, regions: %#v", currentRegion, response.Payload)
		for _, r := range response.Payload {
			if r.ID == currentRegion {
				return true
			}
		}
		return false
	}
}

// zoneValidator validates the zone in the recommendation request.
func zoneValidator(pc *client.Productinfo) validator.Func {
	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value,
		fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {

		provider := reflect.Indirect(topStruct).FieldByName("Provider").String()
		region := reflect.Indirect(topStruct).FieldByName("Region").String()
		response, _ := pc.Regions.GetRegion(regions.NewGetRegionParams().WithProvider(provider).WithRegion(region))
		for _, zone := range response.Payload.Zones {
			if zone == field.String() {
				return true
			}
		}
		return false
	}
}

// networkPerfValidator validates the network performance in the recommendation request.
func networkPerfValidator() validator.Func {
	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value,
		fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {

		for _, n := range []string{ntwLow, ntwMedium, ntwHigh, ntwExtra} {
			if field.String() == n {
				return true
			}
		}
		return false
	}
}
