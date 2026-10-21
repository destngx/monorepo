package domain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestGivenNewSettingsWhenValidatingThenTheDefaultsPass(t *testing.T) {
	s, err := DefaultSettings().Validate()
	if err != nil {
		t.Fatal(err)
	}
	if s.Addr() != "127.0.0.1:9191" || s.AIEnabled() {
		t.Fatalf("defaults = %+v", s)
	}
}

func TestGivenSettingsWithSpacesWhenValidatingThenTheyAreCleaned(t *testing.T) {
	in := DefaultSettings()
	in.Host, in.MacOCRBin, in.AIBaseURL, in.AIAPIKey = " ::1 ", " /opt/tools/../bin/mac-ocr ", " http://localhost:8080/v1/ ", " sk-test "
	s, err := in.Validate()
	if err != nil {
		t.Fatal(err)
	}
	if s.Host != "::1" || s.Addr() != "[::1]:9191" || s.MacOCRBin != "/opt/bin/mac-ocr" || s.AIBaseURL != "http://localhost:8080/v1" ||
		s.AIAPIKey != "sk-test" || !s.AIEnabled() {
		t.Fatalf("cleaned = %+v", s)
	}
}

func TestGivenBadSettingsWhenValidatingThenEachIsRejected(t *testing.T) {
	cases := map[string]func(*Settings){
		"host name":       func(s *Settings) { s.Host = "localhost" },
		"every network":   func(s *Settings) { s.Host = "0.0.0.0" },
		"zone":            func(s *Settings) { s.Host = "fe80::1%en0" },
		"low port":        func(s *Settings) { s.Port = 80 },
		"high port":       func(s *Settings) { s.Port = 70000 },
		"zero file":       func(s *Settings) { s.MaxFileMB = 0 },
		"huge batch":      func(s *Settings) { s.MaxBatchMB = MaxBatchMBLimit + 1 },
		"batch < file":    func(s *Settings) { s.MaxFileMB, s.MaxBatchMB = 100, 50 },
		"relative binary": func(s *Settings) { s.MacOCRBin = "bin/mac-ocr" },
		"other program":   func(s *Settings) { s.MacOCRBin = "/bin/sh" },
		"no workers":      func(s *Settings) { s.Workers = 0 },
		"many workers":    func(s *Settings) { s.Workers = 9 },
		"ftp url":         func(s *Settings) { s.AIBaseURL = "ftp://example.test/v1" },
		"no host url":     func(s *Settings) { s.AIBaseURL = "http:///v1" },
		"login in url":    func(s *Settings) { s.AIBaseURL = "https://user:secret@example.test/v1" },
		"query in url":    func(s *Settings) { s.AIBaseURL = "https://example.test/v1?key=1" },
		"key with space":  func(s *Settings) { s.AIAPIKey = "sk test" },
		"long key":        func(s *Settings) { s.AIAPIKey = strings.Repeat("k", 4097) },
		"short timeout":   func(s *Settings) { s.AITimeoutSeconds = 4 },
		"long timeout":    func(s *Settings) { s.AITimeoutSeconds = 1801 },
	}
	for name, change := range cases {
		s := DefaultSettings()
		change(&s)
		if _, err := s.Validate(); !errors.Is(err, ErrValidation) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestGivenSettingsWhenStoredAndReadBackThenTheyRoundTrip(t *testing.T) {
	in := DefaultSettings()
	in.Host, in.Port, in.Workers, in.AIBaseURL, in.AIAPIKey, in.MacOCRBin = "100.64.0.1", 9292, 4, "https://example.test/v1", "sk-test", "/opt/bin/mac-ocr"
	in.MaxFileMB, in.MaxBatchMB = 300, 400
	got, rejected := SettingsFromValues(in.Values())
	if rejected != nil || !reflect.DeepEqual(got, in) {
		t.Fatalf("got %+v, %v", got, rejected)
	}
	partial, rejected := SettingsFromValues(map[string]string{"workers": "3", "unknown": "x"})
	if rejected != nil || partial.Workers != 3 || partial.Port != 9191 {
		t.Fatalf("partial = %+v, %v", partial, rejected)
	}
}

func TestGivenABadStoredValueWhenReadingThenOnlyThatValueKeepsItsDefault(t *testing.T) {
	got, rejected := SettingsFromValues(map[string]string{"port": "abc", "host": "0.0.0.0", "ai_api_key": "sk-test", "workers": "4",
		"max_file_mb": "300", "max_batch_mb": "250"})
	// A 300 MB file limit cannot fit the stored 250 MB upload limit, so the
	// file limit keeps its default and the upload limit stays.
	want := DefaultSettings()
	want.AIAPIKey, want.Workers, want.MaxBatchMB = "sk-test", 4, 250
	if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(rejected, []string{"host", "port", "max_file_mb"}) {
		t.Fatalf("got %+v, rejected %v", got, rejected)
	}
	if SettingField("max_file_mb") != "maxFileMb" || SettingField("ai_api_key") != "aiApiKey" {
		t.Fatal("field names")
	}
}

func TestGivenChangedSettingsWhenComparingWithTheStartThenOnlyStartFieldsWait(t *testing.T) {
	started := DefaultSettings()
	s := started
	s.Workers, s.AIBaseURL, s.MaxFileMB = 4, "http://localhost:8080/v1", 10
	if got := s.RestartFields(started); len(got) != 0 {
		t.Fatalf("live changes wait: %v", got)
	}
	s.MaxBatchMB = started.MaxBatchMB - 1
	if got := s.RestartFields(started); len(got) != 0 {
		t.Fatalf("a smaller batch waits: %v", got)
	}
	s.Host, s.Port, s.MaxBatchMB = "::1", 9292, started.MaxBatchMB+1
	if got := s.RestartFields(started); !reflect.DeepEqual(got, []string{"host", "port", "maxBatchMb"}) {
		t.Fatalf("restart fields = %v", got)
	}
}
