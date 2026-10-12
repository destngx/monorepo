# Variable Definitions and Secrets

The full list of environment variables, with meanings and defaults, is in the [setup README §3](../README.md#3-environment-variables). It is derived from `apps/wealth-management/adapter/config/config.go`, `adapter/config/cards.go`, `main.go` and the CLIs in `cmd/`; update it when those change.

## Rules

- **Source**: `config.Load` reads the repo root `.env.local`, then `apps/wealth-management/.env.local`; values in these files override the shell environment.
- **Secrets**: `.env.local` is gitignored runtime input. Never commit, print or log its values (WM-SEC-2).
- **Optional services degrade**: missing cache, AI, search or market variables disable only that feature; missing Google Sheets variables disable the sheet and AI routes. `/api/health` and `/api/ai/status` report what is missing.
- **Fixed conventions** (not configurable): base currency VND, timezone `Asia/Ho_Chi_Minh`. Cache TTLs follow REQUIREMENTS.md WM-RULE-10.
- **Test isolation**: live tests use a separate sheet copy and their own `CACHE_KEY_PREFIX`; see REQUIREMENTS.md §14.3.
