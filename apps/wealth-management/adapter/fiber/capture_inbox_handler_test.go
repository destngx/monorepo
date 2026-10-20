package fiber

import (
	configAdapter "apps/wealth-management/adapter/config"
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

type fakeCaptureInbox struct {
	submitted []domain.QuickCaptureInput
	rejected  []string
	approval  domain.CaptureDraftApproval
	editedID  string
	edit      domain.CaptureDraftEdit
	materials map[string]domain.CaptureMaterial
}

func (f *fakeCaptureInbox) Submit(_ context.Context, input domain.QuickCaptureInput) (domain.QuickCapture, bool, error) {
	if len(input.Image) == 0 && strings.TrimSpace(input.Note) == "" {
		return domain.QuickCapture{}, false, fmt.Errorf("%w: an image or a note is required", domain.ErrValidation)
	}
	f.submitted = append(f.submitted, input)
	return domain.QuickCapture{ID: "cap_1", Status: domain.QuickCaptureQueued, CreatedAt: time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC), HasImage: len(input.Image) > 0}, len(f.submitted) > 1, nil
}
func (f *fakeCaptureInbox) Overview(context.Context) (domain.QuickCaptureOverview, error) {
	return domain.QuickCaptureOverview{Captures: []domain.QuickCapture{}}, nil
}
func (f *fakeCaptureInbox) Retry(_ context.Context, id string) error {
	return fmt.Errorf("%w: capture %s", domain.ErrConflict, id)
}
func (f *fakeCaptureInbox) Discard(context.Context, string) error { return nil }
func (f *fakeCaptureInbox) Drafts(context.Context) (domain.CaptureDrafts, error) {
	return domain.CaptureDrafts{}, nil
}
func (f *fakeCaptureInbox) Material(_ context.Context, id string) (domain.CaptureMaterial, error) {
	material, ok := f.materials[id]
	if !ok {
		return material, fmt.Errorf("%w: material %s", domain.ErrNotFound, id)
	}
	return material, nil
}
func (f *fakeCaptureInbox) Approve(_ context.Context, approval domain.CaptureDraftApproval) (domain.TransactionBatchResult, error) {
	f.approval = approval
	return domain.TransactionBatchResult{BatchID: approval.BatchID, Created: len(approval.Drafts)}, nil
}
func (f *fakeCaptureInbox) Edit(_ context.Context, id string, edit domain.CaptureDraftEdit) error {
	f.editedID, f.edit = id, edit
	return nil
}
func (f *fakeCaptureInbox) Reject(_ context.Context, ids []string) error {
	f.rejected = ids
	return nil
}

type fakeCaptureLearning struct {
	rows []domain.CaptureWrittenRow
}

func (f *fakeCaptureLearning) LearnWritten(_ context.Context, rows []domain.CaptureWrittenRow) (domain.CaptureLearnResult, error) {
	if err := domain.ValidateCaptureWrittenRows(rows); err != nil {
		return domain.CaptureLearnResult{}, err
	}
	f.rows = rows
	return domain.CaptureLearnResult{Remembered: len(rows), Rules: domain.CaptureRulesSaveResult{Created: 1}}, nil
}

func newCaptureInboxApp(t *testing.T, inbox *fakeCaptureInbox) *fiber.App {
	return newCaptureLearningApp(t, inbox, &fakeCaptureLearning{})
}

func newCaptureLearningApp(t *testing.T, inbox *fakeCaptureInbox, learning *fakeCaptureLearning) *fiber.App {
	t.Helper()
	app := fiber.New()
	RegisterCaptureInboxRoutes(NewRouteRegistrar(app, "/api", ""), NewCaptureInboxHandler(inbox, learning, logger.NewTestLogger(t)), NewWriteGuard(staticToken(testToken), "0.0.0.0"))
	return app
}

