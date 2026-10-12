package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"strings"
)

// Investment holdings parsing (WM-DATA-7) and valuation (WM-RULE-8).

type holdingSection struct {
	headerKeywords  []string
	symbolHeaders   []string
	defaultCurrency domain.Currency
}

var (
	cryptoSection = holdingSection{
		headerKeywords:  []string{"Currency", "Spot/Fund", "Token"},
		symbolHeaders:   []string{"Token", "Currency", "Asset", "Symbol"},
		defaultCurrency: domain.CurrencyUSD,
	}
	fundSection = holdingSection{
		headerKeywords:  []string{"Index", "Total Unit", "Certificate"},
		symbolHeaders:   []string{"Certificate", "Index", "Asset", "Symbol"},
		defaultCurrency: domain.CurrencyVND,
	}
)

func parseHoldings(rows [][]string, section holdingSection) domain.InvestmentHoldingSection {
	empty := domain.InvestmentHoldingSection{Headers: []string{}, Holdings: []map[string]any{}, Items: []domain.Holding{}}
	headerIdx := findHeaderRow(rows, section.headerKeywords)
	if headerIdx < 0 {
		return empty
	}
	headers := normaliseHeaders(rows[headerIdx])
	columns := classifyHoldingColumns(headers, section)

	result := empty
	result.Headers = headers
	lastPlatform := ""
	for _, row := range rows[headerIdx+1:] {
		if platform := strings.TrimSpace(cellAt(row, 0)); platform != "" {
			lastPlatform = platform
		}
		item := map[string]any{}
		hasData := false
		for index, header := range headers {
			if index == 0 {
				item[header] = lastPlatform
				continue
			}
			value := cellAt(row, index)
			item[header] = value
			if strings.TrimSpace(value) != "" {
				hasData = true
			}
		}
		if !hasData {
			continue
		}
		result.Holdings = append(result.Holdings, item)
		result.Items = append(result.Items, typedHolding(row, lastPlatform, columns, section))
	}
	return result
}

func findHeaderRow(rows [][]string, keywords []string) int {
	for i, row := range rows {
		for _, cell := range row {
			trimmed := strings.TrimSpace(cell)
			for _, keyword := range keywords {
				if trimmed == keyword {
					return i
				}
			}
		}
	}
	return -1
}

func normaliseHeaders(raw []string) []string {
	headers := make([]string, len(raw))
	for i, header := range raw {
		trimmed := strings.TrimSpace(header)
		switch {
		case trimmed != "":
			headers[i] = header
		case i == 0:
			headers[i] = "Platform"
		default:
			headers[i] = fmt.Sprintf("Col%d", i)
		}
	}
	return headers
}

type holdingColumns struct {
	symbol, quantity, price, value int
	priceCurrency, valueCurrency   domain.Currency
}

func classifyHoldingColumns(headers []string, section holdingSection) holdingColumns {
	columns := holdingColumns{symbol: -1, quantity: -1, price: -1, value: -1}
	for _, wanted := range section.symbolHeaders {
		for i, header := range headers {
			if columns.symbol < 0 && strings.TrimSpace(header) == wanted {
				columns.symbol = i
			}
		}
	}
	for i, header := range headers {
		lower := strings.ToLower(strings.TrimSpace(header))
		switch {
		case columns.value < 0 && isValueHeader(lower):
			columns.value = i
			columns.valueCurrency = headerCurrency(lower, section.defaultCurrency)
		case columns.price < 0 && containsAny(lower, "price", "rate", "cost"):
			columns.price = i
			columns.priceCurrency = headerCurrency(lower, section.defaultCurrency)
		case columns.quantity < 0 && i != columns.symbol && containsAny(lower, "amount", "qty", "quantity", "unit", "holding", "balance", "spot", "coin"):
			columns.quantity = i
		}
	}
	return columns
}

// isValueHeader: a VND value/balance/total column, or exactly "total"/"value".
func isValueHeader(lower string) bool {
	if lower == "total" || lower == "value" {
		return true
	}
	return strings.Contains(lower, "vnd") && containsAny(lower, "value", "balance", "total")
}

func headerCurrency(lower string, fallback domain.Currency) domain.Currency {
	switch {
	case strings.Contains(lower, "vnd"):
		return domain.CurrencyVND
	case strings.Contains(lower, "usdt"):
		return domain.CurrencyUSDT
	case strings.Contains(lower, "usd"):
		return domain.CurrencyUSD
	default:
		return fallback
	}
}

func typedHolding(row []string, platform string, columns holdingColumns, section holdingSection) domain.Holding {
	holding := domain.Holding{Platform: platform, Currency: section.defaultCurrency}
	if columns.symbol >= 0 {
		holding.Symbol = strings.TrimSpace(cellAt(row, columns.symbol))
	}
	if columns.quantity >= 0 {
		holding.Quantity = parseOptionalNumber(cellAt(row, columns.quantity))
	}
	if columns.price >= 0 {
		if price := parseOptionalNumber(cellAt(row, columns.price)); price != nil && *price > 0 {
			holding.Price = price
			holding.Currency = columns.priceCurrency
		}
	}
	if columns.value >= 0 {
		if value := parseOptionalNumber(cellAt(row, columns.value)); value != nil && *value != 0 {
			holding.Value = value
			holding.Currency = columns.valueCurrency
			holding.ValueSource = domain.HoldingValueFromSheet
			return holding
		}
	}
	if holding.Quantity != nil && holding.Price != nil {
		value := *holding.Quantity * *holding.Price
		holding.Value = &value
		holding.ValueSource = domain.HoldingValueFromQuantityPrice
	}
	return holding
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
