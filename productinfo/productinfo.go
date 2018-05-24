package productinfo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

const (
	// Memory represents the memory attribute for the recommender
	Memory = "memory"

	// Cpu represents the cpu attribute for the recommender
	Cpu = "vcpu"

	// VmKeyTemplate format for generating vm cache keys
	VmKeyTemplate = "/banzaicloud.com/recommender/ec2/%s/vms/%s/%f"

	// AttrKeyTemplate format for generating attribute cache keys
	AttrKeyTemplate = "/banzaicloud.com/recommender/ec2/attrValues/%s"
)

// ProductInfoer gathers operations for retrieving cloud provider information for recommendations
// it also decouples provider api specific code from the recommender
type ProductInfoer interface {
	// GetAttributeValues gets the attribute values for the given attribute from the external system
	GetAttributeValues(attribute string) (AttrValues, error)

	// GetProducts gets product information based on the given arguments from an external system
	GetProducts(regionId string, attrKey string, attrValue AttrValue) ([]Ec2Vm, error)

	// GetRegions retrieves the available regions form the external system
	GetRegions() map[string]string
}

// ProductInfo is the module struct, holds configuration and cache
// It's the entry point for the product info retrieval and management subsystem
type ProductInfo struct {
	productInfoer   ProductInfoer `validate:"required"`
	renewalInterval time.Duration
	vmAttrStore     *cache.Cache
}

// AttrValue represents an attribute value
type AttrValue struct {
	StrValue string
	Value    float64
}

// AttrValues a slice of AttrValues
type AttrValues []AttrValue

func (v AttrValues) floatValues() []float64 {
	floatValues := make([]float64, len(v))
	for _, av := range v {
		floatValues = append(floatValues, av.Value)
	}
	return floatValues
}

// Ec2Vm representation of a virtual machine
type Ec2Vm struct {
	Type          string  `json:"type"`
	OnDemandPrice float64 `json:"onDemandPrice"`
	Cpus          float64 `json:"cpusPerVm"`
	Mem           float64 `json:"memPerVm"`
	Gpus          float64 `json:"gpusPerVm"`
}

// NewProductInfo creates a new ProductInfo instance
func NewProductInfo(ri time.Duration, cache *cache.Cache, provider ProductInfoer) (*ProductInfo, error) {

	if provider == nil || cache == nil {
		return nil, errors.New("vould not create product infoer")
	}

	pi := ProductInfo{
		productInfoer:   provider,
		vmAttrStore:     cache,
		renewalInterval: ri,
	}

	// todo add validator here
	return &pi, nil
}

// Start starts the information retrieval in a new goroutine
func (pi *ProductInfo) Start(ctx context.Context) {

	renew := func() {
		log.Info("renewing product info")
		attributes := []string{Memory, Cpu}
		for _, attr := range attributes {
			attrValues, err := pi.renewAttrValues(attr)
			if err != nil {
				log.Errorf("couldn't renew ec2 attribute values in cache", err.Error())
				return
			}

			for _, regionId := range pi.productInfoer.GetRegions() {
				for _, v := range attrValues {
					_, err := pi.renewVmsWithAttr(regionId, attr, v)
					if err != nil {
						log.Errorf("couldn't renew ec2 attribute values in cache", err.Error())
					}
				}
			}
		}
		log.Info("finished renewing product info")
	}

	go renew()
	ticker := time.NewTicker(pi.renewalInterval)
	for {
		select {
		case <-ticker.C:
			renew()
		case <-ctx.Done():
			log.Debugf("closing ticker")
			ticker.Stop()
			return
		}
	}
}

// GetAttrValues returns a slice with the values for the given attribute name
func (pi *ProductInfo) GetAttrValues(attribute string) ([]float64, error) {
	v, err := pi.getAttrValues(attribute)
	if err != nil {
		return nil, err
	}
	return v.floatValues(), nil
}

func (pi *ProductInfo) getAttrValues(attribute string) (AttrValues, error) {
	attrCacheKey := pi.getAttrKey(attribute)
	if cachedVal, ok := pi.vmAttrStore.Get(attrCacheKey); ok {
		log.Debugf("Getting available %s values from cache.", attribute)
		return cachedVal.(AttrValues), nil
	}
	values, err := pi.renewAttrValues(attribute)
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (pi *ProductInfo) getAttrKey(attribute string) string {
	return fmt.Sprintf(AttrKeyTemplate, attribute)
}

// renewAttrValues retrieves attribute values from the cloud provider and refreshes the attribute store with them
func (pi *ProductInfo) renewAttrValues(attribute string) (AttrValues, error) {
	values, err := pi.productInfoer.GetAttributeValues(attribute)
	if err != nil {
		return nil, err
	}
	pi.vmAttrStore.Set(pi.getAttrKey(attribute), values, pi.renewalInterval)
	return values, nil
}

// GetVmsWithAttrValue returns a slice with the virtual machines for the given region, attribute and value
func (pi *ProductInfo) GetVmsWithAttrValue(regionId string, attrKey string, value float64) ([]Ec2Vm, error) {

	log.Debugf("Getting instance types and on demand prices. [regionId=%s, %s=%v]", regionId, attrKey, value)
	vmCacheKey := pi.getVmKey(regionId, attrKey, value)
	if cachedVal, ok := pi.vmAttrStore.Get(vmCacheKey); ok {
		log.Debugf("Getting available instance types from cache. [regionId=%s, %s=%v]", regionId, attrKey, value)
		return cachedVal.([]Ec2Vm), nil
	}
	attrValue, err := pi.getAttrValue(attrKey, value)
	if err != nil {
		return nil, err
	}
	vms, err := pi.renewVmsWithAttr(regionId, attrKey, *attrValue)
	if err != nil {
		return nil, err
	}
	return vms, nil
}

func (pi *ProductInfo) getVmKey(region string, attrKey string, attrValue float64) string {
	return fmt.Sprintf(VmKeyTemplate, region, attrKey, attrValue)
}

func (pi *ProductInfo) renewVmsWithAttr(regionId string, attrKey string, attrValue AttrValue) ([]Ec2Vm, error) {
	values, err := pi.productInfoer.GetProducts(regionId, attrKey, attrValue)
	if err != nil {
		return nil, err
	}
	pi.vmAttrStore.Set(pi.getVmKey(regionId, attrKey, attrValue.Value), values, pi.renewalInterval)
	return values, nil
}

func (pi *ProductInfo) getAttrValue(attrKey string, attrValue float64) (*AttrValue, error) {
	attrValues, err := pi.getAttrValues(attrKey)
	if err != nil {
		return nil, err
	}
	for _, av := range attrValues {
		if av.Value == attrValue {
			return &av, nil
		}
	}
	return nil, errors.New("couldn't find attribute Value")
}
