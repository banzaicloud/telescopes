package api

import "github.com/banzaicloud/telescopes/pkg/productinfo"

// GetProductDetailsParams is a placeholder for the get products path parameters
// swagger:parameters getProductDetails
type GetProductDetailsParams struct {
	// in:path
	Provider string `json:"provider"`
	// in:path
	Region string `json:"region"`
}

// RegionResp holds the list of available regions of a cloud provider
// swagger:response regionsResponse
type RegionResp struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

// ProductDetailsResponse Api object to be mapped to product info response
// swagger:model ProductDetailsResponse
type ProductDetailsResponse struct {
	// Products represents a slice of products for a given provider (VMs with attributes and process)
	Products []productinfo.ProductDetails `json:"products"`
}
