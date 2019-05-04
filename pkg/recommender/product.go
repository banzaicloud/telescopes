// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package recommender

import (
	"context"

	"github.com/banzaicloud/telescopes/.gen/cloudinfo"
	"github.com/go-openapi/runtime"
	"github.com/goph/emperror"
)

// CloudInfoSource declares operations for retrieving information required for the recommender engine
type CloudInfoSource interface {
	// GetProductDetails retrieves the product details for the provider and region
	GetProductDetails(provider string, service string, region string) ([]VirtualMachine, error)

	// GetRegions retrieves the regions
	GetRegions(provider, service string) ([]cloudinfo.Region, error)

	GetContinentsData(provider, service string) ([]cloudinfo.Continent, error)

	GetZones(prv, svc, reg string) ([]string, error)
}

// CloudInfoClient application struct to retrieve data for the recommender; wraps the generated product info client
// It implements the CloudInfoSource interface, delegates to the embedded generated client
type CloudInfoClient struct {
	*cloudinfo.APIClient
}

const (
	cloudInfoErrTag    = "cloud-info"
	cloudInfoCliErrTag = "cloud-info-client"
)

// NewCloudInfoClient creates a new product info client wrapper instance
func NewCloudInfoClient(ciUrl string) *CloudInfoClient {
	apiCli := cloudinfo.NewAPIClient(&cloudinfo.Configuration{
		BasePath:      ciUrl,
		DefaultHeader: make(map[string]string),
		UserAgent:     "Telescopes/go",
	})
	return &CloudInfoClient{APIClient: apiCli}
}

// GetProductDetails gets the available product details from the provider in the region
func (ciCli *CloudInfoClient) GetProductDetails(provider string, service string, region string) ([]VirtualMachine, error) {

	allProducts, _, err := ciCli.ProductsApi.GetProducts(context.Background(), provider, service, region)
	if err != nil {
		return nil, discriminateErrCtx(err)
	}

	vms := make([]VirtualMachine, 0)

	for _, p := range allProducts.Products {
		vms = append(vms, VirtualMachine{
			Category:       p.Category,
			Type:           p.Type,
			OnDemandPrice:  p.OnDemandPrice,
			AvgPrice:       avg(p.SpotPrice),
			Cpus:           p.CpusPerVm,
			Mem:            p.MemPerVm,
			Gpus:           p.GpusPerVm,
			Burst:          p.Burst,
			NetworkPerf:    p.NtwPerf,
			NetworkPerfCat: p.NtwPerfCategory,
			CurrentGen:     p.CurrentGen,
			Zones:          p.Zones,
		})
	}

	return vms, nil
}

func avg(prices []cloudinfo.ZonePrice) float64 {
	if len(prices) == 0 {
		return 0.0
	}
	avgPrice := 0.0
	for _, price := range prices {
		avgPrice += price.Price
	}
	return avgPrice / float64(len(prices))
}

// GetProvider validates provider
func (ciCli *CloudInfoClient) GetProvider(prv string) (string, error) {

	provider, _, err := ciCli.ProviderApi.GetProvider(context.Background(), prv)
	if err != nil {
		return "", discriminateErrCtx(err)
	}

	return provider.Provider.Provider, nil
}

// GetService validates service
func (ciCli *CloudInfoClient) GetService(prv string, svc string) (string, error) {

	service, _, err := ciCli.ServiceApi.GetService(context.Background(), prv, svc)
	if err != nil {
		return "", discriminateErrCtx(err)
	}

	return service.Service.Service, nil
}

// GetRegion validates region
func (ciCli *CloudInfoClient) GetRegion(prv, svc, reg string) (string, error) {

	r, _, err := ciCli.RegionApi.GetRegion(context.Background(), prv, svc, reg)
	if err != nil {
		return "", discriminateErrCtx(err)
	}

	return r.Name, nil
}

// GetZones get zones
func (ciCli *CloudInfoClient) GetZones(provider, service, region string) ([]string, error) {

	r, _, err := ciCli.RegionApi.GetRegion(context.Background(), provider, service, region)
	if err != nil {
		return nil, discriminateErrCtx(err)
	}

	return r.Zones, nil
}

// GetRegions gets regions
func (ciCli *CloudInfoClient) GetRegions(provider, service string) ([]cloudinfo.Region, error) {
	r, _, err := ciCli.RegionsApi.GetRegions(context.Background(), provider, service)
	if err != nil {
		return nil, discriminateErrCtx(err)
	}
	return r, nil
}

func (ciCli *CloudInfoClient) GetContinentsData(provider, service string) ([]cloudinfo.Continent, error) {
	r, _, err := ciCli.ContinentsApi.GetContinentsData(context.Background(), provider, service)
	if err != nil {
		return nil, discriminateErrCtx(err)
	}
	return r, nil

}

// GetContinents gets continents
func (ciCli *CloudInfoClient) GetContinents() ([]string, error) {
	c, _, err := ciCli.ContinentsApi.GetContinents(context.Background())
	if err != nil {
		return nil, discriminateErrCtx(err)
	}

	return c, nil
}

func discriminateErrCtx(err error) error {

	if _, ok := err.(*runtime.APIError); ok {
		// the service can be reached
		return emperror.With(err, cloudInfoErrTag)
	}
	// handle other cloud info errors here

	// probably connectivity error (should it be analized further?!)
	return emperror.With(err, cloudInfoCliErrTag)
}
