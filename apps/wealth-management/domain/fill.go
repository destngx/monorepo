package domain

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

// AI form filling (§10.5, D47). The model only reads a request into field
// text; the engine checks every value against the form's schema and
// matches names against the sheet. Nothing is written.

// FillFieldType is how a field's value is checked (WM-FILL-3).
type FillFieldType string

const (
	FillText FillFieldType = "text"
	// FillMoney is an amount in the form's currency; "50M", "260k" and
	// "1,5tr" are read too.
	FillMoney   FillFieldType = "money"
	FillNumber  FillFieldType = "number"
	FillPercent FillFieldType = "percent"
	// FillDay is a day of the month, 1-31.
	FillDay FillFieldType = "day"
	// FillDate is yyyy-mm-dd; FillMonth is yyyy-mm.
	FillDate   FillFieldType = "date"
	FillMonth  FillFieldType = "month"
	FillChoice FillFieldType = "choice"
	FillFlag   FillFieldType = "flag"
	// FillName is a name from one of the sheet's lists (FillField.List).
	FillName FillFieldType = "name"
	// FillNames is a list of names, e.g. tags.
	FillNames FillFieldType = "names"
)

// FillList names a list of sheet names a field matches against.
type FillList string

const (
	FillAccounts   FillList = "accounts"
	FillCategories FillList = "categories"
	FillTags       FillList = "tags"
	FillGoals      FillList = "goals"
	FillCards      FillList = "cards"
	FillPrograms   FillList = "programs"
	FillFunds      FillList = "funds"
	FillPlatforms  FillList = "platforms"
	FillProducts   FillList = "products"
	FillPayees     FillList = "payees"
)

// FillField is one field of a form schema (WM-FILL-2). Name is the key the
// UI form uses.
type FillField struct {
	Name  string        `json:"name"`
	Label string        `json:"label"`
	Type  FillFieldType `json:"type"`
	// List is what a name field matches against.
	List FillList `json:"list,omitempty"`
	// New lets a name field keep a name that is not in the list yet (a new
	// payee, fund code or platform).
	New      bool     `json:"new,omitempty"`
	Choices  []string `json:"choices,omitempty"`
	Required bool     `json:"required,omitempty"`
	// Hint tells the model what the field holds.
	Hint string `json:"hint,omitempty"`
	// Suggest is how the model picks the value when the request does not
	// give it; a field without it is left out instead.
	Suggest string   `json:"suggest,omitempty"`
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
}

// FillForm is a form the engine can fill. Rows are the fields of each row
// of a list form (WM-FILL-5).
type FillForm struct {
	Key    string      `json:"key"`
	Title  string      `json:"title"`
	Fields []FillField `json:"fields"`
	Rows   []FillField `json:"rows,omitempty"`
	// RowsLabel names a row in notes, e.g. "limit".
	RowsLabel string `json:"rowsLabel,omitempty"`
	// Images is whether the form takes images (WM-FILL-15).
	Images bool `json:"images"`
	// Guide is form-specific advice for the model.
	Guide string `json:"-"`
	// Page is the app route the chat opens the form on (WM-FILL-7); a form
	// without one edits a chosen item and is not offered in the chat.
	Page string `json:"page,omitempty"`
}

// FillChatForm is a fill the chat ran: the user opens it in its form on
// Page, checks it and saves there (WM-FILL-7).
type FillChatForm struct {
	Form   string     `json:"form"`
	Title  string     `json:"title"`
	Page   string     `json:"page"`
	Text   string     `json:"text"`
	Result FillResult `json:"result"`
}

func (f FillForm) lists() []FillList {
	var lists []FillList
	for _, field := range slices.Concat(f.Fields, f.Rows) {
		if field.List != "" && !slices.Contains(lists, field.List) {
			lists = append(lists, field.List)
		}
	}
	return lists
}

// Lists are the sheet lists the form's fields match against.
func (f FillForm) Lists() []FillList { return f.lists() }

// FillImage is one image of a fill request; JSON carries Data as base64.
type FillImage struct {
	MediaType string `json:"mediaType"`
	Data      []byte `json:"data"`
}

