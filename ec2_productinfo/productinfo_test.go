package ec2_productinfo

import (
	"testing"
	"time"

	"github.com/banzaicloud/cluster-recommender/cloudprovider"
	"github.com/stretchr/testify/assert"
)

type DummyProductInfoProvider struct {
	// implement the interface
	cloudprovider.CloudProductInfoProvider
}

func TestNewProductInfo(t *testing.T) {
	testCases := []struct {
		Name                string
		ProductInfoProvider cloudprovider.CloudProductInfoProvider
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
