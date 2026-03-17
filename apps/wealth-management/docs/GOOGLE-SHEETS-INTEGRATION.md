# Google Sheets Integration

## Wealth Management System — Data Layer Design

**Version:** 1.0
**Last Updated:** 2026-02-28

---

## 1. Your Current Sheet Structure

Based on the provided headers, here's how your Google Sheet tabs are structured and how we map them to the application's data model.

---

### 1.1 Accounts Tab

#### Raw Headers

```
ACCOUNTS | Date to pay | Goal | % | Cleared | Balance | Type | Note
```

#### Column Mapping

| Sheet Column | App Field | Type | Description |
|---|---|---|---|
| `ACCOUNTS` | `name` | `string` | Account name (e.g., "VietcomBank", "Binance") |
| `Date to pay` | `dueDate` | `Date \| null` | Bill payment due date (e.g., credit card) |
| `Goal` | `goalAmount` | `number \| null` | Savings goal amount |
| `%` | `goalProgress` | `number \| null` | Progress toward goal (0–100) |
| `Cleared` | `clearedBalance` | `number` | Cleared/confirmed balance |
| `Balance` | `balance` | `number` | Current balance (including pending) |
| `Type` | `type` | `AccountType` | Account type (see below) |
| `Note` | `note` | `string \| null` | Free-form notes |

#### Derived Fields (not in sheet, computed by app)

| Field | Source | Description |
|---|---|---|
| `currency` | Inferred from Type or Name | VND for banks, USDT for Binance |
| `balanceInVND` | `balance × exchangeRate` | Normalized balance for net worth |

#### Account Types

```typescript
type AccountType =
  | 'bank'       // Vietnamese bank accounts (VND)
  | 'crypto'     // Crypto exchanges (Binance = USDT)
  | 'cash'       // Cash on hand
  | 'credit'     // Credit cards
  | 'investment' // Investment accounts
  | 'savings'    // Savings accounts
  | 'ewallet'    // MoMo, ZaloPay, etc.
  | 'other';
```

#### Example Data

| ACCOUNTS | Date to pay | Goal | % | Cleared | Balance | Type | Note |
|---|---|---|---|---|---|---|---|
| VietcomBank | | 100,000,000 | 45 | 48,500,000 | 50,000,000 | bank | Main account |
| Binance | | | | 1,450 | 1,500 | crypto | Spot + Earn |
| MoMo | | | | 200,000 | 200,000 | ewallet | |

---

### 1.2 Transactions Tab

#### Raw Headers

```
Account | Date | Num | Payee | Tag | Memo | Category | Clr | PAYMENT | DEPOSIT | Account Balance | Cleared Balance | BALANCE
```

#### Column Mapping

| Sheet Column | App Field | Type | Description |
|---|---|---|---|
| `Account` | `accountName` | `string` | Which account this transaction belongs to |
| `Date` | `date` | `Date` | Transaction date |
| `Num` | `referenceNumber` | `string \| null` | Check number or reference ID |
| `Payee` | `payee` | `string` | Who was paid / who paid you |
| `Tag` | `tags` | `string[]` | Comma-separated tags (parsed to array) |
| `Memo` | `memo` | `string \| null` | Transaction description/notes |
| `Category` | `category` | `string` | Spending category |
| `Clr` | `cleared` | `boolean` | Cleared status (`*` or `R` = true) |
| `PAYMENT` | `payment` | `number \| null` | Debit amount (money out) |
| `DEPOSIT` | `deposit` | `number \| null` | Credit amount (money in) |
| `Account Balance` | `accountBalance` | `number` | Running account balance |
| `Cleared Balance` | `clearedBalance` | `number` | Running cleared balance |
| `BALANCE` | `runningBalance` | `number` | Overall running balance |

#### Derived Fields

| Field | Source | Description |
|---|---|---|
| `id` | Row index (2-based, header = row 1) | Unique identifier for CRUD operations |
| `amount` | `deposit - payment` (or vice versa) | Signed amount (+income, -expense) |
| `type` | `payment > 0 ? 'expense' : 'income'` | Transaction type |
| `isTransfer` | Category starts with "Transfer" | Inter-account transfer detection |

