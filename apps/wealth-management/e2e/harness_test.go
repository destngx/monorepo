// Package e2e runs the real engine against a local .xlsx export of the
// workbook (REQUIREMENTS §14.3). The engine serves the sheet from that file
// in memory (WM_TEST_WORKBOOK), so a run never reaches Google and never
// changes the file. It is opt-in:
//
//	E2E_SOURCE_XLSX=~/path/to/book.xlsx RUN_LIVE_API_TESTS=true go test ./e2e/ -run '^TestE2E' -count=1 -v
//
// or `pnpx nx run wealth-management:test-e2e` with E2E_SOURCE_XLSX exported.
// The engine runs with its own settings store, seeded from the user's when
// one exists (for AI and market data) minus every Google setting, so the run
// never changes the user's settings and has no way to reach their sheet.
package e2e

import (
	configAdapter "apps/wealth-management/adapter/config"
	settingsAdapter "apps/wealth-management/adapter/settings"
	"apps/wealth-management/domain"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
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
)

const (
	startupTimeout = 90 * time.Second
	requestTimeout = 60 * time.Second
)

// harness is the running engine shared by every test in the run (WM-TEST-7).
type harness struct {
	runID    string
	baseURL  string
	apiToken string
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
	source, err := sourceWorkbook(os.Getenv("E2E_SOURCE_XLSX"))
	if err != nil {
		return abort("find the test workbook", err)
	}
	settings, err := loadSettings(ctx)
	if err != nil {
		return abort("read the settings store", err)
	}
	runID := randomHex(4)
	logf("run %s serves the sheet from %s", runID, filepath.Base(source))

	workDir, err := filepath.Abs(filepath.Join("..", "tmp", "e2e-"+runID))
	if err != nil {
		return abort("resolve the work directory", err)
	}
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return abort("create the work directory", err)
	}
	h, stop, err := startEngine(ctx, settings, runID, source, workDir)
	if err != nil {
		return abort("start the engine (log: "+filepath.Join(workDir, "engine.log")+")", err)
	}
	live = h
	code := m.Run()
	h.invalidateCache()
	if err := stop(); err != nil {
		logf("stop engine: %v", err)
	}
	if code == 0 {
		_ = os.RemoveAll(workDir)
	} else {
		logf("engine log kept at %s", filepath.Join(workDir, "engine.log"))
	}
	return code
}

// sourceWorkbook resolves E2E_SOURCE_XLSX, expanding a leading "~/".
func sourceWorkbook(raw string) (string, error) {
	path, err := configAdapter.ExpandHome(raw)
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", errors.New("set E2E_SOURCE_XLSX to an .xlsx export of the workbook")
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("test workbook: %w", err)
	}
	return filepath.Abs(path)
}

// loadSettings reads the user's settings store without creating it; with no
// store the run starts from defaults.
func loadSettings(ctx context.Context) (map[string]string, error) {
	path, err := settingsAdapter.ResolvePath(configAdapter.LoadSettingsPath())
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}, nil
	} else if err != nil {
		return nil, err
	}
	store, err := settingsAdapter.Open(path)
	if err != nil {
		return nil, err
	}
	defer store.Close()
	return store.Values(ctx)
}

// seedSettings writes the engine's settings for the run: the user's known
// settings minus every Google one, so nothing can reach the real sheet, with
// a cache namespace and a write token of its own (WM-TEST-4).
func seedSettings(ctx context.Context, path string, settings map[string]string, runID string, apiToken string) error {
	values := map[string]string{}
	for key, value := range settings {
		if _, known := domain.LookupSetting(key); known && !strings.HasPrefix(key, "GOOGLE_") {
			values[key] = value
		}
	}
	values["CACHE_KEY_PREFIX"] = "wm-test:" + runID + ":"
	values["LOG_LEVEL"] = "info"
	values[domain.APITokenSettingKey] = apiToken

	store, err := settingsAdapter.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(store.Save(ctx, values), store.Close())
}

// startEngine builds the engine and runs it on the workbook with its own
// cache namespace and write token (WM-TEST-2, WM-TEST-4).
func startEngine(ctx context.Context, settings map[string]string, runID string, workbook string, workDir string) (*harness, func() error, error) {
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
		apiToken: randomHex(32),
		client:   &http.Client{Timeout: requestTimeout},
	}
	settingsPath := filepath.Join(workDir, "settings.db")
	if err := seedSettings(ctx, settingsPath, settings, runID, h.apiToken); err != nil {
		return nil, nil, fmt.Errorf("seed the run settings: %w", err)
	}
	// Only bootstrap values: integration settings come from the run's store.
	environment := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"TMPDIR=" + os.TempDir(),
		"WM_SETTINGS_PATH=" + settingsPath,
		"CACHE_SQLITE_PATH=" + filepath.Join(workDir, "cache.db"),
		"WM_TEST_WORKBOOK=" + workbook,
		"HOST=127.0.0.1",
		"PORT=" + strconv.Itoa(port),
		"LOG_COLOR=false",
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
		t.Skip("set RUN_LIVE_API_TESTS=true to run live e2e tests against a local workbook export")
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
