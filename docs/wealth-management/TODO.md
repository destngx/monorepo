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

Rejected for now: unofficial bank libraries (credentials on disk, against bank terms, only MB Bank is maintained) and SePay/Casso for the main banks (no free official route, third party sees all activity).

### 1.2 Candidate work

- [ ] **Check statement export** in the Techcombank and Sacombank apps (sao kê: PDF or Excel, which period). If available, build a statement importer feeding the Smart Capture pipeline: one export per bank per month instead of screenshots.
- [ ] **Screenshot automation, local first** (free, private):
  - iOS Shortcut on a schedule (iOS 17+ personal automation) uploads screenshots taken since the last run to the engine over the tailnet with `API_TOKEN`.
  - Server-side capture inbox: materials stored encrypted with a TTL, processed in the background, deleted after commit. **Changes WM-CAP-10.**
  - Local OCR with `apps/mac-ocr` (Vision, bounding boxes) plus deterministic per-app parsers; fall back to the model on PII-masked text only, never images. Drop screenshots with no banking text before anything leaves the Mac.
  - Verify Vision OCR quality on Vietnamese bank screens with one sample per app first.
- [ ] **Auto-commit trusted rows**: high confidence, account and payee matched by `CaptureRules`, no dedup hit, under an amount limit; everything else goes to a review queue with undo. Weekly digest ("N recorded, M need you"). **Changes the "nothing written without approval" rule (§6.2).**
- [ ] **Recurring transactions** (salary, rent, subscriptions, loan payments) proposed or recorded on schedule.
- [ ] **Balance-only tracking** option for wallets: enter the weekly balance, record one adjustment for the difference.
- [ ] **Binance read-only API** source (IP-restricted key).
- [ ] **Bank feed source port**: design a generic "feed" source for the Smart Capture pipeline (WM-ING-3) so a Circular 64 Open API aggregator can be plugged in around 2027. Revisits D11.

## 2. Financial health improvements

Ordered by impact.

- [ ] **Monthly snapshots**: `Snapshots` sheet tab written at month close (net worth, assets, liabilities, health score and components, savings rate). Trend charts on `/health` and the Dashboard; split each month's net worth change into savings vs market gains. Net worth history today is back-calculated from transactions and misses price changes.
- [ ] **Fix the diversification score**: `domain/asset_allocation.go` gives full marks for an even cash / investment / crypto split (33% crypto scores as healthy). Score distance from a user-set target allocation, with a crypto ceiling.
- [ ] **Real goals** (WM-GOAL-3..5): deadline and monthly contribution columns in the Goals sheet; required monthly vs actual pace; check that all goals fit the average monthly surplus. Deadline is hardcoded `"TBD"` in `service/sheet_mappers.go`.
- [ ] **Debt payoff planner**: add interest rate, principal and term to the Loan sheet; avalanche vs snowball, payoff date, total interest, prepay vs invest.
- [ ] **Recurring bill and subscription detection** from transaction history; upcoming-bills calendar; price-increase and unused-subscription flags.
- [ ] **Deterministic cash flow forecast** (30/90 days per account) from recurring items, budget limits and card due dates; replaces the AI-generated projection in the budget advisor.
- [ ] **Better projection**: `ProjectNetWorth` compounds current net worth only; include average monthly savings and add an FI view (target = annual expenses x 25, year reached, effect of +5 points of savings rate).
- [ ] **Investment performance**: cost basis per holding, unrealised gain/loss, money-weighted return vs bank savings rate.
- [ ] **Protection pillar** in the health score (insurance coverage, dependants).
- [ ] **Spending anomaly alerts** (category > 2x usual, duplicate charge).
- [ ] **Monthly close checklist page**: reconcile, review health, apply budget advice, check cards and loans, write the snapshot.

## 3. Sources

- [SePay pricing and bank list](https://sepay.vn/bang-gia.html)
- [Casso free plans](https://casso.vn/casso-flow-sap-co-goi-mien-phi-giao-dich-khong-gioi-han/), [Casso getting started](https://docs.casso.vn/huong-dan/bat-dau-su-dung-casso), [Casso Techcombank](https://casso.vn/api-techcombank/)
- [Open Banking Tracker: Techcombank](https://www.openbankingtracker.com/provider/techcombank)
- [Sacombank Open Banking (2026)](https://www.sacombank.com.vn/trang-chu/tin-tuc/tin-sacombank/2026/sacombank-mo-rong-he-sinh-thai-doi-tac-thong-qua-giai-phap-ngan-hang-mo.html)
- [ZaloPay order query API](https://docs.zalopay.vn/docs/specs/order-query/)
- [Circular 64/2024/TT-NHNN](https://thuvienphapluat.vn/van-ban/Tien-te-Ngan-hang/Thong-tu-64-2024-TT-NHNN-trien-khai-giao-dien-lap-trinh-ung-dung-mo-nganh-Ngan-hang-641360.aspx)
