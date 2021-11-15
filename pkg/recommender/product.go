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
	"github.com/goph/logur"
)

// CloudInfoSource declares operations for retrieving information required for the recommender engine
type CloudInfoSource interface {
	// GetProductDetails retrieves the product details for the provider and region
	GetProductDetails(provider string, service string, region string) ([]VirtualMachine, error)

	// GetRegions retrieves the regions
	GetRegions(provider, service string) ([]cloudinfo.Region, error)

	//GetContinentsData retrieves continents data
	GetContinentsData(provider, service string) ([]cloudinfo.Continent, error)

	//GetZones retrieves zones
	GetZones(provider, service, region string) ([]string, error)

	//GetContinents retrieves supported continents
	GetContinents() ([]string, error)

	//GetRegion retrieves the region for the provided arguments, returns error if not found
	GetRegion(provider string, service string, region string) (string, error)

	//GetProvider retrieves the given provider,returns error if not found
	GetProvider(provider string) (string, error)

	//GetService  retrieves the given service, returns error if not found
	GetService(provider string, service string) (string, error)
}

// cloudInfoClient component struct to retrieve data for the recommender; wraps the generated product info client
// It implements the CloudInfoSource interface, delegates to the embedded generated client
type cloudInfoClient struct {
	logger logur.Logger
	*cloudinfo.APIClient
}

const (
	cloudInfoService         = "cloud-info"
	cloudInfoClientComponent = "cloud-info-client"
)

// NewCloudInfoClient creates a new product info client wrapper instance
func NewCloudInfoClient(ciUrl string, logger logur.Logger) CloudInfoSource {
	apiCli := cloudinfo.NewAPIClient(&cloudinfo.Configuration{
		BasePath:      ciUrl,
		DefaultHeader: make(map[string]string),
		UserAgent:     "Telescopes/go",
	})
	return &cloudInfoClient{
		APIClient: apiCli,
		logger:    logur.WithFields(logger, map[string]interface{}{"cli": cloudInfoClientComponent}),
	}
}

// GetProductDetails gets the available product details from the provider in the region
func (ciCli *cloudInfoClient) GetProductDetails(provider string, service string, region string) ([]VirtualMachine, error) {
	tags := map[string]interface{}{"provider": provider, "service": service, "region": region}
	ciCli.logger.Info("retrieving product details", tags)

	allProducts, _, err := ciCli.ProductsApi.GetProducts(context.Background(), provider, service, region)
	if err != nil {

		ciCli.logger.Error("failed to retrieve product details", tags)
		return nil, discriminateErrCtx(err)
	}

	vms := make([]VirtualMachine, 0)

	for _, p := range allProducts.Products {
		vms = append(vms, VirtualMachine{
			Category:       p.Category,
			Series: 		p.Series,
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

	ciCli.logger.Info("retrieved product details", tags)
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
func (ciCli *cloudInfoClient) GetProvider(prv string) (string, error) {
	tags := map[string]interface{}{"provider": prv}
	ciCli.logger.Info("retrieving provider", tags)

	provider, _, err := ciCli.ProviderApi.GetProvider(context.Background(), prv)
	if err != nil {

		ciCli.logger.Error("failed to retrieve provider", tags)
		return "", discriminateErrCtx(err)
	}

	ciCli.logger.Info("retrieved provider", tags)
	return provider.Provider.Provider, nil
}

// GetService validates service
func (ciCli *cloudInfoClient) GetService(prv string, svc string) (string, error) {
	tags := map[string]interface{}{"provider": prv, "service": svc}
	ciCli.logger.Info("retrieving service", tags)

	service, _, err := ciCli.ServiceApi.GetService(context.Background(), prv, svc)
	if err != nil {

		ciCli.logger.Error("failed to retrieve service", tags)
		return "", discriminateErrCtx(err)
	}

	ciCli.logger.Info("retrieved service", tags)
	return service.Service.Service, nil
}

// GetRegion validates region
func (ciCli *cloudInfoClient) GetRegion(prv, svc, reg string) (string, error) {
	tags := map[string]interface{}{"provider": prv, "service": svc, "region": reg}
	ciCli.logger.Info("retrieving region", tags)

	r, _, err := ciCli.RegionApi.GetRegion(context.Background(), prv, svc, reg)
	if err != nil {

		ciCli.logger.Error("failed to retrieve region", tags)
		return "", discriminateErrCtx(err)
	}

	ciCli.logger.Info("retrieved region", tags)
	return r.Name, nil
}

// GetZones get zones
func (ciCli *cloudInfoClient) GetZones(provider, service, region string) ([]string, error) {
	tags := map[string]interface{}{"provider": provider, "service": service, "region": region}
	ciCli.logger.Info("retrieving zones", tags)

	r, _, err := ciCli.RegionApi.GetRegion(context.Background(), provider, service, region)
	if err != nil {

		ciCli.logger.Error("failed to retrieve zones", tags)
		return nil, discriminateErrCtx(err)
	}

	ciCli.logger.Info("retrieved zones", tags)
	return r.Zones, nil
}

// GetRegions gets regions
func (ciCli *cloudInfoClient) GetRegions(provider, service string) ([]cloudinfo.Region, error) {

	tags := map[string]interface{}{"provider": provider, "service": service}
	ciCli.logger.Info("retrieving regions", tags)

	r, _, err := ciCli.RegionsApi.GetRegions(context.Background(), provider, service)
	if err != nil {

		ciCli.logger.Error("failed to retrieve regions", tags)
		return nil, discriminateErrCtx(err)
	}

	ciCli.logger.Info("retrieved regions", tags)
	return r, nil
}

func (ciCli *cloudInfoClient) GetContinentsData(provider, service string) ([]cloudinfo.Continent, error) {
	tags := map[string]interface{}{"provider": provider, "service": service}
	ciCli.logger.Info("retrieving continent data", tags)

	r, _, err := ciCli.ContinentsApi.GetContinentsData(context.Background(), provider, service)
	if err != nil {

		ciCli.logger.Error("failed to retrieve continent data", tags)
		return nil, discriminateErrCtx(err)
	}

	ciCli.logger.Info("retrieved continent data", tags)
	return r, nil
}

// GetContinents gets continents
func (ciCli *cloudInfoClient) GetContinents() ([]string, error) {
	ciCli.logger.Info("retrieving continents")
	c, _, err := ciCli.ContinentsApi.GetContinents(context.Background())

	if err != nil {

		ciCli.logger.Error("failed to retrieve continents")
		return nil, discriminateErrCtx(err)
	}
	ciCli.logger.Info("retrieved continents")
	return c, nil
}

// discriminateErrCtx adds tags to the error context in order to classify them later
func discriminateErrCtx(err error) error {

	if _, ok := err.(*runtime.APIError); ok {
		// the service can be reached
		return emperror.With(err, cloudInfoService)
	}
	// handle other cloud info errors here

	// probably connectivity error (should it be analized further?!)
	return emperror.With(err, cloudInfoClientComponent)
}
