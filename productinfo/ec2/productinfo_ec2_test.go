package ec2

import (
	"testing"

	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/banzaicloud/telescopes/productinfo"
	"github.com/stretchr/testify/assert"
)

type DummyPricingSource struct {
	TcId int
}

var data = priceData{
	awsData: aws.JSONValue{
		"product": map[string]interface{}{
			"attributes": map[string]interface{}{
				"instanceType":     "db.t2.small",
				productinfo.Cpu:    "1",
				productinfo.Memory: "2",
				"gpu":              "3",
			},
		},
		"terms": map[string]interface{}{
			"OnDemand": map[string]interface{}{
				"2ZP4J8GPBP6QFK3Y.JRTCKXETXF": map[string]interface{}{
					"priceDimensions": map[string]interface{}{
						"2ZP4J8GPBP6QFK3Y.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
							"pricePerUnit": map[string]interface{}{
								"USD": "5",
							},
						},
					},
				},
			},
		},
	},
}

var wrongCast = priceData{
	awsData: aws.JSONValue{
		"product": map[string]interface{}{
			"attributes": map[string]interface{}{
				"instanceType":     0,
				productinfo.Cpu:    1,
				productinfo.Memory: 2,
				"gpu":              3,
			},
		},
	},
}

var missingData = priceData{
	awsData: aws.JSONValue{
		"product": map[string]interface{}{
			"attributes": map[string]interface{}{}}}}

var missingAttributes = priceData{awsData: aws.JSONValue{
	"product": map[string]interface{}{}}}

var wrongMapCast = priceData{awsData: aws.JSONValue{
	"product": ""}}

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
			PriceList: []aws.JSONValue{
				{
					"product": map[string]interface{}{
						"attributes": map[string]interface{}{
							"instanceType":     "db.t2.small",
							productinfo.Cpu:    "1",
							productinfo.Memory: "2",
						},
					},
					"terms": map[string]interface{}{
						"OnDemand": map[string]interface{}{
							"2ZP4J8GPBP6QFK3Y.JRTCKXETXF": map[string]interface{}{
								"priceDimensions": map[string]interface{}{
									"2ZP4J8GPBP6QFK3Y.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
										"pricePerUnit": map[string]interface{}{
											"USD": "5",
										},
									},
								},
							},
						},
					},
				},
			},
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
			productInfoer, err := NewEc2Infoer(test.pricingServie, "", "")
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
			productInfoer, err := NewEc2Infoer(test.pricingServie, "", "")
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
		check         func(vm []productinfo.VmInfo, err error)
	}{
		{
			name:          "successful",
			regionId:      "eu-central-1",
			attrKey:       productinfo.Cpu,
			attrValue:     productinfo.AttrValue{Value: float64(2), StrValue: productinfo.Cpu},
			pricingServie: &DummyPricingSource{TcId: 4},
			check: func(vm []productinfo.VmInfo, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.NotNil(t, vm, "the vm should not be nil")
			},
		},
		{
			name:          "error - GetProducts",
			regionId:      "eu-central-1",
			attrKey:       productinfo.Cpu,
			attrValue:     productinfo.AttrValue{Value: float64(2), StrValue: productinfo.Cpu},
			pricingServie: &DummyPricingSource{TcId: 5},
			check: func(vm []productinfo.VmInfo, err error) {
				assert.Equal(t, err, errors.New("failed to retrieve values"))
				assert.Nil(t, vm, "the vm should be nil")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfoer, err := NewEc2Infoer(test.pricingServie, "", "")
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
			productInfoer, err := NewEc2Infoer(test.pricingServie, "", "")
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

func TestPriceData_GetInstanceType(t *testing.T) {
	tests := []struct {
		name  string
		price priceData
		check func(s string, err error)
	}{
		{
			name:  "successful",
			price: data,
			check: func(s string, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, s, "db.t2.small")
			},
		},
		{
			name:  "cast problem",
			price: wrongCast,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not cast instance type to string")
			},
		},
		{
			name:  "missing data",
			price: missingData,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get instance type")
			},
		},
		{
			name:  "missing attributes key",
			price: missingAttributes,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get map for key: [ attributes ]")
			},
		},
		{
			name:  "could not be cast to map[string]interface{}",
			price: wrongMapCast,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "the value for key: [ product ] could not be cast to map[string]interface{}")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			test.check(test.price.GetInstanceType())
		})
	}
}

func TestPriceData_GetVcpu(t *testing.T) {
	tests := []struct {
		name  string
		price priceData
		check func(s string, err error)
	}{
		{
			name:  "successful",
			price: data,
			check: func(s string, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, s, "1")
			},
		},
		{
			name:  "cast problem",
			price: wrongCast,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not cast vcpu to string")
			},
		},
		{
			name:  "missing data",
			price: missingData,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get vcpu")
			},
		},
		{
			name:  "missing attributes key",
			price: missingAttributes,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get map for key: [ attributes ]")
			},
		},
		{
			name:  "could not be cast to map[string]interface{}",
			price: wrongMapCast,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "the value for key: [ product ] could not be cast to map[string]interface{}")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(test.price.GetVcpu())
		})
	}
}

