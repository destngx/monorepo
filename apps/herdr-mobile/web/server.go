// Package web is the phone-facing HTTP server: a small JSON API over herdr,
// a live stream of agent states, and the built UI.
package web

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"apps/herdr-mobile/herdr"
)

// Herdr is what the server needs from herdr; *herdr.Client implements it.
type Herdr interface {
	Agents(ctx context.Context) ([]herdr.Agent, error)
	Screen(ctx context.Context, pane string, lines int) (string, error)
	Prompt(ctx context.Context, pane, text string) error
	SendKeys(ctx context.Context, pane string, keys []string) error
}

const (
	// screenLines is how much recent output the agent page shows.
	screenLines    = 400
	maxPromptRunes = 8000
	maxKeys        = 8
	maxBodyBytes   = 64 << 10
	uiMount        = "/ui"
)

// paneID matches herdr's pane ids, such as "w1:p1" or "wE:p7N".
var paneID = regexp.MustCompile(`^[A-Za-z0-9]+:[A-Za-z0-9]+$`)

// allowedKeys are the keys the phone may press: enough to answer an agent's
// menu or interrupt it. Anything else (ctrl+c, ctrl+d) is refused.
var allowedKeys = map[string]bool{
	"enter": true, "esc": true, "tab": true, "shift+tab": true, "space": true, "backspace": true,
	"up": true, "down": true, "left": true, "right": true,
	"1": true, "2": true, "3": true, "4": true, "5": true, "6": true, "7": true, "8": true, "9": true,
	"y": true, "n": true,
}

// Options configures the server.
type Options struct {
	Herdr Herdr
	// UIDir is the built UI (adapter-static output). Without index.html, only the API is served.
	UIDir string
	Log   *slog.Logger
	// AllowedHosts are extra host names the server answers to, besides
	// localhost and any Tailscale *.ts.net name.
	AllowedHosts []string
	// PollEvery is how often the live stream asks herdr for agent states while a phone is watching.
	PollEvery time.Duration
	// Heartbeat keeps idle streams open through proxies.
	Heartbeat time.Duration
}

type server struct {
	opts Options
	hub  *hub
}

// New returns the server's handler.
func New(opts Options) http.Handler {
	if opts.PollEvery <= 0 {
		opts.PollEvery = 2 * time.Second
	}
	if opts.Heartbeat <= 0 {
		opts.Heartbeat = 20 * time.Second
	}
	s := &server{opts: opts}
	s.hub = newHub(opts.PollEvery, s.snapshot, opts.Log)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/agents", s.listAgents)
	mux.HandleFunc("GET /api/agents/{pane}/screen", s.screen)
	mux.HandleFunc("POST /api/agents/{pane}/prompt", s.prompt)
	mux.HandleFunc("POST /api/agents/{pane}/keys", s.keys)
	mux.HandleFunc("GET /api/events", s.events)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mountUI(mux, opts.UIDir)
	return s.guard(mux)
}

// guard keeps other sites out. The host check stops DNS rebinding; the
// origin, fetch-site and content-type checks stop a page in another tab from
// posting a prompt to an agent.
func (s *server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		if !s.hostAllowed(r.Host) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Unknown host name.")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if !sameOrigin(r) {
				writeError(w, http.StatusForbidden, "FORBIDDEN", "Requests must come from this page.")
				return
			}
			if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				writeError(w, http.StatusUnsupportedMediaType, "VALIDATION", "Send JSON.")
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) hostAllowed(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	switch {
	case host == "localhost", host == "127.0.0.1", host == "::1", host == "[::1]":
		return true
	case strings.HasSuffix(host, ".ts.net"):
		return true
	}
	return slices.Contains(s.opts.AllowedHosts, host)
}

func sameOrigin(r *http.Request) bool {
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" {
		return false
	}
	origin, err := url.Parse(r.Header.Get("Origin"))
	if err != nil || origin.Host == "" {
		return false
	}
	return strings.EqualFold(origin.Host, r.Host)
}

func (s *server) listAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := s.opts.Herdr.Agents(r.Context())
	if err != nil {
		s.herdrError(w, "list agents", err)
		return
	}
	writeJSON(w, http.StatusOK, agentsBody{Agents: nonNil(agents)})
}

