package ec2

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/banzaicloud/cluster-recommender/productinfo"
	"github.com/stretchr/testify/assert"
)

type DummyPricingSource struct {
	TcId int
}

func (dps *DummyPricingSource) GetAttributeValues(input *pricing.GetAttributeValuesInput) (*pricing.GetAttributeValuesOutput, error) {

	// example json sequence
	//{
	//	"Value": "256 GiB"
	//},
	//{
	//"Value": "3,904 GiB"
	//},
	//{
	//"Value": "3.75 GiB"
	//},

	switch dps.TcId {
	case 1:
		return &pricing.GetAttributeValuesOutput{
			AttributeValues: []*pricing.AttributeValue{
				{
					Value: dps.strPointer("256 GiB"),
				},
				{
					Value: dps.strPointer("3,904 GiB"),
				},
				{
					Value: dps.strPointer("3.75 GiB"),
				},
			},
		}, nil
	case 2:
		return &pricing.GetAttributeValuesOutput{
			AttributeValues: []*pricing.AttributeValue{
				{
					Value: dps.strPointer("invalid float 256 GiB"),
				},
				{
					Value: dps.strPointer("3,904 GiB"),
				},
				{
					Value: dps.strPointer("3.75 GiB"),
				},
			},
		}, nil
	case 3:
		return nil, fmt.Errorf("failed to retrieve values")
	}

	return nil, nil
}
func (dps *DummyPricingSource) GetProducts(input *pricing.GetProductsInput) (*pricing.GetProductsOutput, error) {

	return nil, nil
}

// strPointer gets the pointer to the passed string
func (dps *DummyPricingSource) strPointer(str string) *string {
	return &str
}

func TestEc2Infoer_GetAttributeValues(t *testing.T) {
	tests := []struct {
		name          string
		pricingServie PricingSource
		attrName      string
		check         func(values productinfo.AttrValues, err error)
	}{
		{
			name:          "successfully retrieve attributes",
			pricingServie: &DummyPricingSource{TcId: 1},
			check: func(values productinfo.AttrValues, err error) {
				assert.Equal(t, 3, len(values), "invalid number of values returned")
				assert.Nil(t, err, "should not get error")
			},
		},
		{
			name:          "error - invalid values zeroed out",
			pricingServie: &DummyPricingSource{TcId: 2},
			check: func(values productinfo.AttrValues, err error) {
				assert.Equal(t, values[0].StrValue, "invalid float 256 GiB", "the invalid value is not the first element")
				assert.Equal(t, values[0].Value, float64(0), "the invalid value is not zeroed out")
				assert.Equal(t, 3, len(values))
			},
		},
		{
			name:          "error - error when retrieving values",
			pricingServie: &DummyPricingSource{TcId: 3},
			check: func(values productinfo.AttrValues, err error) {
				assert.Equal(t, "failed to retrieve values", err.Error())
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfoer, err := NewEc2Infoer(test.pricingServie, "", "")
			if err != nil {
				t.Fatalf("failed to create productinfoer; [%s]", err.Error())
			}

			test.check(productInfoer.GetAttributeValues(test.attrName))

		})
	}
}
