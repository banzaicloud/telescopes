package productinfo

import (
	"testing"
	"time"

	"fmt"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

// DummyProductInfoer type implements the ProductInfoer interface for mockig of external calls
// the struct is to be extended according to the needs of test cases
type DummyProductInfoer struct {
	AttrValues AttrValues
	Vms        []Ec2Vm

	// implement the interface
	ProductInfoer
}

// GetAttributeValues mocks the call for attribute values
func (dpi *DummyProductInfoer) GetAttributeValues(attribute string) (AttrValues, error) {
	if attribute == "error" {
		return nil, fmt.Errorf("attribute value error")
	}
	return dpi.AttrValues, nil
}

// GetProducts mocks the call for products
func (dpi *DummyProductInfoer) GetProducts(regionId string, attrKey string, attrValue AttrValue) ([]Ec2Vm, error) {
	if regionId == "errorRegion" {
		return nil, fmt.Errorf("regionId error")
	}
	return dpi.Vms, nil
}

func (dpi *DummyProductInfoer) GetRegion(id string) *endpoints.Region {
	return nil
}

func (dpi *DummyProductInfoer) GetRegions() map[string]string {
	return nil
}

func TestNewProductInfo(t *testing.T) {
	testCases := []struct {
		Name          string
		ProductInfoer ProductInfoer
		Assert        func(info *ProductInfo, err error)
		Cache         *cache.Cache
	}{
		{
			Name:          "product info successfully created",
			ProductInfoer: &DummyProductInfoer{},
			Cache:         cache.New(5*time.Minute, 10*time.Minute),
			Assert: func(info *ProductInfo, err error) {
				assert.Nil(t, err, "should not get error")
			},
		},
		{
			Name:          "validation should fail nil values",
			ProductInfoer: nil,
			Cache:         cache.New(5*time.Minute, 10*time.Minute),
			Assert: func(info *ProductInfo, err error) {
				assert.Nil(t, info, "the productinfo should be nil in case of error")
				assert.NotNil(t, err, "should get validation error when nil values provided")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.Assert(NewProductInfo(10*time.Second, tc.Cache, tc.ProductInfoer))
		})
	}

}

func TestRenewAttributeValues(t *testing.T) {
	testCases := []struct {
		Name        string
		ProductInfo ProductInfoer
		Cache       *cache.Cache
		Attribute   string
		Assert      func(cache *cache.Cache, values AttrValues, err error)
	}{
		{
			Name: "attribute successfully renewed - empty cache",
			ProductInfo: &DummyProductInfoer{
				// this is returned by the AWS
				AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: Cpu}},
			},
			Cache:     cache.New(5*time.Minute, 10*time.Minute),
			Attribute: Cpu,
			Assert: func(cache *cache.Cache, values AttrValues, err error) {
				assert.Nil(t, err, "no error expected")
				assert.NotNil(t, values, "the retreived attribute slice shouldn't be nil")
				assert.Equal(t, 1, cache.ItemCount(), "there should be exactly one item in the cache")
				vals, _ := cache.Get("/banzaicloud.com/recommender/ec2/attrValues/vcpu")

				for _, val := range vals.(AttrValues) {
					assert.Equal(t, float64(21), val.Value, "the value in the cache is not as expected")
				}

			},
		},
		{
			Name: "attribute error",
			ProductInfo: &DummyProductInfoer{
				// this is returned by the AWS
				AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: Cpu}},
			},
			Cache:     cache.New(5*time.Minute, 10*time.Minute),
			Attribute: "error",
			Assert: func(cache *cache.Cache, values AttrValues, err error) {
				assert.NotNil(t, err, "should receive attribute error")
				assert.Nil(t, values, "no values expected")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			productInfo, _ := NewProductInfo(10*time.Second, tc.Cache, tc.ProductInfo)
			values, err := productInfo.renewAttrValues(tc.Attribute)
			tc.Assert(productInfo.vmAttrStore, values, err)
		})
	}

}