#### Example Data

| Account | Date | Num | Payee | Tag | Memo | Category | Clr | PAYMENT | DEPOSIT | Account Balance | Cleared Balance | BALANCE |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| VietcomBank | 28/02/2026 | | Grab | transport | Ride to office | Transportation | * | 45,000 | | 49,955,000 | 49,955,000 | 49,955,000 |
| VietcomBank | 28/02/2026 | | GrabFood | food | Lunch delivery | Food & Dining | | 85,000 | | 49,870,000 | 49,955,000 | 49,870,000 |
| VietcomBank | 27/02/2026 | | Company | salary | February salary | Income | * | | 25,000,000 | 49,955,000 | 49,955,000 | 49,955,000 |

---

### 1.3 Budget Tab

> **Note:** You mentioned having budget data but didn't share the exact headers. I'll design a recommended structure. If your current sheet differs, we'll add a mapper.

#### Recommended Structure

```
Category | Monthly Limit | Yearly Limit | Notes
```

| Sheet Column | App Field | Type | Description |
|---|---|---|---|
| `Category` | `category` | `string` | Category name (matches transaction categories) |
| `Monthly Limit` | `monthlyLimit` | `number` | Monthly spending budget |
| `Yearly Limit` | `yearlyLimit` | `number` | Yearly spending budget |
| `Notes` | `notes` | `string \| null` | Notes about this budget line |

#### Derived Fields (computed by app)

| Field | Source | Description |
|---|---|---|
| `monthlySpent` | Sum of transactions for category in current month | Actual spending |
| `yearlySpent` | Sum of transactions for category in current year | Actual spending |
| `monthlyRemaining` | `monthlyLimit - monthlySpent` | Budget remaining |
| `yearlyRemaining` | `yearlyLimit - yearlySpent` | Budget remaining |
| `monthlyPercentUsed` | `(monthlySpent / monthlyLimit) * 100` | Usage percentage |

---

### 1.4 Investments Tab

#### Raw Headers

```
Asset Name | Type | Symbol | Quantity | Purchase Price | Purchase Date | Currency | Account | Notes
```

#### Column Mapping

| Sheet Column | App Field | Type | Description |
|---|---|---|---|
| `Asset Name` | `name` | `string` | Human-readable name ("Bitcoin", "SJC Gold 1 tael", "FPT Stock") |
| `Type` | `type` | `InvestmentType` | Asset class (see below) |
| `Symbol` | `symbol` | `string` | Ticker/symbol for price lookup ("BTC", "ETH", "GOLD_SJC", "FPT", "") |
| `Quantity` | `quantity` | `number` | How much you own (0.5 BTC, 2 taels, 100 shares, 1 apartment) |
| `Purchase Price` | `purchasePrice` | `number` | Cost basis per unit in original currency |
| `Purchase Date` | `purchaseDate` | `Date` | When purchased |
| `Currency` | `currency` | `Currency` | VND, USD, or USDT |
| `Account` | `account` | `string` | Where it's held ("Binance", "VCBS", "Physical", "Bank X") |
| `Notes` | `notes` | `string | null` | Free-form notes ("Spot account", "24K SJC bar", "District 7 apartment") |

#### Derived Fields (computed by app)

| Field | Source | Description |
|---|---|---|
| `id` | Row index (2-based) | Unique identifier for CRUD |
| `currentPrice` | Market data API | Live price per unit |
| `currentValueVND` | `quantity * currentPrice * exchangeRate` | Total value in VND |
| `costBasisVND` | `quantity * purchasePrice * exchangeRate` | Total cost in VND |
| `pnlVND` | `currentValueVND - costBasisVND` | Unrealized P&L in VND |
| `pnlPercent` | `(pnlVND / costBasisVND) * 100` | P&L percentage |
| `allocationPercent` | `currentValueVND / totalPortfolioVND * 100` | Portfolio weight |

#### Investment Types

