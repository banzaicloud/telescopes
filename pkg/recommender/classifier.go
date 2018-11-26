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
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

// Classifier represents a contract to classify passed in structs
type Classifier interface {
	// Classify classifies the passed in struct based on arbitrary, implementation specific criteria
	Classify(in interface{}) (string, error)
}

// errCtxClassifier struct for classifying errors based on their context
type errCtxClassifier struct {
	// a map keyed by the error code identified by the tags in the value slice
	errorTags map[string][]string
}

const (
	// constants representing error codes
	errProductInfo = "PRODUCTINFO"
	errRecommender = "RECOMMENDER"
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

	errCode, _ := ec.rank(emperror.Context(err))

	return errCode, nil
}

// NewErrorContextClassifier creates a new Classifier instance, configured with error codes and related flags
func NewErrorContextClassifier() Classifier {
	return &errCtxClassifier{
		errorTags: map[string][]string{
			errProductInfo: []string{productInfoErrTag, productInfoCliErrTag},
			errRecommender: []string{recommenderErrorTag},
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
