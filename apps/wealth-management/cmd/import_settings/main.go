// Command import_settings copies the settings from existing env files (by
// default the repo root and app .env.local, the app file winning) into the
// settings store, so an old setup keeps working without the env file. It
// prints key names only, never values.
//
//	pnpx nx run wealth-management:import-settings [-- path/to/.env ...]
package main

import (
	configAdapter "apps/wealth-management/adapter/config"
	settingsAdapter "apps/wealth-management/adapter/settings"
	"apps/wealth-management/domain"
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/joho/godotenv"
)

var defaultFiles = []string{"../../.env.local", ".env.local"}

// bootstrapKeys stay in the process environment: the engine needs them
// before it can open the settings store.
var bootstrapKeys = []string{"HOST", "PORT", "API_TOKEN", "WM_SETTINGS_PATH", "CACHE_SQLITE_PATH", "WM_TEST_WORKBOOK", "LOG_COLOR"}

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "import settings: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	files, explicit := defaultFiles, len(args) > 0
	if explicit {
		files = args
	}
	plan, err := planImport(files, explicit)
	if err != nil {
		return err
	}
	if len(plan.update.Values) == 0 {
		fmt.Println("No settings to import.")
		return nil
	}

	path, err := settingsAdapter.ResolvePath(configAdapter.LoadSettingsPath())
	if err != nil {
		return err
	}
	store, err := settingsAdapter.Open(path)
	if err != nil {
		return err
	}
	defer store.Close()
	current, err := store.Values(ctx)
	if err != nil {
		return err
	}
	changes, _, err := domain.ApplySettingsUpdate(current, plan.update)
	if err != nil {
		return err
	}
	if err := store.Save(ctx, changes); err != nil {
		return err
	}

	fmt.Printf("Imported into %s: %s\n", path, strings.Join(sortedKeys(changes), ", "))
	if len(plan.bootstrap) > 0 {
		fmt.Printf("Not imported, set them in the shell instead: %s\n", strings.Join(plan.bootstrap, ", "))
	}
	if len(plan.unknown) > 0 {
		fmt.Printf("Not imported, the engine does not use them: %s\n", strings.Join(plan.unknown, ", "))
	}
	fmt.Println("Check them on the Settings page, restart the engine, then delete the env files.")
	return nil
}

type importPlan struct {
	update    domain.SettingsUpdate
	bootstrap []string
	unknown   []string
}

// planImport reads the files in order (later files win) and sorts their keys
// into settings to import, bootstrap keys and unknown keys. Empty values are
// skipped so an import never clears a stored setting. A missing default file
// is fine; a missing explicit file is an error.
func planImport(files []string, explicit bool) (importPlan, error) {
	merged := map[string]string{}
	for _, file := range files {
		values, err := godotenv.Read(file)
		if errors.Is(err, os.ErrNotExist) && !explicit {
			continue
		}
		if err != nil {
			return importPlan{}, fmt.Errorf("read %s: %w", file, err)
		}
		for key, value := range values {
			merged[key] = value
		}
	}

	plan := importPlan{update: domain.SettingsUpdate{Values: map[string]*string{}}}
	for _, key := range sortedKeys(merged) {
		value := strings.TrimSpace(merged[key])
		switch _, known := domain.LookupSetting(key); {
		case slices.Contains(bootstrapKeys, key):
			plan.bootstrap = append(plan.bootstrap, key)
		case !known:
			plan.unknown = append(plan.unknown, key)
		case value != "":
			plan.update.Values[key] = &value
		}
	}
	return plan, nil
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
