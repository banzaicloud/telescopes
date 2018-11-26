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

type ErrorResponder interface {
	Respond(err error)
}

type errorResponder struct {
	errClassifier Classifier
	gCtx          *gin.Context
}

func (er *errorResponder) Respond(err error) {
	if errCode, err := er.errClassifier.Classify(err); err != nil {
		er.gCtx.JSON(http.StatusInternalServerError, gin.H{"code": errCode})
		return
	}

	er.gCtx.JSON(http.StatusInternalServerError, gin.H{"code": "unknown", "err": err})
}

func NewErrorResponder(ginCtx *gin.Context) ErrorResponder {
	return &errorResponder{
		errClassifier: NewErrorContextClassifier(),
		gCtx:          ginCtx,
	}
}
