package ec2_productinfo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/patrickmn/go-cache"
)

type DummyProductInfoProvider struct {
	// implement the interface
	CloudProductInfoProvider
}

func TestNewProductInfo(t *testing.T) {
	testCases := []struct {
		Name                string
		ProductInfoProvider CloudProductInfoProvider
		Assert              func(info *ProductInfo, err error)
	}{
		{
			Name:                "product info successfully created",
			ProductInfoProvider: &DummyProductInfoProvider{},
			Assert: func(info *ProductInfo, err error) {
				assert.Nil(t, err, "should not get error")
			},
		},
		{
			Name:                "validation should fail nil values",
			ProductInfoProvider: nil,
			Assert: func(info *ProductInfo, err error) {
				assert.Nil(t, info, "the productinfo should be nil in case of error")
				assert.NotNil(t, err, "should get validation error when nill values provided")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			tc.Assert(NewProductInfo(10*time.Second, nil, tc.ProductInfoProvider))
		})
	}

}

func TestRenewAttributeValues(t *testing.T) {
	testCases := []struct {
		Name                string
		ProductInfoProvider CloudProductInfoProvider
		Cache               *cache.Cache
		Attribute           string
		Assert              func(values AttrValues, err error)
	}{
		{
			Name:                "attribute successfully renewed",
			ProductInfoProvider: nil,
			Cache:               nil,
			Attribute:           Memory,
			Assert: func(values AttrValues, err error) {
				assert.Nil(t, err, "no error expected")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			productInfo, _ := NewProductInfo(10*time.Second, tc.Cache, tc.ProductInfoProvider)
			tc.Assert(productInfo.renewAttrValues(tc.Attribute))
		})
	}

}
