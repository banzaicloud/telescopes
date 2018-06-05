package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2017-12-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/preview/commerce/mgmt/2015-06-01-preview/commerce"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2016-06-01/subscriptions"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/banzaicloud/telescopes/productinfo"
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
	panic("implement me")
}

func (a *AzureInfoer) GetAttributeValues(attribute string) (productinfo.AttrValues, error) {
	panic("implement me")
}

func (a *AzureInfoer) GetProducts(regionId string, attrKey string, attrValue productinfo.AttrValue) ([]productinfo.VmInfo, error) {
	panic("implement me")
}

func (a *AzureInfoer) GetZones(region string) ([]string, error) {
	panic("implement me")
}

func (a *AzureInfoer) GetRegions() (map[string]string, error) {
	panic("implement me")
}

func (a *AzureInfoer) HasShortLivedPriceInfo() bool {
	panic("implement me")
}

func (a *AzureInfoer) GetCurrentPrices(region string) (map[string]productinfo.Price, error) {
	panic("implement me")
}

func (a *AzureInfoer) GetMemoryAttrName() string {
	panic("implement me")
}

func (a *AzureInfoer) GetCpuAttrName() string {
	panic("implement me")
}

func (a *AzureInfoer) GetNetworkPerformanceMapper() (productinfo.NetworkPerfMapper, error) {
	panic("implement me")
}

func (a *AzureInfoer) teszting() bool {
	locations, err := a.subscriptionsClient.ListLocations(context.TODO(), a.subscriptionId)

	for _, loc := range *locations.Value {
		fmt.Println(*loc.Name, *loc.DisplayName, *loc.ID)
	}

	fmt.Println(time.Now().UTC())

	result1, err := a.vmSizesClient.List(context.TODO(), "eastus")
	if err != nil {
		fmt.Println(err)
	}
	for _, v := range *result1.Value {
		fmt.Println(*v.Name, *v.NumberOfCores, *v.MemoryInMB)
	}
	fmt.Println(time.Now().UTC())

	result, err := a.rateCardClient.Get(context.TODO(), "OfferDurableId eq 'MS-AZR-0003p' and Currency eq 'USD' and Locale eq 'en-US' and RegionInfo eq 'US'")
	if err != nil {
		fmt.Println(err)
	}
	for _, v := range *result.Meters {
		fmt.Println(*v.MeterName, *v.MeterCategory, *v.MeterSubCategory, *v.MeterRegion, *v.MeterTags, *v.IncludedQuantity, *v.Unit)
		for key, value := range v.MeterRates {
			fmt.Println(key, *value)
		}
	}
	return true
}
