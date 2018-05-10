package cloudprovider

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/pricing"
)

// cloudInfoProvider gathers operations for retrieving cloud provider information for recommendations
type CloudProductInfoProvider interface {
	//DescribeAvailabilityZones()
	//
	//DescribeSpotPriceHistory()

	// todo transform the structs here, change the response type
	GetAttributeValues(attribute string) (*pricing.GetAttributeValuesOutput, error)

	// todo transform the structs here, change the response type
	GetProducts(regionId string, attrKey string, attrValue string) (*pricing.GetProductsOutput, error)

	GetRegion(id string) *endpoints.Region

	GetRegions() map[string]endpoints.Region
}

// AwsClientWrapper encapsulates the data and operations needed to access external resources
type AwsClientWrapper struct {
	session *session.Session
	// embedded interface to ensure operations are implemented (todo research if this can be avoided)
	CloudProductInfoProvider
}

// NewAwsClientWrapper encapsulates the creation of a wrapper instance
func NewAwsClientWrapper() (*AwsClientWrapper, error) {
	newSession, err := session.NewSession(&aws.Config{})

	if err != nil {
		return &AwsClientWrapper{}, fmt.Errorf("could not create session: %s ", err.Error())
	}

	return &AwsClientWrapper{
		session: newSession,
	}, nil
}

func (wr *AwsClientWrapper) GetAttributeValues(attribute string) (*pricing.GetAttributeValuesOutput, error) {
	return wr.pricingService().GetAttributeValues(wr.newAttributeValuesInput(attribute))
}

func (wr *AwsClientWrapper) GetProducts(regionId string, attrKey string, attrValue string) (*pricing.GetProductsOutput, error) {
	return wr.pricingService().GetProducts(wr.newGetProductsInput(regionId, attrKey, attrValue))
}

func (wr *AwsClientWrapper) GetRegion(id string) *endpoints.Region {
	aws := endpoints.AwsPartition()
	for _, r := range aws.Regions() {
		if r.ID() == id {
			return &r
		}
	}
	return nil
}

func (wr *AwsClientWrapper) pricingService() *pricing.Pricing {
	return pricing.New(wr.session, &aws.Config{Region: aws.String("us-east-1")})
}

// newAttributeValuesInput assembles a GetAttributeValuesInput instance for querying the provider
func (wr *AwsClientWrapper) newAttributeValuesInput(attr string) *pricing.GetAttributeValuesInput {
	return &pricing.GetAttributeValuesInput{
		ServiceCode:   aws.String("AmazonEC2"),
		AttributeName: aws.String(attr),
	}
}

// newAttributeValuesInput assembles a GetAttributeValuesInput instance for querying the provider
func (wr *AwsClientWrapper) newGetProductsInput(regionId string, attrKey string, attrValue string) *pricing.GetProductsInput {
	return &pricing.GetProductsInput{

		ServiceCode: aws.String("AmazonEC2"),
		Filters: []*pricing.Filter{
			{
				Type:  aws.String("TERM_MATCH"),
				Field: aws.String("operatingSystem"),
				Value: aws.String("Linux"),
			},
			{
				Type:  aws.String("TERM_MATCH"),
				Field: aws.String("location"),
				Value: aws.String(wr.GetRegion(regionId).Description()),
			},
			{
				Type:  aws.String("TERM_MATCH"),
				Field: aws.String("tenancy"),
				Value: aws.String("shared"),
			},
			{
				Type:  aws.String("TERM_MATCH"),
				Field: aws.String("preInstalledSw"),
				Value: aws.String("NA"),
			},
			{
				Type:  aws.String("TERM_MATCH"),
				Field: aws.String(attrKey),
				Value: aws.String(attrValue),
			},
		},
	}
}

func (wr *AwsClientWrapper) GetRegions() map[string]endpoints.Region {
	return endpoints.AwsPartition().Regions()
}
