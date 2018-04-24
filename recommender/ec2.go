package recommender

import (
	"errors"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	pi "github.com/banzaicloud/cluster-recommender/ec2_productinfo"
	log "github.com/sirupsen/logrus"
)

type Ec2VmRegistry struct {
	session     *session.Session
	productInfo *pi.ProductInfo
}

func NewEc2VmRegistry(pi *pi.ProductInfo) (VmRegistry, error) {
	s, err := session.NewSession()
	if err != nil {
		log.WithError(err).Error("Error creating AWS session")
		return nil, err
	}
	return &Ec2VmRegistry{
		session:     s,
		productInfo: pi,
	}, nil
}

func (e *Ec2VmRegistry) findVmsWithCpuUnits(region string, zones []string, cpuUnits []float64) ([]VirtualMachine, error) {
	log.Infof("Getting instance types and on demand prices with %v vcpus", cpuUnits)
	var vms []VirtualMachine
	for _, cpu := range cpuUnits {
		ec2Vms, err := e.productInfo.GetVmsWithCpu(region, pi.Cpu, cpu)
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

	currentAvgSpotPrices, err := e.getCurrentSpotPrices(region, zones, instanceTypes)
	if err != nil {
		return nil, err
	}

	for i := range vms {
		if currentPrice, ok := currentAvgSpotPrices[vms[i].Type]; ok {
			vms[i].AvgPrice = currentPrice
		}
	}

	log.Debugf("found vms with cpu units %v: %v", cpuUnits, vms)
	return vms, nil
}

func (e *Ec2VmRegistry) findCpuUnits(min float64, max float64) ([]float64, error) {
	log.Debugf("finding cpu units between: [%v, %v]", min, max)
	cpuValues, err := e.productInfo.GetSortedAttrValues(pi.Cpu)
	if err != nil {
		return nil, err
	}
	log.Debugf("cpu attribute values sorted: %v", cpuValues)

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

func (e *Ec2VmRegistry) getCurrentSpotPrices(region string, zones []string, instanceTypes []string) (map[string]float64, error) {
	log.Debug("getting current spot price of instance types")
	ec2Svc := ec2.New(e.session, &aws.Config{Region: aws.String(region)})

	var availabilityZones []string

	if len(zones) == 0 {
		azs, err := ec2Svc.DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{})
		if err != nil {
			return nil, err
		}
		for _, az := range azs.AvailabilityZones {
			if *az.State == "available" {
				availabilityZones = append(availabilityZones, *az.ZoneName)
			}
		}
		// TODO: these should be returned
	} else {
		availabilityZones = zones
	}

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
		for _, value := range availabilityZones {
			if value == *priceEntry.AvailabilityZone {
				spotPrices[*priceEntry.InstanceType] = append(spotPrices[*priceEntry.InstanceType], SpotPrice{*priceEntry.AvailabilityZone, spotPrice})
				continue
			}
		}
	}

	for vmType, prices := range spotPrices {
		if len(prices) != len(availabilityZones) {
			// some instance types are not available in all zones
			continue
		}
		var sumPrice float64
		for _, p := range prices {
			sumPrice += p.Price
		}
		zoneAvgSpotPrices[vmType] = sumPrice / float64(len(availabilityZones))
	}

	return zoneAvgSpotPrices, nil
}
