# MarketPulse Dashboard — Real-time Dual-Market Correlation Monitor

Build a MarketPulse Dashboard into the Investments page with **two market panels** — American 🇺🇸 and Vietnamese 🇻🇳 — providing real-time % change visualization, market driver analysis, capital flow signals, and correlation indicators for both markets.

---

## Features

| # | Feature | Description |
|---|---------|-------------|
| 1 | **Dual Market % Change Charts** | Two bar charts — US and VN — showing % change per asset |
| 2 | **Drivers Analysis** | Top movers summary per market |
| 3 | **Capital Flow Signals** | Defensive/offensive signal per market |
| 4 | **Correlation Heatmap** | Full NxN matrix with color scale (-1 red → 0 white → +1 green), hover tooltips |
| 5 | **Scenario Detection** | Computed market regime (e.g. "Normal Growth (Risk-ON)") with confidence % |
| 6 | **Market Snapshot Table** | Per-ticker: Price, 1-Candle %, ~1-Day %, ~1-Week %, Direction arrow, Momentum icon |
| 7 | **How to Read Card** | Bilingual (EN/VI) educational card explaining correlations and dashboard usage |
| 8 | **Timeframe Selector** | 1H, 4H, 1D, 1W |
| 9 | **Auto-Refresh + Timestamp** | Configurable interval with last updated display |
| 10 | **Ticker Guide** | Bilingual reference explaining each symbol |

---

## Tracked Assets

| Market | Ticker | Yahoo Symbol | Description |
|--------|--------|-------------|-------------|
| 🇺🇸 US | VIX | `^VIX` | Volatility Index — Chỉ số biến động |
| 🇺🇸 US | DXY | `DX-Y.NYB` | US Dollar Index — Chỉ số sức mạnh USD |
| 🇺🇸 US | US10Y | `^TNX` | US 10Y Treasury Yield — Lợi suất TPCP Mỹ 10 năm |
| 🇺🇸 US | WTI | `CL=F` | Crude Oil — Dầu thô WTI |
| 🇺🇸 US | Gold | `GC=F` | Gold (XAU/USD) — Vàng |
| 🇺🇸 US | S&P500 | `^GSPC` | S&P 500 Index — Chỉ số cổ phiếu vốn hóa lớn Mỹ |
| 🇺🇸 US | NQ100 | `^NDX` | Nasdaq-100 — Chỉ số công nghệ Mỹ |
| 🇺🇸 US | BTC | `BTC-USD` | Bitcoin |
| 🇻🇳 VN | VN-Index | `^VNINDEX` | Vietnam Stock Index — Chỉ số chứng khoán VN |
| 🇻🇳 VN | HNX | `^HNXINDEX` | Hanoi Exchange Index — Sàn Hà Nội |
| 🇻🇳 VN | VN30 | `^VN30` | VN30 Blue-chip Index — 30 cổ phiếu vốn hóa lớn |
| 🇻🇳 VN | USD/VND | `VND=X` | Exchange Rate — Tỷ giá USD/VND |
| 🇻🇳 VN | VN Gold | `GC=F` (converted) | Gold in VND (proxy via XAU × USD/VND rate) |

---

## Architecture

```
┌────────────────────────────────────────────────────────────────────┐
│  Frontend: MarketPulseDashboard          │  Frontend: Think Tank  │
│  (SWR → /api/market-pulse)               │  (fetch → /api/ai/     │
│                                          │   investment-analysis)  │
└─────────┬────────────────────────────────┴──────────┬─────────────┘
          ▼                                          ▼
┌──────────────────────┐          ┌──────────────────────────────┐
│ GET /api/market-pulse │          │ POST /api/ai/investment-     │
│ Returns JSON for UI   │          │      analysis                │
└─────────┬────────────┘          │ Injects MarketPulse data     │
          │                       │ into AI prompt as context     │
          │                       └──────────┬───────────────────┘
          │                                  │
          ▼                                  ▼
┌────────────────────────────────────────────────────────────────────┐
│  Service: market-data-service.ts  (SHARED)                        │
│  - Yahoo Finance v8 API (server-side)                             │
│  - In-memory cache (5 min trading / 30 min off-hours)             │
│  - Exports: getMarketPulseData() → both markets + matrix + etc   │
│  - Used by BOTH /api/market-pulse AND /api/ai/investment-analysis │
└───────────────────────────────────────────────────────────────────┘
```

