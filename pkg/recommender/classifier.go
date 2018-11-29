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

// errResponseClassifier type implementing the Classifier interface
type errResponseClassifier struct {
}

// NewErrorResponseClassifier returns a reference to a classifier instance
func NewErrorResponseClassifier() Classifier {
	return &errResponseClassifier{}
}

// Classify classification implementation based on the error and its context
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

// classifyApiError assembles data to be sent in the response to the caller when the error originates from the cloud info service
func (erc *errResponseClassifier) classifyApiError(e *runtime.APIError, ctx []interface{}) (int, int, string) {

	var (
		httpCode int    = -1
		tcCode   int    = -1
		tcMsg    string = "unknown failure"
	)

	// determine http status code
	switch c := e.Code; {
	case c < http.StatusInternalServerError:
		// all non-server error status codes translated to user error staus code
		httpCode = http.StatusBadRequest
	default:
		// all server errors left unchanged
		httpCode = c
	}

	// determine error code and status message - from the error and the context
	// the message should contain the flow related information and
	if has(ctx, "validation") {
		// provider, service, region - path data
		tcCode = 1000
		tcMsg = "validation failed - no cloud information available for the request path data"
	}

	if has(ctx, recommenderErrorTag) {
		// zone, network etc ..
		tcCode = 5000
		tcMsg = "recommendation failed - no cloud info available for the requested resources"
	}

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