func (s *server) screen(w http.ResponseWriter, r *http.Request) {
	pane, ok := paneFrom(w, r)
	if !ok {
		return
	}
	text, err := s.opts.Herdr.Screen(r.Context(), pane, screenLines)
	if err != nil {
		s.herdrError(w, "read screen", err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, struct {
		Text string `json:"text"`
	}{text})
}

func (s *server) prompt(w http.ResponseWriter, r *http.Request) {
	pane, ok := paneFrom(w, r)
	if !ok {
		return
	}
	var body struct {
		Text string `json:"text"`
	}
	if !decode(w, r, &body) {
		return
	}
	text := strings.TrimSpace(body.Text)
	if text == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION", "Write a message first.")
		return
	}
	if utf8.RuneCountInString(text) > maxPromptRunes {
		writeError(w, http.StatusBadRequest, "VALIDATION", "The message is too long.")
		return
	}
	if err := s.opts.Herdr.Prompt(r.Context(), pane, text); err != nil {
		s.herdrError(w, "send prompt", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) keys(w http.ResponseWriter, r *http.Request) {
	pane, ok := paneFrom(w, r)
	if !ok {
		return
	}
	var body struct {
		Keys []string `json:"keys"`
	}
	if !decode(w, r, &body) {
		return
	}
	if len(body.Keys) == 0 || len(body.Keys) > maxKeys {
		writeError(w, http.StatusBadRequest, "VALIDATION", "Send between 1 and 8 keys.")
		return
	}
	for _, k := range body.Keys {
		if !allowedKeys[k] {
			writeError(w, http.StatusBadRequest, "VALIDATION", "Key not allowed: "+k)
			return
		}
	}
	if err := s.opts.Herdr.SendKeys(r.Context(), pane, body.Keys); err != nil {
		s.herdrError(w, "send keys", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// events streams the agent list as server-sent events, one event per change.
func (s *server) events(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	if err := rc.Flush(); err != nil {
		s.opts.Log.Warn("events: flush", "error", err)
		return
	}

	updates, unsubscribe := s.hub.subscribe()
	defer unsubscribe()
	beat := time.NewTicker(s.opts.Heartbeat)
	defer beat.Stop()
	for {
		var chunk []byte
		select {
		case <-r.Context().Done():
			return
		case <-beat.C:
			chunk = []byte(": ping\n\n")
		case data := <-updates:
			chunk = append(append([]byte("data: "), data...), '\n', '\n')
		}
		if _, err := w.Write(chunk); err != nil {
			return
		}
		if err := rc.Flush(); err != nil {
			return
		}
	}
}

// snapshot is one stream event: the agents, or why herdr could not list them.
func (s *server) snapshot(ctx context.Context) []byte {
	agents, err := s.opts.Herdr.Agents(ctx)
	body := agentsBody{Agents: nonNil(agents)}
	if err != nil {
		code, _, message := classify(err)
		body = agentsBody{Agents: []herdr.Agent{}, Error: message, Code: code}
	}
	data, err := json.Marshal(body)
	if err != nil {
		// Marshalling plain strings and bools cannot fail; keep the stream alive if it does.
		return []byte(`{"agents":[],"error":"Could not encode the agent list.","code":"INTERNAL"}`)
	}
	return data
}

type agentsBody struct {
	Agents []herdr.Agent `json:"agents"`
	Error  string        `json:"error,omitempty"`
	Code   string        `json:"code,omitempty"`
}

func nonNil(agents []herdr.Agent) []herdr.Agent {
	if agents == nil {
		return []herdr.Agent{}
	}
	return agents
}

func paneFrom(w http.ResponseWriter, r *http.Request) (string, bool) {
	pane := r.PathValue("pane")
	if !paneID.MatchString(pane) {
		writeError(w, http.StatusBadRequest, "VALIDATION", "Unknown agent id.")
		return "", false
	}
	return pane, true
}

func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(r.Body).Decode(into); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION", "Could not read the request body.")
		return false
	}
	return true
}

// classify maps a herdr failure to an API code, status and message for the phone.
func classify(err error) (code string, status int, message string) {
	var herr *herdr.Error
	switch {
	case errors.Is(err, herdr.ErrUnreachable):
		return "HERDR_UNREACHABLE", http.StatusServiceUnavailable, "herdr is not running on the laptop."
	case errors.As(err, &herr) && herr.Code == "agent_blocked":
		return "AGENT_BLOCKED", http.StatusConflict, "The agent is waiting for an answer. Answer it with the keys first."
	case errors.As(err, &herr) && herr.Code == "not_found":
		return "NOT_FOUND", http.StatusNotFound, "That agent is gone. It may have been closed."
	case errors.As(err, &herr):
		return "HERDR_ERROR", http.StatusBadGateway, "herdr refused: " + herr.Message
	case errors.Is(err, context.DeadlineExceeded):
		return "HERDR_TIMEOUT", http.StatusGatewayTimeout, "herdr did not answer in time."
	}
	return "INTERNAL", http.StatusBadGateway, "Could not reach herdr."
}

func (s *server) herdrError(w http.ResponseWriter, what string, err error) {
	code, status, message := classify(err)
	if status >= 500 {
		s.opts.Log.Warn(what, "error", err)
	}
	writeError(w, status, code, message)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// The status line is already out, so an encoding failure can only cut the body short.
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}{message, code})
}

// mountUI serves the built SPA at /ui and redirects / to it. Client-side
// routes fall back to index.html; a missing file with an extension is a 404.
func mountUI(mux *http.ServeMux, dir string) {
	index := filepath.Join(dir, "index.html")
	if info, err := os.Stat(index); err != nil || info.IsDir() {
		return
	}
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, uiMount+"/", http.StatusFound)
	})
	mux.HandleFunc("GET "+uiMount+"/", func(w http.ResponseWriter, r *http.Request) {
		// Clean against a rooted path so ".." can never climb out of dir.
		rel := path.Clean("/" + strings.TrimPrefix(r.URL.Path, uiMount))
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			if strings.HasPrefix(rel, "/_app/immutable/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.ServeFile(w, r, full)
			return
		}
		if path.Ext(rel) != "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, index)
	})
}
