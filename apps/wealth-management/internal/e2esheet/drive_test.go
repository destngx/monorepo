package e2esheet

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"apps/wealth-management/domain"

	"google.golang.org/api/option"
)

// fakeDrive serves the four Drive calls the harness makes.
type fakeDrive struct {
	mu      sync.Mutex
	files   map[string]map[string]string // id -> {name, createdTime}
	copyAs  string                       // id the next copy gets
	deleted []string
}

func (f *fakeDrive) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	path := strings.TrimPrefix(r.URL.Path, "/files")
	switch {
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/copy"):
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.files[f.copyAs] = map[string]string{"name": body["name"]}
		_ = json.NewEncoder(w).Encode(map[string]string{"id": f.copyAs, "name": body["name"]})
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

func TestGivenProductionSheetWhenCopyingThenReturnsNamedCopy(t *testing.T) {
	fake := &fakeDrive{files: map[string]map[string]string{"prod": {"name": "Wealth"}}, copyAs: "copy-1"}
	created, err := newFake(t, fake, now).CopyProduction(context.Background(), "run1")
	if err != nil || created != (Copy{ID: "copy-1", Name: "wm-e2e-run1-20261013T010000Z"}) {
		t.Fatalf("copy = %+v, err = %v", created, err)
	}
}

func TestGivenCopyReturningProductionIDWhenCopyingThenRefusesIt(t *testing.T) {
	fake := &fakeDrive{files: map[string]map[string]string{"prod": {"name": "Wealth"}}, copyAs: "prod"}
	if _, err := newFake(t, fake, now).CopyProduction(context.Background(), "run1"); !errors.Is(err, domain.ErrNotTestSheet) {
		t.Fatalf("expected the production id to be refused, got %v", err)
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