**Key design**: `market-data-service.ts` exports a single `getMarketPulseData()` function consumed by:
1. `/api/market-pulse` — returns raw data to the dashboard UI
2. `/api/ai/investment-analysis` — injects as structured context into the AI prompt

---

## Technical Considerations & Build Notes

### 1. Yahoo Finance API

**Endpoint pattern** (no API key required):
```
GET https://query1.finance.yahoo.com/v8/finance/chart/{SYMBOL}?range={RANGE}&interval={INTERVAL}
```

| Timeframe | `range` | `interval` | Notes |
|-----------|---------|-----------|-------|
| 1 Hour    | `1d`    | `5m`      | Intraday 5-min candles, calculate 1h change from last 12 candles |
| 4 Hours   | `5d`    | `15m`     | 4h change from last 16 candles |
| 1 Day     | `5d`    | `1d`      | Daily close-to-close |
| 1 Week    | `1mo`   | `1d`      | Calculate from 5 trading days ago |

**Considerations:**
- ⚠️ **Rate limiting**: Yahoo Finance may throttle at ~2000 requests/hour. Batch all symbols into one request using comma-separated symbols where possible, or use sequential fetching with cache.
- ⚠️ **No CORS**: Must be called server-side (API route), NOT from client. This is why we use `/api/market-pulse` as proxy.
- ⚠️ **VN tickers availability**: `^VNINDEX`, `^HNXINDEX`, and `^VN30` may not always be available on Yahoo Finance v8. **Fallback strategy**: If Yahoo returns empty data for VN tickers, use the existing `executeSearch` (AI web search from `search-agent.ts`) to scrape latest values from CafeF or VNDirect.
- Yahoo Finance sometimes returns `null` for `regularMarketChangePercent` on futures/indices outside trading hours. Must handle gracefully.

### 2. Trading Hours & Data Freshness

| Market | Trading Hours (Local) | UTC | Status Logic |
|--------|----------------------|-----|-------------|
| 🇺🇸 US | 09:30–16:00 ET | 14:30–21:00 UTC | Pre-market 04:00–09:30 ET |
| 🇻🇳 VN | 09:00–11:30, 13:00–15:00 ICT | 02:00–08:00 UTC | Lunch break 11:30–13:00 |
| 🌐 Crypto | 24/7 | 24/7 | Always live |

**Considerations:**
- During off-hours, show "Market Closed" badge with last closing data
- VN market has a **lunch break** (11:30–13:00 ICT) — during this period, show "Lunch Break" status
- Crypto (BTC) is always live — handle differently from equity indices
- Cache TTL should be **shorter during trading hours** (5 min) and **longer off-hours** (30 min)
- The `lastUpdated` timestamp should reflect actual data time, not request time

### 3. Correlation Matrix Computation

The correlation heatmap requires **historical price data**, not just a single snapshot.

**Approach:**
1. Fetch `range=1mo&interval=1d` to get ~22 daily closes per ticker
2. Compute **Pearson correlation** between each pair of daily % returns:
   ```
   correlation(A, B) = Σ((Aᵢ - μA)(Bᵢ - μB)) / (σA × σB × n)
   ```
3. Result is an NxN symmetric matrix where `matrix[i][j]` = correlation between asset i and asset j
4. Diagonal is always 1.0 (perfect self-correlation)

