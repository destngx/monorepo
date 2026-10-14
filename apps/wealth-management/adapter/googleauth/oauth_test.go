package googleauth

import (
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/oauth2"
)

var client = domain.GoogleOAuthClient{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://127.0.0.1:8080/api/auth/google/callback"}

func TestGivenAConsentWhenBuildingTheURLThenItAsksForOfflineAccessWithPKCE(t *testing.T) {
	raw := New().AuthCodeURL(client, domain.GoogleAuthSheets.Scopes(), "state-1", "verifier-1")

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	want := map[string]string{
		"client_id":             "id",
		"redirect_uri":          client.RedirectURL,
		"state":                 "state-1",
		"access_type":           "offline",
		"prompt":                "consent",
		"code_challenge_method": "S256",
		"code_challenge":        oauth2.S256ChallengeFromVerifier("verifier-1"),
		"scope":                 "https://www.googleapis.com/auth/spreadsheets",
	}
	for key, value := range want {
		if query.Get(key) != value {
			t.Fatalf("%s = %q, want %q", key, query.Get(key), value)
		}
	}
}

func tokenServer(t *testing.T, body map[string]any) context.Context {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil || r.Form.Get("code_verifier") != "verifier-1" || r.Form.Get("code") != "code-1" {
			http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(server.Close)
	// Route the token request to the fake server.
	transport := &rewriteTransport{target: server.URL}
	return context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Transport: transport})
}

type rewriteTransport struct{ target string }

func (r *rewriteTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	target, _ := url.Parse(r.target)
	request.URL.Scheme, request.URL.Host = target.Scheme, target.Host
	return http.DefaultTransport.RoundTrip(request)
}

func TestGivenACodeWhenExchangingThenTheRefreshTokenIsReturned(t *testing.T) {
	ctx := tokenServer(t, map[string]any{"access_token": "a", "token_type": "Bearer", "refresh_token": "r1", "expires_in": 3600})

	token, err := New().Exchange(ctx, client, "code-1", "verifier-1")

	if err != nil || token != "r1" {
		t.Fatalf("token %q err %v", token, err)
	}
}

func TestGivenNoRefreshTokenWhenExchangingThenItFails(t *testing.T) {
	ctx := tokenServer(t, map[string]any{"access_token": "a", "token_type": "Bearer", "expires_in": 3600})

	if _, err := New().Exchange(ctx, client, "code-1", "verifier-1"); !errors.Is(err, domain.ErrGoogleNoRefreshToken) {
		t.Fatalf("err = %v", err)
	}
}

func TestGivenAWrongVerifierWhenExchangingThenTheCodeIsRejected(t *testing.T) {
	ctx := tokenServer(t, nil)

	_, err := New().Exchange(ctx, client, "code-1", "other")
	var retrieve *oauth2.RetrieveError
	if !errors.As(err, &retrieve) || !errors.Is(err, domain.ErrGoogleCodeRejected) {
		t.Fatalf("err = %v, want a rejected code keeping the RetrieveError", err)
	}
}
