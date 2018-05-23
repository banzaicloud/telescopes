package ec2

import (
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/banzaicloud/cluster-recommender/productinfo"
	log "github.com/sirupsen/logrus"
)

// Ec2Infoer encapsulates the data and operations needed to access external resources
type Ec2Infoer struct {
	pricing *pricing.Pricing
}

// NewEc2Infoer creates a new instance of the infoer
func NewEc2Infoer(pricing *pricing.Pricing) (*Ec2Infoer, error) {

	return &Ec2Infoer{
		pricing: pricing,
	}, nil
}

func NewPricing(cfg *aws.Config) *pricing.Pricing {

	s, err := session.NewSession(cfg)
	if err != nil {
		log.Fatalf("could not create session. error: [%s]", err.Error())
	}

	pr := pricing.New(s, cfg)
	return pr
}

func NewConfig() *aws.Config {
	// getting the reference can be extracted
	cfg := &aws.Config{}

	return cfg
}

func (e *Ec2Infoer) GetAttributeValues(attribute string) (productinfo.AttrValues, error) {
	apiValues, err := e.pricing.GetAttributeValues(e.newAttributeValuesInput(attribute))
	if err != nil {
		return nil, err
	}
	var values productinfo.AttrValues
	for _, v := range apiValues.AttributeValues {
		dotValue := strings.Replace(*v.Value, ",", ".", -1)
		floatValue, err := strconv.ParseFloat(strings.Split(dotValue, " ")[0], 64)
		if err != nil {
			log.Warnf("Couldn't parse attribute Value: [%s=%s]: %v", attribute, dotValue, err.Error())
		}
		values = append(values, productinfo.AttrValue{
			Value:    floatValue,
			StrValue: *v.Value,
		})
	}
	log.Debugf("found %s values: %v", attribute, values)
	return values, nil
}

func (e *Ec2Infoer) GetProducts(regionId string, attrKey string, attrValue productinfo.AttrValue) ([]productinfo.Ec2Vm, error) {

	var vms []productinfo.Ec2Vm
	log.Debugf("Getting available instance types from AWS API. [region=%s, %s=%s]", regionId, attrKey, attrValue.StrValue)

	products, err := e.pricing.GetProducts(e.newGetProductsInput(regionId, attrKey, attrValue))

	if err != nil {
		return nil, err
	}
	for _, price := range products.PriceList {
		var onDemandPrice float64
		// TODO: this is unsafe, check for nil values if needed
		instanceType := price["product"].(map[string]interface{})["attributes"].(map[string]interface{})["instanceType"].(string)
		cpusStr := price["product"].(map[string]interface{})["attributes"].(map[string]interface{})[productinfo.Cpu].(string)
		memStr := price["product"].(map[string]interface{})["attributes"].(map[string]interface{})[productinfo.Memory].(string)
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
		vm := productinfo.Ec2Vm{
			Type:          instanceType,
			OnDemandPrice: onDemandPrice,
			Cpus:          cpus,
			Mem:           mem,
			Gpus:          gpus,
		}
		vms = append(vms, vm)
	}
	log.Debugf("found vms [%s=%s]: %#v", attrKey, attrValue.StrValue, vms)
	return vms, nil
}

func (e *Ec2Infoer) GetRegion(id string) *endpoints.Region {
	awsp := endpoints.AwsPartition()
	for _, r := range awsp.Regions() {
		if r.ID() == id {
			return &r
		}
	}
	return nil
}

// newAttributeValuesInput assembles a GetAttributeValuesInput instance for querying the provider
func (e *Ec2Infoer) newAttributeValuesInput(attr string) *pricing.GetAttributeValuesInput {
	return &pricing.GetAttributeValuesInput{
		ServiceCode:   aws.String("AmazonEC2"),
		AttributeName: aws.String(attr),
	}
}

// newAttributeValuesInput assembles a GetAttributeValuesInput instance for querying the provider
func (e *Ec2Infoer) newGetProductsInput(regionId string, attrKey string, attrValue productinfo.AttrValue) *pricing.GetProductsInput {
	return &pricing.GetProductsInput{

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
				Value: aws.String(e.GetRegion(regionId).Description()),
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
				Value: aws.String(attrValue.StrValue),
			},
		},
	}
}

func (e *Ec2Infoer) GetRegions() map[string]string {
	regionIdMap := make(map[string]string)
	for key, region := range endpoints.AwsPartition().Regions() {
		regionIdMap[key] = region.ID()
	}
	return regionIdMap
}
