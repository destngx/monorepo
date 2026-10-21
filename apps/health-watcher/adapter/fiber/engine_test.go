package fiber

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"apps/health-watcher/domain"
	"apps/health-watcher/service"
)

// member signs up through an invite to the owner's tenant and returns a
// harness signed in as them.
func (h *harness) member(role string) (*harness, string) {
	h.t.Helper()
	sc, err := h.svc.Accounts.ScopeFor(h.t.Context(), h.ownerID(), h.tenant)
	if err != nil {
		h.t.Fatal(err)
	}
	inv, err := h.svc.Tenants.CreateInvite(h.t.Context(), sc, role)
	if err != nil {
		h.t.Fatal(err)
	}
	m := &harness{t: h.t, app: h.app, svc: h.svc}
	resp, body := m.json(http.MethodPost, "/api/invites/"+inv.Token+"/signup",
		map[string]string{"email": "member@example.test", "displayName": "Member", "password": testPassword})
	wantStatus(h.t, resp, body, http.StatusCreated, "")
	for _, c := range resp.Cookies() {
		if c.Name == SessionCookie {
			m.cookie = c.Value
		}
	}
	var me domain.Me
	_ = json.Unmarshal(body, &me)
	m.csrf, m.tenant = me.CSRFToken, h.tenant
	return m, me.User.ID
}

func (h *harness) ownerID() string {
	h.t.Helper()
	resp, body := h.json(http.MethodGet, "/api/auth/me", nil)
	wantStatus(h.t, resp, body, http.StatusOK, "")
	var me domain.Me
	_ = json.Unmarshal(body, &me)
	return me.User.ID
}

func TestGivenTheAdminWhenUsingEngineSettingsThenTheyChangeAndTheKeyNeverComesBack(t *testing.T) {
	h := newHarness(t)
	h.login()

	resp, body := h.json(http.MethodGet, "/api/auth/me", nil)
	if !strings.Contains(string(body), `"admin":true`) {
		t.Fatalf("me = %s", body)
	}
	resp, body = h.json(http.MethodGet, "/api/engine/settings", nil)
	wantStatus(t, resp, body, http.StatusOK, "")
	if resp.Header.Get("Cache-Control") != "private, no-store" {
		t.Fatalf("cache = %q", resp.Header.Get("Cache-Control"))
	}
	var v service.SettingsView
	_ = json.Unmarshal(body, &v)
	if v.Settings.Port != 9191 || v.AIModel != "gpt-6-luna" || v.Listening != "127.0.0.1:9191" {
		t.Fatalf("view = %s", body)
	}

	in := map[string]any{"host": "127.0.0.1", "port": 9191, "maxFileMb": 50, "maxBatchMb": 200, "macOcrBin": "", "workers": 3,
		"aiBaseUrl": "http://localhost:8080/v1", "aiTimeoutSeconds": 120, "aiApiKey": "sk-test-secret"}
	resp, body = h.json(http.MethodPut, "/api/engine/settings", in)
	wantStatus(t, resp, body, http.StatusOK, "")
	if strings.Contains(string(body), "sk-test-secret") || !strings.Contains(string(body), `"aiApiKeySet":true`) ||
		!strings.Contains(string(body), `"workers":3`) {
		t.Fatalf("saved = %s", body)
	}

	in["workers"] = 0
	resp, body = h.json(http.MethodPut, "/api/engine/settings", in)
	wantStatus(t, resp, body, http.StatusBadRequest, CodeValidation)

	resp, body = h.json(http.MethodGet, "/api/engine/users", nil)
	wantStatus(t, resp, body, http.StatusOK, "")
	if !strings.Contains(string(body), "owner@example.test") {
		t.Fatalf("users = %s", body)
	}
}

func TestGivenAMemberWhoIsNotAdminWhenUsingEngineSettingsThenItIsForbiddenUntilGranted(t *testing.T) {
	h := newHarness(t)
	h.login()
	m, memberID := h.member("owner")

	for _, call := range []struct{ method, path string }{
		{http.MethodGet, "/api/engine/settings"}, {http.MethodPut, "/api/engine/settings"},
		{http.MethodPost, "/api/engine/settings/check-ai"}, {http.MethodGet, "/api/engine/users"},
	} {
		resp, body := m.json(call.method, call.path, map[string]any{})
		wantStatus(t, resp, body, http.StatusForbidden, CodeForbidden)
	}

	resp, body := h.json(http.MethodPut, "/api/engine/users/"+memberID+"/admin", map[string]bool{"admin": true})
	wantStatus(t, resp, body, http.StatusOK, "")
	resp, body = m.json(http.MethodGet, "/api/engine/settings", nil)
	wantStatus(t, resp, body, http.StatusOK, "")

	resp, body = m.json(http.MethodPut, "/api/engine/users/"+memberID+"/admin", map[string]bool{"admin": false})
	wantStatus(t, resp, body, http.StatusOK, "")
	resp, body = h.json(http.MethodPut, "/api/engine/users/"+h.ownerID()+"/admin", map[string]bool{"admin": false})
	wantStatus(t, resp, body, http.StatusConflict, CodeConflict)
}

func TestGivenNoAIAddressWhenCheckingThenItExplainsWhy(t *testing.T) {
	h := newHarness(t)
	h.login()
	resp, body := h.json(http.MethodPost, "/api/engine/settings/check-ai", nil)
	wantStatus(t, resp, body, http.StatusConflict, CodeConflict)
}
