package api

// GetRecommendationParams is a placeholder for the recommendation route's path parameters
// swagger:parameters recommendClusterSetup
type GetRecommendationParams struct {
	// in:path
	Provider string `json:"provider"`
	// in:path
	Region string `json:"region"`
}
