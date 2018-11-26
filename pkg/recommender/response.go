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
	"github.com/gin-gonic/gin"
	"net/http"
)

// Responder marks the capability for respondings
type Responder interface {
	Respond(err error)
}

// errorResponder struct in charge for assembling classified error responses
type errorResponder struct {
	errClassifier Classifier
	gCtx          *gin.Context
}

// Respond assembles the error response corresponding to the passed in error
func (er *errorResponder) Respond(err error) {

	if errCode, e := er.errClassifier.Classify(err); e == nil {
		er.gCtx.JSON(http.StatusInternalServerError, gin.H{"code": errCode})
		return
	}

	er.gCtx.JSON(http.StatusInternalServerError, gin.H{"code": "unknown", "err": err})
}

// NewErrorResponder configures a new error responder
func NewErrorResponder(ginCtx *gin.Context) Responder {
	return &errorResponder{
		errClassifier: NewErrorContextClassifier(),
		gCtx:          ginCtx,
	}
}