func TestRenewVmsWithAttr(t *testing.T) {
	tests := []struct {
		name        string
		regionId    string
		attrKey     string
		attrValue   AttrValue
		ProductInfo ProductInfoer
		Cache       *cache.Cache
		Assert      func(cache *cache.Cache, vms []Ec2Vm, err error)
	}{
		{
			name:      "vm successfully renewed",
			regionId:  "testRegion",
			attrKey:   Cpu,
			attrValue: AttrValue{Value: float64(2), StrValue: Cpu},
			ProductInfo: &DummyProductInfoer{
				// test data
				Vms: []Ec2Vm{{Cpus: float64(2), Mem: float64(32), OnDemandPrice: float64(0.32)}},
			},
			Cache: cache.New(5*time.Minute, 10*time.Minute),
			Assert: func(cache *cache.Cache, vms []Ec2Vm, err error) {
				assert.Nil(t, err, "should not get error on vm renewal")
				assert.Equal(t, 1, len(vms), "there should be a single entry in values")
				vals, _ := cache.Get("/banzaicloud.com/recommender/ec2/testRegion/vms/vcpu/2.000000")

				for _, val := range vals.([]Ec2Vm) {
					assert.Equal(t, float64(32), val.Mem, "the value in the cache is not as expected")
				}

			},
		},
		{
			name:      "region id error",
			regionId:  "errorRegion",
			attrKey:   Cpu,
			attrValue: AttrValue{Value: float64(2), StrValue: Cpu},
			ProductInfo: &DummyProductInfoer{
				// test data
				Vms: []Ec2Vm{{Cpus: float64(2), Mem: float64(32), OnDemandPrice: float64(0.32)}},
			},
			Cache: cache.New(5*time.Minute, 10*time.Minute),
			Assert: func(cache *cache.Cache, vms []Ec2Vm, err error) {
				assert.NotNil(t, err, "should receive error")
				assert.Nil(t, vms, "no vms expected")

			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewProductInfo(10*time.Second, test.Cache, test.ProductInfo)
			values, err := productInfo.renewVmsWithAttr(test.regionId, test.attrKey, test.attrValue)
			test.Assert(test.Cache, values, err)
		})
	}
}

func TestGetAttrKey(t *testing.T) {
	tests := []struct {
		name        string
		Attribute   string
		ProductInfo ProductInfoer
		Cache       *cache.Cache
		Assert      func(values string)
	}{
		{
			name:        "print attribute key",
			Attribute:   Cpu,
			ProductInfo: &DummyProductInfoer{},
			Cache:       cache.New(5*time.Minute, 10*time.Minute),
			Assert: func(values string) {
				assert.Equal(t, values, "/banzaicloud.com/recommender/ec2/attrValues/vcpu")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewProductInfo(10*time.Second, test.Cache, test.ProductInfo)
			values := productInfo.getAttrKey(test.Attribute)
			test.Assert(values)
		})
	}
}

func TestGetVmKey(t *testing.T) {
	tests := []struct {
		name        string
		region      string
		attrKey     string
		attrValue   float64
		ProductInfo ProductInfoer
		Cache       *cache.Cache
		Assert      func(values string)
	}{
		{
			name:        "print vm key",
			region:      "testRegion",
			attrKey:     Cpu,
			attrValue:   float64(3),
			ProductInfo: &DummyProductInfoer{},
			Cache:       cache.New(5*time.Minute, 10*time.Minute),
			Assert: func(values string) {
				assert.Equal(t, values, "/banzaicloud.com/recommender/ec2/testRegion/vms/vcpu/3.000000")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewProductInfo(10*time.Second, test.Cache, test.ProductInfo)
			values := productInfo.getVmKey(test.region, test.attrKey, test.attrValue)
			test.Assert(values)
		})
	}
}

func TestGetAttrValues(t *testing.T) {
	tests := []struct {
		name        string
		ProductInfo ProductInfoer
		Cache       *cache.Cache
		Attribute   string
		Assert      func(Attr AttrValues, err error)
	}{
		{
			name: "successful",
			ProductInfo: &DummyProductInfoer{
				AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: Cpu}},
			},
			Cache:     cache.New(5*time.Minute, 10*time.Minute),
			Attribute: Memory,
			Assert: func(Attr AttrValues, err error) {
				assert.Nil(t, err, "the returned error must be nil")
				assert.Equal(t, Attr, AttrValues{AttrValue{StrValue: "vcpu", Value: 21}})
			},
		},
		{
			name: "attribute values error",
			ProductInfo: &DummyProductInfoer{
				AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: Cpu}},
			},
			Cache:     cache.New(5*time.Minute, 10*time.Minute),
			Attribute: "error",
			Assert: func(Attr AttrValues, err error) {
				assert.NotNil(t, err, "should receive attribute error")
				assert.Nil(t, Attr, "no attribute values expected")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewProductInfo(10*time.Second, test.Cache, test.ProductInfo)
			values, err := productInfo.getAttrValues(test.Attribute)
			test.Assert(values, err)
		})
	}
}

