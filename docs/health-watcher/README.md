# Health Watcher

A family's health records in one place: scans, PDFs and photos of results from hospitals, clinics and dentists, filed by person and visit, searchable later. What the app must do is in [REQUIREMENTS.md](./REQUIREMENTS.md); this page covers running it.

Phases 1 to 3 are built: accounts, families (tenants) with roles and invites, people, visits, the inbox, reading each upload with mac-ocr on the Mac and then with AI on masked pages, drafts that a person approves, search, the document viewer and the metric catalog. Charts and the AI review come in later phases ([REQUIREMENTS.md §13](./REQUIREMENTS.md#13-roadmap)).

## 1. Run it

```sh
pnpm exec nx run health-watcher:serve   # engine on :9191 and the UI dev server on :5174
# The engine restarts when Go code or a migration changes (air, .air.toml); the UI reloads through vite.
# The dev server proxies /api to http://127.0.0.1:9191; set HW_DEV_ENGINE when the engine listens elsewhere.
pnpm exec nx run health-watcher:start   # production: builds the UI, the engine serves it at http://127.0.0.1:9191/ui
```

Reading needs the mac-ocr binary with schema 1.2 (`--render-dir`): `pnpm exec nx build mac-ocr` writes `apps/mac-ocr/bin/mac-ocr`, the engine's default. After setup, turn AI reading on by setting the AI address in Engine settings (section 2). An older build still reads text, but its pages have no images, so nothing is sent to the AI provider until it is rebuilt.

On the first start the engine writes a one-time setup code to `setup-code` in the data folder and logs that file's path. The UI shows a setup page that asks for the code, then creates the first account and family. Only someone who can read the Mac's data folder can do setup.

## 2. Engine settings

Settings live in the database, not in environment variables. An engine admin edits them at **Settings → Engine settings** (`/ui/settings/engine`). The account that ran first-time setup is the first admin and can make other accounts admins; one admin always remains. The settings apply to every family on the Mac.

| Setting                 | Default                                  | Applies           | Meaning                                                                                                                              |
| ----------------------- | ---------------------------------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| AI address              | empty                                    | at once           | OpenAI Responses API base, e.g. `http://localhost:8080/v1` for ai-gateway. Empty turns AI reading off; OCR and search still work.    |
| API key                 | empty                                    | at once           | Bearer token for the provider, if it needs one. Never shown again after saving.                                                      |
| Time limit per reading  | 180 s                                    | at once           | 5 to 1800 seconds.                                                                                                                   |
| mac-ocr program         | `../mac-ocr/bin/mac-ocr` from the engine | at once           | A full path to a program named `mac-ocr`; empty uses the build next to the engine.                                                   |
| Background jobs at once | 2                                        | at once           | OCR and AI readings, 1 to 8.                                                                                                         |
| Largest file            | 50 MB                                    | at once           | Up to 1024 MB.                                                                                                                       |
| Largest upload          | 200 MB                                   | restart if raised | Up to 4096 MB, at least the largest file.                                                                                            |
| Host                    | `127.0.0.1`                              | restart           | An address of this Mac. Use its Tailscale address to reach it from a phone; the engine also answers on `127.0.0.1` on the same port. |
| Port                    | `9191`                                   | restart           | 1024 to 65535.                                                                                                                       |

Items read while AI reading was off are sent as soon as an admin sets the AI address; no _Retry_ is needed. With AI on, the menu bar shows the quota left on a two-ring gauge (outer ring weekly, inner ring 5 hours); hover it for the percentages and reset times.

The model is always `gpt-6-luna` at low reasoning effort (D6); it is not a setting. _Check connection_ sends a small request with the saved AI settings and no health data.

If the engine cannot listen on the saved host and port for about 30 seconds, for example because Tailscale is down, it starts on `127.0.0.1:9191` and logs why; if only the Tailscale address fails, it keeps answering on `127.0.0.1`, so the page stays reachable on the Mac. A saved value that fails the checks, for example after editing the database by hand, keeps its default; the page names it until the next save.

The only environment variable left is `HW_DATA_DIR`, the data folder (default `~/Library/Application Support/health-watcher`), because the database lives in it. Upgrading from the environment-variable setup: on the first start with an empty settings table, the engine imports the retired variables it finds (`HOST`, `PORT`, `AI_BASE_URL`, `AI_API_KEY` and the rest) once. After that it logs a warning for each one still set; they do nothing.

## 3. Data

Everything stays on the Mac, in the data folder (`HW_DATA_DIR`):

- `health.db`: SQLite, mode `0600`. It holds the engine settings too, the AI API key included. Schema changes are numbered SQL files in `adapter/sqlite/migrations`, applied at start.
- `files/ab/cd/<sha256>`: originals, named by their SHA-256, mode `0600`. The same file uploaded twice is stored once. Rendered page images and their masked copies live here too.
- `work/`: scratch space for mac-ocr, emptied after each file.

Encryption at rest relies on FileVault (D1). Back up both the database and `files/` together.

## 4. Accounts and families

- A **tenant** is a family or group. A login can belong to several, and switches between them in the header.
- Roles: **owner** (members, invites, settings), **editor** (adds and files records), **viewer** (reads only).
- A **person** is whoever the records are about. A child or a parent needs no login.
- Invite links are one-time and expire after 7 days (Settings, Invite someone).
- Forgotten password: run this on the Mac, then type the new password twice:

  ```sh
  pnpm exec nx run health-watcher:reset-password -- --email someone@example.com
  ```

## 5. Getting papers in

- **Browser**: Inbox page. Drop files or a whole folder, or pick them. On a phone, _Take photos_ opens the camera; take each page, then tick _pages of one document_.
- **iPhone camera from the Mac**: Inbox, _Take photos with iPhone_ shows a QR code. Point the iPhone's Camera app at it, take each page on the page that opens, then press Send. The link needs no sign-in, adds only to that inbox, and expires after 15 minutes. The iPhone reaches the Mac over Tailscale: set Host in Engine settings to the Mac's Tailscale address, or open the app on the Mac at that address. The link uses the port the Mac's browser is on, so it opens the UI dev server (`:5174`) during `serve` and the engine (`:9191`) with `start`.
- Files waiting to be sent are kept in the browser until sent, so a page that reloads while the camera is open (iOS does this) loses nothing. On the phone, use the engine's own address (`:9191/ui`, from `start`) rather than the dev server on `:5174`, which reloads the page whenever its connection drops.
- **iPhone share sheet**: a Shortcut posts scans and PDFs to the inbox with a personal upload token. See [\_setup/iphone-shortcut.md](./_setup/iphone-shortcut.md).
- Every upload lands in the inbox first. _File_ puts it under an existing or new visit for a person.
- **Backlog**: for a pile of one person's old papers, pick the person under _Whose papers_ before uploading. Once read, the inbox suggests one visit per date and provider; _File as one visit_ files the group.

## 6. Reading

1. A background worker runs each new file through mac-ocr. The page text is searchable at once, with no AI call.
2. When every file of an item has pages, the engine masks each page on the Mac (HW-AI-7): lines with personal labels, the family's names, long digit runs and barcodes are blacked out in the image and replaced with `[ẩn]` in the text. A page it is unsure about waits for someone to check the masked preview on the review page, draw more boxes if needed, and press _Pages look safe, send_. Each box has a number. Untick a box that hides a result, and the engine sends that line to the AI.
3. The masked images and text go to the AI provider in one request, always to `gpt-6-luna` at low reasoning effort (D6). The answer becomes drafts: values linked to the metric catalog, findings, and a proposed person, date, provider and document kind. The person always comes from the local name match.
4. _File_ approves the drafts that were not rejected. Approved values can be edited or deleted later; each change is kept in their history.

Failed readings retry after 1, 5 and 30 minutes, then 6 hours, then stop with a _Retry_ button. Logs carry ids only, never page text or values.

## 7. Code

`apps/health-watcher`: Go engine at the root, SvelteKit UI in `ui/`.

| Path                 | Holds                                                                                                                                            |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| `domain/`            | Types and rules: roles and scope, validation, accent folding, file sniffing, masking, search snippets, the metric catalog, reading output checks |
| `port/`              | Interfaces the services use: store, file store, password hasher, OCR, image masker, AI reader                                                    |
| `service/`           | Use cases. Each tenant method takes a `domain.Scope` and checks the role                                                                         |
| `adapter/sqlite/`    | Store and migrations                                                                                                                             |
| `adapter/filestore/` | Files by SHA-256                                                                                                                                 |
| `adapter/password/`  | argon2id                                                                                                                                         |
| `adapter/fiber/`     | Routes, session and CSRF middleware, error envelope, UI mount                                                                                    |
| `adapter/macocr/`    | Runs mac-ocr and parses its JSON (schema 1.1 and 1.2)                                                                                            |
| `adapter/imaging/`   | Blacks out boxes in a page JPEG                                                                                                                  |
| `adapter/ai/openai/` | Responses API reader with a strict JSON schema                                                                                                   |
| `internal/app/`      | Wiring shared by `main.go` and `cmd/reset_password`                                                                                              |

Tests:

```sh
pnpm exec nx run health-watcher:test      # Go: domain, adapters, services on a temporary SQLite file, HTTP
pnpm exec nx run health-watcher:test-ui   # vitest
pnpm exec nx run health-watcher:check-ui  # svelte-check
```

`service/isolation_test.go` and the readings case in `service/clinical_test.go` check that a member of one tenant gets "not found" for every kind of resource in another tenant (HW-TEN-6). Add a case with every new resource. Service tests drive the background worker with a fake OCR tool and a fake AI reader (`service/pipeline_test.go`).
