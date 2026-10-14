package domain

import (
	"fmt"
	"net/url"
	"strings"
)

// GoogleAuthPurpose picks which refresh token a Google consent mints. The
// only one is the sheets token, which reads and writes sheets and nothing
// else; the live e2e tests use a local workbook and need no token (WM-TEST-3).
type GoogleAuthPurpose string

const GoogleAuthSheets GoogleAuthPurpose = "sheets"

const googleSheetsScope = "https://www.googleapis.com/auth/spreadsheets"

// GoogleAuthCallbackPath is the engine route Google redirects to.
const GoogleAuthCallbackPath = "/api/auth/google/callback"

// ErrGoogleAuthState is an unknown, used or expired consent: start again.
var ErrGoogleAuthState = fmt.Errorf("%w: this Google sign-in is unknown or expired; start it again from Settings", ErrValidation)

// ErrGoogleClientMissing means the OAuth client is not configured yet.
var ErrGoogleClientMissing = fmt.Errorf("%w: set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET in Settings and save them first", ErrValidation)

func ParseGoogleAuthPurpose(raw string) (GoogleAuthPurpose, error) {
	switch purpose := GoogleAuthPurpose(strings.TrimSpace(raw)); purpose {
	case "", GoogleAuthSheets:
		return GoogleAuthSheets, nil
	default:
		return "", fmt.Errorf("%w: purpose must be %q", ErrValidation, GoogleAuthSheets)
	}
}

func (p GoogleAuthPurpose) Scopes() []string {
	return []string{googleSheetsScope}
}

// SettingKey is where the minted refresh token is saved.
func (p GoogleAuthPurpose) SettingKey() string {
	return "GOOGLE_REFRESH_TOKEN"
}

// GoogleOAuthClient is the OAuth client a consent runs against.
type GoogleOAuthClient struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// ErrGoogleCodeRejected is a code Google refused to exchange (expired, used
// or issued for another client): start again.
var ErrGoogleCodeRejected = fmt.Errorf("%w: Google rejected the sign-in code; start again from Settings", ErrValidation)

// ErrGoogleNoRefreshToken is a consent that returned no refresh token,
// which happens when Google keeps an earlier grant.
var ErrGoogleNoRefreshToken = fmt.Errorf("%w: Google returned no refresh token; remove the app's access in your Google account and connect again", ErrValidation)

// GoogleAuthStart is what the UI opens to ask for consent.
type GoogleAuthStart struct {
	AuthURL     string            `json:"authUrl"`
	RedirectURL string            `json:"redirectUrl"`
	Purpose     GoogleAuthPurpose `json:"purpose"`
}

// GoogleAuthResult reports a saved token; never the token itself.
type GoogleAuthResult struct {
	Purpose         GoogleAuthPurpose `json:"purpose"`
	SettingKey      string            `json:"settingKey"`
	RestartRequired bool              `json:"restartRequired"`
}

// ParseGoogleRedirect reads the state and code from the address the browser
// landed on after consent (the paste fallback when the callback is not
// reachable from that browser). A consent Google refused carries an error.
func ParseGoogleRedirect(raw string) (state, code string, err error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.RawQuery == "" {
		return "", "", fmt.Errorf("%w: paste the full address the browser landed on after Google sign-in", ErrValidation)
	}
	query := parsed.Query()
	if denied := query.Get("error"); denied != "" {
		return "", "", GoogleConsentError(denied)
	}
	state, code = query.Get("state"), query.Get("code")
	if state == "" || code == "" {
		return "", "", fmt.Errorf("%w: the address has no code and state; paste the full address after Google sign-in", ErrValidation)
	}
	return state, code, nil
}

// GoogleConsentError is a consent Google reports as failed (e.g.
// access_denied when the user cancels).
func GoogleConsentError(reason string) error {
	return fmt.Errorf("%w (%s)", ErrGoogleConsentDenied, reason)
}

// ErrGoogleConsentDenied is a consent the user cancelled or Google refused.
var ErrGoogleConsentDenied = fmt.Errorf("%w: Google sign-in did not complete", ErrValidation)

// GoogleCallbackForOrigin is the callback on the origin the UI was opened
// from (e.g. the Vite dev server, which forwards /api to the engine), so
// Google returns the browser where it started. Only a loopback origin
// qualifies: Google accepts plain-http redirects on loopback only, and any
// other host could not be the engine's own address anyway.
func GoogleCallbackForOrigin(origin string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil ||
		(parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}
	switch parsed.Hostname() {
	case "127.0.0.1", "::1", "localhost":
		return parsed.Scheme + "://" + parsed.Host + GoogleAuthCallbackPath, true
	default:
		return "", false
	}
}
