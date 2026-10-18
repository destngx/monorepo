package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// fakeAPITokens starts with testToken, like the guard of every handler test.
type fakeAPITokens struct{ token string }

func (f *fakeAPITokens) Token() string { return f.token }

func (f *fakeAPITokens) Status() domain.APITokenStatus {
	return domain.APITokenStatus{Set: f.token != "", Loopback: false}
}

func (f *fakeAPITokens) Set(_ context.Context, update domain.APITokenUpdate) (domain.APITokenStatus, error) {
	token, err := domain.NormalizeAPIToken(update.Token)
	if err != nil {
		return domain.APITokenStatus{}, err
	}
	f.token = token
	return f.Status(), nil
}

func newAPITokenApp(t *testing.T, tokens *fakeAPITokens) *fiber.App {
	t.Helper()
	app := fiber.New()
	RegisterAPITokenRoutes(NewRouteRegistrar(app, "/api", ""), NewAPITokenHandler(tokens, logger.NewTestLogger(t)), NewWriteGuard(tokens, "0.0.0.0"))
	return app
}

func TestGivenATokenWhenAskingItsStatusThenTheTokenIsNotReturned(t *testing.T) {
	app := newAPITokenApp(t, &fakeAPITokens{token: testToken})

	status, body := call(t, app, http.MethodGet, "/api/settings/api-token", "", "Authorization", "")

	var view domain.APITokenStatus
	if status != http.StatusOK || json.Unmarshal([]byte(body), &view) != nil || !view.Set || view.Loopback {
		t.Fatalf("status %d body %s", status, body)
	}
	if strings.Contains(body, testToken) {
		t.Fatalf("the status leaks the token: %s", body)
	}
}

func TestGivenTheCurrentTokenWhenSettingANewOneThenOnlyTheNewOneWrites(t *testing.T) {
	tokens := &fakeAPITokens{token: testToken}
	app := newAPITokenApp(t, tokens)
	next := strings.Repeat("b", 64)

	status, body := call(t, app, http.MethodPost, "/api/settings/api-token", `{"token":"`+next+`"}`)

	if status != http.StatusOK || strings.Contains(body, next) || tokens.token != next {
		t.Fatalf("status %d body %s token %q", status, body, tokens.token)
	}
	if status, _ := call(t, app, http.MethodPost, "/api/settings/api-token", `{"token":"`+strings.Repeat("c", 64)+`"}`); status != http.StatusUnauthorized {
		t.Fatalf("the old token must stop working at once: %d", status)
	}
}

func TestGivenNoCurrentTokenWhenSettingOneThenItIsRefused(t *testing.T) {
	tokens := &fakeAPITokens{token: testToken}
	app := newAPITokenApp(t, tokens)

	status, _ := call(t, app, http.MethodPost, "/api/settings/api-token", `{"token":"`+strings.Repeat("b", 64)+`"}`, "Authorization", "")

	if status != http.StatusUnauthorized || tokens.token != testToken {
		t.Fatalf("status %d token %q", status, tokens.token)
	}
}

func TestGivenAnInvalidTokenWhenSettingThenItIsABadRequest(t *testing.T) {
	tokens := &fakeAPITokens{token: testToken}
	app := newAPITokenApp(t, tokens)

	for _, body := range []string{`{"token":"short"}`, `{"token":""}`, `not json`} {
		if status, _ := call(t, app, http.MethodPost, "/api/settings/api-token", body); status != http.StatusBadRequest {
			t.Fatalf("%s: status %d", body, status)
		}
	}
	if tokens.token != testToken {
		t.Fatal("a refused token must not replace the current one")
	}
}
