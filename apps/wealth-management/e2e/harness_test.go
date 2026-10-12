// Package e2e runs the real engine against a disposable Google Sheet
// uploaded from a local .xlsx export of the workbook (REQUIREMENTS §14.3).
// It is opt-in:
//
//	RUN_LIVE_API_TESTS=true go test ./e2e/ -run '^TestE2E' -count=1 -v
//
// or `pnpx nx run wealth-management:test-e2e`. It needs GOOGLE_CLIENT_ID,
// GOOGLE_CLIENT_SECRET, GOOGLE_TEST_REFRESH_TOKEN (minted by
// `pnpx nx run wealth-management:auth-sheets -- --test`) and E2E_SOURCE_XLSX
// (path to the workbook) in the environment or .env.local. GOOGLE_SHEETS_ID,
// when set, is only used to refuse the production sheet.
// KEEP_TEST_SHEET=true keeps the test sheet for debugging.
package e2e

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/e2esheet"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

const (
	startupTimeout = 90 * time.Second
	requestTimeout = 60 * time.Second
)

// Env files, relative to this package, in the order the engine loads them.
var envFiles = []string{"../../../.env.local", "../.env.local"}

// harness is the running engine shared by every test in the run (WM-TEST-7).
type harness struct {
	runID    string
	baseURL  string
	apiToken string
	copy     e2esheet.Copy
	client   *http.Client
}

// live is nil when RUN_LIVE_API_TESTS is not "true".
var live *harness

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	if !strings.EqualFold(os.Getenv("RUN_LIVE_API_TESTS"), "true") {
		return m.Run()
	}
	ctx := context.Background()
	env, err := loadEnv()
	if err != nil {
		return abort("read .env.local", err)
	}
	runID := randomHex(4)
	productionID := strings.TrimSpace(env["GOOGLE_SHEETS_ID"])
	source, err := sourceWorkbook(env["E2E_SOURCE_XLSX"])
	if err != nil {
		return abort("find the test workbook", err)
	}
	drive, err := e2esheet.New(ctx, e2esheet.Credentials{
		ClientID:     strings.TrimSpace(env["GOOGLE_CLIENT_ID"]),
		ClientSecret: strings.TrimSpace(env["GOOGLE_CLIENT_SECRET"]),
		RefreshToken: strings.TrimSpace(env["GOOGLE_TEST_REFRESH_TOKEN"]),
	}, productionID)
	if err != nil {
		return abort("set up Drive (run `pnpx nx run wealth-management:auth-sheets -- --test` first)", err)
	}

	// WM-TEST-6: clear copies that earlier runs failed to delete.
	if swept, err := drive.SweepStale(ctx); err != nil {
		logf("janitor: %v", err)
	} else if len(swept) > 0 {
		logf("janitor deleted %d leftover copies: %s", len(swept), strings.Join(swept, ", "))
	}

	copied, err := drive.Upload(ctx, runID, source)
	if err != nil {
		return abort("upload the test workbook", err)
	}
	logf("run %s uses test sheet %s (%s) uploaded from %s", runID, copied.Name, copied.ID, filepath.Base(source))
	keep := strings.EqualFold(os.Getenv("KEEP_TEST_SHEET"), "true")
	defer func() {
		if keep {
			logf("KEEP_TEST_SHEET=true: kept %s (%s)", copied.Name, copied.ID)
			return
		}
		if err := drive.Delete(context.Background(), copied.ID); err != nil {
			logf("could not delete %s; the janitor removes it after 24h: %v", copied.Name, err)
		}
	}()

	workDir, err := filepath.Abs(filepath.Join("..", "tmp", "e2e-"+runID))
	if err != nil {
		return abort("resolve the work directory", err)
	}
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return abort("create the work directory", err)
	}
	h, stop, err := startEngine(ctx, env, runID, copied, workDir)
	if err != nil {
		return abort("start the engine (log: "+filepath.Join(workDir, "engine.log")+")", err)
	}
	live = h
	code := m.Run()
	h.invalidateCache()
	if err := stop(); err != nil {
		logf("stop engine: %v", err)
	}
	if code == 0 && !keep {
		_ = os.RemoveAll(workDir)
	} else {
		logf("engine log kept at %s", filepath.Join(workDir, "engine.log"))
	}
	return code
}

// sourceWorkbook resolves E2E_SOURCE_XLSX, expanding a leading "~/".
func sourceWorkbook(raw string) (string, error) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "", errors.New("set E2E_SOURCE_XLSX to an .xlsx export of the workbook")
	}
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand %s: %w", path, err)
		}
		path = filepath.Join(home, rest)
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("test workbook: %w", err)
	}
	return path, nil
}

