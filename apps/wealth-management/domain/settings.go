package domain

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// Engine settings are edited in the UI and stored by the engine; there is
// no env file. Only bootstrap values the engine needs before it can open the
// store (HOST, PORT, API_TOKEN, WM_SETTINGS_PATH, CACHE_SQLITE_PATH,
// WM_TEST_MODE, LOG_COLOR) stay process environment. Keys keep the former
// variable names so docs, errors and the import tool line up.

// SettingKind decides how a value is validated and edited.
type SettingKind string

const (
	SettingText    SettingKind = "text"
	SettingSecret  SettingKind = "secret"
	SettingInteger SettingKind = "integer"
	SettingURL     SettingKind = "url"
	SettingChoice  SettingKind = "choice"
	SettingPath    SettingKind = "path"
)

// SettingGroup is one card in the Settings UI.
type SettingGroup struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// SettingDefinition describes one editable setting. Default is the value
// the engine uses when the setting is empty; empty means none (or the
// provider's own default, as the description says).
type SettingDefinition struct {
	Key         string      `json:"key"`
	Group       string      `json:"group"`
	Label       string      `json:"label"`
	Description string      `json:"description"`
	Kind        SettingKind `json:"kind"`
	Default     string      `json:"default,omitempty"`
	Options     []string    `json:"options,omitempty"`
}

// Secret settings are write-only: the API reports whether they are set,
// never their value.
func (d SettingDefinition) IsSecret() bool { return d.Kind == SettingSecret }

// SettingView is a definition with its stored state. Value is omitted for
// secrets.
type SettingView struct {
	SettingDefinition
	Secret bool   `json:"secret"`
	IsSet  bool   `json:"isSet"`
	Value  string `json:"value,omitempty"`
}

// SettingsView is the response of the settings API.
type SettingsView struct {
	Groups   []SettingGroup `json:"groups"`
	Settings []SettingView  `json:"settings"`
	// Path is where the settings are stored.
	Path string `json:"path"`
	// RestartRequired is true once a change was saved and the running engine
	// does not use it yet.
	RestartRequired bool `json:"restartRequired"`
}

// SettingsUpdate maps keys to new values; nil or an empty string clears a
// setting.
type SettingsUpdate struct {
	Values map[string]*string `json:"values"`
}

const maxSettingLength = 4096

var settingGroups = []SettingGroup{
	{ID: "sheets", Label: "Google Sheets", Description: "The spreadsheet that is the system of record. Save the OAuth client, then use Connect Google account to mint the refresh token."},
	{ID: "cache", Label: "Cache", Description: "Upstash Redis when both URL and token are set; otherwise a local SQLite cache."},
	{ID: "ai", Label: "AI provider", Description: "OpenAI-compatible endpoint, e.g. ai-gateway. AI features need a base URL and a model."},
	{ID: "search", Label: "Web search", Description: "Optional keys; DuckDuckGo is always the last fallback."},
	{ID: "market", Label: "Market data", Description: "Provider endpoints; empty uses each provider's public default."},
	{ID: "capture", Label: "Smart Capture", Description: "Limits for one batch of screenshots and text."},
	{ID: "general", Label: "General", Description: "Logging and card programs."},
	{ID: "testing", Label: "Live tests", Description: "Used only by the e2e harness, never by the engine."},
}