```typescript
type InvestmentType =
  | 'crypto'           // BTC, ETH, etc. on Binance or other exchanges
  | 'stock'            // Vietnamese stocks (FPT, VNM, VN30), international
  | 'gold'             // Physical gold (SJC), gold savings
  | 'savings_deposit'  // Tiết kiệm kỳ hạn (term deposits at banks)
  | 'real_estate'      // Property (apartment, land)
  | 'business_equity'  // Business ownership stake
  | 'other';           // Anything else
```

#### Example Data

| Asset Name | Type | Symbol | Quantity | Purchase Price | Purchase Date | Currency | Account | Notes |
|---|---|---|---|---|---|---|---|---|
| Bitcoin | crypto | BTC | 0.5 | 60000 | 15/03/2024 | USD | Binance | Spot account |
| Ethereum | crypto | ETH | 3 | 3200 | 20/06/2024 | USD | Binance | Spot account |
| SJC Gold 1 tael | gold | GOLD_SJC | 2 | 79000000 | 10/01/2025 | VND | Physical | 24K SJC bars, safe deposit |
| FPT Stock | stock | FPT | 100 | 125000 | 05/09/2024 | VND | VCBS | Long-term hold |
| Savings 6-month | savings_deposit | | 1 | 200000000 | 01/12/2025 | VND | VietcomBank | 6.5% annual, matures Jun 2026 |
| District 7 Apartment | real_estate | | 1 | 3500000000 | 15/06/2023 | VND | Physical | 2BR, Phu My Hung |

#### Symbol Conventions

| Type | Symbol Format | Example | Price Source |
|---|---|---|---|
| Crypto | CoinGecko ID | BTC, ETH, SOL, BNB | CoinGecko API |
| Gold | GOLD_SJC, GOLD_24K | GOLD_SJC | Vietnamese gold price API / manual |
| Stock (VN) | Ticker | FPT, VNM, VIC | Free stock API (Yahoo Finance) |
| Stock (US) | Ticker | AAPL, MSFT | Yahoo Finance API |
| Savings deposit | (empty) | | No live price needed (fixed return) |
| Real estate | (empty) | | No live price (manual valuation) |
| Business equity | (empty) | | No live price (manual valuation) |

Note: Assets with empty symbols don't get live price updates. Their current value = purchase price (or manually updated value).

## 2. Data Access Architecture

### 2.1 Service Layer Flow

```
┌───────────┐     ┌───────────────┐     ┌────────────┐     ┌──────────────┐
│  API Route │ ──▶ │ Service Layer  │ ──▶ │ Cache Layer │ ──▶ │ Google Sheets│
│            │     │               │     │ (SQLite)    │     │ API v4       │
│ /api/txns  │     │ transaction-  │     │             │     │              │
│            │ ◀── │ service.ts    │ ◀── │ Hit? Return │ ◀── │ Response     │
└───────────┘     └───────────────┘     └────────────┘     └──────────────┘
```

### 2.2 Cache Strategy

| Data | Cache Key Pattern | TTL | Invalidation |
|---|---|---|---|
| Accounts | `sheets:accounts` | 5 min | On account update, manual sync |
| Transactions (month) | `sheets:txns:2026-02` | 5 min | On transaction CRUD, manual sync |
| Transactions (all) | `sheets:txns:all` | 10 min | On any transaction change |
| Budget | `sheets:budget` | 5 min | On budget update, manual sync |
| Exchange Rate | `exchange:usdt-vnd` | 15 min | Time-based only |
| Investments | `sheets:investments` | 5 min | On investment CRUD, manual sync |
| Crypto Prices | `market:crypto` | 15 min | Time-based only |
| Gold Prices | `market:gold` | 60 min | Time-based only |
| Stock Prices | `market:stocks` | 60 min | Time-based only |

### 2.3 Read Pattern

```typescript
async function getTransactions(filters: TransactionFilters): Promise<Transaction[]> {
  const cacheKey = buildCacheKey('txns', filters);
  
  // 1. Check cache
  const cached = await getCached<Transaction[]>(cacheKey);
  if (cached) return cached;
  
  // 2. Fetch from Google Sheets
  const rawRows = await sheetsClient.readSheet('Transactions', 'A:M');
  
  // 3. Map to types
  const transactions = rawRows
    .slice(1) // Skip header
    .map((row, index) => mapRowToTransaction(row, index + 2)) // 2-based row index
    .filter(txn => applyFilters(txn, filters));
  
  // 4. Cache result
  await setCache(cacheKey, transactions, 5 * 60 * 1000); // 5 min
  
  return transactions;
}
```

