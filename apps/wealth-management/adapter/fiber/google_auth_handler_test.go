package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

type fakeGoogleAuth struct {
	purpose     domain.GoogleAuthPurpose
	origin      string
	state, code string
	err         error
}

func (f *fakeGoogleAuth) Start(_ context.Context, purpose domain.GoogleAuthPurpose, origin string) (domain.GoogleAuthStart, error) {
	f.purpose, f.origin = purpose, origin
	if f.err != nil {
		return domain.GoogleAuthStart{}, f.err
	}
	return domain.GoogleAuthStart{AuthURL: "https://accounts.example/auth", RedirectURL: "http://127.0.0.1:8080" + domain.GoogleAuthCallbackPath, Purpose: purpose}, nil
}

func (f *fakeGoogleAuth) Complete(_ context.Context, state, code string) (domain.GoogleAuthResult, error) {
	f.state, f.code = state, code
	if f.err != nil {
		return domain.GoogleAuthResult{}, f.err
	}
	return domain.GoogleAuthResult{Purpose: domain.GoogleAuthSheets, SettingKey: "GOOGLE_REFRESH_TOKEN", RestartRequired: true}, nil
}

func newGoogleAuthApp(t *testing.T, auth *fakeGoogleAuth) *fiber.App {
	t.Helper()
	app := fiber.New()
	RegisterGoogleAuthRoutes(NewRouteRegistrar(app, "/api", ""), NewGoogleAuthHandler(auth, logger.NewTestLogger(t)), NewWriteGuard(testToken, "0.0.0.0"))
	return app
}

func TestGivenAPurposeWhenStartingThenTheConsentAddressIsReturned(t *testing.T) {
	auth := &fakeGoogleAuth{}
	app := newGoogleAuthApp(t, auth)

	status, body := call(t, app, http.MethodPost, "/api/auth/google/start", `{"purpose":"sheets"}`, "Origin", "http://127.0.0.1:5173")

	if auth.origin != "http://127.0.0.1:5173" {
		t.Fatalf("origin = %q", auth.origin)
	}
	if status != http.StatusOK || auth.purpose != domain.GoogleAuthSheets || !strings.Contains(body, `"authUrl":"https://accounts.example/auth"`) {
		t.Fatalf("status %d purpose %q body %s", status, auth.purpose, body)
	}
	auth.purpose = ""
	if status, _ := call(t, app, http.MethodPost, "/api/auth/google/start", ""); status != http.StatusOK || auth.purpose != domain.GoogleAuthSheets {
		t.Fatalf("default purpose: status %d purpose %q", status, auth.purpose)
	}
}

func TestGivenAnUnknownPurposeOrMissingClientWhenStartingThenItIsAValidationError(t *testing.T) {
	app := newGoogleAuthApp(t, &fakeGoogleAuth{})
	if status, body := call(t, app, http.MethodPost, "/api/auth/google/start", `{"purpose":"test"}`); status != http.StatusBadRequest || !strings.Contains(body, CodeValidation) {
		t.Fatalf("status %d body %s", status, body)
	}

	app = newGoogleAuthApp(t, &fakeGoogleAuth{err: domain.ErrGoogleClientMissing})
	if status, body := call(t, app, http.MethodPost, "/api/auth/google/start", `{}`); status != http.StatusBadRequest || !strings.Contains(body, "GOOGLE_CLIENT_ID") {
		t.Fatalf("status %d body %s", status, body)
	}
}

func TestGivenNoTokenWhenStartingRemotelyThenItIsRefused(t *testing.T) {
	app := newGoogleAuthApp(t, &fakeGoogleAuth{})
	request := httptest.NewRequest(http.MethodPost, "/api/auth/google/start", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", response.StatusCode)
	}
}

