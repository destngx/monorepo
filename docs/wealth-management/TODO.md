# Wealth Management: TODO (parked ideas)

> **Status**: Parked for later review, 2026-10-15. Nothing here is approved or scheduled. Items that change scope must land in [REQUIREMENTS.md](./REQUIREMENTS.md) (with an ID and status) before they are built.

## 1. Reduce the weekly capture chore

**Problem**: transactions are captured 1-5 times a week with Smart Capture. It is repetitive: collect screenshots per app, open the capture page, approve every row.

**Constraints**: iPhone; the banks and wallets in scope are listed below; no SMS or email alerts, changes are only visible in the apps; solutions must cost nothing; prefer privacy.

### 1.1 Research findings (2026-10-15)

No free, official, automatic feed exists today for the three main accounts.

| Source      | Official API / SDK                                                                                                                        | Third-party aggregators                                                                                                                                              |
| ----------- | ----------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Techcombank | No public developer portal; APIs are for corporate partners                                                                               | SePay: SMS route only (Techcombank SMS Banking ~9.900đ/month). Casso / Pay2S: internet-banking username and password, paid after trial. No open-source library found |
| Sacombank   | Sacombank Open API (2026, `openapi.sacombank.com.vn`) for business partners; no individual transaction-history endpoint found             | SePay official API, FREE plan 50 tx/month (overage billed). Built for payment accounts; credit card coverage not confirmed                                           |
| ZaloPay     | Merchant APIs only (query one order by its ID); no wallet history                                                                         | None; no statement export found in the app                                                                                                                           |
| Binance     | Read-only API key, free, official                                                                                                         | Not needed                                                                                                                                                           |
| Regulation  | Circular 64/2024/TT-NHNN: banks must offer a consent-based "get transaction history" Open API (existing connections comply by 01/03/2027) | Access goes to contracted third parties, not individuals; consent lasts at most 180 days; e-wallets not covered                                                      |

