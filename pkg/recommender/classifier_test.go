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
	"github.com/go-openapi/runtime"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
)

func TestErrResponseClassifier_Classify(t *testing.T) {
	tests := []struct {
		name    string
		error   interface{}
		checker func(t *testing.T, er errorResponse, e error)
	}{
		{
			name:  "url error - cloud info service unavailable",
			error: emperror.With(&url.Error{}, cloudInfoCliErrTag),
			checker: func(t *testing.T, er errorResponse, e error) {
				assert.Nil(t, e, "could not create classifier")
				assert.Equal(t, http.StatusInternalServerError, er.HttpResponseCode, "invalid http status code")
				assert.Equal(t, ErrCloudInfoUnavailable, er.ErrorCode, "invalid error code")
			},
		},
		{
			name:  "api error - no resource available, validation",
			error: emperror.With(&runtime.APIError{Code: http.StatusBadRequest}, "validation"),
			checker: func(t *testing.T, er errorResponse, e error) {
				assert.Nil(t, e, "could not create classifier")
				assert.Equal(t, http.StatusBadRequest, er.HttpResponseCode, "invalid http status code")
				assert.Equal(t, ErrCloudInfoUnavailable, er.ErrorCode, "invalid error code")
			},
		},
		{
			name:  "api error - no resource available, recommendation",
			error: emperror.With(&runtime.APIError{Code: http.StatusBadRequest}, recommenderErrorTag),
			checker: func(t *testing.T, er errorResponse, e error) {
				assert.Nil(t, e, "could not create classifier")
				assert.Equal(t, http.StatusBadRequest, er.HttpResponseCode, "invalid http status code")
				assert.Equal(t, ErrNoCloudInfoForRequestedResources, er.ErrorCode, "invalid error code")
			},
		},
		{
			name:  "generic error -  recommendation",
			error: emperror.With(errors.New("test recommender error with context"), recommenderErrorTag),
			checker: func(t *testing.T, er errorResponse, e error) {
				assert.Nil(t, e, "could not create classifier")
				assert.Equal(t, http.StatusBadRequest, er.HttpResponseCode, "invalid http status code")
				assert.Equal(t, ErrNoCloudInfoForRequestedResources, er.ErrorCode, "invalid error code")
			},
		},
		{
			name:  "generic error -  no tags",
			error: emperror.With(errors.New("test error - no context")),
			checker: func(t *testing.T, er errorResponse, e error) {
				assert.Nil(t, e, "could not create classifier")
				assert.Equal(t, http.StatusInternalServerError, er.HttpResponseCode, "invalid http status code")
				assert.Equal(t, -1, er.ErrorCode, "invalid error code")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create the classifier
			erc := NewErrorResponseClassifier()

			// execute the classification
			rsp, e := erc.Classify(test.error)

			// check the response
			test.checker(t, rsp.(errorResponse), e)

		})
	}
}