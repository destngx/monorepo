package config

import (
	"apps/wealth-management/domain"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadServerConfig(t *testing.T) {
	cases := []struct {
		name, host, port, token string
		wantHost                string
		wantPort                int
		wantErr                 bool
	}{
		{name: "defaults", wantHost: "127.0.0.1", wantPort: 8080},
		{name: "overrides", host: "0.0.0.0", port: "9090", token: "t", wantHost: "0.0.0.0", wantPort: 9090},
		{name: "bad port", port: "http", wantErr: true},
		{name: "port out of range", port: "70000", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HOST", tc.host)
			t.Setenv("PORT", tc.port)
			t.Setenv("API_TOKEN", tc.token)
			config, err := LoadServerConfig()
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidServerConfig) {
					t.Fatalf("err = %v", err)
				}
				return
			}
			if err != nil || config.Host != tc.wantHost || config.Port != tc.wantPort || config.APIToken != tc.token {
				t.Fatalf("config = %+v, err = %v", config, err)
			}
		})
	}
}

func TestBudgetSheetTemplate(t *testing.T) {
	if BudgetSheetTemplate(Values{}) != "Budget_{year}" {
		t.Fatal("default template")
	}
	if BudgetSheetTemplate(Values{"BUDGET_SHEET_TEMPLATE": "Ngân sách {year}"}) != "Ngân sách {year}" {
		t.Fatal("override template")
	}
}

func TestGivenProcessEnvWhenLoadingIntegrationSettingsThenItIsIgnored(t *testing.T) {
	t.Setenv("AI_BASE_URL", "http://from-shell/v1")
	t.Setenv("AI_MODEL", "from-shell")
	t.Setenv("GOOGLE_SHEETS_ID", "from-shell")

	if _, err := LoadAIConfig(Values{}); !errors.Is(err, ErrMissingAIConfig) {
		t.Fatalf("AI must not read the environment, got %v", err)
	}
	if _, err := LoadSheetsConfig(Values{}); !errors.Is(err, ErrMissingSheetsConfig) {
		t.Fatalf("sheets must not read the environment, got %v", err)
	}
}

func TestLoadAIConfig(t *testing.T) {
	if _, err := LoadAIConfig(Values{}); !errors.Is(err, ErrMissingAIConfig) {
		t.Fatalf("unconfigured AI must report missing config, got %v", err)
	}

	config, err := LoadAIConfig(Values{
		"AI_BASE_URL":      "http://127.0.0.1:8081/v1",
		"AI_MODEL":         "gpt-main",
		"AI_MODEL_SUGGEST": "gpt-mini",
		"AI_PII_NAMES":     " Nguyen Van A , ,B",
	})
	if err != nil {
		t.Fatal(err)
	}
	if config.Models[domain.AIUseCaseSuggest] != "gpt-mini" || config.Models[domain.AIUseCaseChat] != "gpt-main" {
		t.Fatalf("unexpected models %v", config.Models)
	}
	if len(config.PIINames) != 2 || config.Timeout.Seconds() != 90 {
		t.Fatalf("unexpected config %+v", config)
	}
}

func TestLoadCaptureLimits(t *testing.T) {
	limits, err := LoadCaptureLimits(Values{})
	if err != nil || limits != domain.DefaultCaptureLimits() {
		t.Fatalf("expected defaults, got %+v %v", limits, err)
	}

	limits, err = LoadCaptureLimits(Values{"CAPTURE_MAX_ITEMS": "8", "CAPTURE_MAX_ITEM_MB": "2", "CAPTURE_MAX_BATCH_MB": "10"})
	if err != nil || limits != (domain.CaptureLimits{MaxItems: 8, MaxItemBytes: 2 << 20, MaxBatchBytes: 10 << 20}) {
		t.Fatalf("unexpected limits %+v %v", limits, err)
	}

	for key, value := range map[string]string{"CAPTURE_MAX_ITEMS": "0", "CAPTURE_MAX_ITEM_MB": "abc", "CAPTURE_MAX_BATCH_MB": "1"} {
		if _, err := LoadCaptureLimits(Values{key: value}); !errors.Is(err, ErrInvalidCaptureConfig) {
			t.Fatalf("%s=%s must be rejected, got %v", key, value, err)
		}
	}
}

func TestLoadSearchConfigTrimsKeys(t *testing.T) {
	if config := LoadSearchConfig(Values{"TAVILY_API_KEY": " tvly "}); config.TavilyAPIKey != "tvly" || config.ExaAPIKey != "" {
		t.Fatalf("unexpected config %+v", config)
	}
}

func TestLoadLogging(t *testing.T) {
	if LoadLogLevel(Values{}) != "info" || LoadLogLevel(Values{"LOG_LEVEL": "debug"}) != "debug" {
		t.Fatal("log level")
	}
	t.Setenv("LOG_COLOR", "false")
	if LoadLogColor() {
		t.Fatal("LOG_COLOR=false must disable colour")
	}
}

