package gce

import (
	"github.com/banzaicloud/cluster-recommender/productinfo"
)

// GceInfoer encapsulates the data and operations needed to access external resources
type GceInfoer struct {
}

// NewGceInfoer creates a new instance of the infoer
func NewGceInfoer() (*GceInfoer, error) {
	return &GceInfoer{
	}, nil
}

// GetAttributeValues gets the AttributeValues for the given attribute name
// Delegates to the underlying PricingSource instance and unifies (transforms) the response
func (g *GceInfoer) GetAttributeValues(attribute string) (productinfo.AttrValues, error) {
	var values productinfo.AttrValues
	return values, nil
}

// GetProducts retrieves the available virtual machines based on the arguments provided
// Delegates to the underlying PricingSource instance and performs transformations
func (g *GceInfoer) GetProducts(regionId string, attrKey string, attrValue productinfo.AttrValue) ([]productinfo.VmInfo, error) {
	var vms []productinfo.VmInfo
	return vms, nil
}

// GetRegions returns a map with available regions
// transforms the api representation into a "plain" map
func (g *GceInfoer) GetRegions() map[string]string {
	regionIdMap := make(map[string]string)
	return regionIdMap
}

// GetZones returns the availability zones in a region
func (g *GceInfoer) GetZones(region string) ([]string, error) {
	return []string{}, nil
}

func (g *GceInfoer) GetCurrentSpotPrices(region string) (map[string]productinfo.PriceInfo, error) {
	priceInfo := make(map[string]productinfo.PriceInfo)
	return priceInfo, nil
}

func (e *GceInfoer) GetMemoryAttrName() string {
	return "memory"
}

func (e *GceInfoer) GetCpuAttrName() string {
	return "cpu"
}
