package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Smart Capture (§6.2). The model only reads materials into events; every
// later step (consolidation, account matching, transfer pairing, ledger
// dedup and the reconciliation inputs) is deterministic engine code.

type CaptureMaterialKind string

const (
	CaptureImage CaptureMaterialKind = "image"
	CaptureText  CaptureMaterialKind = "text"
)

// CaptureImageTypes are the image formats sent to the model. The UI converts
// anything else (HEIC included) to JPEG while downscaling.
var CaptureImageTypes = []string{"image/png", "image/jpeg", "image/webp", "image/gif"}

// CaptureLimits bound one capture batch (WM-CAP-1).
type CaptureLimits struct {
	MaxItems      int `json:"maxItems"`
	MaxItemBytes  int `json:"maxItemBytes"`
	MaxBatchBytes int `json:"maxBatchBytes"`
}

func DefaultCaptureLimits() CaptureLimits {
	return CaptureLimits{MaxItems: 20, MaxItemBytes: 5 << 20, MaxBatchBytes: 20 << 20}
}

// CaptureMaterial is one screenshot or pasted text. Materials live only for
// the request: they are never logged, cached or written (WM-CAP-10).
type CaptureMaterial struct {
	ID        string              `json:"id"`
	Kind      CaptureMaterialKind `json:"kind"`
	Name      string              `json:"name,omitempty"`
	MediaType string              `json:"mediaType,omitempty"`
	// Data is the image; JSON carries it as base64.
	Data []byte `json:"data,omitempty"`
	Text string `json:"text,omitempty"`
}

// Size is the number of bytes the material contributes to a batch.
func (m CaptureMaterial) Size() int {
	return len(m.Data) + len(m.Text)
}

// Label names the material in memos, e.g. "screenshot".
func (m CaptureMaterial) Label() string {
	if m.Kind == CaptureImage {
		return "screenshot"
	}
	return "text"
}

func ValidateCaptureMaterials(materials []CaptureMaterial, limits CaptureLimits) error {
	if len(materials) == 0 {
		return fmt.Errorf("%w: at least one material is required", ErrValidation)
	}
	if len(materials) > limits.MaxItems {
		return fmt.Errorf("%w: at most %d materials per batch, got %d", ErrValidation, limits.MaxItems, len(materials))
	}
	var problems []string
	seen := map[string]bool{}
	total := 0
	for index, material := range materials {
		label := fmt.Sprintf("material %d", index+1)
		id := strings.TrimSpace(material.ID)
		switch {
		case id == "":
			problems = append(problems, label+": id is required")
		case seen[id]:
			problems = append(problems, fmt.Sprintf("%s: duplicate id %q", label, id))
		}
		seen[id] = true
		switch material.Kind {
		case CaptureImage:
			if len(material.Data) == 0 {
				problems = append(problems, label+": image data is required")
			}
			if !containsString(CaptureImageTypes, material.MediaType) {
				problems = append(problems, fmt.Sprintf("%s: media type %q is not one of %s", label, material.MediaType, strings.Join(CaptureImageTypes, ", ")))
			}
		case CaptureText:
			if strings.TrimSpace(material.Text) == "" {
				problems = append(problems, label+": text is required")
			}
		default:
			problems = append(problems, fmt.Sprintf("%s: kind must be image or text", label))
		}
		if material.Size() > limits.MaxItemBytes {
			problems = append(problems, fmt.Sprintf("%s: %d bytes exceeds the %d byte limit", label, material.Size(), limits.MaxItemBytes))
		}
		total += material.Size()
	}
	if total > limits.MaxBatchBytes {
		problems = append(problems, fmt.Sprintf("batch: %d bytes exceeds the %d byte limit", total, limits.MaxBatchBytes))
	}
	return validationError(problems)
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

type CaptureDirection string

const (
	CaptureIn  CaptureDirection = "in"
	CaptureOut CaptureDirection = "out"
)

// CaptureEvent is one money movement read from one material (WM-CAP-2).
type CaptureEvent struct {
	OccurredAt   string           `json:"occurredAt"`
	Amount       float64          `json:"amount"`
	Currency     string           `json:"currency"`
	Direction    CaptureDirection `json:"direction"`
	Counterparty string           `json:"counterparty"`
	Description  string           `json:"description"`
	Reference    string           `json:"reference"`
	SourceApp    string           `json:"sourceApp"`
	AccountHint  string           `json:"accountHint"`
	BalanceAfter *float64         `json:"balanceAfter"`
	Fee          *float64         `json:"fee"`
	Confidence   float64          `json:"confidence"`
	Quote        string           `json:"quote"`

	// Set by the engine after parsing.
	MaterialID string    `json:"materialId"`
	At         time.Time `json:"-"`
	HasTime    bool      `json:"-"`
}

// CaptureExtraction is the model's reply for one material.
type CaptureExtraction struct {
	Events     []CaptureEvent `json:"events"`
	Unreadable bool           `json:"unreadable"`
	Reason     string         `json:"reason"`
	// Dropped counts events without a usable date, amount or direction.
	Dropped int `json:"-"`
}

const (
	captureDefaultConfidence = 0.5
	// CaptureLowConfidence highlights fields below it on the review screen (WM-CAP-8).
	CaptureLowConfidence = 0.7
)

// ParseCaptureExtraction parses a reply to CaptureFormatInstruction and
// normalises its events. Events that cannot become a transaction are
// dropped and counted rather than guessed (WM-CAP-2).
func ParseCaptureExtraction(content string, materialID string) (CaptureExtraction, error) {
	var extraction CaptureExtraction
	if err := ParseAIJSON(content, &extraction); err != nil {
		return extraction, err
	}
	extraction.Reason = strings.TrimSpace(extraction.Reason)
	events := make([]CaptureEvent, 0, len(extraction.Events))
	for _, event := range extraction.Events {
		event.MaterialID = materialID
		if !event.normalize() {
			extraction.Dropped++
			continue
		}
		events = append(events, event)
	}
	extraction.Events = events
	if len(events) == 0 && !extraction.Unreadable && extraction.Dropped > 0 {
		extraction.Unreadable = true
		if extraction.Reason == "" {
			extraction.Reason = "no event had a usable date, amount and direction"
		}
	}
	return extraction, nil
}

func (e *CaptureEvent) normalize() bool {
	e.Counterparty = strings.TrimSpace(e.Counterparty)
	e.Description = strings.TrimSpace(e.Description)
	e.Reference = strings.TrimSpace(e.Reference)
	e.SourceApp = strings.TrimSpace(e.SourceApp)
	e.AccountHint = strings.TrimSpace(e.AccountHint)
	e.Quote = strings.TrimSpace(e.Quote)
	e.Currency = normalizeCurrency(e.Currency)

	switch strings.ToLower(strings.TrimSpace(string(e.Direction))) {
	case "in", "credit", "deposit", "+":
		e.Direction = CaptureIn
	case "out", "debit", "payment", "-":
		e.Direction = CaptureOut
	case "":
		if e.Amount >= 0 {
			return false
		}
		e.Direction = CaptureOut
	default:
		return false
	}
	e.Amount = math.Abs(e.Amount)
	if e.Amount == 0 || math.IsNaN(e.Amount) || math.IsInf(e.Amount, 0) {
		return false
	}
	at, hasTime, ok := parseCaptureTime(e.OccurredAt)
	if !ok {
		return false
	}
	e.At, e.HasTime = at, hasTime
	if e.Fee != nil && (*e.Fee <= 0 || math.IsNaN(*e.Fee)) {
		e.Fee = nil
	}
	if e.BalanceAfter != nil && math.IsNaN(*e.BalanceAfter) {
		e.BalanceAfter = nil
	}
	if math.IsNaN(e.Confidence) || e.Confidence <= 0 {
		e.Confidence = captureDefaultConfidence
	}
	e.Confidence = math.Min(e.Confidence, 1)
	return true
}

func normalizeCurrency(value string) string {
	switch upper := strings.ToUpper(strings.TrimSpace(value)); upper {
	case "", "Đ", "D", "VNĐ", "VND", "VNDONG":
		return string(CurrencyVND)
	case "$", "US$", "USD":
		return string(CurrencyUSD)
	default:
		return upper
	}
}

// parseCaptureTime reads "yyyy-mm-dd[ HH:MM[:SS]]" (or a sheet date) as a
// local wall-clock time.
func parseCaptureTime(value string) (time.Time, bool, bool) {
	trimmed := strings.TrimSpace(value)
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02T15:04", "2006-01-02 15:04"} {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return parsed, true, true
		}
	}
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		local := parsed.In(AppLocation)
		return time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), local.Minute(), local.Second(), 0, time.UTC), true, true
	}
	if parsed, err := ParseSheetDate(trimmed); err == nil {
		return parsed, false, true
	}
	return time.Time{}, false, false
}