const (
	FillMaxText   = 2000
	FillMaxImages = 4
	// fillMaxNotes bounds the model's own notes.
	fillMaxNotes   = 5
	fillMaxNoteLen = 300
	fillMaxTextLen = 500
	fillMaxRows    = 50
)

// FillRequest is one fill (WM-FILL-1). Current is what the form holds, for
// edits (WM-FILL-4).
type FillRequest struct {
	Form    string         `json:"form"`
	Text    string         `json:"text,omitempty"`
	Images  []FillImage    `json:"images,omitempty"`
	Current map[string]any `json:"current,omitempty"`
	Model   string         `json:"model,omitempty"`
}

// Validate checks the request against the form and the image limits.
func (r FillRequest) Validate(limits CaptureLimits) (FillForm, error) {
	form, ok := FillFormFor(r.Form)
	if !ok {
		return FillForm{}, fmt.Errorf("%w: unknown form %q", ErrValidation, r.Form)
	}
	text := strings.TrimSpace(r.Text)
	if text == "" && len(r.Images) == 0 {
		return form, fmt.Errorf("%w: describe what to fill or add an image", ErrValidation)
	}
	if n := len([]rune(text)); n > FillMaxText {
		return form, fmt.Errorf("%w: the text is %d characters, at most %d", ErrValidation, n, FillMaxText)
	}
	if len(r.Images) > 0 && !form.Images {
		return form, fmt.Errorf("%w: the %s form takes text only", ErrValidation, form.Title)
	}
	if len(r.Images) > FillMaxImages {
		return form, fmt.Errorf("%w: at most %d images, got %d", ErrValidation, FillMaxImages, len(r.Images))
	}
	total := 0
	for index, image := range r.Images {
		label := fmt.Sprintf("image %d", index+1)
		switch {
		case len(image.Data) == 0:
			return form, fmt.Errorf("%w: %s is empty", ErrValidation, label)
		case !slices.Contains(CaptureImageTypes, image.MediaType):
			return form, fmt.Errorf("%w: %s: media type %q is not one of %s", ErrValidation, label, image.MediaType, strings.Join(CaptureImageTypes, ", "))
		case len(image.Data) > limits.MaxItemBytes:
			return form, fmt.Errorf("%w: %s: %d bytes exceeds the %d byte limit", ErrValidation, label, len(image.Data), limits.MaxItemBytes)
		}
		total += len(image.Data)
	}
	if total > limits.MaxBatchBytes {
		return form, fmt.Errorf("%w: images total %d bytes, over the %d byte limit", ErrValidation, total, limits.MaxBatchBytes)
	}
	return form, nil
}

// FillContext is what the engine knows when it checks an answer.
type FillContext struct {
	Today time.Time
	// Lists holds the sheet names per list.
	Lists map[FillList][]string
	// Facts is form-specific context for the model, e.g. the goals with
	// their priorities; it is never used to check values.
	Facts string
}

// FillNote explains a field left blank or something the model wants to say
// (WM-FILL-3). Row is the 0-based row of a list form.
type FillNote struct {
	Field string `json:"field,omitempty"`
	// Label is the field's label, for the UI to show with the message.
	Label   string `json:"label,omitempty"`
	Row     *int   `json:"row,omitempty"`
	Message string `json:"message"`
}

// FillResult is the answer of a fill (WM-FILL-1). Values hold only the
// fields the answer set; Filled names those that differ from Current.
type FillResult struct {
	Form   string           `json:"form"`
	Values map[string]any   `json:"values"`
	Rows   []map[string]any `json:"rows,omitempty"`
	Filled []string         `json:"filled"`
	Notes  []FillNote       `json:"notes"`
}

type fillAnswer struct {
	Values map[string]any   `json:"values"`
	Rows   []map[string]any `json:"rows"`
	Notes  []string         `json:"notes"`
}

