package productinfo

import (
	"fmt"
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
	Vms        []VmInfo

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
func (dpi *DummyProductInfoer) GetProducts(regionId string, attrKey string, attrValue AttrValue) ([]VmInfo, error) {
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

func (dpi *DummyProductInfoer) GetCurrentSpotPrices(region string) (map[string]PriceInfo, error) {
	return nil, nil
}

func (dpi *DummyProductInfoer) GetMemoryAttrName() string {
	return "memory"
}

func (dpi *DummyProductInfoer) GetCpuAttrName() string {
	return "vcpu"
}

func TestNewProductInfo(t *testing.T) {
	testCases := []struct {
		Name          string
		ProductInfoer map[string]ProductInfoer
		Assert        func(info *CachingProductInfo, err error)
		Cache         *cache.Cache
	}{
		{
			Name: "product info successfully created",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			Cache: cache.New(5*time.Minute, 10*time.Minute),
			Assert: func(info *CachingProductInfo, err error) {
				assert.Nil(t, err, "should not get error")
			},
		},
		{
			Name:          "validation should fail nil values",
			ProductInfoer: nil,
			Cache:         cache.New(5*time.Minute, 10*time.Minute),
			Assert: func(info *CachingProductInfo, err error) {
				assert.Nil(t, info, "the productinfo should be nil in case of error")
				assert.NotNil(t, err, "should get validation error when nil values provided")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.Assert(NewCachingProductInfo(10*time.Second, tc.Cache, tc.ProductInfoer))
		})
	}

}

func TestRenewAttributeValues(t *testing.T) {
	testCases := []struct {
		Name          string
		Provider      string
		ProductInfoer map[string]ProductInfoer
		Cache         *cache.Cache
		Attribute     string
		Assert        func(cache *cache.Cache, values AttrValues, err error)
	}{
		{
			Name:     "attribute successfully renewed - empty cache",
			Provider: "dummy",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: Cpu}},
				},
			},
			Cache:     cache.New(5*time.Minute, 10*time.Minute),
			Attribute: Cpu,
			Assert: func(cache *cache.Cache, values AttrValues, err error) {
				assert.Nil(t, err, "no error expected")
				assert.NotNil(t, values, "the retreived attribute slice shouldn't be nil")
				assert.Equal(t, 1, cache.ItemCount(), "there should be exactly one item in the cache")
				vals, _ := cache.Get("/banzaicloud.com/recommender/dummy/attrValues/cpu")

				for _, val := range vals.(AttrValues) {
					assert.Equal(t, float64(21), val.Value, "the value in the cache is not as expected")
				}

			},
		},
		{
			Name: "attribute error",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: Cpu}},
				},
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
			productInfo, _ := NewCachingProductInfo(10*time.Second, tc.Cache, tc.ProductInfoer)
			values, err := productInfo.renewAttrValues(tc.Provider, tc.Attribute)
			tc.Assert(productInfo.vmAttrStore, values, err)
		})
	}

}

func TestRenewVmsWithAttr(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		regionId      string
		attrKey       string
		attrValue     AttrValue
		ProductInfoer map[string]ProductInfoer
		Cache         *cache.Cache
		Assert        func(cache *cache.Cache, vms []VmInfo, err error)
	}{
		{
			name:      "vm successfully renewed",
			provider:  "dummy",
			regionId:  "testRegion",
			attrKey:   Cpu,
			attrValue: AttrValue{Value: float64(2), StrValue: Cpu},
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					Vms: []VmInfo{{Cpus: float64(2), Mem: float64(32), OnDemandPrice: float64(0.32)}},
				},
			},
			Cache: cache.New(5*time.Minute, 10*time.Minute),
			Assert: func(cache *cache.Cache, vms []VmInfo, err error) {
				assert.Nil(t, err, "should not get error on vm renewal")
				assert.Equal(t, 1, len(vms), "there should be a single entry in values")
				vals, _ := cache.Get("/banzaicloud.com/recommender/dummy/testRegion/vms/cpu/2.000000")

				for _, val := range vals.([]VmInfo) {
					assert.Equal(t, float64(32), val.Mem, "the value in the cache is not as expected")
				}

			},
		},
		{
			name:      "region id error",
			provider:  "dummy",
			regionId:  "errorRegion",
			attrKey:   Cpu,
			attrValue: AttrValue{Value: float64(2), StrValue: Cpu},
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					Vms: []VmInfo{{Cpus: float64(2), Mem: float64(32), OnDemandPrice: float64(0.32)}},
				},
			},
			Cache: cache.New(5*time.Minute, 10*time.Minute),
			Assert: func(cache *cache.Cache, vms []VmInfo, err error) {
				assert.NotNil(t, err, "should receive error")
				assert.Nil(t, vms, "no vms expected")

			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, test.Cache, test.ProductInfoer)
			values, err := productInfo.renewVmsWithAttr(test.provider, test.regionId, test.attrKey, test.attrValue)
			test.Assert(test.Cache, values, err)
		})
	}
}

