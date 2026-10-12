package e2esheet

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"apps/wealth-management/domain"

	"google.golang.org/api/option"
)

// fakeDrive serves the four Drive calls the harness makes.
type fakeDrive struct {
	mu       sync.Mutex
	files    map[string]map[string]string // id -> {name, createdTime}
	uploadAs string                       // id the next upload gets
	uploaded []byte
	deleted  []string
}

// upload records a multipart create: JSON metadata, then the xlsx media.
func (f *fakeDrive) upload(w http.ResponseWriter, r *http.Request) {
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	parts := multipart.NewReader(r.Body, params["boundary"])
	var metadata map[string]string
	for {
		part, err := parts.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if part.Header.Get("Content-Type") == xlsxMime {
			f.uploaded, _ = io.ReadAll(part)
			continue
		}
		_ = json.NewDecoder(part).Decode(&metadata)
	}
	if metadata["mimeType"] != spreadsheetsMime || f.uploaded == nil {
		http.Error(w, "expected an xlsx converted to a Google Sheet", http.StatusBadRequest)
		return
	}
	f.files[f.uploadAs] = map[string]string{"name": metadata["name"]}
	_ = json.NewEncoder(w).Encode(map[string]string{"id": f.uploadAs, "name": metadata["name"]})
}

func (f *fakeDrive) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	path := strings.TrimPrefix(r.URL.Path, "/files")
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/upload/drive/v3/files":
		f.upload(w, r)
	case r.Method == http.MethodGet && path == "":
		var files []map[string]string
		for id, file := range f.files {
			if strings.Contains(file["name"], domain.TestSheetPrefix) {
				files = append(files, map[string]string{"id": id, "name": file["name"], "createdTime": file["createdTime"]})
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"files": files})
	case r.Method == http.MethodGet:
		id := strings.TrimPrefix(path, "/")
		file, ok := f.files[id]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id, "name": file["name"]})
	case r.Method == http.MethodDelete:
		id := strings.TrimPrefix(path, "/")
		delete(f.files, id)
		f.deleted = append(f.deleted, id)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
	}
}

func newFake(t *testing.T, fake *fakeDrive, now time.Time) *Drive {
	t.Helper()
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	d, err := NewWithOptions(context.Background(), "prod", option.WithEndpoint(server.URL+"/"), option.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	d.now = func() time.Time { return now }
	return d
}

var now = time.Date(2026, 10, 13, 1, 0, 0, 0, time.UTC)

func writeWorkbook(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("PK fake xlsx"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestGivenWorkbookWhenUploadingThenConvertsToNamedTestSheet(t *testing.T) {
	fake := &fakeDrive{files: map[string]map[string]string{}, uploadAs: "copy-1"}
	created, err := newFake(t, fake, now).Upload(context.Background(), "run1", writeWorkbook(t, "Money Manager.xlsx"))
	if err != nil || created != (Copy{ID: "copy-1", Name: "wm-e2e-run1-20261013T010000Z"}) {
		t.Fatalf("upload = %+v, err = %v", created, err)
	}
	if string(fake.uploaded) != "PK fake xlsx" {
		t.Fatalf("uploaded %q", fake.uploaded)
	}
}

func TestGivenUploadReturningProductionIDWhenUploadingThenRefusesIt(t *testing.T) {
	fake := &fakeDrive{files: map[string]map[string]string{"prod": {"name": "Wealth"}}, uploadAs: "prod"}
	if _, err := newFake(t, fake, now).Upload(context.Background(), "run1", writeWorkbook(t, "book.xlsx")); !errors.Is(err, domain.ErrNotTestSheet) {
		t.Fatalf("expected the production id to be refused, got %v", err)
	}
}

func TestGivenNonXLSXWhenUploadingThenRefusesBeforeCallingDrive(t *testing.T) {
	fake := &fakeDrive{files: map[string]map[string]string{}, uploadAs: "copy-1"}
	if _, err := newFake(t, fake, now).Upload(context.Background(), "run1", writeWorkbook(t, "book.csv")); err == nil || len(fake.files) != 0 {
		t.Fatalf("expected a refusal without an upload, got %v, files %v", err, fake.files)
	}
}

func TestGivenNonCopyWhenDeletingThenRefusesAndKeepsIt(t *testing.T) {
	fake := &fakeDrive{files: map[string]map[string]string{"prod": {"name": "Wealth"}, "other": {"name": "Budget 2026"}}}
	d := newFake(t, fake, now)
	for _, id := range []string{"prod", "other"} {
		if err := d.Delete(context.Background(), id); !errors.Is(err, domain.ErrNotTestSheet) {
			t.Fatalf("delete %s: expected refusal, got %v", id, err)
		}
	}
	if len(fake.deleted) != 0 {
		t.Fatalf("deleted %v", fake.deleted)
	}
}

func TestGivenLeftoverCopiesWhenSweepingThenDeletesOnlyStaleOnes(t *testing.T) {
	fake := &fakeDrive{files: map[string]map[string]string{
		"old":     {"name": "wm-e2e-a-20261010T000000Z", "createdTime": "2026-10-10T00:00:00Z"},
		"fresh":   {"name": "wm-e2e-b-20261013T000000Z", "createdTime": "2026-10-13T00:00:00Z"},
		"renamed": {"name": "Copy of wm-e2e-c", "createdTime": "2026-10-01T00:00:00Z"},
	}}
	deleted, err := newFake(t, fake, now).SweepStale(context.Background())
	if err != nil || len(deleted) != 1 || deleted[0] != "wm-e2e-a-20261010T000000Z" {
		t.Fatalf("deleted = %v, err = %v", deleted, err)
	}
	if _, kept := fake.files["fresh"]; !kept {
		t.Fatal("a copy younger than 24h may belong to a running test")
	}
}
