package recommender

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/banzaicloud/telescopes/productinfo"
	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
)

// Ec2VmRegistry represents a vm registry
type Ec2VmRegistry struct {
	session     *session.Session
	productInfo *productinfo.ProductInfo
	prometheus  v1.API
	promQuery   string
}

// NewEc2VmRegistry creates a new VmRegistry instance
func NewEc2VmRegistry(pi *productinfo.ProductInfo, prom string, pq string) (VmRegistry, error) {
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

	return &Ec2VmRegistry{
		session:     s,
		productInfo: pi,
		prometheus:  promApi,
		promQuery:   pq,
	}, nil
}

func (e *Ec2VmRegistry) findVmsWithAttrValues(region string, zones []string, attr string, values []float64) ([]VirtualMachine, error) {
	log.Infof("Getting instance types and on demand prices with %v %s", values, attr)
	var vms []VirtualMachine
	ec2Attr, err := e.toEC2Attribute(attr)
	if err != nil {
		return nil, err
	}
	for _, v := range values {
		ec2Vms, err := e.productInfo.GetVmsWithAttrValue(region, ec2Attr, v)
		if err != nil {
			return nil, err
		}
		for _, ec2vm := range ec2Vms {
			vm := VirtualMachine{
				Type:          ec2vm.Type,
				OnDemandPrice: ec2vm.OnDemandPrice,
				AvgPrice:      99,
				Cpus:          ec2vm.Cpus,
				Mem:           ec2vm.Mem,
				Gpus:          ec2vm.Gpus,
				Burst:         ec2vm.IsBurst(),
			}
			vms = append(vms, vm)
		}
	}

	instanceTypes := make([]string, len(vms))
	for i, vm := range vms {
		instanceTypes[i] = vm.Type
	}
	if zones == nil || len(zones) == 0 {
		zones = []string{}
	}

	if len(zones) == 0 {
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
	}

	var avgSpotPrices map[string]float64
	pricesParsed := false
	if e.prometheus != nil {
		zoneAvgSpotPrices, err := e.getSpotPriceAvgsFromPrometheus(region, zones, instanceTypes)
		if err != nil {
			log.WithError(err).Warn("Couldn't get spot price info from Prometheus API, fallback to direct AWS API access.")
		} else {
			pricesParsed = true
			avgSpotPrices = zoneAvgSpotPrices
		}
	}

	if e.prometheus == nil || !pricesParsed {
		log.Debug("getting current spot prices directly from the AWS API")
		currentZoneAvgSpotPrices, err := e.getCurrentSpotPrices(region, zones, instanceTypes)
		if err != nil {
			return nil, err
		}
		avgSpotPrices = currentZoneAvgSpotPrices
	}

	for i := range vms {
		if currentPrice, ok := avgSpotPrices[vms[i].Type]; ok {
			vms[i].AvgPrice = currentPrice
		}
	}

	log.Debugf("found vms with %s values %v: %v", attr, values, vms)
	return vms, nil
}

func (e *Ec2VmRegistry) getSpotPriceAvgsFromPrometheus(region string, zones []string, instanceTypes []string) (map[string]float64, error) {
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

func (e *Ec2VmRegistry) getAvailableAttributeValues(attr string) ([]float64, error) {
	ec2Attr, err := e.toEC2Attribute(attr)
	if err != nil {
		return nil, err
	}
	values, err := e.productInfo.GetAttrValues(ec2Attr)
	if err != nil {
		return nil, err
	}
	log.Debugf("%s attribute values: %v", attr, values)
	return values, nil
}

func (e *Ec2VmRegistry) toEC2Attribute(attr string) (string, error) {
	switch attr {
	case Cpu:
		return productinfo.Cpu, nil
	case Memory:
		return productinfo.Memory, nil
	}
	return "", fmt.Errorf("unsupported attribute: %s", attr)
}

func (e *Ec2VmRegistry) getCurrentSpotPrices(region string, zones []string, instanceTypes []string) (map[string]float64, error) {
	log.Debug("getting current spot prices from AWS API")
	ec2Svc := ec2.New(e.session, &aws.Config{Region: aws.String(region)})

	history, err := ec2Svc.DescribeSpotPriceHistory(&ec2.DescribeSpotPriceHistoryInput{
		StartTime:           aws.Time(time.Now()),
		ProductDescriptions: []*string{aws.String("Linux/UNIX")},
		InstanceTypes:       aws.StringSlice(instanceTypes),
	})
	if err != nil {
		return nil, err
	}

	type SpotPrice struct {
		AZ    string
		Price float64
	}

	type SpotPrices []SpotPrice

	zoneAvgSpotPrices := make(map[string]float64)
	spotPrices := make(map[string]SpotPrices)

	for _, priceEntry := range history.SpotPriceHistory {
		spotPrice, err := strconv.ParseFloat(*priceEntry.SpotPrice, 32)
		if err != nil {
			return nil, err
		}
		for _, value := range zones {
			if value == *priceEntry.AvailabilityZone {
				spotPrices[*priceEntry.InstanceType] = append(spotPrices[*priceEntry.InstanceType], SpotPrice{*priceEntry.AvailabilityZone, spotPrice})
				continue
			}
		}
	}

	for vmType, prices := range spotPrices {
		if len(prices) != len(zones) {
			// some instance types are not available in all zones
			continue
		}
		var sumPrice float64
		for _, p := range prices {
			sumPrice += p.Price
		}
		zoneAvgSpotPrices[vmType] = sumPrice / float64(len(zones))
	}

	return zoneAvgSpotPrices, nil
}
