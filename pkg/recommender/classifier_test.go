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
	"testing"
)

func TestClassifyErrorContext(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "productinfo tags",
			err:  emperror.With(errors.New("test"), productInfoErroTag, productInfoCliTag),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := NewErrorContextClassifier()
			c.Classify(test.err)
		})
	}
}
