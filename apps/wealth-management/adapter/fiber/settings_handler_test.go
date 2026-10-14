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

type fakeSettings struct {
	values    map[string]string
	restarted bool
}

func (f *fakeSettings) View(context.Context) (domain.SettingsView, error) {
	return domain.BuildSettingsView(f.values, "/cfg/settings.db", false), nil
}

func (f *fakeSettings) Update(_ context.Context, update domain.SettingsUpdate) (domain.SettingsView, error) {
	_, merged, err := domain.ApplySettingsUpdate(f.values, update)
	if err != nil {
		return domain.SettingsView{}, err
	}
	f.values = merged
	return domain.BuildSettingsView(merged, "/cfg/settings.db", true), nil
}

func (f *fakeSettings) Restart(context.Context) error {
	f.restarted = true
	return nil
}

func newSettingsApp(t *testing.T, settings *fakeSettings) *fiber.App {
	t.Helper()
	app := fiber.New()
	RegisterSettingsRoutes(NewRouteRegistrar(app, "/api", ""), NewSettingsHandler(settings, logger.NewTestLogger(t)), NewWriteGuard(testToken, "0.0.0.0"))
	return app
}

func TestGivenAStoredSecretWhenListingSettingsThenOnlyItsStateIsReturned(t *testing.T) {
	app := newSettingsApp(t, &fakeSettings{values: map[string]string{"AI_API_KEY": "sk-live-123", "AI_MODEL": "gpt"}})

	status, body := call(t, app, http.MethodGet, "/api/settings", "")

	if status != http.StatusOK || strings.Contains(body, "sk-live-123") {
		t.Fatalf("status %d body leaks the secret: %s", status, body)
	}
	var view domain.SettingsView
	if err := json.Unmarshal([]byte(body), &view); err != nil || view.Path != "/cfg/settings.db" || len(view.Groups) == 0 {
		t.Fatalf("view = %+v err %v", view, err)
	}
}

func TestGivenAnUpdateWhenPatchingThenItIsSavedAndRestartIsRequired(t *testing.T) {
	settings := &fakeSettings{values: map[string]string{}}
	app := newSettingsApp(t, settings)

	status, body := call(t, app, http.MethodPatch, "/api/settings", `{"values":{"AI_MODEL":"gpt-main","EXA_API_KEY":null}}`)

	if status != http.StatusOK || settings.values["AI_MODEL"] != "gpt-main" || !strings.Contains(body, `"restartRequired":true`) {
		t.Fatalf("status %d values %v body %s", status, settings.values, body)
	}
}

func TestGivenAnInvalidUpdateWhenPatchingThenItIsAValidationError(t *testing.T) {
	app := newSettingsApp(t, &fakeSettings{values: map[string]string{}})

	for _, payload := range []string{`{"values":{"HOST":"0.0.0.0"}}`, `{"values":{"AI_BASE_URL":"nope"}}`, `not json`} {
		status, body := call(t, app, http.MethodPatch, "/api/settings", payload)
		if status != http.StatusBadRequest || !strings.Contains(body, CodeValidation) {
			t.Fatalf("%s: status %d body %s", payload, status, body)
		}
	}
}

func TestGivenNoTokenWhenWritingSettingsThenTheGuardRejects(t *testing.T) {
	settings := &fakeSettings{values: map[string]string{}}
	app := newSettingsApp(t, settings)

	for _, route := range [][2]string{{http.MethodPatch, "/api/settings"}, {http.MethodPost, "/api/settings/restart"}} {
		status, _ := call(t, app, route[0], route[1], `{"values":{"AI_MODEL":"x"}}`, "Authorization", "")
		if status != http.StatusUnauthorized {
			t.Fatalf("%s %s: status %d", route[0], route[1], status)
		}
	}
	if settings.restarted || len(settings.values) != 0 {
		t.Fatal("an unauthorised request changed state")
	}
}

func TestGivenARestartRequestThenItIsAcceptedAndTheEngineRestarts(t *testing.T) {
	settings := &fakeSettings{values: map[string]string{}}
	app := newSettingsApp(t, settings)

	status, _ := call(t, app, http.MethodPost, "/api/settings/restart", "")

	if status != http.StatusAccepted || !settings.restarted {
		t.Fatalf("status %d restarted %v", status, settings.restarted)
	}
}
