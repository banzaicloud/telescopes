package focus

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAttributeValues_SelectAttributeValues(t *testing.T) {
	tests := []struct {
		name   string
		values AttributeValues
		min    float64
		max    float64
		check  func(selected []float64, err error)
	}{
		{
			name:   "limits out of range - minimum higher than greatest value",
			values: AttributeValues{9.0, 5.6, 4.2, 4.0, 7},
			min:    11,
			max:    20,
			check: func(selected []float64, err error) {
				assert.NotNil(t, selected, "")
				assert.Equal(t, 9.0, selected[0], "invalid selection")
			},
		},
		{
			name:   "limits out of range - maximum lower than smallest value",
			values: AttributeValues{9.0, 5.6, 4.2, 4.0, 7.0},
			min:    1,
			max:    3,
			check: func(selected []float64, err error) {
				assert.NotNil(t, selected, "")
				assert.Equal(t, 4.0, selected[0], "invalid selection")
			},
		},
		{
			name:   "limits out of range - minimum and maximum equal",
			values: AttributeValues{9.0, 5.6, 4.2, 4.0, 7.0},
			min:    6,
			max:    6,
			check: func(selected []float64, err error) {
				assert.NotNil(t, selected, "")
				assert.Equal(t, 7.0, selected[0], "invalid selection")
			},
		},
		{
			name:   "no values between min and max - there is at least 1 higher value",
			values: AttributeValues{9.0, 8.0, 4.0, 3.0, 2.0, 1.0},
			min:    5,
			max:    7,
			check: func(selected []float64, err error) {
				assert.NotNil(t, selected, "")
				assert.Equal(t, 8.0, selected[0], "invalid selection")
			},
		},
		{
			name:   "empty attribute values slice",
			values: AttributeValues{},
			min:    5,
			max:    7,
			check: func(selected []float64, err error) {
				assert.Nil(t, selected, "")
				assert.NotNil(t, err, "invalid selection")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(test.values.SelectAttributeValues(test.min, test.max))
		})
	}
}
