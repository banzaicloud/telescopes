package productinfo

import (
	"errors"
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
	TcId       int
	// implement the interface
	ProductInfoer
}

const (
	getProductsError = "invalid region"
)

func (dpi *DummyProductInfoer) Initialize() (map[string]map[string]Price, error) {
	if dpi.TcId == 1 {
		return nil, errors.New("initialization error")
	}
	return map[string]map[string]Price{"dummy": {"t2.nano": {OnDemandPrice: 25, SpotPrice: SpotPriceInfo{"testZone2": 43}}}}, nil
}

func (dpi *DummyProductInfoer) GetAttributeValues(attribute string) (AttrValues, error) {
	if attribute == Memory {
		return nil, errors.New("memory is invalid attribute if we call the GetAttributeValues function")
	}
	return dpi.AttrValues, nil
}

func (dpi *DummyProductInfoer) GetProducts(regionId string) ([]VmInfo, error) {
	if regionId == getProductsError {
		return nil, errors.New("could not retrieve virtual machines")
	}
	return dpi.Vms, nil
}

func (dpi *DummyProductInfoer) GetZones(region string) ([]string, error) {
	return []string{"Zone1", "Zone2"}, nil
}

func (dpi *DummyProductInfoer) GetRegion(id string) *endpoints.Region {
	return nil
}

func (dpi *DummyProductInfoer) GetRegions() (map[string]string, error) {
	return nil, nil
}

func (dpi *DummyProductInfoer) HasShortLivedPriceInfo() bool {
	if dpi.TcId == 2 {
		return false
	}
	return true
}

func (dpi *DummyProductInfoer) GetCurrentPrices(region string) (map[string]Price, error) {
	if dpi.TcId == 3 {
		return nil, errors.New("could not get current prices")
	}

	return map[string]Price{"t2.nano": {OnDemandPrice: float64(32), SpotPrice: SpotPriceInfo{"testZone1": 32}}}, nil
}

func (dpi *DummyProductInfoer) GetMemoryAttrName() string {
	return "memory"
}

func (dpi *DummyProductInfoer) GetCpuAttrName() string {
	return "vcpu"
}

func TestNewCachingProductInfo(t *testing.T) {
	tests := []struct {
		Name          string
		ProductInfoer map[string]ProductInfoer
		checker       func(info *CachingProductInfo, err error)
	}{
		{
			Name: "product info successfully created",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			checker: func(info *CachingProductInfo, err error) {
				assert.Nil(t, err, "should not get error")
				assert.NotNil(t, info, "the product info should not be nil")
			},
		},
		{
			Name:          "validation should fail nil values",
			ProductInfoer: nil,
			checker: func(info *CachingProductInfo, err error) {
				assert.Nil(t, info, "the productinfo should be nil in case of error")
				assert.EqualError(t, err, "could not create product infoer")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test.checker(NewCachingProductInfo(10*time.Second, cache.New(5*time.Minute, 10*time.Minute), test.ProductInfoer))
		})
	}

}

func TestCachingProductInfo_renewAttrValues(t *testing.T) {
	tests := []struct {
		Name          string
		Provider      string
		ProductInfoer map[string]ProductInfoer
		Cache         *cache.Cache
		Attribute     string
		checker       func(cache *cache.Cache, values AttrValues, err error)
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
			checker: func(cache *cache.Cache, values AttrValues, err error) {
				assert.Nil(t, err, "no error expected")
				assert.Equal(t, AttrValues{AttrValue{StrValue: "cpu", Value: 21}}, values)
				assert.Equal(t, 1, cache.ItemCount(), "there should be exactly one item in the cache")
				vals, _ := cache.Get("/banzaicloud.com/recommender/dummy/attrValues/cpu")

				for _, val := range vals.(AttrValues) {
					assert.Equal(t, float64(21), val.Value, "the value in the cache is not as expected")
				}

			},
		},
		{
			Name:     "invalid attribute",
			Provider: "dummy",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: Cpu}},
				},
			},
			Cache:     cache.New(5*time.Minute, 10*time.Minute),
			Attribute: Memory,
			checker: func(cache *cache.Cache, values AttrValues, err error) {
				assert.EqualError(t, err, "memory is invalid attribute if we call the GetAttributeValues function")
				assert.Nil(t, values, "no values expected")
			},
		},
		{
			Name:     "unsupported attribute - error",
			Provider: "dummy",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: Cpu}},
				},
			},
			Cache:     cache.New(5*time.Minute, 10*time.Minute),
			Attribute: "invalid value",
			checker: func(cache *cache.Cache, values AttrValues, err error) {
				assert.EqualError(t, err, "unsupported attribute: invalid value")
				assert.Nil(t, values, "no values expected")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, test.Cache, test.ProductInfoer)
			values, err := productInfo.renewAttrValues(test.Provider, test.Attribute)
			test.checker(productInfo.vmAttrStore, values, err)
		})
	}

}