func settingDefinitions() []SettingDefinition {
	definitions := []SettingDefinition{
		{Key: "GOOGLE_CLIENT_ID", Group: "sheets", Label: "OAuth client ID", Kind: SettingText, Description: "Google Cloud OAuth client (Desktop app)."},
		{Key: "GOOGLE_CLIENT_SECRET", Group: "sheets", Label: "OAuth client secret", Kind: SettingSecret},
		{Key: "GOOGLE_REFRESH_TOKEN", Group: "sheets", Label: "Refresh token", Kind: SettingSecret, Description: "Written by Connect Google account, or paste one."},
		{Key: "GOOGLE_SHEETS_ID", Group: "sheets", Label: "Spreadsheet ID", Kind: SettingText, Description: "The id in the sheet URL, between /d/ and /edit."},
		{Key: "GOOGLE_REDIRECT_URL", Group: "sheets", Label: "OAuth redirect URL", Kind: SettingURL, Description: "Where Google returns after sign-in. Empty uses /api/auth/google/callback on the loopback address the UI is open on; a Web OAuth client must list it."},
		{Key: "SHEETS_READS_PER_MINUTE", Group: "sheets", Label: "Reads per minute", Kind: SettingInteger, Default: "60", Description: "Pace of Sheets read requests (WM-NFR-3)."},
		{Key: "BUDGET_SHEET_TEMPLATE", Group: "sheets", Label: "Budget tab name", Kind: SettingText, Default: "Budget_{year}", Description: "{year} is replaced with the requested year."},

		{Key: "UPSTASH_REDIS_REST_URL", Group: "cache", Label: "Upstash REST URL", Kind: SettingURL},
		{Key: "UPSTASH_REDIS_REST_TOKEN", Group: "cache", Label: "Upstash REST token", Kind: SettingSecret},
		{Key: "CACHE_KEY_PREFIX", Group: "cache", Label: "Key prefix", Kind: SettingText, Default: "wm:", Description: "Namespaces every key so environments never share keys (WM-OPS-2b)."},

		{Key: "AI_BASE_URL", Group: "ai", Label: "Base URL", Kind: SettingURL, Description: "e.g. http://127.0.0.1:8081/v1 for ai-gateway."},
		{Key: "AI_API_KEY", Group: "ai", Label: "API key", Kind: SettingSecret, Description: "Only if the endpoint requires one."},
		{Key: "AI_PROVIDER", Group: "ai", Label: "Provider header", Kind: SettingText, Description: "X-AI-Provider sent to ai-gateway."},
		{Key: "AI_MODEL", Group: "ai", Label: "Default model", Kind: SettingText, Description: "Used by every use case without its own model."},
	}
	for _, useCase := range AIUseCases {
		name := string(useCase)
		definitions = append(definitions, SettingDefinition{
			Key: "AI_MODEL_" + strings.ToUpper(name), Group: "ai", Label: strings.ToUpper(name[:1]) + name[1:] + " model", Kind: SettingText,
			Description: "Overrides the default model for " + name + ".",
		})
	}
	definitions = append(definitions, []SettingDefinition{
		{Key: "AI_FALLBACK_MODEL", Group: "ai", Label: "Fallback model", Kind: SettingText, Description: "Tried once when the main model fails."},
		{Key: "AI_TIMEOUT_SECONDS", Group: "ai", Label: "Timeout (seconds)", Kind: SettingInteger, Default: "90"},
		{Key: "AI_PII_NAMES", Group: "ai", Label: "Names to mask", Kind: SettingText, Description: "Comma separated; masked before text reaches the provider."},

		{Key: "TAVILY_API_KEY", Group: "search", Label: "Tavily API key", Kind: SettingSecret},
		{Key: "EXA_API_KEY", Group: "search", Label: "Exa API key", Kind: SettingSecret},

		{Key: "FMARKET_BASE_URL", Group: "market", Label: "Fmarket URL", Kind: SettingURL, Default: "https://api.fmarket.vn"},
		{Key: "VNSTOCK_SERVER_URL", Group: "market", Label: "vnstock-server URL", Kind: SettingURL, Default: "http://localhost:8000"},
		{Key: "COINGECKO_BASE_URL", Group: "market", Label: "CoinGecko URL", Kind: SettingURL},
		{Key: "COINGECKO_API_KEY", Group: "market", Label: "CoinGecko demo key", Kind: SettingSecret, Description: "Raises the public rate limit."},
		{Key: "CURRENCY_API_BASE_URL", Group: "market", Label: "currency-api URL", Kind: SettingURL},
		{Key: "YAHOO_FINANCE_BASE_URL", Group: "market", Label: "Yahoo Finance URL", Kind: SettingURL},

		{Key: "CAPTURE_MAX_ITEMS", Group: "capture", Label: "Max items per batch", Kind: SettingInteger, Default: strconv.Itoa(DefaultCaptureLimits().MaxItems)},
		{Key: "CAPTURE_MAX_ITEM_MB", Group: "capture", Label: "Max item size (MB)", Kind: SettingInteger, Default: strconv.Itoa(DefaultCaptureLimits().MaxItemBytes >> 20)},
		{Key: "CAPTURE_MAX_BATCH_MB", Group: "capture", Label: "Max batch size (MB)", Kind: SettingInteger, Default: strconv.Itoa(DefaultCaptureLimits().MaxBatchBytes >> 20)},

		{Key: "LOG_LEVEL", Group: "general", Label: "Log level", Kind: SettingChoice, Default: "info", Options: []string{"debug", "info", "warn", "error"}},
		{Key: "CARD_PROGRAMS_FILE", Group: "general", Label: "Card programs file", Kind: SettingPath, Description: "JSON file replacing the built-in card programs."},

		{Key: "GOOGLE_TEST_REFRESH_TOKEN", Group: "testing", Label: "Test refresh token", Kind: SettingSecret, Description: "Spreadsheets + Drive scopes; written by Connect test account."},
		{Key: "E2E_SOURCE_XLSX", Group: "testing", Label: "Source workbook", Kind: SettingPath, Description: "Local .xlsx export uploaded as the disposable test sheet."},
	}...)
	return definitions
}

