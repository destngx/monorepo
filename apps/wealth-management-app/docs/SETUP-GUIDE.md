# Setup Guide

## Wealth Management System — Installation & Configuration

**Version:** 1.0
**Last Updated:** 2026-02-28

---

## Prerequisites

| Requirement | Version | Purpose |
|---|---|---|
| **Node.js** | ≥ 20.x | Runtime |
| **pnpm** | ≥ 9.x | Package manager |
| **Google Account** | — | Google Sheets API access |

| **Google Sheet** | — | Must have Accounts, Transactions, Budget, and Investments tabs (see Tab Structure section) |
| **AI API Key** | — | At least one: OpenAI, Google AI, or Anthropic |

---

## Step 1: Clone & Install

```bash
git clone <your-repo-url> wealth-management
cd wealth-management
pnpm install
```

---

## Step 2: Google Cloud Setup (Sheets API)

This is the most involved step. Follow carefully.

### 2.1 Create Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Click **Select a project** → **New Project**
3. Name: `wealth-management` (or anything you like)
4. Click **Create**

### 2.2 Enable Google Sheets API

1. In your project, go to **APIs & Services** → **Library**
2. Search for **"Google Sheets API"**
3. Click **Enable**

### 2.3 Create Service Account

1. Go to **APIs & Services** → **Credentials**
2. Click **+ CREATE CREDENTIALS** → **Service Account**
3. Name: `wealth-sheets-bot`
4. Click **Create and Continue**
5. Role: Skip (no role needed)
6. Click **Done**

### 2.4 Generate Key

1. Click on your new service account (`wealth-sheets-bot@...`)
2. Go to **Keys** tab
3. Click **Add Key** → **Create new key**
4. Select **JSON** → **Create**
5. A JSON file will download. Keep it safe — **never commit this file**.

### 2.5 Extract Credentials

Open the downloaded JSON file and extract these values:

```json
{
  "client_email": "wealth-sheets-bot@project-id.iam.gserviceaccount.com",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvgIB...\n-----END PRIVATE KEY-----\n"
}
```

### 2.6 Share Your Google Sheet

1. Open your financial Google Sheet
2. Click **Share** button
3. Add the service account email (from step 2.5) as an **Editor**
4. Uncheck "Notify people"
5. Click **Share**

### 2.7 Get Spreadsheet ID

Your spreadsheet URL looks like:
```
https://docs.google.com/spreadsheets/d/1AbCdEfGhIjKlMnOpQrStUvWxYz/edit
                                       └──────────────────────────────┘
                                            This is your SPREADSHEET_ID
```

---

## Step 3: Get AI API Keys

You need **at least one** AI provider configured. You can add more later.

### Option A: OpenAI (Recommended for best quality)

