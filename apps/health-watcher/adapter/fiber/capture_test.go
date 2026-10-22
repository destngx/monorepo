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

func TestGivenACaptureLinkMadeOnAnItemWhenThePhoneSendsThenThePhotosJoinTheItem(t *testing.T) {
	h := newHarness(t)
	h.login()
	base := "/api/tenants/" + h.tenant
	buf, ct := multipartBody(t, "files", map[string][]byte{"report.pdf": []byte("%PDF-1.7 synthetic report")}, nil)
	req := httptest.NewRequest(http.MethodPost, base+"/inbox", buf)
	req.Header.Set("Content-Type", ct)
	resp, body := h.do(req)
	wantStatus(t, resp, body, http.StatusAccepted, "")
	var up service.UploadResponse
	_ = json.Unmarshal(body, &up)
	itemID := up.Items[0].ID

	resp, body = h.json(http.MethodPost, base+"/capture", map[string]string{"origin": "http://100.64.0.7:9191", "itemId": itemID})
	wantStatus(t, resp, body, http.StatusCreated, "")
	var link service.CaptureLink
	_ = json.Unmarshal(body, &link)
	_, raw, _ := strings.Cut(link.URL, "#")
	phone := &harness{t: t, app: h.app, svc: h.svc}

	req = httptest.NewRequest(http.MethodGet, "/api/capture", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	resp, body = phone.do(req)
	wantStatus(t, resp, body, http.StatusOK, "")
	if strings.TrimSpace(string(body)) != `{"item":true}` {
		t.Fatalf("target = %s", body)
	}

	buf, ct = multipartBody(t, "files", map[string][]byte{"xray.jpg": []byte("%PDF-1.7 synthetic x-ray")}, nil)
	req = httptest.NewRequest(http.MethodPost, "/api/inbox", buf)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+raw)
	resp, body = phone.do(req)
	wantStatus(t, resp, body, http.StatusAccepted, "")
	var added service.UploadResponse
	_ = json.Unmarshal(body, &added)
	if len(added.Items) != 1 || added.Items[0].ID != itemID || len(added.Items[0].Files) != 2 {
		t.Fatalf("added = %s", body)
	}
	resp, body = h.json(http.MethodGet, base+"/inbox", nil)
	wantStatus(t, resp, body, http.StatusOK, "")
	var items []json.RawMessage
	_ = json.Unmarshal(body, &items)
	if len(items) != 1 {
		t.Fatalf("inbox = %s", body)
	}
}

func TestGivenACaptureLinkWhenThePhoneSendsPhotosAheadThenSendFilesThemByID(t *testing.T) {
	h := newHarness(t)
	h.login()
	resp, body := h.json(http.MethodPost, "/api/tenants/"+h.tenant+"/capture", map[string]string{"origin": "http://100.64.0.7:9191"})
	wantStatus(t, resp, body, http.StatusCreated, "")
	var link service.CaptureLink
	_ = json.Unmarshal(body, &link)
	_, raw, _ := strings.Cut(link.URL, "#")
	phone := &harness{t: t, app: h.app, svc: h.svc}

	buf, ct := multipartBody(t, "files", map[string][]byte{"IMG_0001.jpg": []byte("%PDF-1.7 synthetic photo")}, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/uploads", buf)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+raw)
	resp, body = phone.do(req)
	wantStatus(t, resp, body, http.StatusAccepted, "")
	var staged struct {
		Files []struct {
			File struct {
				ID string `json:"id"`
			} `json:"file"`
			Duplicate bool `json:"duplicate"`
		} `json:"files"`
	}
	_ = json.Unmarshal(body, &staged)
	if len(staged.Files) != 1 || staged.Files[0].File.ID == "" || staged.Files[0].Duplicate {
		t.Fatalf("staged = %s", body)
	}

	buf, ct = multipartBody(t, "files", nil, map[string]string{"fileId": staged.Files[0].File.ID})
	req = httptest.NewRequest(http.MethodPost, "/api/inbox", buf)
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Authorization", "Bearer "+raw)
	resp, body = phone.do(req)
	wantStatus(t, resp, body, http.StatusAccepted, "")
	var sent service.UploadResponse
	_ = json.Unmarshal(body, &sent)
	if len(sent.Items) != 1 || len(sent.Items[0].Files) != 1 || sent.Items[0].Source != "phone" {
		t.Fatalf("sent = %s", body)
	}
}