func TestPriceData_GetMem(t *testing.T) {
	tests := []struct {
		name  string
		price priceData
		check func(s string, err error)
	}{
		{
			name:  "successful",
			price: data,
			check: func(s string, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, s, "2")
			},
		},
		{
			name:  "cast problem",
			price: wrongCast,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not cast memory to string")
			},
		},
		{
			name:  "missing data",
			price: missingData,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get memory")
			},
		},
		{
			name:  "missing attributes key",
			price: missingAttributes,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get map for key: [ attributes ]")
			},
		},
		{
			name:  "could not be cast to map[string]interface{}",
			price: wrongMapCast,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "the value for key: [ product ] could not be cast to map[string]interface{}")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(test.price.GetMem())
		})
	}
}

func TestPriceData_GetGpu(t *testing.T) {
	tests := []struct {
		name  string
		price priceData
		check func(s string, err error)
	}{
		{
			name:  "successful",
			price: data,
			check: func(s string, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, s, "3")
			},
		},
		{
			name:  "cast problem",
			price: wrongCast,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not cast gpu to string")
			},
		},
		{
			name:  "missing data",
			price: missingData,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get gpu")
			},
		},
		{
			name:  "missing attributes key",
			price: missingAttributes,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get map for key: [ attributes ]")
			},
		},
		{
			name:  "could not be cast to map[string]interface{}",
			price: wrongMapCast,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "the value for key: [ product ] could not be cast to map[string]interface{}")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(test.price.GetGpu())
		})
	}
}

func TestPriceData_GetOnDemandPrice(t *testing.T) {
	tests := []struct {
		name  string
		price priceData
		check func(s string, err error)
	}{
		{
			name:  "successful",
			price: data,
			check: func(s string, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, s, "5")
			},
		},
		{
			name: "cast problem",
			price: priceData{
				awsData: aws.JSONValue{
					"terms": map[string]interface{}{
						"OnDemand": map[string]interface{}{
							"2ZP4J8GPBP6QFK3Y.JRTCKXETXF": map[string]interface{}{
								"priceDimensions": map[string]interface{}{
									"2ZP4J8GPBP6QFK3Y.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
										"pricePerUnit": map[string]interface{}{
											"USD": 5,
										}}}}}}}},
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not cast on demand price to string")
			},
		},
		{
			name: "missing data",
			price: priceData{
				awsData: aws.JSONValue{
					"terms": map[string]interface{}{
						"OnDemand": map[string]interface{}{
							"2ZP4J8GPBP6QFK3Y.JRTCKXETXF": map[string]interface{}{
								"priceDimensions": map[string]interface{}{
									"2ZP4J8GPBP6QFK3Y.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
										"pricePerUnit": map[string]interface{}{}}}}}}}},
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get on demand price")
			},
		},
		{
			name:  "could not get pricePerUnit",
			price: priceData{
				awsData: aws.JSONValue{
					"terms": map[string]interface{}{
						"OnDemand": map[string]interface{}{
							"2ZP4J8GPBP6QFK3Y.JRTCKXETXF": map[string]interface{}{
								"priceDimensions": map[string]interface{}{
									"2ZP4J8GPBP6QFK3Y.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{}}}}}}},
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get map for key: [ pricePerUnit ]")
			},
		},
		{
			name:  "could not get 2ZP4J8GPBP6QFK3Y.JRTCKXETXF.6YS6EN2CT7",
			price: priceData{
				awsData: aws.JSONValue{
					"terms": map[string]interface{}{
						"OnDemand": map[string]interface{}{
							"2ZP4J8GPBP6QFK3Y.JRTCKXETXF": map[string]interface{}{
								"priceDimensions": map[string]interface{}{}}}}}},
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get map for key: [ 2ZP4J8GPBP6QFK3Y.JRTCKXETXF.6YS6EN2CT7 ]")
			},
		},
		{
			name:  "could not get priceDimensions",
			price: priceData{
				awsData: aws.JSONValue{
					"terms": map[string]interface{}{
						"OnDemand": map[string]interface{}{
							"2ZP4J8GPBP6QFK3Y.JRTCKXETXF": map[string]interface{}{}}}}},
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get map for key: [ priceDimensions ]")
			},
		},
		{
			name:  "could not get 2ZP4J8GPBP6QFK3Y.JRTCKXETXF",
			price: priceData{
				awsData: aws.JSONValue{
					"terms": map[string]interface{}{
						"OnDemand": map[string]interface{}{}}}},
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get map for key: [ 2ZP4J8GPBP6QFK3Y.JRTCKXETXF ]")
			},
		},
		{
			name:  "could not get OnDemand",
			price: priceData{
				awsData: aws.JSONValue{
					"terms": map[string]interface{}{}}},
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get map for key: [ OnDemand ]")
			},
		},
		{
			name:  "could not get terms",
			price: priceData{
				awsData: aws.JSONValue{}},
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get map for key: [ terms ]")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(test.price.GetOnDemandPrice())
		})
	}
}
