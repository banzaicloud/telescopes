package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAzureInfoer_toRegionID(t *testing.T) {

	regionMap := map[string]string{
		"japanwest":          "Japan West",
		"centralindia":       "Central India",
		"francesouth":        "France South",
		"northcentralus":     "North Central US",
		"japaneast":          "Japan East",
		"australiaeast":      "Australia East",
		"southindia":         "South India",
		"canadaeast":         "Canada East",
		"westus2":            "West US 2",
		"westus":             "West US",
		"northeurope":        "North Europe",
		"westeurope":         "West Europe",
		"uksouth":            "UK South",
		"centralus":          "Central US",
		"australiasoutheast": "Australia Southeast",
		"ukwest":             "UK West",
		"koreacentral":       "Korea Central",
		"koreanorthcentral":  "Korea North Central",
		"koreanorthcentral2": "Korea North Central 2",
		"francecentral":      "France Central",
		"eastasia":           "East Asia",
		"canadacentral":      "Canada Central",
		"eastus":             "East US",
		"eastus2":            "East US 2",
		"southcentralus":     "South Central US",
		"southcentralus2":    "South Central US 2",
		"australiacentral":   "Australia Central",
		"westindia":          "West India",
		"koreasouth":         "Korea South",
		"australiacentral2":  "Australia Central 2",
		"southeastasia":      "Southeast Asia",
		"brazilsouth":        "Brazil South",
		"westcentralus":      "West Central US",
	}

	tests := []struct {
		name         string
		sourceRegion string
		check        func(regionId string, err error)
	}{
		{
			name:         "successful check without postfix, len = 2",
			sourceRegion: "JA West",
			check: func(regionId string, err error) {
				assert.Equal(t, "japanwest", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check without postfix, len = 2, inverted",
			sourceRegion: "EU North",
			check: func(regionId string, err error) {
				assert.Equal(t, "northeurope", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check without postfix, len = 3",
			sourceRegion: "KR North Central",
			check: func(regionId string, err error) {
				assert.Equal(t, "koreanorthcentral", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check without postfix, len = 3, inverted",
			sourceRegion: "US North Central",
			check: func(regionId string, err error) {
				assert.Equal(t, "northcentralus", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check with postfix, len = 2",
			sourceRegion: "AU Central 2",
			check: func(regionId string, err error) {
				assert.Equal(t, "australiacentral2", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check with postfix, len = 2, inverted",
			sourceRegion: "US West 2",
			check: func(regionId string, err error) {
				assert.Equal(t, "westus2", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check with postfix, len = 3",
			sourceRegion: "KR North Central 2",
			check: func(regionId string, err error) {
				assert.Equal(t, "koreanorthcentral2", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check with postfix, len = 3, inverted",
			sourceRegion: "US South Central 2",
			check: func(regionId string, err error) {
				assert.Equal(t, "southcentralus2", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check india",
			sourceRegion: "IN Central",
			check: func(regionId string, err error) {
				assert.Equal(t, "centralindia", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check brazil",
			sourceRegion: "BR South",
			check: func(regionId string, err error) {
				assert.Equal(t, "brazilsouth", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check france",
			sourceRegion: "FR South",
			check: func(regionId string, err error) {
				assert.Equal(t, "francesouth", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check canada",
			sourceRegion: "CA Central",
			check: func(regionId string, err error) {
				assert.Equal(t, "canadacentral", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check asia",
			sourceRegion: "AP East",
			check: func(regionId string, err error) {
				assert.Equal(t, "eastasia", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check uk",
			sourceRegion: "UK West",
			check: func(regionId string, err error) {
				assert.Equal(t, "ukwest", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "check not supported region",
			sourceRegion: "US Gov TX",
			check: func(regionId string, err error) {
				assert.Empty(t, regionId, "empty region ID should be returned")
				assert.Equal(t, "couldn't find region", err.Error(), "error should be ")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			azureInfoer := AzureInfoer{}
			test.check(azureInfoer.toRegionID(test.sourceRegion, regionMap))
		})
	}
}