// ParseFillAnswer checks the model's answer field by field. Only an answer
// that is not a JSON object is an error; a bad value becomes a note.
func ParseFillAnswer(content string, form FillForm, context FillContext, current map[string]any) (FillResult, error) {
	var answer fillAnswer
	if err := ParseAIJSON(content, &answer); err != nil {
		return FillResult{}, err
	}
	result := FillResult{Form: form.Key, Values: map[string]any{}, Filled: []string{}, Notes: []FillNote{}}
	for _, field := range form.Fields {
		raw, present := answer.Values[field.Name]
		if !present || isBlankFill(raw) {
			continue
		}
		value, problem := field.check(raw, context)
		if problem != "" {
			result.Notes = append(result.Notes, FillNote{Field: field.Name, Label: field.Label, Message: problem})
			continue
		}
		result.Values[field.Name] = value
		if !sameFillValue(value, current[field.Name]) {
			result.Filled = append(result.Filled, field.Name)
		}
	}
	if len(form.Rows) > 0 {
		result.Rows = []map[string]any{}
		for index, raw := range answer.Rows {
			if index == fillMaxRows {
				result.Notes = append(result.Notes, FillNote{Message: fmt.Sprintf("Only the first %d %ss were kept.", fillMaxRows, form.RowsLabel)})
				break
			}
			row := map[string]any{}
			for _, field := range form.Rows {
				value, present := raw[field.Name]
				if !present || isBlankFill(value) {
					continue
				}
				checked, problem := field.check(value, context)
				if problem != "" {
					// The row stays so nothing asked for disappears (WM-FILL-5).
					row[field.Name] = fillText(value)
					rowIndex := len(result.Rows)
					result.Notes = append(result.Notes, FillNote{Field: field.Name, Label: field.Label, Row: &rowIndex, Message: problem})
					continue
				}
				row[field.Name] = checked
			}
			if len(row) > 0 {
				result.Rows = append(result.Rows, row)
			}
		}
		if len(result.Rows) > 0 {
			result.Filled = append(result.Filled, "rows")
		}
	}
	for _, field := range form.Fields {
		if !field.Required {
			continue
		}
		if _, set := result.Values[field.Name]; set || !isBlankFill(current[field.Name]) || hasFieldNote(result.Notes, field.Name) {
			continue
		}
		result.Notes = append(result.Notes, FillNote{Field: field.Name, Label: field.Label, Message: field.Label + " is still needed."})
	}
	for _, note := range answer.Notes {
		if len(result.Notes) >= fillMaxNotes+len(form.Fields) {
			break
		}
		if note = strings.TrimSpace(note); note != "" {
			result.Notes = append(result.Notes, FillNote{Message: truncateRunes(note, fillMaxNoteLen)})
		}
	}
	return result, nil
}

func hasFieldNote(notes []FillNote, field string) bool {
	for _, note := range notes {
		if note.Field == field && note.Row == nil {
			return true
		}
	}
	return false
}

func isBlankFill(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	case []any:
		return len(typed) == 0
	}
	return false
}

func sameFillValue(value, current any) bool {
	if isBlankFill(current) {
		return false
	}
	a, _ := json.Marshal(value)
	b, _ := json.Marshal(normaliseFillCurrent(current))
	return string(a) == string(b)
}

// normaliseFillCurrent turns JSON numbers into float64 so 5 and 5.0 compare
// equal.
func normaliseFillCurrent(value any) any {
	switch typed := value.(type) {
	case int:
		return float64(typed)
	case json.Number:
		if number, err := typed.Float64(); err == nil {
			return number
		}
	}
	return value
}

