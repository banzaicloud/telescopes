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

package recommender

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-openapi/runtime"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

// Classifier represents a contract to classify passed in structs
type Classifier interface {
	// Classify classifies the passed in struct based on arbitrary, implementation specific criteria
	Classify(in interface{}) (interface{}, error)
}

// errCtxClassifier struct for classifying errors based on their context
type errCtxClassifier struct {
	// a map keyed by the error code identified by the tags in the value slice
	errorTags map[string][]string
}

const (
	// constants representing error codes
	errProductInfo    = "CLOUDINFO"
	errCloudInfoDown  = "CLOUD-INFO-N/A"
	errRecommender    = "RECOMMENDER"
	errPathValidation = "INVALIDPATH"
)

// Classify classifies the error passed in based on its context. Returns the error code corresponding to the context
func (ec *errCtxClassifier) Classify(in interface{}) (interface{}, error) {
	var (
		err error
		ok  bool
	)
	if err, ok = in.(error); !ok {
		return "", errors.New("unsupported type for classifier")
	}

	errCode, _ := ec.rank(emperror.Context(err))

	return errCode, nil
}

// NewErrorContextClassifier creates a new Classifier instance, configured with error codes and related flags
func NewErrorContextClassifier() Classifier {
	return &errCtxClassifier{
		errorTags: map[string][]string{
			errProductInfo:    []string{cloudInfoErrTag},
			errCloudInfoDown:  []string{cloudInfoCliErrTag},
			errRecommender:    []string{recommenderErrorTag},
			errPathValidation: []string{"validation"},
		},
	}
}

func (ec *errCtxClassifier) rank(flags []interface{}) (string, int) {

	var (
		errCode string
		maxRank int = -1
	)

	for errKey, errFlags := range ec.errorTags {
		currRate := ec.rate(errFlags, flags)
		if currRate > 0 && currRate > maxRank {
			maxRank = currRate
			errCode = errKey
		}
	}

	return errCode, maxRank
}

// rate computes the numeric value representing the number or error flags matched by the context flags
// the higher the computed rank the higher the possibility that the set of flags identify the right error code
func (ec *errCtxClassifier) rate(errFlags []string, ctxFlags []interface{}) int {
	rate := 0
	for _, errFlag := range errFlags {
		for _, ctxFlag := range ctxFlags {
			if errFlag == ctxFlag {
				rate = rate + 1
			}
		}
	}
	return rate
}

type errResponseClassifier struct {
}

func (erc *errResponseClassifier) Classify(in interface{}) (interface{}, error) {
	var (
		err error
		ok  bool
	)

	if err, ok = in.(error); !ok {
		return nil, errors.New("failed to classify error")
	}

	cause := errors.Cause(err)

	switch cause.(type) {

	case *runtime.APIError:
		// (cloud info) service is reachable - operation failed (eg.: bad request)
		httpCode, tcCode, tcMsq := erc.classifyApiError(cause.(*runtime.APIError), emperror.Context(err))

		return NewErrorResponse(httpCode, tcCode, tcMsq), nil
	case *url.Error:
		// the cloud info service is not available
		httpCode, tcCode, tcMsq := erc.classifyUrlError(cause.(*url.Error), emperror.Context(err))

		return NewErrorResponse(httpCode, tcCode, tcMsq), nil
	default:
		httpCode, tcCode, tcMsq := erc.classifyGenericError(cause, emperror.Context(err))
		// unclassified error
		return NewErrorResponse(httpCode, tcCode, tcMsq), nil
	}

	return nil, nil
}

func NewErrorResponseClassifier() Classifier {
	return &errResponseClassifier{}
}

type ErrorResponse struct {
	HttpResponseCode int    `json:"http_response_code"`
	ErrorCode        int    `json:"error_code"`
	Message          string `json:"message"`
}

func NewErrorResponse(rCode, eCode int, message string) ErrorResponse {
	return ErrorResponse{
		HttpResponseCode: rCode,
		ErrorCode:        eCode,
		Message:          message,
	}
}

func (erc *errResponseClassifier) classifyApiError(e *runtime.APIError, ctx []interface{}) (int, int, string) {

	var (
		httpCode int    = -1
		tcCode   int    = -1
		tcMsg    string = "unknown failure"
	)

	// determine http status code
	switch c := e.Code; {
	case c < 500:
		// all non-server error status codes translated to user error staus code
		httpCode = 400
	default:
		// all server errors left unchanged
		httpCode = c
	}

	// determine error code and status message - from the error and the context
	// the message should contain the flow related information and
	tcCode, tcMsg = erc.computeCodeAndMsg(e, ctx)

	return httpCode, tcCode, tcMsg
}

func (erc *errResponseClassifier) classifyUrlError(e *url.Error, ctx []interface{}) (int, int, string) {

	var (
		httpCode int    = http.StatusInternalServerError
		tcCode   int    = -1
		tcMsg    string = "unknown failure"
	)

	if has(ctx, cloudInfoCliErrTag) {
		return httpCode, 2000, fmt.Sprint("failed to connect to cloud info service") // connectivity to CI service
	}

	return httpCode, tcCode, tcMsg
}

func (erc *errResponseClassifier) computeCodeAndMsg(e *runtime.APIError, ctx []interface{}) (int, string) {

	if has(ctx, "validation") {
		// todo enrich the message with more information (path parameter, etc ...)
		return 1000, fmt.Sprint("validation failed")
	}

	if has(ctx, recommenderErrorTag) {
		// todo enrich the message with more information
		return 5000, fmt.Sprint("recommendation failed")
	}

	return 0, ""
}

func (erc *errResponseClassifier) classifyGenericError(e error, ctx []interface{}) (int, int, string) {

	if has(ctx, recommenderErrorTag) {
		// todo enrich the message with more information
		return http.StatusBadRequest, 5000, fmt.Sprint("recommendation failed")
	}

	return 500, -1, "recommendation failed"
}

func has(slice []interface{}, s interface{}) bool {
	for _, e := range slice {
		if e == s {
			return true
		}
	}
	return false
}
