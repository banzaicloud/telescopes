package ec2_productinfo

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

type ProductInfo struct {
	renewalInterval time.Duration
	session         *session.Session
	vmAttrStore     *cache.Cache
}

func NewProductInfo(ri time.Duration, cache *cache.Cache) (*ProductInfo, error) {
	session, err := session.NewSession(&aws.Config{})
	if err != nil {
		log.WithError(err).Error("Error creating AWS session")
		return nil, err
	}
	return &ProductInfo{
		session:         session,
		vmAttrStore:     cache,
		renewalInterval: ri,
	}, nil
}

func (e *ProductInfo) Start(ctx context.Context) {

	renew := func() {
		log.Info("renewing product info")
		attributes := []string{"mem", "vcpu"}
		for _, attr := range attributes {
			e.renewAttrValues(attr)
		}
	}

	renew()
	ticker := time.NewTicker(e.renewalInterval)
	for {
		select {
		case <-ticker.C:
			renew()
		case <-ctx.Done():
			log.Debugf("closing ticker")
			ticker.Stop()
			return
		}
	}
}

func (e *ProductInfo) GetSortedAttrValues(attribute string) ([]float64, error) {
	var attrValues []float64
	attrCacheKey := e.getAttrKey(attribute)
	if cachedVal, ok := e.vmAttrStore.Get(attrCacheKey); ok {
		log.Debugf("Getting available %s values from cache.", attribute)
		attrValues = cachedVal.([]float64)
	} else {
		values, err := e.renewAttrValues(attribute)
		if err != nil {
			return nil, err
		}
		attrValues = values
	}
	log.Debugf("%s attribute values sorted: %v", attribute, attrValues)
	return attrValues, nil
}

func (e *ProductInfo) getAttrKey(attribute string) string {
	return fmt.Sprintf("/banzaicloud.com/recommender/ec2/attrValues/%s", attribute)
}

func (e *ProductInfo) renewAttrValues(attribute string) ([]float64, error) {
	values, err := e.getSortedAttrValuesFromAPI(attribute)
	if err != nil {
		return nil, err
	}
	e.vmAttrStore.Set(e.getAttrKey(attribute), values, e.renewalInterval)
	return values, nil
}

func (e *ProductInfo) getSortedAttrValuesFromAPI(attribute string) ([]float64, error) {
	log.Debugf("Getting available %s values from AWS API.", attribute)
	pricingSvc := pricing.New(e.session, &aws.Config{Region: aws.String("us-east-1")})
	apiValues, err := pricingSvc.GetAttributeValues(&pricing.GetAttributeValuesInput{
		ServiceCode:   aws.String("AmazonEC2"),
		AttributeName: aws.String(attribute),
	})
	if err != nil {
		return nil, err
	}

	var values []float64
	for _, attrValue := range apiValues.AttributeValues {
		floatValue, err := strconv.ParseFloat(strings.Split(*attrValue.Value, " ")[0], 32)
		if err != nil {
			return nil, err
		}
		values = append(values, floatValue)
	}
	sort.Float64s(values)
	return values, nil
}