func TestCachingProductInfo_renewVms(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		regionId      string
		attrKey       string
		attrValue     AttrValue
		ProductInfoer map[string]ProductInfoer
		Cache         *cache.Cache
		checker       func(cache *cache.Cache, vms []VmInfo, err error)
	}{
		{
			name:      "vm successfully renewed",
			provider:  "dummy",
			regionId:  "dummyRegion",
			attrKey:   Cpu,
			attrValue: AttrValue{Value: float64(2), StrValue: Cpu},
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					Vms: []VmInfo{{Cpus: float64(2), Mem: float64(32), OnDemandPrice: float64(0.32)}},
				},
			},
			Cache: cache.New(5*time.Minute, 10*time.Minute),
			checker: func(cache *cache.Cache, vms []VmInfo, err error) {
				assert.Nil(t, err, "should not get error on vm renewal")
				assert.Equal(t, 1, len(vms), "there should be a single entry in values")
				vals, _ := cache.Get("/banzaicloud.com/recommender/dummy/dummyRegion/vms")

				for _, val := range vals.([]VmInfo) {
					assert.Equal(t, float64(32), val.Mem, "the value in the cache is not as expected")
				}

			},
		},
		{
			name:      "could not retrieve virtual machines - GetProducts error",
			provider:  "dummy",
			regionId:  getProductsError,
			attrKey:   Cpu,
			attrValue: AttrValue{Value: float64(2), StrValue: Cpu},
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					Vms: []VmInfo{{Cpus: float64(2), Mem: float64(32), OnDemandPrice: float64(0.32)}},
				},
			},
			Cache: cache.New(5*time.Minute, 10*time.Minute),
			checker: func(cache *cache.Cache, vms []VmInfo, err error) {
				assert.EqualError(t, err, "could not retrieve virtual machines")
				assert.Nil(t, vms, "no vms expected")

			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, test.Cache, test.ProductInfoer)
			values, err := productInfo.renewVms(test.provider, test.regionId)
			test.checker(test.Cache, values, err)
		})
	}
}

func TestCachingProductInfo_getAttrValues(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		ProductInfoer map[string]ProductInfoer
		Attribute     string
		checker       func(Attr AttrValues, err error)
	}{
		{
			name:     "successful - return a slice with the values",
			provider: "dummy",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: "21"}},
				},
			},
			Attribute: Cpu,
			checker: func(Attr AttrValues, err error) {
				assert.Nil(t, err, "the returned error must be nil")
				assert.Equal(t, AttrValues{AttrValue{StrValue: "21", Value: 21}}, Attr)
			},
		},
		{
			name:     "unsupported attribute - error",
			provider: "dummy",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: "21"}},
				},
			},
			Attribute: "invalid",
			checker: func(Attr AttrValues, err error) {
				assert.EqualError(t, err, "unsupported attribute: invalid")
				assert.Nil(t, Attr, "no attribute values expected")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, cache.New(5*time.Minute, 10*time.Minute), test.ProductInfoer)
			values, err := productInfo.getAttrValues(test.provider, test.Attribute)
			test.checker(values, err)
		})
	}
}

