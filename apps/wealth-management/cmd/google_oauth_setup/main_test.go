package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestGivenTestFlagWhenSettingUpThenMintsSeparateDriveToken(t *testing.T) {
	production, test := setupFor(false), setupFor(true)
	if production.envKey != "GOOGLE_REFRESH_TOKEN" || len(production.scopes) != 1 || slices.Contains(production.scopes, "https://www.googleapis.com/auth/drive") {
		t.Fatalf("the production token must stay spreadsheets-only: %+v", production)
	}
	if test.envKey != "GOOGLE_TEST_REFRESH_TOKEN" || !slices.Contains(test.scopes, "https://www.googleapis.com/auth/drive") {
		t.Fatalf("the test token needs Drive to copy the sheet: %+v", test)
	}
}

func TestGivenEnvFileWhenUpsertingThenReplacesOrAppendsOneKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env.local")
	if err := os.WriteFile(path, []byte("A=1\nGOOGLE_TEST_REFRESH_TOKEN=old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := upsertEnvValue(path, "GOOGLE_TEST_REFRESH_TOKEN", "new"); err != nil {
		t.Fatal(err)
	}
	if err := upsertEnvValue(path, "B", "2"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "A=1\nGOOGLE_TEST_REFRESH_TOKEN=new\nB=2\n" {
		t.Fatalf("got %q", data)
	}
}
