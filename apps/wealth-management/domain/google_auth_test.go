package domain

import (
	"errors"
	"slices"
	"testing"
)

func TestGivenAPurposeWhenParsingThenScopesAndKeyFollow(t *testing.T) {
	sheets, err := ParseGoogleAuthPurpose("")
	if err != nil || sheets != GoogleAuthSheets || sheets.SettingKey() != "GOOGLE_REFRESH_TOKEN" || len(sheets.Scopes()) != 1 {
		t.Fatalf("default purpose = %q %v %v", sheets, sheets.Scopes(), err)
	}
	if !slices.Equal(sheets.Scopes(), []string{googleSheetsScope}) {
		t.Fatalf("the sheets token must not ask for more than spreadsheets: %v", sheets.Scopes())
	}
	for _, retired := range []string{"test", "drive"} {
		if _, err := ParseGoogleAuthPurpose(retired); !errors.Is(err, ErrValidation) {
			t.Fatalf("purpose %q err = %v", retired, err)
		}
	}
}

func TestGivenARedirectedAddressWhenParsingThenStateAndCodeAreRead(t *testing.T) {
	state, code, err := ParseGoogleRedirect(" http://127.0.0.1:8080/api/auth/google/callback?state=s1&code=c1&scope=x ")
	if err != nil || state != "s1" || code != "c1" {
		t.Fatalf("state %q code %q err %v", state, code, err)
	}
	for _, raw := range []string{"", "not a url", "http://127.0.0.1:8080/?code=c1", "http://127.0.0.1:8080/?error=access_denied&state=s1"} {
		if _, _, err := ParseGoogleRedirect(raw); !errors.Is(err, ErrValidation) {
			t.Fatalf("%q: err %v, want ErrValidation", raw, err)
		}
	}
}

func TestGivenABrowserOriginWhenDerivingTheGoogleCallbackThenOnlyLoopbackOriginsQualify(t *testing.T) {
	cases := map[string]string{
		"http://127.0.0.1:5173":   "http://127.0.0.1:5173/api/auth/google/callback",
		"http://localhost:5173/":  "http://localhost:5173/api/auth/google/callback",
		"http://[::1]:8080":       "http://[::1]:8080/api/auth/google/callback",
		"http://ezmacmini:5173":   "",
		"https://evil.example":    "",
		"http://127.0.0.1:5173/x": "",
		"null":                    "",
		"":                        "",
		"ftp://127.0.0.1":         "",
	}
	for origin, want := range cases {
		got, ok := GoogleCallbackForOrigin(origin)
		if got != want || ok != (want != "") {
			t.Errorf("GoogleCallbackForOrigin(%q) = %q, %v; want %q", origin, got, ok, want)
		}
	}
}
