# Variable Definitions and Secrets

The full list of settings and bootstrap variables, with meanings and defaults, is in the [setup README §3](../README.md#3-settings-reference). It is derived from the registry in `apps/wealth-management/domain/settings.go`, `adapter/config/config.go`, `main.go` and the CLIs in `cmd/`; update it when those change.

## Rules

- **Source**: integration settings come only from the settings store (`settings.db` in the user config directory, or `WM_SETTINGS_PATH`), edited on the Settings page. The process environment supplies only the bootstrap variables; it is ignored for every other setting. There is no env file.
- **Secrets**: stored as plain text in a `0600` file in a `0700` directory, never returned by the API (only whether they are set). Never commit, print or log their values (WM-SEC-2).
- **Optional services degrade**: missing cache, AI, search or market settings disable only that feature; missing Google Sheets settings disable the sheet and AI routes. `/api/health` and `/api/ai/status` report what is missing.
- **Fixed conventions** (not configurable): base currency VND, timezone `Asia/Ho_Chi_Minh`. Cache TTLs follow REQUIREMENTS.md WM-RULE-10.
- **Test isolation**: live tests use a separate sheet copy and their own `CACHE_KEY_PREFIX`; see REQUIREMENTS.md §14.3.
