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

	"github.com/banzaicloud/telescopes/internal/platform/classifier"
	"github.com/banzaicloud/telescopes/pkg/recommender"
	"github.com/gin-gonic/gin/binding"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/go-playground/validator/v10"
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
func ConfigureValidator() error {
	v := binding.Validator.Engine().(*validator.Validate)

	if err := v.RegisterValidation("networkPerf", networkPerfValidator()); err != nil {
		return emperror.Wrap(err, "could not register networkPerf validator")
	}
	if err := v.RegisterValidation("category", categoryValidator()); err != nil {
		return emperror.Wrap(err, "could not register category validator")
	}

	return nil
}

// networkPerfValidator validates the network performance in the recommendation request.
func networkPerfValidator() validator.Func {

	return func(fl validator.FieldLevel) bool {
		for _, n := range []string{ntwLow, ntwMedium, ntwHigh, ntwExtra} {
			if fl.Field().String() == n {
				return true
			}
		}
		return false

	}
}

// categoryValidator validates the category in the recommendation request.
func categoryValidator() validator.Func {

	return func(fl validator.FieldLevel) bool {
		for _, c := range []string{categoryCompute, categoryGeneral, categoryGpu, categoryMemory, categoryStorage} {
			if fl.Field().String() == c {
				return true
			}
		}
		return false
	}

}

// CloudInfoValidator contract for validating cloud info data
type CloudInfoValidator interface {
	// Validate checks the existence, correctness etc... of the parameters
	ValidatePathParams(params interface{}) error

	// ValidateContinents checks the existence of provided continents
	ValidateContinents(continents []string) error
}

type pathParamValidator struct {
	ciCli recommender.CloudInfoSource
}

func (ppV *pathParamValidator) ValidateContinents(continents []string) error {

	ciContinents, err := ppV.ciCli.GetContinents()
	if err != nil {

		return err
	}

	var (
		found    bool
		notfound = make([]string, 0)
	)

	for _, continent := range continents {

		found = false
		for _, ciContinent := range ciContinents {

			if continent == ciContinent {

				found = true
				continue
			}
		}

		if !found {
			notfound = append(notfound, continent)
		}
	}

	if len(notfound) > 0 {

		return errors.Errorf("unsupported continent(s) %s", notfound)
	}

	return nil
}

// Validate validates path parameters against the connected cloud info service
func (ppV *pathParamValidator) ValidatePathParams(params interface{}) error {

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

func NewCloudInfoValidator(ciCli recommender.CloudInfoSource) CloudInfoValidator {
	return &pathParamValidator{ciCli}
}
