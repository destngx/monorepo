package fiber

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"apps/health-watcher/service"
)

func TestGivenACaptureLinkWhenThePhoneUploadsThenTheItemComesFromThePhone(t *testing.T) {
	h := newHarness(t)
	h.login()
	resp, body := h.json(http.MethodPost, "/api/tenants/"+h.tenant+"/capture", map[string]string{"origin": "http://100.64.0.7:9191"})
	wantStatus(t, resp, body, http.StatusCreated, "")
	if resp.Header.Get("Cache-Control") != "private, no-store" {
		t.Fatalf("cache = %q", resp.Header.Get("Cache-Control"))
	}
	var link service.CaptureLink
	_ = json.Unmarshal(body, &link)
	_, raw, _ := strings.Cut(link.URL, "#")

	buf, ct := multipartBody(t, "files", map[string][]byte{"page.pdf": []byte("%PDF-1.4\n%%EOF\n")}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/inbox", buf)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+raw)
	phone := &harness{t: t, app: h.app, svc: h.svc}
	resp, body = phone.do(req)
	wantStatus(t, resp, body, http.StatusAccepted, "")

	resp, body = h.json(http.MethodGet, "/api/tenants/"+h.tenant+"/inbox", nil)
	wantStatus(t, resp, body, http.StatusOK, "")
	if !strings.Contains(string(body), `"source":"phone"`) {
		t.Fatalf("inbox = %s", body)
	}
}