func TestGetAttrKey(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		Attribute     string
		ProductInfoer map[string]ProductInfoer
		Cache         *cache.Cache
		Assert        func(values string)
	}{
		{
			name:      "print attribute key",
			provider:  "dummy",
			Attribute: Cpu,
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			Cache: cache.New(5*time.Minute, 10*time.Minute),
			Assert: func(values string) {
				assert.Equal(t, values, "/banzaicloud.com/recommender/dummy/attrValues/cpu")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, test.Cache, test.ProductInfoer)
			values := productInfo.getAttrKey(test.provider, test.Attribute)
			test.Assert(values)
		})
	}
}

func TestGetVmKey(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		region        string
		attrKey       string
		attrValue     float64
		ProductInfoer map[string]ProductInfoer
		Cache         *cache.Cache
		Assert        func(values string)
	}{
		{
			name:      "print vm key",
			provider:  "dummy",
			region:    "testRegion",
			attrKey:   Cpu,
			attrValue: float64(3),
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			Cache: cache.New(5*time.Minute, 10*time.Minute),
			Assert: func(values string) {
				assert.Equal(t, values, "/banzaicloud.com/recommender/dummy/testRegion/vms/cpu/3.000000")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, test.Cache, test.ProductInfoer)
			values := productInfo.getVmKey(test.provider, test.region, test.attrKey, test.attrValue)
			test.Assert(values)
		})
	}
}

func TestGetAttrValues(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		ProductInfoer map[string]ProductInfoer
		Cache         *cache.Cache
		Attribute     string
		Assert        func(Attr AttrValues, err error)
	}{
		{
			name:     "successful",
			provider: "dummy",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: "21"}},
				},
			},
			Cache:     cache.New(5*time.Minute, 10*time.Minute),
			Attribute: Memory,
			Assert: func(Attr AttrValues, err error) {
				assert.Nil(t, err, "the returned error must be nil")
				assert.Equal(t, Attr, AttrValues{AttrValue{StrValue: "21", Value: 21}})
			},
		},
		{
			name:     "attribute values error",
			provider: "dummy",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: "21"}},
				},
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
			productInfo, _ := NewCachingProductInfo(10*time.Second, test.Cache, test.ProductInfoer)
			values, err := productInfo.getAttrValues(test.provider, test.Attribute)
			test.Assert(values, err)
		})
	}
}

func TestGetAttrValue(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		attrKey       string
		AttrValue     float64
		ProductInfoer map[string]ProductInfoer
		Cache         *cache.Cache
		Assert        func(Attr *AttrValue, err error)
	}{
		{
			name:      "could not find attribute value",
			provider:  "dummy",
			attrKey:   Cpu,
			AttrValue: float64(2),
			Cache:     cache.New(5*time.Minute, 10*time.Minute),
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: "21"}},
				},
			},
			Assert: func(Attr *AttrValue, err error) {
				assert.NotNil(t, err, "could not find attribute value")
				assert.Nil(t, Attr, "the retrieved values should be nil")
			},
		},
		{
			name:      "successful",
			provider:  "dummy",
			attrKey:   Cpu,
			AttrValue: float64(21),
			Cache:     cache.New(5*time.Minute, 10*time.Minute),
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: "21"}},
				},
			},
			Assert: func(Attr *AttrValue, err error) {
				assert.Nil(t, err, "the returned error must be nil")
				assert.Equal(t, Attr, &AttrValue{StrValue: "21", Value: float64(21)})
			},
		},
		{
			name:      "attribute error",
			provider:  "dummy",
			attrKey:   "error",
			AttrValue: float64(21),
			Cache:     cache.New(5*time.Minute, 10*time.Minute),
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: "21"}},
				},
			},
			Assert: func(Attr *AttrValue, err error) {
				assert.NotNil(t, err, "should receive attribute error")
				assert.Nil(t, Attr, "the retrieved values should be nil")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, test.Cache, test.ProductInfoer)
			value, err := productInfo.getAttrValue(test.provider, test.attrKey, test.AttrValue)
			test.Assert(value, err)
		})
	}
}

