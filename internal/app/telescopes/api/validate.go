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
	"reflect"

	"github.com/banzaicloud/telescopes/internal/platform/classifier"
	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/gin-gonic/gin/binding"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"gopkg.in/go-playground/validator.v8"
)

const (
	ntwLow    = "low"
	ntwMedium = "medium"
	ntwHigh   = "high"
	ntwExtra  = "extra"

	categoryGeneral = "General purpose"
	categoryCompute = "Compute optimized"
	categoryMemory  = "Memory optimized"
	categoryGpu     = "GPU instance"
	categoryStorage = "Storage optimized"
)

// ConfigureValidator configures the Gin validator with custom validator functions
func ConfigureValidator(ciCli *recommender.CloudInfoClient) error {
	v := binding.Validator.Engine().(*validator.Validate)

	if err := v.RegisterValidation("networkPerf", networkPerfValidator()); err != nil {
		return emperror.Wrap(err, "could not register networkPerf validator")
	}
	if err := v.RegisterValidation("category", categoryValidator()); err != nil {
		return emperror.Wrap(err, "could not register category validator")
	}
	if err := v.RegisterValidation("continents", continentValidator(ciCli)); err != nil {
		return emperror.Wrap(err, "could not register continent validator")
	}
	return nil
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

// categoryValidator validates the category in the recommendation request.
func categoryValidator() validator.Func {
	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value,
		fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		for _, c := range []string{categoryCompute, categoryGeneral, categoryGpu, categoryMemory, categoryStorage} {
			if field.String() == c {
				return true
			}
		}
		return false
	}
}

// continentValidator validates the continent in the recommendation request.
func continentValidator(ciCli *recommender.CloudInfoClient) validator.Func {
	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value,
		fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		continents, err := ciCli.GetRegions("azure", "compute")
		if err != nil {
			return false
		}
		for _, continent := range continents {
			if field.String() == continent.Name {
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
	ciCli *recommender.CloudInfoClient
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
		return emperror.With(e, classifier.ValidationErrTag)
	}

	if e := ppV.validateService(pathParams.Provider, pathParams.Service); e != nil {
		return emperror.With(e, classifier.ValidationErrTag)
	}

	if e := ppV.validateRegion(pathParams.Provider, pathParams.Service, pathParams.Region); e != nil {
		return emperror.With(e, classifier.ValidationErrTag)
	}

	return nil
}

func (ppV *pathParamValidator) validateProvider(prv string) error {
	if ciPrv, e := ppV.ciCli.GetProvider(prv); e != nil {
		return e
	} else if ciPrv == "" {
		return errors.New("provider not found")
	}
	return nil
}

func (ppV *pathParamValidator) validateService(prv, svc string) error {
	if cis, e := ppV.ciCli.GetService(prv, svc); e != nil {
		return e
	} else if cis == "" {
		return errors.New("service not found")
	}
	return nil
}

func (ppV *pathParamValidator) validateRegion(prv, svc, region string) error {
	if ciReg, e := ppV.ciCli.GetRegion(prv, svc, region); e != nil {
		return e
	} else if ciReg == "" {
		return errors.New("region not found")
	}
	return nil
}

func NewCloudInfoValidator(ciCli *recommender.CloudInfoClient) CloudInfoValidator {
	return &pathParamValidator{ciCli}
}
