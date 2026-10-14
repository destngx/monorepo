package service

import (
	"apps/wealth-management/domain"
	"context"
	"errors"
	"net/url"
	"testing"
	"time"
)

type fakeGoogleOAuth struct {
	client   domain.GoogleOAuthClient
	scopes   []string
	verifier string
	code     string
	token    string
	err      error
}

func (f *fakeGoogleOAuth) AuthCodeURL(client domain.GoogleOAuthClient, scopes []string, state, verifier string) string {
	f.client, f.scopes, f.verifier = client, scopes, verifier
	return "https://accounts.example/auth?state=" + url.QueryEscape(state)
}

func (f *fakeGoogleOAuth) Exchange(_ context.Context, client domain.GoogleOAuthClient, code, verifier string) (string, error) {
	if verifier != f.verifier {
		return "", errors.New("verifier mismatch")
	}
	f.client, f.code = client, code
	return f.token, f.err
}

type googleAuthFixture struct {
	store *memorySettingsStore
	oauth *fakeGoogleOAuth
	svc   *googleAuthService
	now   time.Time
}

func newGoogleAuthFixture(values map[string]string) *googleAuthFixture {
	f := &googleAuthFixture{
		store: &memorySettingsStore{values: values},
		oauth: &fakeGoogleOAuth{token: "1//refresh"},
		now:   time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC),
	}
	f.svc = newGoogleAuthService(f.store, NewSettingsService(f.store, func() {}), f.oauth, "http://127.0.0.1:8080/api/auth/google/callback", func() time.Time { return f.now })
	return f
}

func configuredClient() map[string]string {
	return map[string]string{"GOOGLE_CLIENT_ID": "id.apps", "GOOGLE_CLIENT_SECRET": "secret"}
}

func stateOf(t *testing.T, start domain.GoogleAuthStart) string {
	t.Helper()
	parsed, err := url.Parse(start.AuthURL)
	if err != nil {
		t.Fatal(err)
	}
	return parsed.Query().Get("state")
}

func TestGivenAConfiguredClientWhenConsentCompletesThenTheRefreshTokenIsSavedAndARestartIsRequired(t *testing.T) {
	f := newGoogleAuthFixture(configuredClient())
	ctx := context.Background()

	start, err := f.svc.Start(ctx, domain.GoogleAuthSheets, "")
	if err != nil {
		t.Fatal(err)
	}
	if start.RedirectURL != "http://127.0.0.1:8080/api/auth/google/callback" || f.oauth.client.RedirectURL != start.RedirectURL {
		t.Fatalf("redirect = %q, client got %q", start.RedirectURL, f.oauth.client.RedirectURL)
	}
	if len(f.oauth.scopes) != 1 || len(f.oauth.verifier) < 43 {
		t.Fatalf("scopes = %v, verifier length %d", f.oauth.scopes, len(f.oauth.verifier))
	}

	result, err := f.svc.Complete(ctx, stateOf(t, start), "4/code")
	if err != nil {
		t.Fatal(err)
	}
	if f.oauth.code != "4/code" {
		t.Fatalf("exchanged code = %q", f.oauth.code)
	}
	if f.store.values["GOOGLE_REFRESH_TOKEN"] != "1//refresh" {
		t.Fatalf("stored = %v", f.store.values)
	}
	want := domain.GoogleAuthResult{Purpose: domain.GoogleAuthSheets, SettingKey: "GOOGLE_REFRESH_TOKEN", RestartRequired: true}
	if result != want {
		t.Fatalf("result = %+v, want %+v", result, want)
	}
}

func TestGivenTheTestPurposeWhenConsentCompletesThenTheTestTokenGetsDriveScope(t *testing.T) {
	f := newGoogleAuthFixture(configuredClient())
	ctx := context.Background()

	start, err := f.svc.Start(ctx, domain.GoogleAuthTest, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.oauth.scopes) != 2 {
		t.Fatalf("scopes = %v", f.oauth.scopes)
	}
	if _, err := f.svc.Complete(ctx, stateOf(t, start), "4/code"); err != nil {
		t.Fatal(err)
	}
	if f.store.values["GOOGLE_TEST_REFRESH_TOKEN"] != "1//refresh" || f.store.values["GOOGLE_REFRESH_TOKEN"] != "" {
		t.Fatalf("stored = %v", f.store.values)
	}
}

func TestGivenAConfiguredRedirectWhenStartingThenItOverridesTheEngineCallback(t *testing.T) {
	values := configuredClient()
	values["GOOGLE_REDIRECT_URL"] = "https://wm.example/api/auth/google/callback"
	f := newGoogleAuthFixture(values)

	start, err := f.svc.Start(context.Background(), domain.GoogleAuthSheets, "")
	if err != nil {
		t.Fatal(err)
	}
	if start.RedirectURL != values["GOOGLE_REDIRECT_URL"] {
		t.Fatalf("redirect = %q", start.RedirectURL)
	}
}

