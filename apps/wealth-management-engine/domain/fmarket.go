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