func TestGetAttrValue(t *testing.T) {
	tests := []struct {
		name        string
		attrKey     string
		AttrValue   float64
		ProductInfo ProductInfoer
		Cache       *cache.Cache
		Assert      func(Attr *AttrValue, err error)
	}{
		{
			name:      "could not find attribute value",
			attrKey:   Cpu,
			AttrValue: float64(2),
			Cache:     cache.New(5*time.Minute, 10*time.Minute),
			ProductInfo: &DummyProductInfoer{
				AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: Cpu}},
			},
			Assert: func(Attr *AttrValue, err error) {
				assert.NotNil(t, err, "could not find attribute value")
				assert.Nil(t, Attr, "the retrieved values should be nil")
			},
		},
		{
			name:      "successful",
			attrKey:   Cpu,
			AttrValue: float64(21),
			Cache:     cache.New(5*time.Minute, 10*time.Minute),
			ProductInfo: &DummyProductInfoer{
				AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: Cpu}},
			},
			Assert: func(Attr *AttrValue, err error) {
				assert.Nil(t, err, "the returned error must be nil")
				assert.Equal(t, Attr, &AttrValue{StrValue: Cpu, Value: float64(21)})
			},
		},
		{
			name:      "attribute error",
			attrKey:   "error",
			AttrValue: float64(21),
			Cache:     cache.New(5*time.Minute, 10*time.Minute),
			ProductInfo: &DummyProductInfoer{
				AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: Cpu}},
			},
			Assert: func(Attr *AttrValue, err error) {
				assert.NotNil(t, err, "should receive attribute error")
				assert.Nil(t, Attr, "the retrieved values should be nil")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewProductInfo(10*time.Second, test.Cache, test.ProductInfo)
			value, err := productInfo.getAttrValue(test.attrKey, test.AttrValue)
			test.Assert(value, err)
		})
	}
}

func Test_getAttrValues(t *testing.T) {
	tests := []struct {
		name        string
		ProductInfo ProductInfoer
		Cache       *cache.Cache
		Attribute   string
		Assert      func(float []float64, err error)
	}{
		{
			name:        "successful",
			ProductInfo: &DummyProductInfoer{},
			Cache:       cache.New(5*time.Minute, 10*time.Minute),
			Attribute:   Memory,
			Assert: func(float []float64, err error) {
				assert.Nil(t, err, "the returned error must be nil")
				assert.Equal(t, float, []float64{})
			},
		},
		{
			name:        "attribute error",
			ProductInfo: &DummyProductInfoer{},
			Cache:       cache.New(5*time.Minute, 10*time.Minute),
			Attribute:   "error",
			Assert: func(float []float64, err error) {
				assert.NotNil(t, err, "should receive attribute error")
				assert.Nil(t, float, "the retrieved values should be nil")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewProductInfo(10*time.Second, test.Cache, test.ProductInfo)
			values, err := productInfo.GetAttrValues(test.Attribute)
			test.Assert(values, err)
		})
	}
}

func TestGetVmsWithAttrValue(t *testing.T) {
	tests := []struct {
		name        string
		regionId    string
		attrKey     string
		value       float64
		ProductInfo ProductInfoer
		Cache       *cache.Cache
		Assert      func(vms []Ec2Vm, err error)
	}{
		{
			name:     "successful",
			regionId: "testRegion",
			attrKey:  Cpu,
			value:    float64(21),
			ProductInfo: &DummyProductInfoer{
				Vms:        []Ec2Vm{{Cpus: float64(2), Mem: float64(32), OnDemandPrice: float64(0.32)}},
				AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: Cpu}},
			},
			Cache: cache.New(5*time.Minute, 10*time.Minute),
			Assert: func(vms []Ec2Vm, err error) {
				assert.Nil(t, err, "the returned error must be nil")
				assert.Equal(t, vms, []Ec2Vm{Ec2Vm{Type: "", OnDemandPrice: 0.32, Cpus: 2, Mem: 32, Gpus: 0}})
			},
		},
		{
			name:     "vm error",
			regionId: "errorRegion",
			attrKey:  Cpu,
			value:    float64(21),
			ProductInfo: &DummyProductInfoer{
				Vms:        []Ec2Vm{{Cpus: float64(2), Mem: float64(32), OnDemandPrice: float64(0.32)}},
				AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: Cpu}},
			},
			Cache: cache.New(5*time.Minute, 10*time.Minute),
			Assert: func(vms []Ec2Vm, err error) {
				assert.NotNil(t, err, "should receive error")
				assert.Nil(t, vms, "the retrieved values should be nil")
			},
		},
		{
			name:     "attribute error",
			regionId: "testRegion",
			attrKey:  "error",
			value:    float64(21),
			ProductInfo: &DummyProductInfoer{
				Vms:        []Ec2Vm{{Cpus: float64(2), Mem: float64(32), OnDemandPrice: float64(0.32)}},
				AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: Cpu}},
			},
			Cache: cache.New(5*time.Minute, 10*time.Minute),
			Assert: func(vms []Ec2Vm, err error) {
				assert.NotNil(t, err, "should receive attribute error")
				assert.Nil(t, vms, "the retrieved values should be nil")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewProductInfo(10*time.Second, test.Cache, test.ProductInfo)
			values, err := productInfo.GetVmsWithAttrValue(test.regionId, test.attrKey, test.value)
			test.Assert(values, err)
		})
	}
}
