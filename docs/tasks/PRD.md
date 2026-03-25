# PRD: Refactor Gold and Exchange Rate Indicators

## 🎯 Objective

Update the wealth management investment page (fmarket tab) to show specific gold price and exchange rate information for a fixed date (25/03/2026) and remove the "Đối tác Quản lý quỹ Chiến lược" component.

## 📝 User Requirements

- Refactor indicators for gold price and exchange rate in fmarket tab.
- Include the following information:
  - Ngày 25/03/2026
  - SJC mua vào: 172,000,000 VND
  - SJC bán ra: 175,000,000 VND
  - Vàng nhẫn mua vào: 172,000,000 VND
  - Vàng nhẫn bán ra: 175,000,000 VND
  - Vàng thế giới: 145,093,896 VND
  - Tỷ giá (USD/VND): 26,359 VND
- Remove component "Đối tác Quản lý quỹ Chiến lược".

## 🛠 Actionable Tasks

- [ ] Refactor `FmarketDashboard` component in `apps/wealth-management/src/features/investments/ui/fmarket-dashboard.tsx`.
  - [ ] Update Gold Price Card to show SJC and Vàng nhẫn (Mua/Bán).
  - [ ] Update USD Rate Card.
  - [ ] Include World Gold Price info.
  - [ ] Set specific date "Ngày 25/03/2026".
- [ ] Refactor Gold and USD rate charts to support multiple ranges (YTD, 6M, 1Y, 3Y, 5Y, All).
  - [ ] Add range selection buttons/tabs to the charts.
  - [ ] Update `useFmarket` hook or component logic to fetch/filter data based on the selected range.
- [ ] Refactor Fund subtabs to display available tickers based on new API requests.
  - [ ] Add "Tiền tệ" (MMF) and "Cân bằng" (BALANCED) subtabs.
  - [ ] Update API calls to fetch 1000 items and include `isMMFFund` for Money Market Funds.
  - [ ] Ensure `isIpo: false` is used in all fund filter requests.
- [ ] Implement Ticker Details view.
  - [ ] Fetch product details using `getProductDetails(code)`.
  - [ ] Include price, details, summary, and nav history chart.
  - [ ] Design a premium view for fund details when a ticker is selected.
- [ ] Update Bank Interest UI according to the new API response structure.
  - [ ] Display the note from the API response.
  - [ ] Update bank rate display to use `value` and `name` from the `bankList`.
  - [ ] Display "Cập nhật lúc [updateAt] theo [sourceNote]".
- [ ] Remove the Partner Issuers section ("Đối tác Quản lý quỹ Chiến lược") from `FmarketDashboard`.
- [ ] Verify the changes visually.

## 📅 Timeline

- Start date: 2026-03-25
- Target completion: 2026-03-25
