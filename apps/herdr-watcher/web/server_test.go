package web

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"apps/herdr-watcher/herdr"
)

// fakeHerdr records what the server asked for and answers from its fields.
type fakeHerdr struct {
	mu      sync.Mutex
	agents  []herdr.Agent
	err     error
	screen  string
	prompts []string
	keys    [][]string
	lines   int
}

func (f *fakeHerdr) Agents(context.Context) ([]herdr.Agent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]herdr.Agent(nil), f.agents...), f.err
}

func (f *fakeHerdr) Screen(_ context.Context, pane string, lines int) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lines = lines
	return pane + ": " + f.screen, f.err
}

func (f *fakeHerdr) Prompt(_ context.Context, pane, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.prompts = append(f.prompts, pane+" "+text)
	return f.err
}

func (f *fakeHerdr) SendKeys(_ context.Context, pane string, keys []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.keys = append(f.keys, append([]string{pane}, keys...))
	return f.err
}

func (f *fakeHerdr) set(agents []herdr.Agent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.agents = agents
}

func newTestServer(t *testing.T, h Herdr) http.Handler {
	t.Helper()
	ui := t.TempDir()
	if err := os.WriteFile(filepath.Join(ui, "index.html"), []byte("<!doctype html>shell"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(ui, "_app", "immutable"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ui, "_app", "immutable", "a.js"), []byte("js"), 0o600); err != nil {
		t.Fatal(err)
	}
	return New(Options{
		Herdr: h, UIDir: ui, Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		PollEvery: 10 * time.Millisecond, Heartbeat: time.Hour,
	})
}

// do sends a request the way the phone does: through Tailscale's host name,
// with a same-origin JSON body for writes.
func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Host = "laptop.example.ts.net"
	if method != http.MethodGet {
		req.Header.Set("Origin", "https://laptop.example.ts.net")
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestListAgents(t *testing.T) {
	fake := &fakeHerdr{agents: []herdr.Agent{{PaneID: "w1:p1", Status: herdr.StatusBlocked, Title: "Fix the form"}}}
	rec := do(t, newTestServer(t, fake), http.MethodGet, "/api/agents", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var body struct{ Agents []herdr.Agent }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(body.Agents, fake.agents) {
		t.Fatalf("agents = %+v", body.Agents)
	}
}

func TestNoAgentsIsAnEmptyList(t *testing.T) {
	rec := do(t, newTestServer(t, &fakeHerdr{}), http.MethodGet, "/api/agents", "")
	if got := strings.TrimSpace(rec.Body.String()); got != `{"agents":[]}` {
		t.Fatalf("body = %s", got)
	}
}

func TestHerdrDownIsServiceUnavailable(t *testing.T) {
	fake := &fakeHerdr{err: fmt.Errorf("%w at /x: refused", herdr.ErrUnreachable)}
	rec := do(t, newTestServer(t, fake), http.MethodGet, "/api/agents", "")
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), `"code":"HERDR_UNREACHABLE"`) {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
}

func TestScreen(t *testing.T) {
	fake := &fakeHerdr{screen: "hello"}
	rec := do(t, newTestServer(t, fake), http.MethodGet, "/api/agents/w1%3Ap1/screen", "")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"text":"w1:p1: hello"}` {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if fake.lines != screenLines {
		t.Fatalf("lines = %d", fake.lines)
	}
}

func TestBadPaneIDIsRejected(t *testing.T) {
	fake := &fakeHerdr{}
	rec := do(t, newTestServer(t, fake), http.MethodGet, "/api/agents/..%2Fx/screen", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
}

func TestPrompt(t *testing.T) {
	fake := &fakeHerdr{}
	h := newTestServer(t, fake)

	rec := do(t, h, http.MethodPost, "/api/agents/w1:p1/prompt", `{"text":"  run the tests  "}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if !reflect.DeepEqual(fake.prompts, []string{"w1:p1 run the tests"}) {
		t.Fatalf("prompts = %q", fake.prompts)
	}

	if rec := do(t, h, http.MethodPost, "/api/agents/w1:p1/prompt", `{"text":"   "}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty prompt: status %d", rec.Code)
	}
	long := `{"text":"` + strings.Repeat("a", maxPromptRunes+1) + `"}`
	if rec := do(t, h, http.MethodPost, "/api/agents/w1:p1/prompt", long); rec.Code != http.StatusBadRequest {
		t.Fatalf("long prompt: status %d", rec.Code)
	}
}

func TestPromptToBlockedAgentIsConflict(t *testing.T) {
	fake := &fakeHerdr{err: &herdr.Error{Code: "agent_blocked", Message: "agent is blocked"}}
	rec := do(t, newTestServer(t, fake), http.MethodPost, "/api/agents/w1:p1/prompt", `{"text":"hi"}`)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"code":"AGENT_BLOCKED"`) {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
}

func TestUnknownPaneIsNotFound(t *testing.T) {
	fake := &fakeHerdr{err: &herdr.Error{Code: "not_found", Message: "pane not found"}}
	rec := do(t, newTestServer(t, fake), http.MethodGet, "/api/agents/w1:p9/screen", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
}

func TestKeys(t *testing.T) {
	fake := &fakeHerdr{}
	h := newTestServer(t, fake)

	if rec := do(t, h, http.MethodPost, "/api/agents/w1:p1/keys", `{"keys":["1","enter"]}`); rec.Code != http.StatusNoContent {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if !reflect.DeepEqual(fake.keys, [][]string{{"w1:p1", "1", "enter"}}) {
		t.Fatalf("keys = %q", fake.keys)
	}
	for _, body := range []string{`{"keys":["ctrl+c"]}`, `{"keys":[]}`, `{"keys":["enter","enter","enter","enter","enter","enter","enter","enter","enter"]}`} {
		if rec := do(t, h, http.MethodPost, "/api/agents/w1:p1/keys", body); rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status %d", body, rec.Code)
		}
	}
}

func TestGuard(t *testing.T) {
	fake := &fakeHerdr{}
	h := newTestServer(t, fake)
	cases := []struct {
		name   string
		mutate func(*http.Request)
		want   int
	}{
		{"unknown host (DNS rebinding)", func(r *http.Request) { r.Host = "evil.example.com" }, http.StatusForbidden},
		{"no origin", func(r *http.Request) { r.Header.Del("Origin") }, http.StatusForbidden},
		{"other origin", func(r *http.Request) { r.Header.Set("Origin", "https://evil.example.com") }, http.StatusForbidden},
		{"cross-site fetch", func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") }, http.StatusForbidden},
		{"form post", func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, http.StatusUnsupportedMediaType},
		{"tailscale IP", func(r *http.Request) {
			r.Host = "100.101.102.103:9393"
			r.Header.Set("Origin", "http://100.101.102.103:9393")
		}, http.StatusNoContent},
		{"origin on another IP", func(r *http.Request) {
			r.Host = "100.101.102.103:9393"
			r.Header.Set("Origin", "http://100.101.102.104:9393")
		}, http.StatusForbidden},
		{"localhost", func(r *http.Request) {
			r.Host = "127.0.0.1:9393"
			r.Header.Set("Origin", "http://127.0.0.1:9393")
		}, http.StatusNoContent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/agents/w1:p1/prompt", strings.NewReader(`{"text":"hi"}`))
			req.Host = "laptop.example.ts.net"
			req.Header.Set("Origin", "https://laptop.example.ts.net")
			req.Header.Set("Content-Type", "application/json")
			tc.mutate(req)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status %d, want %d: %s", rec.Code, tc.want, rec.Body)
			}
		})
	}
	if len(fake.prompts) != 2 {
		t.Fatalf("prompts sent: %q", fake.prompts)
	}
}

func TestEventsStreamSendsChanges(t *testing.T) {
	fake := &fakeHerdr{agents: []herdr.Agent{{PaneID: "w1:p1", Status: herdr.StatusWorking}}}
	srv := httptest.NewServer(newTestServer(t, fake))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/events", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if ct := res.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type %q", ct)
	}

	lines := bufio.NewScanner(res.Body)
	next := func() string {
		for lines.Scan() {
			if data, ok := strings.CutPrefix(lines.Text(), "data: "); ok {
				return data
			}
		}
		t.Fatalf("stream ended: %v", lines.Err())
		return ""
	}
	if got := next(); !strings.Contains(got, `"status":"working"`) {
		t.Fatalf("first event %s", got)
	}
	fake.set([]herdr.Agent{{PaneID: "w1:p1", Status: herdr.StatusBlocked}})
	if got := next(); !strings.Contains(got, `"status":"blocked"`) {
		t.Fatalf("second event %s", got)
	}
}

func TestUI(t *testing.T) {
	h := newTestServer(t, &fakeHerdr{})
	cases := []struct {
		path, want string
		code       int
	}{
		{"/", "", http.StatusFound},
		{"/ui/", "shell", http.StatusOK},
		{"/ui/agents/w1:p1", "shell", http.StatusOK},
		{"/ui/_app/immutable/a.js", "js", http.StatusOK},
		{"/ui/missing.js", "", http.StatusNotFound},
	}
	for _, tc := range cases {
		rec := do(t, h, http.MethodGet, tc.path, "")
		if rec.Code != tc.code || (tc.want != "" && !strings.Contains(rec.Body.String(), tc.want)) {
			t.Fatalf("%s: status %d body %q", tc.path, rec.Code, rec.Body)
		}
	}
	if rec := do(t, h, http.MethodGet, "/ui/%2E%2E/%2E%2E/index.html", ""); strings.Contains(rec.Body.String(), "module") {
		t.Fatal("served a file outside the UI folder")
	}
	rec := do(t, h, http.MethodGet, "/ui/_app/immutable/a.js", "")
	if !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("cache %q", rec.Header().Get("Cache-Control"))
	}
}
