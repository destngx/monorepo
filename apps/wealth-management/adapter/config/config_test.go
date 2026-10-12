package config

import (
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
