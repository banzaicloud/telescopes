package ec2_productinfo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"gopkg.in/go-playground/validator.v9"
)

const (
	Memory          = "memory"
	Cpu             = "vcpu"
	VmKeyTemplate   = "/banzaicloud.com/recommender/ec2/%s/vms/%s/%f"
	AttrKeyTemplate = "/banzaicloud.com/recommender/ec2/attrValues/%s"
)

var (
	validate *validator.Validate
)

func init() {
	validate = validator.New()
}

type ProductInfo struct {
	CloudInfoProvider ProductInfoer `validate:"required"`
	renewalInterval   time.Duration
	vmAttrStore       *cache.Cache
}

type AttrValue struct {
	StrValue string
	Value    float64
}

type AttrValues []AttrValue

func (v AttrValues) floatValues() []float64 {
	floatValues := make([]float64, len(v))
	for _, av := range v {
		floatValues = append(floatValues, av.Value)
	}
	return floatValues
}

type Ec2Vm struct {
	Type          string  `json:"type"`
	OnDemandPrice float64 `json:"onDemandPrice"`
	Cpus          float64 `json:"cpusPerVm"`
	Mem           float64 `json:"memPerVm"`
	Gpus          float64 `json:"gpusPerVm"`
}

func NewProductInfo(ri time.Duration, cache *cache.Cache, provider ProductInfoer) (*ProductInfo, error) {
	pi := ProductInfo{
		CloudInfoProvider: provider,
		vmAttrStore:       cache,
		renewalInterval:   ri,
	}

	err := validate.Struct(pi)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %s", err.Error())
	}

	return &pi, nil
}

func (pi *ProductInfo) Start(ctx context.Context) {

	renew := func() {
		logrus.Info("renewing product info")
		attributes := []string{Memory, Cpu}
		for _, attr := range attributes {
			attrValues, err := pi.renewAttrValues(attr)
			if err != nil {
				logrus.Errorf("couldn't renew ec2 attribute values in cache", err.Error())
				return
			}

			for _, regionId := range pi.CloudInfoProvider.GetRegions() {
				for _, v := range attrValues {
					_, err := pi.renewVmsWithAttr(regionId, attr, v)
					if err != nil {
						logrus.Errorf("couldn't renew ec2 attribute values in cache", err.Error())
					}
				}
			}
		}
		logrus.Info("finished renewing product info")
	}

	go renew()
	ticker := time.NewTicker(pi.renewalInterval)
	for {
		select {
		case <-ticker.C:
			renew()
		case <-ctx.Done():
			logrus.Debugf("closing ticker")
			ticker.Stop()
			return
		}
	}
}

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
		logrus.Debugf("Getting available %s values from cache.", attribute)
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
	values, err := pi.CloudInfoProvider.GetAttributeValues(attribute)
	if err != nil {
		return nil, err
	}
	pi.vmAttrStore.Set(pi.getAttrKey(attribute), values, pi.renewalInterval)
	return values, nil
}

func (pi *ProductInfo) GetVmsWithAttrValue(regionId string, attrKey string, value float64) ([]Ec2Vm, error) {

	logrus.Debugf("Getting instance types and on demand prices. [regionId=%s, %s=%v]", regionId, attrKey, value)
	vmCacheKey := pi.getVmKey(regionId, attrKey, value)
	if cachedVal, ok := pi.vmAttrStore.Get(vmCacheKey); ok {
		logrus.Debugf("Getting available instance types from cache. [regionId=%s, %s=%v]", regionId, attrKey, value)
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
	values, err := pi.CloudInfoProvider.GetProducts(regionId, attrKey, attrValue)
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