// callback follows Google's redirect and returns where the engine sends the
// browser next.
func callback(t *testing.T, app *fiber.App, target string) (int, string) {
	t.Helper()
	response, err := app.Test(httptest.NewRequest(http.MethodGet, target, nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	return response.StatusCode, response.Header.Get(fiber.HeaderLocation)
}

func TestGivenGoogleRedirectsBackWhenTheCallbackRunsThenTheTokenIsSavedAndTheBrowserReturnsToSettings(t *testing.T) {
	auth := &fakeGoogleAuth{}
	app := newGoogleAuthApp(t, auth)

	status, location := callback(t, app, "/api/auth/google/callback?state=s1&code=4%2Fabc&scope=x")

	if auth.state != "s1" || auth.code != "4/abc" {
		t.Fatalf("completed with state %q code %q", auth.state, auth.code)
	}
	if status != http.StatusSeeOther || location != "/ui/settings?google=connected&purpose=sheets" {
		t.Fatalf("status %d location %q", status, location)
	}
}

func TestGivenAFailedSignInWhenTheCallbackRunsThenSettingsGetsAFixedReason(t *testing.T) {
	cases := []struct {
		name   string
		target string
		err    error
		reason string
	}{
		{"cancelled", "/api/auth/google/callback?error=access_denied&state=s1", nil, "denied"},
		{"crafted text", "/api/auth/google/callback?error=%3Cscript%3E&state=s1", nil, "denied"},
		{"no code", "/api/auth/google/callback", nil, "invalid"},
		{"expired", "/api/auth/google/callback?state=old&code=c", domain.ErrGoogleAuthState, "expired"},
		{"rejected", "/api/auth/google/callback?state=s&code=c", domain.ErrGoogleCodeRejected, "rejected"},
		{"no refresh token", "/api/auth/google/callback?state=s&code=c", domain.ErrGoogleNoRefreshToken, "no_refresh_token"},
		{"store down", "/api/auth/google/callback?state=s&code=c", errors.New("database is locked"), "failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newGoogleAuthApp(t, &fakeGoogleAuth{err: tc.err})

			status, location := callback(t, app, tc.target)

			if status != http.StatusSeeOther || location != "/ui/settings?google=error&reason="+tc.reason {
				t.Fatalf("status %d location %q", status, location)
			}
		})
	}
}

func TestGivenAPastedAddressWhenExchangingThenItCompletesTheSignIn(t *testing.T) {
	auth := &fakeGoogleAuth{}
	app := newGoogleAuthApp(t, auth)

	status, body := call(t, app, http.MethodPost, "/api/auth/google/exchange", `{"redirectedUrl":"http://127.0.0.1:8080/api/auth/google/callback?state=s2&code=4%2Fxyz"}`)

	if status != http.StatusOK || auth.state != "s2" || auth.code != "4/xyz" || !strings.Contains(body, `"restartRequired":true`) {
		t.Fatalf("status %d state %q code %q body %s", status, auth.state, auth.code, body)
	}
	if strings.Contains(body, "4/xyz") {
		t.Fatalf("body echoes the code: %s", body)
	}
}

func TestGivenABadPasteOrUpstreamFailureWhenExchangingThenTheErrorIsClassified(t *testing.T) {
	app := newGoogleAuthApp(t, &fakeGoogleAuth{})
	if status, body := call(t, app, http.MethodPost, "/api/auth/google/exchange", `{"redirectedUrl":"not a url"}`); status != http.StatusBadRequest || !strings.Contains(body, CodeValidation) {
		t.Fatalf("status %d body %s", status, body)
	}

	app = newGoogleAuthApp(t, &fakeGoogleAuth{err: errors.New("dial tcp: connection refused")})
	if status, body := call(t, app, http.MethodPost, "/api/auth/google/exchange", `{"redirectedUrl":"http://x/cb?state=s&code=c"}`); status != http.StatusBadGateway || !strings.Contains(body, CodeUpstream) {
		t.Fatalf("status %d body %s", status, body)
	}
}
