# Vnstock API Walkthrough

This document provides a guide to the available endpoints in the `vnstock-server` wrapper.

## Base URL

`http://localhost:8000/api/v1`

## Authentication

Configure `VNSTOCK_API_KEY` in `.env.local` to enable higher rate limits and reliable data access.

## Caching

All endpoints use Upstash Redis caching by default.

- Quotes: 1 minute
- Listings: 1 hour
- Historical: 1 hour
- Finance: 24 hours
- Company: 24 hours

## Available Endpoints

### 1. Stock Listing

- `GET /stocks/listing`: All symbols.
- `GET /listing/bonds`: Corporate bonds.
- `GET /listing/covered-warrants`: Warrants.
- `GET /listing/future-indices`: Future indices.
- `GET /listing/government-bonds`: Gov bonds.
- `GET /listing/industries-icb`: ICB industry tree.
- `GET /listing/symbols-by-exchange?exchange=HOSE`: Filter by exchange.
- `GET /listing/symbols-by-group?group=VN30`: Filter by group.
- `GET /listing/symbols-by-industries?industry=Finance`: Filter by industry.

### 2. Market Data

- `GET /stocks/quote?symbol=VND`: Current price and change.
- `GET /stocks/historical?symbol=VND&start_date=2024-01-01`: Historical OHLCV.
- `GET /stocks/intraday?symbol=VND`: Intraday price action.
- `GET /stocks/price-depth?symbol=VND`: Level 2 order book.
- `GET /stocks/price-board?symbols=VND,SSI`: Real-time price board for multiple tickers.

### 3. Financial Statements

- `GET /finance/balance-sheet?symbol=VND&period=QUARTERLY`: Assets, liabilities, equity.
- `GET /finance/income-statement?symbol=VND&period=QUARTERLY`: Revenue, profit, expenses.
- `GET /finance/cash-flow?symbol=VND&period=QUARTERLY`: Operating, investing, financing flows.
- `GET /finance/ratio?symbol=VND&period=QUARTERLY`: Financial ratios.
- `GET /finance/get-all?symbol=VND&period=QUARTERLY`: Combined financial view.

### 4. Company Profile & Data

- `GET /company/overview?symbol=VND`: High-level overview.
- `GET /company/profile?symbol=VND`: Detailed profile.
- `GET /company/ratio-summary?symbol=VND`: Quick ratio view.
- `GET /company/trading-stats?symbol=VND`: Trading statistics.
- `GET /company/news?symbol=VND&page_size=20`: Recent news.
- `GET /company/events?symbol=VND`: Corporate events.
- `GET /company/dividends?symbol=VND`: Dividend history.
- `GET /company/officers?symbol=VND`: List of officers.
- `GET /company/shareholders?symbol=VND`: Main shareholders.
- `GET /company/insider-deals?symbol=VND`: Insider trading.
- `GET /company/subsidiaries?symbol=VND`: List of subsidiaries.
- `GET /company/affiliate?symbol=VND`: Affiliated companies.

### 5. Health & Status

- `GET /health`: Basic health check.
- `GET /health/cache`: Cache configuration and backend status.
- `GET /health/ratelimit`: Current rate limit usage.
- `GET /health/quota`: Detailed quota analysis and next reset time.
