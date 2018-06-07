package azure

import (
	"context"
	"fmt"
	"strings"
	"time"

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
		if *v.MeterCategory == "Virtual Machines" && len(*v.MeterTags) == 0 && *v.MeterRegion != "" { //&& *v.MeterRegion == "US East"
			if !strings.Contains(*v.MeterSubCategory, "(Windows)") {

				region := *v.MeterRegion
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

func (a *AzureInfoer) GetProducts(regionId string, attrKey string, attrValue productinfo.AttrValue) ([]productinfo.VmInfo, error) {
	// TODO
	fmt.Println(time.Now().UTC())

	result1, err := a.vmSizesClient.List(context.TODO(), "eastus")
	if err != nil {
		fmt.Println(err)
	}
	for _, v := range *result1.Value {
		if strings.Contains(*v.Name, "Standard_M64") {
			fmt.Println(*v.Name, *v.NumberOfCores, *v.MemoryInMB)
		}
	}
	fmt.Println(time.Now().UTC())
	// TODO
	return nil, nil
}

func (a *AzureInfoer) GetZones(region string) ([]string, error) {
	// TODO: check if it works an empty slice
	return []string{}, nil
}

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

func (a *AzureInfoer) HasShortLivedPriceInfo() bool {
	return false
}

func (a *AzureInfoer) GetCurrentPrices(region string) (map[string]productinfo.Price, error) {
	panic("implement me")
}

func (a *AzureInfoer) GetMemoryAttrName() string {
	return memory
}

func (a *AzureInfoer) GetCpuAttrName() string {
	return cpu
}

func (a *AzureInfoer) GetNetworkPerformanceMapper() (productinfo.NetworkPerfMapper, error) {
	return newAzureNetworkMapper(), nil
}