func TestLoadSettingsPath(t *testing.T) {
	t.Setenv("WM_SETTINGS_PATH", " /tmp/wm/settings.db ")
	if LoadSettingsPath() != "/tmp/wm/settings.db" {
		t.Fatalf("path = %q", LoadSettingsPath())
	}
}

func TestGivenATestWorkbookWhenLoadingThenHomeIsExpanded(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	for value, want := range map[string]string{
		"":                           "",
		" /data/book.xlsx ":          "/data/book.xlsx",
		"~/projects/Money Book.xlsx": filepath.Join(home, "projects", "Money Book.xlsx"),
	} {
		t.Setenv("WM_TEST_WORKBOOK", value)
		if got, err := LoadTestWorkbook(); err != nil || got != want {
			t.Fatalf("WM_TEST_WORKBOOK=%q: got %q, %v", value, got, err)
		}
	}
}

func sheetsValues(extra Values) Values {
	values := Values{"GOOGLE_CLIENT_ID": "id", "GOOGLE_CLIENT_SECRET": "secret", "GOOGLE_REFRESH_TOKEN": "token", "GOOGLE_SHEETS_ID": "sheet"}
	for key, value := range extra {
		values[key] = value
	}
	return values
}

func TestGivenNoReadQuotaWhenLoadingSheetsConfigThenItDefaultsToSixtyPerMinute(t *testing.T) {
	config, err := LoadSheetsConfig(sheetsValues(nil))

	if err != nil || config.ReadsPerMinute != 60 {
		t.Fatalf("config = %+v err = %v", config.ReadsPerMinute, err)
	}
}

func TestGivenAReadQuotaWhenLoadingSheetsConfigThenItIsUsed(t *testing.T) {
	config, err := LoadSheetsConfig(sheetsValues(Values{"SHEETS_READS_PER_MINUTE": "300"}))

	if err != nil || config.ReadsPerMinute != 300 {
		t.Fatalf("config = %+v err = %v", config.ReadsPerMinute, err)
	}
}

func TestGivenAnInvalidReadQuotaWhenLoadingSheetsConfigThenItFails(t *testing.T) {
	for _, value := range []string{"0", "-5", "fast"} {
		if _, err := LoadSheetsConfig(sheetsValues(Values{"SHEETS_READS_PER_MINUTE": value})); !errors.Is(err, ErrInvalidSheetsConfig) {
			t.Fatalf("value %q: err = %v, want ErrInvalidSheetsConfig", value, err)
		}
	}
}

func TestGivenUpstashCredentialsWhenLoadingCacheConfigThenUpstashIsUsed(t *testing.T) {
	config, err := LoadCacheConfig(Values{"UPSTASH_REDIS_REST_URL": "https://redis.example", "UPSTASH_REDIS_REST_TOKEN": "token"})

	if err != nil || config.Backend != domain.CacheBackendUpstash || config.RESTURL != "https://redis.example" || config.KeyPrefix != "wm:" {
		t.Fatalf("config = %+v err = %v", config, err)
	}
}

func TestGivenNoUpstashWhenLoadingCacheConfigThenSQLiteInTheSystemTempDirIsUsed(t *testing.T) {
	t.Setenv("CACHE_SQLITE_PATH", "")

	config, err := LoadCacheConfig(Values{"CACHE_KEY_PREFIX": "wm-test:1:"})

	want := filepath.Join(os.TempDir(), "wealth-management", "cache.db")
	if err != nil || config.Backend != domain.CacheBackendSQLite || config.SQLitePath != want || config.KeyPrefix != "wm-test:1:" {
		t.Fatalf("config = %+v err = %v, want sqlite at %s", config, err, want)
	}
}

func TestGivenASQLitePathInTheEnvironmentWhenLoadingCacheConfigThenItIsUsed(t *testing.T) {
	t.Setenv("CACHE_SQLITE_PATH", "/var/cache/wm.db")

	config, err := LoadCacheConfig(Values{})

	if err != nil || config.SQLitePath != "/var/cache/wm.db" {
		t.Fatalf("config = %+v err = %v", config, err)
	}
}

func TestGivenHalfTheUpstashCredentialsWhenLoadingCacheConfigThenItFails(t *testing.T) {
	for _, values := range []Values{{"UPSTASH_REDIS_REST_URL": "https://redis.example"}, {"UPSTASH_REDIS_REST_TOKEN": "token"}} {
		if _, err := LoadCacheConfig(values); !errors.Is(err, ErrMissingCacheConfig) {
			t.Fatalf("values %v: err = %v, want ErrMissingCacheConfig", values, err)
		}
	}
}