**Considerations:**
- Need at least **10+ data points** for meaningful correlation (1 month = ~22 trading days is good)
- Different markets have different trading calendars — only use overlapping dates when computing cross-market correlations
- VN Gold is synthetic (XAU × rate), so compute its returns from the product, not independently
- Cache the correlation matrix separately with a longer TTL (1 hour) since it changes slowly

### 4. AI-Driven Scenario Detection (Intelligence Layer)

Classify the market into one of these regimes using a specialized **LLM-driven synthesis** (e.g., GPT-4o).

**Process:**
1. Feed the AI the current terminal state (all 13+ assets, % changes, and correlation matrix).
2. AI reasons about "Broken Correlations" (e.g., Gold and Equities both up) and Geopolitical risk.
3. **Output**:
   - `Regime`: Risk-ON, Risk-OFF, Crisis, Stagflation, Goldilocks.
   - `Scenario Name`: Context-specific title (e.g., "AI-Led Momentum", "Safe Haven Pivot").
   - `Summaries`: Nuanced, data-backed bilingual (EN/VI) explanations.

**Regime Archetypes:**
| Regime | Icon | Traditional Conditions (AI Reference) |
|--------|------|--------------------------------------|
| **Normal Growth (Risk-ON)** | ✅ | VIX < 20, S&P500 ↑, Gold flat/↓, Equities leading |
| **Defensive (Risk-OFF)** | ⚠️ | VIX 20–30, Gold ↑, Equities ↓, Bonds ↑ (US10Y ↓) |
| **Crisis Mode** | 🚨 | VIX > 30, strong Gold surge, Equities crash, DXY spike |
| **Stagflation** | 🔴 | Oil ↑↑, Equities ↓, Bonds ↓ (US10Y ↑), DXY mixed |
| **Goldilocks** | 🟢 | VIX < 15, Equities ↑, Bonds stable, low volatility |

**Resilience Strategy:**
- If the AI request fails or times out (>5s), the system must automatically fallback to the **Rules-Based Heuristic** logic to ensure the dashboard remains functional.

### 5. AI-Driven Capital Flow Analysis

**Approach:**
- Instead of simple "If X then Y" logic, the AI analyzes the **relative velocity** of money between asset classes (Crypto vs Equities vs Gold vs Bonds).
- **Signal Types**: `DEFENSIVE`, `RISK-ON`, `MIXED`.
- **Output**: A sharp, one-sentence insight in both English and Vietnamese.

Columns and data sources:

| Column | Source | Notes |
|--------|--------|-------|
| **Ticker** | Static config | Display name (e.g., "VIX", "VN-Index") |
| **Price** | Yahoo Finance `regularMarketPrice` | Format: $XX.XX for US, ₫XX,XXX for VN |
| **1-Candle %** | Computed from last candle's open→close | Depends on selected timeframe interval |
| **~1-Day %** | `regularMarketChangePercent` from Yahoo | Or computed from previous close |
| **~1-Week %** | Computed from 5 trading days ago close | Need historical data |
| **DIR** | Direction arrow | ↑ green if positive, ↓ red if negative (based on 1-Day %) |
| **Momentum** | Trend icon | 📈 uptrend, 📉 downtrend, 📊 sideways (based on 1-week direction consistency) |

**Considerations:**
- Number formatting: US uses `$`, VN uses `₫` prefix and dot thousands separator
- Color coding: green for positive %, red for negative %, gray for 0% / unavailable
- Sort order should match the chart order for visual consistency
- Extremely large % changes (>20%) should get a special highlight

### 6. Drivers & Capital Flow Analysis

**Drivers** (computed server-side):
1. Sort all assets by `|percentChange|` descending
2. Take top 3 as "movers"
3. Generate summary: `"Top movers: Gold (+12.29%), BTC (+1.62%), US10Y (-0.82%)"`
4. Bilingual: translate template strings