func (field FillField) check(raw any, context FillContext) (any, string) {
	switch field.Type {
	case FillText:
		return truncateRunes(strings.TrimSpace(fillText(raw)), fillMaxTextLen), ""
	case FillMoney:
		amount, ok := ParseFillAmount(raw)
		if !ok {
			return nil, fmt.Sprintf("%q is not an amount.", fillText(raw))
		}
		return field.bounded(math.Round(amount*100) / 100)
	case FillNumber, FillPercent:
		number, ok := parseFillNumber(raw, field.Type == FillPercent)
		if !ok {
			return nil, fmt.Sprintf("%q is not a number.", fillText(raw))
		}
		return field.bounded(number)
	case FillDay:
		number, ok := parseFillNumber(raw, false)
		if !ok || number != math.Trunc(number) || number < 1 || number > 31 {
			return nil, fmt.Sprintf("%q is not a day of the month (1-31).", fillText(raw))
		}
		return field.bounded(number)
	case FillDate:
		date, ok := parseFillDate(fillText(raw))
		if !ok {
			return nil, fmt.Sprintf("%q is not a date.", fillText(raw))
		}
		return date, ""
	case FillMonth:
		text := strings.TrimSpace(fillText(raw))
		if len(text) >= 7 {
			if _, err := time.Parse("2006-01", text[:7]); err == nil {
				return text[:7], ""
			}
		}
		return nil, fmt.Sprintf("%q is not a month.", text)
	case FillChoice:
		text := fillText(raw)
		for _, choice := range field.Choices {
			if FoldText(choice) == FoldText(text) {
				return choice, ""
			}
		}
		return nil, fmt.Sprintf("%q is not one of %s.", text, strings.Join(field.Choices, ", "))
	case FillFlag:
		switch typed := raw.(type) {
		case bool:
			return typed, ""
		case string:
			switch FoldText(typed) {
			case "true", "yes", "y", "x", "1":
				return true, ""
			case "false", "no", "n", "0":
				return false, ""
			}
		}
		return nil, fmt.Sprintf("%q is not yes or no.", fillText(raw))
	case FillName:
		return field.matchName(fillText(raw), context)
	case FillNames:
		var names []string
		switch typed := raw.(type) {
		case []any:
			for _, item := range typed {
				names = append(names, fillText(item))
			}
		default:
			names = strings.Split(fillText(raw), ",")
		}
		var matched []string
		var problems []string
		for _, name := range names {
			if strings.TrimSpace(name) == "" {
				continue
			}
			value, problem := field.matchName(name, context)
			if problem != "" {
				problems = append(problems, problem)
				continue
			}
			if name := value.(string); !slices.Contains(matched, name) {
				matched = append(matched, name)
			}
		}
		if len(matched) == 0 && len(problems) > 0 {
			return nil, strings.Join(problems, " ")
		}
		return matched, ""
	}
	return nil, fmt.Sprintf("unknown field type %q", field.Type)
}

func (field FillField) bounded(value float64) (any, string) {
	if field.Min != nil && value < *field.Min {
		return nil, fmt.Sprintf("%s must be at least %s.", field.Label, formatFillNumber(*field.Min))
	}
	if field.Max != nil && value > *field.Max {
		return nil, fmt.Sprintf("%s must be at most %s.", field.Label, formatFillNumber(*field.Max))
	}
	return value, ""
}

// matchName finds the sheet name a written name means: equal ignoring case
// and accents, or the only name containing it (WM-FILL-3).
func (field FillField) matchName(written string, context FillContext) (any, string) {
	written = strings.TrimSpace(written)
	names := context.Lists[field.List]
	if name, ok := MatchFillName(written, names); ok {
		return name, ""
	}
	if field.New {
		return truncateRunes(written, fillMaxTextLen), ""
	}
	what := strings.TrimSuffix(string(field.List), "s")
	if field.List == FillCategories {
		what = "category"
	}
	if candidates := fillCandidates(written, names); len(candidates) > 1 {
		return nil, fmt.Sprintf("%q could be %s; pick one.", written, strings.Join(candidates, " or "))
	}
	return nil, fmt.Sprintf("No %s matches %q.", what, written)
}

// MatchFillName returns the one name in names that written means.
func MatchFillName(written string, names []string) (string, bool) {
	folded := FoldText(written)
	if folded == "" {
		return "", false
	}
	for _, name := range names {
		if FoldText(name) == folded {
			return name, true
		}
	}
	if candidates := fillCandidates(written, names); len(candidates) == 1 {
		return candidates[0], true
	}
	return "", false
}

func fillCandidates(written string, names []string) []string {
	folded := " " + FoldText(written) + " "
	if strings.TrimSpace(folded) == "" {
		return nil
	}
	var candidates []string
	for _, name := range names {
		if strings.Contains(" "+FoldText(name)+" ", folded) && !slices.Contains(candidates, name) {
			candidates = append(candidates, name)
		}
	}
	if len(candidates) > 3 {
		candidates = candidates[:3]
	}
	return candidates
}

func fillText(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return formatFillNumber(typed)
	case bool:
		return strconv.FormatBool(typed)
	case nil:
		return ""
	}
	payload, _ := json.Marshal(value)
	return string(payload)
}

func formatFillNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// fillMultipliers are the amount suffixes people write, longest first.
var fillMultipliers = []struct {
	suffix string
	factor float64
}{
	{"trieu", 1e6}, {"ngan", 1e3}, {"nghin", 1e3}, {"ty", 1e9}, {"tr", 1e6},
	{"mil", 1e6}, {"bn", 1e9}, {"m", 1e6}, {"k", 1e3}, {"b", 1e9},
}

// ParseFillAmount reads an amount: a number, or text such as "50M",
// "260k", "1,5tr", "2 tỷ", "50.000.000 đ" or "1,250,000".
func ParseFillAmount(value any) (float64, bool) {
	if number, ok := value.(float64); ok {
		return number, !math.IsNaN(number) && !math.IsInf(number, 0)
	}
	text := FoldText(fillText(value))
	text = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(text), "vnd"), "d"))
	// FoldText turned separators into spaces; put the number back together.
	raw := strings.ToLower(strings.TrimSpace(fillText(value)))
	factor := 1.0
	for _, multiplier := range fillMultipliers {
		if strings.HasSuffix(text, multiplier.suffix) {
			number := strings.TrimSpace(strings.TrimSuffix(text, multiplier.suffix))
			if number == "" {
				return 0, false
			}
			factor = multiplier.factor
			raw = numericPart(raw)
			break
		}
	}
	if factor == 1 {
		raw = numericPart(raw)
	}
	number, ok := parseGroupedNumber(raw, factor != 1)
	if !ok {
		return 0, false
	}
	return number * factor, true
}

// numericPart keeps the digits, separators and sign of a written amount.
func numericPart(text string) string {
	var b strings.Builder
	for _, r := range text {
		if (r >= '0' && r <= '9') || r == '.' || r == ',' || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// parseGroupedNumber reads "1.250.000", "1,250,000", "1,5" or "2.5". With
// a suffix ("1,5tr") a single separator is a decimal point.
func parseGroupedNumber(text string, scaled bool) (float64, bool) {
	if text == "" || text == "-" {
		return 0, false
	}
	dots, commas := strings.Count(text, "."), strings.Count(text, ",")
	switch {
	case dots > 0 && commas > 0:
		// The last separator is the decimal point.
		if strings.LastIndex(text, ".") > strings.LastIndex(text, ",") {
			text = strings.ReplaceAll(text, ",", "")
		} else {
			text = strings.ReplaceAll(strings.ReplaceAll(text, ".", ""), ",", ".")
		}
	case dots > 1:
		text = strings.ReplaceAll(text, ".", "")
	case commas > 1:
		text = strings.ReplaceAll(text, ",", "")
	case commas == 1:
		before, after, _ := strings.Cut(text, ",")
		if !scaled && len(after) == 3 && before != "" && before != "-" {
			text = before + after
		} else {
			text = before + "." + after
		}
	case dots == 1:
		before, after, _ := strings.Cut(text, ".")
		if !scaled && len(after) == 3 && before != "" && before != "-" && before != "0" {
			text = before + after
		}
	}
	number, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, false
	}
	return number, true
}

func parseFillNumber(value any, percent bool) (float64, bool) {
	if number, ok := value.(float64); ok {
		return number, !math.IsNaN(number) && !math.IsInf(number, 0)
	}
	text := strings.TrimSpace(fillText(value))
	if percent {
		text = strings.TrimSpace(strings.TrimSuffix(text, "%"))
	}
	text = strings.ReplaceAll(text, " ", "")
	if strings.Count(text, ",") == 1 && !strings.Contains(text, ".") {
		text = strings.Replace(text, ",", ".", 1)
	}
	number, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, false
	}
	return number, true
}

func parseFillDate(text string) (string, bool) {
	text = strings.TrimSpace(text)
	for _, layout := range []string{"2006-01-02", "2/1/2006", "2-1-2006"} {
		if date, err := time.Parse(layout, text); err == nil {
			return date.Format("2006-01-02"), true
		}
	}
	if len(text) > 10 {
		if date, err := time.Parse("2006-01-02", text[:10]); err == nil {
			return date.Format("2006-01-02"), true
		}
	}
	return "", false
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