// Date is the calendar day of the event.
func (e CaptureEvent) Date() time.Time {
	return time.Date(e.At.Year(), e.At.Month(), e.At.Day(), 0, 0, 0, 0, time.UTC)
}

// CaptureFormatInstruction is the extraction output contract (WM-CAP-2),
// appended by the engine to the tasks/capture-extract prompt.
var CaptureFormatInstruction = `IMPORTANT: Respond with ONLY a valid JSON object (no markdown, no backticks, no explanation) in this exact format:
{
  "events": [
    {
      "occurredAt": "yyyy-mm-dd HH:MM (or yyyy-mm-dd when no time is shown)",
      "amount": 1250000,
      "currency": "VND",
      "direction": "<in or out>",
      "counterparty": "Who was paid or who paid, as written",
      "description": "Transfer content or merchant description, as written",
      "reference": "Transaction reference / code, or empty",
      "sourceApp": "Bank or wallet app the material comes from, e.g. Sacombank, MoMo, Binance",
      "accountHint": "Account identity shown: bank or wallet name and last 4 digits, e.g. Sacombank *1234",
      "balanceAfter": 3456789,
      "fee": null,
      "confidence": 0.9,
      "quote": "The exact text the event was read from"
    }
  ],
  "unreadable": false,
  "reason": ""
}

Rules:
- One event per money movement; a screenshot listing several transactions yields several events.
- amount is a positive plain number in the shown currency: 1.250.000đ = 1250000, -50k = 50000, +1,250,000 VND = 1250000. The sign goes into direction ("out" for money leaving the account, "in" for money arriving).
- Dates shown as dd/MM/yyyy are day first. When the year is missing, use the year that makes the date closest to, but not after, today.
- balanceAfter is the balance shown after the movement (e.g. "SD: 3,456,789VND"), else null. fee is a separate fee amount, else null.
- confidence is 0..1 for the event as a whole; lower it for any field you had to infer.
- Never invent events. If nothing can be read, return no events, "unreadable": true and a short reason.`