**Capital Flow Signal** (computed server-side):
- **DEFENSIVE**: Gold ↑ AND (S&P500 ↓ OR VIX ↑) → "Defensive signal led by Gold"
- **RISK-ON**: S&P500 ↑ AND (Gold ↓ OR flat) AND VIX ↓ → "Risk appetite increasing"
- **MIXED**: Neither clearly defensive nor risk-on → "Mixed signals"
- For VN market: similar logic using VN-Index, VN Gold, USD/VND

### 7. Error Handling & Resilience

| Failure | Handling |
|---------|---------|
| Yahoo Finance API down | Return cached data with `stale: true` flag; show "⚠️ Stale data" badge in UI |
| Individual ticker fails | Show "N/A" for that ticker; don't fail the entire response |
| VN tickers unavailable | Fallback to AI web search (existing `executeSearch` infra) |
| All tickers fail | Return `{ error: "Market data unavailable" }` with 503 status |
| Correlation insufficient data | Show "—" in heatmap cells; skip scenario detection |
| Network timeout | 10-second timeout per Yahoo request; abort and use cache |

### 8. UI Component Breakdown

```
MarketPulseDashboard/
├── Controls Bar
│   ├── Timeframe <Select> (1H | 4H | 1D | 1W)
│   ├── Auto-Refresh Toggle + Interval Select (1min | 5min | 15min | 1h)
│   ├── Refresh Now <Button>
│   ├── Last Updated display
│   └── Export Report <Button> (optional: download as PNG/PDF)
│
├── Analysis Cards Row
│   ├── Drivers Card (EN + VI)
│   ├── Capital Flow Card (EN + VI)
│   └── Correlation Signals Summary
│
├── Charts Row (responsive: side-by-side on lg, stacked on sm)
│   ├── 🇺🇸 US Market State - % Change (Recharts BarChart)
│   └── 🇻🇳 VN Market State - % Change (Recharts BarChart)
│
├── Scenario Detection Card
│   ├── Regime badge (icon + name)
│   ├── Confidence badge (e.g., "75% Confidence")
│   └── Bilingual description (EN + VI)
│
├── Correlation Heatmap (HTML table, NOT Recharts — simpler & more flexible)
│   ├── Row/Column headers = asset tickers
│   ├── Cell bg color = interpolated from correlation value
│   └── Hover tooltip = "Asset A vs Asset B\nCorrelation: X.XXX"
│
├── Market Snapshot Table (HTML table)
│   ├── Sortable columns
│   ├── Color-coded % values
│   └── Direction arrows + Momentum icons
│
├── How to Read This Dashboard Card
│   ├── EN explanation
│   └── VI explanation
│
└── Ticker Guide (collapsible section or sidebar)
    ├── Per ticker: symbol, full name, category, EN + VI description
    └── Links to more info
```

### 9. Caching Strategy

| Data | Cache Key | TTL (Trading) | TTL (Off-hours) |
|------|-----------|---------------|-----------------|
| Asset prices & % changes | `market-pulse:prices:{market}` | 5 min | 30 min |
| Correlation matrix | `market-pulse:corr:{market}` | 1 hour | 6 hours |
| Scenario detection | `market-pulse:scenario` | 5 min | 30 min |
| Historical data (for correlation) | `market-pulse:history:{symbol}` | 1 hour | 6 hours |

Use **in-memory cache** (simple `Map` with TTL) since the app runs as a single Next.js process. No need for Redis at this scale.

### 10. Performance Considerations

