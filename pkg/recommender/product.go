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
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo-client/client"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo-client/client/products"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo-client/client/provider"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo-client/client/region"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo-client/client/regions"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo-client/client/service"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo-client/models"
	"github.com/go-openapi/runtime"
	"github.com/goph/emperror"
)

// CloudInfoSource declares operations for retrieving information required for the recommender engine
type CloudInfoSource interface {
	// GetProductDetails retrieves the product details for the provider and region
	GetProductDetails(provider string, service string, region string) ([]VirtualMachine, error)

	// GetRegions retrieves the regions
	GetRegions(provider, service string) ([]*models.Continent, error)
}

// CloudInfoClient application struct to retrieve data for the recommender; wraps the generated product info client
// It implements the CloudInfoSource interface, delegates to the embedded generated client
type CloudInfoClient struct {
	*client.Cloudinfo
}

const (
	cloudInfoErrTag    = "cloud-info"
	cloudInfoCliErrTag = "cloud-info-client"
)

// NewCloudInfoClient creates a new product info client wrapper instance
func NewCloudInfoClient(pic *client.Cloudinfo) *CloudInfoClient {
	return &CloudInfoClient{Cloudinfo: pic}
}

// GetProductDetails gets the available product details from the provider in the region
func (ciCli *CloudInfoClient) GetProductDetails(provider string, service string, region string) ([]VirtualMachine, error) {
	gpdp := products.NewGetProductsParams().WithRegion(region).WithProvider(provider).WithService(service)

	allProducts, err := ciCli.Products.GetProducts(gpdp)
	if err != nil {
		return nil, discriminateErrCtx(err)
	}

	var vms []VirtualMachine

	for _, p := range allProducts.Payload.Products {
		vms = append(vms, VirtualMachine{
			Category:       p.Category,
			Type:           p.Type,
			OnDemandPrice:  p.OnDemandPrice,
			AvgPrice:       avg(p.SpotPrice),
			Cpus:           p.Cpus,
			Mem:            p.Mem,
			Gpus:           p.Gpus,
			Burst:          p.Burst,
			NetworkPerf:    p.NtwPerf,
			NetworkPerfCat: p.NtwPerfCat,
			CurrentGen:     p.CurrentGen,
			Zones:          p.Zones,
		})
	}

	return vms, nil
}

func avg(prices []*models.ZonePrice) float64 {
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
	gpp := provider.NewGetProviderParams().WithProvider(prv)

	provider, err := ciCli.Provider.GetProvider(gpp)
	if err != nil {
		return "", discriminateErrCtx(err)
	}

	return provider.Payload.Provider.Provider, nil
}

// GetService validates service
func (ciCli *CloudInfoClient) GetService(prv string, svc string) (string, error) {
	gsp := service.NewGetServiceParams().WithProvider(prv).WithService(svc)

	service, err := ciCli.Service.GetService(gsp)
	if err != nil {
		return "", discriminateErrCtx(err)
	}

	return service.Payload.Service.Service, nil
}

// GetRegion validates region
func (ciCli *CloudInfoClient) GetRegion(prv, svc, reg string) (string, error) {
	grp := region.NewGetRegionParams().WithProvider(prv).WithService(svc).WithRegion(reg)

	r, err := ciCli.Region.GetRegion(grp)
	if err != nil {
		return "", discriminateErrCtx(err)
	}

	return r.Payload.Name, nil
}

// GetRegions gets regions
func (ciCli *CloudInfoClient) GetRegions(provider, service string) ([]*models.Continent, error) {
	grp := regions.NewGetRegionsParams().WithProvider(provider).WithService(service)
	r, err := ciCli.Regions.GetRegions(grp)
	if err != nil {
		return nil, discriminateErrCtx(err)
	}
	return r.Payload, nil
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
