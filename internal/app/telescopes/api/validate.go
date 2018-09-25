// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"fmt"
	"github.com/banzaicloud/productinfo/pkg/productinfo-client/client/service"
	"github.com/mitchellh/mapstructure"
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
	v.RegisterValidation("service", serviceValidator(pc))
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

// ValidatePathData middleware function to validate region information in the request path
func ValidatePathData(validate *validator.Validate) gin.HandlerFunc {
	return func(c *gin.Context) {

		// extract the path parameter data into the param struct
		pathData := &GetRecommendationParams{}
		mapstructure.Decode(getPathParamMap(c), pathData)

		logrus.Debugf("region data being validated: %s", pathData)

		// invoke validation on the param struct
		err := validate.Struct(pathData)
		if err != nil {
			logrus.Errorf("validation failed. err: %s", err.Error())
			c.Abort()
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    "bad_params",
				"message": fmt.Sprintf("invalid path data: %s", pathData),
				"params":  pathData,
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

// validationFn validation logic for the region data to be registered with the validator
func regionValidator(pc *client.Productinfo) validator.Func {

	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		currentProvider := digValueForName(currentStruct, "Provider")
		currentService := digValueForName(currentStruct, "Service")
		currentRegion := digValueForName(currentStruct, "Region")

		response, err := pc.Regions.GetRegions(regions.NewGetRegionsParams().WithProvider(currentProvider).WithService(currentService))
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

// serviceValidator validates the `service` path parameter
func serviceValidator(pc *client.Productinfo) validator.Func {

	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {

		currentProvider := digValueForName(currentStruct, "Provider")
		currentService := digValueForName(currentStruct, "Service")

		svcOk, err := pc.Service.GetService(service.NewGetServiceParams().WithProvider(currentProvider).WithService(currentService))

		logrus.Debugf("svcOK: %s", svcOk)

		if err != nil {
			return false
		}

		return true
	}
}

func digValueForName(value reflect.Value, field string) string {
	var ret string
	switch value.Kind() {
	case reflect.Struct:
		ret = value.FieldByName(field).String()
	case reflect.Ptr:
		ret = value.Elem().FieldByName(field).String()
	}
	return ret
}
