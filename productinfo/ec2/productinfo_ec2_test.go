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
				"instanceType": "db.t2.small",
				Cpu:            "1",
				Memory:         "2",
				"gpu":          "5",
			}},
		"terms": map[string]interface{}{
			"OnDemand": map[string]interface{}{
				"2ZP4J8GPBP6QFK3Y.JRTCKXETXF": map[string]interface{}{
					"priceDimensions": map[string]interface{}{
						"2ZP4J8GPBP6QFK3Y.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
							"pricePerUnit": map[string]interface{}{
								"USD": "5",
							}}}}}},
	},
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
			PriceList: []aws.JSONValue{
				{
					"product": map[string]interface{}{
						"attributes": map[string]interface{}{
							"instanceType": "db.t2.small",
							Cpu:            "1",
							Memory:         "2",
						}},
					"terms": map[string]interface{}{
						"OnDemand": map[string]interface{}{
							"2ZP4J8GPBP6QFK3Y.JRTCKXETXF": map[string]interface{}{
								"priceDimensions": map[string]interface{}{
									"2ZP4J8GPBP6QFK3Y.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
										"pricePerUnit": map[string]interface{}{
											"USD": "5",
										}}}}}}},
			},
		}, nil
	case 5:
		return nil, errors.New("failed to retrieve values")
	case 6:
		return &pricing.GetProductsOutput{
			PriceList: []aws.JSONValue{
				{
					"product": map[string]interface{}{
						"attributes": map[string]interface{}{
							"instanceType": "db.t2.small",
							Cpu:            "1",
							Memory:         "2",
						}},
					"terms": map[string]interface{}{
						"OnDemand": map[string]interface{}{
							"2ZP4J8GPBP6QFK3Y.JRTCKXETXF": map[string]interface{}{
								"priceDimensions": map[string]interface{}{
									"2ZP4J8GPBP6QFK3Y.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
										"pricePerUnit": map[string]interface{}{},
									}}}}}},
			},
		}, nil
	case 7:
		return &pricing.GetProductsOutput{
			PriceList: []aws.JSONValue{
				{
					"product": map[string]interface{}{
						"attributes": map[string]interface{}{
							"instanceType": "db.t2.small",
							Cpu:            "1",
						}}},
			},
		}, nil
	case 8:
		return &pricing.GetProductsOutput{
			PriceList: []aws.JSONValue{
				{
					"product": map[string]interface{}{
						"attributes": map[string]interface{}{
							"instanceType": "db.t2.small",
						}}},
			},
		}, nil
	case 9:
		return &pricing.GetProductsOutput{
			PriceList: []aws.JSONValue{
				{
					"product": map[string]interface{}{
						"attributes": map[string]interface{}{},
					}},
			},
		}, nil

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
			attrKey:       Cpu,
			attrValue:     productinfo.AttrValue{Value: float64(2), StrValue: productinfo.Cpu},
			pricingServie: &DummyPricingSource{TcId: 4},
			check: func(vm []productinfo.VmInfo, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, vm, []productinfo.VmInfo{productinfo.VmInfo{Type: "db.t2.small", OnDemandPrice: 5, SpotPrice: productinfo.PriceInfo(nil), Cpus: 1, Mem: 2, Gpus: 0}})
			},
		},
		{
			name:          "error - GetProducts",
			regionId:      "eu-central-1",
			attrKey:       Cpu,
			attrValue:     productinfo.AttrValue{Value: float64(2), StrValue: productinfo.Cpu},
			pricingServie: &DummyPricingSource{TcId: 5},
			check: func(vm []productinfo.VmInfo, err error) {
				assert.Equal(t, err, errors.New("failed to retrieve values"))
				assert.Nil(t, vm, "the vm should be nil")
			},
		},
		{
			name:          "error - on demand price",
			regionId:      "eu-central-1",
			attrKey:       Cpu,
			attrValue:     productinfo.AttrValue{Value: float64(2), StrValue: productinfo.Cpu},
			pricingServie: &DummyPricingSource{TcId: 6},
			check: func(vm []productinfo.VmInfo, err error) {
				assert.EqualError(t, err, "could not find available virtual machines with these parameters")
				assert.Nil(t, vm, "the vm should be nil")
			},
		},
		{
			name:          "error - memory",
			regionId:      "eu-central-1",
			attrKey:       Cpu,
			attrValue:     productinfo.AttrValue{Value: float64(2), StrValue: productinfo.Cpu},
			pricingServie: &DummyPricingSource{TcId: 7},
			check: func(vm []productinfo.VmInfo, err error) {
				assert.EqualError(t, err, "could not find available virtual machines with these parameters")
				assert.Nil(t, vm, "the vm should be nil")
			},
		},
		{
			name:          "error - cpu",
			regionId:      "eu-central-1",
			attrKey:       Cpu,
			attrValue:     productinfo.AttrValue{Value: float64(2), StrValue: productinfo.Cpu},
			pricingServie: &DummyPricingSource{TcId: 8},
			check: func(vm []productinfo.VmInfo, err error) {
				assert.EqualError(t, err, "could not find available virtual machines with these parameters")
				assert.Nil(t, vm, "the vm should be nil")
			},
		},
		{
			name:          "error - instance type",
			regionId:      "eu-central-1",
			attrKey:       Cpu,
			attrValue:     productinfo.AttrValue{Value: float64(2), StrValue: productinfo.Cpu},
			pricingServie: &DummyPricingSource{TcId: 9},
			check: func(vm []productinfo.VmInfo, err error) {
				assert.EqualError(t, err, "could not find available virtual machines with these parameters")
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

func TestPriceData_GetDataForKey(t *testing.T) {
	var missingData = priceData{
		awsData: aws.JSONValue{
			"product": map[string]interface{}{
				"attributes": map[string]interface{}{}}}}
	var wrongCast = priceData{
		awsData: aws.JSONValue{
			"product": map[string]interface{}{
				"attributes": map[string]interface{}{
					"instanceType": 0,
					Cpu:            1,
					Memory:         2,
					"gpu":          3,
				}},
		},
	}
	tests := []struct {
		name  string
		attr  string
		price priceData
		check func(s string, err error)
	}{
		{
			name:  "successful - get instance type",
			attr:  "instanceType",
			price: data,
			check: func(s string, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, s, "db.t2.small")
			},
		},
		{
			name:  "cast problem - get instance type",
			attr:  "instanceType",
			price: wrongCast,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get instanceType or could not cast instanceType to string")
			},
		},
		{
			name:  "missing data - get instance type",
			attr:  "instanceType",
			price: missingData,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get instanceType or could not cast instanceType to string")
			},
		},
		{
			name:  "successful - get cpu",
			attr:  Cpu,
			price: data,
			check: func(s string, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, s, "1")
			},
		},
		{
			name:  "cast problem - get cpu",
			attr:  Cpu,
			price: wrongCast,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get vcpu or could not cast vcpu to string")
			},
		},
		{
			name:  "missing data - get cpu",
			attr:  Cpu,
			price: missingData,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get vcpu or could not cast vcpu to string")
			},
		},
		{
			name:  "successful - get memory",
			attr:  Memory,
			price: data,
			check: func(s string, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, s, "2")
			},
		},
		{
			name:  "cast problem - get memory",
			attr:  Memory,
			price: wrongCast,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get memory or could not cast memory to string")
			},
		},
		{
			name:  "missing data - get memory",
			attr:  Memory,
			price: missingData,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get memory or could not cast memory to string")
			},
		},
		{
			name:  "successful - get gpu",
			attr:  "gpu",
			price: data,
			check: func(s string, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, s, "5")
			},
		},
		{
			name:  "cast problem - get gpu",
			attr:  "gpu",
			price: wrongCast,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get gpu or could not cast gpu to string")
			},
		},
		{
			name:  "missing data - get gpu",
			attr:  "gpu",
			price: missingData,
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get gpu or could not cast gpu to string")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pricedata, _ := newPriceData(test.price.awsData)
			test.check(pricedata.GetDataForKey(test.attr))
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
				assert.EqualError(t, err, "could not get on demand price or could not cast on demand price to string")
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
				assert.EqualError(t, err, "could not get on demand price or could not cast on demand price to string")
			},
		},
		{
			name: "could not get pricePerUnit",
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
			name: "could not get priceDimensions",
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
			name: "could not get OnDemand",
			price: priceData{
				awsData: aws.JSONValue{
					"terms": map[string]interface{}{}}},
			check: func(s string, err error) {
				assert.Equal(t, s, "")
				assert.EqualError(t, err, "could not get map for key: [ OnDemand ]")
			},
		},
		{
			name: "could not get terms",
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

func TestPriceData_newPriceData(t *testing.T) {
	tests := []struct {
		name   string
		prData aws.JSONValue
		check  func(data *priceData, err error)
	}{
		{
			name:   "successful",
			prData: aws.JSONValue{"product": map[string]interface{}{"attributes": map[string]interface{}{}}},
			check: func(data *priceData, err error) {
				assert.NotNil(t, data, "")
				assert.Nil(t, err)
			},
		},
		{
			name:   "could not get map for key attributes",
			prData: aws.JSONValue{"product": map[string]interface{}{}},
			check: func(data *priceData, err error) {
				assert.Nil(t, data, "")
				assert.EqualError(t, err, "could not get map for key: [ attributes ]")
			},
		},
		{
			name:   "could not get map for key product",
			prData: aws.JSONValue{},
			check: func(data *priceData, err error) {
				assert.Nil(t, data, "")
				assert.EqualError(t, err, "could not get map for key: [ product ]")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(newPriceData(test.prData))
		})
	}
}
