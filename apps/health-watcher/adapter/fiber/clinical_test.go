package fiber

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"apps/health-watcher/domain"
	"apps/health-watcher/service"
)

// fileOne uploads one synthetic PDF, for a backlog person when personID is
// set, and files it under a new visit.
func (h *harness) fileOne(personID string) (service.UploadResponse, service.Filed) {
	h.t.Helper()
	values := map[string]string{}
	if personID != "" {
		values["personId"] = personID
	}
	buf, ct := multipartBody(h.t, "files", map[string][]byte{"result.pdf": []byte("%PDF-1.7 synthetic " + personID)}, values)
	req := httptest.NewRequest(http.MethodPost, "/api/tenants/"+h.tenant+"/inbox", buf)
	req.Header.Set("Content-Type", ct)
	resp, body := h.do(req)
	wantStatus(h.t, resp, body, http.StatusAccepted, "")
	var up service.UploadResponse
	_ = json.Unmarshal(body, &up)
	resp, body = h.json(http.MethodGet, "/api/tenants/"+h.tenant+"/inbox/"+up.Items[0].ID, nil)
	wantStatus(h.t, resp, body, http.StatusOK, "")
	var detail service.ItemDetail
	_ = json.Unmarshal(body, &detail)
	if detail.Item.PersonID != personID || detail.AIEnabled || detail.Item.OCR == nil || detail.Item.OCR.Total != 1 {
		h.t.Fatalf("detail = %s", body)
	}
	resp, body = h.json(http.MethodPost, "/api/tenants/"+h.tenant+"/inbox/"+up.Items[0].ID+"/file", map[string]any{
		"kind": "lab_result", "visit": map[string]string{"personId": personID, "date": "2025-02-03", "providerName": "Lab A"},
	})
	wantStatus(h.t, resp, body, http.StatusOK, "")
	var filed service.Filed
	_ = json.Unmarshal(body, &filed)
	return up, filed
}

func (h *harness) firstPerson() string {
	h.t.Helper()
	_, body := h.json(http.MethodGet, "/api/tenants/"+h.tenant+"/people", nil)
	var people []domain.Person
	_ = json.Unmarshal(body, &people)
	return people[0].ID
}

func TestGivenAFiledDocumentWhenUsingTheViewerRoutesThenRecordsCanBeAddedEditedAndAudited(t *testing.T) {
	h := newHarness(t)
	h.login()
	_, filed := h.fileOne(h.firstPerson())
	base := "/api/tenants/" + h.tenant

	resp, body := h.json(http.MethodPost, base+"/documents/"+filed.Document.ID+"/observations",
		map[string]string{"analyte": "Glucose", "value": "5.1", "unit": "mmol/L"})
	wantStatus(t, resp, body, http.StatusCreated, "")
	var o domain.Observation
	_ = json.Unmarshal(body, &o)
	if o.MetricCode != "glucose" || o.Status != domain.StatusApproved {
		t.Fatalf("observation = %s", body)
	}
	resp, body = h.json(http.MethodPatch, base+"/observations/"+o.ID, map[string]string{"analyte": "Glucose", "value": "5.3", "unit": "mmol/L"})
	wantStatus(t, resp, body, http.StatusOK, "")
	resp, body = h.json(http.MethodGet, base+"/observations/"+o.ID+"/history", nil)
	wantStatus(t, resp, body, http.StatusOK, "")
	if !strings.Contains(string(body), `\"value\":\"5.1\"`) {
		t.Fatalf("history = %s", body)
	}
	resp, body = h.json(http.MethodPut, base+"/observations/"+o.ID+"/metric", map[string]string{"code": "nope"})
	wantStatus(t, resp, body, http.StatusBadRequest, CodeValidation)

	resp, body = h.json(http.MethodPost, base+"/documents/"+filed.Document.ID+"/findings",
		map[string]string{"kind": "diagnosis", "text": "Viêm họng", "icd10": "J02.9"})
	wantStatus(t, resp, body, http.StatusCreated, "")
	resp, body = h.json(http.MethodGet, base+"/search?q=viem+hong", nil)
	wantStatus(t, resp, body, http.StatusOK, "")
	var res service.SearchResult
	_ = json.Unmarshal(body, &res)
	if len(res.Hits) != 1 || res.Hits[0].Kind != domain.SearchFinding || res.Hits[0].DocumentID != filed.Document.ID {
		t.Fatalf("search = %s", body)
	}

	resp, body = h.json(http.MethodPatch, base+"/documents/"+filed.Document.ID, map[string]string{"kind": "prescription", "title": "Đơn"})
	wantStatus(t, resp, body, http.StatusOK, "")
	resp, body = h.json(http.MethodGet, base+"/documents/"+filed.Document.ID, nil)
	var doc service.DocumentDetail
	_ = json.Unmarshal(body, &doc)
	if doc.Document.Kind != "prescription" || len(doc.Observations) != 1 || len(doc.Findings) != 1 {
		t.Fatalf("document = %s", body)
	}
	resp, body = h.json(http.MethodDelete, base+"/observations/"+o.ID, nil)
	wantStatus(t, resp, body, http.StatusNoContent, "")
}

