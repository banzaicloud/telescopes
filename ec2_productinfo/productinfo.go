package ec2_productinfo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/banzaicloud/cluster-recommender/cloudprovider"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"gopkg.in/go-playground/validator.v9"
)

const (
	Memory = "memory"
	Cpu    = "vcpu"
)

var (
	validate *validator.Validate
)

func init() {
	validate = validator.New()
}

type ProductInfo struct {
	CloudInfoProvider cloudprovider.CloudProductInfoProvider `validate:"required"`
	renewalInterval   time.Duration
	vmAttrStore       *cache.Cache
}

type AttrValue struct {
	StrValue string
	value    float64
}

type AttrValues []AttrValue

func (v AttrValues) floatValues() []float64 {
	floatValues := make([]float64, len(v))
	for _, av := range v {
		floatValues = append(floatValues, av.value)
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

func NewProductInfo(ri time.Duration, cache *cache.Cache, provider cloudprovider.CloudProductInfoProvider) (*ProductInfo, error) {
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

			for _, r := range pi.CloudInfoProvider.GetRegions() {
				for _, v := range attrValues {
					_, err := pi.renewVmsWithAttr(r.ID(), attr, v)
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
	return fmt.Sprintf("/banzaicloud.com/recommender/ec2/attrValues/%s", attribute)
}

func (pi *ProductInfo) renewAttrValues(attribute string) (AttrValues, error) {
	values, err := pi.getAttrValuesFromAPI(attribute)
	if err != nil {
		return nil, err
	}
	pi.vmAttrStore.Set(pi.getAttrKey(attribute), values, pi.renewalInterval)
	return values, nil
}

func (pi *ProductInfo) getAttrValuesFromAPI(attribute string) (AttrValues, error) {

	apiValues, err := pi.CloudInfoProvider.GetAttributeValues(attribute)
	if err != nil {
		return nil, err
	}
	var values AttrValues
	for _, v := range apiValues.AttributeValues {
		dotValue := strings.Replace(*v.Value, ",", ".", -1)
		floatValue, err := strconv.ParseFloat(strings.Split(dotValue, " ")[0], 64)
		if err != nil {
			logrus.Warnf("Couldn't parse attribute value: [%s=%s]: %v", attribute, dotValue, err.Error())
		}
		values = append(values, AttrValue{
			value:    floatValue,
			StrValue: *v.Value,
		})
	}
	logrus.Debugf("found %s values: %v", attribute, values)
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
	return fmt.Sprintf("/banzaicloud.com/recommender/ec2/%s/vms/%s/%s", region, attrKey, strconv.FormatFloat(attrValue, 'b', 2, 5))
}

func (pi *ProductInfo) renewVmsWithAttr(regionId string, attrKey string, attrValue AttrValue) ([]Ec2Vm, error) {
	values, err := pi.getVmsWithAttrFromAPI(regionId, attrKey, attrValue)
	if err != nil {
		return nil, err
	}
	pi.vmAttrStore.Set(pi.getVmKey(regionId, attrKey, attrValue.value), values, pi.renewalInterval)
	return values, nil
}

func (pi *ProductInfo) getVmsWithAttrFromAPI(regionId string, attrKey string, attrValue AttrValue) ([]Ec2Vm, error) {
	var vms []Ec2Vm
	logrus.Debugf("Getting available instance types from AWS API. [region=%s, %s=%s]", regionId, attrKey, attrValue.StrValue)

	products, err := pi.CloudInfoProvider.GetProducts(regionId, attrKey, attrValue.StrValue)

	if err != nil {
		return nil, err
	}
	for _, price := range products.PriceList {
		var onDemandPrice float64
		// TODO: this is unsafe, check for nil values if needed
		instanceType := price["product"].(map[string]interface{})["attributes"].(map[string]interface{})["instanceType"].(string)
		cpusStr := price["product"].(map[string]interface{})["attributes"].(map[string]interface{})[Cpu].(string)
		memStr := price["product"].(map[string]interface{})["attributes"].(map[string]interface{})[Memory].(string)
		var gpus float64
		if price["product"].(map[string]interface{})["attributes"].(map[string]interface{})["gpu"] != nil {
			gpuStr := price["product"].(map[string]interface{})["attributes"].(map[string]interface{})["gpu"].(string)
			gpus, _ = strconv.ParseFloat(gpuStr, 32)
		}
		onDemandTerm := price["terms"].(map[string]interface{})["OnDemand"].(map[string]interface{})
		for _, term := range onDemandTerm {
			priceDimensions := term.(map[string]interface{})["priceDimensions"].(map[string]interface{})
			for _, dimension := range priceDimensions {
				odPriceStr := dimension.(map[string]interface{})["pricePerUnit"].(map[string]interface{})["USD"].(string)
				onDemandPrice, _ = strconv.ParseFloat(odPriceStr, 32)
			}
		}
		cpus, _ := strconv.ParseFloat(cpusStr, 32)
		mem, _ := strconv.ParseFloat(strings.Split(memStr, " ")[0], 32)
		vm := Ec2Vm{
			Type:          instanceType,
			OnDemandPrice: onDemandPrice,
			Cpus:          cpus,
			Mem:           mem,
			Gpus:          gpus,
		}
		vms = append(vms, vm)
	}
	logrus.Debugf("found vms [%s=%s]: %#v", attrKey, attrValue.StrValue, vms)
	return vms, nil
}

func (pi *ProductInfo) getAttrValue(attrKey string, attrValue float64) (*AttrValue, error) {
	attrValues, err := pi.getAttrValues(attrKey)
	if err != nil {
		return nil, err
	}
	for _, av := range attrValues {
		if av.value == attrValue {
			return &av, nil
		}
	}
	return nil, errors.New("couldn't find attribute value")
}