1. Go to [platform.openai.com/api-keys](https://platform.openai.com/api-keys)
2. Click **Create new secret key**
3. Name: `wealth-management`
4. Copy the key (starts with `sk-`)

### Option B: Google Gemini (Free tier available)

1. Go to [aistudio.google.com/apikey](https://aistudio.google.com/apikey)
2. Click **Create API Key**
3. Select your project
4. Copy the key

### Option C: Anthropic

1. Go to [console.anthropic.com/settings/keys](https://console.anthropic.com/settings/keys)
2. Click **Create Key**
3. Name: `wealth-management`
4. Copy the key

## Step 3.5: Set Up Market Data APIs

The app can automatically fetch prices from free APIs. No API keys required for basic usage.

### CoinGecko (Crypto Prices)

- Free tier works without an API key (10-30 calls/min)
- Optional: Create a free account at [coingecko.com/api](https://www.coingecko.com/en/api) for higher rate limits
- Used for: BTC, ETH, and all crypto prices in VND and USD

### Gold Prices (SJC)

- Vietnamese gold prices can be fetched from free Vietnamese sources
- Fallback: Manual entry in Google Sheets if automated source is unavailable
- No API key required

### Stock Prices (Yahoo Finance)

- Uses unofficial Yahoo Finance API (no key required)
- Works for Vietnamese stocks (add `.VN` suffix: `FPT.VN`, `VNM.VN`)
- Works for US stocks (`AAPL`, `MSFT`)
- Rate limited — app caches prices with 1-hour TTL

**Note:** All market data APIs are free. No paid subscriptions required.
---

## Step 3.6: OAuth Setup (Optional)

OAuth lets you connect your OpenAI or Google account directly instead of manually copying API keys.

### Why OAuth?

- No API key management
- Automatic token refresh
- Google OAuth: Single login grants both Sheets and Gemini AI access

### OpenAI OAuth Setup

1. Register your app at [platform.openai.com](https://platform.openai.com)
2. Set callback URL: `http://localhost:3000/api/auth/callback` (dev) or your production URL
3. Note your Client ID
4. Add to `.env.local`:
```env
OPENAI_OAUTH_CLIENT_ID=app_...
OPENAI_OAUTH_REDIRECT_URI=http://localhost:3000/api/auth/callback
```
5. In the app: Settings → AI Provider → OpenAI → Click "Connect with OAuth"

### Google OAuth Setup (Recommended — replaces both API key and Service Account)

1. In [Google Cloud Console](https://console.cloud.google.com/), go to **APIs & Services** → **Credentials**
2. Click **+ CREATE CREDENTIALS** → **OAuth 2.0 Client ID**
3. Application type: **Web application**
4. Name: `wealth-management`
5. Authorized redirect URIs: `http://localhost:3000/api/auth/callback`
6. Click **Create**
7. Note the Client ID and Client Secret
8. Go to **OAuth consent screen** → Add scopes:
   - `https://www.googleapis.com/auth/spreadsheets` (Google Sheets)
   - `https://www.googleapis.com/auth/generativelanguage` (Gemini AI)
9. Add to `.env.local`:
```env
GOOGLE_OAUTH_CLIENT_ID=xxx.apps.googleusercontent.com
GOOGLE_OAUTH_CLIENT_SECRET=GOCSPX-...
GOOGLE_OAUTH_REDIRECT_URI=http://localhost:3000/api/auth/callback
```
10. In the app: Settings → Click "Connect with Google" → Authorize → Done

**Note:** Google OAuth replaces the Service Account for Sheets access AND provides Gemini AI access. One login, two services.

### Anthropic

Anthropic does **not** support OAuth (banned Feb 2026). Use API key only.

---

## Step 4: Environment Variables

Create `.env.local` in the project root:

```bash
cp .env.example .env.local
```

Fill in the values:

```env
# ===========================
# Google Sheets Configuration
# ===========================
GOOGLE_SHEETS_ID=1AbCdEfGhIjKlMnOpQrStUvWxYz
GOOGLE_SERVICE_ACCOUNT_EMAIL=wealth-sheets-bot@project-id.iam.gserviceaccount.com
GOOGLE_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\nMIIEvgIB...\n-----END PRIVATE KEY-----\n"

# Sheet tab names (adjust to match your actual tab names)
SHEET_TAB_ACCOUNTS=Accounts
SHEET_TAB_TRANSACTIONS=Transactions
SHEET_TAB_BUDGET=Budget
SHEET_TAB_INVESTMENTS=Investments

# ===========================
# AI Provider API Keys
# (configure at least one)
# ===========================
OPENAI_API_KEY=sk-...
GOOGLE_GENERATIVE_AI_API_KEY=AIza...
ANTHROPIC_API_KEY=sk-ant-...

# ===========================
# OAuth (Optional — alternative to API keys)
# ===========================
OPENAI_OAUTH_CLIENT_ID=
OPENAI_OAUTH_REDIRECT_URI=http://localhost:3000/api/auth/callback

GOOGLE_OAUTH_CLIENT_ID=
GOOGLE_OAUTH_CLIENT_SECRET=
GOOGLE_OAUTH_REDIRECT_URI=http://localhost:3000/api/auth/callback

# ===========================
# App Configuration
# ===========================
# Default AI model (options: gpt-4o, gpt-4o-mini, gemini-2.0-flash, gemini-2.5-pro, claude-sonnet)
DEFAULT_AI_MODEL=gpt-4o-mini

# Base currency for display
BASE_CURRENCY=VND

# Cache TTL in seconds
CACHE_TTL_SECONDS=300

# ===========================
# Optional
# ===========================
# CoinGecko API (free, no key required for basic usage)
# Only needed if you want higher rate limits
COINGECKO_API_KEY=

# Encryption key for OAuth tokens (generate with: openssl rand -hex 32)
OAUTH_ENCRYPTION_KEY=
```

### Important Notes

- **GOOGLE_PRIVATE_KEY:** Must include the `-----BEGIN PRIVATE KEY-----` and `-----END PRIVATE KEY-----` markers. Keep the `\n` characters — they represent newlines in the key.
- **At least one AI key** must be configured. The app will show which providers are available in settings.
- **Never commit `.env.local`** — it's in `.gitignore` by default.
- **OAUTH_ENCRYPTION_KEY:** Required if using OAuth. Generate with `openssl rand -hex 32`. Used to encrypt OAuth tokens stored in SQLite.
- **SHEET_TAB_INVESTMENTS:** Must match the exact tab name in your Google Sheet for investment tracking.

---

## Step 5: Initialize Database

```bash
pnpm prisma db push
```

This creates the SQLite database (`prisma/dev.db`) with all required tables.

---

## Step 6: Verify Google Sheets Connection

```bash
pnpm dev
```

Visit `http://localhost:3000/settings` and check:
- ✅ Google Sheets: Connected
- ✅ Spreadsheet ID: Valid
- ✅ Accounts tab: Found (N accounts)
- ✅ Transactions tab: Found (N transactions)
- ✅ Budget tab: Found (N categories)
- ✅ Investments tab: Found (N assets)

If any show ❌:
- **"Not connected"** → Check `GOOGLE_SHEETS_ID` and service account credentials
- **"Tab not found"** → Check `SHEET_TAB_*` names match your actual tab names
- **"Permission denied"** → Share the sheet with the service account email (Step 2.6)

---

## Step 7: Verify AI Connection

On the Settings page:
- ✅ OpenAI: Configured (if key provided)
- ✅ Google AI: Configured (if key provided)
- ✅ Anthropic: Configured (if key provided)

Test by going to `/chat` and asking: "What are my account balances?"

---

## Step 8: Run the App

### Development

```bash
pnpm dev
```

Open [http://localhost:3000](http://localhost:3000)

### Production Build

```bash
pnpm build
pnpm start
```

---

## Google Sheet Tab Structure

Your sheet should have these tabs with these exact headers in row 1:

### Tab: Accounts

| Column A | Column B | Column C | Column D | Column E | Column F | Column G | Column H |
|---|---|---|---|---|---|---|---|
| ACCOUNTS | Date to pay | Goal | % | Cleared | Balance | Type | Note |

### Tab: Transactions

| Col A | Col B | Col C | Col D | Col E | Col F | Col G | Col H | Col I | Col J | Col K | Col L | Col M |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Account | Date | Num | Payee | Tag | Memo | Category | Clr | PAYMENT | DEPOSIT | Account Balance | Cleared Balance | BALANCE |

### Tab: Budget

| Column A | Column B | Column C | Column D |
|---|---|---|---|
| Category | Monthly Limit | Yearly Limit | Notes |

### Tab: Investments

| Col A | Col B | Col C | Col D | Col E | Col F | Col G | Col H | Col I |
|---|---|---|---|---|---|---|---|---|
| Asset Name | Type | Symbol | Quantity | Purchase Price | Purchase Date | Currency | Account | Notes |

The Investments tab is required for portfolio tracking. If you don't have one yet, create it with the headers above. You can start with zero rows — add investments through the app or directly in the sheet.

> **Note:** If your sheet uses different header names, update `SHEET_TAB_*` env vars and the app will adapt. The header mapping is configurable.

---

## Troubleshooting

### "Error: The caller does not have permission"

→ You haven't shared the Google Sheet with the service account email. See Step 2.6.

### "Error: Requested entity was not found"

→ The `GOOGLE_SHEETS_ID` is wrong. Double-check the ID from your spreadsheet URL.

### "Error: invalid_grant"

→ The `GOOGLE_PRIVATE_KEY` is malformed. Make sure:
- It's wrapped in double quotes in `.env.local`
- Newlines are represented as `\n`
- The full key is present (including BEGIN/END markers)

### "Error: API key not valid"

→ AI provider API key is incorrect or expired. Generate a new one from the provider's console.

### "Error: Rate limit exceeded"

→ Too many Google Sheets API calls. The app has built-in caching (5-minute TTL). If you're hitting limits:
- Increase `CACHE_TTL_SECONDS` in `.env.local`
- Avoid rapid manual syncs

### Database issues

```bash
# Reset the database (deletes all local cache and chat history)
rm prisma/dev.db
pnpm prisma db push
```

### "Investments tab not found"

→ Create a new tab in your Google Sheet named "Investments" (or whatever name is set in `SHEET_TAB_INVESTMENTS`). Add headers in row 1: Asset Name | Type | Symbol | Quantity | Purchase Price | Purchase Date | Currency | Account | Notes

### "OAuth: redirect_uri_mismatch"

→ The redirect URI in your OAuth app must exactly match `GOOGLE_OAUTH_REDIRECT_URI` or `OPENAI_OAUTH_REDIRECT_URI` in `.env.local`. For local dev, use `http://localhost:3000/api/auth/callback`. For production, update to your actual domain.

### "Market prices not updating"

→ CoinGecko free tier has rate limits (10-30 calls/min). The app caches prices for 15 minutes (crypto) and 60 minutes (stocks/gold). If prices seem stale, click "Sync" to force refresh. For assets without symbols (real estate, business equity), prices must be updated manually.

---

## Updating

```bash
git pull
pnpm install
pnpm prisma db push  # In case schema changed
pnpm dev
```

---

## Project Scripts

| Command | Description |
|---|---|
| `pnpm dev` | Start development server |
| `pnpm build` | Production build |
| `pnpm start` | Start production server |
| `pnpm lint` | Run ESLint |
| `pnpm prisma studio` | Open Prisma database viewer |
| `pnpm prisma db push` | Sync Prisma schema to database |
