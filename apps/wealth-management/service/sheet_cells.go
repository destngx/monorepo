package service

import (
	"apps/wealth-management/domain"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// Cell-level parsing and formatting for the Google Sheet data contract.

func cellAt(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return row[index]
}

func rowAt(rows [][]string, index int) []string {
	if index < 0 || index >= len(rows) {
		return nil
	}
	return rows[index]
}

func parseNumber(value string) float64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	trimmed = strings.ReplaceAll(trimmed, ",", "")
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0
	}
	return parsed
}

// parseOptionalNumber distinguishes an empty or non-numeric cell (nil) from zero.
func parseOptionalNumber(value string) *float64 {
	trimmed := strings.ReplaceAll(strings.TrimSpace(value), ",", "")
	if trimmed == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func nullableNumber(value string) *float64 {
	parsed := parseNumber(value)
	if parsed == 0 {
		return nil
	}
	return &parsed
}

func nullableGoalProgress(value string) *float64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parsed := parseNumber(trimmed) * 100
	return &parsed
}

func nullableString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func splitTags(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	return cleanStrings(strings.Split(value, ","))
}

// parseCleared reads column H; "x" is what the app writes, the others are
// marks entered by hand or by older versions.
func parseCleared(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "x", "y", "yes", "true", "✓":
		return true
	}
	return false
}

func formatDueDate(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if serial, err := strconv.Atoi(trimmed); err == nil && serial > 1000 {
		date := domain.SerialToDate(float64(serial))
		return fmt.Sprintf("%d/%d/%d", date.Day(), int(date.Month()), date.Year())
	}
	return trimmed
}

func parseVND(value string) float64 {
	clean := strings.NewReplacer(".", "", "₫", "", " ", "", ",", "").Replace(value)
	parsed, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0
	}
	return parsed
}

// resolveCurrency prefers the stored currency (column I) and falls back to
// the name heuristic: Binance accounts are USDT, everything else VND.
func resolveCurrency(stored string, name string) domain.Currency {
	currency := domain.Currency(strings.ToUpper(strings.TrimSpace(stored)))
	if currency.Valid() {
		return currency
	}
	if strings.Contains(strings.ToLower(name), "binance") {
		return domain.CurrencyUSDT
	}
	return domain.CurrencyVND
}

func onlyDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func nilToString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func cleanStrings(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}

// clearedCell renders the cleared flag for column H.
func clearedCell(cleared bool) string {
	if cleared {
		return "x"
	}
	return ""
}

// formatTransactionAmount writes amounts of USD and USDT accounts (entered
// in USD) as a formula that converts to VND at the sheet's live USD/VND rate.
// The account's currency comes from its column I (WM-TX-9).
func formatTransactionAmount(amount *float64, accountName string, currency domain.Currency) any {
	if amount == nil {
		return ""
	}
	if currency == domain.CurrencyUSD || currency == domain.CurrencyUSDT {
		return fmt.Sprintf(`=%v * IF(INDIRECT("A" & ROW())="%s"; GOOGLEFINANCE("CURRENCY:USDVND"); 1)`, *amount, accountName)
	}
	return *amount
}

func normalizeKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// rowETag fingerprints the first columns of a row exactly as read from the
// sheet, so a later write can detect that the row changed in between.
func rowETag(row []string, columns int) string {
	hash := sha256.New()
	for i := 0; i < columns; i++ {
		hash.Write([]byte(cellAt(row, i)))
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))[:16]
}

// columnLetter converts a zero-based column index to A1 notation (0 = A, 26 = AA).
func columnLetter(index int) string {
	letters := ""
	for index >= 0 {
		letters = string(rune('A'+index%26)) + letters
		index = index/26 - 1
	}
	return letters
}