### 2.4 Write Pattern

```typescript
async function addTransaction(input: CreateTransactionInput): Promise<Transaction> {
  // 1. Validate input
  const validated = createTransactionSchema.parse(input);
  
  // 2. Auto-categorize if no category provided
  if (!validated.category) {
    const suggestion = await categorizeTransaction(validated);
    validated.category = suggestion.category;
  }
  
  // 3. Map to sheet row
  const row = mapTransactionToRow(validated);
  
  // 4. Append to Google Sheet
  await sheetsClient.appendRow('Transactions', row);
  
  // 5. Invalidate cache
  await invalidateCache('sheets:txns:*');
  
  // 6. Return created transaction
  return mapRowToTransaction(row, -1); // ID will be assigned on next read
}
```

---

## 3. Google Sheets API Setup

### 3.1 Authentication Method: Service Account

**Why Service Account over OAuth:**
- Simpler setup (no login flow required)
- Server-side only (no token refresh in browser)
- Perfect for single-user app
- Share the sheet with the service account email → instant access

**Setup Steps:**

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project (or use existing)
3. Enable **Google Sheets API**
4. Go to **Credentials** → Create **Service Account**
5. Download the JSON key file
6. Share your Google Sheet with the service account email (e.g., `finance-bot@project-id.iam.gserviceaccount.com`) as **Editor**
7. Extract credentials for `.env.local`:

```env
GOOGLE_SHEETS_ID=your-spreadsheet-id-from-url
GOOGLE_SERVICE_ACCOUNT_EMAIL=finance-bot@project-id.iam.gserviceaccount.com
GOOGLE_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"
```

### 3.2 Sheet ID

The spreadsheet ID is in the URL:
```
https://docs.google.com/spreadsheets/d/{SPREADSHEET_ID}/edit
```

### 3.3 API Client Implementation

```typescript
// src/lib/sheets/client.ts
import { google } from 'googleapis';

const auth = new google.auth.GoogleAuth({
  credentials: {
    client_email: process.env.GOOGLE_SERVICE_ACCOUNT_EMAIL,
    private_key: process.env.GOOGLE_PRIVATE_KEY?.replace(/\\n/g, '\n'),
  },
  scopes: ['https://www.googleapis.com/auth/spreadsheets'],
});

const sheets = google.sheets({ version: 'v4', auth });
const SPREADSHEET_ID = process.env.GOOGLE_SHEETS_ID!;

export async function readSheet(
  sheetName: string,
  range: string = 'A:Z'
): Promise<string[][]> {
  const response = await sheets.spreadsheets.values.get({
    spreadsheetId: SPREADSHEET_ID,
    range: `${sheetName}!${range}`,
  });
  return response.data.values || [];
}

export async function appendRow(
  sheetName: string,
  values: (string | number | null)[]
): Promise<void> {
  await sheets.spreadsheets.values.append({
    spreadsheetId: SPREADSHEET_ID,
    range: `${sheetName}!A:A`,
    valueInputOption: 'USER_ENTERED',
    requestBody: {
      values: [values],
    },
  });
}

export async function updateRow(
  sheetName: string,
  rowIndex: number, // 1-based
  values: (string | number | null)[]
): Promise<void> {
  await sheets.spreadsheets.values.update({
    spreadsheetId: SPREADSHEET_ID,
    range: `${sheetName}!A${rowIndex}:M${rowIndex}`,
    valueInputOption: 'USER_ENTERED',
    requestBody: {
      values: [values],
    },
  });
}

export async function deleteRow(
  sheetName: string,
  rowIndex: number // 1-based
): Promise<void> {
  // Get sheet ID (not spreadsheet ID)
  const spreadsheet = await sheets.spreadsheets.get({
    spreadsheetId: SPREADSHEET_ID,
  });
  
  const sheet = spreadsheet.data.sheets?.find(
    s => s.properties?.title === sheetName
  );
  
  if (!sheet?.properties?.sheetId) {
    throw new Error(`Sheet "${sheetName}" not found`);
  }
  
  await sheets.spreadsheets.batchUpdate({
    spreadsheetId: SPREADSHEET_ID,
    requestBody: {
      requests: [{
        deleteDimension: {
          range: {
            sheetId: sheet.properties.sheetId,
            dimension: 'ROWS',
            startIndex: rowIndex - 1, // 0-based
            endIndex: rowIndex,
          },
        },
      }],
    },
  });
}
```

