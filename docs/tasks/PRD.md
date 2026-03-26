# PRD: Vnstock API Expansion (Community Level)

## 🎯 Objective

Expand `vnstock-server` to provide full access to all community-level features available in `vnstock` v3.5.0. This will enable the wealth management application and other consumers to access comprehensive Vietnamese market data including fundamental and technical analysis, company profiles, and trading statistics.

## 📝 User Requirements

1.  **Identify currently implemented APIs** in `vnstock-server`.
2.  **Identify all other community-level APIs** from `vnstock` v3.5.0.
3.  **Implement all missing APIs** with:
    - Consistent error handling and logging.
    - Upstash Redis caching where applicable.
    - Community tier rate limit management (60 req/min, 10k req/day).
    - API key authentication for better reliability.

## 🛠 Actionable Tasks

### Phase 1: Planning & Setup

- [x] Audit current implementation (8 endpoints identified).
- [x] Research `vnstock` v3.5.0 Community API surface area.
- [x] Update project documentation (`PRD.md`, `progress.md`).

### Propose plan for next phases

mplementation Plan - Vnstock API Expansion
Expand vnstock-server to include all available community-level APIs from the
vnstock
v3.5.0 library.

Proposed Changes
vnstock-server
[MODIFY]
main.py
Refactor existing endpoints to use a more consistent structure if needed.
Implement new endpoints for:
Listing: all_bonds, all_covered_warrant, all_future_indices, all_government_bonds, industries_icb, symbols_by_exchange, symbols_by_group, symbols_by_industries.
Quote & Price: intraday, price_depth.
Finance: balance_sheet, cash_flow, income_statement, ratio, get_all.
Company: affiliate, dividends, events, insider_deals, news, officers, overview, profile, ratio_summary, shareholders, subsidiaries, trading_stats.
Trading: price_board.
[MODIFY]
cache.py
Update CacheConfig with new TTL settings for the newly added data types (e.g., fundamental data can be cached longer than price data).
Verification Plan
Automated Tests
Create a new test file apps/vnstock-server/tests/test_new_endpoints.py to verify each new endpoint.
Use TestClient from FastAPI to simulate requests.
Command: /Users/ez2/projects/personal/monorepo/apps/vnstock-server/.venv/bin/pytest apps/vnstock-server/tests/test_new_endpoints.py
Manual Verification
Run the server locally: bun nx serve vnstock-server (or the appropriate command from
project.json
).

### Phase 2: Core Stock & Market APIs

- [ ] Implement Listing APIs:
  - `all_bonds`, `all_covered_warrant`, `all_future_indices`, `all_government_bonds`.
  - `industries_icb`, `symbols_by_exchange`, `symbols_by_group`, `symbols_by_industries`.
- [ ] Implement Price & Trading APIs:
  - `intraday`, `price_depth`, `price_board`.

### Phase 3: Financial & Fundamental APIs

- [ ] Implement Financial Statement APIs:
  - `balance_sheet`, `cash_flow`, `income_statement`, `ratio`, `get_all`.
- [ ] Implement Company Profile & Data APIs:
  - `overview`, `profile`, `ratio_summary`, `trading_stats`.
  - `news`, `events`, `dividends`.
  - `officers`, `shareholders`, `insider_deals`.
  - `subsidiaries`, `affiliate`.

### Phase 4: Verification

- [ ] Create automated tests for all new endpoints.
- [ ] Verify caching and rate limiting for new endpoints.
- [ ] Update `walkthrough.md`.

## 📅 Timeline

- Start date: 2026-03-26
- Target completion: 2026-03-27