func Test_getAttrValues(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		ProductInfoer map[string]ProductInfoer
		Cache         *cache.Cache
		Attribute     string
		Assert        func(float []float64, err error)
	}{
		{
			name:     "successful",
			provider: "dummy",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			Cache:     cache.New(5*time.Minute, 10*time.Minute),
			Attribute: Memory,
			Assert: func(float []float64, err error) {
				assert.Nil(t, err, "the returned error must be nil")
				assert.Equal(t, float, []float64{})
			},
		},
		{
			name:     "attribute error",
			provider: "dummy",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			Cache:     cache.New(5*time.Minute, 10*time.Minute),
			Attribute: "error",
			Assert: func(float []float64, err error) {
				assert.NotNil(t, err, "should receive attribute error")
				assert.Nil(t, float, "the retrieved values should be nil")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, test.Cache, test.ProductInfoer)
			values, err := productInfo.GetAttrValues(test.provider, test.Attribute)
			test.Assert(values, err)
		})
	}
}

func TestGetVmsWithAttrValue(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		regionId      string
		attrKey       string
		value         float64
		ProductInfoer map[string]ProductInfoer
		Cache         *cache.Cache
		Assert        func(vms []VmInfo, err error)
	}{
		{
			name:     "successful",
			provider: "dummy",
			regionId: "testRegion",
			attrKey:  Cpu,
			value:    float64(21),
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					Vms:        []VmInfo{{Cpus: float64(2), Mem: float64(32), OnDemandPrice: float64(0.32)}},
					AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: "21"}},
				},
			},
			Cache: cache.New(5*time.Minute, 10*time.Minute),
			Assert: func(vms []VmInfo, err error) {
				assert.Nil(t, err, "the returned error must be nil")
				assert.Equal(t, vms, []VmInfo{VmInfo{Type: "", OnDemandPrice: 0.32, Cpus: 2, Mem: 32, Gpus: 0}})
			},
		},
		{
			name:     "vm error",
			provider: "dummy",
			regionId: "errorRegion",
			attrKey:  Cpu,
			value:    float64(21),
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					Vms:        []VmInfo{{Cpus: float64(2), Mem: float64(32), OnDemandPrice: float64(0.32)}},
					AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: "21"}},
				},
			},
			Cache: cache.New(5*time.Minute, 10*time.Minute),
			Assert: func(vms []VmInfo, err error) {
				assert.NotNil(t, err, "should receive error")
				assert.Nil(t, vms, "the retrieved values should be nil")
			},
		},
		{
			name:     "attribute error",
			provider: "dummy",
			regionId: "testRegion",
			attrKey:  "error",
			value:    float64(21),
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					Vms:        []VmInfo{{Cpus: float64(2), Mem: float64(32), OnDemandPrice: float64(0.32)}},
					AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: "21"}},
				},
			},
			Cache: cache.New(5*time.Minute, 10*time.Minute),
			Assert: func(vms []VmInfo, err error) {
				assert.NotNil(t, err, "should receive attribute error")
				assert.Nil(t, vms, "the retrieved values should be nil")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, test.Cache, test.ProductInfoer)
			values, err := productInfo.GetVmsWithAttrValue(test.provider, test.regionId, test.attrKey, test.value)
			test.Assert(values, err)
		})
	}
}