---

## 4. Data Transformation

### 4.1 Number Parsing

Vietnamese number format uses `.` as thousands separator:
```typescript
function parseVNDAmount(value: string): number {
  if (!value || value.trim() === '') return 0;
  // Remove dots (thousand separators) and replace comma with dot (decimal)
  return parseFloat(value.replace(/\./g, '').replace(',', '.')) || 0;
}

function parseCryptoAmount(value: string): number {
  if (!value || value.trim() === '') return 0;
  return parseFloat(value.replace(/,/g, '')) || 0;
}
```

### 4.2 Date Parsing

```typescript
function parseSheetDate(value: string): Date {
  // Support multiple formats: DD/MM/YYYY, MM/DD/YYYY, YYYY-MM-DD
  // Default assumption: DD/MM/YYYY (Vietnamese convention)
  const parts = value.split('/');
  if (parts.length === 3) {
    const [day, month, year] = parts;
    return new Date(parseInt(year), parseInt(month) - 1, parseInt(day));
  }
  return new Date(value); // Fallback to native parsing
}

function formatDateForSheet(date: Date): string {
  return `${date.getDate().toString().padStart(2, '0')}/${
    (date.getMonth() + 1).toString().padStart(2, '0')}/${
    date.getFullYear()}`;
}
```

### 4.3 Cleared Status

```typescript
function parseClearedStatus(value: string): boolean {
  return value === '*' || value === 'R' || value.toLowerCase() === 'true';
}
```

### 4.4 Investments Data Mapping

```typescript
// src/lib/sheets/investments.ts

function mapRowToInvestment(row: string[], rowIndex: number): Investment {
  return {
    id: `inv-${rowIndex}`,
    name: row[0] || '',
    type: parseInvestmentType(row[1]),
    symbol: row[2] || '',
    quantity: parseNumber(row[3]),
    purchasePrice: parseNumber(row[4]),
    purchaseDate: parseSheetDate(row[5]),
    currency: parseCurrency(row[6]),
    account: row[7] || '',
    notes: row[8] || null,
  };
}

function mapInvestmentToRow(investment: CreateInvestmentInput): (string | number | null)[] {
  return [
    investment.name,
    investment.type,
    investment.symbol,
    investment.quantity,
    investment.purchasePrice,
    formatDateForSheet(investment.purchaseDate),
    investment.currency,
    investment.account,
    investment.notes || '',
  ];
}
```

---

## 5. Sync Strategy

### 5.1 Read Sync (Sheet → App)

- **Automatic:** Every page load checks cache TTL, fetches if stale
- **Manual:** "Sync" button in header forces cache invalidation + re-fetch
- **Background:** Optional polling (every 5 minutes) for dashboard auto-refresh

### 5.2 Write Sync (App → Sheet)

- **Immediate:** Every CRUD operation writes to Google Sheets synchronously
- **Optimistic UI:** UI updates immediately, rolls back on failure
- **Cache Invalidation:** Relevant cache keys invalidated after successful write

### 5.3 Conflict Resolution

Since this is a single-user system, conflicts are rare but possible (e.g., editing in both app and sheet simultaneously):

| Scenario | Resolution |
|---|---|
| App writes, sheet had new rows | No conflict — append adds new row |
| App edits row, same row changed in sheet | **Last write wins** (app overwrites) |
| Sheet structure changed (headers moved) | Error — notify user to check sheet |
| Row deleted in sheet, app tries to edit | Error — re-sync required |

### 5.4 Rate Limiting

Google Sheets API limits:
- **Read:** 300 requests per minute per project
- **Write:** 60 requests per minute per project

