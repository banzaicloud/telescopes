package api

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/banzaicloud/telescopes/pkg/productinfo"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"
	"gopkg.in/go-playground/validator.v8"
)

// ConfigureValidator configures the Gin validator with custom validator functions
func ConfigureValidator(providers []string, pi *productinfo.CachingProductInfo) {
	// retrieve the gin validator
	v := binding.Validator.Engine().(*validator.Validate)

	// register validator for the provider parameter in the request path
	var providerString = fmt.Sprintf("^%s$", strings.Join(providers, "|"))
	var passwordRegexProvider = regexp.MustCompile(providerString)
	v.RegisterValidation("provider_supported", func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		return passwordRegexProvider.MatchString(field.String())
	})

	// register validator for the region parameter in the request path
	rd := regionData{}
	v.RegisterValidation("region", rd.validationFn(pi))

	// register validator for zones
	v.RegisterValidation("zone", ZoneValidatorFn(pi))

	// register validator for network performance
	v.RegisterValidation("network", NetworkPerfValidatorFn())

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

// ValidateRegionData middleware function to validate region information in the request path.
// It succeeds if the region is valid for the current provider
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

// regionData struct encapsulating data for region validation in the request path
type regionData struct {
	// Cloud the cloud provider from the request path
	Cloud string `binding:"required"`
	// Region the region in the request path
	Region string `binding:"region"`
}

// String representation of the path data
func (rd *regionData) String() string {
	return fmt.Sprintf("Cloud: %s, Region: %s", rd.Cloud, rd.Region)
}

// newRegionData constructs a new
func newRegionData(cloud string, region string) regionData {
	return regionData{Cloud: cloud, Region: region}
}

// validationFn validation logic for the region data to be registered with the validator
func (rd *regionData) validationFn(cpi *productinfo.CachingProductInfo) validator.Func {

	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		currentProvider := currentStruct.FieldByName("Cloud").String()
		currentRegion := currentStruct.FieldByName("Region").String()

		regions, err := cpi.GetRegions(currentProvider)
		if err != nil {
			logrus.Errorf("could not get regions for provider: %s, err: %s", currentProvider, err.Error())
		}

		logrus.Debugf("current region: %s, regions: %#v", currentRegion, regions)
		for reg := range regions {
			if reg == currentRegion {
				return true
			}
		}
		return false
	}
}

// ZoneValidatorFn validates the zone in the recommendation request.
// Returns true if the zone is valid on the current cloud provider
func ZoneValidatorFn(cpi *productinfo.CachingProductInfo) validator.Func {
	// caching product info may be available here, but the provider is not
	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value,
		fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {

		// dig out the provider and region from the "topStruct"
		cloud := reflect.Indirect(topStruct).FieldByName("P").String()
		region := reflect.Indirect(topStruct).FieldByName("R").String()

		// retrieve the zones
		zones, _ := cpi.GetZones(cloud, region)
		for _, zone := range zones {
			if zone == field.String() {
				return true
			}
		}
		return false
	}
}

// NetworkPerfValidatorFn validates the network performance in the recommendation request.
// Returns true if the network performance is valid
func NetworkPerfValidatorFn() validator.Func {
	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value,
		fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {

		switch field.String() {
		case productinfo.NTW_LOW:
			return true
		case productinfo.NTW_MEDIUM:
			return true
		case productinfo.NTW_HIGH:
			return true
		case productinfo.NTW_EXTRA:
			return true
		default:
			logrus.Errorf("could not retrieve network mapper")
			return false
		}
	}
}