Notification routes (tested 2026-10-16, see D29): notifications on Mac need iOS 18 and the iPhone 12 Pro stays on iOS 17; macOS hides ANCS from apps (a CoreBluetooth probe saw the iPhone's other services but never ANCS); a fitness band cannot export what it receives; iOS apps cannot read other apps' notifications; Techcombank notifications are unreliable.

Rejected for now: unofficial bank libraries (credentials on disk, against bank terms, only MB Bank is maintained) and SePay/Casso for the main banks (no free official route, third party sees all activity).

### 1.2 Candidate work

- [x] **Screenshot capture from the phone**: moved to REQUIREMENTS.md §6.3 (WM-CAP-15..20, D29) on 2026-10-16. AssistiveTouch runs a Shortcut that sends a screenshot to a capture inbox; captures become drafts approved later.
- [ ] **Local OCR** with `apps/mac-ocr` (Vision, bounding boxes) plus deterministic per-app parsers in the §6.3 pipeline; fall back to the model on PII-masked text only, never images. Drop screenshots with no banking text before anything leaves the Mac. Verify Vision OCR quality on Vietnamese bank screens with one sample per app first.
- [ ] **On-demand statements**: check whether the Techcombank and Sacombank apps export a statement (sao kê) for any date range, not only monthly. Monthly-only is too slow to be the main source.
- [ ] **ESP32 ANCS client** (~100k VND) for real-time Sacombank alerts, if wanted later. Techcombank notifications are unreliable, so it cannot replace capture.
- [x] **Recurring transactions** (salary, rent, subscriptions, loan payments) proposed or recorded on schedule: moved to [ROADMAP](./ROADMAP.md) R4.
- [ ] **Balance-only tracking** option for wallets: enter the weekly balance, record one adjustment for the difference.
- [ ] **Binance read-only API** source (IP-restricted key).
- [ ] **Bank feed source port**: design a generic "feed" source for the Smart Capture pipeline (WM-ING-3) so a Circular 64 Open API aggregator can be plugged in around 2027. Revisits D11.

## 2. Financial health improvements

Ordered by impact.

- [ ] **Monthly snapshots**: `Snapshots` sheet tab written at month close (net worth, assets, liabilities, health score and components, savings rate). Trend charts on `/health` and the Dashboard; split each month's net worth change into savings vs market gains. Net worth history today is back-calculated from transactions and misses price changes.
- [ ] **Fix the diversification score**: `domain/asset_allocation.go` gives full marks for an even cash / investment / crypto split (33% crypto scores as healthy). Score distance from a user-set target allocation, with a crypto ceiling.
- [x] **Real goals**: moved to [ROADMAP](./ROADMAP.md) R3 (pooled savings, D31, D32). Original note: (WM-GOAL-3..5): deadline and monthly contribution columns in the Goals sheet; required monthly vs actual pace; check that all goals fit the average monthly surplus. Deadline is hardcoded `"TBD"` in `service/sheet_mappers.go`.
- [x] ~~**Debt payoff planner**~~ dropped: the user lends rather than borrows (D30, [ROADMAP](./ROADMAP.md) R1). Original note:: add interest rate, principal and term to the Loan sheet; avalanche vs snowball, payoff date, total interest, prepay vs invest.
- [x] **Recurring bill and subscription detection**: moved to [ROADMAP](./ROADMAP.md) R4. Original note: from transaction history; upcoming-bills calendar; price-increase and unused-subscription flags.
- [ ] **Deterministic cash flow forecast** (30/90 days per account) from recurring items, budget limits and card due dates; replaces the AI-generated projection in the budget advisor.
- [ ] **Better projection**: `ProjectNetWorth` compounds current net worth only; include average monthly savings and add an FI view (target = annual expenses x 25, year reached, effect of +5 points of savings rate).
- [x] **Investment performance**: moved to [ROADMAP](./ROADMAP.md) M3. Original note:: cost basis per holding, unrealised gain/loss, money-weighted return vs bank savings rate.
- [ ] **Protection pillar** in the health score (insurance coverage, dependants).
- [ ] **Spending anomaly alerts** (category > 2x usual, duplicate charge).
- [x] **Monthly close checklist page**: moved to [ROADMAP](./ROADMAP.md) R8. Original note:: reconcile, review health, apply budget advice, check cards and loans, write the snapshot.

## 2b. Follow-ups from the roadmap build

- [ ] **Crypto 24h change from the market-data-server**: the exposure view (WM-INV-24) builds Yahoo-style `<COIN>-USD` candle symbols in the engine to get a 24h move. Add a `change24h` field to the server's crypto ticker so source symbols stay server-side, then drop the engine mapping.
- [ ] **One valuation per Investments page load**: exposure, hedges and the portfolio each recompute the valuation (the market cache absorbs it); share one valuation per request.

## 3. Sources

- [SePay pricing and bank list](https://sepay.vn/bang-gia.html)
- [Casso free plans](https://casso.vn/casso-flow-sap-co-goi-mien-phi-giao-dich-khong-gioi-han/), [Casso getting started](https://docs.casso.vn/huong-dan/bat-dau-su-dung-casso), [Casso Techcombank](https://casso.vn/api-techcombank/)
- [Open Banking Tracker: Techcombank](https://www.openbankingtracker.com/provider/techcombank)
- [Sacombank Open Banking (2026)](https://www.sacombank.com.vn/trang-chu/tin-tuc/tin-sacombank/2026/sacombank-mo-rong-he-sinh-thai-doi-tac-thong-qua-giai-phap-ngan-hang-mo.html)
- [ZaloPay order query API](https://docs.zalopay.vn/docs/specs/order-query/)
- [Circular 64/2024/TT-NHNN](https://thuvienphapluat.vn/van-ban/Tien-te-Ngan-hang/Thong-tu-64-2024-TT-NHNN-trien-khai-giao-dien-lap-trinh-ung-dung-mo-nganh-Ngan-hang-641360.aspx)
