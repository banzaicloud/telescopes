package ec2

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/banzaicloud/cluster-recommender/productinfo"
	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
)

const (
	// Memory represents the memory attribute for the recommender
	Memory = "memory"

	// Cpu represents the cpu attribute for the recommender
	Cpu = "vcpu"
)

// PricingSource list of operations for retrieving pricing information
// Decouples the pricing logic from the aws api
type PricingSource interface {
	GetAttributeValues(input *pricing.GetAttributeValuesInput) (*pricing.GetAttributeValuesOutput, error)
	GetProducts(input *pricing.GetProductsInput) (*pricing.GetProductsOutput, error)
}

// Ec2Infoer encapsulates the data and operations needed to access external resources
type Ec2Infoer struct {
	pricing    PricingSource
	session    *session.Session
	prometheus v1.API
	promQuery  string
}

// NewEc2Infoer creates a new instance of the infoer
func NewEc2Infoer(pricing PricingSource, prom string, pq string) (*Ec2Infoer, error) {
	s, err := session.NewSession()
	if err != nil {
		log.WithError(err).Error("Error creating AWS session")
		return nil, err
	}
	var promApi v1.API
	if prom == "" {
		log.Warn("Prometheus API address is not set, fallback to direct API access.")
		promApi = nil
	} else {
		promClient, err := api.NewClient(api.Config{
			Address: prom,
		})
		if err != nil {
			log.WithError(err).Warn("Error creating Prometheus client, fallback to direct API access.")
			promApi = nil
		} else {
			promApi = v1.NewAPI(promClient)
		}
	}

	return &Ec2Infoer{
		pricing:    pricing,
		session:    s,
		prometheus: promApi,
		promQuery:  pq,
	}, nil
}

// NewPricing creates a new PricingSource with the given configuration
func NewPricing(cfg *aws.Config) PricingSource {

	s, err := session.NewSession(cfg)
	if err != nil {
		log.Fatalf("could not create session. error: [%s]", err.Error())
	}

	pr := pricing.New(s, cfg)
	return pr
}

// NewConfig creates a new  Config instance and returns a pointer to it
// todo the region to be passed as argument
func NewConfig() *aws.Config {
	// getting the reference can be extracted
	cfg := &aws.Config{Region: aws.String("us-east-1")}

	return cfg
}

// GetAttributeValues gets the AttributeValues for the given attribute name
// Delegates to the underlying PricingSource instance and unifies (transforms) the response
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

// GetProducts retrieves the available virtual machines based on the arguments provided
// Delegates to the underlying PricingSource instance and performs transformations
func (e *Ec2Infoer) GetProducts(regionId string, attrKey string, attrValue productinfo.AttrValue) ([]productinfo.VmInfo, error) {

	var vms []productinfo.VmInfo
	log.Debugf("Getting available instance types from AWS API. [region=%s, %s=%s]", regionId, attrKey, attrValue.StrValue)

	products, err := e.pricing.GetProducts(e.newGetProductsInput(regionId, attrKey, attrValue))

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
		vm := productinfo.VmInfo{
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

// GetRegion gets the api specific region representation based on the provided id
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

// GetRegions returns a map with available regions
// transforms the api representation into a "plain" map
func (e *Ec2Infoer) GetRegions() map[string]string {
	regionIdMap := make(map[string]string)
	for key, region := range endpoints.AwsPartition().Regions() {
		regionIdMap[key] = region.ID()
	}
	return regionIdMap
}

// GetZones returns the availability zones in a region
func (e *Ec2Infoer) GetZones(region string) ([]string, error) {
	var zones []string
	ec2Svc := ec2.New(e.session, &aws.Config{Region: aws.String(region)})
	azs, err := ec2Svc.DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{})
	if err != nil {
		return nil, err
	}
	for _, az := range azs.AvailabilityZones {
		if *az.State == "available" {
			zones = append(zones, *az.ZoneName)
		}
	}
	return zones, nil
}

func (e *Ec2Infoer) GetSpotPriceAvgsFromPrometheus(region string, zones []string, instanceTypes []string) (map[string]float64, error) {
	log.Debug("getting spot price averages from Prometheus API")
	avgSpotPrices := make(map[string]float64, len(instanceTypes))
	for _, it := range instanceTypes {
		query := fmt.Sprintf(e.promQuery, region, it, strings.Join(zones, "|"))
		log.Debugf("sending prometheus query: %s", query)
		result, err := e.prometheus.Query(context.Background(), query, time.Now())
		if err != nil {
			return nil, err
		} else if result.String() == "" {
			log.Warnf("Prometheus metric is empty, instance type won't be recommended [type=%s]", it)
		} else {
			r := result.(model.Vector)
			log.Debugf("query result: %s", result.String())
			if len(r) > 0 {
				avgPrice, err := strconv.ParseFloat(r[0].Value.String(), 64)
				if err != nil {
					return nil, err
				}
				avgSpotPrices[it] = avgPrice
			} else {
				log.Warnf("Prometheus metric is empty, instance type won't be recommended [type=%s]", it)
			}
		}
	}
	// query returned empty response for every instance type
	if len(avgSpotPrices) == 0 {
		return nil, errors.New("query returned empty response for every instance type")
	}
	return avgSpotPrices, nil
}

func (e *Ec2Infoer) GetCurrentSpotPrices(region string) (map[string]productinfo.PriceInfo, error) {

	//pricesParsed := false
	//if e.prometheus != nil {
	//	zoneAvgSpotPrices, err := e.getSpotPriceAvgsFromPrometheus(region, zones, instanceTypes)
	//	if err != nil {
	//		log.WithError(err).Warn("Couldn't get spot price info from Prometheus API, fallback to direct AWS API access.")
	//	} else {
	//		pricesParsed = true
	//		avgSpotPrices = zoneAvgSpotPrices
	//	}
	//}
	//
	//if e.prometheus == nil || !pricesParsed {
	//	log.Debug("getting current spot prices directly from the AWS API")
	//	currentZoneAvgSpotPrices, err := e.getCurrentSpotPrices(region, zones, instanceTypes)
	//	if err != nil {
	//		return nil, err
	//	}
	//	avgSpotPrices = currentZoneAvgSpotPrices
	//}


	priceInfo := make(map[string]productinfo.PriceInfo)

	ec2Svc := ec2.New(e.session, &aws.Config{Region: aws.String(region)})
	log.Info("AWS API request here!!!!!!")
	err := ec2Svc.DescribeSpotPriceHistoryPages(&ec2.DescribeSpotPriceHistoryInput{
		StartTime:           aws.Time(time.Now()),
		ProductDescriptions: []*string{aws.String("Linux/UNIX")},
	}, func(history *ec2.DescribeSpotPriceHistoryOutput, lastPage bool) bool {
		for _, pe := range history.SpotPriceHistory {
			price, err := strconv.ParseFloat(*pe.SpotPrice, 64)
			if err != nil {
				// TODO: it doesn't look good, at least log something
				return false
			}
			if priceInfo[*pe.InstanceType] == nil {
				priceInfo[*pe.InstanceType] = make(productinfo.PriceInfo)
			}
			priceInfo[*pe.InstanceType][*pe.AvailabilityZone] = price
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	return priceInfo, nil
}

func (e *Ec2Infoer) GetMemoryAttrName() string {
	return Memory
}

func (e *Ec2Infoer) GetCpuAttrName() string {
	return Cpu
}
