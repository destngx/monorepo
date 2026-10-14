package e2e

import (
	settingsAdapter "apps/wealth-management/adapter/settings"
	"apps/wealth-management/internal/e2esheet"
	"context"
	"maps"
	"path/filepath"
	"testing"
)

func TestGivenUserSettingsWhenSeedingTheRunThenTheEngineGetsTheCopyAndTestToken(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "settings.db")
	user := map[string]string{
		"GOOGLE_CLIENT_ID":          "id",
		"GOOGLE_SHEETS_ID":          "production",
		"GOOGLE_REFRESH_TOKEN":      "production-token",
		"GOOGLE_TEST_REFRESH_TOKEN": "test-token",
		"E2E_SOURCE_XLSX":           "~/book.xlsx",
		"LOG_LEVEL":                 "debug",
		"AI_MODEL":                  "gpt",
		"NOT_A_SETTING":             "x",
	}

	if err := seedSettings(ctx, path, user, "abcd", e2esheet.Copy{ID: "copy"}); err != nil {
		t.Fatal(err)
	}

	store, err := settingsAdapter.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	values, err := store.Values(ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"GOOGLE_CLIENT_ID":     "id",
		"GOOGLE_SHEETS_ID":     "copy",
		"GOOGLE_REFRESH_TOKEN": "test-token",
		"CACHE_KEY_PREFIX":     "wm-test:abcd:",
		"LOG_LEVEL":            "info",
		"AI_MODEL":             "gpt",
	}
	if !maps.Equal(values, want) {
		t.Fatalf("values = %v", values)
	}
}
