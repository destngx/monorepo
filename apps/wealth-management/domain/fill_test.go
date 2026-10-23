package domain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func fillContext() FillContext {
	return FillContext{
		Today: time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC),
		Lists: map[FillList][]string{
			FillAccounts:   {"Ví MoMo", "Bank A Savings", "Bank A Card", "Cash"},
			FillCategories: {"Food", "Fast Food", "Transport", "Shopping", "Entertainment"},
			FillTags:       {"work", "family"},
		},
	}
}

func mustForm(t *testing.T, key string) FillForm {
	t.Helper()
	form, ok := FillFormFor(key)
	if !ok {
		t.Fatalf("form %q is not registered", key)
	}
	return form
}

func TestParseFillAmount(t *testing.T) {
	cases := map[string]float64{
		"50M": 50e6, "50m": 50e6, "50tr": 50e6, "50 triệu": 50e6, "1,5tr": 1.5e6, "1.2M": 1.2e6,
		"260k": 260e3, "260K": 260e3, "10 nghìn": 10e3, "2 tỷ": 2e9, "1.5B": 1.5e9,
		"50.000.000": 50e6, "50.000.000 đ": 50e6, "1,250,000": 1.25e6, "1,250,000.50": 1250000.5,
		"260.000": 260e3, "2.5": 2.5, "0.5": 0.5, "12000": 12000, "45,000 VND": 45e3,
	}
	for written, want := range cases {
		got, ok := ParseFillAmount(written)
		if !ok || got != want {
			t.Errorf("ParseFillAmount(%q) = %v, %v; want %v", written, got, ok, want)
		}
	}
	if got, ok := ParseFillAmount(float64(1250)); !ok || got != 1250 {
		t.Errorf("a JSON number reads as itself, got %v %v", got, ok)
	}
	for _, written := range []string{"", "lots", "M", "k"} {
		if _, ok := ParseFillAmount(written); ok {
			t.Errorf("ParseFillAmount(%q) should fail", written)
		}
	}
}

