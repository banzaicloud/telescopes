package recommender

import (
	"errors"
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

type Ec2VmRegistry struct {
	Session     *session.Session
	VmAttrStore *cache.Cache
}

func NewEc2VmRegistry(region string, cache *cache.Cache) (VmRegistry, error) {
	session, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		log.WithError(err).Error("Error creating AWS session")
		return nil, err
	}
	return &Ec2VmRegistry{
		Session:     session,
		VmAttrStore: cache,
	}, nil
}

// TODO: collect VM info in a separate goroutine (ticker)

// TODO: unit tests
func (e *Ec2VmRegistry) findVmsWithCpuUnits(cpuUnits []float64) ([]VirtualMachine, error) {

	pricingSvc := pricing.New(e.Session, &aws.Config{Region: aws.String("us-east-1")})
	log.Infof("Getting instance types and on demand prices with %v vcpus", cpuUnits)

	var allVms []VirtualMachine
	for _, cpu := range cpuUnits {
		vmCacheKey := "/banzaicloud.com/recommender/ec2/attrValues/" + fmt.Sprint(cpu)
		if cachedVal, ok := e.VmAttrStore.Get(vmCacheKey); ok {
			log.Debugf("Getting available instance types with %v cpu from cache.", cpu)
			log.Debug(cachedVal.([]VirtualMachine))
			allVms = append(allVms, cachedVal.([]VirtualMachine)...)
		} else {
			var vms []VirtualMachine
			log.Debugf("Getting available instance types with %v cpu from AWS API.", cpu)
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
						Field: aws.String("vcpu"),
						Value: aws.String(fmt.Sprint(cpu)),
					},
					{
						Type:  aws.String("TERM_MATCH"),
						Field: aws.String("location"),
						Value: aws.String(regionMap[*e.Session.Config.Region]),
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
				},
			})
			if err != nil {
				return nil, err
			}
			for _, price := range products.PriceList {
				var onDemandPrice float64
				// TODO: this is unsafe, check for nil values if needed
				instanceType := price["product"].(map[string]interface{})["attributes"].(map[string]interface{})["instanceType"].(string)
				cpusStr := price["product"].(map[string]interface{})["attributes"].(map[string]interface{})["vcpu"].(string)
				memStr := price["product"].(map[string]interface{})["attributes"].(map[string]interface{})["memory"].(string)
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
				vm := VirtualMachine{
					Type:          instanceType,
					OnDemandPrice: onDemandPrice,
					// TODO: spot price
					AvgPrice: onDemandPrice,
					Cpus:     cpus,
					Mem:      mem,
					Gpus:     gpus,
				}
				vms = append(vms, vm)
			}
			e.VmAttrStore.Set(vmCacheKey, vms, 24*time.Hour)
			allVms = append(allVms, vms...)
		}
	}
	log.Debugf("found vms with cpu units [%v]: %v", cpuUnits, allVms)
	return allVms, nil
}

// TODO: unit tests
func (e *Ec2VmRegistry) findCpuUnits(min float64, max float64) ([]float64, error) {
	log.Debugf("finding cpu units between: [%v, %v]", min, max)
	pricingSvc := pricing.New(e.Session, &aws.Config{Region: aws.String("us-east-1")})
	cpuValues, err := e.getSortedAttrValues(pricingSvc, "vcpu")
	if err != nil {
		return nil, err
	}

	if min > max {
		return nil, errors.New("min value cannot be larger than the max value")
	}

	if max < cpuValues[0] {
		log.Debug("returning smallest CPU unit: %v", cpuValues[0])
		return []float64{cpuValues[0]}, nil
	} else if min > cpuValues[len(cpuValues)-1] {
		log.Debugf("returning largest CPU unit: %v", cpuValues[len(cpuValues)-1])
		return []float64{cpuValues[len(cpuValues)-1]}, nil
	}

	var values []float64

	for i := 0; i < len(cpuValues); i++ {
		if cpuValues[i] >= min && cpuValues[i] <= max {
			values = append(values, cpuValues[i])
		} else if cpuValues[i] > max && len(values) < 1 {
			// 1 2 4 8 16 32 64....
			// min value: 4.2 max 7.8
			log.Debugf("couldn't find values between min and max, returning nearest values: [%v, %v]", cpuValues[i-1], cpuValues[i])
			return []float64{cpuValues[i-1], cpuValues[i]}, nil
		}
	}
	log.Debugf("returning CPU units: %v", values)
	return values, nil
}

func (e *Ec2VmRegistry) getSortedAttrValues(pricingSvc *pricing.Pricing, attribute string) ([]float64, error) {
	var attrValues []float64
	attrCacheKey := "/banzaicloud.com/recommender/ec2/attrValues/" + attribute
	if cachedVal, ok := e.VmAttrStore.Get(attrCacheKey); ok {
		log.Debugf("Getting available %s values from cache.", attribute)
		attrValues = cachedVal.([]float64)
	} else {
		log.Debugf("Getting available %s values from AWS API.", attribute)
		apiValues, err := pricingSvc.GetAttributeValues(&pricing.GetAttributeValuesInput{
			ServiceCode:   aws.String("AmazonEC2"),
			AttributeName: aws.String(attribute),
		})
		if err != nil {
			return nil, err
		}

		var values []float64
		for _, attrValue := range apiValues.AttributeValues {
			floatValue, err := strconv.ParseFloat(strings.Split(*attrValue.Value, " ")[0], 32)
			if err != nil {
				return nil, err
			}
			values = append(values, floatValue)
		}
		sort.Float64s(values)
		attrValues = values
		e.VmAttrStore.Set(attrCacheKey, values, 24*time.Hour)
	}
	log.Debugf("%s attribute values sorted: %v", attribute, attrValues)
	return attrValues, nil
}