func TestGivenTheUIOnALoopbackOriginWhenStartingThenGoogleReturnsThere(t *testing.T) {
	f := newGoogleAuthFixture(configuredClient())
	ctx := context.Background()

	start, err := f.svc.Start(ctx, domain.GoogleAuthSheets, "http://127.0.0.1:5173")
	if err != nil {
		t.Fatal(err)
	}
	if start.RedirectURL != "http://127.0.0.1:5173/api/auth/google/callback" || f.oauth.client.RedirectURL != start.RedirectURL {
		t.Fatalf("redirect = %q, client got %q", start.RedirectURL, f.oauth.client.RedirectURL)
	}
	if _, err := f.svc.Complete(ctx, stateOf(t, start), "4/code"); err != nil || f.oauth.client.RedirectURL != start.RedirectURL {
		t.Fatalf("exchange used %q, err %v", f.oauth.client.RedirectURL, err)
	}

	start, err = f.svc.Start(ctx, domain.GoogleAuthSheets, "http://ezmacmini:5173")
	if err != nil || start.RedirectURL != "http://127.0.0.1:8080/api/auth/google/callback" {
		t.Fatalf("remote origin: redirect = %q, err %v", start.RedirectURL, err)
	}
}

func TestGivenAConfiguredRedirectAndALoopbackOriginWhenStartingThenTheSettingWins(t *testing.T) {
	values := configuredClient()
	values["GOOGLE_REDIRECT_URL"] = "http://127.0.0.1:3000"
	f := newGoogleAuthFixture(values)

	start, err := f.svc.Start(context.Background(), domain.GoogleAuthSheets, "http://127.0.0.1:5173")
	if err != nil || start.RedirectURL != "http://127.0.0.1:3000" {
		t.Fatalf("redirect = %q, err %v", start.RedirectURL, err)
	}
}

func TestGivenNoClientWhenStartingThenItAsksForTheClientFirst(t *testing.T) {
	f := newGoogleAuthFixture(map[string]string{"GOOGLE_CLIENT_ID": "id.apps"})

	_, err := f.svc.Start(context.Background(), domain.GoogleAuthSheets, "")
	if !errors.Is(err, domain.ErrGoogleClientMissing) || !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v", err)
	}
}

func TestGivenACompletedConsentWhenTheCallbackIsReplayedThenItIsRejected(t *testing.T) {
	f := newGoogleAuthFixture(configuredClient())
	ctx := context.Background()
	start, err := f.svc.Start(ctx, domain.GoogleAuthSheets, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Complete(ctx, stateOf(t, start), "4/code"); err != nil {
		t.Fatal(err)
	}

	if _, err := f.svc.Complete(ctx, stateOf(t, start), "4/code"); !errors.Is(err, domain.ErrGoogleAuthState) {
		t.Fatalf("replay err = %v", err)
	}
}

func TestGivenAnUnknownOrExpiredStateWhenCompletingThenNothingIsSaved(t *testing.T) {
	f := newGoogleAuthFixture(configuredClient())
	ctx := context.Background()
	start, err := f.svc.Start(ctx, domain.GoogleAuthSheets, "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.svc.Complete(ctx, "forged", "4/code"); !errors.Is(err, domain.ErrGoogleAuthState) {
		t.Fatalf("forged err = %v", err)
	}
	f.now = f.now.Add(googleAuthTTL)
	if _, err := f.svc.Complete(ctx, stateOf(t, start), "4/code"); !errors.Is(err, domain.ErrGoogleAuthState) {
		t.Fatalf("expired err = %v", err)
	}
	if f.store.saves != 0 {
		t.Fatalf("saves = %d", f.store.saves)
	}
}

func TestGivenARejectedCodeWhenCompletingThenTheErrorIsReturnedAndNothingIsSaved(t *testing.T) {
	f := newGoogleAuthFixture(configuredClient())
	f.oauth.err = domain.ErrGoogleCodeRejected
	ctx := context.Background()
	start, err := f.svc.Start(ctx, domain.GoogleAuthSheets, "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.svc.Complete(ctx, stateOf(t, start), "4/bad"); !errors.Is(err, domain.ErrGoogleCodeRejected) {
		t.Fatalf("err = %v", err)
	}
	if f.store.saves != 0 {
		t.Fatalf("saves = %d", f.store.saves)
	}
}

func TestGivenManyAbandonedConsentsWhenStartingThenTheyAreCappedUntilTheyExpire(t *testing.T) {
	f := newGoogleAuthFixture(configuredClient())
	ctx := context.Background()
	for range maxPendingGoogleAuth {
		if _, err := f.svc.Start(ctx, domain.GoogleAuthSheets, ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.svc.Start(ctx, domain.GoogleAuthSheets, ""); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("over cap err = %v", err)
	}

	f.now = f.now.Add(googleAuthTTL)
	if _, err := f.svc.Start(ctx, domain.GoogleAuthSheets, ""); err != nil {
		t.Fatalf("after expiry err = %v", err)
	}
}
