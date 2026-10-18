package e2e

import (
	settingsAdapter "apps/wealth-management/adapter/settings"
	"context"
	"maps"
	"path/filepath"
	"testing"
)

func TestGivenUserSettingsWhenSeedingTheRunThenGoogleSettingsAndTheUserTokenAreDropped(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "settings.db")
	user := map[string]string{
		"GOOGLE_CLIENT_ID":     "id",
		"GOOGLE_SHEETS_ID":     "production",
		"GOOGLE_REFRESH_TOKEN": "production-token",
		"LOG_LEVEL":            "debug",
		"AI_MODEL":             "gpt",
		"NOT_A_SETTING":        "x",
		"API_TOKEN":            "the-user-token",
	}

	if err := seedSettings(ctx, path, user, "abcd", "run-token"); err != nil {
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
		"CACHE_KEY_PREFIX": "wm-test:abcd:",
		"LOG_LEVEL":        "info",
		"AI_MODEL":         "gpt",
		"API_TOKEN":        "run-token",
	}
	if !maps.Equal(values, want) {
		t.Fatalf("values = %v", values)
	}
}
