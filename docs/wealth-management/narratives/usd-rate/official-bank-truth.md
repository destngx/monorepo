# Details: The Official Bank USD/VND Truth

## 1. Feature Meaning

The **Official Bank Rate** is the "Safe-Haven" benchmark for the Wealth Management application. It represents the liquidity and exchange reality of the Vietnamese banking sector.

## 2. API Sources & Logic

- **Primary Source**: `POST /api/fmarket` (action: `getUsdRateHistory`).
- **Institutional Context**: Fmarket provides history for Vietcombank and Interbank rates, which are the basis for legal financial reporting in Vietnam.

## 3. Business Use Cases

- **Portfolio Reporting**: Used to value accounts that are held in USD but reported to the user in the base currency (VND).
- **Tax/Legal Benchmark**: Any capital gains from overseas investments must be converted using this official rate for compliance during the "Summary" phase of the Financial Health report.

## 4. Why Fmarket?

Using Fmarket's proxy bypasses CORS issues while ensuring the data is "Vietnamese-first," reflecting the actual sell/buy rates that a domestic user encounters at physical banks.
