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
	"reflect"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo-client/client/regions"
	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/gin-gonic/gin/binding"
	"github.com/goph/emperror"
	"gopkg.in/go-playground/validator.v8"
)

const (
	ntwLow    = "low"
	ntwMedium = "medium"
	ntwHigh   = "high"
	ntwExtra  = "extra"

	validationErrTag = "validation"
	providerErrTag   = "provider"
	serviceErrTag    = "service"
	regionErrTag     = "region"
)

// ConfigureValidator configures the Gin validator with custom validator functions
func ConfigureValidator(ctx context.Context, pc *recommender.CloudInfoClient) error {
	v := binding.Validator.Engine().(*validator.Validate)

	if err := v.RegisterValidation("zone", zoneValidator(pc)); err != nil {
		return emperror.Wrap(err, "could not register zone validator")
	}

	if err := v.RegisterValidation("network", networkPerfValidator()); err != nil {
		return emperror.Wrap(err, "could not register network validator")
	}
	return nil
}

// zoneValidator validates the zone in the recommendation request.
func zoneValidator(pc *recommender.CloudInfoClient) validator.Func {
	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value,
		fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {

		provider := reflect.Indirect(topStruct).FieldByName("Provider").String()
		svc := reflect.Indirect(topStruct).FieldByName("Service").String()
		region := reflect.Indirect(topStruct).FieldByName("Region").String()

		response, _ := pc.Regions.GetRegion(regions.NewGetRegionParams().
			WithProvider(provider).
			WithService(svc).
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

// CloudInfoValidator contract for validating cloud info data
type CloudInfoValidator interface {
	// Validate checks the existence, correctness etc... of the parameters
	Validate(params interface{}) error
}

type pathParamValidator struct {
	piCli *recommender.CloudInfoClient
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
		return emperror.With(e, validationErrTag, providerErrTag)
	}

	if e := ppV.validateService(pathParams.Provider, pathParams.Service); e != nil {
		return emperror.With(e, validationErrTag, serviceErrTag)
	}

	if e := ppV.validateRegion(pathParams.Provider, pathParams.Service, pathParams.Region); e != nil {
		return emperror.With(e, validationErrTag, regionErrTag)
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

func NewCloudInfoValidator(ciCli *recommender.CloudInfoClient) CloudInfoValidator {
	return &pathParamValidator{ciCli}
}
