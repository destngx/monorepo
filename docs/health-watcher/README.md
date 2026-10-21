# Health Watcher

A family's health records in one place: scans, PDFs and photos of results from hospitals, clinics and dentists, filed by person and visit, searchable later. What the app must do is in [REQUIREMENTS.md](./REQUIREMENTS.md); this page covers running it.

Phases 1 to 3 are built: accounts, families (tenants) with roles and invites, people, visits, the inbox, reading each upload with mac-ocr on the Mac and then with AI on masked pages, drafts that a person approves, search, the document viewer and the metric catalog. Charts and the AI review come in later phases ([REQUIREMENTS.md §13](./REQUIREMENTS.md#13-roadmap)).

## 1. Run it

```sh
pnpm exec nx run health-watcher:serve   # engine on :9191 and the UI dev server on :5174
pnpm exec nx run health-watcher:start   # production: builds the UI, the engine serves it at http://127.0.0.1:9191/ui
```

Reading needs the mac-ocr binary with schema 1.2 (`--render-dir`): `pnpm exec nx build mac-ocr` writes `apps/mac-ocr/bin/mac-ocr`, the engine's default. An older build still reads text, but its pages have no images, so nothing is sent to the AI provider until it is rebuilt.

On the first start the engine writes a one-time setup code to `setup-code` in the data folder and logs that file's path. The UI shows a setup page that asks for the code, then creates the first account and family. Only someone who can read the Mac's data folder can do setup.

## 2. Configuration

Environment variables, all optional:

| Variable             | Default                                        | Meaning                                                                                                                           |
| -------------------- | ---------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `HOST`               | `127.0.0.1`                                    | Listen address. Use the Mac's Tailscale address to reach it from a phone.                                                         |
| `PORT`               | `9191`                                         | Listen port.                                                                                                                      |
| `HW_DATA_DIR`        | `~/Library/Application Support/health-watcher` | Database and files.                                                                                                               |
| `HW_UI_BUILD_DIR`    | `ui/build`                                     | The built UI the engine serves.                                                                                                   |
| `HW_MAX_FILE_MB`     | `50`                                           | Largest accepted file.                                                                                                            |
| `HW_MAX_BATCH_MB`    | `200`                                          | Largest upload batch.                                                                                                             |
| `HW_MAC_OCR_BIN`     | `../mac-ocr/bin/mac-ocr`                       | The mac-ocr binary, relative to the engine's working directory.                                                                   |
| `HW_WORKERS`         | `2`                                            | Background jobs (OCR and AI reading) that run at once, 1 to 8.                                                                    |
| `AI_BASE_URL`        | empty                                          | OpenAI Responses API base, e.g. `http://localhost:8080/v1` for ai-gateway. Empty turns AI reading off; OCR and search still work. |
| `AI_API_KEY`         | empty                                          | Bearer token for the provider, if it needs one.                                                                                   |
| `AI_TIMEOUT_SECONDS` | `180`                                          | Limit for one reading request.                                                                                                    |

## 3. Data

Everything stays on the Mac, in `HW_DATA_DIR`:

- `health.db`: SQLite, mode `0600`. Schema changes are numbered SQL files in `adapter/sqlite/migrations`, applied at start.
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
- **iPhone share sheet**: a Shortcut posts scans and PDFs to the inbox with a personal upload token. See [\_setup/iphone-shortcut.md](./_setup/iphone-shortcut.md).
- Every upload lands in the inbox first. _File_ puts it under an existing or new visit for a person.
- **Backlog**: for a pile of one person's old papers, pick the person under _Whose papers_ before uploading. Once read, the inbox suggests one visit per date and provider; _File as one visit_ files the group.

## 6. Reading

1. A background worker runs each new file through mac-ocr. The page text is searchable at once, with no AI call.
2. When every file of an item has pages, the engine masks each page on the Mac (HW-AI-7): lines with personal labels, the family's names, long digit runs and barcodes are blacked out in the image and replaced with `[ẩn]` in the text. A page it is unsure about waits for someone to check the masked preview on the review page, draw more boxes if needed, and press _Pages look safe, send_.
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