- **Batch Yahoo requests**: Fetch all US symbols in one call, all VN symbols in another → only 2 HTTP requests
- **Parallel fetching**: US and VN data fetched in `Promise.all`
- **Lazy correlation**: Only compute correlation matrix if the dashboard tab is active (don't compute on every API call — use a separate endpoint or query param)
- **Frontend memoization**: Memoize chart components with `React.memo` to prevent re-renders on unrelated state changes
- **SWR deduplication**: SWR automatically deduplicates identical requests within a window

### 11. Integration with Investment Analysis API

The existing AI "Global Macro War Room" at `/api/ai/investment-analysis/route.ts` must consume MarketPulse data so the Think Tank has **real-time market context** — not just web search results.

**Current flow (before):**
```
User clicks "Initiate Scan" → extractInvestmentData() → web search → AI prompt
                                                                ↑
                                                    (only web search for macro)
```

**New flow (after):**
```
User clicks "Initiate Scan" → extractInvestmentData() ─┐
                               getMarketPulseData()  ───┤→ merged into AI prompt
                               web search            ───┘
```

**What gets injected into the AI prompt:**

```
REAL-TIME MARKET STATE (from MarketPulse Engine):

🇺🇸 US Market:
- VIX: $14.15 (0.46% 1H | 0.10% 1D | 1.89% 1W) ↑
- S&P500: $5,200 (+0.24% 1H | +1.2% 1D | +2.1% 1W) ↑
- Gold: $2,375 (+12.29% 1H | +19.59% 1D | +9.10% 1W) ↑🔥
- ... (all 8 US assets)

🇻🇳 VN Market:
- VN-Index: 1,246 (+0.5% 1D | +1.2% 1W) ↑
- USD/VND: 25,650 (+0.1% 1D) →
- ... (all 5 VN assets)

Correlation Alerts:
- VIX↔S&P: -0.85 (strong inverse — normal)
- Gold↔S&P: +0.3 (BROKEN — historically inverse, now moving together)
- Oil↔S&P: 0.001 (decorrelated)

Scenario: Normal Growth (Risk-ON) — 75% confidence
Capital Flow: Defensive signal led by Gold (+12.29%). Equities gaining (+0.24%).
```

**Implementation changes to `route.ts`:**

1. Import `getMarketPulseData` from `market-data-service.ts`
2. Call it in parallel with `extractInvestmentData()` and web search
3. Add the result as a new `REAL-TIME MARKET STATE` block in `taskInstruction`
4. The AI now sees both user portfolio data AND live market state — enabling much sharper analysis

**Why this matters:**
- Currently the AI only gets market context from **web search** (which is slow, limited, and sometimes fails)
- MarketPulse provides **structured, quantitative** data (exact prices, % changes, correlations)
- The scenario detection gives the AI a pre-computed "threat level" to anchor its analysis
- Correlation anomalies ("broken correlations") give the AI non-obvious signals to reason about
- If web search fails (INTELLIGENCE BLACKOUT), the AI still has real-time market data to work with

---

## File Structure

```
src/
├── lib/
│   └── services/
│       └── market-data-service.ts              [NEW] Core data fetching + computation
│
├── app/
│   ├── api/
│   │   ├── market-pulse/
│   │   │   └── route.ts                        [NEW] GET /api/market-pulse
│   │   └── ai/
│   │       └── investment-analysis/
│   │           └── route.ts                    [MODIFY] Import & inject MarketPulse data
│   └── investments/
│       └── page.tsx                            [MODIFY] Add "Market Pulse" tab
│
└── components/
    └── dashboard/
        └── market-pulse-dashboard.tsx          [NEW] Full dashboard component
```

---

## Verification Plan

### Automated Tests
```bash
cd /Users/destnguyxn/projects/wealth-management && npx vitest run
```

### Manual Verification (Browser)
1. `pnpm run dev` → navigate to `/investments` → "Market Pulse" tab
2. Verify both US and VN charts render with correct number of bars
3. Positive bars are green, negative are red
4. Correlation heatmap renders with correct color interpolation
5. Scenario detection card shows appropriate regime
6. Market snapshot table shows multi-timeframe data
7. Drivers, Capital Flow cards display for both markets
8. Timeframe selector changes data
9. Auto-refresh works when enabled
10. Error states render correctly (disconnect wifi to test)