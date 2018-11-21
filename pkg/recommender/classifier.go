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
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

type Classifier interface {
	Classify(in interface{}) (string, error)
}

type errCtxClassifier struct {
	// a map keyed by the error code identified by the tags in the value slice
	errorTags map[string][]string
}

const (
	errPiClient = "PRODUCTINFO"
	errRec      = "RECOMMENDER"
)

// Classify classifies the error passed in based on its context. Returns the error code corresponding to the context
func (ec *errCtxClassifier) Classify(in interface{}) (string, error) {
	var (
		err error
		ok  bool
	)
	if err, ok = in.(error); !ok {
		return "", errors.New("unsupported type for classifier")
	}

	for k, vs := range ec.errorTags {

	}
	for i, v := range emperror.Context(err) {
		fmt.Printf("key: %v, value: %v \n", i, v)
	}

	return "", nil
}

func NewErrorContextClassifier() Classifier {
	return &errCtxClassifier{
		errorTags: map[string][]string{
			errPiClient: []string{productInfoErroTag, productInfoCliTag},
			errRec:      []string{recommenderTag},
		},
	}
}