func TestGivenNoBackgroundReaderWhenUsingReviewRoutesThenTheyExplainWhy(t *testing.T) {
	h := newHarness(t)
	h.login()
	base := "/api/tenants/" + h.tenant
	buf, ct := multipartBody(t, "files", map[string][]byte{"a.pdf": []byte("%PDF-1.7 synthetic a")}, nil)
	req := httptest.NewRequest(http.MethodPost, base+"/inbox", buf)
	req.Header.Set("Content-Type", ct)
	resp, body := h.do(req)
	var up service.UploadResponse
	_ = json.Unmarshal(body, &up)

	resp, body = h.json(http.MethodPost, base+"/inbox/"+up.Items[0].ID+"/reading/retry", nil)
	wantStatus(t, resp, body, http.StatusConflict, "")
	resp, body = h.json(http.MethodGet, base+"/inbox/groups", nil)
	wantStatus(t, resp, body, http.StatusOK, "")
	if strings.TrimSpace(string(body)) != "[]" {
		t.Fatalf("groups = %s", body)
	}
	resp, body = h.json(http.MethodGet, base+"/pages/nope/image", nil)
	wantStatus(t, resp, body, http.StatusNotFound, CodeNotFound)
	resp, body = h.json(http.MethodGet, base+"/metrics", nil)
	wantStatus(t, resp, body, http.StatusOK, "")
	var metrics []domain.Metric
	_ = json.Unmarshal(body, &metrics)
	if len(metrics) < 50 {
		t.Fatalf("metrics = %d", len(metrics))
	}
	resp, body = h.json(http.MethodGet, base+"/search?q=", nil)
	wantStatus(t, resp, body, http.StatusBadRequest, CodeValidation)
}

func TestGivenAnUnknownBacklogPersonWhenUploadingThenItIsRejected(t *testing.T) {
	h := newHarness(t)
	h.login()
	buf, ct := multipartBody(t, "files", map[string][]byte{"a.pdf": []byte("%PDF-1.7 synthetic b")}, map[string]string{"personId": "nobody"})
	req := httptest.NewRequest(http.MethodPost, "/api/tenants/"+h.tenant+"/inbox", buf)
	req.Header.Set("Content-Type", ct)
	resp, body := h.do(req)
	wantStatus(t, resp, body, http.StatusBadRequest, CodeValidation)
}

func TestGivenAFiledDocumentWithoutValuesWhenUnfiledThenItsItemIsBackInTheInbox(t *testing.T) {
	h := newHarness(t)
	h.login()
	up, filed := h.fileOne(h.firstPerson())
	base := "/api/tenants/" + h.tenant

	resp, body := h.json(http.MethodPost, base+"/documents/"+filed.Document.ID+"/unfile", nil)
	wantStatus(t, resp, body, http.StatusOK, "")
	var out struct{ ItemID string }
	_ = json.Unmarshal(body, &out)
	if out.ItemID != up.Items[0].ID {
		t.Fatalf("unfile = %s", body)
	}
	resp, body = h.json(http.MethodGet, base+"/inbox/"+out.ItemID, nil)
	wantStatus(t, resp, body, http.StatusOK, "")
	var detail service.ItemDetail
	_ = json.Unmarshal(body, &detail)
	if detail.Item.Status != domain.InboxNew {
		t.Fatalf("item = %s", body)
	}
	resp, body = h.json(http.MethodPost, base+"/documents/"+filed.Document.ID+"/unfile", nil)
	wantStatus(t, resp, body, http.StatusNotFound, CodeNotFound)
}

func TestGivenAnUnfiledItemWhenFilesAreAddedThenTheItemHoldsThemAll(t *testing.T) {
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

	buf, ct = multipartBody(t, "files", map[string][]byte{"xray.pdf": []byte("%PDF-1.7 synthetic x-ray")}, nil)
	req = httptest.NewRequest(http.MethodPost, base+"/inbox/"+up.Items[0].ID+"/files", buf)
	req.Header.Set("Content-Type", ct)
	resp, body = h.do(req)
	wantStatus(t, resp, body, http.StatusAccepted, "")
	var added service.UploadResponse
	_ = json.Unmarshal(body, &added)
	if len(added.Items) != 1 || len(added.Items[0].Files) != 2 || added.Items[0].Files[1].OriginalName != "xray.pdf" {
		t.Fatalf("added = %s", body)
	}

	req = httptest.NewRequest(http.MethodPost, base+"/inbox/"+up.Items[0].ID+"/files", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	resp, body = h.do(req)
	wantStatus(t, resp, body, http.StatusBadRequest, CodeValidation)
}

func TestGivenAFiledDocumentWhenAnExamFindingIsAddedThenItsSectionIsKept(t *testing.T) {
	h := newHarness(t)
	h.login()
	_, filed := h.fileOne(h.firstPerson())

	resp, body := h.json(http.MethodPost, "/api/tenants/"+h.tenant+"/documents/"+filed.Document.ID+"/findings",
		map[string]string{"kind": "exam", "section": "Khám Mắt", "text": "Không kính MP: 1/10.\nCó kính MP: 10/10."})
	wantStatus(t, resp, body, http.StatusCreated, "")
	var f domain.Finding
	_ = json.Unmarshal(body, &f)
	if f.Kind != "exam" || f.Section != "Khám Mắt" || f.Text != "Không kính MP: 1/10.\nCó kính MP: 10/10." {
		t.Fatalf("finding = %s", body)
	}
}
