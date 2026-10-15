package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Merchant category codes (WM-RULE-7, WM-CARD-9): a payment's cashback rate
// depends on the MCC of the merchant's terminal, which neither the ledger
// nor the statement shows (F2). A payment's MCC comes from an "mcc:NNNN"
// tag (confirmed), else from the Merchants tab mapping payee patterns to
// codes, each an observation that is assumed or confirmed (C4). Merchants
// can change their MCC (F3), so a mapping may start on a date: past
// payments keep the code in effect when they were made.

const (
	maxMCC    = 9999
	mccTagKey = "mcc:"
)

// MCCRange is an inclusive range of merchant category codes; a single code
// has From == To.
type MCCRange struct {
	From int `json:"from"`
	To   int `json:"to"`
}

// Contains reports whether code falls in the range.
func (r MCCRange) Contains(code int) bool {
	return code >= r.From && code <= r.To
}

// MCCStatus is how far a payment's MCC is known.
type MCCStatus string

const (
	MCCConfirmed MCCStatus = "confirmed"
	MCCAssumed   MCCStatus = "assumed"
	MCCUnknown   MCCStatus = "unknown"
)

// PaymentMCC is a payment's merchant category code and how sure it is.
type PaymentMCC struct {
	Code   int       `json:"code,omitempty"`
	Status MCCStatus `json:"status"`
}

// Known reports whether the payment has an MCC, confirmed or assumed.
func (m PaymentMCC) Known() bool {
	return m.Status == MCCConfirmed || m.Status == MCCAssumed
}

// MerchantMCC maps payees containing Pattern to an MCC, from the From date
// on; nil From means since always. Confidence is MCCAssumed or MCCConfirmed.
type MerchantMCC struct {
	Pattern    string     `json:"pattern"`
	MCC        int        `json:"mcc"`
	From       *time.Time `json:"from,omitempty"`
	Confidence MCCStatus  `json:"confidence"`
	Note       string     `json:"note,omitempty"`
}

func (m MerchantMCC) inEffectOn(date time.Time) bool {
	return m.From == nil || !m.From.After(date)
}

// supersedes reports whether m starts later than other.
func (m MerchantMCC) supersedes(other MerchantMCC) bool {
	return m.From != nil && (other.From == nil || m.From.After(*other.From))
}

var mccRangeDash = regexp.MustCompile(`\s*-\s*`)

// ParseMCCList reads codes and ranges separated by commas, semicolons or
// spaces, e.g. "5411, 5499, 5812-5814". Codes are 0-9999; a sheet cell that
// dropped a leading zero (742 for 0742) still means the same code.
func ParseMCCList(text string) ([]MCCRange, error) {
	var ranges []MCCRange
	normalized := mccRangeDash.ReplaceAllString(strings.TrimSpace(text), "-")
	for _, token := range strings.FieldsFunc(normalized, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n'
	}) {
		from, to, isRange := strings.Cut(token, "-")
		if !isRange {
			to = from
		}
		low, lowErr := parseMCC(from)
		high, highErr := parseMCC(to)
		if lowErr != nil || highErr != nil {
			return nil, fmt.Errorf("%q is not an MCC or MCC range (0-9999)", token)
		}
		if low > high {
			return nil, fmt.Errorf("MCC range %q runs backwards", token)
		}
		ranges = append(ranges, MCCRange{From: low, To: high})
	}
	return ranges, nil
}

func parseMCC(text string) (int, error) {
	code, err := strconv.Atoi(text)
	if err != nil || code < 0 || code > maxMCC || strings.HasPrefix(text, "+") {
		return 0, fmt.Errorf("invalid MCC %q", text)
	}
	return code, nil
}

// FormatMCCList writes ranges back in the sheet's notation.
func FormatMCCList(ranges []MCCRange) string {
	parts := make([]string, 0, len(ranges))
	for _, r := range ranges {
		if r.From == r.To {
			parts = append(parts, fmt.Sprintf("%04d", r.From))
		} else {
			parts = append(parts, fmt.Sprintf("%04d-%04d", r.From, r.To))
		}
	}
	return strings.Join(parts, ", ")
}

func mccListContains(ranges []MCCRange, code int) bool {
	for _, r := range ranges {
		if r.Contains(code) {
			return true
		}
	}
	return false
}

// TransactionMCC is a payment's MCC: an "mcc:NNNN" tag wins (confirmed),
// else the longest merchant pattern found as whole words in the payee among
// the mappings in effect on the payment date, the latest-starting one for a
// pattern mapped more than once, with that mapping's confidence.
func TransactionMCC(t Transaction, merchants []MerchantMCC) PaymentMCC {
	for _, tag := range t.Tags {
		value, found := strings.CutPrefix(strings.ToLower(strings.TrimSpace(tag)), mccTagKey)
		if !found {
			continue
		}
		if code, err := parseMCC(strings.TrimSpace(value)); err == nil {
			return PaymentMCC{Code: code, Status: MCCConfirmed}
		}
	}
	payee := " " + FoldText(t.Payee) + " "
	var best *MerchantMCC
	bestLength := 0
	for i := range merchants {
		merchant := &merchants[i]
		pattern := FoldText(merchant.Pattern)
		if pattern == "" || !merchant.inEffectOn(t.Date) || !strings.Contains(payee, " "+pattern+" ") {
			continue
		}
		if len(pattern) > bestLength || (len(pattern) == bestLength && merchant.supersedes(*best)) {
			best, bestLength = merchant, len(pattern)
		}
	}
	if best == nil {
		return PaymentMCC{Status: MCCUnknown}
	}
	status := best.Confidence
	if status != MCCConfirmed {
		status = MCCAssumed
	}
	return PaymentMCC{Code: best.MCC, Status: status}
}
