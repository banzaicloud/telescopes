package ec2_productinfo

import (
	"testing"
	"time"

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
	return dpi.AttrValues, nil
}

// GetProducts mocks the call for products
func (dpi *DummyProductInfoer) GetProducts(regionId string, attrKey string, attrValue AttrValue) ([]Ec2Vm, error) {

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
	}{
		{
			Name:          "product info successfully created",
			ProductInfoer: &DummyProductInfoer{},
			Assert: func(info *ProductInfo, err error) {
				assert.Nil(t, err, "should not get error")
			},
		},
		{
			Name:          "validation should fail nil values",
			ProductInfoer: nil,
			Assert: func(info *ProductInfo, err error) {
				assert.Nil(t, info, "the productinfo should be nil in case of error")
				assert.NotNil(t, err, "should get validation error when nil values provided")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.Assert(NewProductInfo(10*time.Second, nil, tc.ProductInfoer))
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
				AttrValues: []AttrValue{AttrValue{Value: float64(21), StrValue: Cpu}},
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
				Vms: []Ec2Vm{
					Ec2Vm{Cpus: float64(2), Mem: float64(32), OnDemandPrice: float64(0.32)},
				},
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewProductInfo(10*time.Second, test.Cache, test.ProductInfo)
			values, err := productInfo.renewVmsWithAttr(test.regionId, test.attrKey, test.attrValue)
			test.Assert(test.Cache, values, err)
		})
	}
}