func TestMatchFillName(t *testing.T) {
	names := []string{"Ví MoMo", "Bank A Savings", "Bank A Card", "Food", "Fast Food"}
	cases := []struct {
		written string
		want    string
		ok      bool
	}{
		{"vi momo", "Ví MoMo", true},
		{"MoMo", "Ví MoMo", true},
		{"savings", "Bank A Savings", true},
		// Equal wins over containing: "food" is in both foods.
		{"food", "Food", true},
		{"Bank A", "", false},
		{"Bank B", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := MatchFillName(c.written, names)
		if got != c.want || ok != c.ok {
			t.Errorf("MatchFillName(%q) = %q, %v; want %q, %v", c.written, got, ok, c.want, c.ok)
		}
	}
}

func TestParseFillAnswerGoal(t *testing.T) {
	form := mustForm(t, "goal")
	answer := `{"values": {"name": "Trip to Japan", "target": "50M", "targetDate": "2027-06", "priority": 2, "status": "Active"}, "notes": ["No starting amount was given."]}`
	result, err := ParseFillAnswer(answer, form, fillContext(), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"name": "Trip to Japan", "target": 50e6, "targetDate": "2027-06", "priority": float64(2), "status": "active"}
	if !reflect.DeepEqual(result.Values, want) {
		t.Errorf("values = %#v, want %#v", result.Values, want)
	}
	if strings.Join(result.Filled, ",") != "name,target,targetDate,priority,status" {
		t.Errorf("filled = %v", result.Filled)
	}
	if len(result.Notes) != 1 || result.Notes[0].Message != "No starting amount was given." {
		t.Errorf("notes = %#v", result.Notes)
	}
}

func TestParseFillAnswerLeavesBadValuesBlankWithANote(t *testing.T) {
	form := mustForm(t, "goal")
	answer := `{"values": {"target": "a lot", "targetDate": "someday", "priority": 0, "status": "maybe"}}`
	result, err := ParseFillAnswer(answer, form, fillContext(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Values) != 0 || len(result.Filled) != 0 {
		t.Fatalf("bad values must not be filled: %#v", result)
	}
	notes := map[string]string{}
	for _, note := range result.Notes {
		notes[note.Field] = note.Message
	}
	for field, want := range map[string]string{
		"target":     `"a lot" is not an amount.`,
		"targetDate": `"someday" is not a month.`,
		"priority":   "Priority must be at least 1.",
		"status":     `"maybe" is not one of active, paused, done.`,
		"name":       "Name is still needed.",
	} {
		if notes[field] != want {
			t.Errorf("note for %s = %q, want %q", field, notes[field], want)
		}
	}
}

func TestParseFillAnswerEditMarksOnlyChangedFields(t *testing.T) {
	form := mustForm(t, "goal")
	current := map[string]any{"name": "Laptop", "target": float64(30e6), "targetDate": "2026-12", "status": "active"}
	answer := `{"values": {"name": "Laptop", "targetDate": "2027-12"}}`
	result, err := ParseFillAnswer(answer, form, fillContext(), current)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(result.Filled, ",") != "targetDate" {
		t.Errorf("filled = %v, want only targetDate", result.Filled)
	}
	for _, note := range result.Notes {
		if note.Field == "name" {
			t.Errorf("a name the form holds is not missing: %#v", note)
		}
	}
}

func TestParseFillAnswerMatchesSheetNames(t *testing.T) {
	form := mustForm(t, "transaction")
	answer := `{"values": {"accountName": "momo", "date": "2026-10-22", "payee": "Highlands", "category": "transport", "tags": ["Work", "trip"], "payment": "45k"}}`
	result, err := ParseFillAnswer(answer, form, fillContext(), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"accountName": "Ví MoMo", "date": "2026-10-22", "payee": "Highlands", "category": "Transport",
		"tags": []string{"work", "trip"}, "payment": float64(45000),
	}
	if !reflect.DeepEqual(result.Values, want) {
		t.Errorf("values = %#v, want %#v", result.Values, want)
	}
}

func TestParseFillAnswerUnknownAccountIsANote(t *testing.T) {
	form := mustForm(t, "transaction")
	result, err := ParseFillAnswer(`{"values": {"accountName": "Bank A", "category": "Groceries"}}`, form, fillContext(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, set := result.Values["accountName"]; set {
		t.Fatal("an ambiguous account must stay blank")
	}
	messages := map[string]string{}
	for _, note := range result.Notes {
		messages[note.Field] = note.Message
	}
	if messages["accountName"] != `"Bank A" could be Bank A Savings or Bank A Card; pick one.` {
		t.Errorf("account note = %q", messages["accountName"])
	}
	if messages["category"] != `No category matches "Groceries".` {
		t.Errorf("category note = %q", messages["category"])
	}
}

func TestParseFillAnswerRowsKeepUnmatchedRows(t *testing.T) {
	form := mustForm(t, "budget-limits")
	answer := `{"values": {}, "rows": [{"category": "food", "limit": "6M", "scope": "month"}, {"category": "Pets", "limit": 500000}]}`
	result, err := ParseFillAnswer(answer, form, fillContext(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("rows = %#v", result.Rows)
	}
	if !reflect.DeepEqual(result.Rows[0], map[string]any{"category": "Food", "limit": 6e6, "scope": "month"}) {
		t.Errorf("first row = %#v", result.Rows[0])
	}
	if result.Rows[1]["category"] != "Pets" {
		t.Errorf("an unmatched row keeps what was written, got %#v", result.Rows[1])
	}
	if len(result.Notes) != 1 || result.Notes[0].Row == nil || *result.Notes[0].Row != 1 || result.Notes[0].Field != "category" {
		t.Errorf("notes = %#v", result.Notes)
	}
	if !reflect.DeepEqual(result.Filled, []string{"rows"}) {
		t.Errorf("filled = %v", result.Filled)
	}
}

func TestParseFillAnswerRejectsNonJSON(t *testing.T) {
	_, err := ParseFillAnswer("Sure, here is the goal.", mustForm(t, "goal"), fillContext(), nil)
	if !errors.Is(err, ErrAIOutput) {
		t.Fatalf("err = %v, want ErrAIOutput", err)
	}
}

func TestFillRequestValidate(t *testing.T) {
	limits := DefaultCaptureLimits()
	image := FillImage{MediaType: "image/png", Data: []byte{1}}
	cases := []struct {
		name    string
		request FillRequest
		want    string
	}{
		{"unknown form", FillRequest{Form: "taxes", Text: "x"}, `unknown form "taxes"`},
		{"nothing to read", FillRequest{Form: "goal", Text: "  "}, "describe what to fill or add an image"},
		{"too long", FillRequest{Form: "goal", Text: strings.Repeat("a", FillMaxText+1)}, "at most 2000"},
		{"image on a text form", FillRequest{Form: "budget-move", Images: []FillImage{image}}, "takes text only"},
		{"too many images", FillRequest{Form: "recurring", Images: []FillImage{image, image, image, image, image}}, "at most 4 images"},
		{"bad image type", FillRequest{Form: "recurring", Images: []FillImage{{MediaType: "image/heic", Data: []byte{1}}}}, "media type"},
	}
	for _, c := range cases {
		_, err := c.request.Validate(limits)
		if !errors.Is(err, ErrValidation) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want a validation error containing %q", c.name, err, c.want)
		}
	}
	if _, err := (FillRequest{Form: "recurring", Text: "Netflix", Images: []FillImage{image}}).Validate(limits); err != nil {
		t.Errorf("a valid request failed: %v", err)
	}
}

func TestFillFormsAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, form := range FillForms() {
		if seen[form.Key] {
			t.Errorf("form %q is registered twice", form.Key)
		}
		seen[form.Key] = true
		if len(form.Fields)+len(form.Rows) == 0 {
			t.Errorf("form %q has no fields", form.Key)
		}
		if len(form.Rows) > 0 && form.RowsLabel == "" {
			t.Errorf("form %q has rows but no row label", form.Key)
		}
		for _, field := range append(append([]FillField{}, form.Fields...), form.Rows...) {
			if (field.Type == FillName || field.Type == FillNames) && field.List == "" {
				t.Errorf("%s.%s matches names but has no list", form.Key, field.Name)
			}
			if field.Type == FillChoice && len(field.Choices) == 0 {
				t.Errorf("%s.%s is a choice without choices", form.Key, field.Name)
			}
		}
	}
	if len(seen) != 17 {
		t.Errorf("%d forms, want the 17 of WM-FILL-15", len(seen))
	}
}

func TestFillInstruction(t *testing.T) {
	form := mustForm(t, "transaction")
	instruction := FillInstruction(form, fillContext(), map[string]any{"payee": "Grab"}, "Payees are written in Vietnamese.")
	for _, want := range []string{
		`"Transaction" form`, "Today is 2026-10-23 (Friday)", "- accountName (Account): a name from the accounts list",
		"Ví MoMo", "Return only the fields the request changes", `{"payee":"Grab"}`, "Payees are written in Vietnamese.",
		"Set either payment (money out) or deposit (money in), never both.",
	} {
		if !strings.Contains(instruction, want) {
			t.Errorf("instruction lacks %q", want)
		}
	}
}

func TestGivenTheFillFormsThenOnlyTheFourTextFormsRefuseImages(t *testing.T) {
	textOnly := map[string]bool{"budget-limits": true, "budget-move": true, "category-definition": true, "allocation": true}
	for _, form := range FillForms() {
		if form.Images == textOnly[form.Key] {
			t.Errorf("%s: images = %v, want %v (WM-FILL-15)", form.Key, form.Images, !textOnly[form.Key])
		}
	}
}
