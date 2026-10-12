package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDashboardHandlerServesUTF8HTML(t *testing.T) {
	rec := httptest.NewRecorder()
	NewDashboardHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics/dashboard", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	// The page renders non-ASCII status glyphs, so the charset must be declared.
	assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), "<!doctype html>")
	// The page polls these endpoints; keep it in sync with the routes in main.go.
	assert.Contains(t, rec.Body.String(), "fetch('/metrics'")
	assert.Contains(t, rec.Body.String(), "fetch('/metrics/reset'")
}
