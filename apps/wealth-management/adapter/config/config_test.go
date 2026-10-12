package config

import (
	"apps/wealth-management/domain"
	"errors"
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
	t.Setenv("BUDGET_SHEET_TEMPLATE", "")
	if BudgetSheetTemplate() != "Budget_{year}" {
		t.Fatal("default template")
	}
	t.Setenv("BUDGET_SHEET_TEMPLATE", "Ngân sách {year}")
	if BudgetSheetTemplate() != "Ngân sách {year}" {
		t.Fatal("override template")
	}
}

func TestLoadAIConfig(t *testing.T) {
	for _, key := range []string{"AI_BASE_URL", "AI_API_KEY", "AI_PROVIDER", "AI_MODEL", "AI_MODEL_INSIGHT", "AI_MODEL_SUGGEST", "AI_MODEL_VISION", "AI_MODEL_CHAT", "AI_FALLBACK_MODEL", "AI_TIMEOUT_SECONDS", "AI_PII_NAMES"} {
		t.Setenv(key, "")
	}
	if _, err := LoadAIConfig(); !errors.Is(err, ErrMissingAIConfig) {
		t.Fatalf("unconfigured AI must report missing config, got %v", err)
	}

	t.Setenv("AI_BASE_URL", "http://127.0.0.1:8081/v1")
	t.Setenv("AI_MODEL", "gpt-main")
	t.Setenv("AI_MODEL_SUGGEST", "gpt-mini")
	t.Setenv("AI_PII_NAMES", " Nguyen Van A , ,B")
	config, err := LoadAIConfig()
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
