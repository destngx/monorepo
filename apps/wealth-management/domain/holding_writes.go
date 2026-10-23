package domain

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Holding writes (WM-FILL-10, WM-FILL-11): the fund and crypto tabs were
// edited in the sheet only; these record a trade or a balance from the app.

// FundTradeInput records units bought (positive) or sold (negative) of a
// fund certificate on a date (yyyy-mm-dd).
type FundTradeInput struct {
	Code  string  `json:"code"`
	Date  string  `json:"date"`
	Units float64 `json:"units"`
}

// maxFundUnits bounds one trade; a larger number is a typo.
const maxFundUnits = 1e9

var holdingSymbolPattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9.\-]{0,19}$`)

// Normalized upper-cases the code and trims the date.
func (input FundTradeInput) Normalized() FundTradeInput {
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Date = strings.TrimSpace(input.Date)
	input.Units = math.Round(input.Units*100) / 100
	return input
}

// Validate checks a normalized trade; today bounds the date.
func (input FundTradeInput) Validate(today time.Time) (time.Time, error) {
	if !holdingSymbolPattern.MatchString(input.Code) {
		return time.Time{}, fmt.Errorf("%w: fund code %q must be letters and digits, like VESAF", ErrValidation, input.Code)
	}
	date, err := time.Parse(time.DateOnly, input.Date)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: date %q must be yyyy-mm-dd", ErrValidation, input.Date)
	}
	if date.After(Today(today)) {
		return time.Time{}, fmt.Errorf("%w: date %s is in the future", ErrValidation, input.Date)
	}
	if input.Units == 0 || math.IsNaN(input.Units) || math.Abs(input.Units) > maxFundUnits {
		return time.Time{}, fmt.Errorf("%w: units must be a number other than 0, negative for a sale", ErrValidation)
	}
	return date, nil
}

// FundHistoryEntry formats a trade as ParseFundHistory reads it:
// `d-m-yyyy,units`, with a comma as the decimal separator.
func FundHistoryEntry(date time.Time, units float64) string {
	return fmt.Sprintf("%d-%d-%d,%s", date.Day(), int(date.Month()), date.Year(), strings.Replace(strconv.FormatFloat(units, 'f', -1, 64), ".", ",", 1))
}

// AppendFundHistory adds an entry to a History cell.
func AppendFundHistory(cell string, entry string) string {
	cell = strings.TrimRight(strings.TrimSpace(cell), ";")
	if cell == "" {
		return entry
	}
	return cell + ";" + entry
}

// FundTradeResult is the fund's row after the trade.
type FundTradeResult struct {
	Code    string  `json:"code"`
	Units   float64 `json:"units"`
	Entry   string  `json:"entry"`
	Row     int     `json:"row"`
	Created bool    `json:"created"`
}

// CryptoHoldingInput sets how much of a coin a platform holds now.
type CryptoHoldingInput struct {
	Platform string  `json:"platform"`
	Symbol   string  `json:"symbol"`
	Quantity float64 `json:"quantity"`
	Note     string  `json:"note,omitempty"`
}

func (input CryptoHoldingInput) Normalized() CryptoHoldingInput {
	input.Platform = strings.TrimSpace(input.Platform)
	input.Symbol = strings.ToUpper(strings.TrimSpace(input.Symbol))
	input.Note = strings.TrimSpace(input.Note)
	return input
}

func (input CryptoHoldingInput) Validate() error {
	switch {
	case input.Platform == "":
		return fmt.Errorf("%w: platform is required", ErrValidation)
	case len([]rune(input.Platform)) > 60:
		return fmt.Errorf("%w: platform is longer than 60 characters", ErrValidation)
	case !holdingSymbolPattern.MatchString(input.Symbol):
		return fmt.Errorf("%w: coin %q must be a ticker like BTC", ErrValidation, input.Symbol)
	case input.Quantity < 0 || math.IsNaN(input.Quantity) || math.IsInf(input.Quantity, 0):
		return fmt.Errorf("%w: quantity of %s must be 0 or more", ErrValidation, input.Symbol)
	case len([]rune(input.Note)) > 200:
		return fmt.Errorf("%w: note is longer than 200 characters", ErrValidation)
	}
	return nil
}

// CryptoHoldingsInput is a batch from one screen (WM-FILL-11).
type CryptoHoldingsInput struct {
	Holdings []CryptoHoldingInput `json:"holdings"`
}

const maxCryptoHoldingsBatch = 50

// Normalized validates and normalizes the batch; a coin listed twice for
// a platform is refused, since only one quantity can be its balance.
func (input CryptoHoldingsInput) Normalized() ([]CryptoHoldingInput, error) {
	if len(input.Holdings) == 0 {
		return nil, fmt.Errorf("%w: at least one holding is required", ErrValidation)
	}
	if len(input.Holdings) > maxCryptoHoldingsBatch {
		return nil, fmt.Errorf("%w: at most %d holdings at once", ErrValidation, maxCryptoHoldingsBatch)
	}
	seen := map[string]bool{}
	holdings := make([]CryptoHoldingInput, len(input.Holdings))
	for index, holding := range input.Holdings {
		holding = holding.Normalized()
		if err := holding.Validate(); err != nil {
			return nil, fmt.Errorf("holding %d: %w", index+1, err)
		}
		key := FoldText(holding.Platform) + "|" + holding.Symbol
		if seen[key] {
			return nil, fmt.Errorf("%w: %s on %s is listed twice", ErrValidation, holding.Symbol, holding.Platform)
		}
		seen[key] = true
		holdings[index] = holding
	}
	return holdings, nil
}

// CryptoHoldingResult is one coin's row after the write.
type CryptoHoldingResult struct {
	Platform string  `json:"platform"`
	Symbol   string  `json:"symbol"`
	Quantity float64 `json:"quantity"`
	Row      int     `json:"row"`
	Created  bool    `json:"created"`
}
