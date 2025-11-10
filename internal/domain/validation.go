package domain

import (
	"strings"

	amznads "github.com/LittleAksMax/amazon-ads-api-sdk-go"
)

// IsValidMarketplace checks if a marketplace code is valid, done using the country
// map from the amazon-ads-api-sdk-go package's AmazonCountryToRegionMap
func IsValidMarketplace(marketplace string) bool {
	_, ok := amznads.AmazonCountryToRegionMap[NormaliseMarketplace(marketplace)]
	return ok
}

func NormaliseMarketplace(marketplace string) string {
	return strings.TrimSpace(strings.ToUpper(marketplace))
}
