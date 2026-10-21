package domain

import (
	"errors"
	"testing"
)

const today = "2026-10-21"

func TestGivenAVisitInputWhenValidatingThenItIsCleanedAndDefaulted(t *testing.T) {
	in, err := VisitInput{PersonID: "p", Date: "2025-03-04", Title: "  Khám   tổng quát ", ProviderName: " BV  Chợ Rẫy "}.Validate(today)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if in.Type != "other" || in.Title != "Khám tổng quát" || in.ProviderName != "BV Chợ Rẫy" || in.ProviderKind != "other" {
		t.Fatalf("cleaned = %+v", in)
	}
}

func TestGivenBadVisitInputsWhenValidatingThenEachIsRejected(t *testing.T) {
	cases := map[string]VisitInput{
		"no person":    {Date: "2025-01-01"},
		"bad date":     {PersonID: "p", Date: "04/03/2025"},
		"future date":  {PersonID: "p", Date: "2026-10-22"},
		"unknown type": {PersonID: "p", Date: "2025-01-01", Type: "spa"},
		"unknown kind": {PersonID: "p", Date: "2025-01-01", ProviderKind: "gym"},
	}
	for name, in := range cases {
		if _, err := in.Validate(today); !errors.Is(err, ErrValidation) {
			t.Errorf("%s: err = %v, want validation error", name, err)
		}
	}
}

func TestGivenAPersonInputWhenValidatingThenBirthDateAndSexAreChecked(t *testing.T) {
	if _, err := (PersonInput{DisplayName: "An", BirthDate: "2030-01-01"}).Validate(today); !errors.Is(err, ErrValidation) {
		t.Fatalf("future birth date accepted: %v", err)
	}
	if _, err := (PersonInput{DisplayName: "An", Sex: "x"}).Validate(today); !errors.Is(err, ErrValidation) {
		t.Fatalf("unknown sex accepted: %v", err)
	}
	in, err := PersonInput{DisplayName: " An ", BirthDate: "2019-05-01", Sex: "female"}.Validate(today)
	if err != nil || in.DisplayName != "An" {
		t.Fatalf("Validate = %+v, %v", in, err)
	}
}

func TestGivenFileBytesWhenSniffingThenOnlyAcceptedTypesPass(t *testing.T) {
	heic := append([]byte{0, 0, 0, 24}, []byte("ftypheic\x00\x00\x00\x00mif1heic")...)
	cases := map[string]struct {
		data []byte
		want string
	}{
		"pdf":  {[]byte("%PDF-1.7\n..."), "application/pdf"},
		"jpeg": {[]byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0x10, 'J', 'F', 'I', 'F', 0}, "image/jpeg"},
		"png":  {[]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), "image/png"},
		"heic": {heic, "image/heic"},
	}
	for name, tc := range cases {
		if got, err := SniffMIME(tc.data); err != nil || got != tc.want {
			t.Errorf("%s: SniffMIME = %q, %v", name, got, err)
		}
	}
	for name, data := range map[string][]byte{"html": []byte("<html><body>"), "text": []byte("hello"), "empty": nil} {
		if _, err := SniffMIME(data); !errors.Is(err, ErrValidation) {
			t.Errorf("%s accepted: %v", name, err)
		}
	}
}
