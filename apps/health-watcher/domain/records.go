package domain

import (
	"time"
	"unicode/utf8"
)

// DateLayout is the storage and API layout of calendar dates.
const DateLayout = "2006-01-02"

// Date validates a calendar date in DateLayout.
func Date(s string) (string, error) {
	d, err := time.Parse(DateLayout, s)
	if err != nil {
		return "", Invalid("date must be YYYY-MM-DD")
	}
	return d.Format(DateLayout), nil
}

// Sex is the sex at birth, used later to pick reference ranges (HW-TEN-3).
type Sex string

const (
	SexUnknown Sex = ""
	SexFemale  Sex = "female"
	SexMale    Sex = "male"
)

// ParseSex validates a sex value; empty means unknown.
func ParseSex(s string) (Sex, error) {
	switch v := Sex(s); v {
	case SexUnknown, SexFemale, SexMale:
		return v, nil
	}
	return "", Invalid("sex must be female, male or empty")
}

// Person is the subject of records (HW-TEN-3).
type Person struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"displayName"`
	BirthDate   string    `json:"birthDate,omitempty"`
	Sex         Sex       `json:"sex,omitempty"`
	UserID      string    `json:"userId,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	LastVisitOn string    `json:"lastVisitOn,omitempty"`
	VisitCount  int       `json:"visitCount"`
}

// PersonInput is a new or changed person.
type PersonInput struct {
	DisplayName string `json:"displayName"`
	BirthDate   string `json:"birthDate"`
	Sex         string `json:"sex"`
	UserID      string `json:"userId"`
}

// Validate cleans the input and checks its fields. today bounds the birth date.
func (in PersonInput) Validate(today string) (PersonInput, error) {
	name, err := DisplayName(in.DisplayName)
	if err != nil {
		return in, err
	}
	in.DisplayName = name
	if in.BirthDate != "" {
		if in.BirthDate, err = Date(in.BirthDate); err != nil {
			return in, Invalid("birth date must be YYYY-MM-DD")
		}
		if in.BirthDate > today {
			return in, Invalid("birth date is in the future")
		}
	}
	sex, err := ParseSex(in.Sex)
	if err != nil {
		return in, err
	}
	in.Sex = string(sex)
	return in, nil
}

// ProviderKind is the kind of place that produced a record (HW-REC-5).
type ProviderKind string

const (
	ProviderHospital ProviderKind = "hospital"
	ProviderClinic   ProviderKind = "clinic"
	ProviderDental   ProviderKind = "dental"
	ProviderLab      ProviderKind = "lab"
	ProviderPharmacy ProviderKind = "pharmacy"
	ProviderOther    ProviderKind = "other"
)

// ParseProviderKind validates a provider kind; empty means other.
func ParseProviderKind(s string) (ProviderKind, error) {
	switch k := ProviderKind(s); k {
	case "":
		return ProviderOther, nil
	case ProviderHospital, ProviderClinic, ProviderDental, ProviderLab, ProviderPharmacy, ProviderOther:
		return k, nil
	}
	return "", Invalid("unknown provider kind %q", s)
}

// Provider is a hospital, clinic, dentist or lab in a tenant.
type Provider struct {
	ID   string       `json:"id"`
	Name string       `json:"name"`
	Kind ProviderKind `json:"kind"`
}

// VisitType is the kind of encounter (HW-REC-1).
type VisitType string

// VisitTypes lists every visit type in display order.
var VisitTypes = []VisitType{
	"checkup", "lab", "imaging", "dental", "specialist", "emergency", "inpatient", "vaccination", "other",
}

// ParseVisitType validates a visit type; empty means other.
func ParseVisitType(s string) (VisitType, error) {
	if s == "" {
		return "other", nil
	}
	for _, t := range VisitTypes {
		if string(t) == s {
			return t, nil
		}
	}
	return "", Invalid("unknown visit type %q", s)
}

// Visit is one encounter on one date with one provider for one person.
type Visit struct {
	ID            string    `json:"id"`
	PersonID      string    `json:"personId"`
	Date          string    `json:"date"`
	Type          VisitType `json:"type"`
	Title         string    `json:"title"`
	Note          string    `json:"note"`
	Provider      *Provider `json:"provider"`
	DocumentCount int       `json:"documentCount"`
	// FlaggedCount counts approved values the paper flagged (HW-FIND-1).
	FlaggedCount int        `json:"flaggedCount"`
	Documents    []Document `json:"documents,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

// VisitInput is a new or changed visit. ProviderName creates or reuses a
// provider by its folded name.
type VisitInput struct {
	PersonID     string `json:"personId"`
	Date         string `json:"date"`
	Type         string `json:"type"`
	Title        string `json:"title"`
	Note         string `json:"note"`
	ProviderName string `json:"providerName"`
	ProviderKind string `json:"providerKind"`
}

// Validate cleans the input and checks its fields. today bounds the date.
func (in VisitInput) Validate(today string) (VisitInput, error) {
	if in.PersonID == "" {
		return in, Invalid("person is required")
	}
	var err error
	if in.Date, err = Date(in.Date); err != nil {
		return in, err
	}
	if in.Date > today {
		return in, Invalid("visit date is in the future")
	}
	t, err := ParseVisitType(in.Type)
	if err != nil {
		return in, err
	}
	in.Type = string(t)
	in.Title = CleanText(in.Title)
	if utf8.RuneCountInString(in.Title) > 120 {
		return in, Invalid("title is longer than 120 characters")
	}
	if utf8.RuneCountInString(in.Note) > 4000 {
		return in, Invalid("note is longer than 4000 characters")
	}
	in.ProviderName = CleanText(in.ProviderName)
	if utf8.RuneCountInString(in.ProviderName) > 120 {
		return in, Invalid("provider name is longer than 120 characters")
	}
	kind, err := ParseProviderKind(in.ProviderKind)
	if err != nil {
		return in, err
	}
	in.ProviderKind = string(kind)
	return in, nil
}

// DocumentKind is the kind of original (HW-REC-2).
type DocumentKind string

// DocumentKinds lists every document kind in display order.
var DocumentKinds = []DocumentKind{
	"lab_result", "imaging_report", "imaging_film", "prescription", "discharge_summary",
	"dental_record", "record_book_page", "invoice", "other",
}

// ParseDocumentKind validates a document kind; empty means other.
func ParseDocumentKind(s string) (DocumentKind, error) {
	if s == "" {
		return "other", nil
	}
	for _, k := range DocumentKinds {
		if string(k) == s {
			return k, nil
		}
	}
	return "", Invalid("unknown document kind %q", s)
}

// Document is one original filed under a visit. It has one or more files:
// one PDF, or several photographed pages.
type Document struct {
	ID        string       `json:"id"`
	VisitID   string       `json:"visitId"`
	PersonID  string       `json:"personId"`
	Kind      DocumentKind `json:"kind"`
	Title     string       `json:"title"`
	Files     []StoredFile `json:"files"`
	CreatedAt time.Time    `json:"createdAt"`
}

// StoredFile is an original file kept by its SHA-256 (HW-OPS-2).
type StoredFile struct {
	ID           string    `json:"id"`
	SHA256       string    `json:"sha256"`
	MIME         string    `json:"mime"`
	Size         int64     `json:"size"`
	OriginalName string    `json:"originalName"`
	CreatedAt    time.Time `json:"createdAt"`
}