// loadEnv merges the process environment with .env.local, the file winning
// as it does in the engine (config.Load).
func loadEnv() (map[string]string, error) {
	env := map[string]string{}
	for _, entry := range os.Environ() {
		if key, value, ok := strings.Cut(entry, "="); ok {
			env[key] = value
		}
	}
	files := envFiles
	if override := strings.TrimSpace(os.Getenv("WM_ENV_FILE")); override != "" && override != "none" {
		files = []string{override}
	}
	for _, path := range files {
		values, err := godotenv.Read(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		for key, value := range values {
			env[key] = value
		}
	}
	return env, nil
}

// startEngine builds the engine and runs it against the copy with its own
// cache namespace, write token and test mode on (WM-TEST-2, WM-TEST-4).
func startEngine(ctx context.Context, env map[string]string, runID string, copied e2esheet.Copy, workDir string) (*harness, func() error, error) {
	binary := filepath.Join(workDir, "wm")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	build.Dir = ".."
	if output, err := build.CombinedOutput(); err != nil {
		return nil, nil, fmt.Errorf("go build: %w\n%s", err, output)
	}
	port, err := freePort()
	if err != nil {
		return nil, nil, err
	}
	h := &harness{
		runID:    runID,
		baseURL:  "http://127.0.0.1:" + strconv.Itoa(port),
		apiToken: randomHex(16),
		copy:     copied,
		client:   &http.Client{Timeout: requestTimeout},
	}
	overrides := map[string]string{
		"WM_ENV_FILE":          "none",
		"WM_TEST_MODE":         "true",
		"GOOGLE_SHEETS_ID":     copied.ID,
		"GOOGLE_REFRESH_TOKEN": env["GOOGLE_TEST_REFRESH_TOKEN"],
		"CACHE_KEY_PREFIX":     "wm-test:" + runID + ":",
		"HOST":                 "127.0.0.1",
		"PORT":                 strconv.Itoa(port),
		"API_TOKEN":            h.apiToken,
		"LOG_LEVEL":            "info",
		"LOG_COLOR":            "false",
	}
	environment := make([]string, 0, len(env)+len(overrides))
	for key, value := range env {
		if _, overridden := overrides[key]; !overridden {
			environment = append(environment, key+"="+value)
		}
	}
	for key, value := range overrides {
		environment = append(environment, key+"="+value)
	}

	logFile, err := os.Create(filepath.Join(workDir, "engine.log"))
	if err != nil {
		return nil, nil, err
	}
	engine := exec.Command(binary)
	engine.Dir = ".."
	engine.Env = environment
	engine.Stdout = logFile
	engine.Stderr = logFile
	if err := engine.Start(); err != nil {
		logFile.Close()
		return nil, nil, err
	}
	// done closes when the engine exits; waitErr is set before that.
	done := make(chan struct{})
	var waitErr error
	go func() {
		waitErr = engine.Wait()
		close(done)
	}()
	stop := func() error {
		defer logFile.Close()
		select {
		case <-done:
			return nil
		default:
		}
		_ = engine.Process.Signal(syscall.SIGTERM)
		select {
		case <-done:
			return nil
		case <-time.After(10 * time.Second):
			return engine.Process.Kill()
		}
	}
	exited := func() error {
		select {
		case <-done:
			return fmt.Errorf("engine exited during startup: %v", waitErr)
		default:
			return nil
		}
	}
	if err := h.waitReady(exited); err != nil {
		_ = stop()
		return nil, nil, err
	}
	return h, stop, nil
}

// waitReady polls /api/health until the sheets check is ok. Other
// dependencies (cache, market providers) may be unconfigured.
func (h *harness) waitReady(exited func() error) error {
	deadline := time.Now().Add(startupTimeout)
	var last string
	for time.Now().Before(deadline) {
		if err := exited(); err != nil {
			return err
		}
		var health domain.HealthStatus
		if _, raw, err := h.call(http.MethodGet, "/api/health", nil, false); err == nil && json.Unmarshal(raw, &health) == nil {
			for _, check := range health.Checks {
				if check.Name == "sheets" && check.Status == domain.HealthOK {
					return nil
				}
			}
			last = string(raw)
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("sheets not healthy after %s; last health: %s", startupTimeout, last)
}

// invalidateCache drops the run's sheet-derived cache keys (they also expire).
func (h *harness) invalidateCache() {
	if status, raw, err := h.call(http.MethodPost, "/api/sync", nil, true); err != nil || status != http.StatusOK {
		logf("cache invalidation after the run failed: status %d, %v, %s", status, err, raw)
	}
}

func (h *harness) call(method string, path string, body any, authorized bool, headers ...string) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(payload)
	}
	request, err := http.NewRequest(method, h.baseURL+path, reader)
	if err != nil {
		return 0, nil, err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if authorized {
		request.Header.Set("Authorization", "Bearer "+h.apiToken)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		request.Header.Set(headers[i], headers[i+1])
	}
	response, err := h.client.Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	return response.StatusCode, raw, err
}

// engine returns the live harness or skips the test.
func engine(t *testing.T) *harness {
	t.Helper()
	if live == nil {
		t.Skip("set RUN_LIVE_API_TESTS=true to run live e2e tests against a disposable test sheet")
	}
	return live
}

// request sends a call, fails the test on a transport error or another
// status, and decodes the JSON body into out when out is not nil.
func (h *harness) request(t *testing.T, method string, path string, body any, want int, out any, headers ...string) []byte {
	t.Helper()
	authorized := method != http.MethodGet
	status, raw, err := h.call(method, path, body, authorized, headers...)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	if status != want {
		t.Fatalf("%s %s: status %d, want %d: %s", method, path, status, want, raw)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			t.Fatalf("%s %s: decode: %v: %s", method, path, err, raw)
		}
	}
	return raw
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func randomHex(size int) string {
	buffer := make([]byte, size)
	_, _ = rand.Read(buffer)
	return hex.EncodeToString(buffer)
}

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "e2e: "+format+"\n", args...)
}

func abort(step string, err error) int {
	logf("%s: %v", step, err)
	return 1
}
