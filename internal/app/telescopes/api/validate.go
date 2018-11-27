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
	"context"
	"errors"
	"fmt"
	"github.com/banzaicloud/cloudinfo/pkg/logger"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo-client/client/service"
	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/goph/emperror"
	"github.com/mitchellh/mapstructure"
	"net/http"
	"reflect"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo-client/client/providers"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo-client/client/regions"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"gopkg.in/go-playground/validator.v8"
)

const (
	ntwLow    = "low"
	ntwMedium = "medium"
	ntwHigh   = "high"
	ntwExtra  = "extra"

	validationErrTag = "validation"
)

// ConfigureValidator configures the Gin validator with custom validator functions
func ConfigureValidator(ctx context.Context, pc *recommender.CloudInfoClient) error {
	v := binding.Validator.Engine().(*validator.Validate)

	if err := v.RegisterValidation("zone", zoneValidator(pc)); err != nil {
		return fmt.Errorf("could not register zone validator. error: %s", err)
	}

	if err := v.RegisterValidation("network", networkPerfValidator()); err != nil {
		return fmt.Errorf("could not register network validator. error: %s", err)
	}
	return nil
}

// ValidatePathData middleware function to validate provider, service and region information in the request path
func ValidatePathData(ctx context.Context, validate *validator.Validate) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctxLog := logger.Extract(ctx)

		// extract the path parameter data into the param struct
		pathData := &GetRecommendationParams{}
		if err := mapstructure.Decode(getPathParamMap(c), pathData); err != nil {
			ctxLog.WithError(err).Error("validation failed")
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    "bad_params",
				"message": fmt.Sprintf("invalid path data: %s", pathData),
				"params":  pathData,
			})
			return
		}

		ctxLog.Debugf("path data being validated: %s", pathData)

		// invoke validation on the param struct
		err := validate.Struct(pathData)
		if err != nil {
			ctxLog.WithError(err).Error("validation failed")
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

func providerValidator(ctx context.Context, pc *recommender.CloudInfoClient) validator.Func {
	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		cProviders, err := pc.Providers.GetProviders(providers.NewGetProvidersParams())
		if err != nil {
			logger.Extract(ctx).WithError(err).Error("failed to get providers")
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

// regionValidator validates the zone in the recommendation request.
func regionValidator(ctx context.Context, pc *recommender.CloudInfoClient) validator.Func {

	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		currentProvider := digValueForName(currentStruct, "Provider")
		currentService := digValueForName(currentStruct, "Service")
		currentRegion := digValueForName(currentStruct, "Region")

		ctxWithFields := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(currentProvider).
			WithService(currentService).
			Build())
		ctxLog := logger.Extract(ctxWithFields)

		response, err := pc.Regions.GetRegions(regions.NewGetRegionsParams().WithProvider(currentProvider).WithService(currentService))
		if err != nil {
			ctxLog.WithError(err).Error("could not get regions")
			return false
		}

		ctxLog.Debugf("current region: %s, regions: %#v", currentRegion, response.Payload)
		for _, r := range response.Payload {
			if r.ID == currentRegion {
				return true
			}
		}
		return false
	}
}

// zoneValidator validates the zone in the recommendation request.
func zoneValidator(pc *recommender.CloudInfoClient) validator.Func {
	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value,
		fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {

		provider := reflect.Indirect(topStruct).FieldByName("Provider").String()
		service := reflect.Indirect(topStruct).FieldByName("Service").String()
		region := reflect.Indirect(topStruct).FieldByName("Region").String()

		response, _ := pc.Regions.GetRegion(regions.NewGetRegionParams().
			WithProvider(provider).
			WithService(service).
			WithRegion(region))
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
func serviceValidator(ctx context.Context, pc *recommender.CloudInfoClient) validator.Func {

	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {

		currentProvider := digValueForName(currentStruct, "Provider")
		currentService := digValueForName(currentStruct, "Service")

		ctxWithFields := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(currentProvider).
			Build())
		ctxLog := logger.Extract(ctxWithFields)

		svcOk, err := pc.Service.GetService(service.NewGetServiceParams().WithProvider(currentProvider).WithService(currentService))
		if err != nil {
			ctxLog.WithError(err).Error("could not get services")
			return false
		}

		ctxLog.Debugf("current service: %s", svcOk)

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

// CloudInfoValidator contract for validating cloud info data
type CloudInfoValidator interface {
	// Validate checks the existence, correctness etc... of the parameters
	Validate(params interface{}) error
}

type pathParamValidator struct {
	piCli *recommender.ProductInfoClient
}

// Validate validates path parameters against the connected cloud info service
func (ppV *pathParamValidator) Validate(params interface{}) error {

	var (
		pathParams GetRecommendationParams
		ok         bool
	)

	if pathParams, ok = params.(GetRecommendationParams); !ok {
		return errors.New("invalid path params")
	}

	if e := ppV.validateProvider(pathParams.Provider); e != nil {
		return emperror.With(e, validationErrTag)
	}

	if e := ppV.validateService(pathParams.Provider, pathParams.Service); e != nil {
		return emperror.With(e, validationErrTag)
	}

	if e := ppV.validateRegion(pathParams.Provider, pathParams.Service, pathParams.Region); e != nil {
		return emperror.With(e, validationErrTag)
	}

	return nil
}

func (ppV *pathParamValidator) validateProvider(prv string) error {
	if ciPrv, e := ppV.piCli.GetProvider(prv); e != nil {
		return e
	} else if ciPrv == "" {
		return errors.New("provider not found")
	}
	return nil
}

func (ppV *pathParamValidator) validateService(prv, svc string) error {
	if cis, e := ppV.piCli.GetService(prv, svc); e != nil {
		return e
	} else if cis == "" {
		return errors.New("service not found")
	}
	return nil
}

func (ppV *pathParamValidator) validateRegion(prv, svc, region string) error {
	if ciReg, e := ppV.piCli.GetRegion(prv, svc, region); e != nil {
		return e
	} else if ciReg == "" {
		return errors.New("region not found")
	}
	return nil
}

func NewCloudInfoValidator(ciCli *recommender.ProductInfoClient) CloudInfoValidator {
	return &pathParamValidator{ciCli}
}
