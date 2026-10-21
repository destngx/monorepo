package fiber

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"apps/health-watcher/adapter/filestore"
	"apps/health-watcher/adapter/password"
	"apps/health-watcher/adapter/qrcode"
	"apps/health-watcher/adapter/sqlite"
	"apps/health-watcher/domain"
	"apps/health-watcher/service"
)

type harness struct {
	t      *testing.T
	app    *fiber.App
	svc    Services
	cookie string
	csrf   string
	tenant string
}

const testPassword = "a long test password"

// newHarness builds the app on a temporary store with one owner, set up
// through the service.
func newHarness(t *testing.T) *harness {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	store, err := sqlite.Open(ctx, filepath.Join(dir, "health.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	files, err := filestore.New(filepath.Join(dir, "files"))
	if err != nil {
		t.Fatal(err)
	}
	deps := service.Deps{Store: store, Files: files, Hasher: password.New(password.Params{MemoryKiB: 64, Passes: 1, Lanes: 1}),
		Clock: time.Now, Location: time.UTC, FileLock: &sync.Mutex{}}
	accounts, err := service.NewAccounts(deps)
	if err != nil {
		t.Fatal(err)
	}
	records := &service.Records{Deps: deps}
	svc := Services{
		Accounts: accounts,
		Tenants:  &service.Tenants{Deps: deps, Accounts: accounts},
		Records:  records,
		Inbox:    &service.Inbox{Deps: deps, Records: records, Limits: service.DefaultLimits},
		Clinical: &service.Clinical{Deps: deps, Catalog: domain.DefaultCatalog},
		Engine: &service.Engine{Deps: deps, Log: slog.New(slog.DiscardHandler), DefaultMacOCRBin: filepath.Join(dir, "no-mac-ocr"),
			LocalAddrs: func() ([]netip.Addr, error) { return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil }},
	}
	svc.Capture = &service.Capture{Deps: deps, QR: qrcode.Coder{}, Listening: svc.Engine.Listening}
	if _, _, err := svc.Engine.Load(ctx, nil); err != nil {
		t.Fatal(err)
	}
	code, err := accounts.PrepareSetup(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := accounts.Setup(ctx, service.SetupInput{Email: "owner@example.test", DisplayName: "Owner",
		Password: testPassword, TenantName: "Test Family", SetupCode: code}); err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &harness{t: t, app: New(svc, Options{Log: log, BodyLimit: 8 << 20}), svc: svc}
}

func (h *harness) do(req *http.Request) (*http.Response, []byte) {
	h.t.Helper()
	if h.cookie != "" {
		req.AddCookie(&http.Cookie{Name: SessionCookie, Value: h.cookie})
	}
	if h.csrf != "" && req.Method != http.MethodGet {
		req.Header.Set(CSRFHeader, h.csrf)
	}
	resp, err := h.app.Test(req, -1)
	if err != nil {
		h.t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return resp, body
}

func (h *harness) json(method, path string, body any) (*http.Response, []byte) {
	h.t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return h.do(req)
}

func (h *harness) login() {
	h.t.Helper()
	resp, body := h.json(http.MethodPost, "/api/auth/login", map[string]string{"email": "owner@example.test", "password": testPassword})
	if resp.StatusCode != http.StatusOK {
		h.t.Fatalf("login = %d %s", resp.StatusCode, body)
	}
	for _, c := range resp.Cookies() {
		if c.Name == SessionCookie {
			h.cookie = c.Value
			if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode {
				h.t.Fatalf("cookie flags: %+v", c)
			}
		}
	}
	var me domain.Me
	if err := json.Unmarshal(body, &me); err != nil {
		h.t.Fatal(err)
	}
	h.csrf, h.tenant = me.CSRFToken, me.Memberships[0].Tenant.ID
}

func multipartBody(t *testing.T, field string, files map[string][]byte, values map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	for name, data := range files {
		part, err := w.CreateFormFile(field, name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write(data)
	}
	for k, v := range values {
		_ = w.WriteField(k, v)
	}
	_ = w.Close()
	return buf, w.FormDataContentType()
}

func wantStatus(t *testing.T, resp *http.Response, body []byte, status int, code string) {
	t.Helper()
	if resp.StatusCode != status {
		t.Fatalf("status = %d, want %d: %s", resp.StatusCode, status, body)
	}
	if code != "" {
		var e ErrorBody
		_ = json.Unmarshal(body, &e)
		if e.Code != code {
			t.Fatalf("code = %q, want %q: %s", e.Code, code, body)
		}
	}
}

func TestGivenNoSessionWhenCallingTenantRoutesThenTheyAreUnauthorized(t *testing.T) {
	h := newHarness(t)
	resp, body := h.json(http.MethodGet, "/api/auth/me", nil)
	wantStatus(t, resp, body, http.StatusUnauthorized, CodeUnauthorized)
	resp, body = h.json(http.MethodGet, "/api/tenants/anything/people", nil)
	wantStatus(t, resp, body, http.StatusUnauthorized, CodeUnauthorized)
}

func TestGivenSetupIsDoneWhenPostingSetupAgainThenItIsForbidden(t *testing.T) {
	h := newHarness(t)
	resp, body := h.json(http.MethodGet, "/api/setup", nil)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"needsSetup":false`) {
		t.Fatalf("GET setup = %d %s", resp.StatusCode, body)
	}
	resp, body = h.json(http.MethodPost, "/api/setup", map[string]string{"email": "x@example.test", "setupCode": "0000-0000-0000"})
	wantStatus(t, resp, body, http.StatusForbidden, CodeForbidden)
}

func TestGivenASessionWhenWritingWithoutTheCSRFTokenThenItIsForbidden(t *testing.T) {
	h := newHarness(t)
	h.login()
	csrf := h.csrf
	h.csrf = ""
	resp, body := h.json(http.MethodPost, "/api/tenants/"+h.tenant+"/people", map[string]string{"displayName": "An"})
	wantStatus(t, resp, body, http.StatusForbidden, CodeForbidden)
	h.csrf = "wrong"
	resp, body = h.json(http.MethodPost, "/api/tenants/"+h.tenant+"/people", map[string]string{"displayName": "An"})
	wantStatus(t, resp, body, http.StatusForbidden, CodeForbidden)
	h.csrf = csrf
	resp, body = h.json(http.MethodPost, "/api/tenants/"+h.tenant+"/people", map[string]string{"displayName": "An"})
	wantStatus(t, resp, body, http.StatusCreated, "")
}

func TestGivenAFormEncodedBodyWhenPostingJSONRoutesThenItIsRejected(t *testing.T) {
	h := newHarness(t)
	h.login()
	req := httptest.NewRequest(http.MethodPost, "/api/tenants/"+h.tenant+"/people", strings.NewReader("displayName=An"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, body := h.do(req)
	wantStatus(t, resp, body, http.StatusBadRequest, CodeValidation)
}

func TestGivenAnotherTenantIdWhenCallingItsRoutesThenItIsNotFound(t *testing.T) {
	h := newHarness(t)
	h.login()
	resp, body := h.json(http.MethodGet, "/api/tenants/not-mine/people", nil)
	wantStatus(t, resp, body, http.StatusNotFound, CodeNotFound)
}

func TestGivenTheShortcutWhenPostingWithAnUploadTokenThenTheFileLandsInTheInbox(t *testing.T) {
	h := newHarness(t)
	h.login()
	resp, body := h.json(http.MethodPost, "/api/tenants/"+h.tenant+"/upload-tokens", map[string]string{"name": "iPhone"})
	wantStatus(t, resp, body, http.StatusCreated, "")
	var tok service.CreatedUploadToken
	_ = json.Unmarshal(body, &tok)

	shortcut := &harness{t: t, app: h.app}
	buf, ct := multipartBody(t, "file", map[string][]byte{"scan.pdf": []byte("%PDF-1.7 synthetic scan")}, map[string]string{"note": "khám răng"})
	req := httptest.NewRequest(http.MethodPost, "/api/inbox", buf)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+tok.Token)
	resp, body = shortcut.do(req)
	wantStatus(t, resp, body, http.StatusAccepted, "")

	buf, ct = multipartBody(t, "file", map[string][]byte{"x.pdf": []byte("%PDF-1.7 other")}, nil)
	req = httptest.NewRequest(http.MethodPost, "/api/inbox", buf)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer hwu_wrong")
	resp, body = shortcut.do(req)
	wantStatus(t, resp, body, http.StatusUnauthorized, CodeUnauthorized)

	resp, body = h.json(http.MethodGet, "/api/tenants/"+h.tenant+"/inbox", nil)
	var items []domain.InboxItem
	_ = json.Unmarshal(body, &items)
	if resp.StatusCode != http.StatusOK || len(items) != 1 || items[0].Source != domain.SourceShortcut || items[0].Note != "khám răng" {
		t.Fatalf("inbox = %d %s", resp.StatusCode, body)
	}
}

func TestGivenAFiledDocumentWhenFetchingItsFileThenItIsServedPrivately(t *testing.T) {
	h := newHarness(t)
	h.login()
	content := []byte("%PDF-1.7 synthetic result")
	buf, ct := multipartBody(t, "files", map[string][]byte{"kết quả.pdf": content}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/tenants/"+h.tenant+"/inbox", buf)
	req.Header.Set("Content-Type", ct)
	resp, body := h.do(req)
	wantStatus(t, resp, body, http.StatusAccepted, "")
	var up service.UploadResponse
	_ = json.Unmarshal(body, &up)

	resp, body = h.json(http.MethodGet, "/api/tenants/"+h.tenant+"/people", nil)
	var people []domain.Person
	_ = json.Unmarshal(body, &people)
	resp, body = h.json(http.MethodPost, "/api/tenants/"+h.tenant+"/inbox/"+up.Items[0].ID+"/file", map[string]any{
		"kind": "lab_result", "visit": map[string]string{"personId": people[0].ID, "date": "2025-02-03", "providerName": "Lab A"},
	})
	wantStatus(t, resp, body, http.StatusOK, "")

	resp, body = h.json(http.MethodGet, "/api/tenants/"+h.tenant+"/files/"+up.Files[0].File.ID, nil)
	wantStatus(t, resp, body, http.StatusOK, "")
	if !bytes.Equal(body, content) {
		t.Fatalf("body = %q", body)
	}
	for header, want := range map[string]string{
		"Content-Type":           "application/pdf",
		"Cache-Control":          "private, no-store",
		"X-Content-Type-Options": "nosniff",
		"Content-Disposition":    "inline; filename*=UTF-8''k%E1%BA%BFt%20qu%E1%BA%A3.pdf",
	} {
		if got := resp.Header.Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestGivenAnHTMLFileWhenUploadingThenItIsRejected(t *testing.T) {
	h := newHarness(t)
	h.login()
	buf, ct := multipartBody(t, "files", map[string][]byte{"fake.pdf": []byte("<html><script>alert(1)</script>")}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/tenants/"+h.tenant+"/inbox", buf)
	req.Header.Set("Content-Type", ct)
	resp, body := h.do(req)
	wantStatus(t, resp, body, http.StatusBadRequest, CodeValidation)
}

func TestGivenLogoutWhenUsingTheOldCookieThenItIsUnauthorized(t *testing.T) {
	h := newHarness(t)
	h.login()
	resp, body := h.json(http.MethodPost, "/api/auth/logout", nil)
	wantStatus(t, resp, body, http.StatusNoContent, "")
	resp, body = h.json(http.MethodGet, "/api/auth/me", nil)
	wantStatus(t, resp, body, http.StatusUnauthorized, CodeUnauthorized)
}

func TestGivenAnUnknownAPIPathWhenSignedInThenItIsNotFound(t *testing.T) {
	h := newHarness(t)
	h.login()
	resp, body := h.json(http.MethodGet, "/api/nope", nil)
	wantStatus(t, resp, body, http.StatusNotFound, CodeNotFound)
}

func TestGivenAFrameworkErrorQuotingTheRequestWhenAnsweringThenOnlyTheStatusTextGoesBack(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))})
	app.Get("/bad", func(c *fiber.Ctx) error {
		return fiber.NewError(http.StatusBadRequest, `invalid header, contents: {"password":"secret-value"}`)
	})
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/bad", nil))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), "secret-value") {
		t.Fatalf("body echoes the request: %s", body)
	}
	wantStatus(t, resp, body, http.StatusBadRequest, CodeValidation)
}
