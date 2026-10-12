package domain

type FmarketAction string

const (
	FmarketActionProductsFilterNav FmarketAction = "getProductsFilterNav"
	FmarketActionIssuers           FmarketAction = "getIssuers"
	FmarketActionBankInterestRates FmarketAction = "getBankInterestRates"
	FmarketActionProductDetails    FmarketAction = "getProductDetails"
	FmarketActionNavHistory        FmarketAction = "getNavHistory"
	FmarketActionGoldPriceHistory  FmarketAction = "getGoldPriceHistory"
	FmarketActionUsdRateHistory    FmarketAction = "getUsdRateHistory"
	FmarketActionGoldProducts      FmarketAction = "getGoldProducts"
)

type FmarketRequest struct {
	Action FmarketAction  `json:"action"`
	Params map[string]any `json:"params,omitempty"`
}

// FmarketActions lists the supported action names, for schemas.
func FmarketActions() []string {
	return []string{
		string(FmarketActionProductsFilterNav), string(FmarketActionIssuers), string(FmarketActionBankInterestRates),
		string(FmarketActionProductDetails), string(FmarketActionNavHistory), string(FmarketActionGoldPriceHistory),
		string(FmarketActionUsdRateHistory), string(FmarketActionGoldProducts),
	}
}
