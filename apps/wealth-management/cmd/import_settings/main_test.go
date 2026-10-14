package main

import (
	settingsAdapter "apps/wealth-management/adapter/settings"
	"context"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestGivenEnvFilesWhenPlanningThenLaterFilesWinAndKeysAreSorted(t *testing.T) {
	dir := t.TempDir()
	root := writeFile(t, dir, "root.env", "AI_MODEL=old\nGOOGLE_SHEETS_ID=sheet\nPORT=9000\nGITHUB_TOKEN=x\nTAVILY_API_KEY=\n")
	app := writeFile(t, dir, "app.env", "AI_MODEL=new\n")

	plan, err := planImport([]string{root, filepath.Join(dir, "missing.env"), app}, false)

	if err != nil {
		t.Fatal(err)
	}
	if got := slices.Sorted(maps.Keys(plan.update.Values)); !slices.Equal(got, []string{"AI_MODEL", "GOOGLE_SHEETS_ID"}) {
		t.Fatalf("imported %v", got)
	}
	if *plan.update.Values["AI_MODEL"] != "new" {
		t.Fatalf("AI_MODEL = %s, want the later file to win", *plan.update.Values["AI_MODEL"])
	}
	if !slices.Equal(plan.bootstrap, []string{"PORT"}) || !slices.Equal(plan.unknown, []string{"GITHUB_TOKEN"}) {
		t.Fatalf("bootstrap %v unknown %v", plan.bootstrap, plan.unknown)
	}
}

func TestGivenAMissingExplicitFileWhenPlanningThenItFails(t *testing.T) {
	if _, err := planImport([]string{filepath.Join(t.TempDir(), "missing.env")}, true); err == nil {
		t.Fatal("want an error for a missing explicit file")
	}
}

func TestGivenAnEnvFileWhenImportingThenTheStoreKeepsOtherSettings(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "settings.db")
	t.Setenv("WM_SETTINGS_PATH", storePath)
	store, err := settingsAdapter.Open(storePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.Save(ctx, map[string]string{"EXA_API_KEY": "kept"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	if err := run(ctx, []string{writeFile(t, dir, ".env", "AI_MODEL=gpt\n")}); err != nil {
		t.Fatal(err)
	}

	store, err = settingsAdapter.Open(storePath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	values, err := store.Values(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(values, map[string]string{"EXA_API_KEY": "kept", "AI_MODEL": "gpt"}) {
		t.Fatalf("values = %v", values)
	}
}

func TestGivenAnInvalidValueWhenImportingThenNothingIsSaved(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WM_SETTINGS_PATH", filepath.Join(dir, "settings.db"))

	if err := run(context.Background(), []string{writeFile(t, dir, ".env", "AI_MODEL=gpt\nAI_TIMEOUT_SECONDS=soon\n")}); err == nil {
		t.Fatal("want a validation error")
	}
}
