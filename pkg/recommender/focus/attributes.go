package focus

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"math"
	"sort"
)

// AttributeValueSelector interface comprises attribute selection algorythm entrypoints
type AttributeValueSelector interface {
	// SelectAttributeValues selects a range of attributes from the given
	SelectAttributeValues(min float64, max float64, focus Focuser) ([]float64, error)
}

// Focuser interface for decoupling attribute selection strategies
type Focuser interface {
	// Focus returns the attribute selection strategy
	Focus() string
}

// AttributeValues type representing a slice of attribute values
type AttributeValues []float64

// sort method for sorting this slice in ascending order
func (av AttributeValues) sort() {
	sort.Float64s(av)
}

// SelectAttributeValues selects values between the min and max values considering the focus strategy
// When the interval between min and max is "out of range" with respect to this slice the lowest or highest values are returned
func (av AttributeValues) SelectAttributeValues(min float64, max float64, focus Focuser) ([]float64, error) {
	logrus.Debugf("selecting attributes from %f, min [%f], max [%f]", av, min, max)
	if len(av) == 0 {
		return nil, fmt.Errorf("empty attribute values")
	}

	var selectedValues []float64
	// sort the slice in increasing order
	av.sort()
	logrus.Debugf("sorted attributes: [%f]", av)

	// min is greater than the highest value
	if min > av[len(av)-1] {
		// return the highest value
		return []float64{av[len(av)-1]}, nil
	}

	// max is less than the lowest value
	if max < av[0] {
		// return the lowest value
		return []float64{av[0]}, nil
	}

	var (
		// vars representing "distances" to the max from the "left" and right
		lDist, rDist = math.MaxFloat64, math.MaxFloat64
		// indexes of the "closest" values to max in the values slice
		rIdx, lIdx = -1, -1
	)

	for i, v := range av {

		if v < max {
			// distance to from max from "left"
			if lDist > max-v {
				lDist = max - v
				lIdx = i
			}
		} else {
			// distance to from max from "right"
			if rDist > v-max {
				rDist = v - max
				rIdx = i
			}
		}

		if min <= v && v <= max {
			logrus.Debugf("found value between min[%f]-max[%f]: [%f], index: [%d]", min, max, v, i)
			selectedValues = append(selectedValues, v)
		}
	}

	logrus.Debugf("lower-closest index: [%d], higher-closest index: [%d]", lIdx, rIdx)
	if len(selectedValues) == 0 {
		// there are no values between the two limits
		// todo currently the "higher-closest" value is only considered when no values found in the interval
		if rIdx > -1 {
			// if there is higher values than the max, return the closest to it
			return []float64{av[rIdx]}, nil
		}

		return nil, fmt.Errorf("could not find any attribute values")
	}
	return selectedValues, nil
}