func TestCachingProductInfo_getAttrValue(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		attrKey       string
		AttrValue     float64
		ProductInfoer map[string]ProductInfoer
		checker       func(Attr *AttrValue, err error)
	}{
		{
			name:      "invalid StrValue",
			provider:  "dummy",
			attrKey:   Cpu,
			AttrValue: float64(2),
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: "invalid"}},
				},
			},
			checker: func(Attr *AttrValue, err error) {
				assert.EqualError(t, err, "couldn't find attribute Value")
				assert.Nil(t, Attr, "the retrieved values should be nil")
			},
		},
		{
			name:      "successful - get attribute value",
			provider:  "dummy",
			attrKey:   Cpu,
			AttrValue: float64(21),
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: "21"}},
				},
			},
			checker: func(Attr *AttrValue, err error) {
				assert.Nil(t, err, "the returned error must be nil")
				assert.Equal(t, &AttrValue{StrValue: "21", Value: float64(21)}, Attr)
			},
		},
		{
			name:      "unsupported attribute - error",
			provider:  "dummy",
			attrKey:   "invalid",
			AttrValue: float64(21),
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					AttrValues: AttrValues{AttrValue{Value: float64(21), StrValue: "21"}},
				},
			},
			checker: func(Attr *AttrValue, err error) {
				assert.EqualError(t, err, "unsupported attribute: invalid")
				assert.Nil(t, Attr, "the retrieved values should be nil")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, cache.New(5*time.Minute, 10*time.Minute), test.ProductInfoer)
			value, err := productInfo.getAttrValue(test.provider, test.attrKey, test.AttrValue)
			test.checker(value, err)
		})
	}
}

func TestCachingProductInfo_GetAttrValues(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		ProductInfoer map[string]ProductInfoer
		Attribute     string
		checker       func(value []float64, err error)
	}{
		{
			name:     "successful - get attribute values",
			provider: "dummy",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					AttrValues: AttrValues{AttrValue{Value: float64(21)}, AttrValue{Value: float64(23)}},
				},
			},
			Attribute: Cpu,
			checker: func(value []float64, err error) {
				assert.Nil(t, err, "the returned error must be nil")
				assert.Equal(t, []float64{0, 0, 21, 23}, value)
			},
		},
		{
			name:     "unsupported attribute - error",
			provider: "dummy",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					AttrValues: AttrValues{AttrValue{Value: float64(21)}, AttrValue{Value: float64(23)}},
				},
			},
			Attribute: "invalid",
			checker: func(value []float64, err error) {
				assert.EqualError(t, err, "unsupported attribute: invalid")
				assert.Nil(t, value, "the retrieved values should be nil")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, cache.New(5*time.Minute, 10*time.Minute), test.ProductInfoer)
			values, err := productInfo.GetAttrValues(test.provider, test.Attribute)
			test.checker(values, err)
		})
	}
}

func TestCachingProductInfo_toProviderAttribute(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		attr          string
		ProductInfoer map[string]ProductInfoer
		checker       func(str string, err error)
	}{
		{
			name:     "get cpu",
			provider: "dummy",
			attr:     Cpu,
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			checker: func(str string, err error) {
				assert.Equal(t, "vcpu", str)
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name:     "get memory",
			provider: "dummy",
			attr:     Memory,
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			checker: func(str string, err error) {
				assert.Equal(t, "memory", str)
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name:     "invalid attribute",
			provider: "dummy",
			attr:     "invalid",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			checker: func(str string, err error) {
				assert.Equal(t, "", str)
				assert.EqualError(t, err, "unsupported attribute: invalid")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, cache.New(5*time.Minute, 10*time.Minute), test.ProductInfoer)
			values, err := productInfo.toProviderAttribute(test.provider, test.attr)
			test.checker(values, err)
		})
	}
}

func TestCachingProductInfo_GetZones(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		region        string
		ProductInfoer map[string]ProductInfoer
		checker       func(cpi *CachingProductInfo, zones []string, err error)
	}{
		{
			name:     "zones retrieved and cached",
			provider: "dummy",
			region:   "dummyRegion",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			checker: func(cpi *CachingProductInfo, str []string, err error) {
				assert.Equal(t, []string{"Zone1", "Zone2"}, str)
				assert.Nil(t, err, "the error should be nil")

				// get the values from the cache
				cachedZones, _ := cpi.vmAttrStore.Get(cpi.getZonesKey("dummy", "dummyRegion"))
				assert.EqualValues(t, []string{"Zone1", "Zone2"}, cachedZones, "zones not cached")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, cache.New(5*time.Minute, 10*time.Minute), test.ProductInfoer)
			values, err := productInfo.GetZones(test.provider, test.region)
			test.checker(productInfo, values, err)
		})
	}
}

