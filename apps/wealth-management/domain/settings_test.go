package domain

import (
	"errors"
	"strings"
	"testing"
)

func strPtr(value string) *string { return &value }

func TestGivenTheRegistryWhenListingThenKeysAreUniqueAndGroupsKnown(t *testing.T) {
	groups := map[string]bool{}
	for _, group := range SettingGroups() {
		groups[group.ID] = true
	}
	seen := map[string]bool{}
	for _, definition := range SettingDefinitions() {
		if seen[definition.Key] {
			t.Fatalf("duplicate key %s", definition.Key)
		}
		seen[definition.Key] = true
		if !groups[definition.Group] {
			t.Fatalf("%s has unknown group %s", definition.Key, definition.Group)
		}
		if definition.Default != "" {
			if _, err := NormalizeSettingValue(definition, definition.Default); err != nil {
				t.Fatalf("%s default is invalid: %v", definition.Key, err)
			}
		}
	}
	for _, key := range []string{"AI_MODEL_INSIGHT", "AI_MODEL_SUGGEST", "AI_MODEL_VISION", "AI_MODEL_CHAT"} {
		if !seen[key] {
			t.Fatalf("missing per use case model %s", key)
		}
	}
	for _, bootstrap := range []string{"HOST", "PORT", "API_TOKEN", "WM_SETTINGS_PATH", "CACHE_SQLITE_PATH", "WM_TEST_WORKBOOK"} {
		if seen[bootstrap] {
			t.Fatalf("%s must not be editable through the generic settings save", bootstrap)
		}
	}
}

func TestGivenTheRegistryWhenRenderingThenEveryHintResolves(t *testing.T) {
	for _, definition := range SettingDefinitions() {
		if definition.Kind == SettingChoice && len(definition.Options) == 0 {
			t.Fatalf("%s is a choice without options", definition.Key)
		}
		if definition.Inherits != "" {
			if _, found := LookupSetting(definition.Inherits); !found || definition.Inherits == definition.Key {
				t.Fatalf("%s inherits unknown setting %s", definition.Key, definition.Inherits)
			}
		}
	}
	for _, key := range []string{"AI_MODEL", "AI_MODEL_CHAT", "AI_FALLBACK_MODEL"} {
		if definition, _ := LookupSetting(key); definition.Kind != SettingModel {
			t.Fatalf("%s should offer the endpoint's models, kind is %s", key, definition.Kind)
		}
	}
	if definition, _ := LookupSetting("AI_MODEL_VISION"); definition.Inherits != "AI_MODEL" {
		t.Fatalf("per use case models fall back to AI_MODEL, got %q", definition.Inherits)
	}
	for _, key := range []string{"GOOGLE_SHEETS_ID", "AI_BASE_URL", "AI_MODEL", "MARKET_DATA_SERVER_URL"} {
		if definition, _ := LookupSetting(key); definition.Advanced {
			t.Fatalf("%s is essential and must not hide under Advanced", key)
		}
	}
}

func TestGivenValuesWhenNormalizingThenKindsAreEnforced(t *testing.T) {
	cases := []struct {
		key, value, want string
		ok               bool
	}{
		{"SHEETS_READS_PER_MINUTE", " 120 ", "120", true},
		{"SHEETS_READS_PER_MINUTE", "0", "", false},
		{"SHEETS_READS_PER_MINUTE", "fast", "", false},
		{"AI_BASE_URL", "http://127.0.0.1:8081/v1", "http://127.0.0.1:8081/v1", true},
		{"AI_BASE_URL", "ftp://example", "", false},
		{"AI_BASE_URL", "example.com", "", false},
		{"LOG_LEVEL", "debug", "debug", true},
		{"LOG_LEVEL", "verbose", "", false},
		{"AI_MODEL", " gpt-5 ", "gpt-5", true},
		{"GOOGLE_SHEETS_ID", "abc\ndef", "", false},
		{"GOOGLE_SHEETS_ID", strings.Repeat("a", 5000), "", false},
		{"GOOGLE_SHEETS_ID", "   ", "", true},
	}
	for _, tc := range cases {
		definition, _ := LookupSetting(tc.key)
		got, err := NormalizeSettingValue(definition, tc.value)
		if tc.ok && (err != nil || got != tc.want) {
			t.Fatalf("%s=%q: got %q err %v", tc.key, tc.value, got, err)
		}
		if !tc.ok && !errors.Is(err, ErrValidation) {
			t.Fatalf("%s=%q: err %v, want ErrValidation", tc.key, tc.value, err)
		}
	}
}

func TestGivenAnUpdateWhenApplyingThenChangesAndMergedValuesAreReturned(t *testing.T) {
	current := map[string]string{"GOOGLE_SHEETS_ID": "old", "AI_MODEL": "gpt"}

	changes, merged, err := ApplySettingsUpdate(current, SettingsUpdate{Values: map[string]*string{
		"GOOGLE_SHEETS_ID": strPtr(" new "),
		"AI_MODEL":         nil,
		"TAVILY_API_KEY":   strPtr(""),
	}})

	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if changes["GOOGLE_SHEETS_ID"] != "new" || changes["AI_MODEL"] != "" || len(changes) != 3 {
		t.Fatalf("changes = %v", changes)
	}
	if merged["GOOGLE_SHEETS_ID"] != "new" || len(merged) != 1 {
		t.Fatalf("merged = %v", merged)
	}
	if current["GOOGLE_SHEETS_ID"] != "old" {
		t.Fatal("current must not be mutated")
	}
}

func TestGivenAnInvalidUpdateWhenApplyingThenItIsRejected(t *testing.T) {
	cases := map[string]SettingsUpdate{
		"empty":         {},
		"unknown key":   {Values: map[string]*string{"HOST": strPtr("0.0.0.0")}},
		"bad value":     {Values: map[string]*string{"AI_TIMEOUT_SECONDS": strPtr("-1")}},
		"half upstash":  {Values: map[string]*string{"UPSTASH_REDIS_REST_URL": strPtr("https://redis.example")}},
		"capture order": {Values: map[string]*string{"CAPTURE_MAX_ITEM_MB": strPtr("30")}},
	}
	for name, update := range cases {
		if _, _, err := ApplySettingsUpdate(map[string]string{}, update); !errors.Is(err, ErrValidation) {
			t.Fatalf("%s: err %v, want ErrValidation", name, err)
		}
	}
}

func TestGivenStoredSecretsWhenBuildingTheViewThenTheirValuesAreNeverIncluded(t *testing.T) {
	view := BuildSettingsView(map[string]string{"AI_API_KEY": "sk-secret", "AI_MODEL": "gpt"}, "/p/settings.db", true)

	if view.Path != "/p/settings.db" || !view.RestartRequired || len(view.Groups) == 0 {
		t.Fatalf("view = %+v", view)
	}
	for _, setting := range view.Settings {
		switch setting.Key {
		case "AI_API_KEY":
			if !setting.Secret || !setting.IsSet || setting.Value != "" {
				t.Fatalf("secret view = %+v", setting)
			}
		case "AI_MODEL":
			if setting.Value != "gpt" || !setting.IsSet {
				t.Fatalf("model view = %+v", setting)
			}
		case "EXA_API_KEY":
			if setting.IsSet {
				t.Fatalf("unset secret reported set")
			}
		}
	}
}