var definitionsByKey = func() map[string]SettingDefinition {
	byKey := map[string]SettingDefinition{}
	for _, definition := range settingDefinitions() {
		byKey[definition.Key] = definition
	}
	return byKey
}()

// SettingGroups returns the groups in display order.
func SettingGroups() []SettingGroup { return slices.Clone(settingGroups) }

// SettingDefinitions returns every editable setting in display order.
func SettingDefinitions() []SettingDefinition { return settingDefinitions() }

// LookupSetting returns the definition of key.
func LookupSetting(key string) (SettingDefinition, bool) {
	definition, found := definitionsByKey[key]
	return definition, found
}

// NormalizeSettingValue trims value and checks it against the definition.
func NormalizeSettingValue(definition SettingDefinition, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if len(value) > maxSettingLength {
		return "", fmt.Errorf("%w: %s is longer than %d characters", ErrValidation, definition.Key, maxSettingLength)
	}
	if strings.ContainsAny(value, "\r\n\x00") {
		return "", fmt.Errorf("%w: %s must be a single line", ErrValidation, definition.Key)
	}
	switch definition.Kind {
	case SettingInteger:
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			return "", fmt.Errorf("%w: %s must be a positive integer", ErrValidation, definition.Key)
		}
		return strconv.Itoa(parsed), nil
	case SettingURL:
		parsed, err := url.Parse(value)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return "", fmt.Errorf("%w: %s must be an http(s) URL", ErrValidation, definition.Key)
		}
	case SettingChoice:
		if !slices.Contains(definition.Options, value) {
			return "", fmt.Errorf("%w: %s must be one of %s", ErrValidation, definition.Key, strings.Join(definition.Options, ", "))
		}
	}
	return value, nil
}

// ApplySettingsUpdate validates update against the definitions and returns
// the normalised changes (an empty value clears the key) and the values the
// engine would run with afterwards, checked across settings.
func ApplySettingsUpdate(current map[string]string, update SettingsUpdate) (map[string]string, map[string]string, error) {
	if len(update.Values) == 0 {
		return nil, nil, fmt.Errorf("%w: no settings to change", ErrValidation)
	}
	changes := make(map[string]string, len(update.Values))
	merged := make(map[string]string, len(current)+len(update.Values))
	for key, value := range current {
		merged[key] = value
	}
	for key, value := range update.Values {
		definition, found := LookupSetting(key)
		if !found {
			return nil, nil, fmt.Errorf("%w: unknown setting %s", ErrValidation, key)
		}
		normalized := ""
		if value != nil {
			var err error
			if normalized, err = NormalizeSettingValue(definition, *value); err != nil {
				return nil, nil, err
			}
		}
		changes[key] = normalized
		if normalized == "" {
			delete(merged, key)
		} else {
			merged[key] = normalized
		}
	}
	if err := ValidateSettings(merged); err != nil {
		return nil, nil, err
	}
	return changes, merged, nil
}

// ValidateSettings checks rules that span settings.
func ValidateSettings(values map[string]string) error {
	if (values["UPSTASH_REDIS_REST_URL"] == "") != (values["UPSTASH_REDIS_REST_TOKEN"] == "") {
		return fmt.Errorf("%w: set both UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN, or neither for the local cache", ErrValidation)
	}
	itemMB, batchMB := settingInt(values, "CAPTURE_MAX_ITEM_MB"), settingInt(values, "CAPTURE_MAX_BATCH_MB")
	if itemMB > batchMB {
		return fmt.Errorf("%w: CAPTURE_MAX_ITEM_MB must not exceed CAPTURE_MAX_BATCH_MB", ErrValidation)
	}
	return nil
}

// settingInt reads an integer setting, falling back to its default.
func settingInt(values map[string]string, key string) int {
	value := values[key]
	if value == "" {
		value = definitionsByKey[key].Default
	}
	parsed, _ := strconv.Atoi(value)
	return parsed
}

// BuildSettingsView lists every definition with its stored state.
func BuildSettingsView(values map[string]string, path string, restartRequired bool) SettingsView {
	definitions := settingDefinitions()
	views := make([]SettingView, 0, len(definitions))
	for _, definition := range definitions {
		value := values[definition.Key]
		view := SettingView{SettingDefinition: definition, Secret: definition.IsSecret(), IsSet: value != ""}
		if !definition.IsSecret() {
			view.Value = value
		}
		views = append(views, view)
	}
	return SettingsView{Groups: SettingGroups(), Settings: views, Path: path, RestartRequired: restartRequired}
}