func TestCachingProductInfo_Initialize(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		ProductInfoer map[string]ProductInfoer
		checker       func(price map[string]map[string]Price, err error)
	}{
		{
			name:     "successful - store the result in cache",
			provider: "dummy",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			checker: func(price map[string]map[string]Price, err error) {
				assert.Equal(t, map[string]map[string]Price{"dummy": {"t2.nano": {OnDemandPrice: 25, SpotPrice: SpotPriceInfo{"testZone2": 43}}}}, price)
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name:     "could not get the output of the Infoer's Initialize function",
			provider: "dummy",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{TcId: 1},
			},
			checker: func(price map[string]map[string]Price, err error) {
				assert.Nil(t, price, "the price should be nil")
				assert.EqualError(t, err, "initialization error")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, cache.New(5*time.Minute, 10*time.Minute), test.ProductInfoer)
			values, err := productInfo.Initialize(test.provider)
			test.checker(values, err)
		})
	}
}

func TestCachingProductInfo_renewShortLivedInfo(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		region        string
		ProductInfoer map[string]ProductInfoer
		checker       func(price map[string]Price, err error)
	}{
		{
			name:     "successful - retrieve attribute values from the cloud provider",
			provider: "dummy",
			region:   "dummyRegion",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			checker: func(price map[string]Price, err error) {
				assert.Equal(t, map[string]Price{"t2.nano": {OnDemandPrice: 32, SpotPrice: SpotPriceInfo{"testZone1": 32}}}, price)
				assert.Equal(t, 1, len(price))
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name:     "error - could not get current prices",
			provider: "dummy",
			region:   "dummyRegion",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{TcId: 3},
			},
			checker: func(price map[string]Price, err error) {
				assert.Nil(t, price, "the price should be nil")
				assert.EqualError(t, err, "could not get current prices")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, cache.New(5*time.Minute, 10*time.Minute), test.ProductInfoer)
			values, err := productInfo.renewShortLivedInfo(test.provider, test.region)
			test.checker(values, err)
		})
	}
}

func TestCachingProductInfo_GetPrice(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		region        string
		instanceType  string
		p             Price
		zones         []string
		ProductInfoer map[string]ProductInfoer
		checker       func(i float64, f float64, err error)
	}{
		{
			name:         "successful - return on demand price and averaged spot price",
			provider:     "dummy",
			region:       "dummyRegion",
			instanceType: "t2.nano",
			zones:        []string{"testZone1", "testZone2"},
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			checker: func(i float64, f float64, err error) {
				assert.Equal(t, float64(32), i)
				assert.Equal(t, float64(16), f)
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name:         "error - could not get current prices",
			provider:     "dummy",
			region:       "dummyRegion",
			instanceType: "t2.nano",
			zones:        []string{"testZone1", "testZone2"},
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{TcId: 3},
			},
			checker: func(i float64, f float64, err error) {
				assert.Equal(t, float64(0), i)
				assert.Equal(t, float64(0), f)
				assert.EqualError(t, err, "could not get current prices")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, cache.New(5*time.Minute, 10*time.Minute), test.ProductInfoer)
			values, value, err := productInfo.GetPrice(test.provider, test.region, test.instanceType, test.zones)
			test.checker(values, value, err)
		})
	}
}

func TestCachingProductInfo_HasShortLivedPriceInfo(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		ProductInfoer map[string]ProductInfoer
		checker       func(bool)
	}{
		{
			name:     "return true",
			provider: "dummy",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			checker: func(b bool) {
				assert.Equal(t, true, b)
			},
		},
		{
			name:     "return false",
			provider: "dummy",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{TcId: 2},
			},
			checker: func(b bool) {
				assert.Equal(t, false, b)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, cache.New(5*time.Minute, 10*time.Minute), test.ProductInfoer)
			values := productInfo.HasShortLivedPriceInfo(test.provider)
			test.checker(values)
		})
	}
}