// postForm sends what the iOS Shortcut sends: a multipart form.
func postForm(t *testing.T, app *fiber.App, token string, image []byte, note string) (int, string) {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	if image != nil {
		part, err := form.CreateFormFile("image", "screenshot.png")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write(image)
	}
	if note != "" {
		_ = form.WriteField("note", note)
	}
	_ = form.Close()
	request := httptest.NewRequest(http.MethodPost, "/api/capture/inbox", &body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(data)
}

func TestGivenAShortcutFormWhenSubmittingThenTheCaptureIsAcceptedBeforeReading(t *testing.T) {
	inbox := &fakeCaptureInbox{}
	app := newCaptureInboxApp(t, inbox)

	status, body := postForm(t, app, testToken, []byte("\x89PNG\r\n\x1a\nimage"), "team lunch")

	if status != http.StatusAccepted {
		t.Fatalf("status %d body %s, want 202", status, body)
	}
	var receipt QuickCaptureReceipt
	if err := json.Unmarshal([]byte(body), &receipt); err != nil || receipt.ID != "cap_1" || receipt.Duplicate {
		t.Fatalf("receipt = %+v err %v", receipt, err)
	}
	if len(inbox.submitted) != 1 || inbox.submitted[0].Note != "team lunch" || !bytes.HasPrefix(inbox.submitted[0].Image, []byte("\x89PNG")) {
		t.Fatalf("submitted = %+v, want the image and the note", inbox.submitted)
	}
}

func TestGivenNoTokenWhenSubmittingThenTheCaptureIsRefused(t *testing.T) {
	inbox := &fakeCaptureInbox{}

	status, _ := postForm(t, newCaptureInboxApp(t, inbox), "", []byte("image"), "")

	if status != http.StatusUnauthorized || len(inbox.submitted) != 0 {
		t.Fatalf("status %d submitted %d, want 401 and nothing stored", status, len(inbox.submitted))
	}
}

func TestGivenAnEmptyFormWhenSubmittingThenItIsAValidationError(t *testing.T) {
	status, body := postForm(t, newCaptureInboxApp(t, &fakeCaptureInbox{}), testToken, nil, "")

	if status != http.StatusBadRequest {
		t.Fatalf("status %d body %s, want 400", status, body)
	}
}

func TestGivenJSONWhenSubmittingThenItAsksForAForm(t *testing.T) {
	status, body := call(t, newCaptureInboxApp(t, &fakeCaptureInbox{}), http.MethodPost, "/api/capture/inbox", `{"note":"x"}`)

	if status != http.StatusBadRequest || !strings.Contains(body, "multipart") {
		t.Fatalf("status %d body %s, want 400 asking for multipart", status, body)
	}
}

func TestGivenAStoredScreenshotWhenFetchingItThenItIsServedUncached(t *testing.T) {
	inbox := &fakeCaptureInbox{materials: map[string]domain.CaptureMaterial{
		"cap_1-image": {ID: "cap_1-image", Kind: domain.CaptureImage, MediaType: "image/png", Data: []byte("png")},
	}}
	app := newCaptureInboxApp(t, inbox)

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/capture/materials/cap_1-image", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK || string(data) != "png" || response.Header.Get("Content-Type") != "image/png" || response.Header.Get("Cache-Control") != "private, no-store" {
		t.Fatalf("status %d type %q cache %q body %q", response.StatusCode, response.Header.Get("Content-Type"), response.Header.Get("Cache-Control"), data)
	}
	if status, _ := call(t, app, http.MethodGet, "/api/capture/materials/cap_9-image", ""); status != http.StatusNotFound {
		t.Fatalf("missing material status %d, want 404", status)
	}
}

func TestGivenDraftsWhenApprovingAndRejectingThenTheBodiesReachTheInbox(t *testing.T) {
	inbox := &fakeCaptureInbox{}
	app := newCaptureInboxApp(t, inbox)

	status, body := call(t, app, http.MethodPost, "/api/capture/drafts/approve", `{"batchId":"b1","drafts":[{"id":"drf_1","transaction":{"accountName":"Sacombank","date":"16/10/2026","payment":50000}}]}`)
	if status != http.StatusOK || inbox.approval.BatchID != "b1" || inbox.approval.Drafts[0].Transaction.AccountName != "Sacombank" {
		t.Fatalf("approve status %d body %s approval %+v", status, body, inbox.approval)
	}
	if status, _ := call(t, app, http.MethodPost, "/api/capture/drafts/reject", `{"ids":["drf_2"]}`); status != http.StatusNoContent || inbox.rejected[0] != "drf_2" {
		t.Fatalf("reject status %d rejected %v", status, inbox.rejected)
	}
	if status, body := call(t, app, http.MethodPatch, "/api/capture/drafts/drf_3", `{"transaction":{"accountName":"Sacombank","payee":"Pho 24","category":"Food"},"categorySource":"ai"}`); status != http.StatusNoContent || inbox.editedID != "drf_3" || inbox.edit.Transaction.Payee != "Pho 24" || inbox.edit.CategorySource != "ai" {
		t.Fatalf("edit status %d body %s edit %s %+v", status, body, inbox.editedID, inbox.edit)
	}
	if status, _ := call(t, app, http.MethodPost, "/api/capture/inbox/cap_1/retry", ""); status != http.StatusConflict {
		t.Fatalf("retry status %d, want the inbox conflict as 409", status)
	}
}

func TestGivenRowsWrittenFromTheCapturePageWhenLearningThenTheyReachTheLearner(t *testing.T) {
	learning := &fakeCaptureLearning{}
	app := newCaptureLearningApp(t, &fakeCaptureInbox{}, learning)

	status, body := call(t, app, http.MethodPost, "/api/capture/learn", `{"rows":[{"suggestion":{"counterparty":"NGUYEN VAN A","direction":"out","amount":25000},"transaction":{"accountName":"Vietcombank","payee":"Banh mi stall"},"transactionId":"tx1"}]}`)

	if status != http.StatusOK || !strings.Contains(body, `"remembered":1`) || len(learning.rows) != 1 || learning.rows[0].Suggestion.Counterparty != "NGUYEN VAN A" || learning.rows[0].TransactionID != "tx1" {
		t.Fatalf("status %d body %s rows %+v", status, body, learning.rows)
	}
	if status, _ := call(t, app, http.MethodPost, "/api/capture/learn", `{"rows":[]}`); status != http.StatusBadRequest {
		t.Fatalf("no rows: status %d, want 400", status)
	}
}

func TestGivenNoSheetWhenCallingCaptureRoutesThenTheyAnswerWithTheCause(t *testing.T) {
	app := fiber.New()
	RegisterUnavailableCaptureInboxRoutes(NewRouteRegistrar(app, "/api", ""), fmt.Errorf("%w: quick capture needs the sheet", configAdapter.ErrMissingSheetsConfig), logger.NewTestLogger(t))

	status, body := call(t, app, http.MethodGet, "/api/capture/drafts", "")

	if status != http.StatusServiceUnavailable || !strings.Contains(body, CodeSheetsNotConfigured) {
		t.Fatalf("status %d body %s", status, body)
	}
}
