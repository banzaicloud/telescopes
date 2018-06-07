package azure

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/preview/commerce/mgmt/2015-06-01-preview/commerce"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2016-06-01/subscriptions"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/banzaicloud/telescopes/productinfo"
	log "github.com/sirupsen/logrus"
)

const (
	cpu    = "cpu"
	memory = "memory"
)

var regionCodeMappings = map[string]string{
	"ap": "asia",
	"au": "australia",
	"br": "brazil",
	"ca": "canada",
	"eu": "europe",
	"fr": "france",
	"in": "india",
	"ja": "japan",
	"kr": "korea",
	"uk": "uk",
	"us": "us",
}

// AzureInfoer encapsulates the data and operations needed to access external Azure resources
type AzureInfoer struct {
	subscriptionId      string
	subscriptionsClient subscriptions.Client
	vmSizesClient       compute.VirtualMachineSizesClient
	rateCardClient      commerce.RateCardClient
}

// NewAzureInfoer creates a new instance of the Azure infoer
func NewAzureInfoer(subscriptionId string) (*AzureInfoer, error) {
	authorizer, err := auth.NewAuthorizerFromFile(azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	sClient := subscriptions.NewClient()
	sClient.Authorizer = authorizer

	vmClient := compute.NewVirtualMachineSizesClient(subscriptionId)
	vmClient.Authorizer = authorizer

	rcClient := commerce.NewRateCardClient(subscriptionId)
	rcClient.Authorizer = authorizer

	return &AzureInfoer{
		subscriptionId:      subscriptionId,
		subscriptionsClient: sClient,
		vmSizesClient:       vmClient,
		rateCardClient:      rcClient,
	}, nil
}

type regionParts []string

func (r regionParts) String() string {
	var result string
	for _, p := range r {
		result += p
	}
	return result
}

func (a *AzureInfoer) toRegionID(meterRegion string, regions map[string]string) (string, error) {
	var rp regionParts = strings.Split(strings.ToLower(meterRegion), " ")
	regionCode := regionCodeMappings[rp[0]]
	lastPart := rp[len(rp)-1]
	var regionIds []string
	if _, err := strconv.Atoi(lastPart); err == nil {
		regionIds = []string{
			fmt.Sprintf("%s%s%s", regionCode, rp[1:len(rp)-1], lastPart),
			fmt.Sprintf("%s%s%s", rp[1:len(rp)-1], regionCode, lastPart),
		}
	} else {
		regionIds = []string{
			fmt.Sprintf("%s%s", regionCode, rp[1:]),
			fmt.Sprintf("%s%s", rp[1:], regionCode),
		}
	}
	for _, regionID := range regionIds {
		if a.checkRegionID(regionID, regions) {
			return regionID, nil
		}
	}
	return "", errors.New("couldn't find region")
}

func (a *AzureInfoer) checkRegionID(regionID string, regions map[string]string) bool {
	for region := range regions {
		if regionID == region {
			return true
		}
	}
	return false
}

// Initialize downloads and parses the Rate Card API's meter list on Azure
func (a *AzureInfoer) Initialize() (map[string]map[string]productinfo.Price, error) {
	log.Debug("initializing Azure price info")
	allPrices := make(map[string]map[string]productinfo.Price)

	regions, err := a.GetRegions()
	if err != nil {
		return nil, err
	}

	log.Debugf("queried regions: %v", regions)

	rateCardFilter := "OfferDurableId eq 'MS-AZR-0003p' and Currency eq 'USD' and Locale eq 'en-US' and RegionInfo eq 'US'"
	result, err := a.rateCardClient.Get(context.TODO(), rateCardFilter)
	if err != nil {
		return nil, err
	}
	for _, v := range *result.Meters {
		if *v.MeterCategory == "Virtual Machines" && len(*v.MeterTags) == 0 && *v.MeterRegion != "" {
			if !strings.Contains(*v.MeterSubCategory, "(Windows)") {
				region, err := a.toRegionID(*v.MeterRegion, regions)
				if err != nil {
					log.Debugf("region not found among Azure reported locations")
				}
				instanceType := strings.Split(*v.MeterSubCategory, " ")[0]
				if instanceType == "" {
					log.Debugf("instance type is empty: %s, region=%s", *v.MeterSubCategory, *v.MeterRegion)
					continue
				}
				var priceInUsd float64

				if len(v.MeterRates) < 1 {
					log.Debugf("%s doesn't have rate info in region %s", *v.MeterSubCategory, *v.MeterRegion)
					continue
				}
				for _, rate := range v.MeterRates {
					priceInUsd += *rate
				}
				if allPrices[region] == nil {
					allPrices[region] = make(map[string]productinfo.Price)
				}
				price := allPrices[region][instanceType]
				if !strings.Contains(*v.MeterSubCategory, "Low Priority") {
					price.OnDemandPrice = priceInUsd
				} else {
					spotPrice := make(productinfo.SpotPriceInfo)
					spotPrice[region] = priceInUsd
					price.SpotPrice = spotPrice
				}
				allPrices[region][instanceType] = price
				log.Debugf("price info added: [region=%s, machinetype=%s, price=%v]", region, instanceType, price)
			}
		}
	}

	log.Debug("finished initializing Azure price info")
	return allPrices, nil
}

// GetAttributeValues gets the AttributeValues for the given attribute name
func (a *AzureInfoer) GetAttributeValues(attribute string) (productinfo.AttrValues, error) {

	log.Debugf("getting %s values", attribute)

	values := make(productinfo.AttrValues, 0)
	valueSet := make(map[productinfo.AttrValue]interface{})

	regions, err := a.GetRegions()
	if err != nil {
		return nil, err
	}

	for region := range regions {
		vmSizes, err := a.vmSizesClient.List(context.TODO(), region)
		if err != nil {
			log.WithError(err).Warnf("[Azure] couldn't get VM sizes in region %s", region)
			continue
		}
		for _, v := range *vmSizes.Value {
			switch attribute {
			case cpu:
				valueSet[productinfo.AttrValue{
					Value:    float64(*v.NumberOfCores),
					StrValue: fmt.Sprintf("%v", *v.NumberOfCores),
				}] = ""
			case memory:
				valueSet[productinfo.AttrValue{
					Value:    float64(*v.MemoryInMB) / 1000,
					StrValue: fmt.Sprintf("%v", *v.MemoryInMB),
				}] = ""
			}
		}
	}

	for attr := range valueSet {
		values = append(values, attr)
	}

	log.Debugf("found %s values: %v", attribute, values)
	return values, nil
}

// GetProducts retrieves the available virtual machines based on the arguments provided
func (a *AzureInfoer) GetProducts(regionId string, attrKey string, attrValue productinfo.AttrValue) ([]productinfo.VmInfo, error) {
	log.Debugf("getting product info [region=%s, %s=%v]", regionId, attrKey, attrValue.Value)
	var vms []productinfo.VmInfo

	vmSizes, err := a.vmSizesClient.List(context.TODO(), regionId)
	if err != nil {
		return nil, err
	}
	for _, v := range *vmSizes.Value {
		switch attrKey {
		case cpu:
			if *v.NumberOfCores != int32(attrValue.Value) {
				continue
			}
		case memory:
			if *v.MemoryInMB != int32(attrValue.Value*1000) {
				continue
			}
		}
		vms = append(vms, productinfo.VmInfo{
			Type: *v.Name,
			Cpus: float64(*v.NumberOfCores),
			Mem:  float64(*v.MemoryInMB) / 1000,
			// TODO: netw perf
		})
	}

	log.Debugf("found vms [%s=%v]: %#v", attrKey, attrValue.Value, vms)
	return vms, nil
}

// GetZones returns the availability zones in a region
func (a *AzureInfoer) GetZones(region string) ([]string, error) {
	return []string{region}, nil
}

// GetRegions returns a map with available regions transforms the api representation into a "plain" map
func (a *AzureInfoer) GetRegions() (map[string]string, error) {
	regions := make(map[string]string)
	locations, err := a.subscriptionsClient.ListLocations(context.TODO(), a.subscriptionId)
	if err != nil {
		return nil, err
	}
	for _, loc := range *locations.Value {
		regions[*loc.Name] = *loc.DisplayName
	}
	return regions, nil
}

// HasShortLivedPriceInfo - Azure doesn't have frequently changing prices
func (a *AzureInfoer) HasShortLivedPriceInfo() bool {
	return false
}

// TODO: We have some VM types, where we don't have pricing info, e.g.: M64-16ms
// it's stored as a VM, and when we want to have pricing info for them, it cannot be found in cache
// -> calls initialize for all vmInfos that's not found -> takes an eternity

// GetCurrentPrices retrieves all the price info in a region
func (a *AzureInfoer) GetCurrentPrices(region string) (map[string]productinfo.Price, error) {
	log.Debugf("getting current prices in region %s", region)
	allPrices, err := a.Initialize()
	if err != nil {
		return nil, err
	}
	log.Debugf("found prices in region %s", region)
	return allPrices[region], nil
}

// GetMemoryAttrName returns the provider representation of the memory attribute
func (a *AzureInfoer) GetMemoryAttrName() string {
	return memory
}

// GetCpuAttrName returns the provider representation of the cpu attribute
func (a *AzureInfoer) GetCpuAttrName() string {
	return cpu
}

// GetNetworkPerformanceMapper returns the network performance mappier implementation for this provider
func (a *AzureInfoer) GetNetworkPerformanceMapper() (productinfo.NetworkPerfMapper, error) {
	return newAzureNetworkMapper(), nil
}
