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
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestClassifyErrorContext(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		check func(t *testing.T, code string, err error)
	}{
		{
			name: "productinfo errorcode",
			err:  emperror.With(errors.New("test"), cloudInfoErrTag, cloudInfoCliErrTag),
			check: func(t *testing.T, code string, err error) {
				assert.Equal(t, errProductInfo, code, "unexpected error code by classifier")
			},
		},
		{
			name: "no error code",
			err:  errors.New("test"),
			check: func(t *testing.T, code string, err error) {
				assert.Equal(t, "", code, "unexpected error code by classifier")
			},
		},
		{
			name: "recommender error code",
			err:  emperror.With(errors.New("test"), recommenderErrorTag),
			check: func(t *testing.T, code string, err error) {
				assert.Equal(t, errRecommender, code, "unexpected error code by classifier")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := NewErrorContextClassifier()
			code, err := c.Classify(test.err)
			test.check(t, fmt.Sprintf("%s", code), err)
		})
	}
}

func TestErrCtxClassifier_rank(t *testing.T) {
	tests := []struct {
		name    string
		flags   []interface{}
		checker func(t *testing.T, code string, rank int)
	}{
		{
			name:  "context rank productinfo flags",
			flags: []interface{}{cloudInfoCliErrTag},
			checker: func(t *testing.T, code string, rank int) {
				assert.Equal(t, code, errProductInfo, "unexpected code")
				assert.Equal(t, rank, 1, "unexpected rank")
			},
		},
		{
			name:  "context rank productinfo flags",
			flags: []interface{}{cloudInfoErrTag, cloudInfoCliErrTag},
			checker: func(t *testing.T, code string, rank int) {
				assert.Equal(t, code, errProductInfo, "unexpected code")
				assert.Equal(t, rank, 2, "unexpected rank")
			},
		},
		{
			name:  "context rank recommender flags",
			flags: []interface{}{recommenderErrorTag},
			checker: func(t *testing.T, code string, rank int) {
				assert.Equal(t, code, errRecommender, "unexpected code")
				assert.Equal(t, rank, 1, "unexpected rank")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, _ := NewErrorContextClassifier().(*errCtxClassifier)
			code, rank := c.rank(test.flags)
			test.checker(t, code, rank)
		})
	}
}