Our handling:
- Cache reduces read frequency to ~12 requests/hour (1 per page load, 5-min TTL)
- Batch writes where possible (bulk categorization → single batch update)
- Exponential backoff on 429 responses

---

## 6. Tab Name Configuration

Tab names are configurable in settings to match your actual sheet:

```typescript
// Default tab names (can be overridden in settings)
const DEFAULT_TAB_NAMES = {
  accounts: 'Accounts',       // Or whatever your tab is named
  transactions: 'Transactions', // Or 'Register', 'Ledger', etc.
  budget: 'Budget',            // Or 'Monthly Budget', etc.
  investments: 'Investments',  // Or 'Portfolio', etc.
};
```

The app will auto-detect tab names on first connection and let you map them in settings if they differ.

---

## 7. Error Handling

### 7.1 Common Errors

| Error | Cause | Handling |
|---|---|---|
| `403 Forbidden` | Sheet not shared with service account | Setup guide prompt |
| `404 Not Found` | Wrong spreadsheet ID or tab name | Settings check |
| `429 Rate Limited` | Too many API calls | Backoff + queue |
| `400 Invalid Range` | Column count mismatch | Re-detect headers |
| Network timeout | API latency | Retry (max 3x) + cache fallback |

### 7.2 Header Validation

On first connection, validate that expected headers exist:

```typescript
async function validateSheetStructure(): Promise<ValidationResult> {
  const accountHeaders = await readSheet('Accounts', 'A1:H1');
  const investmentHeaders = await readSheet('Investments', 'A1:I1');

  const expectedAccountHeaders = ['ACCOUNTS', 'Date to pay', 'Goal', '%', 'Cleared', 'Balance', 'Type', 'Note'];
  const expectedTxnHeaders = ['Account', 'Date', 'Num', 'Payee', 'Tag', 'Memo', 'Category', 'Clr', 'PAYMENT', 'DEPOSIT', 'Account Balance', 'Cleared Balance', 'BALANCE'];
  const expectedInvestmentHeaders = ['Asset Name', 'Type', 'Symbol', 'Quantity', 'Purchase Price', 'Purchase Date', 'Currency', 'Account', 'Notes'];
  
  // Compare and report mismatches
  return {
    accountsValid: compareHeaders(accountHeaders[0], expectedAccountHeaders),
    transactionsValid: compareHeaders(txnHeaders[0], expectedTxnHeaders),
    investmentsValid: compareHeaders(investmentHeaders[0], expectedInvestmentHeaders),
    mismatches: [...] // List of missing/extra columns
  };
}
```

---

## 8. Market Data Integration

### 8.1 Price Sources

| Asset Type | API | Endpoint | Free Tier Limits |
|---|---|---|---|
| Crypto | CoinGecko | `/api/v3/simple/price` | 10-30 calls/min (free) |
| Gold (SJC) | Vietnamese gold APIs | TBD (scrape or manual) | Varies |
| Gold (International) | Free gold APIs | Various free endpoints | Varies |
| Stocks (VN) | Yahoo Finance | `/v8/finance/chart/{symbol}.VN` | Unofficial, rate limited |
| Stocks (US) | Yahoo Finance | `/v8/finance/chart/{symbol}` | Unofficial, rate limited |

### 8.2 Price Fetching Strategy

```
1. On dashboard/investments page load:
   - Check MarketPrice cache (SQLite)
   - If fresh (within TTL) -> use cached
   - If stale -> fetch from API -> update cache -> return

2. Batch fetching:
   - CoinGecko: fetch all crypto prices in one call (ids=bitcoin,ethereum,solana&vs_currencies=usd,vnd)
   - Stocks: batch Yahoo Finance calls
   - Gold: single call for SJC price

3. Manual valuation fallback:
   - Real estate, business equity: no API, user enters current value manually
   - Savings deposits: calculate based on interest rate and maturity date
```

### 8.3 Exchange Rate Handling

- CoinGecko provides VND prices directly for crypto
- Stock prices in VND (Vietnamese stocks) or USD (international) -> convert using USDT/VND rate
- Gold SJC prices natively in VND
- All portfolio values normalized to VND for net worth calculation
