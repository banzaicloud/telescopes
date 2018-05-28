package ec2

import (
	"testing"

	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
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
		return nil, errors.New("failed to retrieve values")
	}

	return nil, nil
}
func (dps *DummyPricingSource) GetProducts(input *pricing.GetProductsInput) (*pricing.GetProductsOutput, error) {
	switch dps.TcId {
	case 4:
		return &pricing.GetProductsOutput{
			PriceList: []aws.JSONValue{},
		}, nil
	case 5:
		return nil, errors.New("failed to retrieve values")

	}
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
			productInfoer, err := NewEc2Infoer(test.pricingServie)
			if err != nil {
				t.Fatalf("failed to create productinfoer; [%s]", err.Error())
			}

			test.check(productInfoer.GetAttributeValues(test.attrName))

		})
	}
}

func TestEc2Infoer_GetRegions(t *testing.T) {
	tests := []struct {
		name          string
		pricingServie PricingSource
		check         func(regionId map[string]string)
	}{
		{
			name:          "receive all regions",
			pricingServie: &DummyPricingSource{},
			check: func(regionId map[string]string) {
				assert.NotNil(t, regionId, "the regionId shouldn't be nil")
				assert.Equal(t, regionId, map[string]string{"ap-southeast-1": "ap-southeast-1",
					"ap-south-1": "ap-south-1", "us-west-1": "us-west-1", "us-east-1": "us-east-1", "us-east-2": "us-east-2",
					"eu-central-1": "eu-central-1", "eu-west-1": "eu-west-1", "ca-central-1": "ca-central-1",
					"eu-west-3": "eu-west-3", "ap-northeast-2": "ap-northeast-2", "ap-southeast-2": "ap-southeast-2",
					"sa-east-1": "sa-east-1", "us-west-2": "us-west-2", "ap-northeast-1": "ap-northeast-1",
					"eu-west-2": "eu-west-2"})
				assert.Equal(t, len(regionId), 15)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfoer, err := NewEc2Infoer(test.pricingServie)
			if err != nil {
				t.Fatalf("failed to create productinfoer; [%s]", err.Error())
			}

			test.check(productInfoer.GetRegions())
		})
	}
}

func TestEc2Infoer_GetProducts(t *testing.T) {
	tests := []struct {
		name          string
		regionId      string
		attrKey       string
		attrValue     productinfo.AttrValue
		pricingServie PricingSource
		check         func(vm []productinfo.Ec2Vm, err error)
	}{
		{
			name:          "successful",
			regionId:      "eu-central-1",
			attrKey:       productinfo.Cpu,
			attrValue:     productinfo.AttrValue{Value: float64(2), StrValue: productinfo.Cpu},
			pricingServie: &DummyPricingSource{TcId: 4},
			check: func(vm []productinfo.Ec2Vm, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Nil(t, vm, "the vm should be nil")
			},
		},
		{
			name:          "error - GetProducts",
			regionId:      "eu-central-1",
			attrKey:       productinfo.Cpu,
			attrValue:     productinfo.AttrValue{Value: float64(2), StrValue: productinfo.Cpu},
			pricingServie: &DummyPricingSource{TcId: 5},
			check: func(vm []productinfo.Ec2Vm, err error) {
				assert.Equal(t, err, errors.New("failed to retrieve values"))
				assert.Nil(t, vm, "the vm should be nil")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfoer, err := NewEc2Infoer(test.pricingServie)
			if err != nil {
				t.Fatalf("failed to create productinfoer; [%s]", err.Error())
			}

			test.check(productInfoer.GetProducts(test.regionId, test.attrKey, test.attrValue))
		})
	}
}

func TestEc2Infoer_GetRegion(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		pricingServie PricingSource
		check         func(region *endpoints.Region)
	}{
		{
			name:          "known region",
			id:            "eu-west-3",
			pricingServie: &DummyPricingSource{},
			check: func(region *endpoints.Region) {
				assert.Equal(t, region.Description(), "EU (Paris)")
				assert.Equal(t, region.ID(), "eu-west-3")
			},
		},
		{
			name:          "unknown region",
			id:            "testRegion",
			pricingServie: &DummyPricingSource{},
			check: func(region *endpoints.Region) {
				assert.Nil(t, region, "the region should be nil")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfoer, err := NewEc2Infoer(test.pricingServie)
			if err != nil {
				t.Fatalf("failed to create productinfoer; [%s]", err.Error())
			}

			test.check(productInfoer.GetRegion(test.id))
		})
	}
}

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name  string
		check func(config *aws.Config)
	}{
		{
			name: "success - create a new config instance",
			check: func(config *aws.Config) {
				assert.NotNil(t, config, "the config should be nil")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(NewConfig())
		})
	}
}

func TestNewPricing(t *testing.T) {
	tests := []struct {
		name  string
		cfg   *aws.Config
		check func(source PricingSource)
	}{
		{
			name: "success - create a new PricingSource",
			cfg:  NewConfig(),
			check: func(source PricingSource) {
				assert.NotNil(t, source, "the source should be nil")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(NewPricing(test.cfg))
		})
	}
}
