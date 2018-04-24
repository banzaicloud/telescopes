package ec2_productinfo

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

const (
	Memory = "memory"
	Cpu    = "vcpu"
)

type ProductInfo struct {
	renewalInterval time.Duration
	session         *session.Session
	vmAttrStore     *cache.Cache
}

type Ec2Vm struct {
	Type          string  `json:type`
	OnDemandPrice float64 `json:onDemandPrice`
	Cpus          float64 `json:cpusPerVm`
	Mem           float64 `json:memPerVm`
	Gpus          float64 `json:gpusPerVm`
}

var regionMap = map[string]string{
	"ap-northeast-1": "Asia Pacific (Tokyo)",
	"ap-northeast-2": "Asia Pacific (Seoul)",
	"ap-south-1":     "Asia Pacific (Mumbai)",
	"ap-southeast-1": "Asia Pacific (Singapore)",
	"ap-southeast-2": "Asia Pacific (Sydney)",
	"ca-central-1":   "Canada (Central)",
	"eu-central-1":   "EU (Frankfurt)",
	"eu-west-1":      "EU (Ireland)",
	"eu-west-2":      "EU (London)",
	"sa-east-1":      "South America (Sao Paulo)",
	"us-east-1":      "US East (N. Virginia)",
	"us-east-2":      "US East (Ohio)",
	"us-west-1":      "US West (N. California)",
	"us-west-2":      "US West (Oregon)",
}

func NewProductInfo(ri time.Duration, cache *cache.Cache) (*ProductInfo, error) {
	session, err := session.NewSession(&aws.Config{})
	if err != nil {
		log.WithError(err).Error("Error creating AWS session")
		return nil, err
	}
	return &ProductInfo{
		session:         session,
		vmAttrStore:     cache,
		renewalInterval: ri,
	}, nil
}

func (e *ProductInfo) Start(ctx context.Context) {

	renew := func() {
		log.Info("renewing product info")
		attributes := []string{Memory, Cpu}
		for _, attr := range attributes {
			attrValues, err := e.renewAttrValues(attr)
			if err != nil {
				log.Errorf("couldn't renew ec2 attribute values in cache", err.Error())
				return
			}
			for region := range regionMap {
				for _, value := range attrValues {
					_, err := e.renewVmsWithAttr(region, attr, value)
					if err != nil {
						log.Errorf("couldn't renew ec2 attribute values in cache", err.Error())
					}
				}
			}
		}
		log.Info("finished renewing product info")
	}

	go renew()
	ticker := time.NewTicker(e.renewalInterval)
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

func (e *ProductInfo) GetSortedAttrValues(attribute string) ([]float64, error) {
	attrCacheKey := e.getAttrKey(attribute)
	if cachedVal, ok := e.vmAttrStore.Get(attrCacheKey); ok {
		log.Debugf("Getting available %s values from cache.", attribute)
		return cachedVal.([]float64), nil
	}
	values, err := e.renewAttrValues(attribute)
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (e *ProductInfo) getAttrKey(attribute string) string {
	return fmt.Sprintf("/banzaicloud.com/recommender/ec2/attrValues/%s", attribute)
}

func (e *ProductInfo) renewAttrValues(attribute string) ([]float64, error) {
	values, err := e.getSortedAttrValuesFromAPI(attribute)
	if err != nil {
		return nil, err
	}
	e.vmAttrStore.Set(e.getAttrKey(attribute), values, e.renewalInterval)
	return values, nil
}

func (e *ProductInfo) getSortedAttrValuesFromAPI(attribute string) ([]float64, error) {
	log.Debugf("Getting available %s values from AWS API.", attribute)
	pricingSvc := pricing.New(e.session, &aws.Config{Region: aws.String("us-east-1")})
	apiValues, err := pricingSvc.GetAttributeValues(&pricing.GetAttributeValuesInput{
		ServiceCode:   aws.String("AmazonEC2"),
		AttributeName: aws.String(attribute),
	})
	if err != nil {
		return nil, err
	}
	var values []float64
	for _, attrValue := range apiValues.AttributeValues {
		dotValue := strings.Replace(*attrValue.Value, ",", ".", -1)
		floatValue, err := strconv.ParseFloat(strings.Split(dotValue, " ")[0], 32)
		if err != nil {
			log.Warnf("Couldn't parse attribute value: [%s=%s]: %v", attribute, dotValue, err.Error())
		}
		values = append(values, floatValue)
	}
	sort.Float64s(values)
	return values, nil
}

func (e *ProductInfo) GetVmsWithCpu(region string, attrKey string, attrValue float64) ([]Ec2Vm, error) {
	log.Debugf("Getting instance types and on demand prices. [region=%s, %s=%v]", region, attrKey, attrValue)
	vmCacheKey := e.getVmKey(region, attrKey, attrValue)
	if cachedVal, ok := e.vmAttrStore.Get(vmCacheKey); ok {
		log.Debugf("Getting available instance types from cache. [region=%s, %s=%v]", region, attrKey, attrValue)
		return cachedVal.([]Ec2Vm), nil
	}

	vms, err := e.renewVmsWithAttr(region, attrKey, attrValue)
	if err != nil {
		return nil, err
	}
	return vms, nil
}

func (e *ProductInfo) getVmKey(region string, attrKey string, attrValue float64) string {
	return fmt.Sprintf("/banzaicloud.com/recommender/ec2/%s/vms/%s/%s", region, attrKey, attrValue)
}

func (e *ProductInfo) renewVmsWithAttr(region string, attrKey string, attrValue float64) ([]Ec2Vm, error) {
	values, err := e.getVmsWithAttrFromAPI(region, attrKey, attrValue)
	if err != nil {
		return nil, err
	}
	e.vmAttrStore.Set(e.getVmKey(region, attrKey, attrValue), values, e.renewalInterval)
	return values, nil
}

func (e *ProductInfo) getVmsWithAttrFromAPI(region string, attrKey string, attrValue float64) ([]Ec2Vm, error) {
	var vms []Ec2Vm
	pricingSvc := pricing.New(e.session, &aws.Config{Region: aws.String("us-east-1")})
	log.Debugf("Getting available instance types from AWS API. [region=%s, %s=%v]", region, attrKey, attrValue)
	products, err := pricingSvc.GetProducts(&pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEC2"),
		Filters: []*pricing.Filter{
			{
				Type:  aws.String("TERM_MATCH"),
				Field: aws.String("operatingSystem"),
				Value: aws.String("Linux"),
			},
			{
				Type:  aws.String("TERM_MATCH"),
				Field: aws.String("location"),
				Value: aws.String(regionMap[region]),
			},
			{
				Type:  aws.String("TERM_MATCH"),
				Field: aws.String("tenancy"),
				Value: aws.String("shared"),
			},
			{
				Type:  aws.String("TERM_MATCH"),
				Field: aws.String("preInstalledSw"),
				Value: aws.String("NA"),
			},
			{
				Type:  aws.String("TERM_MATCH"),
				Field: aws.String(attrKey),
				Value: aws.String(fmt.Sprint(attrValue)),
			},
		},
	})
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
	return vms, nil
}
